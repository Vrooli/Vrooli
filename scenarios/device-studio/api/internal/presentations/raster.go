package presentations

import (
	"context"
	"image"
	"image/color"
	"math"
)

func rasterize(ctx context.Context, app image.Image, c Config, layout Layout) (Result, error) {
	out := image.NewNRGBA(image.Rect(0, 0, c.Target.X, c.Target.Y))
	matte := opaqueMatte(c.Matte)
	if c.Phone != nil {
		if err := paintPhone(ctx, out, c, layout, matte); err != nil {
			return Result{}, err
		}
	} else {
		if err := paintBrowser(ctx, out, c, layout); err != nil {
			return Result{}, err
		}
	}
	if err := compositeContain(ctx, out, app, layout, matte); err != nil {
		return Result{}, err
	}
	return Result{Raster: out, Layout: layout}, nil
}

func paintPhone(ctx context.Context, out *image.NRGBA, c Config, l Layout, matte color.NRGBA) error {
	if err := fill(ctx, out, out.Bounds(), matte); err != nil {
		return err
	}
	status := image.Rect(0, 0, out.Bounds().Dx(), l.AppViewport.Min.Y)
	home := image.Rect(0, l.AppViewport.Max.Y, out.Bounds().Dx(), out.Bounds().Dy())
	if err := fill(ctx, out, status, opaque(l.ResolvedTopFill)); err != nil {
		return err
	}
	if err := fill(ctx, out, home, opaque(l.ResolvedBottomFill)); err != nil {
		return err
	}
	if c.Phone.ShowStatus {
		statusInkValue := colorFromRGB(fixedAppearanceInk(c.Phone.Appearance))
		if l.ProfileVersion == PhoneProfileVersionAuto && l.ResolvedStatusInk != nil {
			statusInkValue = colorFromRGB(*l.ResolvedStatusInk)
		}
		if err := drawPhoneIndicators(ctx, out, status, c.Phone, statusInkValue); err != nil {
			return err
		}
		sx := float64(out.Bounds().Dx()) / phoneRefWidth
		sy := float64(out.Bounds().Dy()) / phoneRefHeight
		ink := statusInkValue
		if c.Phone.Clock != "" && l.ProfileVersion != PhoneProfileVersionAuto {
			if err := drawPinnedText(ctx, out, c.Phone.Clock, int(math.Round(24*sx)), status.Min.Y+status.Dy()/2+int(math.Round(5*sx)), int(math.Round(50*sx)), 14*sx, ink); err != nil {
				return err
			}
		}
		if l.ProfileVersion == PhoneProfileVersionAuto {
			s := math.Min(sx, sy)
			if c.Phone.Clock != "" {
				if err := drawPinnedText(ctx, out, c.Phone.Clock, int(math.Round(24*sx)), status.Min.Y+int(math.Round(43*sy)), int(math.Round(50*sx)), 14*s, ink); err != nil {
					return err
				}
			}
			if c.Phone.Carrier != "" {
				if err := drawPinnedText(ctx, out, c.Phone.Carrier, int(math.Round(24*sx)), status.Min.Y+int(math.Round(67*sy)), int(math.Round(118*sx)), 11*s, ink); err != nil {
					return err
				}
			}
		} else if c.Phone.Carrier != "" {
			x := int(math.Round(78 * sx))
			if c.Phone.Clock != "" {
				x += int(math.Round(48 * sx))
			}
			carrierSafeWidth := int(math.Round(200*sx)) - x
			if err := drawPinnedText(ctx, out, c.Phone.Carrier, x, status.Min.Y+status.Dy()/2+int(math.Round(5*sx)), carrierSafeWidth, 11*sx, ink); err != nil {
				return err
			}
		}
	}
	if c.Phone.ShowHome {
		var resolvedInk *color.NRGBA
		if l.ProfileVersion == PhoneProfileVersionAuto && l.ResolvedHomeInk != nil {
			ink := colorFromRGB(*l.ResolvedHomeInk)
			resolvedInk = &ink
		}
		drawHomeIndicator(out, home, c.Phone.Appearance, resolvedInk)
	}
	return nil
}

