package presentation

// Finite content vocabulary: these are semantic references, never an arbitrary
// JSON-key scan. A block with private app claims is removed as a whole so its
// surrounding heading/description cannot leak the private product narrative.
func blockCapabilityRefs(content BlockContent) []string {
	var refs []string
	switch value := content.(type) {
	case CapabilityStripContent:
		for _, item := range value.Items {
			refs = append(refs, item.CapabilityID)
		}
	case ArtifactExplorerContent:
		refs = append(refs, value.CapabilityID)
	case VoiceStoryContent:
		refs = append(refs, value.CapabilityIDs...)
		for _, feature := range value.Features {
			refs = append(refs, feature.CapabilityID)
		}
	case CapabilityRoadmapContent:
		refs = append(refs, value.CapabilityIDs...)
	}
	return refs
}

func publicCapabilities(apps []App) map[string]bool {
	result := map[string]bool{}
	for _, app := range apps {
		for _, capability := range app.Capabilities {
			result[capability.ID] = true
		}
	}
	return result
}

func blockCapabilitiesPublic(content BlockContent, capabilities map[string]bool) bool {
	for _, ref := range blockCapabilityRefs(content) {
		if !capabilities[ref] {
			return false
		}
	}
	return true
}

func pageCapabilityKeys(page ResolvedPage, profileKeys []string, eligible []App) []string {
	result := copyStrings(profileKeys)
	seen, referenced := map[string]bool{}, map[string]bool{}
	for _, key := range result {
		seen[key] = true
	}
	for _, block := range page.Blocks {
		for _, ref := range blockCapabilityRefs(block.Content) {
			referenced[ref] = true
		}
	}
	for _, app := range eligible {
		if seen[app.Key] {
			continue
		}
		for _, capability := range app.Capabilities {
			if referenced[capability.ID] {
				result = append(result, app.Key)
				seen[app.Key] = true
				break
			}
		}
	}
	return result
}

func pruneDroppedAnchors(page *ResolvedPage, dropped, capabilities map[string]bool) {
	links := func(values []NavigationItem) []NavigationItem {
		result := make([]NavigationItem, 0, len(values))
		for _, value := range values {
			if !dropped[value.Target] {
				result = append(result, value)
			}
		}
		return result
	}
	actions := func(values []Action) []Action {
		result := make([]Action, 0, len(values))
		for _, value := range values {
			if !dropped[value.Target] {
				result = append(result, value)
			}
		}
		return result
	}
	page.Navigation.Items, page.Footer.Links = links(page.Navigation.Items), links(page.Footer.Links)
	if action := page.Display.Shell.HeaderAction; action != nil && dropped[action.Target] {
		page.Display.Shell.HeaderAction = nil
	}
	for i, block := range page.Blocks {
		switch value := block.Content.(type) {
		case ProductHeroContent:
			value.Actions = actions(value.Actions)
			page.Blocks[i].Content = value
		case BundleHeroContent:
			value.Actions = actions(value.Actions)
			page.Blocks[i].Content = value
		case PricingContent:
			value.Actions = actions(value.Actions)
			page.Blocks[i].Content = value
		case ClosingActionContent:
			value.Actions = actions(value.Actions)
			page.Blocks[i].Content = value
		case FooterContent:
			value.Links = links(value.Links)
			page.Blocks[i].Content = value
		}
	}
	for id, display := range page.Display.Blocks {
		for ref, target := range display.Anchors {
			if !capabilities[ref] || dropped[target] {
				delete(display.Anchors, ref)
			}
		}
		page.Display.Blocks[id] = display
	}
}
