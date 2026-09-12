package brands

import (
	"brand-manager/internal/brands"

	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// domainToProto converts an internal brands.Brand into the wire shape the
// brands proto declares. Lives in the handler package by intent — the
// conversion is mechanical and only used at the transport edge.
func domainToProto(b brands.Brand) *brandsv1.Brand {
	return &brandsv1.Brand{
		Id:          b.ID,
		Name:        b.Name,
		Description: b.Description,
		Identity:    identityToProto(b.Identity),
		Colors:      colorsToProto(b.Colors),
		Typography:  typographyToProto(b.Typography),
		Voice:       voiceToProto(b.Voice),
		Notes:       b.Notes,
		Version:     int32(b.Version),
		CreatedAt:   timestamppb.New(b.CreatedAt.UTC()),
		UpdatedAt:   timestamppb.New(b.UpdatedAt.UTC()),
	}
}

func versionToProto(v brands.BrandVersion) *brandsv1.BrandVersion {
	return &brandsv1.BrandVersion{
		Id:        v.ID,
		BrandId:   v.BrandID,
		Version:   int32(v.Version),
		Snapshot:  v.Snapshot,
		CreatedAt: timestamppb.New(v.CreatedAt.UTC()),
	}
}

func identityToProto(i brands.Identity) *brandsv1.Identity {
	return &brandsv1.Identity{
		DisplayName: i.DisplayName,
		Tagline:     i.Tagline,
		LogoPath:    i.LogoPath,
		FaviconPath: i.FaviconPath,
		IconPath:    i.IconPath,
	}
}

func colorsToProto(c brands.Colors) *brandsv1.Colors {
	return &brandsv1.Colors{
		Primary:    c.Primary,
		Secondary:  c.Secondary,
		Accent:     c.Accent,
		Background: c.Background,
		Surface:    c.Surface,
		Text:       c.Text,
		Error:      c.Error,
	}
}

func typographyToProto(t brands.Typography) *brandsv1.Typography {
	return &brandsv1.Typography{
		HeadingFont:  t.HeadingFont,
		BodyFont:     t.BodyFont,
		MonoFont:     t.MonoFont,
		BaseFontSize: t.BaseFontSize,
	}
}

func voiceToProto(v brands.Voice) *brandsv1.Voice {
	return &brandsv1.Voice{
		Tone:     v.Tone,
		Style:    v.Style,
		Keywords: append([]string(nil), v.Keywords...),
	}
}

// identityFromProto converts a wire Identity (which may be nil) into the domain
// value. A nil message yields the zero value — "no facet supplied".
func identityFromProto(i *brandsv1.Identity) brands.Identity {
	if i == nil {
		return brands.Identity{}
	}
	return brands.Identity{
		DisplayName: i.GetDisplayName(),
		Tagline:     i.GetTagline(),
		LogoPath:    i.GetLogoPath(),
		FaviconPath: i.GetFaviconPath(),
		IconPath:    i.GetIconPath(),
	}
}

func colorsFromProto(c *brandsv1.Colors) brands.Colors {
	if c == nil {
		return brands.Colors{}
	}
	return brands.Colors{
		Primary:    c.GetPrimary(),
		Secondary:  c.GetSecondary(),
		Accent:     c.GetAccent(),
		Background: c.GetBackground(),
		Surface:    c.GetSurface(),
		Text:       c.GetText(),
		Error:      c.GetError(),
	}
}

func typographyFromProto(t *brandsv1.Typography) brands.Typography {
	if t == nil {
		return brands.Typography{}
	}
	return brands.Typography{
		HeadingFont:  t.GetHeadingFont(),
		BodyFont:     t.GetBodyFont(),
		MonoFont:     t.GetMonoFont(),
		BaseFontSize: t.GetBaseFontSize(),
	}
}

func voiceFromProto(v *brandsv1.Voice) brands.Voice {
	if v == nil {
		return brands.Voice{}
	}
	return brands.Voice{
		Tone:     v.GetTone(),
		Style:    v.GetStyle(),
		Keywords: append([]string(nil), v.GetKeywords()...),
	}
}