func paintBrowser(ctx context.Context, out *image.NRGBA, c Config, l Layout) error {
	b := c.Browser
	wall := RGB{R: 78, G: 93, B: 116}
	if b.DesktopColor != nil {
		wall = *b.DesktopColor
	}
	if err := paintWallpaper(ctx, out, wall); err != nil {
		return err
	}
	if b.ShowMenuBar {
		h := scaled(32, out.Bounds().Dy(), browserRefHeight)
		if err := fillOver(ctx, out, image.Rect(0, 0, out.Bounds().Dx(), h), color.NRGBA{R: 28, G: 35, B: 48, A: 245}); err != nil {
			return err
		}
		if err := drawMenuMarks(ctx, out, h); err != nil {
			return err
		}
	}
	if b.ShowDock {
		h := scaled(78, out.Bounds().Dy(), browserRefHeight)
		y := out.Bounds().Dy() - h
		sx, sy := float64(out.Bounds().Dx())/browserRefWidth, float64(out.Bounds().Dy())/browserRefHeight
		pillW := min(out.Bounds().Dx()-int(math.Round(40*sx)), int(math.Round(440*sx)))
		pill := image.Rect((out.Bounds().Dx()-pillW)/2, y+int(math.Round(5*sy)), (out.Bounds().Dx()+pillW)/2, out.Bounds().Dy()-int(math.Round(5*sy)))
		if err := fillRoundedOver(ctx, out, pill, 22*math.Min(sx, sy), color.NRGBA{R: 30, G: 36, B: 47, A: 232}); err != nil {
			return err
		}
		if err := drawDockIcons(ctx, out, y, sx, sy); err != nil {
			return err
		}
	}
	window, err := browserWindow(c.Target, b.Window)
	if err != nil {
		return err
	}
	windowColor := browserWindowFill(b)
	if err := fillRoundedOver(ctx, out, image.Rect(window.Min.X+4, window.Min.Y+8, window.Max.X+4, window.Max.Y+8), 14*math.Min(float64(out.Bounds().Dx())/browserRefWidth, float64(out.Bounds().Dy())/browserRefHeight), color.NRGBA{R: 0, G: 0, B: 0, A: 62}); err != nil {
		return err
	}
	if err := fillRounded(ctx, out, window, 14*math.Min(float64(out.Bounds().Dx())/browserRefWidth, float64(out.Bounds().Dy())/browserRefHeight), opaque(windowColor)); err != nil {
		return err
	}
	border := color.NRGBA{R: 132, G: 145, B: 164, A: 255}
	strokeRounded(out, window, 14*math.Min(float64(out.Bounds().Dx())/browserRefWidth, float64(out.Bounds().Dy())/browserRefHeight), border)
	if b.ShowToolbar {
		barH := scaled(74, out.Bounds().Dy(), browserRefHeight)
		bar := image.Rect(window.Min.X, window.Min.Y, window.Max.X, window.Min.Y+barH)
		chrome := RGB{}
		if b.ChromeColor != nil {
			chrome = *b.ChromeColor
		} else {
			if b.Appearance == AppearanceLight {
				chrome = RGB{R: 226, G: 230, B: 236}
			} else {
				chrome = RGB{R: 42, G: 48, B: 60}
			}
		}
		if err := fillTopRounded(ctx, out, bar, 14*math.Min(float64(out.Bounds().Dx())/browserRefWidth, float64(out.Bounds().Dy())/browserRefHeight), opaque(chrome)); err != nil {
			return err
		}
		if err := drawToolbarMarks(ctx, out, bar, b); err != nil {
			return err
		}
	}
	return nil
}

