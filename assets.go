package main

import (
	"embed"
	"math/rand/v2"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets/*.png audio/*.wav
var assetFiles embed.FS

var (
	//go:embed audio/ambient.wav
	ambientWav []byte
	//go:embed audio/station_theme.wav
	themeWav []byte
)

type Art struct {
	Textures map[string]rl.Texture2D
}

func LoadArt() Art {
	art := Art{Textures: make(map[string]rl.Texture2D)}
	entries, err := assetFiles.ReadDir("assets")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		data, err := assetFiles.ReadFile("assets/" + entry.Name())
		if err != nil {
			panic(err)
		}
		image := rl.LoadImageFromMemory(".png", data, int32(len(data)))
		art.Textures[strings.TrimSuffix(entry.Name(), ".png")] = rl.LoadTextureFromImage(image)
		rl.UnloadImage(image)
	}
	return art
}

func (a *Art) Get(name string) rl.Texture2D {
	return a.Textures[name]
}

func (a *Art) Unload() {
	for _, texture := range a.Textures {
		rl.UnloadTexture(texture)
	}
}

type SoundBank struct {
	Sounds map[string]rl.Sound
}

func LoadSounds() SoundBank {
	bank := SoundBank{Sounds: make(map[string]rl.Sound)}
	entries, err := assetFiles.ReadDir("audio")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".wav")
		if name == "ambient" || name == "station_theme" {
			continue
		}
		data, err := assetFiles.ReadFile("audio/" + entry.Name())
		if err != nil {
			panic(err)
		}
		wave := rl.LoadWaveFromMemory(".wav", data, int32(len(data)))
		bank.Sounds[name] = rl.LoadSoundFromWave(wave)
		rl.UnloadWave(wave)
	}
	return bank
}

func (s *SoundBank) Play(name string) {
	if sound, ok := s.Sounds[name]; ok {
		rl.SetSoundPitch(sound, 1)
		rl.PlaySound(sound)
	}
}

func (s *SoundBank) PlayVaried(name string, low, high float32) {
	if sound, ok := s.Sounds[name]; ok {
		rl.SetSoundPitch(sound, low+rand.Float32()*(high-low))
		rl.PlaySound(sound)
	}
}

func (s *SoundBank) Unload() {
	for _, sound := range s.Sounds {
		rl.UnloadSound(sound)
	}
}

type Animation struct {
	Name         string
	SpriteSheet  rl.Texture2D
	MaxIndex     int
	CurrentIndex int
	Timer        float32
	SwitchTime   float32
	Loop         bool
}

func NewAnimation(name string, sheet rl.Texture2D, switchTime float32, loop bool) Animation {
	frames := 1
	if sheet.Height > 0 {
		frames = int(sheet.Width / sheet.Height)
	}
	return Animation{
		Name:        name,
		SpriteSheet: sheet,
		MaxIndex:    frames - 1,
		SwitchTime:  switchTime,
		Loop:        loop,
	}
}

func (a *Animation) UpdateTime() {
	a.Timer += FrameTime()
	if a.Timer > a.SwitchTime {
		a.Timer = 0
		a.CurrentIndex++
	}
	if a.CurrentIndex > a.MaxIndex {
		if a.Loop {
			a.CurrentIndex = 0
		} else {
			a.CurrentIndex = a.MaxIndex
		}
	}
}

func (a *Animation) Reset() {
	a.Timer = 0
	a.CurrentIndex = 0
}

func (a Animation) DrawAnimation(pos rl.Vector2, size float32, rotation float32, tint rl.Color) {
	frameSize := float32(a.SpriteSheet.Height)
	source := rl.NewRectangle(frameSize*float32(a.CurrentIndex), 0, frameSize, frameSize)
	dest := rl.NewRectangle(pos.X, pos.Y, size, size)
	origin := rl.NewVector2(size/2, size/2)
	rl.DrawTexturePro(a.SpriteSheet, source, dest, origin, rotation, tint)
}

type AnimationFSM struct {
	Animations  map[string]Animation
	CurrentAnim Animation
}

func NewAnimationFSM() AnimationFSM {
	return AnimationFSM{Animations: make(map[string]Animation)}
}

func (a *AnimationFSM) AddAnimation(anim Animation) {
	a.Animations[anim.Name] = anim
	if a.CurrentAnim.Name == "" {
		a.CurrentAnim = anim
	}
}

func (a *AnimationFSM) ChangeAnimationState(newAnim string) {
	if newAnim == a.CurrentAnim.Name {
		return
	}
	anim, ok := a.Animations[newAnim]
	if !ok {
		return
	}
	a.CurrentAnim = anim
	a.CurrentAnim.Reset()
}

func (a *AnimationFSM) DrawWithFSM(pos rl.Vector2, size float32, rotation float32, tint rl.Color) {
	a.CurrentAnim.UpdateTime()
	a.CurrentAnim.DrawAnimation(pos, size, rotation, tint)
}

func FrameTime() float32 {
	dt := rl.GetFrameTime()
	if dt > 0.05 {
		return 0.05
	}
	return dt
}
