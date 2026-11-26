package main

import (
	"fmt"
	"math/rand/v2"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	ScreenWidth  = 960
	ScreenHeight = 600
)

type GameMode int

const (
	Intro   GameMode = 0
	Playing GameMode = 1
	Paused  GameMode = 2
	Lost    GameMode = 3
	Won     GameMode = 4
)

type SectorConfig struct {
	Name      string
	Width     int
	Height    int
	Rooms     int
	Sentinels int
	Hunters   int
	Pickups   int
}

var Sectors = []SectorConfig{
	{Name: "HABITATION RING", Width: 44, Height: 32, Rooms: 9, Sentinels: 4, Hunters: 0, Pickups: 7},
	{Name: "CARGO DECKS", Width: 52, Height: 38, Rooms: 12, Sentinels: 5, Hunters: 2, Pickups: 9},
	{Name: "REACTOR CORE", Width: 60, Height: 44, Rooms: 15, Sentinels: 6, Hunters: 4, Pickups: 11},
}

type Relay struct {
	Pos      rl.Vector2
	Room     int
	Repaired bool
	Progress float32
	Anim     AnimationFSM
}

type Snapshot struct {
	Health int
	Uses   []int
}

type Game struct {
	Mode          GameMode
	Art           *Art
	Sound         *SoundBank
	RunSeed       uint64
	Sector        int
	Dungeon       Dungeon
	StartRoom     int
	ExitRoom      int
	ExitPos       rl.Vector2
	ExitAnim      AnimationFSM
	Player        Player
	AIs           []*DroneAI
	Grid          SpatialGrid
	Relays        []Relay
	Repaired      int
	Pickups       []*Pickup
	Projectiles   []*Projectile
	Decoys        []*Decoy
	Effects       []Effect
	Camera        rl.Camera2D
	CamPos        rl.Vector2
	Shake         float32
	Message       string
	MessageTimer  float32
	ScanTimer     float32
	IntroTimer    float32
	WeldTimer     float32
	RepairNoise   float32
	AlertCooldown float32
	SectorStart   Snapshot
	RunTime       float32
	DamageTaken   int
	DronesStunned int
	TimesSpotted  int
	Deaths        int
	BestTime      float32
	NewRecord     bool
	rng           *rand.Rand
}

func NewGame(art *Art, sound *SoundBank, seed uint64) *Game {
	g := &Game{Art: art, Sound: sound, RunSeed: seed, Sector: 1}
	g.BestTime = LoadBestTime()
	g.Player = NewPlayer(rl.Vector2Zero(), art)
	g.StartSector()
	g.SaveSnapshot()
	return g
}

func (g *Game) SaveSnapshot() {
	g.SectorStart = Snapshot{Health: g.Player.Health}
	for _, tool := range g.Player.Tools {
		g.SectorStart.Uses = append(g.SectorStart.Uses, tool.CurrentUses)
	}
}

func (g *Game) RetrySector() {
	g.Player.Health = g.SectorStart.Health
	if g.Player.Health < 60 {
		g.Player.Health = 60
	}
	for i := range g.Player.Tools {
		g.Player.Tools[i].CurrentUses = g.SectorStart.Uses[i]
	}
	g.StartSector()
}