func browserWindowFill(b *BrowserConfig) RGB {
	if b.WindowColor != nil {
		return *b.WindowColor
	}
	if b.Appearance == AppearanceLight {
		return RGB{R: 246, G: 247, B: 250}
	}
	return RGB{R: 23, G: 28, B: 38}
}

func paintWallpaper(ctx context.Context, out *image.NRGBA, base RGB) error {
	b := out.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		t := float64(y) / float64(max(1, b.Dy()-1))
		r := uint8(math.Round(float64(base.R)*(1-0.14*t) + 12*t))
		g := uint8(math.Round(float64(base.G)*(1-0.10*t) + 8*t))
		bl := uint8(math.Round(float64(base.B) * (1 - 0.04*t)))
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: bl, A: 255})
		}
	}
	// Authored neutral geometry: broad low-contrast circles never cover the
	// separately placed browser window or depend on external image assets.
	if err := drawCircleContext(ctx, out, image.Pt(b.Dx()*3/4, b.Dy()*3/4), min(b.Dx(), b.Dy())/3, color.NRGBA{R: 255, G: 255, B: 255, A: 14}); err != nil {
		return err
	}
	if err := drawCircleContext(ctx, out, image.Pt(b.Dx()/4, b.Dy()/3), min(b.Dx(), b.Dy())/5, color.NRGBA{R: 15, G: 25, B: 43, A: 18}); err != nil {
		return err
	}
	return nil
}

func compositeContain(ctx context.Context, dst *image.NRGBA, src image.Image, l Layout, matte color.NRGBA) error {
	b := l.FittedApp
	bounds := l.SourceBounds
	s := l.SourceToDisplay.Scale
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			sx := (float64(x)+0.5-l.SourceToDisplay.TranslateX)/s - 0.5
			sy := (float64(y)+0.5-l.SourceToDisplay.TranslateY)/s - 0.5
			dst.SetNRGBA(x, y, sampleBilinearOver(src, bounds, sx, sy, matte))
		}
	}
	return nil
}

// sampleBilinearOver interpolates premultiplied source channels before matte
// compositing. This prevents transparent colored edges from producing halos.
func sampleBilinearOver(src image.Image, bounds image.Rectangle, x, y float64, matte color.NRGBA) color.NRGBA {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	var channels [4]float64
	for dy := 0; dy <= 1; dy++ {
		wy := 1 - fy
		if dy == 1 {
			wy = fy
		}
		sy := min(max(y0+dy, bounds.Min.Y), bounds.Max.Y-1)
		for dx := 0; dx <= 1; dx++ {
			wx := 1 - fx
			if dx == 1 {
				wx = fx
			}
			sx := min(max(x0+dx, bounds.Min.X), bounds.Max.X-1)
			weight := wx * wy
			r, g, b, a := src.At(sx, sy).RGBA()
			channels[0] += float64(r) * weight
			channels[1] += float64(g) * weight
			channels[2] += float64(b) * weight
			channels[3] += float64(a) * weight
		}
	}
	remaining := 65535 - channels[3]
	r := channels[0] + float64(matte.R)*257*remaining/65535
	g := channels[1] + float64(matte.G)*257*remaining/65535
	b := channels[2] + float64(matte.B)*257*remaining/65535
	return color.NRGBA{R: uint8(math.Round(r / 257)), G: uint8(math.Round(g / 257)), B: uint8(math.Round(b / 257)), A: 255}
}

func over(src color.Color, matte color.NRGBA) color.NRGBA {
	r, g, b, a := src.RGBA()
	ma := uint64(0xffff - a)
	r = uint32(uint64(r) + uint64(matte.R)*257*ma/0xffff)
	g = uint32(uint64(g) + uint64(matte.G)*257*ma/0xffff)
	b = uint32(uint64(b) + uint64(matte.B)*257*ma/0xffff)
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255}
}

func fill(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, c color.NRGBA) error {
	r := rect.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetNRGBA(x, y, c)
		}
	}
	return nil
}

