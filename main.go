package main

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Screen int

const (
	MenuScreen Screen = 0
	HelpScreen Screen = 1
	GameScreen Screen = 2
)

func main() {
	rl.InitWindow(ScreenWidth, ScreenHeight, "Signal Lost")
	defer rl.CloseWindow()
	rl.SetExitKey(rl.KeyNull)
	rl.SetTargetFPS(60)

	rl.InitAudioDevice()
	defer rl.CloseAudioDevice()

	art := LoadArt()
	defer art.Unload()
	sounds := LoadSounds()
	defer sounds.Unload()

	ambient := rl.LoadMusicStreamFromMemory(".wav", ambientWav, int32(len(ambientWav)))
	theme := rl.LoadMusicStreamFromMemory(".wav", themeWav, int32(len(themeWav)))
	defer rl.UnloadMusicStream(ambient)
	defer rl.UnloadMusicStream(theme)
	rl.SetMusicVolume(ambient, 0.55)
	rl.PlayMusicStream(ambient)
	playingTheme := false

	backdrop := NewMenuBackdrop()
	bestTime := LoadBestTime()
	screen := MenuScreen
	var game *Game
	quit := false

	for !rl.WindowShouldClose() && !quit {
		wantTheme := screen == GameScreen && game.Mode != Won
		if wantTheme != playingTheme {
			if wantTheme {
				rl.StopMusicStream(ambient)
				rl.PlayMusicStream(theme)
			} else {
				rl.StopMusicStream(theme)
				rl.PlayMusicStream(ambient)
			}
			playingTheme = wantTheme
		}
		if playingTheme {
			volume := float32(0.6)
			if game.Mode == Paused || game.Mode == Lost {
				volume = 0.25
			}
			rl.SetMusicVolume(theme, volume)
			rl.UpdateMusicStream(theme)
		} else {
			rl.UpdateMusicStream(ambient)
		}

		wantCursor := screen != GameScreen || game.Mode != Playing
		if wantCursor && rl.IsCursorHidden() {
			rl.ShowCursor()
		} else if !wantCursor && !rl.IsCursorHidden() {
			rl.HideCursor()
		}

		if screen == GameScreen {
			if game.Mode == Paused && rl.IsKeyPressed(rl.KeyEscape) {
				game.Mode = Playing
			} else {
				game.Update()
			}
		}

		rl.BeginDrawing()
		switch screen {
		case MenuScreen:
			start, help, quitPressed := DrawMenu(&backdrop, &art, bestTime)
			if start {
				game = NewGame(&art, &sounds, uint64(time.Now().UnixNano()))
				screen = GameScreen
			}
			if help {
				screen = HelpScreen
			}
			quit = quitPressed
		case HelpScreen:
			if DrawHelp(&backdrop, &art) {
				screen = MenuScreen
			}
		case GameScreen:
			game.Draw()
			switch game.Mode {
			case Paused:
				DrawOverlay("MISSION PAUSED", "The station awaits your command.")
				if DrawButton("RESUME", 345, 240, 270, 50) {
					game.Mode = Playing
				}
				if DrawButton("RESTART SECTOR", 345, 305, 270, 50) {
					game.RetrySector()
				}
				if DrawButton("MAIN MENU", 345, 370, 270, 50) {
					screen = MenuScreen
				}
				if DrawButton("QUIT GAME", 345, 435, 270, 50) {
					quit = true
				}
			case Lost:
				DrawOverlay("SIGNAL TERMINATED", "Your suit integrity reached zero.")
				game.DrawStats(225)
				if DrawButton("RETRY SECTOR", 345, 340, 270, 52) {
					game.RetrySector()
				}
				if DrawButton("MAIN MENU", 345, 410, 270, 52) {
					screen = MenuScreen
				}
			case Won:
				bestTime = game.BestTime
				DrawOverlay("TRANSMISSION RESTORED", "All three sectors secured. You escaped the station!")
				game.DrawStats(225)
				if game.NewRecord {
					DrawCentered("NEW BEST TIME!", 325, 20, ColorGold)
				} else if game.BestTime > 0 {
					DrawCentered("BEST  "+FormatTime(game.BestTime), 325, 18, ColorMuted)
				}
				if DrawButton("PLAY AGAIN", 345, 370, 270, 52) {
					game = NewGame(&art, &sounds, uint64(time.Now().UnixNano()))
				}
				if DrawButton("MAIN MENU", 345, 440, 270, 52) {
					screen = MenuScreen
				}
			}
		}
		rl.EndDrawing()
	}
}
