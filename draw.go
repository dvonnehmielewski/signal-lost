package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	ColorBackground = rl.NewColor(6, 15, 25, 255)
	ColorPanel      = rl.NewColor(9, 23, 35, 235)
	ColorTeal       = rl.NewColor(75, 219, 202, 255)
	ColorGold       = rl.NewColor(244, 184, 79, 255)
	ColorRed        = rl.NewColor(236, 93, 98, 255)
	ColorLight      = rl.NewColor(211, 232, 239, 255)
	ColorMuted      = rl.NewColor(115, 158, 169, 255)
	ColorLine       = rl.NewColor(50, 101, 114, 255)
)

func DrawCentered(text string, y int32, size int32, color rl.Color) {
	width := rl.MeasureText(text, size)
	rl.DrawText(text, ScreenWidth/2-width/2, y, size, color)
}

func DrawWorldText(text string, x, y, size float32, color rl.Color) {
	rl.DrawText(text, int32(x)+1, int32(y)+1, int32(size), rl.NewColor(0, 0, 0, 160))
	rl.DrawText(text, int32(x), int32(y), int32(size), color)
}

func DrawButton(text string, x, y, width, height int32) bool {
	rect := rl.NewRectangle(float32(x), float32(y), float32(width), float32(height))
	mouseOver := rl.CheckCollisionPointRec(rl.GetMousePosition(), rect)
	background := rl.NewColor(23, 40, 58, 255)
	border := rl.NewColor(75, 125, 147, 255)
	if mouseOver {
		background = rl.NewColor(30, 78, 98, 255)
		border = rl.NewColor(84, 239, 216, 255)
	}
	rl.DrawRectangleRec(rect, background)
	rl.DrawRectangleLines(x, y, width, height, border)
	rl.DrawText(text, x+(width-rl.MeasureText(text, 20))/2, y+(height-20)/2, 20, rl.RayWhite)
	return mouseOver && rl.IsMouseButtonPressed(rl.MouseButtonLeft)
}

func DrawStars(offset rl.Vector2) {
	rl.ClearBackground(ColorBackground)
	for i := 0; i < 140; i++ {
		depth := float32(i%3+1) * 0.08
		x := float32((i*151+43)%1200) - offset.X*depth
		y := float32((i*97+27)%800) - offset.Y*depth
		x = float32(math.Mod(float64(x), 1200))
		y = float32(math.Mod(float64(y), 800))
		if x < 0 {
			x += 1200
		}
		if y < 0 {
			y += 800
		}
		brightness := uint8(70 + (i%3)*40)
		rl.DrawCircle(int32(x)-120, int32(y)-100, float32(i%2)+1, rl.NewColor(brightness/2, brightness*3/4, brightness, 255))
	}
}

func DrawBar(x, y, width, height int32, fraction float32, color rl.Color) {
	rl.DrawRectangle(x, y, width, height, rl.NewColor(44, 55, 64, 255))
	rl.DrawRectangle(x, y, int32(float32(width)*fraction), height, color)
	rl.DrawRectangleLines(x, y, width, height, rl.NewColor(132, 167, 172, 255))
}

func DrawOverlay(title, subtitle string) {
	rl.DrawRectangle(0, 0, ScreenWidth, ScreenHeight, rl.NewColor(3, 12, 21, 225))
	DrawCentered(title, 110, 49, rl.NewColor(222, 240, 245, 255))
	DrawCentered(subtitle, 175, 21, rl.NewColor(117, 204, 195, 255))
}

func FormatTime(seconds float32) string {
	total := int(seconds)
	return fmt.Sprintf("%d:%02d.%d", total/60, total%60, int(seconds*10)%10)
}

func (g *Game) Draw() {
	DrawStars(g.Camera.Target)
	g.DrawWorld()
	g.DrawHUD()
	if g.Mode == Intro {
		g.DrawIntro()
	}
}