func (g *Game) StartSector() {
	config := Sectors[g.Sector-1]
	g.Dungeon = NewDungeon(config.Width, config.Height)
	g.Dungeon.Generate(g.RunSeed, uint64(g.Sector), config.Rooms)
	g.rng = rand.New(rand.NewPCG(g.RunSeed, uint64(g.Sector)*1000+7))
	rooms := g.Dungeon.Rooms

	g.StartRoom = 0
	g.Dungeon.Rooms[0].Explored = true
	g.Player.Pos = rooms[0].Center()
	g.Player.Vel = rl.Vector2Zero()
	g.Player.DamageTimer = 0
	g.Player.Stamina = 1

	g.ExitRoom = 1
	farthest := float32(0)
	for i := 1; i < len(rooms); i++ {
		distance := rl.Vector2Distance(rooms[0].Center(), rooms[i].Center())
		if distance > farthest {
			farthest = distance
			g.ExitRoom = i
		}
	}
	g.ExitPos = rooms[g.ExitRoom].Center()
	g.ExitAnim = NewAnimationFSM()
	g.ExitAnim.AddAnimation(NewAnimation("locked", g.Art.Get("airlock_locked"), 0.4, true))
	g.ExitAnim.AddAnimation(NewAnimation("open", g.Art.Get("airlock_open"), 0.15, false))

	nearStart := make(map[int]bool)
	nearStart[g.StartRoom] = true
	for _, neighbor := range g.Dungeon.Neighbors(g.StartRoom) {
		nearStart[neighbor] = true
	}

	candidates := make([]int, 0)
	for i := range rooms {
		if i != g.StartRoom && i != g.ExitRoom {
			candidates = append(candidates, i)
		}
	}
	g.rng.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	relayRooms := make([]int, 0)
	for _, room := range candidates {
		if len(relayRooms) < 3 && !nearStart[room] {
			relayRooms = append(relayRooms, room)
		}
	}
	for _, room := range candidates {
		if len(relayRooms) < 3 && nearStart[room] && room != g.StartRoom {
			relayRooms = append(relayRooms, room)
		}
	}

	g.Relays = nil
	g.Repaired = 0
	for _, room := range relayRooms {
		relay := Relay{Pos: rooms[room].Center(), Room: room}
		relay.Anim = NewAnimationFSM()
		relay.Anim.AddAnimation(NewAnimation("broken", g.Art.Get("relay_broken"), 0.12, true))
		relay.Anim.AddAnimation(NewAnimation("fixed", g.Art.Get("relay_fixed"), 0.25, true))
		g.Relays = append(g.Relays, relay)
	}

	g.Grid = NewSpatialGrid(160)
	g.AIs = nil
	droneRooms := make([]int, 0)
	for i := range rooms {
		if !nearStart[i] {
			droneRooms = append(droneRooms, i)
		}
	}
	if len(droneRooms) == 0 {
		droneRooms = append(droneRooms, g.ExitRoom)
	}
	for i := 0; i < config.Sentinels+config.Hunters; i++ {
		kind := Sentinel
		if i >= config.Sentinels {
			kind = Hunter
		}
		room := rooms[droneRooms[g.rng.IntN(len(droneRooms))]]
		drone := NewDrone(kind, room.RandomPoint(g.rng, 14), g.Sector, g.Art)
		ai := NewDroneAI(&drone, g, g.RunSeed+uint64(g.Sector*100+i))
		g.AIs = append(g.AIs, ai)
		g.Grid.RegisterDrone(ai)
	}

	g.Pickups = nil
	kinds := []string{"emp", "emp", "emp", "decoy", "decoy", "medkit", "medkit", "scanner"}
	for i := 0; i < config.Pickups; i++ {
		room := rooms[candidates[g.rng.IntN(len(candidates))]]
		pos := room.RandomPoint(g.rng, 12)
		for tries := 0; tries < 10 && g.TooCloseToObjective(pos); tries++ {
			pos = room.RandomPoint(g.rng, 12)
		}
		pickup := &Pickup{Pos: pos, Kind: kinds[g.rng.IntN(len(kinds))]}
		pickup.Anim = NewAnimation(pickup.Kind, g.Art.Get("pickup_"+pickup.Kind), 0.4, true)
		g.Pickups = append(g.Pickups, pickup)
		g.Grid.RegisterPickup(pickup)
	}

	g.Projectiles = nil
	g.Decoys = nil
	g.Effects = nil
	g.ScanTimer = 0
	g.MessageTimer = 0
	g.CamPos = g.Player.Pos
	g.Camera = rl.NewCamera2D(rl.NewVector2(ScreenWidth/2, ScreenHeight/2), g.CamPos, 0, 1)
	g.UpdateCamera()
	g.Mode = Intro
	g.IntroTimer = 3
}

func (g *Game) TooCloseToObjective(pos rl.Vector2) bool {
	if rl.Vector2Distance(pos, g.ExitPos) < 70 {
		return true
	}
	for _, relay := range g.Relays {
		if rl.Vector2Distance(pos, relay.Pos) < 50 {
			return true
		}
	}
	return false
}

