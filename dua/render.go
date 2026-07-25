package dua

import (
	"fmt"
	"image/color"
	"image/png"
	"io"
	"math"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/opentype"
)

// Canvas: Instagram portrait.
const (
	Width  = 1080
	Height = 1350
	pad    = 60
)

// Theme holds the four colors used by a du'a card.
type Theme struct {
	BG, FG, Muted, Accent color.RGBA
}

// Themes: palettes evoking Islamic art & architecture.
var Themes = map[string]Theme{
	"emerald": { // Prophetic green — most iconic
		BG:     color.RGBA{240, 253, 244, 255}, // very light green tint
		FG:     color.RGBA{6, 46, 22, 255},     // near-black green
		Muted:  color.RGBA{22, 101, 52, 255},   // emerald-800
		Accent: color.RGBA{5, 150, 105, 255},   // emerald-600
	},
	"midnight": { // Kaaba / Isra journey
		BG:     color.RGBA{15, 23, 42, 255},    // slate-900
		FG:     color.RGBA{254, 243, 199, 255}, // warm cream
		Muted:  color.RGBA{148, 163, 184, 255},
		Accent: color.RGBA{212, 175, 55, 255}, // gold
	},
	"marble": { // Al-Masjid al-Haram — white + gold
		BG:     color.RGBA{250, 250, 249, 255}, // stone-50
		FG:     color.RGBA{28, 25, 23, 255},    // stone-900
		Muted:  color.RGBA{120, 113, 108, 255}, // stone-500
		Accent: color.RGBA{180, 83, 9, 255},    // amber-700 (warm gold)
	},
	"sunset": { // Maghrib-time warmth
		BG:     color.RGBA{254, 243, 199, 255}, // amber-100
		FG:     color.RGBA{120, 53, 15, 255},   // amber-900
		Muted:  color.RGBA{194, 65, 12, 255},   // orange-700
		Accent: color.RGBA{5, 150, 105, 255},   // emerald keeps Islamic feel
	},
	"desert": { // Arabian sand
		BG:     color.RGBA{254, 247, 205, 255},
		FG:     color.RGBA{120, 53, 15, 255},
		Muted:  color.RGBA{146, 64, 14, 255},
		Accent: color.RGBA{22, 101, 52, 255}, // deep green on sand
	},
	"rose": { // Gentle palette (e.g. cards for mothers, daughters)
		BG:     color.RGBA{253, 242, 248, 255}, // pink-50
		FG:     color.RGBA{131, 24, 67, 255},   // pink-900
		Muted:  color.RGBA{190, 24, 93, 255},   // pink-700
		Accent: color.RGBA{161, 98, 7, 255},    // muted gold
	},
}

// RenderParams is the input to Render.
type RenderParams struct {
	Situation string  `json:"situation"`
	Recipient string  `json:"recipient"`
	Verses    []Verse `json:"verses"`
	Dua       string  `json:"dua"`
	Theme     string  `json:"theme"`
	Accent    string  `json:"accent"`
}

// ================= Helpers =================

func parseHex(s string, fallback color.RGBA) color.RGBA {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return fallback
	}
	var r, g, b uint32
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return fallback
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

