# bonzi-go

A tiny desktop pet that plays the original BonziBUDDY character animations. It has no
networking, no installer and no scheduled tasks: it opens one transparent window.

By default he swings to a new spot on his vine every second (`-interval 1s`).
Left-click makes him react; right-click makes him swing away and quit. The empty
space around him is click-through.

    bonzi.exe                 # swing every second
    bonzi.exe -interval 0     # calm: idle animations and the odd wander
    bonzi.exe -interval 5s -scale 2 -mute
    bonzi.exe -quit-after 30s # swing away and exit after 30 seconds

Windows only, and standard library only: the window is a per-pixel-alpha layered window
(`UpdateLayeredWindow`) drawn in software, so it needs no GPU or OpenGL and works in VMs.
Sounds are played with `PlaySound`.

## Layout

- `engine/` is the portable core: the manifest, the animation state machine (frame
  timing, branches, return/exit animations) and the movement maths.
- `pet.go` is what he decides to do; `sprites.go` trims images and blends them;
  `platform_windows.go` is the window and the sound.

## Character bundle

The character art and sounds belong to Bonzi Software, so they are not in this repository
or in the built exe. Produce a bundle from your own `Bonzi.acs` with `acs2bundle` from
github.com/kbennett2000/vivify (MIT), which writes `manifest.json`, `sheet.png` and
`audio/*.wav`. Put those in a `bonzi-bundle` folder next to `bonzi.exe`, or point
`-bundle DIR` at them. (On Node, vivify's sound extraction needs a one-line fix:
`new Uint8Array(r.take(ref.size))` in `readSounds`, because `Buffer.slice()` is only a view.)

For the tests that use the real bundle, copy it to `assets/bonzi/`; they skip without it.

## Building

    go test ./...
    GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o bonzi.exe .

CI builds with `ci/windows-build.ps1`.
