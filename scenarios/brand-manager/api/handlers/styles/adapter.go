// Package styles is the Connect boundary for container styles and product lines.
package styles

import (
	"brand-manager/internal/styles"

	stylesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/styles"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func styleToProto(s styles.ContainerStyle) *stylesv1.ContainerStyle {
	glow := make([]*stylesv1.GlowLayer, 0, len(s.Glow))
	for _, g := range s.Glow {
		glow = append(glow, &stylesv1.GlowLayer{Width: g.Width, Opacity: g.Opacity})
	}
	return &stylesv1.ContainerStyle{
		Id:                   s.ID,
		Name:                 s.Name,
		Shape:                s.Shape,
		CornerRatio:          s.CornerRatio,
		BackgroundKind:       s.BackgroundKind,
		BackgroundTop:        s.BackgroundTop,
		BackgroundBottom:     s.BackgroundBottom,
		MarkScale:            s.MarkScale,
		MaskableScale:        s.MaskableScale,
		AccentColor:          s.AccentColor,
		Glow:                 glow,
		SmallMarkThresholdPx: int32(s.SmallMarkThresholdPx),
		CreatedAt:            timestamppb.New(s.CreatedAt.UTC()),
		UpdatedAt:            timestamppb.New(s.UpdatedAt.UTC()),
	}
}

func glowFromProto(in []*stylesv1.GlowLayer) []styles.GlowLayer {
	out := make([]styles.GlowLayer, 0, len(in))
	for _, g := range in {
		out = append(out, styles.GlowLayer{Width: g.GetWidth(), Opacity: g.GetOpacity()})
	}
	return out
}

func lineToProto(l styles.ProductLine) *stylesv1.ProductLine {
	return &stylesv1.ProductLine{
		Id:               l.ID,
		Name:             l.Name,
		ContainerStyleId: l.ContainerStyleID,
		Products:         append([]string(nil), l.Products...),
		CreatedAt:        timestamppb.New(l.CreatedAt.UTC()),
	}
}