func (g *Game) DrawWorld() {
	view := g.ViewRect()
	rl.BeginMode2D(g.Camera)

	g.Dungeon.Draw(g.Art, view)

	g.ExitAnim.DrawWithFSM(g.ExitPos, 80, 0, rl.White)

	for i := range g.Relays {
		relay := &g.Relays[i]
		relay.Anim.DrawWithFSM(relay.Pos, 40, 0, rl.White)
		if !relay.Repaired && relay.Progress > 0 {
			rl.DrawRing(relay.Pos, 30, 34, -90, -90+360*relay.Progress, 32, ColorTeal)
		}
		color := ColorGold
		if relay.Repaired {
			color = ColorTeal
		}
		DrawWorldText(fmt.Sprint(i+1), relay.Pos.X-4, relay.Pos.Y-44, 16, color)
	}

	for _, pickup := range g.Pickups {
		if !pickup.Taken && rl.CheckCollisionPointRec(pickup.Pos, view) {
			pickup.Draw()
		}
	}
	for _, decoy := range g.Decoys {
		decoy.Anim.UpdateTime()
		decoy.Anim.DrawAnimation(decoy.Pos, 24, 0, rl.White)
	}
	for _, projectile := range g.Projectiles {
		projectile.Draw(g.Art)
	}
	for i := range g.Effects {
		g.Effects[i].Draw()
	}

	for _, ai := range g.AIs {
		if !rl.CheckCollisionPointRec(ai.Drone.Pos, view) {
			continue
		}
		if g.ScanTimer > 0 {
			rl.DrawCircleLines(int32(ai.Drone.Pos.X), int32(ai.Drone.Pos.Y), ai.SightRange, rl.NewColor(233, 87, 95, 50))
		}
		ai.Drone.Draw(ai.State)
		ai.DrawStatus(20)
	}

	g.Player.Draw()
	g.DrawFog(view)
	g.DrawPrompts()

	rl.EndMode2D()
}

func (g *Game) DrawFog(view rl.Rectangle) {
	if g.ScanTimer > 0 {
		return
	}
	for _, room := range g.Dungeon.Rooms {
		if room.Explored {
			continue
		}
		rect := rl.NewRectangle(float32((room.PivotX+1)*TILE), float32((room.PivotY+1)*TILE), float32((room.Width-2)*TILE), float32((room.Height-2)*TILE))
		if rl.CheckCollisionRecs(rect, view) {
			rl.DrawRectangleRec(rect, rl.NewColor(4, 10, 18, 165))
		}
	}
}

func (g *Game) DrawPrompts() {
	pos := g.Player.Pos
	if index := g.NearbyRelay(); index >= 0 {
		text := "HOLD E - REPAIR"
		if g.Relays[index].Progress > 0 {
			text = fmt.Sprintf("REPAIRING %d%%", int(g.Relays[index].Progress*100))
		}
		DrawWorldText(text, pos.X-float32(rl.MeasureText(text, 14))/2, pos.Y+24, 14, ColorTeal)
		return
	}
	if g.NearExit() {
		text, color := "E - ESCAPE", ColorTeal
		if g.Repaired < len(g.Relays) {
			text, color = "AIRLOCK SEALED", ColorRed
		}
		DrawWorldText(text, pos.X-float32(rl.MeasureText(text, 14))/2, pos.Y+24, 14, color)
	}
}

