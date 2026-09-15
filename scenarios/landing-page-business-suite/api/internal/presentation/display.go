package presentation

import (
	"encoding/json"
	"fmt"
)

// PageDisplay is finite, locale-specific renderer vocabulary. It is stored in
// the same immutable page revision as its blocks: there is no runtime copy
// sidecar, executable component slot, CSS string, or inferred product identity.
type PageDisplay struct {
	Shell          ShellDisplay              `json:"shell"`
	AssetLabels    map[string]AssetLabel     `json:"asset_labels"`
	FixtureDisplay map[string]FixtureDisplay `json:"fixture_display"`
	Blocks         map[string]BlockDisplay   `json:"blocks"`
	Apps           map[string]AppDisplay     `json:"apps"`
}

type ShellDisplay struct {
	BrandName         string  `json:"brand_name"`
	BrandMark         string  `json:"brand_mark"`
	BrandTarget       string  `json:"brand_target"`
	BrandSubtitle     string  `json:"brand_subtitle,omitempty"`
	SkipLabel         string  `json:"skip_label"`
	MenuLabel         string  `json:"menu_label"`
	FooterBrandName   string  `json:"footer_brand_name"`
	FooterBrandMark   string  `json:"footer_brand_mark"`
	FooterBrandTarget string  `json:"footer_brand_target"`
	FooterTagline     string  `json:"footer_tagline,omitempty"`
	Copyright         string  `json:"copyright,omitempty"`
	FooterNote        string  `json:"footer_note,omitempty"`
	UnavailableReason string  `json:"unavailable_reason"`
	PreviewLabel      string  `json:"preview_label"`
	HeaderAction      *Action `json:"header_action,omitempty"`
}

type AssetLabel struct {
	Alt   string `json:"alt"`
	Sizes string `json:"sizes,omitempty"`
}

type FixtureDisplay struct {
	Mark          string            `json:"mark"`
	Avatar        string            `json:"avatar,omitempty"`
	Time          string            `json:"time,omitempty"`
	TabsLabel     string            `json:"tabs_label,omitempty"`
	TerminalLabel string            `json:"terminal_label,omitempty"`
	MessagesLabel string            `json:"messages_label,omitempty"`
	FileChanges   map[string]string `json:"file_changes,omitempty"`
}

type BlockDisplay struct {
	Eyebrow            string            `json:"eyebrow,omitempty"`
	Description        string            `json:"description,omitempty"`
	Note               string            `json:"note,omitempty"`
	AccessibilityLabel string            `json:"accessibility_label,omitempty"`
	Badge              string            `json:"badge,omitempty"`
	Mark               string            `json:"mark,omitempty"`
	Formats            []string          `json:"formats,omitempty"`
	Anchors            map[string]string `json:"anchors,omitempty"`
	HeadingBreaks      []int             `json:"heading_breaks,omitempty"`
	FixtureRef         string            `json:"fixture_ref,omitempty"`
	HeroFixtureRefs    map[string]string `json:"hero_fixture_refs,omitempty"`
}

type AppDisplay struct {
	FixtureRef  string `json:"fixture_ref,omitempty"`
	VisualRef   string `json:"visual_ref,omitempty"`
	Mark        string `json:"mark"`
	Tone        string `json:"tone"`
	DetailLabel string `json:"detail_label"`
}

var displayMarks = map[string]bool{"letter-a": true, "landscape": true, "suite": true, "play": true}

