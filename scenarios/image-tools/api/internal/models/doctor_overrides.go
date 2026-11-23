package models

import (
	"fmt"
	"sort"

	"image-tools/internal/catalogstate"
)

// DoctorOverrides reports the stored overrides a person should know about: an
// override that contradicts the seed (so the catalog does not behave as its seed
// reads), and an override a later seed change superseded. It never fails the
// catalog: an override is a legitimate operator choice, and the finding exists
// so it is never invisible.
//
//   - an unpinned override of unknown authorship that contradicts the seed is a
//     warning: nobody can say why it is there, and it is exactly the shape of the
//     stale override that kept RestoreFormer++ disabled after the seed enabled it;
//   - any other contradicting override is informational, naming its author, when
//     and why, and that a seed change will supersede it unless it is pinned;
//   - a superseded override is informational: the seed won.
func DoctorOverrides(overrides map[string]catalogstate.Override, seed catalogstate.SeedState) []CatalogFinding {
	ids := make([]string, 0, len(overrides))
	for id := range overrides {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var findings []CatalogFinding
	for _, id := range ids {
		o := overrides[id]
		seedEnabled, known := seed(id)
		if !known {
			continue
		}
		who := o.SetBy
		if who == "" {
			who = catalogstate.UnknownActor
		}
		switch {
		case !o.Active():
			findings = append(findings, CatalogFinding{
				Severity: FindingInfo, Code: "override_superseded", ModelID: id,
				Message: fmt.Sprintf("override by %s no longer applies: %s", who, o.SupersededNote),
			})
		case o.Contradicts(seedEnabled) && !o.Pinned && who == catalogstate.UnknownActor:
			findings = append(findings, CatalogFinding{
				Severity: FindingWarning, Code: "override_contradicts_seed", ModelID: id,
				Message: fmt.Sprintf("an override with no recorded author sets enabled=%t, but the seed says enabled=%t; keep it with `image-tools models enable %s%s --pin --reason \"...\"` or drop it with `image-tools models reset %s`",
					o.Enabled, seedEnabled, id, disableFlag(o.Enabled), id),
			})
		case o.Contradicts(seedEnabled):
			message := fmt.Sprintf("override by %s sets enabled=%t against the seed's enabled=%t", who, o.Enabled, seedEnabled)
			if o.Reason != "" {
				message += " (" + o.Reason + ")"
			}
			if o.Pinned {
				message += "; pinned, so it stays"
			} else {
				message += "; a seed change will supersede it unless it is pinned"
			}
			findings = append(findings, CatalogFinding{Severity: FindingInfo, Code: "override_contradicts_seed", ModelID: id, Message: message})
		}
	}
	return findings
}

func disableFlag(enabled bool) string {
	if enabled {
		return ""
	}
	return " --disable"
}
