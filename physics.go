package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type PhysicsBody struct {
	Pos    rl.Vector2
	Vel    rl.Vector2
	Radius float32
}

func (pb *PhysicsBody) VelocityTick(d *Dungeon) (hitX bool, hitY bool) {
	adjustedVel := rl.Vector2Scale(pb.Vel, FrameTime())

	nextX := rl.NewVector2(pb.Pos.X+adjustedVel.X, pb.Pos.Y)
	if d.CircleBlocked(nextX, pb.Radius) {
		hitX = true
	} else {
		pb.Pos.X = nextX.X
	}

	nextY := rl.NewVector2(pb.Pos.X, pb.Pos.Y+adjustedVel.Y)
	if d.CircleBlocked(nextY, pb.Radius) {
		hitY = true
	} else {
		pb.Pos.Y = nextY.Y
	}
	return hitX, hitY
}

func (pb *PhysicsBody) ApplyDrag(drag float32) {
	factor := 1 - drag*FrameTime()
	if factor < 0 {
		factor = 0
	}
	pb.Vel = rl.Vector2Scale(pb.Vel, factor)
}

func (pb *PhysicsBody) LimitSpeed(maxSpeed float32) {
	if rl.Vector2Length(pb.Vel) > maxSpeed {
		pb.Vel = rl.Vector2Scale(rl.Vector2Normalize(pb.Vel), maxSpeed)
	}
}

func (pb *PhysicsBody) BounceOffWalls(hitX, hitY bool, restitution float32) {
	if hitX {
		pb.Vel.X = -pb.Vel.X * restitution
	}
	if hitY {
		pb.Vel.Y = -pb.Vel.Y * restitution
	}
}

func (pb *PhysicsBody) Bounce(other *PhysicsBody, speed float32) {
	away := rl.Vector2Normalize(rl.Vector2Subtract(pb.Pos, other.Pos))
	pb.Vel = rl.Vector2Scale(away, speed)
}

func (pb *PhysicsBody) PushOutOf(other *PhysicsBody, d *Dungeon) {
	distance := rl.Vector2Distance(pb.Pos, other.Pos)
	overlap := pb.Radius + other.Radius - distance
	if overlap <= 0 || distance == 0 {
		return
	}
	teleportDir := rl.Vector2Normalize(rl.Vector2Subtract(pb.Pos, other.Pos))
	target := rl.Vector2Add(pb.Pos, rl.Vector2Scale(teleportDir, overlap/2))
	if !d.CircleBlocked(target, pb.Radius) {
		pb.Pos = target
	}
}