func fillOver(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, c color.NRGBA) error {
	r := rect.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetNRGBA(x, y, over(c, dst.NRGBAAt(x, y)))
		}
	}
	return nil
}

func fillRounded(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, radius float64, c color.NRGBA) error {
	return fillRoundedWith(ctx, dst, rect, radius, c, false)
}

func fillRoundedOver(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, radius float64, c color.NRGBA) error {
	return fillRoundedWith(ctx, dst, rect, radius, c, true)
}

func fillRoundedWith(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, radius float64, c color.NRGBA, blend bool) error {
	r := rect.Intersect(dst.Bounds())
	r2 := radius * radius
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			if !insideRounded(rect, x, y, radius, r2) {
				continue
			}
			if blend {
				dst.SetNRGBA(x, y, over(c, dst.NRGBAAt(x, y)))
			} else {
				dst.SetNRGBA(x, y, c)
			}
		}
	}
	return nil
}

func fillTopRounded(ctx context.Context, dst *image.NRGBA, rect image.Rectangle, radius float64, c color.NRGBA) error {
	r := rect.Intersect(dst.Bounds())
	r2 := radius * radius
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y&31 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		for x := r.Min.X; x < r.Max.X; x++ {
			if y < rect.Min.Y+int(radius) && (x < rect.Min.X+int(radius) || x >= rect.Max.X-int(radius)) {
				cx := float64(rect.Min.X) + radius
				if x >= rect.Max.X-int(radius) {
					cx = float64(rect.Max.X) - radius
				}
				cy := float64(rect.Min.Y) + radius
				dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
				if dx*dx+dy*dy > r2 {
					continue
				}
			}
			dst.SetNRGBA(x, y, c)
		}
	}
	return nil
}

func insideRounded(r image.Rectangle, x, y int, radius, radius2 float64) bool {
	if radius <= 0 {
		return image.Pt(x, y).In(r)
	}
	cx := float64(x) + 0.5
	cy := float64(y) + 0.5
	left, right := float64(r.Min.X)+radius, float64(r.Max.X)-radius
	top, bottom := float64(r.Min.Y)+radius, float64(r.Max.Y)-radius
	if cx >= left && cx <= right || cy >= top && cy <= bottom {
		return true
	}
	px, py := cx, cy
	if cx < left {
		px = left
	} else if cx > right {
		px = right
	}
	if cy < top {
		py = top
	} else if cy > bottom {
		py = bottom
	}
	dx, dy := cx-px, cy-py
	return dx*dx+dy*dy <= radius2
}

func strokeRounded(dst *image.NRGBA, r image.Rectangle, radius float64, c color.NRGBA) {
	inner := image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X-1, r.Max.Y-1)
	innerRadius := math.Max(0, radius-1)
	outer2, inner2 := radius*radius, innerRadius*innerRadius
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if insideRounded(r, x, y, radius, outer2) && !insideRounded(inner, x, y, innerRadius, inner2) {
				dst.SetNRGBA(x, y, c)
			}
		}
	}
}

func drawLine(dst *image.NRGBA, from, to image.Point, c color.NRGBA, thickness int) {
	dx, sx := abs(to.X-from.X), 1
	if from.X > to.X {
		sx = -1
	}
	dy, sy := -abs(to.Y-from.Y), 1
	if from.Y > to.Y {
		sy = -1
	}
	err := dx + dy
	for {
		half := thickness / 2
		fillFast(dst, image.Rect(from.X-half, from.Y-half, from.X-half+thickness, from.Y-half+thickness), c)
		if from == to {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			from.X += sx
		}
		if e2 <= dx {
			err += dx
			from.Y += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func opaque(c RGB) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255} }
func strokeRect(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	if r.Empty() {
		return
	}
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.SetNRGBA(x, r.Min.Y, c)
		dst.SetNRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst.SetNRGBA(r.Min.X, y, c)
		dst.SetNRGBA(r.Max.X-1, y, c)
	}
}