func (g *Game) DrawHUD() {
	rl.DrawRectangle(0, 0, ScreenWidth, 58, ColorPanel)
	rl.DrawLine(0, 58, ScreenWidth, 58, ColorLine)
	config := Sectors[g.Sector-1]
	rl.DrawText("SIGNAL LOST", 20, 10, 22, ColorLight)
	rl.DrawText(fmt.Sprintf("SECTOR %d/%d  %s", g.Sector, len(Sectors), config.Name), 20, 36, 14, ColorTeal)

	rl.DrawText("RELAYS", 300, 10, 14, ColorMuted)
	for i, relay := range g.Relays {
		color := rl.NewColor(60, 70, 78, 255)
		if relay.Repaired {
			color = ColorTeal
		}
		rl.DrawRectangle(300+int32(i)*26, 30, 20, 16, color)
		rl.DrawRectangleLines(300+int32(i)*26, 30, 20, 16, ColorMuted)
	}

	status, statusColor := "UNDETECTED", ColorTeal
	if g.AnyDroneIn(Chasing) {
		status, statusColor = "DETECTED", ColorRed
		if int(g.RunTime*4)%2 == 0 {
			statusColor = rl.NewColor(255, 160, 160, 255)
		}
	} else if g.AnyDroneIn(Investigating, Searching) {
		status, statusColor = "SUSPICIOUS", ColorGold
	}
	rl.DrawText("STATUS", 410, 10, 14, ColorMuted)
	rl.DrawText(status, 410, 28, 20, statusColor)

	rl.DrawText("INTEGRITY", 560, 8, 14, ColorMuted)
	healthColor := ColorTeal
	if g.Player.Health <= 40 {
		healthColor = ColorRed
	}
	DrawBar(560, 26, 150, 12, float32(g.Player.Health)/float32(g.Player.MaxHealth), healthColor)
	rl.DrawText("THRUST", 560, 41, 12, ColorMuted)
	DrawBar(612, 43, 98, 8, g.Player.Stamina, ColorGold)

	rl.DrawText("TIME", 735, 10, 14, ColorMuted)
	rl.DrawText(FormatTime(g.RunTime), 735, 28, 20, ColorLight)
	if g.ScanTimer > 0 {
		rl.DrawText(fmt.Sprintf("SCAN %.0fs", g.ScanTimer), 830, 10, 14, ColorTeal)
	}

	g.DrawMinimap(ScreenWidth-14, 70)
	g.DrawToolbar()

	if g.MessageTimer > 0 {
		alpha := float32(math.Min(1, float64(g.MessageTimer)*2))
		DrawCentered(g.Message, 492, 18, rl.Fade(rl.NewColor(255, 218, 145, 255), alpha))
	}

	mouse := rl.GetMousePosition()
	tool := g.Player.Tools[g.Player.Selected]
	rl.DrawCircleLines(int32(mouse.X), int32(mouse.Y), 9, tool.Color)
	rl.DrawLine(int32(mouse.X)-14, int32(mouse.Y), int32(mouse.X)-5, int32(mouse.Y), tool.Color)
	rl.DrawLine(int32(mouse.X)+5, int32(mouse.Y), int32(mouse.X)+14, int32(mouse.Y), tool.Color)
	rl.DrawLine(int32(mouse.X), int32(mouse.Y)-14, int32(mouse.X), int32(mouse.Y)-5, tool.Color)
	rl.DrawLine(int32(mouse.X), int32(mouse.Y)+5, int32(mouse.X), int32(mouse.Y)+14, tool.Color)
}

func (g *Game) DrawToolbar() {
	rl.DrawRectangle(0, ScreenHeight-80, ScreenWidth, 80, ColorPanel)
	rl.DrawLine(0, ScreenHeight-80, ScreenWidth, ScreenHeight-80, ColorLine)
	startX := int32(ScreenWidth/2 - (3*190+2*10)/2)
	for i := range g.Player.Tools {
		tool := &g.Player.Tools[i]
		x := startX + int32(i)*200
		y := int32(ScreenHeight - 72)
		selected := i == g.Player.Selected
		border := rl.NewColor(60, 90, 104, 255)
		background := rl.NewColor(14, 30, 44, 255)
		if selected {
			border = tool.Color
			background = rl.NewColor(22, 48, 62, 255)
		}
		rl.DrawRectangle(x, y, 190, 50, background)
		rl.DrawRectangleLines(x, y, 190, 50, border)
		tool.Icon.UpdateTime()
		tool.Icon.DrawAnimation(rl.NewVector2(float32(x+26), float32(y+25)), 30, 0, rl.White)
		rl.DrawText(fmt.Sprintf("%d %s", i+1, tool.Name), x+48, y+7, 16, ColorLight)
		for u := 0; u < tool.MaxUses; u++ {
			color := rl.NewColor(44, 55, 64, 255)
			if u < tool.CurrentUses {
				color = tool.Color
			}
			rl.DrawRectangle(x+48+int32(u)*14, y+30, 10, 10, color)
		}
	}
	rl.DrawText("WASD MOVE  SHIFT BOOST (LOUD)  HOLD E REPAIR  1-3 / WHEEL SELECT  CLICK / SPACE USE  ESC PAUSE", 20, ScreenHeight-17, 10, ColorMuted)
	desc := g.Player.Tools[g.Player.Selected].Behavior.Description()
	rl.DrawText(desc, ScreenWidth-20-rl.MeasureText(desc, 10), ScreenHeight-17, 10, ColorTeal)
}

