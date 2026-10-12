// Package engine is the display-free core of the player: the character manifest
// (produced by acs2bundle), the animation state machine and the movement logic.
// It has no dependencies beyond the standard library so it can be unit-tested
// without a window.
package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Rect struct{ X, Y, W, H int }

type ImageRef struct {
	ImageIndex int `json:"imageIndex"`
	X          int `json:"x"`
	Y          int `json:"y"`
}

type Branch struct {
	FrameIndex  int `json:"frameIndex"`
	Probability int `json:"probability"`
}

type Frame struct {
	Images     []ImageRef `json:"images"`
	DurationMs int        `json:"durationMs"`
	Branches   []Branch   `json:"branches"`
	ExitFrame  *int       `json:"exitFrame"`
	SoundIndex *int       `json:"soundIndex"`
}

// Transition types, as stored in the .acs file.
const (
	ReturnAnimation = 0 // play Animation.ReturnAnimation to get back to rest
	ExitBranches    = 1 // follow each frame's ExitFrame to a terminal frame
	NoReturn        = 2 // the animation already ends at rest
)

type Animation struct {
	Name            string  `json:"name"`
	TransitionType  int     `json:"transitionType"`
	ReturnAnimation string  `json:"returnAnimation"`
	Frames          []Frame `json:"frames"`
}

type Manifest struct {
	Info struct {
		Name   string `json:"name"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"info"`
	Sheet      string      `json:"sheet"`
	Atlas      []Rect      `json:"atlas"`
	Animations []Animation `json:"animations"`
	Sounds     []struct {
		Src string `json:"src"`
	} `json:"sounds"`
	States map[string][]string `json:"states"`
}

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Info.Width <= 0 || m.Info.Height <= 0 || len(m.Animations) == 0 {
		return nil, fmt.Errorf("manifest has no character size or animations")
	}
	return &m, nil
}

// Lookup finds an animation by name, ignoring case (the .acs file stores some
// names upper-case, e.g. return animations).
func (m *Manifest) Lookup(name string) *Animation {
	for i := range m.Animations {
		if strings.EqualFold(m.Animations[i].Name, name) {
			return &m.Animations[i]
		}
	}
	return nil
}
