package engine

import (
	"math/rand"
	"strings"
	"time"
)

// Player runs one animation at a time. It is driven by Update(dt) and exposes
// the images to draw with Pose(). Frames with no images hold the previous pose.
type Player struct {
	m   *Manifest
	rng *rand.Rand

	// OnSound is called when a frame carrying a sound is entered.
	OnSound func(index int)

	cur     *Animation
	idx     int
	elapsed time.Duration
	ret     bool // play the return animation when cur finishes
	pose    []ImageRef
	done    bool
}

func NewPlayer(m *Manifest, rng *rand.Rand) *Player {
	p := &Player{m: m, rng: rng, done: true}
	p.Rest()
	return p
}

// Rest shows the character's neutral pose.
func (p *Player) Rest() {
	if a := p.m.Lookup("RestPose"); a != nil {
		p.start(a, false)
		return
	}
	p.start(&p.m.Animations[0], false)
}

// Play starts the named animation, interrupting whatever is running. With
// autoReturn the animation's return path (see transition types) plays after it.
func (p *Player) Play(name string, autoReturn bool) bool {
	a := p.m.Lookup(name)
	if a == nil || len(a.Frames) == 0 {
		return false
	}
	p.start(a, autoReturn)
	return true
}

// Busy reports whether an animation is still running.
func (p *Player) Busy() bool { return !p.done }

// Current returns the running (or last) animation's name.
func (p *Player) Current() string {
	if p.cur == nil {
		return ""
	}
	return p.cur.Name
}

// Pose returns the images to draw now, in draw order.
func (p *Player) Pose() []ImageRef { return p.pose }

func (p *Player) start(a *Animation, autoReturn bool) {
	p.cur, p.ret, p.done = a, autoReturn, false
	p.elapsed = 0
	p.enter(0)
}

func (p *Player) enter(i int) {
	// Leftover time in p.elapsed carries into the new frame, so one large
	// Update can step through several frames.
	p.idx = i
	f := &p.cur.Frames[i]
	if len(f.Images) > 0 {
		p.pose = f.Images
	}
	if f.SoundIndex != nil && *f.SoundIndex >= 0 && p.OnSound != nil {
		p.OnSound(*f.SoundIndex)
	}
}

func (p *Player) frameDuration() time.Duration {
	d := time.Duration(p.cur.Frames[p.idx].DurationMs) * time.Millisecond
	if d < 10*time.Millisecond {
		d = 10 * time.Millisecond
	}
	return d
}

// Update advances the animation by dt.
func (p *Player) Update(dt time.Duration) {
	if p.done || p.cur == nil {
		return
	}
	p.elapsed += dt
	for !p.done && p.elapsed >= p.frameDuration() {
		p.elapsed -= p.frameDuration()
		p.advance()
	}
}

func (p *Player) advance() {
	f := &p.cur.Frames[p.idx]
	// Branches: each has a percent chance; otherwise fall through to the next frame.
	if len(f.Branches) > 0 {
		r, acc := p.rng.Intn(100), 0
		for _, b := range f.Branches {
			acc += b.Probability
			if r < acc && b.FrameIndex >= 0 && b.FrameIndex < len(p.cur.Frames) {
				p.enter(b.FrameIndex)
				return
			}
		}
	}
	if p.idx+1 < len(p.cur.Frames) {
		p.enter(p.idx + 1)
		return
	}
	p.finish()
}

func (p *Player) finish() {
	a := p.cur
	if p.ret {
		switch a.TransitionType {
		case ReturnAnimation:
			if r := p.m.Lookup(a.ReturnAnimation); r != nil && r != a && len(r.Frames) > 0 {
				p.start(r, false)
				return
			}
		case ExitBranches:
			if next, ok := p.exitFrame(a, p.idx); ok {
				p.cur = &Animation{Name: a.Name, Frames: a.Frames}
				p.ret = true
				p.enter(next)
				return
			}
		}
	}
	p.done = true
}

// exitFrame follows the exit branch from frame i, if there is one to follow.
func (p *Player) exitFrame(a *Animation, i int) (int, bool) {
	ef := a.Frames[i].ExitFrame
	if ef == nil || *ef < 0 || *ef >= len(a.Frames) || *ef == i {
		return 0, false
	}
	return *ef, true
}

// IdleAnimations lists the animations a state (e.g. IDLINGLEVEL1) maps to.
func (m *Manifest) IdleAnimations(state string) []string {
	var out []string
	for _, n := range m.States[strings.ToUpper(state)] {
		if m.Lookup(n) != nil {
			out = append(out, n)
		}
	}
	return out
}