func drawPhoneIndicators(ctx context.Context, dst *image.NRGBA, r image.Rectangle, p *PhoneConfig, ink color.NRGBA) error {
	if r.Empty() {
		return nil
	}
	s := math.Max(0.5, float64(dst.Bounds().Dx())/phoneRefWidth)
	y := r.Min.Y + r.Dy()/2
	right := r.Max.X - int(math.Round(18*s))
	batteryX := right - int(math.Round(22*s))
	drawBattery(dst, batteryX, y, ink, s, p.BatteryPercent)
	x := batteryX - int(math.Round(8*s))
	if p.ShowBatteryPercent {
		label := percentString(p.BatteryPercent)
		if err := drawPinnedText(ctx, dst, label, x-int(math.Round(31*s)), y+int(math.Round(4*s)), int(math.Round(30*s)), 10*s, ink); err != nil {
			return err
		}
		x -= int(math.Round(38 * s))
	}
	if p.WiFi {
		drawWiFi(dst, image.Pt(x-int(math.Round(16*s)), y), ink, s)
		x -= int(math.Round(30 * s))
	}
	if p.Cellular {
		drawSignal(dst, image.Pt(x-int(math.Round(14*s)), y), ink, s)
	}
	return ctx.Err()
}

func drawHomeIndicator(dst *image.NRGBA, r image.Rectangle, a Appearance, resolvedInk *color.NRGBA) {
	if r.Empty() {
		return
	}
	c := color.NRGBA{R: 235, G: 237, B: 240, A: 255}
	if a == AppearanceLight {
		c = color.NRGBA{R: 35, G: 38, B: 44, A: 255}
	}
	if resolvedInk != nil {
		c = *resolvedInk
	}
	w := min(int(math.Round(108*float64(dst.Bounds().Dx())/phoneRefWidth)), r.Dx()/3)
	y := r.Min.Y + r.Dy()/2
	h := max(1, int(math.Round(2*float64(dst.Bounds().Dy())/phoneRefHeight)))
	fillFast(dst, image.Rect((dst.Bounds().Dx()-w)/2, y-h, (dst.Bounds().Dx()+w)/2, y+h), c)
}

func colorFromRGB(c RGB) color.NRGBA { return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255} }

func drawSignal(dst *image.NRGBA, at image.Point, c color.NRGBA, scale float64) {
	for i := 0; i < 4; i++ {
		h := max(1, int(math.Round(float64(3+i*3)*scale)))
		for y := 0; y < h; y++ {
			for dx := 0; dx < max(1, int(math.Round(2*scale))); dx++ {
				dst.SetNRGBA(at.X+int(math.Round(float64(i*4)*scale))+dx, at.Y-y, c)
			}
		}
	}
}

func drawWiFi(dst *image.NRGBA, at image.Point, c color.NRGBA, scale float64) {
	line := max(1, int(math.Round(1.5*scale)))
	centerY := at.Y + int(math.Round(3*scale))
	for _, baseRadius := range []int{9, 6, 3} {
		radius := max(2, int(math.Round(float64(baseRadius)*scale)))
		prev := image.Point{}
		for step := 0; step <= 24; step++ {
			angle := (225 + float64(step)*90/24) * math.Pi / 180
			pt := image.Pt(at.X+int(math.Round(math.Cos(angle)*float64(radius))), centerY+int(math.Round(math.Sin(angle)*float64(radius))))
			if step > 0 {
				drawLine(dst, prev, pt, c, line)
			}
			prev = pt
		}
	}
	dot := max(2, int(math.Round(2.2*scale)))
	drawCircle(dst, image.Pt(at.X, centerY+int(math.Round(9*scale))), dot/2, c)
}

