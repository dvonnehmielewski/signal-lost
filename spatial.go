package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type SpatialGrid struct {
	GridSize    float32
	DroneGrid   map[rl.Vector2][]*DroneAI
	PickupGrid  map[rl.Vector2][]*Pickup
	Drones      []*DroneAI
	UpdateIndex int
}

func NewSpatialGrid(gridSize float32) SpatialGrid {
	return SpatialGrid{
		GridSize:   gridSize,
		DroneGrid:  make(map[rl.Vector2][]*DroneAI),
		PickupGrid: make(map[rl.Vector2][]*Pickup),
	}
}

func (s *SpatialGrid) CalculateKey(pos rl.Vector2) rl.Vector2 {
	return rl.Vector2{
		X: float32(math.Floor(float64(pos.X / s.GridSize))),
		Y: float32(math.Floor(float64(pos.Y / s.GridSize))),
	}
}

func GetNearby[T any](s *SpatialGrid, grid map[rl.Vector2][]T, pos rl.Vector2, radius float32) []T {
	cellRadius := int(math.Ceil(float64(radius / s.GridSize)))
	centerKey := s.CalculateKey(pos)
	nearby := make([]T, 0)
	for x := -cellRadius; x <= cellRadius; x++ {
		for y := -cellRadius; y <= cellRadius; y++ {
			key := rl.Vector2{X: centerKey.X + float32(x), Y: centerKey.Y + float32(y)}
			if things, exists := grid[key]; exists {
				nearby = append(nearby, things...)
			}
		}
	}
	return nearby
}

func (s *SpatialGrid) GetNearbyDrones(pos rl.Vector2, radius float32) []*DroneAI {
	return GetNearby(s, s.DroneGrid, pos, radius)
}

func (s *SpatialGrid) GetNearbyPickups(pos rl.Vector2, radius float32) []*Pickup {
	return GetNearby(s, s.PickupGrid, pos, radius)
}

func (s *SpatialGrid) RegisterDrone(ai *DroneAI) {
	key := s.CalculateKey(ai.Drone.Pos)
	ai.Drone.GridKey = key
	s.DroneGrid[key] = append(s.DroneGrid[key], ai)
	s.Drones = append(s.Drones, ai)
}

func (s *SpatialGrid) RegisterPickup(p *Pickup) {
	key := s.CalculateKey(p.Pos)
	s.PickupGrid[key] = append(s.PickupGrid[key], p)
}

func RemoveFromCell[T comparable](grid map[rl.Vector2][]T, key rl.Vector2, thing T) {
	cell := grid[key]
	for i, other := range cell {
		if other == thing {
			cell = append(cell[:i], cell[i+1:]...)
			break
		}
	}
	if len(cell) == 0 {
		delete(grid, key)
	} else {
		grid[key] = cell
	}
}

func (s *SpatialGrid) RemovePickup(p *Pickup) {
	RemoveFromCell(s.PickupGrid, s.CalculateKey(p.Pos), p)
}

func (s *SpatialGrid) UpdateDronePosition(ai *DroneAI) {
	newKey := s.CalculateKey(ai.Drone.Pos)
	if newKey == ai.Drone.GridKey {
		return
	}
	RemoveFromCell(s.DroneGrid, ai.Drone.GridKey, ai)
	s.DroneGrid[newKey] = append(s.DroneGrid[newKey], ai)
	ai.Drone.GridKey = newKey
}

func (s *SpatialGrid) UpdateDroneGridSliced(perFrame int) {
	if len(s.Drones) == 0 {
		return
	}
	for i := 0; i < perFrame; i++ {
		s.UpdateDronePosition(s.Drones[s.UpdateIndex])
		s.UpdateIndex++
		if s.UpdateIndex >= len(s.Drones) {
			s.UpdateIndex = 0
		}
	}
}