func loadFace(ttf []byte, size float64) (font.Face, error) {
	parsed, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func resolveTheme(name, accentHex string) Theme {
	th, ok := Themes[strings.ToLower(name)]
	if !ok {
		th = Themes["emerald"]
	}
	if strings.TrimSpace(accentHex) != "" {
		th.Accent = parseHex(accentHex, th.Accent)
	}
	return th
}

// drawWrapped draws wrapped text and returns the y-position immediately after
// the drawn block, using the current font face.
func drawWrapped(dc *gg.Context, text string, x, y, maxWidth, lineSpacing float64, align gg.Align) float64 {
	if strings.TrimSpace(text) == "" {
		return y
	}
	lines := dc.WordWrap(text, maxWidth)
	wrapped := strings.Join(lines, "\n")
	_, h := dc.MeasureMultilineString(wrapped, lineSpacing)
	dc.DrawStringWrapped(text, x, y, 0.5, 0, maxWidth, lineSpacing, align)
	return y + h
}

// ================= Islamic ornaments =================

// drawFivePointStar draws a filled 5-pointed star (Islamic 5-star) centered
// at (cx, cy) with outer radius r.
func drawFivePointStar(dc *gg.Context, cx, cy, r float64) {
	inner := r * 0.4
	for i := 0; i < 10; i++ {
		angle := math.Pi/2 - float64(i)*math.Pi/5
		radius := r
		if i%2 == 1 {
			radius = inner
		}
		x := cx + radius*math.Cos(angle)
		y := cy - radius*math.Sin(angle)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.ClosePath()
	dc.Fill()
}

// drawCrescentMoon draws a crescent moon opening to the right with a small
// 5-point star to its right. Centered visually around (cx, cy). r is the
// outer radius of the moon.
func drawCrescentMoon(dc *gg.Context, cx, cy, r float64, accent, bg color.RGBA) {
	// Full moon
	dc.SetColor(accent)
	dc.DrawCircle(cx-r*0.35, cy, r)
	dc.Fill()

	// Bite out — offset BG-colored circle creates the crescent shape.
	dc.SetColor(bg)
	dc.DrawCircle(cx-r*0.1, cy-r*0.15, r*0.9)
	dc.Fill()

	// Small star at the moon's opening.
	dc.SetColor(accent)
	drawFivePointStar(dc, cx+r*0.85, cy+r*0.05, r*0.35)
}

// drawRubElHizb draws the 8-pointed star (۞) formed by two overlapping squares.
// r is the distance from center to a point.
func drawRubElHizb(dc *gg.Context, cx, cy, r float64) {
	// Diamond (rotated square, vertices N/E/S/W).
	dc.MoveTo(cx-r, cy)
	dc.LineTo(cx, cy-r)
	dc.LineTo(cx+r, cy)
	dc.LineTo(cx, cy+r)
	dc.ClosePath()
	dc.Fill()

	// Axis-aligned square with half-diagonal r (side = r*sqrt(2)).
	s := r * 0.7071
	dc.DrawRectangle(cx-s, cy-s, s*2, s*2)
	dc.Fill()
}

// drawSmallDiamond draws a small filled diamond centered at (cx, cy).
func drawSmallDiamond(dc *gg.Context, cx, cy, r float64) {
	dc.MoveTo(cx, cy-r)
	dc.LineTo(cx+r, cy)
	dc.LineTo(cx, cy+r)
	dc.LineTo(cx-r, cy)
	dc.ClosePath()
	dc.Fill()
}

// drawOrnateFrame draws a double border with a small 8-point star (Rub el
// Hizb) at each inner corner.
func drawOrnateFrame(dc *gg.Context, accent color.RGBA) {
	dc.SetColor(accent)

	// Outer border.
	dc.SetLineWidth(2.5)
	dc.DrawRectangle(20, 20, Width-40, Height-40)
	dc.Stroke()

	// Inner border.
	dc.SetLineWidth(1)
	dc.DrawRectangle(38, 38, Width-76, Height-76)
	dc.Stroke()

	// Corner ornaments — small Rub el Hizb at each inner corner.
	corners := [][2]float64{
		{38, 38}, {Width - 38, 38},
		{38, Height - 38}, {Width - 38, Height - 38},
	}
	for _, c := range corners {
		drawRubElHizb(dc, c[0], c[1], 10)
	}
}

// drawFleuronDivider draws:  ●  ────  ۞  ────  ●
// centered at (cx, y).
func drawFleuronDivider(dc *gg.Context, cx, y, halfWidth float64, dotColor, lineColor color.RGBA) {
	const (
		dotR       = 3.0
		starRadius = 10.0
		gap        = 12.0
	)

	// Central 8-point star.
	dc.SetColor(dotColor)
	drawRubElHizb(dc, cx, y, starRadius)

	// Lines flanking the star.
	dc.SetColor(lineColor)
	dc.SetLineWidth(1)
	dc.DrawLine(cx-starRadius-gap, y, cx-halfWidth+dotR*3, y)
	dc.DrawLine(cx+starRadius+gap, y, cx+halfWidth-dotR*3, y)
	dc.Stroke()

	// End dots.
	dc.SetColor(dotColor)
	dc.DrawCircle(cx-halfWidth, y, dotR)
	dc.Fill()
	dc.DrawCircle(cx+halfWidth, y, dotR)
	dc.Fill()
}

// drawSectionHeader draws small-caps text at (cx, y) with a three-dot
// flourish on either side.
func drawSectionHeader(dc *gg.Context, face font.Face, col color.RGBA, text string, cx, y float64) {
	dc.SetFontFace(face)
	dc.SetColor(col)
	dc.DrawStringAnchored(text, cx, y, 0.5, 0.5)

	textW, _ := dc.MeasureString(text)
	const (
		dotR    = 2.5
		gap     = 20.0
		hOffset = 9.0
		vOffset = 5.0
	)
	left := cx - textW/2 - gap
	right := cx + textW/2 + gap

	// Left flourish.
	dc.DrawCircle(left, y, dotR)
	dc.DrawCircle(left-hOffset, y-vOffset, dotR)
	dc.DrawCircle(left-hOffset, y+vOffset, dotR)
	dc.Fill()

	// Right flourish (mirror).
	dc.DrawCircle(right, y, dotR)
	dc.DrawCircle(right+hOffset, y-vOffset, dotR)
	dc.DrawCircle(right+hOffset, y+vOffset, dotR)
	dc.Fill()
}

// ================= Render =================

// Render writes a PNG-encoded du'a card to w.
func Render(w io.Writer, p RenderParams) error {
	th := resolveTheme(p.Theme, p.Accent)

	dc := gg.NewContext(Width, Height)

	// Background.
	dc.SetColor(th.BG)
	dc.Clear()

	// Ornate frame + Rub el Hizb corners.
	drawOrnateFrame(dc, th.Accent)

	centerX := float64(Width) / 2

	// Crescent moon + star at top.
	drawCrescentMoon(dc, centerX, 118, 34, th.Accent, th.BG)

	// "A Du'a for" small caps with flourishes.
	if face, err := loadFace(gosmallcaps.TTF, 22); err == nil {
		drawSectionHeader(dc, face, th.Muted, "A Du'a for", centerX, 210)
	}

	// Situation (bold, wrapped).
	maxTextWidth := float64(Width - 2*pad - 80)
	if face, err := loadFace(gobold.TTF, 34); err == nil {
		dc.SetFontFace(face)
		dc.SetColor(th.FG)
		_ = drawWrapped(dc, nonEmpty(p.Situation, "a life situation"),
			centerX, 240, maxTextWidth, 1.15, gg.AlignCenter)
	}

	// Verses.
	y := 360.0
	refFace, refErr := loadFace(gobold.TTF, 22)
	transFace, tErr := loadFace(goitalic.TTF, 24)
	litFace, litErr := loadFace(goitalic.TTF, 18)
	if refErr != nil {
		return fmt.Errorf("load ref font: %w", refErr)
	}
	if tErr != nil {
		return fmt.Errorf("load translation font: %w", tErr)
	}
	if litErr != nil {
		return fmt.Errorf("load transliteration font: %w", litErr)
	}
	verses := p.Verses
	if len(verses) == 0 {
		verses = []Verse{
			{Surah: "Al-Baqarah", Reference: "2:286",
				Translation:     "Allah does not burden a soul beyond that it can bear.",
				Transliteration: "Lā yukallifu Allāhu nafsan illā wus'ahā."},
			{Surah: "Ash-Sharh", Reference: "94:5-6",
				Translation:     "For indeed, with hardship comes ease. Indeed, with hardship comes ease.",
				Transliteration: "Fa inna ma'al 'usri yusrā. Inna ma'al 'usri yusrā."},
		}
	}
	for i, v := range verses {
		if i > 0 {
			dc.SetColor(th.Muted)
			drawSmallDiamond(dc, centerX, y+2, 4)
			y += 26
		}

		// Reference: "Al-Baqarah 2:286"
		ref := strings.TrimSpace(strings.Join([]string{v.Surah, v.Reference}, " "))
		if ref == "" {
			ref = "Qur'an"
		}
		dc.SetFontFace(refFace)
		dc.SetColor(th.Accent)
		dc.DrawStringAnchored(ref, centerX, y, 0.5, 0.5)
		y += 34

		// Translation (italic).
		dc.SetFontFace(transFace)
		dc.SetColor(th.FG)
		y = drawWrapped(dc, strings.TrimSpace(v.Translation),
			centerX, y, maxTextWidth, 1.3, gg.AlignCenter)

		// Transliteration (smaller italic, muted).
		if strings.TrimSpace(v.Transliteration) != "" {
			y += 8
			dc.SetFontFace(litFace)
			dc.SetColor(th.Muted)
			y = drawWrapped(dc, strings.TrimSpace(v.Transliteration),
				centerX, y, maxTextWidth, 1.2, gg.AlignCenter)
		}
		y += 18
	}

	// Fleuron divider.
	y += 18
	drawFleuronDivider(dc, centerX, y, 220, th.Accent, th.Muted)
	y += 44

	// "Du'a" section header.
	if face, err := loadFace(gosmallcaps.TTF, 22); err == nil {
		drawSectionHeader(dc, face, th.Accent, "Du'a", centerX, y)
		y += 40
	}

	// Du'a body (italic) — auto-shrink to fit above the recipient/brand.
	// The footer occupies roughly the bottom 140px (recipient at Height-100,
	// brand at Height-55, plus breathing room). Try progressively smaller
	// sizes until the wrapped block fits, defaulting to 12pt as the floor.
	duaText := nonEmpty(p.Dua,
		"Ya Allah, grant us patience in every trial, gratitude in every blessing, and nearness to You in every breath. Ameen.")
	const duaBottomBoundary = float64(Height) - 140

	chosenSize := 12.0
	for _, sz := range []float64{22, 20, 18, 16, 14, 12} {
		face, err := loadFace(goitalic.TTF, sz)
		if err != nil {
			continue
		}
		dc.SetFontFace(face)
		lines := dc.WordWrap(duaText, maxTextWidth)
		wrapped := strings.Join(lines, "\n")
		_, h := dc.MeasureMultilineString(wrapped, 1.4)
		if y+h <= duaBottomBoundary {
			chosenSize = sz
			break
		}
	}
	if face, err := loadFace(goitalic.TTF, chosenSize); err == nil {
		dc.SetFontFace(face)
		dc.SetColor(th.FG)
		y = drawWrapped(dc, duaText, centerX, y, maxTextWidth, 1.4, gg.AlignCenter)
		_ = y
	}

	// Optional recipient footer.
	if r := strings.TrimSpace(p.Recipient); r != "" {
		if face, err := loadFace(goitalic.TTF, 20); err == nil {
			dc.SetFontFace(face)
			dc.SetColor(th.Muted)
			dc.DrawStringAnchored("— for "+r+" —", centerX, float64(Height)-100, 0.5, 0.5)
		}
	}

	// Brand line.
	if face, err := loadFace(goregular.TTF, 14); err == nil {
		dc.SetFontFace(face)
		dc.SetColor(th.Muted)
		dc.DrawStringAnchored("islamic-card.example.com", centerX, float64(Height)-55, 0.5, 0.5)
	}

	return png.Encode(w, dc.Image())
}
