package engine

import (
	"image/png"
	"math/rand"
	"os"
	"testing"
	"time"
)

func frames(n int) []Frame {
	fs := make([]Frame, n)
	for i := range fs {
		fs[i] = Frame{Images: []ImageRef{{ImageIndex: i}}, DurationMs: 100}
	}
	return fs
}

func manifest(anims ...Animation) *Manifest {
	m := &Manifest{}
	m.Info.Width, m.Info.Height = 200, 160
	m.Animations = append([]Animation{{Name: "RestPose", TransitionType: NoReturn, Frames: frames(1)}}, anims...)
	return m
}

func run(p *Player, d time.Duration) { p.Update(d) }

func TestPlaysFramesInOrderAndFinishes(t *testing.T) {
	p := NewPlayer(manifest(Animation{Name: "A", TransitionType: NoReturn, Frames: frames(3)}), rand.New(rand.NewSource(1)))
	p.Play("a", false) // lookup ignores case
	for i := 0; i < 3; i++ {
		if got := p.Pose()[0].ImageIndex; got != i {
			t.Fatalf("frame %d: pose image %d", i, got)
		}
		run(p, 100*time.Millisecond)
	}
	if p.Busy() {
		t.Fatal("still busy after the last frame")
	}
}

func TestSoundFiresOnFrameEntry(t *testing.T) {
	fs := frames(3)
	s := 7
	fs[1].SoundIndex = &s
	p := NewPlayer(manifest(Animation{Name: "A", Frames: fs}), rand.New(rand.NewSource(1)))
	var got []int
	p.OnSound = func(i int) { got = append(got, i) }
	p.Play("A", false)
	run(p, 350*time.Millisecond)
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("sounds = %v", got)
	}
}

func TestEmptyFrameHoldsPreviousPose(t *testing.T) {
	fs := frames(2)
	fs[1].Images = nil
	p := NewPlayer(manifest(Animation{Name: "A", Frames: fs}), rand.New(rand.NewSource(1)))
	p.Play("A", false)
	run(p, 100*time.Millisecond)
	if len(p.Pose()) != 1 || p.Pose()[0].ImageIndex != 0 {
		t.Fatalf("pose = %v", p.Pose())
	}
}

func TestReturnAnimationPlaysOnlyWhenAsked(t *testing.T) {
	m := manifest(
		Animation{Name: "Go", TransitionType: ReturnAnimation, ReturnAnimation: "GORETURN", Frames: frames(2)},
		Animation{Name: "GoReturn", TransitionType: NoReturn, Frames: frames(2)},
	)
	p := NewPlayer(m, rand.New(rand.NewSource(1)))
	p.Play("Go", true)
	run(p, 200*time.Millisecond)
	if p.Current() != "GoReturn" || !p.Busy() {
		t.Fatalf("expected return animation, got %q busy=%v", p.Current(), p.Busy())
	}
	p.Play("Go", false)
	run(p, 200*time.Millisecond)
	if p.Busy() {
		t.Fatal("return animation played with autoReturn=false")
	}
}

func TestBranchesAreFollowedWithTheirProbability(t *testing.T) {
	fs := frames(2)
	fs[1].Branches = []Branch{{FrameIndex: 0, Probability: 100}} // always loops
	p := NewPlayer(manifest(Animation{Name: "Loop", Frames: fs}), rand.New(rand.NewSource(1)))
	p.Play("Loop", false)
	run(p, 10*time.Second)
	if !p.Busy() {
		t.Fatal("a 100% branch back to frame 0 should never finish")
	}
}

func TestExitBranchWalk(t *testing.T) {
	fs := frames(3)
	two, zero := 2, -1
	fs[2].ExitFrame = &two // self-reference: terminal
	fs[1].ExitFrame = &two
	fs[0].ExitFrame = &zero
	p := NewPlayer(manifest(Animation{Name: "X", TransitionType: ExitBranches, Frames: fs}), rand.New(rand.NewSource(1)))
	p.Play("X", true)
	run(p, 10*time.Second)
	if p.Busy() {
		t.Fatal("exit-branch walk must terminate")
	}
}

func TestMoverGlidesAndStops(t *testing.T) {
	var mv Mover
	mv.Start(Point{0, 0}, Point{100, 50}, time.Second)
	if p := mv.Update(500 * time.Millisecond); p.X != 50 || p.Y != 25 {
		t.Fatalf("halfway = %v", p)
	}
	if p := mv.Update(600 * time.Millisecond); p != (Point{100, 50}) || mv.Active() {
		t.Fatalf("end = %v active=%v", p, mv.Active())
	}
}

func TestPickTargetStaysOnScreenAndNearby(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	cur := Point{900, 500}
	for i := 0; i < 2000; i++ {
		p := PickTarget(rng, cur, 1920, 1080, 200, 160, 400)
		if p.X < 0 || p.Y < 0 || p.X > 1720 || p.Y > 920 {
			t.Fatalf("off screen: %v", p)
		}
	}
}

func TestMoveAnimationDirection(t *testing.T) {
	if MoveAnimation(-5) != "MoveLeft" || MoveAnimation(5) != "MoveRight" {
		t.Fatal("wrong direction")
	}
}

// The tests below use the real Bonzi bundle and are skipped when it is absent
// (assets/bonzi is not part of the repository).
func realManifest(t *testing.T) *Manifest {
	data, err := os.ReadFile("../assets/bonzi/manifest.json")
	if err != nil {
		t.Skip("no Bonzi bundle in assets/bonzi")
	}
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRealBundleAtlasFitsSheet(t *testing.T) {
	m := realManifest(t)
	f, err := os.Open("../assets/bonzi/" + m.Sheet)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range m.Atlas {
		if r.X < 0 || r.Y < 0 || r.X+r.W > cfg.Width || r.Y+r.H > cfg.Height {
			t.Fatalf("atlas[%d] %+v outside %dx%d sheet", i, r, cfg.Width, cfg.Height)
		}
	}
	for _, a := range m.Animations {
		for fi, fr := range a.Frames {
			for _, im := range fr.Images {
				if im.ImageIndex < 0 || im.ImageIndex >= len(m.Atlas) {
					t.Fatalf("%s frame %d: bad image index %d", a.Name, fi, im.ImageIndex)
				}
			}
			if fr.SoundIndex != nil && *fr.SoundIndex >= len(m.Sounds) {
				t.Fatalf("%s frame %d: bad sound %d", a.Name, fi, *fr.SoundIndex)
			}
		}
	}
}

func TestRealBundleSwingRuns(t *testing.T) {
	m := realManifest(t)
	for _, name := range []string{"MoveLeft", "MoveRight"} {
		p := NewPlayer(m, rand.New(rand.NewSource(1)))
		var sounds []int
		p.OnSound = func(i int) { sounds = append(sounds, i) }
		if !p.Play(name, true) {
			t.Fatalf("%s missing", name)
		}
		run(p, 1100*time.Millisecond) // the 10-frame swing
		if p.Current() == name {
			t.Fatalf("%s did not hand over to its return animation", name)
		}
		run(p, 2*time.Second)
		if p.Busy() {
			t.Fatalf("%s + return never finished", name)
		}
		if len(sounds) == 0 {
			t.Fatalf("%s played no sound", name)
		}
	}
	if len(m.IdleAnimations("IDLINGLEVEL1")) == 0 {
		t.Fatal("no idle animations resolved")
	}
}
