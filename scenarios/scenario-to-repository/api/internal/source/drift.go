package source

// UpdatePreview keeps source evolution, policy/recipe changes, and destination
// drift separate. A preview never authorizes overwrite or history mutation.
type UpdatePreview struct {
	SourceChanged    bool     `json:"sourceChanged"`
	RecipeChanged    bool     `json:"recipeChanged"`
	PolicyChanged    bool     `json:"policyChanged"`
	DestinationDrift bool     `json:"destinationDrift"`
	CurrentSource    string   `json:"currentSource"`
	PreviousSource   string   `json:"previousSource"`
	CurrentArtifact  string   `json:"currentArtifact"`
	PreviousArtifact string   `json:"previousArtifact"`
	Actions          []string `json:"actions"`
	Disposition      string   `json:"disposition"`
}

func CompareUpdate(currentSource, previousSource, currentRecipe, previousRecipe, currentPolicy, previousPolicy, currentArtifact, previousArtifact, destinationDigest, destinationManifest string) UpdatePreview {
	p := UpdatePreview{CurrentSource: currentSource, PreviousSource: previousSource, CurrentArtifact: currentArtifact, PreviousArtifact: previousArtifact, Disposition: "unchanged"}
	p.SourceChanged = currentSource != previousSource
	p.RecipeChanged = currentRecipe != previousRecipe
	p.PolicyChanged = currentPolicy != previousPolicy
	p.DestinationDrift = destinationDigest != "" && destinationDigest != previousArtifact
	if p.SourceChanged {
		p.Actions = append(p.Actions, "review source changes")
	}
	if p.RecipeChanged {
		p.Actions = append(p.Actions, "review recipe migration")
	}
	if p.PolicyChanged {
		p.Actions = append(p.Actions, "recompute publishability")
	}
	if p.DestinationDrift || (destinationManifest != "" && destinationManifest != previousArtifact) {
		p.DestinationDrift = true
		p.Actions = append(p.Actions, "inspect independent destination edits")
	}
	if p.SourceChanged || p.RecipeChanged || p.PolicyChanged || p.DestinationDrift {
		p.Disposition = "review_required"
		p.Actions = append(p.Actions, "require a new human publication decision")
	}
	return p
}
