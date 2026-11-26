package main

import (
	"math"
	"math/rand/v2"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const TILE = 40

type Block int

const (
	NONE  Block = 0
	FLOOR Block = 1
	WALL  Block = 2
	DOOR  Block = 3
)

type Room struct {
	PivotX   int
	PivotY   int
	Width    int
	Height   int
	Explored bool
}

func (r Room) Center() rl.Vector2 {
	return rl.NewVector2(float32(r.PivotX*TILE)+float32(r.Width*TILE)/2, float32(r.PivotY*TILE)+float32(r.Height*TILE)/2)
}

func (r Room) ContainsTile(x, y int) bool {
	return x > r.PivotX && x < r.PivotX+r.Width-1 && y > r.PivotY && y < r.PivotY+r.Height-1
}

func (r Room) SafeArea(margin float32) rl.Rectangle {
	x := float32((r.PivotX+2)*TILE) + margin
	y := float32((r.PivotY+2)*TILE) + margin
	w := float32((r.Width-4)*TILE) - margin*2
	h := float32((r.Height-4)*TILE) - margin*2
	return rl.NewRectangle(x, y, w, h)
}

func (r Room) RandomPoint(rng *rand.Rand, margin float32) rl.Vector2 {
	area := r.SafeArea(margin)
	return rl.NewVector2(area.X+rng.Float32()*area.Width, area.Y+rng.Float32()*area.Height)
}

type Door struct {
	X        int
	Y        int
	RoomA    int
	RoomB    int
	Vertical bool
}

func (d Door) Center() rl.Vector2 {
	return TileCenter(d.X, d.Y)
}

func (d Door) SidePoint(dungeon *Dungeon, room int) rl.Vector2 {
	if d.Vertical {
		if dungeon.Rooms[room].ContainsTile(d.X-1, d.Y) {
			return TileCenter(d.X-1, d.Y)
		}
		return TileCenter(d.X+1, d.Y)
	}
	if dungeon.Rooms[room].ContainsTile(d.X, d.Y-1) {
		return TileCenter(d.X, d.Y-1)
	}
	return TileCenter(d.X, d.Y+1)
}

type Dungeon struct {
	Blocks         [][]Block
	Variant        [][]int
	Width          int
	Height         int
	Rooms          []Room
	Doors          []Door
	FloorPositions []rl.Vector2
}

func TileCenter(x, y int) rl.Vector2 {
	return rl.NewVector2(float32(x*TILE+TILE/2), float32(y*TILE+TILE/2))
}

func RandInt(rng *rand.Rand, min, max int) int {
	return min + rng.IntN(max-min+1)
}

func NewDungeon(width, height int) Dungeon {
	dungeon := Dungeon{Width: width, Height: height}
	dungeon.Blocks = make([][]Block, width)
	dungeon.Variant = make([][]int, width)
	for x := 0; x < width; x++ {
		dungeon.Blocks[x] = make([]Block, height)
		dungeon.Variant[x] = make([]int, height)
	}
	dungeon.FloorPositions = make([]rl.Vector2, 0)
	return dungeon
}

func (d *Dungeon) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < d.Width && y < d.Height
}

func (d *Dungeon) Get(x, y int) Block {
	if !d.InBounds(x, y) {
		return NONE
	}
	return d.Blocks[x][y]
}

func (d *Dungeon) PlaceBlock(block Block, x, y int) {
	d.Blocks[x][y] = block
	if block == FLOOR {
		d.FloorPositions = append(d.FloorPositions, rl.NewVector2(float32(x), float32(y)))
	}
}

func (d *Dungeon) StampRoom(room Room, softCorners bool) {
	for x := 0; x < room.Width; x++ {
		for y := 0; y < room.Height; y++ {
			bx, by := x+room.PivotX, y+room.PivotY
			if x == 0 || y == 0 || x == room.Width-1 || y == room.Height-1 {
				if d.Blocks[bx][by] == NONE {
					d.PlaceBlock(WALL, bx, by)
				}
			} else {
				d.PlaceBlock(FLOOR, bx, by)
			}
		}
	}
	if softCorners {
		d.Blocks[room.PivotX+1][room.PivotY+1] = WALL
		d.Blocks[room.PivotX+room.Width-2][room.PivotY+1] = WALL
		d.Blocks[room.PivotX+1][room.PivotY+room.Height-2] = WALL
		d.Blocks[room.PivotX+room.Width-2][room.PivotY+room.Height-2] = WALL
	}
}