func (g *Game) DrawMinimap(right, top int32) {
	const scale = 3
	width := int32(g.Dungeon.Width * scale)
	height := int32(g.Dungeon.Height * scale)
	left := right - width
	rl.DrawRectangle(left-6, top-6, width+12, height+12, rl.NewColor(6, 15, 25, 200))
	rl.DrawRectangleLines(left-6, top-6, width+12, height+12, ColorLine)
	scanning := g.ScanTimer > 0

	for i, room := range g.Dungeon.Rooms {
		if !room.Explored && !scanning {
			continue
		}
		color := rl.NewColor(40, 70, 82, 255)
		if i == g.ExitRoom {
			color = rl.NewColor(70, 50, 60, 255)
		}
		rl.DrawRectangle(left+int32(room.PivotX+1)*scale, top+int32(room.PivotY+1)*scale, int32(room.Width-2)*scale, int32(room.Height-2)*scale, color)
	}
	for _, door := range g.Dungeon.Doors {
		if g.Dungeon.Rooms[door.RoomA].Explored || g.Dungeon.Rooms[door.RoomB].Explored || scanning {
			rl.DrawRectangle(left+int32(door.X)*scale, top+int32(door.Y)*scale, scale, scale, rl.NewColor(80, 120, 130, 255))
		}
	}
	toMap := func(pos rl.Vector2) (int32, int32) {
		return left + int32(pos.X/TILE*scale), top + int32(pos.Y/TILE*scale)
	}
	for _, relay := range g.Relays {
		if g.Dungeon.Rooms[relay.Room].Explored || scanning {
			x, y := toMap(relay.Pos)
			color := ColorGold
			if relay.Repaired {
				color = ColorTeal
			}
			rl.DrawRectangle(x-2, y-2, 5, 5, color)
		}
	}
	if g.Dungeon.Rooms[g.ExitRoom].Explored || scanning {
		x, y := toMap(g.ExitPos)
		color := ColorRed
		if g.Repaired == len(g.Relays) {
			color = ColorTeal
		}
		rl.DrawRectangleLines(x-4, y-4, 9, 9, color)
	}
	if scanning {
		for _, ai := range g.AIs {
			x, y := toMap(ai.Drone.Pos)
			rl.DrawCircle(x, y, 2, ColorRed)
		}
	}
	x, y := toMap(g.Player.Pos)
	rl.DrawCircle(x, y, 2.5, rl.RayWhite)
}

func (g *Game) DrawIntro() {
	alpha := float32(math.Min(1, float64(g.IntroTimer)))
	rl.DrawRectangle(0, 200, ScreenWidth, 150, rl.Fade(rl.NewColor(3, 12, 21, 255), 0.85*alpha))
	DrawCentered(fmt.Sprintf("SECTOR %d", g.Sector), 220, 46, rl.Fade(ColorLight, alpha))
	DrawCentered(Sectors[g.Sector-1].Name, 272, 22, rl.Fade(ColorTeal, alpha))
	DrawCentered("Repair 3 relays, then reach the airlock", 304, 18, rl.Fade(ColorMuted, alpha))
	DrawCentered(fmt.Sprintf("RUN SEED %d", g.RunSeed%100000), 330, 12, rl.Fade(ColorLine, alpha))
}

func (g *Game) DrawStats(y int32) {
	lines := []string{
		fmt.Sprintf("TIME  %s", FormatTime(g.RunTime)),
		fmt.Sprintf("DAMAGE TAKEN  %d", g.DamageTaken),
		fmt.Sprintf("DRONES STUNNED  %d", g.DronesStunned),
		fmt.Sprintf("TIMES SPOTTED  %d", g.TimesSpotted),
	}
	for i, line := range lines {
		DrawCentered(line, y+int32(i)*24, 18, ColorLight)
	}
}

type MenuBackdrop struct {
	Dungeon Dungeon
	Camera  rl.Camera2D
	Time    float32
}

