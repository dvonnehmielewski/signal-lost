package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func SavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "signal_lost_best.txt"
	}
	return filepath.Join(dir, "SignalLost", "best_time.txt")
}

func LoadBestTime() float32 {
	data, err := os.ReadFile(SavePath())
	if err != nil {
		return 0
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 32)
	if err != nil {
		return 0
	}
	return float32(value)
}

func SaveBestTime(seconds float32) {
	path := SavePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Println("could not save best time:", err)
		return
	}
	err := os.WriteFile(path, []byte(fmt.Sprintf("%.2f\n", seconds)), 0o644)
	if err != nil {
		fmt.Println("could not save best time:", err)
	}
}