func (d *Dungeon) RoomFits(room Room) bool {
	if room.PivotX < 1 || room.PivotY < 1 || room.PivotX+room.Width >= d.Width || room.PivotY+room.Height >= d.Height {
		return false
	}
	for x := 0; x < room.Width; x++ {
		for y := 0; y < room.Height; y++ {
			block := d.Blocks[x+room.PivotX][y+room.PivotY]
			edge := x == 0 || y == 0 || x == room.Width-1 || y == room.Height-1
			if edge && block != NONE && block != WALL {
				return false
			}
			if !edge && block != NONE {
				return false
			}
		}
	}
	return true
}

func (d *Dungeon) Generate(seed1, seed2 uint64, targetRooms int) {
	drand := rand.New(rand.NewPCG(seed1, seed2))

	first := Room{Width: RandInt(drand, 8, 11), Height: RandInt(drand, 7, 9)}
	first.PivotX = d.Width/2 - first.Width/2 + RandInt(drand, -4, 4)
	first.PivotY = d.Height/2 - first.Height/2 + RandInt(drand, -3, 3)
	d.StampRoom(first, false)
	d.Rooms = append(d.Rooms, first)

	for attempts := 0; len(d.Rooms) < targetRooms && attempts < 3000; attempts++ {
		tempRoom := Room{Width: RandInt(drand, 6, 12), Height: RandInt(drand, 6, 11)}
		sourceIndex := drand.IntN(len(d.Rooms))
		source := d.Rooms[sourceIndex]

		var doorX, doorY int
		vertical := false
		switch drand.IntN(4) {
		case 0:
			doorX = RandInt(drand, source.PivotX+2, source.PivotX+source.Width-3)
			doorY = source.PivotY
			tempRoom.PivotY = doorY - (tempRoom.Height - 1)
			tempRoom.PivotX = doorX - RandInt(drand, 2, tempRoom.Width-3)
		case 1:
			doorX = RandInt(drand, source.PivotX+2, source.PivotX+source.Width-3)
			doorY = source.PivotY + source.Height - 1
			tempRoom.PivotY = doorY
			tempRoom.PivotX = doorX - RandInt(drand, 2, tempRoom.Width-3)
		case 2:
			doorX = source.PivotX
			doorY = RandInt(drand, source.PivotY+2, source.PivotY+source.Height-3)
			tempRoom.PivotX = doorX - (tempRoom.Width - 1)
			tempRoom.PivotY = doorY - RandInt(drand, 2, tempRoom.Height-3)
			vertical = true
		case 3:
			doorX = source.PivotX + source.Width - 1
			doorY = RandInt(drand, source.PivotY+2, source.PivotY+source.Height-3)
			tempRoom.PivotX = doorX
			tempRoom.PivotY = doorY - RandInt(drand, 2, tempRoom.Height-3)
			vertical = true
		}

		if !d.RoomFits(tempRoom) {
			continue
		}
		soft := tempRoom.Width >= 7 && tempRoom.Height >= 7 && drand.IntN(3) == 0
		d.StampRoom(tempRoom, soft)
		d.Blocks[doorX][doorY] = DOOR
		d.Rooms = append(d.Rooms, tempRoom)
		d.Doors = append(d.Doors, Door{X: doorX, Y: doorY, RoomA: sourceIndex, RoomB: len(d.Rooms) - 1, Vertical: vertical})
	}

	d.AddLoops(drand)

	for x := 0; x < d.Width; x++ {
		for y := 0; y < d.Height; y++ {
			roll := drand.IntN(100)
			switch {
			case roll < 76:
				d.Variant[x][y] = 0
			case roll < 84:
				d.Variant[x][y] = 1
			case roll < 92:
				d.Variant[x][y] = 2
			default:
				d.Variant[x][y] = 3
			}
		}
	}
}

func (d *Dungeon) AddLoops(drand *rand.Rand) {
	for x := 1; x < d.Width-1; x++ {
		for y := 1; y < d.Height-1; y++ {
			if d.Blocks[x][y] != WALL {
				continue
			}
			vertical := d.Blocks[x-1][y] == FLOOR && d.Blocks[x+1][y] == FLOOR && d.Blocks[x][y-1] == WALL && d.Blocks[x][y+1] == WALL
			horizontal := d.Blocks[x][y-1] == FLOOR && d.Blocks[x][y+1] == FLOOR && d.Blocks[x-1][y] == WALL && d.Blocks[x+1][y] == WALL
			if !vertical && !horizontal {
				continue
			}
			var a, b int
			if vertical {
				a, b = d.RoomAtTile(x-1, y), d.RoomAtTile(x+1, y)
			} else {
				a, b = d.RoomAtTile(x, y-1), d.RoomAtTile(x, y+1)
			}
			if a < 0 || b < 0 || a == b || d.Connected(a, b) || drand.IntN(100) >= 45 {
				continue
			}
			if !d.DoorFitsWall(x, y, vertical, a) || !d.DoorFitsWall(x, y, vertical, b) {
				continue
			}
			d.Blocks[x][y] = DOOR
			d.Doors = append(d.Doors, Door{X: x, Y: y, RoomA: a, RoomB: b, Vertical: vertical})
		}
	}
}

