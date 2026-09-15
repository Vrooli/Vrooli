package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func rankProvenanceSearch(groups []workspaceSandboxProvenanceRunGroup, rawQuery string, limit int) []ProvenanceSearchHit {
	terms := strings.Fields(strings.ToLower(rawQuery))
	results := make([]ProvenanceSearchHit, 0)
	for _, group := range groups {
		for _, file := range group.Files {
			if strings.EqualFold(strings.TrimSpace(file.Visibility), "private") {
				continue
			}
			text := strings.ToLower(strings.Join([]string{group.RunID, group.SandboxID, group.SandboxOwner, file.FilePath, file.RelativePath, file.ChangeType}, " "))
			matched := 0
			for _, term := range terms {
				if strings.Contains(text, term) {
					matched++
				}
			}
			if matched == 0 {
				continue
			}
			key := group.RunID + "\x00" + group.SandboxID + "\x00" + file.RelativePath
			digest := sha256.Sum256([]byte(key))
			results = append(results, ProvenanceSearchHit{
				ID:               "gct-provenance:" + hex.EncodeToString(digest[:])[:24],
				Title:            file.RelativePath,
				Snippet:          "run=" + group.RunID + " change=" + file.ChangeType,
				Score:            float64(matched) / float64(len(terms)),
				RunID:            group.RunID,
				SandboxID:        group.SandboxID,
				RelativePath:     file.RelativePath,
				EvidenceStanding: "applied",
			})
		}
	}
	// Stable ordering makes exact lookups deterministic without claiming a
	// semantic score. Ties retain provider order, which is owner-supplied.
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score || (results[j].Score == results[i].Score && results[j].ID < results[i].ID) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}
