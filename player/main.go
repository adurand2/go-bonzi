// bonzi is a small desktop pet that plays the original BonziBUDDY character
// animations from an extracted bundle (see README). It contains no networking,
// no installer and no persistence: it is one transparent window.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"bonzi-go/engine"
)

func main() {
	var cfg config
	var bundle string
	var seed int64
	flag.DurationVar(&cfg.interval, "interval", time.Second, "how often he swings to a new spot (0 = calm: occasional idles and wandering)")
	flag.Float64Var(&cfg.hop, "hop", 500, "farthest he moves in one swing, in pixels (0 = anywhere on screen)")
	flag.IntVar(&cfg.scale, "scale", 1, "size multiplier")
	flag.BoolVar(&cfg.mute, "mute", false, "turn sounds off")
	flag.DurationVar(&cfg.quitAt, "quit-after", 0, "swing away and exit after this long (0 = run until right-clicked)")
	flag.StringVar(&bundle, "bundle", "", "directory holding the character bundle (default: bonzi-bundle next to the program)")
	flag.Int64Var(&seed, "seed", 0, "random seed (0 = time based)")
	flag.Parse()
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	if cfg.scale < 1 {
		cfg.scale = 1
	}
	if bundle == "" {
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		bundle = filepath.Join(filepath.Dir(exe), "bonzi-bundle")
	}

	files := os.DirFS(bundle)
	m, sprites, err := loadBundle(files)
	if err != nil {
		log.Fatalf("load %s: %v", bundle, err)
	}
	debug.FreeOSMemory() // hand back the memory used to decode the big sprite sheet
	if err := run(cfg, m, sprites, files, rand.New(rand.NewSource(seed))); err != nil {
		log.Fatal(err)
	}
}

func loadBundle(files fs.FS) (*engine.Manifest, []sprite, error) {
	data, err := fs.ReadFile(files, "manifest.json")
	if err != nil {
		return nil, nil, err
	}
	m, err := engine.ParseManifest(data)
	if err != nil {
		return nil, nil, err
	}
	sheet, err := fs.ReadFile(files, m.Sheet)
	if err != nil {
		return nil, nil, err
	}
	img, err := png.Decode(bytes.NewReader(sheet))
	if err != nil {
		return nil, nil, fmt.Errorf("decode sprite sheet: %w", err)
	}
	return m, loadSprites(img, m.Atlas), nil
}
