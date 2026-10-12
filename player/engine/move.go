package engine

import (
	"math"
	"math/rand"
	"time"
)

type Point struct{ X, Y float64 }

// Mover glides the character from one position to another.
type Mover struct {
	From, To Point
	Dur      time.Duration
	elapsed  time.Duration
	active   bool
}

func (mv *Mover) Start(from, to Point, dur time.Duration) {
	*mv = Mover{From: from, To: to, Dur: dur, active: true}
}

func (mv *Mover) Active() bool { return mv.active }

// Update advances the glide and returns the current position.
func (mv *Mover) Update(dt time.Duration) Point {
	if !mv.active {
		return mv.To
	}
	mv.elapsed += dt
	if mv.elapsed >= mv.Dur {
		mv.active = false
		return mv.To
	}
	t := float64(mv.elapsed) / float64(mv.Dur)
	return Point{mv.From.X + (mv.To.X-mv.From.X)*t, mv.From.Y + (mv.To.Y-mv.From.Y)*t}
}

// PickTarget chooses a destination at most maxHop pixels away (0 = anywhere),
// kept fully on a screen of the given size.
func PickTarget(rng *rand.Rand, cur Point, screenW, screenH, charW, charH int, maxHop float64) Point {
	minX, minY := 0.0, 0.0
	maxX, maxY := float64(screenW-charW), float64(screenH-charH)
	if maxX < 0 {
		maxX = 0
	}
	if maxY < 0 {
		maxY = 0
	}
	for try := 0; try < 20; try++ {
		p := Point{minX + rng.Float64()*(maxX-minX), minY + rng.Float64()*(maxY-minY)}
		if maxHop <= 0 || math.Hypot(p.X-cur.X, p.Y-cur.Y) <= maxHop {
			return p
		}
	}
	// Nothing near enough was found: clamp a random step to the screen.
	a := rng.Float64() * 2 * math.Pi
	return Point{
		math.Min(math.Max(cur.X+math.Cos(a)*maxHop, minX), maxX),
		math.Min(math.Max(cur.Y+math.Sin(a)*maxHop, minY), maxY),
	}
}

// MoveAnimation picks the swing animation for a horizontal direction.
func MoveAnimation(dx float64) string {
	if dx < 0 {
		return "MoveLeft"
	}
	return "MoveRight"
}
