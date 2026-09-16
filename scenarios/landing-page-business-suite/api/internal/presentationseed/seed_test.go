package presentationseed

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"landing-page-business-suite-api/internal/presentation"
)

func TestRecommendedSignalStudioSeedDecodesAndValidates(t *testing.T) {
	document, err := Recommended()
	require.NoError(t, err)
	require.Equal(t, []string{"web-console", "browser-automation-studio", "backdrop-studio"}, document.Bundle.AppOrder)

	apps := make(map[string]struct {
		enabled     bool
		visibility  string
		publication string
	}, len(document.Apps))
	for _, app := range document.Apps {
		apps[app.Key] = struct {
			enabled     bool
			visibility  string
			publication string
		}{app.Enabled, string(app.Visibility), string(app.Publication)}
	}
	require.Equal(t, struct {
		enabled     bool
		visibility  string
		publication string
	}{true, "public", "published"}, apps["web-console"])
	require.Equal(t, struct {
		enabled     bool
		visibility  string
		publication string
	}{false, "private", "draft"}, apps["browser-automation-studio"])
	require.Equal(t, struct {
		enabled     bool
		visibility  string
		publication string
	}{false, "private", "draft"}, apps["backdrop-studio"])

	require.Contains(t, document.Strings["en"]["bas.preservation.source_ref"], "presentation-preservation/inventory.json#after.profile")
	require.Equal(t, "bas-disabled-profile-migration-v1", document.Strings["en"]["bas.preservation.inventory_id"])
	require.Equal(t, "11411b7923ceb2564935ea09710fc0c9c3655471efcaf89a41899b237d827c6e", document.Strings["en"]["bas.preservation.profile_digest_sha256"])
	require.Contains(t, document.Strings["en"]["bas.preservation.fallback_source_ref"], "fallback/fallback.json#browser-automation-studio")
	require.Equal(t, "bas-disabled-profile-migration-v1", document.Apps[1].PreservationRef)
	require.Contains(t, document.Strings["en"]["bas.preservation.asset_ledger_ref"], "inventory.json#assetLedger")
	require.Contains(t, document.Strings["en"]["bas.preservation.unlocated_artifacts_ref"], "inventory.json#unlocatedArtifacts")
	require.Contains(t, document.Strings["en"]["bas.preservation.platforms"], "downloads.vrooli.local")
	require.Contains(t, document.Strings["en"]["bas.preservation.asset_note"], "never public Asset.public_url")
	require.NotContains(t, document.Strings["en"]["bas.preservation.testimonials"], "already using")
	for _, asset := range document.Assets {
		require.Empty(t, asset.PublicURL, "candidate review assets must not be treated as released public assets")
	}

	for _, capability := range document.Apps[0].Capabilities {
		if capability.Status == "coming-soon" {
			require.True(t, strings.Contains(capability.Label, "Remote") || strings.Contains(capability.Label, "Android") || strings.Contains(capability.Label, "iPhone"))
		}
	}
}

func TestRecommendedSignalStudioBackdropStudiesPairStylesWithAssets(t *testing.T) {
	document := loadSeed(t)
	var backdrop *presentation.BackdropFixture
	for i := range document.Fixtures {
		if document.Fixtures[i].ID == "backdrop-studies" {
			backdrop = document.Fixtures[i].Backdrop
			break
		}
	}
	require.NotNil(t, backdrop)
	require.Equal(t, []string{"Survey Relief", "Pale Moon", "Tidal Halftone"}, backdrop.Styles)
	require.Equal(t, []string{"survey-relief", "pale-moon", "tidal-halftone"}, backdrop.AssetRefs)
	require.Contains(t, backdrop.Styles, backdrop.Selected)
}

func TestRecommendedSignalStudioBackdropHeroActionTargetsDetails(t *testing.T) {
	document := loadSeed(t)
	page := seedPage(t, document, "backdrop-studio")
	require.NotEmpty(t, page.Blocks)
	hero, ok := page.Blocks[0].Content.(presentation.ProductHeroContent)
	require.True(t, ok)
	require.Len(t, hero.Actions, 1)
	require.Equal(t, presentation.ActionAnchor, hero.Actions[0].Kind)
	require.Equal(t, "Explore visual surfaces", hero.Actions[0].Label)
	require.Equal(t, "Explore visual surfaces", hero.Actions[0].AccessibleLabel)
	require.Equal(t, "#details", hero.Actions[0].Target)
	require.Empty(t, hero.Actions[0].AppKey)
}

