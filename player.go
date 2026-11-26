package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Player struct {
	PhysicsBody
	Accel       float32
	MaxSpeed    float32
	BoostSpeed  float32
	Health      int
	MaxHealth   int
	Stamina     float32
	Boosting    bool
	DamageTimer float32
	NoiseTimer  float32
	Facing      float32
	Anim        AnimationFSM
	Tools       []Tool
	Selected    int
}

func NewPlayer(pos rl.Vector2, art *Art) Player {
	player := Player{
		Accel:      1300,
		MaxSpeed:   175,
		BoostSpeed: 310,
		Health:     100,
		MaxHealth:  100,
		Stamina:    1,
	}
	player.Pos = pos
	player.Radius = 12
	player.Anim = NewAnimationFSM()
	player.Anim.AddAnimation(NewAnimation("idle", art.Get("player_idle"), 0.6, true))
	player.Anim.AddAnimation(NewAnimation("move", art.Get("player_move"), 0.08, true))
	player.Tools = []Tool{
		NewTool("EMP", 5, 2, rl.NewColor(75, 219, 202, 255), NewAnimation("emp", art.Get("pickup_emp"), 0.4, true), EMPBehavior{}),
		NewTool("DECOY", 3, 1, rl.NewColor(244, 184, 79, 255), NewAnimation("decoy", art.Get("pickup_decoy"), 0.4, true), DecoyBehavior{}),
		NewTool("SCANNER", 3, 1, rl.NewColor(75, 219, 202, 255), NewAnimation("scanner", art.Get("pickup_scanner"), 0.4, true), ScannerBehavior{}),
	}
	return player
}

func (p *Player) Update(g *Game) {
	input := rl.NewVector2(0, 0)
	if rl.IsKeyDown(rl.KeyA) || rl.IsKeyDown(rl.KeyLeft) {
		input.X -= 1
	}
	if rl.IsKeyDown(rl.KeyD) || rl.IsKeyDown(rl.KeyRight) {
		input.X += 1
	}
	if rl.IsKeyDown(rl.KeyW) || rl.IsKeyDown(rl.KeyUp) {
		input.Y -= 1
	}
	if rl.IsKeyDown(rl.KeyS) || rl.IsKeyDown(rl.KeyDown) {
		input.Y += 1
	}
	moving := input.X != 0 || input.Y != 0

	wantsBoost := (rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)) && moving
	if wantsBoost && p.Stamina > 0.05 {
		p.Boosting = true
	} else if !wantsBoost || p.Stamina <= 0 {
		p.Boosting = false
	}

	topSpeed := p.MaxSpeed
	if p.Boosting {
		topSpeed = p.BoostSpeed
		p.Stamina -= FrameTime() * 0.45
		p.NoiseTimer -= FrameTime()
		if p.NoiseTimer <= 0 {
			p.NoiseTimer = 0.35
			g.MakeNoise(p.Pos, 280)
		}
	} else {
		p.Stamina += FrameTime() * 0.22
	}
	p.Stamina = float32(math.Max(0, math.Min(1, float64(p.Stamina))))

	if moving {
		acceleration := rl.Vector2Scale(rl.Vector2Normalize(input), p.Accel*FrameTime())
		p.Vel = rl.Vector2Add(p.Vel, acceleration)
		p.Anim.ChangeAnimationState("move")
	} else {
		p.Anim.ChangeAnimationState("idle")
	}
	p.ApplyDrag(7)
	if p.DamageTimer <= 0.6 {
		p.LimitSpeed(topSpeed)
	}

	hitX, hitY := p.VelocityTick(&g.Dungeon)
	p.BounceOffWalls(hitX, hitY, 0.2)

	if rl.Vector2Length(p.Vel) > 20 {
		p.Facing = float32(math.Atan2(float64(p.Vel.Y), float64(p.Vel.X))*180/math.Pi) + 90
	}
	if p.DamageTimer > 0 {
		p.DamageTimer -= FrameTime()
	}
}

func (p *Player) HandleTools(g *Game) {
	for i := range p.Tools {
		if rl.IsKeyPressed(int32(rl.KeyOne + i)) {
			p.Selected = i
		}
	}
	wheel := rl.GetMouseWheelMove()
	if wheel > 0 {
		p.Selected = (p.Selected + len(p.Tools) - 1) % len(p.Tools)
	} else if wheel < 0 {
		p.Selected = (p.Selected + 1) % len(p.Tools)
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) || rl.IsKeyPressed(rl.KeySpace) {
		p.Tools[p.Selected].TryUse(g)
	}
}

func (p *Player) TakeDamage(amount int) {
	p.Health -= amount
	if p.Health < 0 {
		p.Health = 0
	}
	p.DamageTimer = 1.0
}

func (p *Player) Heal(amount int) bool {
	if p.Health >= p.MaxHealth {
		return false
	}
	p.Health = min(p.Health+amount, p.MaxHealth)
	return true
}

func (p *Player) Draw() {
	if p.DamageTimer > 0 && int(p.DamageTimer*12)%2 == 1 {
		return
	}
	rl.DrawCircleV(rl.NewVector2(p.Pos.X+3, p.Pos.Y+5), p.Radius+1, rl.NewColor(0, 0, 0, 80))
	if p.Boosting {
		rl.DrawCircleV(p.Pos, p.Radius+9, rl.Fade(rl.NewColor(244, 184, 79, 255), 0.15))
	}
	p.Anim.DrawWithFSM(p.Pos, 34, p.Facing, rl.White)
}
