package corpus

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	RequiredQualityClips       = 60
	RequiredQualityRefWords    = 3000
	RequiredQualitySpeakers    = 6
	RequiredCleanClips         = 1
	RequiredNoisy20dBClips     = 1
	RequiredNoisy10dBClips     = 1
	RequiredQualityCorpusSplit = SplitHeldOut
)

// QualityCohortReport is the provenance-only qualification of a corpus
// cohort. It does not grade WER; it proves that a later report is entitled to
// make a clean/noisy held-out quality claim.
type QualityCohortReport struct {
	Eligible        bool
	Revision        string
	ClipCount       int
	ReferenceWords  int
	SpeakerCount    int
	CleanCount      int
	Noisy20dBCount  int
	Noisy10dBCount  int
	LicensedCount   int
	DuplicateHashes int
	Reasons         []string
}

// AssessQualityCohort applies the adopted voice-pilot-en-v1 corpus recipe to
// persisted clip metadata. Missing provenance is a hard failure; callers must
// not turn a partial/local corpus into an acceptance receipt.
func AssessQualityCohort(clips []Clip, revision string) QualityCohortReport {
	report := QualityCohortReport{Revision: strings.TrimSpace(revision), ClipCount: len(clips)}
	speakers := map[string]struct{}{}
	hashes := map[string]int{}
	if report.Revision == "" {
		report.Reasons = append(report.Reasons, "corpus revision is required")
	}
	for _, clip := range clips {
		report.ReferenceWords += len(strings.Fields(clip.ReferenceText))
		if clip.LicenseReference != "" {
			report.LicensedCount++
		}
		if clip.SpeakerPseudonym != "" {
			speakers[clip.SpeakerPseudonym] = struct{}{}
		}
		if clip.ContentSHA256 != "" {
			hashes[clip.ContentSHA256]++
		}
		if clip.CorpusRevision == "" || clip.CorpusRevision != report.Revision {
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has a missing or mismatched corpus revision", clip.ID))
		}
		if clip.Split != RequiredQualityCorpusSplit {
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q is not in the held-out split", clip.ID))
		}
		if clip.LicenseReference == "" {
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has no license reference", clip.ID))
		}
		if clip.SpeakerPseudonym == "" {
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has no speaker pseudonym", clip.ID))
		}
		if clip.ContentSHA256 == "" {
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has no content hash", clip.ID))
		}
		switch clip.NoiseCondition {
		case NoiseClean:
			report.CleanCount++
		case NoiseNoisy:
			switch {
			case math.Abs(clip.SNRDB-20) < 0.01:
				report.Noisy20dBCount++
			case math.Abs(clip.SNRDB-10) < 0.01:
				report.Noisy10dBCount++
			default:
				report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has noisy condition without 20 or 10 dB SNR", clip.ID))
			}
		default:
			report.Reasons = appendUnique(report.Reasons, fmt.Sprintf("clip %q has no clean/noisy condition", clip.ID))
		}
	}
	report.SpeakerCount = len(speakers)
	for _, count := range hashes {
		if count > 1 {
			report.DuplicateHashes += count - 1
		}
	}
	if report.ClipCount < RequiredQualityClips {
		report.Reasons = append(report.Reasons, fmt.Sprintf("requires at least %d clips, got %d", RequiredQualityClips, report.ClipCount))
	}
	if report.ReferenceWords < RequiredQualityRefWords {
		report.Reasons = append(report.Reasons, fmt.Sprintf("requires at least %d reference words, got %d", RequiredQualityRefWords, report.ReferenceWords))
	}
	if report.SpeakerCount < RequiredQualitySpeakers {
		report.Reasons = append(report.Reasons, fmt.Sprintf("requires at least %d speakers, got %d", RequiredQualitySpeakers, report.SpeakerCount))
	}
	if report.CleanCount < RequiredCleanClips || report.Noisy20dBCount < RequiredNoisy20dBClips || report.Noisy10dBCount < RequiredNoisy10dBClips {
		report.Reasons = append(report.Reasons, "requires clean, 20 dB noisy, and 10 dB noisy cohorts")
	}
	if report.DuplicateHashes > 0 {
		report.Reasons = append(report.Reasons, fmt.Sprintf("content hashes contain %d duplicate clip(s)", report.DuplicateHashes))
	}
	sort.Strings(report.Reasons)
	report.Eligible = len(report.Reasons) == 0
	return report
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