func drawBattery(dst *image.NRGBA, x, y int, c color.NRGBA, scale float64, percent uint8) {
	w, h := max(1, int(math.Round(22*scale))), max(1, int(math.Round(6*scale)))
	r := image.Rect(x, y-h, x+w, y+h)
	strokeRect(dst, r, c)
	fillFast(dst, image.Rect(x+w, y-max(1, h/2), x+w+max(1, int(math.Round(2*scale))), y+max(1, h/2)), c)
	left := x + max(1, int(math.Round(3*scale)))
	fillWidth := int(math.Round(float64(max(0, w-6)) * float64(percent) / 100))
	if fillWidth > 0 {
		fillFast(dst, image.Rect(left, y-max(1, h/2), left+fillWidth, y+max(1, h/2)), c)
	}
}

func drawMenuMarks(ctx context.Context, dst *image.NRGBA, h int) error {
	sx, sy := float64(dst.Bounds().Dx())/browserRefWidth, float64(dst.Bounds().Dy())/browserRefHeight
	return drawPinnedText(ctx, dst, "File   Edit   View   Window   Help", int(math.Round(18*sx)), h/2+int(math.Round(5*sy)), dst.Bounds().Dx()-int(math.Round(36*sx)), 12*math.Min(sx, sy), color.NRGBA{R: 225, G: 230, B: 238, A: 255})
}

func drawDockIcons(ctx context.Context, dst *image.NRGBA, y int, sx, sy float64) error {
	cx := dst.Bounds().Dx() / 2
	s := math.Min(sx, sy)
	spacing := int(math.Round(52 * sx))
	size := max(12, int(math.Round(38*s)))
	radius := math.Max(4, 9*s)
	colors := []color.NRGBA{
		{R: 54, G: 146, B: 224, A: 255},
		{R: 236, G: 165, B: 55, A: 255},
		{R: 79, G: 181, B: 139, A: 255},
		{R: 219, G: 93, B: 102, A: 255},
		{R: 130, G: 112, B: 220, A: 255},
		{R: 52, G: 168, B: 196, A: 255},
		{R: 104, G: 118, B: 145, A: 255},
	}
	for i := range colors {
		if err := ctx.Err(); err != nil {
			return err
		}
		cx := cx + (i-3)*spacing
		r := image.Rect(cx-size/2, y+int(math.Round(16*sy)), cx+(size+1)/2, y+int(math.Round(16*sy))+size)
		if err := fillRoundedWith(ctx, dst, r, radius, colors[i], false); err != nil {
			return err
		}
		ink := color.NRGBA{R: 247, G: 249, B: 252, A: 255}
		mark := max(2, int(math.Round(2*s)))
		switch i {
		case 0: // Finder-like crossed lens.
			drawCircle(dst, image.Pt(cx, r.Min.Y+size/2), max(3, size/4), ink)
			fillFast(dst, image.Rect(cx+size/5, r.Min.Y+size/5, cx+size/5+mark, r.Min.Y+size/5+mark), colors[i])
		case 1: // Notes.
			for k := 0; k < 3; k++ {
				fillFast(dst, image.Rect(r.Min.X+size/4, r.Min.Y+size/3+k*mark*2, r.Max.X-size/4, r.Min.Y+size/3+k*mark*2+mark), ink)
			}
		case 2: // Messages.
			if err := fillRoundedWith(ctx, dst, image.Rect(r.Min.X+size/5, r.Min.Y+size/4, r.Max.X-size/5, r.Max.Y-size/4), radius/2, ink, false); err != nil {
				return err
			}
		case 3: // Photos.
			drawCircle(dst, image.Pt(cx, r.Min.Y+size/2), max(3, size/4), ink)
		case 4: // Browser compass.
			drawCircle(dst, image.Pt(cx, r.Min.Y+size/2), max(3, size/3), ink)
			drawLine(dst, image.Pt(cx-mark, r.Min.Y+size/2+mark), image.Pt(cx+mark, r.Min.Y+size/2-mark), colors[i], mark)
		case 5: // Mail.
			strokeRect(dst, image.Rect(r.Min.X+size/5, r.Min.Y+size/3, r.Max.X-size/5, r.Max.Y-size/3), ink)
			drawLine(dst, image.Pt(r.Min.X+size/5, r.Min.Y+size/3), image.Pt(cx, r.Min.Y+size/2), ink, mark)
			drawLine(dst, image.Pt(cx, r.Min.Y+size/2), image.Pt(r.Max.X-size/5, r.Min.Y+size/3), ink, mark)
		case 6: // Settings.
			drawCircle(dst, image.Pt(cx, r.Min.Y+size/2), max(3, size/4), ink)
			drawCircle(dst, image.Pt(cx, r.Min.Y+size/2), max(1, size/8), colors[i])
		}
	}
	return ctx.Err()
}