func (g *Game) NearestRoomCenter(pos rl.Vector2) rl.Vector2 {
	best := g.Dungeon.Rooms[0].Center()
	bestDistance := float32(-1)
	for _, room := range g.Dungeon.Rooms {
		distance := rl.Vector2Distance(pos, room.Center())
		if bestDistance < 0 || distance < bestDistance {
			best, bestDistance = room.Center(), distance
		}
	}
	return best
}

func (g *Game) ShowMessage(message string) {
	g.Message = message
	g.MessageTimer = 2.6
}

func (g *Game) AddEffect(pos rl.Vector2, start, end, duration float32, color rl.Color) {
	g.Effects = append(g.Effects, Effect{Pos: pos, Start: start, End: end, Duration: duration, Color: color})
}

func (g *Game) MakeNoise(pos rl.Vector2, radius float32) {
	g.AddEffect(pos, 8, radius, 0.7, rl.NewColor(244, 184, 79, 120))
	for _, ai := range g.Grid.GetNearbyDrones(pos, radius*1.5) {
		ai.HearNoise(pos, radius)
	}
}

func (g *Game) OnPlayerSpotted(spotter *DroneAI) {
	g.TimesSpotted++
	if g.AlertCooldown <= 0 {
		g.Sound.PlayVaried("alert", 0.95, 1.1)
		g.AlertCooldown = 1.2
	}
	g.AddEffect(spotter.Drone.Pos, 12, 70, 0.4, rl.NewColor(233, 87, 95, 255))
	for _, other := range g.Grid.GetNearbyDrones(spotter.Drone.Pos, 320) {
		if other != spotter && (other.State == Patrolling || other.State == Searching) {
			other.HearNoise(g.Player.Pos, 320)
		}
	}
}

func (g *Game) Throw(kind string, speed, fuse float32) {
	mouse := rl.GetScreenToWorld2D(rl.GetMousePosition(), g.Camera)
	direction := rl.Vector2Subtract(mouse, g.Player.Pos)
	if rl.Vector2Length(direction) < 1 {
		direction = rl.NewVector2(0, -1)
	}
	direction = rl.Vector2Normalize(direction)
	projectile := &Projectile{Kind: kind, Fuse: fuse}
	projectile.Pos = g.Player.Pos
	projectile.Radius = 6
	projectile.Vel = rl.Vector2Add(rl.Vector2Scale(direction, speed), rl.Vector2Scale(g.Player.Vel, 0.4))
	g.Projectiles = append(g.Projectiles, projectile)
	g.Sound.PlayVaried("throw", 0.9, 1.1)
}

func (g *Game) DetonateEMP(pos rl.Vector2) {
	const radius = 150
	g.Sound.Play("emp")
	g.Shake = 0.3
	g.AddEffect(pos, 10, radius, 0.5, rl.NewColor(75, 219, 202, 255))
	for _, ai := range g.Grid.GetNearbyDrones(pos, radius+g.Grid.GridSize) {
		if rl.Vector2Distance(pos, ai.Drone.Pos) <= radius+ai.Drone.Radius {
			ai.Stun(4.5)
			g.DronesStunned++
		}
	}
}

func (g *Game) PlaceDecoy(pos rl.Vector2) {
	decoy := &Decoy{Pos: pos, Life: 9}
	decoy.Anim = NewAnimation("decoy", g.Art.Get("decoy"), 0.3, true)
	g.Decoys = append(g.Decoys, decoy)
	g.ShowMessage("DECOY ACTIVE - drones will investigate the noise")
}

func (g *Game) AnyDroneIn(states ...AIState) bool {
	for _, ai := range g.AIs {
		for _, state := range states {
			if ai.State == state {
				return true
			}
		}
	}
	return false
}

func (g *Game) Update() {
	switch g.Mode {
	case Intro:
		g.IntroTimer -= FrameTime()
		g.UpdateCamera()
		skip := rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) || rl.IsMouseButtonPressed(rl.MouseButtonLeft)
		if g.IntroTimer <= 0 || skip {
			g.Mode = Playing
			g.ShowMessage("Repair the 3 relays, then reach the airlock")
		}
	case Playing:
		g.UpdatePlaying()
	}
}