func (d *Dungeon) DoorFitsWall(x, y int, vertical bool, room int) bool {
	r := d.Rooms[room]
	if vertical {
		return y >= r.PivotY+2 && y <= r.PivotY+r.Height-3
	}
	return x >= r.PivotX+2 && x <= r.PivotX+r.Width-3
}

func (d *Dungeon) Connected(a, b int) bool {
	for _, door := range d.Doors {
		if (door.RoomA == a && door.RoomB == b) || (door.RoomA == b && door.RoomB == a) {
			return true
		}
	}
	return false
}

func (d *Dungeon) DoorBetween(a, b int) (Door, bool) {
	for _, door := range d.Doors {
		if (door.RoomA == a && door.RoomB == b) || (door.RoomA == b && door.RoomB == a) {
			return door, true
		}
	}
	return Door{}, false
}

func (d *Dungeon) Neighbors(room int) []int {
	neighbors := make([]int, 0)
	for _, door := range d.Doors {
		if door.RoomA == room {
			neighbors = append(neighbors, door.RoomB)
		} else if door.RoomB == room {
			neighbors = append(neighbors, door.RoomA)
		}
	}
	return neighbors
}

func (d *Dungeon) RoomAtTile(x, y int) int {
	for i, room := range d.Rooms {
		if room.ContainsTile(x, y) {
			return i
		}
	}
	return -1
}

func (d *Dungeon) TileAt(pos rl.Vector2) (int, int) {
	return int(math.Floor(float64(pos.X / TILE))), int(math.Floor(float64(pos.Y / TILE)))
}

func (d *Dungeon) RoomAt(pos rl.Vector2) int {
	x, y := d.TileAt(pos)
	return d.RoomAtTile(x, y)
}

func (d *Dungeon) Solid(x, y int) bool {
	block := d.Get(x, y)
	return block == WALL || block == NONE
}

func (d *Dungeon) CircleBlocked(pos rl.Vector2, radius float32) bool {
	body := rl.NewRectangle(pos.X-radius, pos.Y-radius, radius*2, radius*2)
	minX, minY := d.TileAt(rl.NewVector2(body.X, body.Y))
	maxX, maxY := d.TileAt(rl.NewVector2(body.X+body.Width, body.Y+body.Height))
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			if !d.Solid(x, y) {
				continue
			}
			tile := rl.NewRectangle(float32(x*TILE), float32(y*TILE), TILE, TILE)
			if rl.CheckCollisionRecs(body, tile) {
				return true
			}
		}
	}
	return false
}

func (d *Dungeon) LineOfSight(from, to rl.Vector2) bool {
	distance := rl.Vector2Distance(from, to)
	steps := int(distance/(TILE/4)) + 1
	for i := 1; i < steps; i++ {
		point := rl.Vector2Lerp(from, to, float32(i)/float32(steps))
		x, y := d.TileAt(point)
		if d.Solid(x, y) {
			return false
		}
	}
	return true
}

func (d *Dungeon) Draw(art *Art, view rl.Rectangle) {
	floor := art.Get("floor")
	wall := art.Get("wall")
	door := art.Get("door")
	minX, minY := d.TileAt(rl.NewVector2(view.X, view.Y))
	maxX, maxY := d.TileAt(rl.NewVector2(view.X+view.Width, view.Y+view.Height))
	for x := max(minX, 0); x <= min(maxX, d.Width-1); x++ {
		for y := max(minY, 0); y <= min(maxY, d.Height-1); y++ {
			pos := rl.NewVector2(float32(x*TILE), float32(y*TILE))
			switch d.Blocks[x][y] {
			case FLOOR:
				source := rl.NewRectangle(float32(d.Variant[x][y]*TILE), 0, TILE, TILE)
				rl.DrawTextureRec(floor, source, pos, rl.White)
			case WALL:
				rl.DrawTexture(wall, int32(pos.X), int32(pos.Y), rl.White)
			case DOOR:
				frame := float32(0)
				if d.Solid(x, y-1) {
					frame = 1
				}
				rl.DrawTextureRec(door, rl.NewRectangle(frame*TILE, 0, TILE, TILE), pos, rl.White)
			}
		}
	}
}