func validateDisplay(page Page, path string, apps map[string]App, capabilities map[string]Capability, assets map[string]Asset, fixtures map[string]Fixture, issues *ValidationError) {
	display := page.Display
	shell := display.Shell
	for key, value := range map[string]string{"brand_name": shell.BrandName, "skip_label": shell.SkipLabel, "menu_label": shell.MenuLabel, "footer_brand_name": shell.FooterBrandName, "unavailable_reason": shell.UnavailableReason, "preview_label": shell.PreviewLabel} {
		validateText(value, path+".shell."+key, issues, true)
	}
	for key, value := range map[string]string{"brand_subtitle": shell.BrandSubtitle, "footer_tagline": shell.FooterTagline, "copyright": shell.Copyright, "footer_note": shell.FooterNote} {
		validateText(value, path+".shell."+key, issues, false)
	}
	for key, mark := range map[string]string{"brand_mark": shell.BrandMark, "footer_brand_mark": shell.FooterBrandMark} {
		if !displayMarks[mark] {
			issues.add(path+".shell."+key, "invalid_display_mark", "must select a registered product mark")
		}
	}
	for key, target := range map[string]string{"brand_target": shell.BrandTarget, "footer_brand_target": shell.FooterBrandTarget} {
		if !isSafePublicURL(target) {
			issues.add(path+".shell."+key, "unsafe_display_target", "must be a same-origin path")
		}
	}
	if shell.HeaderAction != nil {
		data, _ := json.Marshal([]Action{*shell.HeaderAction})
		var actions []any
		_ = json.Unmarshal(data, &actions)
		validateActions(actions, path+".shell.header_action", issues)
		if shell.HeaderAction.AppKey != "" {
			if _, ok := apps[shell.HeaderAction.AppKey]; !ok {
				issues.add(path+".shell.header_action.app_key", "unknown_app_ref", "app is not declared")
			}
		}
	}
	blocks := map[string]Block{}
	for _, block := range page.Blocks {
		blocks[block.ID] = block
	}
	for _, id := range sortedStringKeys(display.Blocks) {
		value := display.Blocks[id]
		p := path + ".blocks[" + id + "]"
		block, ok := blocks[id]
		if !ok {
			issues.add(p, "unknown_block_ref", "display must reference a block on this page")
		}
		for key, text := range map[string]string{"eyebrow": value.Eyebrow, "description": value.Description, "note": value.Note, "accessibility_label": value.AccessibilityLabel, "badge": value.Badge} {
			validateText(text, p+"."+key, issues, false)
		}
		if value.Mark != "" && !displayMarks[value.Mark] {
			issues.add(p+".mark", "invalid_display_mark", "must select a registered product mark")
		}
		validateTextList(value.Formats, p+".formats", issues)
		last := -1
		for i, offset := range value.HeadingBreaks {
			if offset <= last || offset < 0 {
				issues.add(fmt.Sprintf("%s.heading_breaks[%d]", p, i), "invalid_heading_break", "offsets must be nonnegative and strictly increasing")
			}
			last = offset
		}
		for key, target := range value.Anchors {
			if _, ok := capabilities[key]; !ok {
				issues.add(p+".anchors", "unknown_capability_ref", "anchor capability is not declared")
			}
			if len(target) < 2 || target[0] != '#' || !idPattern.MatchString(target[1:]) {
				issues.add(p+".anchors", "unsafe_display_target", "capability anchor must identify a page block")
			} else if _, ok := blocks[target[1:]]; !ok {
				issues.add(p+".anchors", "unknown_block_ref", "anchor block is not on this page")
			}
		}
		if value.FixtureRef != "" {
			if block.Kind != BlockDeviceStory {
				issues.add(p+".fixture_ref", "invalid_display_reference", "only device-story uses this display fixture slot")
			}
			if _, ok := fixtures[value.FixtureRef]; !ok {
				issues.add(p+".fixture_ref", "unknown_fixture_ref", "fixture is not declared")
			}
		}
		heroApps := map[string]bool{}
		if hero, ok := block.Content.(BundleHeroContent); ok {
			for _, item := range hero.HeroItems {
				heroApps[item.AppKey] = true
			}
		}
		for key, ref := range value.HeroFixtureRefs {
			if !heroApps[key] {
				issues.add(p+".hero_fixture_refs", "unknown_hero_app_ref", "fixture must belong to a configured hero item")
			}
			if _, ok := fixtures[ref]; !ok {
				issues.add(p+".hero_fixture_refs", "unknown_fixture_ref", "fixture is not declared")
			}
		}
	}
	for key, value := range display.Apps {
		p := path + ".apps[" + key + "]"
		if _, ok := apps[key]; !ok {
			issues.add(p, "unknown_app_ref", "app is not declared")
		}
		if !displayMarks[value.Mark] {
			issues.add(p+".mark", "invalid_display_mark", "must select a registered product mark")
		}
		if value.Tone != "amber" && value.Tone != "sage" {
			issues.add(p+".tone", "invalid_display_tone", "must select amber or sage")
		}
		validateText(value.DetailLabel, p+".detail_label", issues, true)
		if (value.FixtureRef == "") == (value.VisualRef == "") {
			issues.add(p, "invalid_exhibit_ref", "select exactly one fixture or released visual")
		}
		if value.FixtureRef != "" {
			if _, ok := fixtures[value.FixtureRef]; !ok {
				issues.add(p+".fixture_ref", "unknown_fixture_ref", "fixture is not declared")
			}
		}
		if value.VisualRef != "" {
			if _, ok := assets[value.VisualRef]; !ok {
				issues.add(p+".visual_ref", "unknown_asset_ref", "asset is not declared")
			}
		}
	}
	for _, block := range page.Blocks {
		var appKeys []string
		switch value := block.Content.(type) {
		case DeviceStoryContent:
			if (value.VisualRef == "") == (display.Blocks[block.ID].FixtureRef == "") {
				issues.add(path+".blocks["+block.ID+"]", "invalid_exhibit_ref", "device story requires exactly one configured fixture or released visual")
			}
		case AppSpotlightsContent:
			appKeys = value.AppKeys
		case BundleHeroContent:
			for _, item := range value.HeroItems {
				appKeys = append(appKeys, item.AppKey)
				if (item.VisualRef == "") == (display.Blocks[block.ID].HeroFixtureRefs[item.AppKey] == "") {
					issues.add(path+".blocks["+block.ID+"].hero_fixture_refs", "invalid_exhibit_ref", "hero item requires exactly one configured fixture or released visual")
				}
			}
		}
		for _, key := range appKeys {
			if _, ok := display.Apps[key]; !ok {
				issues.add(path+".apps", "missing_app_display", "every configured spotlight and hero item requires an app exhibit")
			}
		}
	}
	for id, label := range display.AssetLabels {
		if _, ok := assets[id]; !ok {
			issues.add(path+".asset_labels["+id+"]", "unknown_asset_ref", "asset is not declared")
		}
		validateText(label.Alt, path+".asset_labels["+id+"].alt", issues, false)
		validateText(label.Sizes, path+".asset_labels["+id+"].sizes", issues, false)
	}
	for _, id := range sortedStringKeys(referencedPageAssets(page, assets, fixtures)) {
		if _, ok := display.AssetLabels[id]; !ok {
			issues.add(path+".asset_labels["+id+"]", "missing_asset_display_label", "every referenced asset requires a configured display label")
		}
	}
	for id, value := range display.FixtureDisplay {
		p := path + ".fixture_display[" + id + "]"
		fixture, ok := fixtures[id]
		if !ok {
			issues.add(p, "unknown_fixture_ref", "fixture is not declared")
		}
		if !displayMarks[value.Mark] {
			issues.add(p+".mark", "invalid_display_mark", "must select a registered product mark")
		}
		if fixture.Kind == FixtureWorkspace {
			for key, label := range map[string]string{"avatar": value.Avatar, "time": value.Time, "tabs_label": value.TabsLabel, "terminal_label": value.TerminalLabel, "messages_label": value.MessagesLabel} {
				validateText(label, p+"."+key, issues, true)
			}
			for key, label := range value.FileChanges {
				validateText(key, p+".file_changes", issues, true)
				validateText(label, p+".file_changes", issues, true)
			}
		}
	}
	refs := displayFixtureRefs(display)
	for _, block := range page.Blocks {
		if ref := fixtureRef(block.Content); ref != "" {
			refs[ref] = true
		}
	}
	for ref := range refs {
		if fixture, ok := fixtures[ref]; ok && fixture.Kind != FixtureWorkflow {
			if _, ok := display.FixtureDisplay[ref]; !ok {
				issues.add(path+".fixture_display", "missing_fixture_display", "every referenced product fixture requires configured display labels")
			}
		}
	}
}

