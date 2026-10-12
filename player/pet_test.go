package main

import (
	"image"
	"math/rand"
	"os"
	"testing"
	"time"

	"bonzi-go/engine"
)

func realPet(t *testing.T, cfg config) *pet {
	data, err := os.ReadFile("assets/bonzi/manifest.json")
	if err != nil {
		t.Skip("no Bonzi bundle in assets/bonzi")
	}
	m, err := engine.ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	cfg.scale = 1
	return newPet(cfg, m, rand.New(rand.NewSource(7)), image.Rect(0, 0, 1920, 1040))
}

func step(p *pet, total time.Duration) bool {
	for t := time.Duration(0); t < total; t += 16 * time.Millisecond {
		if p.Update(16 * time.Millisecond) {
			return true
		}
	}
	return false
}

func TestSwingsEverySecond(t *testing.T) {
	p := realPet(t, config{interval: time.Second, hop: 500})
	step(p, 3*time.Second) // let the Show drop-in settle, then count swings
	starts := 0
	prev := p.mover.To
	for i := 0; i < 600; i++ { // 9.6 seconds
		p.Update(16 * time.Millisecond)
		if p.mover.To != prev { // a new swing picks a new destination
			starts++
			prev = p.mover.To
			if a := p.player.Current(); a != "MoveLeft" && a != "MoveRight" {
				t.Fatalf("swing started while playing %q", a)
			}
		}
	}
	if starts < 8 || starts > 11 {
		t.Fatalf("%d swings in ~9.6s, want about one per second", starts)
	}
}

func TestStaysOnScreenAndMoves(t *testing.T) {
	p := realPet(t, config{interval: time.Second, hop: 500})
	start := p.pos
	moved := false
	for i := 0; i < 3000; i++ {
		p.Update(16 * time.Millisecond)
		if p.pos.X < 0 || p.pos.Y < 0 || p.pos.X > 1920-200 || p.pos.Y > 1040-160 {
			t.Fatalf("left the desktop: %+v", p.pos)
		}
		if p.pos != start {
			moved = true
		}
	}
	if !moved {
		t.Fatal("never moved")
	}
}

func TestQuitSwingsAwayThenFinishes(t *testing.T) {
	p := realPet(t, config{interval: time.Second, hop: 500})
	step(p, 3*time.Second)
	p.Quit()
	if p.player.Current() != "Hide" {
		t.Fatalf("playing %q after Quit", p.player.Current())
	}
	if !step(p, 5*time.Second) {
		t.Fatal("never finished quitting")
	}
}

func TestQuitAfterFlag(t *testing.T) {
	p := realPet(t, config{interval: time.Second, hop: 500, quitAt: 6 * time.Second})
	if !step(p, 12*time.Second) {
		t.Fatal("-quit-after did not end the program")
	}
}

func TestClickReacts(t *testing.T) {
	p := realPet(t, config{interval: 0})
	step(p, 3*time.Second)
	p.Click()
	if !p.player.Busy() {
		t.Fatal("click did nothing")
	}
}
