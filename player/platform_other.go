//go:build !windows

package main

import (
	"errors"
	"io/fs"
	"math/rand"

	"bonzi-go/engine"
)

func run(config, *engine.Manifest, []sprite, fs.FS, *rand.Rand) error {
	return errors.New("bonzi only runs on Windows")
}