// referencedPageAssets mirrors the resource closure used by resolvedAssets.
// Labels are required only for assets that this page can resolve, not for
// unrelated assets retained elsewhere in the document.
func referencedPageAssets(page Page, assets map[string]Asset, fixtures map[string]Fixture) map[string]bool {
	refs := map[string]bool{}
	add := func(ref string) {
		if ref != "" {
			refs[ref] = true
		}
	}
	for _, app := range page.Display.Apps {
		add(app.VisualRef)
	}
	fixtureRefs := displayFixtureRefs(page.Display)
	for _, block := range page.Blocks {
		for _, ref := range assetRefs(block.Content) {
			add(ref)
		}
		if ref := fixtureRef(block.Content); ref != "" {
			fixtureRefs[ref] = true
		}
	}
	for ref := range fixtureRefs {
		fixture, ok := fixtures[ref]
		if !ok || fixture.Backdrop == nil {
			continue
		}
		for _, assetRef := range fixture.Backdrop.AssetRefs {
			add(assetRef)
		}
	}
	for changed := true; changed; {
		changed = false
		for id := range refs {
			asset, ok := assets[id]
			if !ok {
				continue
			}
			for _, alternative := range asset.ResponsiveAlternatives {
				if alternative.AssetID != "" && !refs[alternative.AssetID] {
					refs[alternative.AssetID] = true
					changed = true
				}
			}
		}
	}
	return refs
}

