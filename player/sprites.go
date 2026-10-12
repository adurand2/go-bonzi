package main

import (
	"image"

	"bonzi-go/engine"
)

// sprite is one character image, trimmed to its visible pixels and stored as
// premultiplied BGRA (the format Windows layered windows want).
type sprite struct {
	x, y, w, h int // trimmed box, relative to the image's own top-left
	pix        []byte
}

// loadSprites cuts every atlas rect out of the sheet.
func loadSprites(sheet image.Image, atlas []engine.Rect) []sprite {
	nrgba, ok := sheet.(*image.NRGBA)
	if !ok {
		b := sheet.Bounds()
		nrgba = image.NewNRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				nrgba.Set(x, y, sheet.At(x, y))
			}
		}
	}
	out := make([]sprite, len(atlas))
	for i, r := range atlas {
		out[i] = cut(nrgba, r)
	}
	return out
}

func cut(src *image.NRGBA, r engine.Rect) sprite {
	minX, minY, maxX, maxY := r.W, r.H, -1, -1
	for y := 0; y < r.H; y++ {
		row := src.Pix[src.PixOffset(r.X, r.Y+y):]
		for x := 0; x < r.W; x++ {
			if row[x*4+3] != 0 {
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
			}
		}
	}
	if maxX < 0 {
		return sprite{} // fully transparent
	}
	s := sprite{x: minX, y: minY, w: maxX - minX + 1, h: maxY - minY + 1}
	s.pix = make([]byte, s.w*s.h*4)
	for y := 0; y < s.h; y++ {
		in := src.Pix[src.PixOffset(r.X+minX, r.Y+minY+y):]
		out := s.pix[y*s.w*4:]
		for x := 0; x < s.w; x++ {
			a := uint32(in[x*4+3])
			out[x*4+0] = byte((uint32(in[x*4+2])*a + 127) / 255) // B
			out[x*4+1] = byte((uint32(in[x*4+1])*a + 127) / 255) // G
			out[x*4+2] = byte((uint32(in[x*4+0])*a + 127) / 255) // R
			out[x*4+3] = byte(a)
		}
	}
	return s
}

// canvas is a premultiplied BGRA pixel buffer.
type canvas struct {
	w, h int
	pix  []byte
}

func newCanvas(w, h int) *canvas { return &canvas{w: w, h: h, pix: make([]byte, w*h*4)} }

func (c *canvas) clear() { clear(c.pix) }

// blit draws s with its image's top-left at (dx, dy), enlarged by scale.
func (c *canvas) blit(s *sprite, dx, dy, scale int) {
	if s.w == 0 {
		return
	}
	for sy := 0; sy < s.h*scale; sy++ {
		y := dy + s.y*scale + sy
		if y < 0 || y >= c.h {
			continue
		}
		in := s.pix[(sy/scale)*s.w*4:]
		for sx := 0; sx < s.w*scale; sx++ {
			x := dx + s.x*scale + sx
			if x < 0 || x >= c.w {
				continue
			}
			p := in[(sx/scale)*4:]
			a := uint32(p[3])
			if a == 0 {
				continue
			}
			d := c.pix[(y*c.w+x)*4:]
			inv := 255 - a
			for k := 0; k < 3; k++ { // source over, both premultiplied
				d[k] = byte(uint32(p[k]) + (uint32(d[k])*inv+127)/255)
			}
			d[3] = byte(a + (uint32(d[3])*inv+127)/255)
		}
	}
}

// draw renders the player's current pose.
func (c *canvas) draw(pose []engine.ImageRef, sprites []sprite, scale int) {
	c.clear()
	for _, ref := range pose {
		if ref.ImageIndex >= 0 && ref.ImageIndex < len(sprites) {
			c.blit(&sprites[ref.ImageIndex], ref.X*scale, ref.Y*scale, scale)
		}
	}
}