func drawToolbarMarks(ctx context.Context, dst *image.NRGBA, r image.Rectangle, b *BrowserConfig) error {
	sx, sy := float64(dst.Bounds().Dx())/browserRefWidth, float64(dst.Bounds().Dy())/browserRefHeight
	s := math.Min(sx, sy)
	muted, field, activeTab := color.NRGBA{R: 161, G: 172, B: 188, A: 255}, color.NRGBA{R: 30, G: 36, B: 48, A: 255}, color.NRGBA{R: 58, G: 66, B: 82, A: 255}
	if b.Appearance == AppearanceLight {
		muted, field, activeTab = color.NRGBA{R: 91, G: 102, B: 118, A: 255}, color.NRGBA{R: 241, G: 243, B: 247, A: 255}, color.NRGBA{R: 250, G: 251, B: 253, A: 255}
	}
	if b.ShowWindowControls {
		cy := r.Min.Y + int(math.Round(16*sy))
		for i, c := range []color.NRGBA{{R: 235, G: 95, B: 88, A: 255}, {R: 239, G: 178, B: 68, A: 255}, {R: 78, G: 190, B: 112, A: 255}} {
			drawCircle(dst, image.Pt(r.Min.X+int(math.Round(float64(20+i*18)*sx)), cy), max(2, int(math.Round(5*s))), c)
		}
	}
	// Raised tab strip with a quiet active tab and a small close affordance.
	tabX := r.Min.X + int(math.Round(82*sx))
	tabWidth := min(r.Dx()/3, int(math.Round(280*sx)))
	tab := image.Rect(tabX, r.Min.Y+int(math.Round(3*sy)), tabX+tabWidth, r.Min.Y+int(math.Round(31*sy)))
	if err := fillTopRounded(ctx, dst, tab, 7*s, activeTab); err != nil {
		return err
	}
	if b.Title != "" {
		textX := tab.Min.X + int(math.Round(17*sx))
		if err := drawPinnedText(ctx, dst, b.Title, textX, tab.Min.Y+int(math.Round(20*sy)), tab.Max.X-textX-int(math.Round(28*sx)), 12*s, muted); err != nil {
			return err
		}
	}
	closeX, closeY := tab.Max.X-int(math.Round(14*sx)), tab.Min.Y+int(math.Round(15*sy))
	drawLine(dst, image.Pt(closeX-int(math.Round(3*sx)), closeY-int(math.Round(3*sy))), image.Pt(closeX+int(math.Round(3*sx)), closeY+int(math.Round(3*sy))), muted, max(1, int(math.Round(1.2*s))))
	drawLine(dst, image.Pt(closeX-int(math.Round(3*sx)), closeY+int(math.Round(3*sy))), image.Pt(closeX+int(math.Round(3*sx)), closeY-int(math.Round(3*sy))), muted, max(1, int(math.Round(1.2*s))))
	// Navigation row: back/forward/reload, then a bounded address field.
	rowY := r.Min.Y + int(math.Round(52*sy))
	iconInk := muted
	drawNavArrow(dst, image.Pt(r.Min.X+int(math.Round(19*sx)), rowY), true, iconInk, s)
	drawNavArrow(dst, image.Pt(r.Min.X+int(math.Round(47*sx)), rowY), false, iconInk, s)
	drawReload(dst, image.Pt(r.Min.X+int(math.Round(77*sx)), rowY), iconInk, s)
	addr := image.Rect(r.Min.X+int(math.Round(98*sx)), r.Min.Y+int(math.Round(39*sy)), r.Max.X-int(math.Round(24*sx)), r.Min.Y+int(math.Round(66*sy)))
	if err := fillRounded(ctx, dst, addr, 12*s, field); err != nil {
		return err
	}
	lockX, lockY := addr.Min.X+int(math.Round(13*sx)), addr.Min.Y+int(math.Round(15*sy))
	lockWidth, lockHeight := max(6, int(math.Round(9*s))), max(5, int(math.Round(7*s)))
	shackleTop := lockY - max(3, int(math.Round(5*s)))
	strokeRect(dst, image.Rect(lockX+max(1, int(math.Round(2*s))), shackleTop, lockX+lockWidth-max(1, int(math.Round(2*s))), lockY+1), muted)
	strokeRect(dst, image.Rect(lockX, lockY, lockX+lockWidth, lockY+lockHeight), muted)
	textX := lockX + int(math.Round(18*sx))
	if b.Address != "" {
		return drawPinnedText(ctx, dst, b.Address, textX, addr.Min.Y+int(math.Round(18*sy)), addr.Max.X-textX-int(math.Round(10*sx)), 12*s, muted)
	}
	return nil
}

