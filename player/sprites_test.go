package main

import (
	"image"
	"image/color"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"bonzi-go/engine"
)

func TestCutTrimsAndPremultiplies(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	img.SetNRGBA(3, 4, color.NRGBA{R: 200, G: 100, B: 50, A: 128})
	s := loadSprites(img, []engine.Rect{{X: 0, Y: 0, W: 10, H: 10}})[0]
	if s.x != 3 || s.y != 4 || s.w != 1 || s.h != 1 {
		t.Fatalf("trim box = %+v", s)
	}
	// premultiplied BGRA: 50*128/255=25, 100*128/255=50, 200*128/255=100
	want := []byte{25, 50, 100, 128}
	for i, w := range want {
		if s.pix[i] != w {
			t.Fatalf("pixel = %v, want %v", s.pix, want)
		}
	}
}

func TestBlankImageIsEmpty(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	if s := loadSprites(img, []engine.Rect{{W: 4, H: 4}})[0]; s.w != 0 {
		t.Fatalf("blank image gave %+v", s)
	}
}

func TestBlitPlacesScalesAndBlends(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	s := loadSprites(img, []engine.Rect{{W: 2, H: 2}})
	c := newCanvas(8, 8)
	c.blit(&s[0], 2, 2, 2) // one opaque red pixel, doubled, at (2,2)
	at := func(x, y int) []byte { return c.pix[(y*8+x)*4 : (y*8+x)*4+4] }
	for _, xy := range [][2]int{{2, 2}, {3, 2}, {2, 3}, {3, 3}} {
		if p := at(xy[0], xy[1]); p[2] != 255 || p[3] != 255 {
			t.Fatalf("pixel %v = %v, want opaque red", xy, p)
		}
	}
	if p := at(4, 2); p[3] != 0 {
		t.Fatalf("pixel outside the sprite = %v", p)
	}
	// A half-transparent pixel over the red one: result stays opaque.
	half := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	half.SetNRGBA(0, 0, color.NRGBA{B: 255, A: 128})
	h := loadSprites(half, []engine.Rect{{W: 1, H: 1}})
	c.blit(&h[0], 2, 2, 1)
	if p := at(2, 2); p[3] != 255 || p[0] == 0 || p[2] == 0 {
		t.Fatalf("blend = %v, want opaque purple", p)
	}
	c.blit(&s[0], -50, -50, 1) // fully off canvas must not panic
}

func TestRealBundleLoadsAndStaysSmall(t *testing.T) {
	files := os.DirFS("assets/bonzi")
	if _, err := os.Stat("assets/bonzi/manifest.json"); err != nil {
		t.Skip("no Bonzi bundle in assets/bonzi")
	}
	start := time.Now()
	m, sprites, err := loadBundle(files)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	px := 0
	for _, s := range sprites {
		px += len(s.pix)
	}
	t.Logf("%d sprites, %.1f MB of pixels, heap %.1f MB, loaded in %v", len(sprites), float64(px)/1e6, float64(ms.HeapAlloc)/1e6, took)

	// Render a real swing frame: Bonzi must actually appear on the canvas.
	p := engine.NewPlayer(m, rand.New(rand.NewSource(1)))
	p.Play("MoveLeft", false)
	p.Update(450 * time.Millisecond)
	c := newCanvas(m.Info.Width, m.Info.Height)
	c.draw(p.Pose(), sprites, 1)
	opaque := 0
	for i := 3; i < len(c.pix); i += 4 {
		if c.pix[i] > 0 {
			opaque++
		}
	}
	if opaque < 2000 {
		t.Fatalf("only %d visible pixels in a swing frame", opaque)
	}
}
