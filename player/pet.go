package main

import (
	"image"
	"math/rand"
	"time"

	"bonzi-go/engine"
)

type config struct {
	interval time.Duration
	hop      float64
	scale    int
	mute     bool
	quitAt   time.Duration // 0 = run until closed
}

// pet is everything he decides to do, with no window or sound code in it.
type pet struct {
	cfg    config
	m      *engine.Manifest
	player *engine.Player
	mover  engine.Mover
	rng    *rand.Rand
	pos    engine.Point
	area   image.Rectangle // usable desktop area, in pixels

	sinceMove, sinceIdle, nextIdle, age time.Duration
	quitting                            bool
}

func newPet(cfg config, m *engine.Manifest, rng *rand.Rand, area image.Rectangle) *pet {
	p := &pet{cfg: cfg, m: m, rng: rng, area: area}
	p.player = engine.NewPlayer(m, rng)
	p.nextIdle = idleDelay(rng)
	cw, ch := p.charSize()
	// Start near the bottom right of the desktop and drop in on the vine.
	p.pos = engine.Point{X: float64(area.Max.X - cw - 40), Y: float64(area.Max.Y - ch - 40)}
	p.player.Play("Show", false)
	return p
}

func idleDelay(rng *rand.Rand) time.Duration {
	return time.Duration(10+rng.Intn(10)) * time.Second
}

func (p *pet) charSize() (int, int) {
	return p.m.Info.Width * p.cfg.scale, p.m.Info.Height * p.cfg.scale
}

// Click makes him react.
func (p *pet) Click() {
	if p.quitting {
		return
	}
	reactions := []string{"Wave", "Surprised", "Greet", "Giggle", "Wink"}
	p.player.Play(reactions[p.rng.Intn(len(reactions))], true)
}

// Quit makes him swing away; Update reports done once he has gone.
func (p *pet) Quit() {
	if p.quitting {
		return
	}
	p.quitting = true
	p.mover = engine.Mover{}
	p.player.Play("Hide", false)
}

// Update advances time and reports whether it is time to exit.
func (p *pet) Update(dt time.Duration) (done bool) {
	p.age += dt
	if p.cfg.quitAt > 0 && p.age >= p.cfg.quitAt {
		p.Quit()
	}
	if !p.quitting {
		p.behave(dt)
	}
	if p.mover.Active() {
		p.pos = p.mover.Update(dt)
	}
	p.player.Update(dt)
	return p.quitting && !p.player.Busy()
}

func (p *pet) behave(dt time.Duration) {
	if p.cfg.interval > 0 {
		p.sinceMove += dt
		if p.sinceMove >= p.cfg.interval {
			p.sinceMove = 0
			p.swing()
		}
		return
	}
	// Calm mode: an idle animation every so often, and an occasional wander.
	if p.player.Busy() || p.mover.Active() {
		return
	}
	p.sinceIdle += dt
	if p.sinceIdle < p.nextIdle {
		return
	}
	p.sinceIdle, p.nextIdle = 0, idleDelay(p.rng)
	if p.rng.Intn(3) == 0 {
		p.swing()
		return
	}
	if idles := p.m.IdleAnimations("IDLINGLEVEL1"); len(idles) > 0 {
		p.player.Play(idles[p.rng.Intn(len(idles))], true)
	}
}

// swing glides to a new spot on the vine. MoveLeft and MoveRight are ten
// 100 ms frames, so the glide takes the same one second.
func (p *pet) swing() {
	cw, ch := p.charSize()
	rel := engine.Point{X: p.pos.X - float64(p.area.Min.X), Y: p.pos.Y - float64(p.area.Min.Y)}
	to := engine.PickTarget(p.rng, rel, p.area.Dx(), p.area.Dy(), cw, ch, p.cfg.hop)
	to = engine.Point{X: to.X + float64(p.area.Min.X), Y: to.Y + float64(p.area.Min.Y)}
	// Play the return-to-rest animation only if there is time before the next
	// swing; in fast mode he goes straight into the next one.
	back := p.cfg.interval <= 0 || p.cfg.interval >= 2*time.Second
	p.player.Play(engine.MoveAnimation(to.X-p.pos.X), back)
	p.mover.Start(p.pos, to, time.Second)
}