func drawNavArrow(dst *image.NRGBA, p image.Point, back bool, c color.NRGBA, scale float64) {
	delta := int(math.Round(6 * scale))
	dir := 1
	if back {
		dir = -1
	}
	thick := max(1, int(math.Round(1.6*scale)))
	tip := p.X + dir*delta
	drawLine(dst, image.Pt(p.X-dir*delta, p.Y), image.Pt(tip, p.Y), c, thick)
	drawLine(dst, image.Pt(tip, p.Y), image.Pt(tip-dir*delta/2, p.Y-delta/2), c, thick)
	drawLine(dst, image.Pt(tip, p.Y), image.Pt(tip-dir*delta/2, p.Y+delta/2), c, thick)
}

func drawReload(dst *image.NRGBA, p image.Point, c color.NRGBA, scale float64) {
	r := max(3, int(math.Round(6*scale)))
	for degree := 35; degree <= 315; degree += 5 {
		a := float64(degree) * math.Pi / 180
		pt := image.Pt(p.X+int(math.Round(math.Cos(a)*float64(r))), p.Y+int(math.Round(math.Sin(a)*float64(r))))
		nextA := float64(degree+5) * math.Pi / 180
		next := image.Pt(p.X+int(math.Round(math.Cos(nextA)*float64(r))), p.Y+int(math.Round(math.Sin(nextA)*float64(r))))
		drawLine(dst, pt, next, c, max(1, int(math.Round(scale))))
	}
	drawLine(dst, image.Pt(p.X+r-1, p.Y-r/2), image.Pt(p.X+r+1, p.Y-r/2+max(2, int(math.Round(3*scale)))), c, max(1, int(math.Round(scale))))
}

func drawCircle(dst *image.NRGBA, center image.Point, r int, c color.NRGBA) {
	if r <= 0 {
		return
	}
	rr := r * r
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= rr {
				px, py := center.X+x, center.Y+y
				dst.SetNRGBA(px, py, over(c, dst.NRGBAAt(px, py)))
			}
		}
	}
}

func drawCircleContext(ctx context.Context, dst *image.NRGBA, center image.Point, r int, c color.NRGBA) error {
	if r <= 0 {
		return nil
	}
	rr := r * r
	for y := -r; y <= r; y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := -r; x <= r; x++ {
			if x*x+y*y <= rr {
				px, py := center.X+x, center.Y+y
				dst.SetNRGBA(px, py, over(c, dst.NRGBAAt(px, py)))
			}
		}
	}
	return nil
}

func fillFast(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.SetNRGBA(x, y, c)
		}
	}
}
