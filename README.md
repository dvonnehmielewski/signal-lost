# Signal Lost

Final project for Game Architecture.

A top-down stealth game written in Go with [raylib](https://www.raylib.com/). You are the last
technician on a dead space station. Sneak past security drones, repair three signal relays in each
sector, and reach the airlock before your suit gives out.

Every run is a new station: layouts are generated from a seed, and three sectors scale up in size,
room count and drone threat.

## Play

```bash
go run .
```

Requires Go 1.23+. raylib is loaded through [raylib-go](https://github.com/gen2brain/raylib-go);
on macOS no extra setup is needed. Art and audio are embedded with `go:embed`, so `go build`
produces a single self-contained binary.

| Action | Keys |
|---|---|
| Move | WASD or arrow keys |
| Boost (uses stamina) | Shift |
| Select tool | 1 EMP, 2 Decoy, 3 Scanner |
| Use tool | Left click or Space |
| Repair relay, open airlock | Hold / press E |
| Pause | Esc |

## How it's built

About 2,600 lines of Go across focused files, with no game engine beyond raylib for drawing,
input and audio.

**Procedural station generation** (`dungeon.go`)
Rooms grow outward from a seed room: each new room is attached to a random wall of an existing one
and kept only if it fits without overlap. A second pass punches extra doors between adjacent,
unconnected rooms (45% chance each), turning the tree into a graph with loops so the player always
has more than one way around a drone. Generation is fully deterministic from a PCG seed, so a run
can be replayed exactly.

**Drone AI** (`drone.go`)
Each drone runs a five-state machine: patrolling, investigating, chasing, searching and stunned.
Drones see the player through ray-marched line of sight within a sight range, hear noises
(repairs, decoys, boosting) scaled by drone type, and alert nearby drones when they spot you.
Movement uses a room/door graph: a direct line when visible, otherwise a route through shared doors
up to two rooms away, with stuck detection to recover from corners. Sentinels see farther; Hunters
hear better and chase faster, and both speed up each sector.

**Spatial hashing** (`spatial.go`)
Drones and pickups live in a uniform-grid spatial hash written with Go generics, so noise,
EMP blasts, drone separation and pickup checks query only nearby cells instead of every entity.
Drone cell membership is re-bucketed in time slices (a couple of drones per frame) to keep the
per-frame cost flat as drone counts grow.

**Physics** (`physics.go`)
Circle-vs-tile collision is resolved one axis at a time, which gives smooth sliding along walls.
Bodies support drag, speed limits, restitution bounces (for thrown EMPs) and overlap separation.

**Tools as strategies** (`tools.go`)
EMP, decoy and scanner share a `ToolBehavior` interface, so adding a new tool means writing one
type with `Use` and `Description`.

**Rendering and feel** (`draw.go`, `assets.go`)
Only tiles inside the camera view are drawn. Unexplored rooms stay under fog until entered (or
until the scanner reveals them), a minimap tracks explored rooms, and sprite animations are driven
by a small animation state machine. Camera shake, pitch-varied sound effects and separate ambient
and mission music tracks round it out. Best run time is saved to the user config directory.

## Project layout

```
main.go      window, audio, screen flow (menu, help, game, pause, win/lose)
game.go      sector setup, game loop, interactions, scoring
dungeon.go   procedural generation, collision queries, line of sight
drone.go     drone types and AI state machine
spatial.go   generic spatial hash grid
physics.go   movement and collision
player.go    player movement, stamina, health, tool selection
tools.go     EMP, decoy, scanner, projectiles, pickups
draw.go      world, fog, HUD, minimap, menus
assets.go    embedded art/audio, animation state machine
save.go      best-time persistence
```
