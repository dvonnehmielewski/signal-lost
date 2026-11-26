package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type ToolBehavior interface {
	Use(t *Tool, g *Game) bool
	Description() string
}

type Tool struct {
	Name        string
	MaxUses     int
	CurrentUses int
	Color       rl.Color
	Icon        Animation
	Behavior    ToolBehavior
}

func NewTool(name string, maxUses, startUses int, color rl.Color, icon Animation, tb ToolBehavior) Tool {
	return Tool{
		Name:        name,
		MaxUses:     maxUses,
		CurrentUses: startUses,
		Color:       color,
		Icon:        icon,
		Behavior:    tb,
	}
}

func (t *Tool) TryUse(g *Game) {
	if t.CurrentUses <= 0 {
		g.ShowMessage(t.Name + " EMPTY - search the station for more")
		return
	}
	if t.Behavior.Use(t, g) {
		t.CurrentUses--
	}
}

func (t *Tool) AddUses(amount int) bool {
	if t.CurrentUses >= t.MaxUses {
		return false
	}
	t.CurrentUses = min(t.CurrentUses+amount, t.MaxUses)
	return true
}

type EMPBehavior struct{}

func (e EMPBehavior) Use(t *Tool, g *Game) bool {
	g.Throw("emp", 520, 1.1)
	return true
}

func (e EMPBehavior) Description() string {
	return "Bouncing pulse - stuns drones"
}

type DecoyBehavior struct{}

func (d DecoyBehavior) Use(t *Tool, g *Game) bool {
	g.Throw("decoy", 420, 0.8)
	return true
}

func (d DecoyBehavior) Description() string {
	return "Noisy beacon - lures drones"
}

type ScannerBehavior struct{}

func (s ScannerBehavior) Use(t *Tool, g *Game) bool {
	if g.ScanTimer > 0 {
		return false
	}
	g.ScanTimer = 8
	g.Sound.Play("scan")
	g.AddEffect(g.Player.Pos, 0, 600, 0.9, rl.NewColor(75, 219, 202, 255))
	g.ShowMessage("SCAN ACTIVE - drones and relays revealed")
	return true
}

func (s ScannerBehavior) Description() string {
	return "Reveals the whole sector"
}

type Projectile struct {
	PhysicsBody
	Kind string
	Fuse float32
	Done bool
}

func (p *Projectile) Update(g *Game) {
	hitX, hitY := p.VelocityTick(&g.Dungeon)
	if hitX || hitY {
		p.BounceOffWalls(hitX, hitY, 0.65)
		if rl.Vector2Length(p.Vel) > 60 {
			g.Sound.PlayVaried("ping", 0.8, 1.2)
		}
	}
	p.ApplyDrag(1.4)
	p.Fuse -= FrameTime()
	if p.Fuse <= 0 {
		p.Done = true
		if p.Kind == "emp" {
			g.DetonateEMP(p.Pos)
		} else {
			g.PlaceDecoy(p.Pos)
		}
	}
}

func (p *Projectile) Draw(art *Art) {
	if p.Kind == "emp" {
		pulse := 0.5 + 0.5*float32(int(p.Fuse*12)%2)
		rl.DrawCircleV(p.Pos, p.Radius+4, rl.Fade(rl.NewColor(75, 219, 202, 255), 0.25*pulse))
		rl.DrawCircleV(p.Pos, p.Radius, rl.NewColor(20, 60, 70, 255))
		rl.DrawCircleLines(int32(p.Pos.X), int32(p.Pos.Y), p.Radius, rl.NewColor(75, 219, 202, 255))
		return
	}
	texture := art.Get("decoy")
	rl.DrawTexturePro(texture, rl.NewRectangle(0, 0, 24, 24), rl.NewRectangle(p.Pos.X, p.Pos.Y, 24, 24), rl.NewVector2(12, 12), 0, rl.White)
}

type Decoy struct {
	Pos       rl.Vector2
	Life      float32
	PingTimer float32
	Anim      Animation
}

func (d *Decoy) Update(g *Game) {
	d.Life -= FrameTime()
	d.PingTimer -= FrameTime()
	if d.PingTimer <= 0 && d.Life > 0 {
		d.PingTimer = 1.2
		g.Sound.PlayVaried("decoy", 0.95, 1.05)
		g.MakeNoise(d.Pos, 360)
	}
}

type Pickup struct {
	Pos   rl.Vector2
	Kind  string
	Taken bool
	Bob   float32
	Anim  Animation
}

func (p *Pickup) Draw() {
	p.Bob += FrameTime() * 3
	offset := float32(0)
	if int(p.Bob)%2 == 0 {
		offset = -1
	}
	p.Anim.UpdateTime()
	p.Anim.DrawAnimation(rl.NewVector2(p.Pos.X, p.Pos.Y+offset), 24, 0, rl.White)
}

type Effect struct {
	Pos      rl.Vector2
	Start    float32
	End      float32
	Duration float32
	Timer    float32
	Color    rl.Color
}

func (e *Effect) Draw() {
	t := e.Timer / e.Duration
	radius := e.Start + (e.End-e.Start)*t
	rl.DrawRing(e.Pos, radius-3, radius, 0, 360, 48, rl.Fade(e.Color, 0.8*(1-t)))
	rl.DrawCircleV(e.Pos, radius, rl.Fade(e.Color, 0.08*(1-t)))
}
