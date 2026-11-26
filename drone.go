package main

import (
	"math"
	"math/rand/v2"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type DroneKind int

const (
	Sentinel DroneKind = 0
	Hunter   DroneKind = 1
)

type Drone struct {
	PhysicsBody
	Kind        DroneKind
	PatrolSpeed float32
	ChaseSpeed  float32
	Damage      int
	HitCooldown float32
	GridKey     rl.Vector2
	Anim        AnimationFSM
}

func NewDrone(kind DroneKind, pos rl.Vector2, sector int, art *Art) Drone {
	drone := Drone{Kind: kind}
	drone.Pos = pos
	drone.Radius = 13
	name := "sentinel"
	if kind == Hunter {
		name = "hunter"
		drone.PatrolSpeed = 85
		drone.ChaseSpeed = 160 + float32(sector)*10
		drone.Damage = 25
	} else {
		drone.PatrolSpeed = 65
		drone.ChaseSpeed = 118 + float32(sector)*10
		drone.Damage = 20
	}
	drone.Anim = NewAnimationFSM()
	drone.Anim.AddAnimation(NewAnimation("patrol", art.Get(name+"_patrol"), 0.18, true))
	drone.Anim.AddAnimation(NewAnimation("alert", art.Get(name+"_alert"), 0.12, true))
	drone.Anim.AddAnimation(NewAnimation("stunned", art.Get(name+"_stunned"), 0.07, true))
	return drone
}

func (d *Drone) MoveTowards(target rl.Vector2, speed float32, dungeon *Dungeon) bool {
	toTarget := rl.Vector2Subtract(target, d.Pos)
	distance := rl.Vector2Length(toTarget)
	if distance < 4 {
		d.Vel = rl.Vector2Zero()
		return true
	}
	step := speed
	if distance < speed*FrameTime() {
		step = distance / FrameTime()
	}
	d.Vel = rl.Vector2Scale(rl.Vector2Normalize(toTarget), step)
	d.VelocityTick(dungeon)
	return false
}

func (d *Drone) Draw(state AIState) {
	switch state {
	case Stunned:
		d.Anim.ChangeAnimationState("stunned")
	case Chasing:
		d.Anim.ChangeAnimationState("alert")
	default:
		d.Anim.ChangeAnimationState("patrol")
	}
	rl.DrawCircleV(rl.NewVector2(d.Pos.X+3, d.Pos.Y+5), d.Radius+2, rl.NewColor(0, 0, 0, 70))
	d.Anim.DrawWithFSM(d.Pos, 36, 0, rl.White)
}

type AIState int

const (
	Patrolling    AIState = 0
	Investigating AIState = 1
	Chasing       AIState = 2
	Searching     AIState = 3
	Stunned       AIState = 4
)

type DroneAI struct {
	Drone        *Drone
	Game         *Game
	State        AIState
	ChangedState bool
	Timer        float32
	TickCount    int
	SightRange   float32
	HearingScale float32
	TargetPos    rl.Vector2
	Waypoints    []rl.Vector2
	WaitTime     float32
	StunTime     float32
	LastSeen     rl.Vector2
	StuckTimer   float32
	LastPos      rl.Vector2
	LookAngle    float32
	rng          *rand.Rand
}

func NewDroneAI(drone *Drone, game *Game, seed uint64) *DroneAI {
	ai := &DroneAI{Drone: drone, Game: game, State: Patrolling, ChangedState: true}
	ai.rng = rand.New(rand.NewPCG(seed, 77))
	if drone.Kind == Hunter {
		ai.SightRange = 210
		ai.HearingScale = 1.4
	} else {
		ai.SightRange = 245
		ai.HearingScale = 1.0
	}
	ai.LastPos = drone.Pos
	return ai
}

func (ai *DroneAI) SetState(newState AIState) {
	ai.ChangedState = true
	ai.State = newState
}

func (ai *DroneAI) Tick() {
	if ai.ChangedState {
		ai.Timer = 0
		ai.TickCount = 0
		ai.StuckTimer = 0
		ai.ChangedState = false
	}
	if ai.Drone.HitCooldown > 0 {
		ai.Drone.HitCooldown -= FrameTime()
	}

	switch ai.State {
	case Patrolling:
		ai.TickPatrol()
	case Investigating:
		ai.TickInvestigate()
	case Chasing:
		ai.TickChase()
	case Searching:
		ai.TickSearch()
	case Stunned:
		ai.TickStunned()
	}

	ai.Timer += FrameTime()
	ai.TickCount++
}

func (ai *DroneAI) CanSeePlayer() bool {
	player := &ai.Game.Player
	if player.Health <= 0 {
		return false
	}
	distance := rl.Vector2Distance(ai.Drone.Pos, player.Pos)
	if distance > ai.SightRange {
		return false
	}
	return ai.Game.Dungeon.LineOfSight(ai.Drone.Pos, player.Pos)
}

func (ai *DroneAI) SpotPlayer() {
	ai.LastSeen = ai.Game.Player.Pos
	ai.SetState(Chasing)
	ai.Game.OnPlayerSpotted(ai)
}

func (ai *DroneAI) RouteTo(target rl.Vector2) []rl.Vector2 {
	dungeon := &ai.Game.Dungeon
	route := make([]rl.Vector2, 0)
	if dungeon.LineOfSight(ai.Drone.Pos, target) {
		return append(route, target)
	}
	current := dungeon.RoomAt(ai.Drone.Pos)
	goal := dungeon.RoomAt(target)
	if current < 0 || goal < 0 || current == goal {
		return append(route, target)
	}
	if door, ok := dungeon.DoorBetween(current, goal); ok {
		route = append(route, door.SidePoint(dungeon, current), door.Center(), door.SidePoint(dungeon, goal))
		return append(route, target)
	}
	for _, middle := range dungeon.Neighbors(current) {
		if !dungeon.Connected(middle, goal) {
			continue
		}
		first, _ := dungeon.DoorBetween(current, middle)
		second, _ := dungeon.DoorBetween(middle, goal)
		route = append(route, first.SidePoint(dungeon, current), first.Center(), first.SidePoint(dungeon, middle))
		route = append(route, second.SidePoint(dungeon, middle), second.Center(), second.SidePoint(dungeon, goal))
		return append(route, target)
	}
	return nil
}

func (ai *DroneAI) FollowWaypoints(speed float32) bool {
	if len(ai.Waypoints) == 0 {
		return true
	}
	if ai.Drone.MoveTowards(ai.Waypoints[0], speed, &ai.Game.Dungeon) {
		ai.Waypoints = ai.Waypoints[1:]
	}
	return len(ai.Waypoints) == 0
}

func (ai *DroneAI) IsStuck() bool {
	if rl.Vector2Distance(ai.Drone.Pos, ai.LastPos) > 0.5 {
		ai.StuckTimer = 0
	} else {
		ai.StuckTimer += FrameTime()
	}
	ai.LastPos = ai.Drone.Pos
	return ai.StuckTimer > 1.0
}

func (ai *DroneAI) ChooseNextPatrol() {
	dungeon := &ai.Game.Dungeon
	current := dungeon.RoomAt(ai.Drone.Pos)
	if current < 0 {
		ai.Waypoints = ai.RouteTo(ai.Game.NearestRoomCenter(ai.Drone.Pos))
		return
	}
	neighbors := dungeon.Neighbors(current)
	if len(neighbors) > 0 && ai.rng.IntN(100) < 30 {
		next := neighbors[ai.rng.IntN(len(neighbors))]
		if next != ai.Game.StartRoom {
			door, _ := dungeon.DoorBetween(current, next)
			ai.Waypoints = []rl.Vector2{
				door.SidePoint(dungeon, current),
				door.Center(),
				door.SidePoint(dungeon, next),
				dungeon.Rooms[next].RandomPoint(ai.rng, ai.Drone.Radius),
			}
			return
		}
	}
	ai.Waypoints = []rl.Vector2{dungeon.Rooms[current].RandomPoint(ai.rng, ai.Drone.Radius)}
}

func (ai *DroneAI) TickPatrol() {
	if ai.CanSeePlayer() {
		ai.SpotPlayer()
		return
	}
	if ai.TickCount == 0 {
		ai.Waypoints = nil
		ai.WaitTime = 0.5 + ai.rng.Float32()
	}
	if ai.WaitTime > 0 {
		ai.WaitTime -= FrameTime()
		ai.LookAngle += FrameTime() * 60
		if ai.WaitTime <= 0 {
			ai.ChooseNextPatrol()
		}
		return
	}
	if ai.FollowWaypoints(ai.Drone.PatrolSpeed) || ai.IsStuck() {
		ai.Waypoints = nil
		ai.StuckTimer = 0
		ai.WaitTime = 0.8 + ai.rng.Float32()*1.6
	}
}

func (ai *DroneAI) HearNoise(pos rl.Vector2, radius float32) {
	if ai.State == Stunned || ai.State == Chasing {
		return
	}
	if rl.Vector2Distance(ai.Drone.Pos, pos) > radius*ai.HearingScale {
		return
	}
	route := ai.RouteTo(pos)
	if route == nil {
		return
	}
	ai.TargetPos = pos
	ai.Waypoints = route
	ai.SetState(Investigating)
}

func (ai *DroneAI) TickInvestigate() {
	if ai.CanSeePlayer() {
		ai.SpotPlayer()
		return
	}
	if ai.FollowWaypoints(ai.Drone.PatrolSpeed*1.6) || ai.IsStuck() {
		ai.Waypoints = nil
		ai.SetState(Searching)
	}
}

func (ai *DroneAI) TickChase() {
	if ai.CanSeePlayer() {
		ai.LastSeen = ai.Game.Player.Pos
		if ai.Drone.HitCooldown > 0.6 {
			return
		}
		ai.Drone.MoveTowards(ai.LastSeen, ai.Drone.ChaseSpeed, &ai.Game.Dungeon)
		if ai.IsStuck() {
			ai.Waypoints = nil
			ai.SetState(Searching)
		}
		return
	}
	ai.Waypoints = ai.RouteTo(ai.LastSeen)
	ai.SetState(Searching)
}

func (ai *DroneAI) TickSearch() {
	if ai.CanSeePlayer() {
		ai.SpotPlayer()
		return
	}
	if len(ai.Waypoints) > 0 {
		if ai.FollowWaypoints(ai.Drone.ChaseSpeed*0.8) || ai.IsStuck() {
			ai.Waypoints = nil
			ai.Timer = 0
		}
		return
	}
	ai.LookAngle += FrameTime() * 220
	if ai.Timer > 3 {
		ai.SetState(Patrolling)
	}
}

func (ai *DroneAI) Stun(duration float32) {
	ai.StunTime = duration
	ai.Waypoints = nil
	ai.Drone.Vel = rl.Vector2Zero()
	ai.SetState(Stunned)
}

func (ai *DroneAI) TickStunned() {
	if ai.Timer > ai.StunTime {
		ai.SetState(Searching)
	}
}

func (ai *DroneAI) DrawStatus(font float32) {
	pos := ai.Drone.Pos
	switch ai.State {
	case Chasing:
		rl.DrawCircleLines(int32(pos.X), int32(pos.Y), ai.SightRange, rl.NewColor(233, 87, 95, 70))
		DrawWorldText("!", pos.X-3, pos.Y-36, font, rl.NewColor(255, 110, 110, 255))
	case Investigating, Searching:
		DrawWorldText("?", pos.X-5, pos.Y-36, font, rl.NewColor(255, 210, 120, 255))
		angle := float64(ai.LookAngle) * math.Pi / 180
		tip := rl.NewVector2(pos.X+float32(math.Cos(angle))*26, pos.Y+float32(math.Sin(angle))*26)
		rl.DrawLineEx(pos, tip, 2, rl.NewColor(255, 210, 120, 120))
	case Stunned:
		remaining := 1 - ai.Timer/ai.StunTime
		rl.DrawRing(pos, 20, 23, -90, -90+360*remaining, 24, rl.NewColor(75, 219, 202, 200))
	}
}