func TestRecommendedSignalStudioBackdropDisplayLabelsCoverDetailAndBundle(t *testing.T) {
	assetIDs := []string{"survey-relief", "pale-moon", "tidal-halftone"}
	for _, assetID := range assetIDs {
		t.Run("missing "+assetID, func(t *testing.T) {
			document := loadSeed(t)
			page := seedPage(t, document, "backdrop-studio")
			delete(page.Display.AssetLabels, assetID)
			err := document.Validate()
			var validation *presentation.ValidationError
			require.ErrorAs(t, err, &validation)
			found := false
			for _, issue := range validation.Issues {
				if issue.Code == "missing_asset_display_label" && strings.Contains(issue.Path, assetID) {
					found = true
					break
				}
			}
			require.True(t, found, "missing label issue for %s: %#v", assetID, validation.Issues)
		})
	}

	document := loadSeed(t)
	detail, err := presentation.Resolve(document, presentation.ResolveRequest{
		Route: "/apps/backdrop-studio", PreviewRevision: "recommended-draft-v1", PreviewAuthorized: true,
	})
	require.NoError(t, err)
	require.Equal(t, "backdrop-studio", detail.AppKey)
	require.Len(t, detail.Assets, len(assetIDs))
	require.Len(t, detail.Page.Display.AssetLabels, len(assetIDs))

	for i := range document.Apps {
		if document.Apps[i].Key == "web-console" || document.Apps[i].Key == "backdrop-studio" {
			document.Apps[i].Enabled = true
			document.Apps[i].Visibility = presentation.VisibilityPublic
			document.Apps[i].Publication = presentation.PublicationPublished
		}
	}
	bundle, err := presentation.Resolve(document, presentation.ResolveRequest{Route: "/"})
	require.NoError(t, err)
	require.Equal(t, presentation.ModeBundle, bundle.Diagnostics.Mode)
	require.Equal(t, "studio-bundle", bundle.Page.ID)
}

func seedPage(t *testing.T, document presentation.Document, id string) *presentation.Page {
	t.Helper()
	for i := range document.Pages {
		if document.Pages[i].ID == id {
			return &document.Pages[i]
		}
	}
	t.Fatalf("seed page %q not found", id)
	return nil
}

func TestRecommendedSignalStudioSeedResolvesAfterExplicitAquilaPromotion(t *testing.T) {
	document := loadSeed(t)
	document.Apps[0].Publication = presentation.PublicationPublished
	require.NoError(t, document.Validate())

	result, err := presentation.Resolve(document, presentation.ResolveRequest{Route: "/"})
	require.NoError(t, err)
	require.Equal(t, presentation.ModeSingleApp, result.Diagnostics.Mode)
	require.Equal(t, "web-console", result.AppKey)
	require.Equal(t, []string{"web-console"}, result.Diagnostics.EligibleAppKeys)
}

func TestRecommendedSignalStudioSeedPrivateDetailPreviewRoundTrips(t *testing.T) {
	document := loadSeed(t)
	for _, appKey := range []string{"browser-automation-studio", "backdrop-studio"} {
		route := "/apps/" + appKey
		_, err := presentation.Resolve(document, presentation.ResolveRequest{Route: route})
		require.ErrorIs(t, err, presentation.ErrNotFound)

		result, err := presentation.Resolve(document, presentation.ResolveRequest{
			Route:             route,
			PreviewRevision:   "recommended-draft-v1",
			PreviewAuthorized: true,
		})
		require.NoError(t, err)
		require.Equal(t, appKey, result.AppKey)
		require.Equal(t, presentation.ModeAppDetail, result.Diagnostics.Mode)
		require.True(t, result.Diagnostics.Preview)
		require.True(t, result.Diagnostics.NoIndex)
		require.True(t, result.Diagnostics.NoStore)
		require.NotEmpty(t, result.Page.Blocks)
	}

	_, err := presentation.Resolve(document, presentation.ResolveRequest{
		Route:           "/apps/browser-automation-studio",
		PreviewRevision: "recommended-draft-v1",
	})
	require.True(t, errors.Is(err, presentation.ErrPreviewUnauthorized))
}

func loadSeed(t *testing.T) presentation.Document {
	t.Helper()
	document, err := Recommended()
	require.NoError(t, err)
	return document
}
