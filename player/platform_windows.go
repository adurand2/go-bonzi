//go:build windows

package main

import (
	"errors"
	"image"
	"io/fs"
	"log"
	"math/rand"
	"path"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"bonzi-go/engine"
)

// The window is a layered popup updated with UpdateLayeredWindow, so every
// pixel carries its own alpha: it needs no GPU or OpenGL, and clicks pass
// through the fully transparent parts.

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassEx  = user32.NewProc("RegisterClassExW")
	pCreateWindowEx   = user32.NewProc("CreateWindowExW")
	pDefWindowProc    = user32.NewProc("DefWindowProcW")
	pDestroyWindow    = user32.NewProc("DestroyWindow")
	pShowWindow       = user32.NewProc("ShowWindow")
	pPeekMessage      = user32.NewProc("PeekMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessage  = user32.NewProc("DispatchMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pUpdateLayered    = user32.NewProc("UpdateLayeredWindow")
	pGetDC            = user32.NewProc("GetDC")
	pReleaseDC        = user32.NewProc("ReleaseDC")
	pLoadCursor       = user32.NewProc("LoadCursorW")
	pSysParams        = user32.NewProc("SystemParametersInfoW")
	pSetDPIAware      = user32.NewProc("SetProcessDPIAware")
	pCreateCompatDC   = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection = gdi32.NewProc("CreateDIBSection")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pDeleteDC         = gdi32.NewProc("DeleteDC")
	pGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	pTimeBeginPeriod  = winmm.NewProc("timeBeginPeriod")
	pPlaySound        = winmm.NewProc("PlaySoundW")
)

const (
	wsPopup        = 0x80000000
	wsExLayered    = 0x00080000
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000
	swShowNoActive = 4
	wmDestroy      = 0x0002
	wmLButtonDown  = 0x0201
	wmRButtonDown  = 0x0204
	wmQuit         = 0x0012
	pmRemove       = 1
	ulwAlpha       = 2
	acSrcOver      = 0
	acSrcAlpha     = 1
	spiGetWorkArea = 0x0030
	idcArrow       = 32512
	sndAsync       = 0x0001
	sndNoDefault   = 0x0002
	sndMemory      = 0x0004
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct{ X, Y int32 }
type size struct{ CX, CY int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}
type blendFunction struct{ Op, Flags, Alpha, Format byte }
type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes, Bits  uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// Events from the window procedure, read by the main loop.
var clicked, rightClicked bool

func wndProc(hwnd, m, wparam, lparam uintptr) uintptr {
	switch m {
	case wmLButtonDown:
		clicked = true
		return 0
	case wmRButtonDown:
		rightClicked = true
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(m), wparam, lparam)
	return r
}

func run(cfg config, m *engine.Manifest, sprites []sprite, files fs.FS, rng *rand.Rand) error {
	runtime.LockOSThread() // the window and its messages belong to this thread
	pSetDPIAware.Call()
	pTimeBeginPeriod.Call(1)

	var wa rect
	if r, _, _ := pSysParams.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0); r == 0 {
		return errors.New("cannot read the desktop work area")
	}
	area := image.Rect(int(wa.Left), int(wa.Top), int(wa.Right), int(wa.Bottom))

	p := newPet(cfg, m, rng, area)
	cw, ch := p.charSize()
	if !cfg.mute {
		p.player.OnSound = newSounds(m, files).play
	}

	inst, _, _ := pGetModuleHandle.Call(0)
	cursor, _, _ := pLoadCursor.Call(0, idcArrow)
	className, _ := syscall.UTF16PtrFromString("BonziPet")
	title, _ := syscall.UTF16PtrFromString("Bonzi")
	wc := wndClassEx{Cursor: cursor, Instance: inst, ClassName: className}
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.WndProc = syscall.NewCallback(wndProc)
	if r, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return err
	}
	hwnd, _, err := pCreateWindowEx.Call(
		wsExLayered|wsExTopmost|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), wsPopup,
		uintptr(int(p.pos.X)), uintptr(int(p.pos.Y)), uintptr(cw), uintptr(ch), 0, 0, inst, 0)
	if hwnd == 0 {
		return err
	}

	// A top-down 32-bit bitmap the canvas is copied into.
	screenDC, _, _ := pGetDC.Call(0)
	defer pReleaseDC.Call(0, screenDC)
	memDC, _, _ := pCreateCompatDC.Call(screenDC)
	defer pDeleteDC.Call(memDC)
	bmi := bitmapInfoHeader{Width: int32(cw), Height: -int32(ch), Planes: 1, Bits: 32}
	bmi.Size = uint32(unsafe.Sizeof(bmi))
	var bits unsafe.Pointer
	bmp, _, err := pCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bmi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 {
		return err
	}
	defer pDeleteObject.Call(bmp)
	pSelectObject.Call(memDC, bmp)
	pixels := unsafe.Slice((*byte)(bits), cw*ch*4)

	cv := newCanvas(cw, ch)
	blend := blendFunction{Op: acSrcOver, Alpha: 255, Format: acSrcAlpha}
	sz := size{int32(cw), int32(ch)}
	src := point{}
	present := func() {
		cv.draw(p.player.Pose(), sprites, cfg.scale)
		copy(pixels, cv.pix)
		dst := point{int32(p.pos.X), int32(p.pos.Y)}
		pUpdateLayered.Call(hwnd, screenDC, uintptr(unsafe.Pointer(&dst)), uintptr(unsafe.Pointer(&sz)),
			memDC, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)
	}
	present()
	pShowWindow.Call(hwnd, swShowNoActive)

	last := time.Now()
	for {
		var ms msg
		for {
			r, _, _ := pPeekMessage.Call(uintptr(unsafe.Pointer(&ms)), 0, 0, 0, pmRemove)
			if r == 0 {
				break
			}
			if ms.Message == wmQuit {
				return nil
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&ms)))
			pDispatchMessage.Call(uintptr(unsafe.Pointer(&ms)))
		}
		if rightClicked {
			rightClicked = false
			p.Quit()
		}
		if clicked {
			clicked = false
			p.Click()
		}

		now := time.Now()
		dt := min(now.Sub(last), 250*time.Millisecond) // after a stall, do not jump ahead
		last = now
		if p.Update(dt) {
			pDestroyWindow.Call(hwnd)
			continue // the loop ends on the WM_QUIT that destroying posts
		}
		present()
		time.Sleep(16 * time.Millisecond)
	}
}

// sounds plays the character's WAV sounds with PlaySound. Each wav is kept
// for the life of the program because an asynchronous PlaySound reads straight
// from the buffer it is given.
type sounds struct {
	m     *engine.Manifest
	files fs.FS
	wavs  map[int][]byte
}

func newSounds(m *engine.Manifest, files fs.FS) *sounds {
	return &sounds{m: m, files: files, wavs: map[int][]byte{}}
}

func (s *sounds) play(i int) {
	if i < 0 || i >= len(s.m.Sounds) {
		return
	}
	wav, ok := s.wavs[i]
	if !ok {
		var err error
		if wav, err = fs.ReadFile(s.files, path.Clean(s.m.Sounds[i].Src)); err != nil {
			log.Printf("sound %d: %v", i, err)
		}
		s.wavs[i] = wav
	}
	if len(wav) > 0 {
		pPlaySound.Call(uintptr(unsafe.Pointer(&wav[0])), 0, sndMemory|sndAsync|sndNoDefault)
	}
}
