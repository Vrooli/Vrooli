// Package styles owns container styles and product lines: the deterministic
// frame a mark is composed into and the set of brands that share it.
package styles

import "time"

// GlowLayer is one accent halo layer.
type GlowLayer struct {
	Width   float64 `json:"width"`
	Opacity float64 `json:"opacity"`
}

// ContainerStyle is the deterministic composition frame.
type ContainerStyle struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Shape                string      `json:"shape"`
	CornerRatio          float64     `json:"corner_ratio"`
	BackgroundKind       string      `json:"background_kind"`
	BackgroundTop        string      `json:"background_top"`
	BackgroundBottom     string      `json:"background_bottom"`
	MarkScale            float64     `json:"mark_scale"`
	MaskableScale        float64     `json:"maskable_scale"`
	AccentColor          string      `json:"accent_color"`
	Glow                 []GlowLayer `json:"glow"`
	SmallMarkThresholdPx int         `json:"small_mark_threshold_px"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

// ProductLine is a set of products sharing one container style.
type ProductLine struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ContainerStyleID string    `json:"container_style_id"`
	Products         []string  `json:"products"`
	CreatedAt        time.Time `json:"created_at"`
}
