package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/contracts"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/presentation"
)

// SEOServicer is the transport-facing SEO contract.
type SEOServicer interface {
	VariantSEO(slug string) (*content.SEOResponse, error)
	SitemapXML(fallbackBase string) (string, error)
	RobotsTXT(fallbackBase string) (string, error)
}

var _ SEOServicer = (*content.SEOService)(nil)

type seoConfigStoreAdapter struct{ store *experimentation.ConfigStore }

func (a seoConfigStoreAdapter) Branding() content.SEOBranding {
	b := a.store.GetBranding()
	return content.SEOBranding{
		SiteName: b.SiteName, DefaultTitle: b.DefaultTitle, DefaultDescription: b.DefaultDescription,
		DefaultOGImageURL: b.DefaultOGImageURL, FaviconURL: b.FaviconURL,
		AppleTouchIconURL: b.AppleTouchIconURL, ThemePrimaryColor: b.ThemePrimaryColor,
		CanonicalBaseURL: b.CanonicalBaseURL, RobotsTxt: b.RobotsTxt,
	}
}

func (a seoConfigStoreAdapter) Variant(slug string) (content.SEOVariant, error) {
	v, err := a.store.GetVariant(slug)
	if err != nil {
		return content.SEOVariant{}, err
	}
	return content.SEOVariant{Slug: v.Variant.Slug, SEOConfig: v.Variant.SEOConfig}, nil
}

func (a seoConfigStoreAdapter) Variants() []content.SEOVariant {
	variants := a.store.ListVariants()
	result := make([]content.SEOVariant, 0, len(variants))
	for _, variant := range variants {
		result = append(result, content.SEOVariant{Slug: variant.Variant.Slug, SEOConfig: variant.Variant.SEOConfig})
	}
	return result
}

func (a seoConfigStoreAdapter) UpdateVariantSEO(slug string, config contracts.VariantSEOConfig) error {
	variant, err := a.store.GetVariant(slug)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	variant.Variant.SEOConfig = encoded
	return a.store.SaveVariant(slug, variant)
}

func (a seoConfigStoreAdapter) PublishedPublicRoutes(ctx context.Context) (content.PublishedRouteSet, error) {
	selectable := make([]*experimentation.VariantSnapshot, 0)
	for _, variant := range a.store.ListVariants() {
		if experimentation.VariantWeight(variant) > 0 {
			selectable = append(selectable, variant)
		}
	}
	if len(selectable) == 0 {
		return content.PublishedRouteSet{Authoritative: true}, nil
	}

	var common map[string]struct{}
	for index, variant := range selectable {
		published, err := a.store.GetPublishedPresentation(ctx, variant.Variant.Slug)
		if err != nil {
			if errors.Is(err, experimentation.ErrPresentationNotFound) {
				return content.PublishedRouteSet{Authoritative: true}, nil
			}
			return content.PublishedRouteSet{}, err
		}
		if err := presentation.Validate(published.Document); err != nil {
			return content.PublishedRouteSet{}, err
		}
		current, err := publicPresentationDetailPaths(published.Document)
		if err != nil {
			return content.PublishedRouteSet{}, err
		}
		if index == 0 {
			common = current
			continue
		}
		for path := range common {
			if _, exists := current[path]; !exists {
				delete(common, path)
			}
		}
	}

	paths := make([]string, 0, len(common))
	for path := range common {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return content.PublishedRouteSet{Authoritative: true, Root: true, AppDetailPaths: paths}, nil
}

func publicPresentationDetailPaths(document presentation.Document) (map[string]struct{}, error) {
	paths := make(map[string]struct{})
	views, err := presentation.PublicViews(document)
	if err != nil {
		return nil, err
	}
	for _, view := range views {
		if view.Diagnostics.ResolvedRoute != "/" {
			paths[view.Diagnostics.ResolvedRoute] = struct{}{}
		}
	}
	return paths, nil
}

// NewSEOService constructs content-domain SEO policy over the JSON configuration adapter.
func NewSEOService(store *experimentation.ConfigStore) *content.SEOService {
	return content.NewSEOService(seoConfigStoreAdapter{store: store}, logx.Error)
}