func (g *Game) UpdatePlaying() {
	if rl.IsKeyPressed(rl.KeyEscape) {
		g.Mode = Paused
		return
	}
	dt := FrameTime()
	g.RunTime += dt
	g.MessageTimer -= dt
	g.AlertCooldown -= dt
	if g.ScanTimer > 0 {
		g.ScanTimer -= dt
	}
	if g.Shake > 0 {
		g.Shake -= dt
	}

	g.Player.Update(g)
	g.Player.HandleTools(g)
	g.UpdateExplored()
	g.UpdateInteraction()
	if g.Mode != Playing {
		return
	}
	g.UpdatePickups()

	alive := g.Projectiles[:0]
	for _, projectile := range g.Projectiles {
		projectile.Update(g)
		if !projectile.Done {
			alive = append(alive, projectile)
		}
	}
	g.Projectiles = alive

	active := g.Decoys[:0]
	for _, decoy := range g.Decoys {
		decoy.Update(g)
		if decoy.Life > 0 {
			active = append(active, decoy)
		}
	}
	g.Decoys = active

	effects := g.Effects[:0]
	for _, effect := range g.Effects {
		effect.Timer += dt
		if effect.Timer < effect.Duration {
			effects = append(effects, effect)
		}
	}
	g.Effects = effects

	for _, ai := range g.AIs {
		ai.Tick()
	}
	g.SeparateDrones()
	g.CheckDroneHits()
	g.Grid.UpdateDroneGridSliced(2)
	g.UpdateCamera()

	if g.Player.Health <= 0 {
		g.Deaths++
		g.Mode = Lost
		g.Sound.Play("hurt")
	}
}

func (g *Game) UpdateExplored() {
	x, y := g.Dungeon.TileAt(g.Player.Pos)
	if room := g.Dungeon.RoomAtTile(x, y); room >= 0 {
		g.Dungeon.Rooms[room].Explored = true
		return
	}
	for _, door := range g.Dungeon.Doors {
		if door.X == x && door.Y == y {
			g.Dungeon.Rooms[door.RoomA].Explored = true
			g.Dungeon.Rooms[door.RoomB].Explored = true
		}
	}
}

func (g *Game) NearbyRelay() int {
	for i, relay := range g.Relays {
		if !relay.Repaired && rl.Vector2Distance(g.Player.Pos, relay.Pos) < 52 {
			return i
		}
	}
	return -1
}

func (g *Game) NearExit() bool {
	return rl.Vector2Distance(g.Player.Pos, g.ExitPos) < 70
}