func displayFixtureRefs(display PageDisplay) map[string]bool {
	refs := map[string]bool{}
	for _, block := range display.Blocks {
		if block.FixtureRef != "" {
			refs[block.FixtureRef] = true
		}
		for _, ref := range block.HeroFixtureRefs {
			refs[ref] = true
		}
	}
	for _, app := range display.Apps {
		if app.FixtureRef != "" {
			refs[app.FixtureRef] = true
		}
	}
	return refs
}

func projectDisplay(display PageDisplay, blocks []ResolvedBlock, selected []string, eligible map[string]bool, publicSlugs map[string]bool) PageDisplay {
	selectedSet := map[string]bool{}
	for _, key := range selected {
		selectedSet[key] = true
	}
	apps := map[string]AppDisplay{}
	for key, value := range display.Apps {
		if selectedSet[key] {
			apps[key] = value
		}
	}
	display.Apps = apps
	visibleBlocks := map[string]BlockDisplay{}
	for _, block := range blocks {
		if value, ok := display.Blocks[block.ID]; ok {
			refs := map[string]string{}
			for key, ref := range value.HeroFixtureRefs {
				if selectedSet[key] {
					refs[key] = ref
				}
			}
			value.HeroFixtureRefs = refs
			visibleBlocks[block.ID] = value
		}
	}
	display.Blocks = visibleBlocks
	if display.Shell.HeaderAction != nil {
		actions := publicActions([]Action{*display.Shell.HeaderAction}, eligible, publicSlugs)
		display.Shell.HeaderAction = nil
		if len(actions) == 1 {
			display.Shell.HeaderAction = &actions[0]
		}
	}
	return display
}

func closeDisplayResources(display *PageDisplay, fixtures []Fixture, assets []ResolvedAsset) {
	fixtureDisplay := map[string]FixtureDisplay{}
	for _, fixture := range fixtures {
		if value, ok := display.FixtureDisplay[fixture.ID]; ok {
			fixtureDisplay[fixture.ID] = value
		}
	}
	display.FixtureDisplay = fixtureDisplay
	labels := map[string]AssetLabel{}
	for _, asset := range assets {
		if value, ok := display.AssetLabels[asset.ID]; ok {
			labels[asset.ID] = value
		}
	}
	display.AssetLabels = labels
}