func NewMenuBackdrop() MenuBackdrop {
	backdrop := MenuBackdrop{Dungeon: NewDungeon(48, 36)}
	backdrop.Dungeon.Generate(2026, 11, 12)
	backdrop.Camera = rl.NewCamera2D(rl.NewVector2(ScreenWidth/2, ScreenHeight/2), rl.Vector2Zero(), 0, 1)
	return backdrop
}

func (m *MenuBackdrop) Draw(art *Art) {
	m.Time += FrameTime()
	center := rl.NewVector2(float32(m.Dungeon.Width*TILE)/2, float32(m.Dungeon.Height*TILE)/2)
	angle := float64(m.Time) * 0.08
	m.Camera.Target = rl.NewVector2(center.X+float32(math.Cos(angle))*420, center.Y+float32(math.Sin(angle))*260)
	DrawStars(m.Camera.Target)
	rl.BeginMode2D(m.Camera)
	topLeft := rl.GetScreenToWorld2D(rl.Vector2Zero(), m.Camera)
	m.Dungeon.Draw(art, rl.NewRectangle(topLeft.X-TILE, topLeft.Y-TILE, ScreenWidth+2*TILE, ScreenHeight+2*TILE))
	rl.EndMode2D()
	rl.DrawRectangle(0, 0, ScreenWidth, ScreenHeight, rl.NewColor(3, 12, 21, 200))
}

func DrawMenu(backdrop *MenuBackdrop, art *Art, bestTime float32) (start, help, quit bool) {
	backdrop.Draw(art)
	DrawCentered("SIGNAL LOST", 100, 64, rl.NewColor(228, 245, 247, 255))
	DrawCentered("A SPACE STATION STEALTH GAME", 176, 19, ColorTeal)
	rl.DrawLine(280, 210, 680, 210, rl.NewColor(54, 120, 132, 255))
	DrawCentered("The station is dark. Its security drones are not.", 232, 19, rl.NewColor(169, 200, 213, 255))
	DrawCentered("Restore the relays in three sectors and escape.", 260, 19, rl.NewColor(169, 200, 213, 255))
	start = DrawButton("START MISSION", 345, 310, 270, 52)
	help = DrawButton("HOW TO PLAY", 345, 374, 270, 52)
	quit = DrawButton("QUIT", 345, 438, 270, 52)
	if bestTime > 0 {
		DrawCentered("BEST ESCAPE  "+FormatTime(bestTime), 520, 18, ColorGold)
	}
	DrawCentered("Every run generates a new station", 560, 14, ColorMuted)
	return start, help, quit
}

func DrawHelp(backdrop *MenuBackdrop, art *Art) bool {
	backdrop.Draw(art)
	DrawCentered("HOW TO PLAY", 50, 40, ColorLight)
	lines := []struct {
		text  string
		color rl.Color
	}{
		{"OBJECTIVE", ColorTeal},
		{"Each sector hides 3 broken relays. Hold E to repair them, then escape through the airlock.", ColorLight},
		{"", ColorLight},
		{"STEALTH", ColorTeal},
		{"Drones see you when you are in range and nothing blocks their view.", ColorLight},
		{"Boosting with SHIFT and repairing relays make noise - nearby drones come to investigate.", ColorLight},
		{"A drone that spots you alerts the drones around it. Break line of sight to lose them.", ColorLight},
		{"", ColorLight},
		{"TOOLS  (1-3 or mouse wheel to select, click or SPACE to use)", ColorTeal},
		{"EMP - thrown toward the cursor, bounces off walls and stuns every drone in the blast.", ColorLight},
		{"DECOY - a beacon that keeps pinging, luring drones away from you.", ColorLight},
		{"SCANNER - reveals the whole sector, every drone and every relay for a few seconds.", ColorLight},
		{"", ColorLight},
		{"Supplies are scattered around the station. Hunter drones (orange) appear in later sectors.", ColorGold},
	}
	for i, line := range lines {
		DrawCentered(line.text, 110+int32(i)*25, 16, line.color)
	}
	back := DrawButton("BACK", 345, 480, 270, 50)
	return back || rl.IsKeyPressed(rl.KeyEscape)
}