func (g *Game) UpdateInteraction() {
	dt := FrameTime()
	if index := g.NearbyRelay(); index >= 0 && rl.IsKeyDown(rl.KeyE) {
		relay := &g.Relays[index]
		relay.Progress += dt / 2.2
		g.WeldTimer -= dt
		if g.WeldTimer <= 0 {
			g.WeldTimer = 0.18
			g.Sound.PlayVaried("weld", 0.8, 1.25)
		}
		g.RepairNoise -= dt
		if g.RepairNoise <= 0 {
			g.RepairNoise = 0.7
			g.MakeNoise(relay.Pos, 300)
		}
		if relay.Progress >= 1 {
			relay.Repaired = true
			relay.Anim.ChangeAnimationState("fixed")
			g.Repaired++
			g.Sound.Play("repair")
			g.AddEffect(relay.Pos, 10, 90, 0.6, rl.NewColor(75, 219, 202, 255))
			if g.Repaired == len(g.Relays) {
				g.ExitAnim.ChangeAnimationState("open")
				g.ShowMessage("ALL RELAYS ONLINE - the airlock is unsealed!")
			} else {
				g.ShowMessage(fmt.Sprintf("Relay restored - %d remaining", len(g.Relays)-g.Repaired))
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyE) && g.NearExit() && g.NearbyRelay() < 0 {
		if g.Repaired == len(g.Relays) {
			g.CompleteSector()
		} else {
			g.ShowMessage(fmt.Sprintf("AIRLOCK SEALED - %d relays still offline", len(g.Relays)-g.Repaired))
		}
	}
}

func (g *Game) CompleteSector() {
	g.Sound.Play("escape")
	if g.Sector == len(Sectors) {
		g.Mode = Won
		if g.BestTime == 0 || g.RunTime < g.BestTime {
			g.BestTime = g.RunTime
			g.NewRecord = true
			SaveBestTime(g.RunTime)
		}
		return
	}
	g.Sector++
	g.Player.Heal(25)
	g.StartSector()
	g.SaveSnapshot()
}

func (g *Game) UpdatePickups() {
	for _, pickup := range g.Grid.GetNearbyPickups(g.Player.Pos, 40) {
		if pickup.Taken || rl.Vector2Distance(pickup.Pos, g.Player.Pos) > g.Player.Radius+14 {
			continue
		}
		collected := false
		label := ""
		switch pickup.Kind {
		case "emp":
			collected, label = g.Player.Tools[0].AddUses(1), "+1 EMP CHARGE"
		case "decoy":
			collected, label = g.Player.Tools[1].AddUses(1), "+1 DECOY BEACON"
		case "scanner":
			collected, label = g.Player.Tools[2].AddUses(1), "+1 SCANNER CHARGE"
		case "medkit":
			collected, label = g.Player.Heal(35), "+35 SUIT INTEGRITY"
		}
		if !collected {
			continue
		}
		pickup.Taken = true
		g.Grid.RemovePickup(pickup)
		g.Sound.Play("pickup")
		g.ShowMessage(label)
		g.AddEffect(pickup.Pos, 4, 30, 0.35, rl.NewColor(211, 232, 239, 255))
	}
}

func (g *Game) SeparateDrones() {
	for _, ai := range g.AIs {
		for _, other := range g.Grid.GetNearbyDrones(ai.Drone.Pos, 40) {
			if other != ai {
				ai.Drone.PushOutOf(&other.Drone.PhysicsBody, &g.Dungeon)
			}
		}
	}
}

func (g *Game) CheckDroneHits() {
	for _, ai := range g.AIs {
		if ai.State == Stunned || ai.Drone.HitCooldown > 0 {
			continue
		}
		touching := rl.Vector2Distance(g.Player.Pos, ai.Drone.Pos) < g.Player.Radius+ai.Drone.Radius
		if !touching || g.Player.DamageTimer > 0 {
			continue
		}
		g.Player.TakeDamage(ai.Drone.Damage)
		g.DamageTaken += ai.Drone.Damage
		g.Player.Bounce(&ai.Drone.PhysicsBody, 430)
		ai.Drone.HitCooldown = 1.2
		g.Sound.PlayVaried("hurt", 0.9, 1.1)
		g.Shake = 0.35
		g.ShowMessage("HULL BREACH - get away from the drones!")
		if ai.State != Chasing {
			ai.LastSeen = g.Player.Pos
			ai.SetState(Chasing)
		}
	}
}

func (g *Game) UpdateCamera() {
	follow := min(1, FrameTime()*8)
	g.CamPos = rl.Vector2Lerp(g.CamPos, g.Player.Pos, follow)

	worldW := float32(g.Dungeon.Width * TILE)
	worldH := float32(g.Dungeon.Height * TILE)
	target := g.CamPos
	target.X = max(ScreenWidth/2, min(target.X, worldW-ScreenWidth/2))
	target.Y = max(ScreenHeight/2-70, min(target.Y, worldH-ScreenHeight/2+50))
	g.Camera.Target = rl.NewVector2(float32(int32(target.X)), float32(int32(target.Y)))

	offset := rl.NewVector2(ScreenWidth/2, ScreenHeight/2)
	if g.Shake > 0 {
		offset.X += (rand.Float32()*2 - 1) * g.Shake * 18
		offset.Y += (rand.Float32()*2 - 1) * g.Shake * 18
	}
	g.Camera.Offset = offset
}

func (g *Game) ViewRect() rl.Rectangle {
	topLeft := rl.GetScreenToWorld2D(rl.NewVector2(0, 0), g.Camera)
	return rl.NewRectangle(topLeft.X-TILE, topLeft.Y-TILE, ScreenWidth+TILE*2, ScreenHeight+TILE*2)
}
