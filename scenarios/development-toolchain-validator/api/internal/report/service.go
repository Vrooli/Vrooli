package report

import (
	"context"
	"sort"
	"strings"

	manifest "development-toolchain-validator/internal/manifest"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	staleness "development-toolchain-validator/internal/staleness"
	vr "development-toolchain-validator/internal/validation_record"
)

// SkillCatalogSource exposes the read-only skill catalog the report
// service consults to enumerate skill subjects.
//
// seam: SkillCatalogSource
type SkillCatalogSource interface {
	List(ctx context.Context) ([]skillcatalog.Skill, error)
}

// ManifestSource exposes manifest reads for the coverage grid.
//
// seam: ManifestSource
type ManifestSource interface {
	List(ctx context.Context) ([]manifest.Manifest, error)
}

// RecordSource exposes append-only record reads.
//
// seam: RecordSource
type RecordSource interface {
	List(ctx context.Context, f vr.ListFilter, pageSize int, pageToken string) (vr.ListResult, error)
}

// StalenessSource exposes the derived staleness list.
//
// seam: StalenessSource
type StalenessSource interface {
	ListStale(ctx context.Context) ([]staleness.Entry, error)
}

// Service is the application-layer surface the report handler depends
// on.
type Service interface {
	GetGoldenSummary(ctx context.Context, goldenSlug string) (GoldenSummary, error)
	GetTupleHistory(ctx context.Context, kind vr.TupleKind, subjectID, goldenSlug string, pageSize int, pageToken string) (TupleHistory, error)
	GetCoverage(ctx context.Context, goldenSlug string) (Coverage, error)
	GetSkillFitness(ctx context.Context, skillID string) (SkillFitness, error)
}

type service struct {
	skills    SkillCatalogSource
	manifests ManifestSource
	records   RecordSource
	stale     StalenessSource
}

// NewService constructs the production Service.
func NewService(skills SkillCatalogSource, manifests ManifestSource, records RecordSource, stale StalenessSource) Service {
	return &service{skills: skills, manifests: manifests, records: records, stale: stale}
}

var _ Service = (*service)(nil)

func (s *service) GetGoldenSummary(ctx context.Context, goldenSlug string) (GoldenSummary, error) {
	goldenSlug = strings.TrimSpace(goldenSlug)
	if goldenSlug == "" {
		return GoldenSummary{}, ErrInvalidReport{Field: "golden_slug", Reason: "required"}
	}

	// Pull every record for this golden (paged through to completion).
	all, err := s.allRecords(ctx, vr.ListFilter{GoldenSlug: goldenSlug})
	if err != nil {
		return GoldenSummary{}, err
	}
	staleByTuple, err := s.staleSet(ctx)
	if err != nil {
		return GoldenSummary{}, err
	}

	// Latest per (kind, subject_id) for this golden.
	latest := latestPerTuple(all)
	summary := GoldenSummary{GoldenSlug: goldenSlug}
	for _, lv := range sortedTuples(latest) {
		entry := TupleVerdict{
			TupleKind:      lv.TupleKind,
			SubjectID:      lv.SubjectID,
			LatestVerdict:  lv.Verdict,
			LatestRecordID: lv.ID,
			Stale:          staleByTuple[[2]string{lv.SubjectID, goldenSlug}],
		}
		switch lv.TupleKind {
		case vr.TupleKindSkill:
			summary.SkillVerdicts = append(summary.SkillVerdicts, entry)
		case vr.TupleKindTool:
			summary.ToolVerdicts = append(summary.ToolVerdicts, entry)
		}
	}
	for k, stale := range staleByTuple {
		if stale && strings.EqualFold(k[1], goldenSlug) {
			summary.StaleCount++
		}
	}
	return summary, nil
}

func (s *service) GetTupleHistory(ctx context.Context, kind vr.TupleKind, subjectID, goldenSlug string, pageSize int, pageToken string) (TupleHistory, error) {
	subjectID = strings.TrimSpace(subjectID)
	goldenSlug = strings.TrimSpace(goldenSlug)
	if subjectID == "" || goldenSlug == "" {
		return TupleHistory{}, ErrInvalidReport{Field: "subject_id/golden_slug", Reason: "both required"}
	}
	res, err := s.records.List(ctx, vr.ListFilter{
		GoldenSlug: goldenSlug, SubjectID: subjectID, TupleKind: kind,
	}, pageSize, pageToken)
	if err != nil {
		return TupleHistory{}, err
	}
	return TupleHistory{
		TupleKind:     kind,
		SubjectID:     subjectID,
		GoldenSlug:    goldenSlug,
		Records:       res.Records,
		NextPageToken: res.NextPageToken,
	}, nil
}

func (s *service) GetCoverage(ctx context.Context, goldenSlug string) (Coverage, error) {
	goldenSlug = strings.TrimSpace(goldenSlug)
	if goldenSlug == "" {
		return Coverage{}, ErrInvalidReport{Field: "golden_slug", Reason: "required"}
	}
	skills, err := s.skills.List(ctx)
	if err != nil {
		return Coverage{}, err
	}
	mans, err := s.manifests.List(ctx)
	if err != nil {
		return Coverage{}, err
	}
	manifestsByTuple := map[[2]string]bool{}
	for _, m := range mans {
		if m.GoldenSlug == goldenSlug {
			manifestsByTuple[[2]string{m.SkillID, goldenSlug}] = true
		}
	}
	staleByTuple, err := s.staleSet(ctx)
	if err != nil {
		return Coverage{}, err
	}
	all, err := s.allRecords(ctx, vr.ListFilter{GoldenSlug: goldenSlug})
	if err != nil {
		return Coverage{}, err
	}
	latest := latestPerTuple(all)

	cov := Coverage{GoldenSlug: goldenSlug}
	for _, sk := range skills {
		key := [2]string{sk.ID, goldenSlug}
		row := CoverageRow{
			TupleKind:   vr.TupleKindSkill,
			SubjectID:   sk.ID,
			HasManifest: manifestsByTuple[key],
			Stale:       staleByTuple[key],
		}
		if lv, ok := latest[latestKey{Kind: vr.TupleKindSkill, Subject: sk.ID}]; ok {
			row.Verdict = lv.Verdict
		}
		cov.Rows = append(cov.Rows, row)
	}
	sort.Slice(cov.Rows, func(i, j int) bool { return cov.Rows[i].SubjectID < cov.Rows[j].SubjectID })
	return cov, nil
}

// GetSkillFitness folds every validation record for one skill, across all
// goldens, into a single trust/cost/convergence view. DTV owns the records, so
// this cross-golden aggregation lives here rather than being re-implemented by
// each consumer (swarm-manager's selection controller is the first).
func (s *service) GetSkillFitness(ctx context.Context, skillID string) (SkillFitness, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return SkillFitness{}, ErrInvalidReport{Field: "skill_id", Reason: "required"}
	}
	all, err := s.allRecords(ctx, vr.ListFilter{SubjectID: skillID, TupleKind: vr.TupleKindSkill})
	if err != nil {
		return SkillFitness{}, err
	}
	staleByTuple, err := s.staleSet(ctx)
	if err != nil {
		return SkillFitness{}, err
	}
	return aggregateSkillFitness(skillID, all, staleByTuple), nil
}

// aggregateSkillFitness is the pure fold over a skill's records (unit-testable
// without the service). staleByTuple is keyed [subject_id, golden_slug].
func aggregateSkillFitness(skillID string, records []vr.Record, staleByTuple map[[2]string]bool) SkillFitness {
	fit := SkillFitness{
		SkillID:  skillID,
		ByGolden: map[string]GoldenSkillSnapshot{},
	}
	if len(records) == 0 {
		fit.Verdict = SkillFitnessVerdictUnknown
		return fit
	}

	diffHashes := map[string]struct{}{}
	var latestOverall vr.Record
	latestPerGolden := map[string]vr.Record{}

	for _, r := range records {
		fit.TotalRuns++
		switch r.Verdict {
		case vr.VerdictPass:
			fit.PassCount++
		case vr.VerdictUnexpectedMutation:
			fit.UnexpectedMutationCount++
		case vr.VerdictRunFailure:
			fit.RunFailureCount++
		case vr.VerdictToolFailure:
			fit.ToolFailureCount++
		}
		fit.TotalTokens += r.TokensUsed
		fit.TotalCostUSDMicro += r.CostUSDMicro
		fit.TotalDurationMS += r.DurationMS
		diffHashes[r.DiffHash] = struct{}{}

		if latestOverall.ID == "" || r.EndedAt.After(latestOverall.EndedAt) {
			latestOverall = r
		}
		if prev, ok := latestPerGolden[r.GoldenSlug]; !ok || r.EndedAt.After(prev.EndedAt) {
			latestPerGolden[r.GoldenSlug] = r
		}
	}

	runsByGolden := map[string]int{}
	for _, r := range records {
		runsByGolden[r.GoldenSlug]++
	}

	runs := float64(fit.TotalRuns)
	fit.PassRate = float64(fit.PassCount) / runs
	fit.AvgTokens = float64(fit.TotalTokens) / runs
	fit.AvgCostUSDMicro = float64(fit.TotalCostUSDMicro) / runs
	fit.AvgDurationMS = float64(fit.TotalDurationMS) / runs
	fit.UniqueDiffHashes = len(diffHashes)
	if fit.UniqueDiffHashes > 0 {
		fit.ConvergenceRatio = 1.0 / float64(fit.UniqueDiffHashes)
	}
	fit.LatestVerdict = latestOverall.Verdict

	anyRunFailure, anyMutation, allPass := false, false, true
	for golden, r := range latestPerGolden {
		stale := staleByTuple[[2]string{skillID, golden}]
		fit.ByGolden[golden] = GoldenSkillSnapshot{
			GoldenSlug:    golden,
			LatestVerdict: r.Verdict,
			Stale:         stale,
			RunCount:      runsByGolden[golden],
		}
		if stale {
			fit.AnyStale = true
		}
		switch r.Verdict {
		case vr.VerdictRunFailure:
			anyRunFailure = true
			allPass = false
		case vr.VerdictUnexpectedMutation:
			anyMutation = true
			allPass = false
		case vr.VerdictPass:
			// keeps allPass true
		default:
			allPass = false
		}
	}

	switch {
	case anyRunFailure:
		fit.Verdict = SkillFitnessVerdictRed
	case anyMutation:
		fit.Verdict = SkillFitnessVerdictYellow
	case allPass:
		fit.Verdict = SkillFitnessVerdictGreen
	default:
		// Records exist but the latest set is neither cleanly passing nor an
		// explicit mutation/failure (e.g. unspecified) — runnable-but-incoherent.
		fit.Verdict = SkillFitnessVerdictYellow
	}
	return fit
}

// allRecords pages through records.List until next_page_token is empty.
func (s *service) allRecords(ctx context.Context, f vr.ListFilter) ([]vr.Record, error) {
	var out []vr.Record
	token := ""
	for {
		res, err := s.records.List(ctx, f, 100, token)
		if err != nil {
			return nil, err
		}
		out = append(out, res.Records...)
		if res.NextPageToken == "" {
			return out, nil
		}
		token = res.NextPageToken
		// Defensive cap: 1000 records is plenty for a P0 summary; if
		// production exceeds this the staleness/report compute path
		// should be denormalized rather than balloon a single response.
		if len(out) >= 1000 {
			return out, nil
		}
	}
}

func (s *service) staleSet(ctx context.Context) (map[[2]string]bool, error) {
	rows, err := s.stale.ListStale(ctx)
	if err != nil {
		return nil, err
	}
	out := map[[2]string]bool{}
	for _, r := range rows {
		out[[2]string{r.SkillID, r.GoldenSlug}] = true
	}
	return out, nil
}

// latestPerTuple keeps only the most-recent (by ended_at) record per
// (tuple_kind, subject_id) pair. The repository orders DESC by
// ended_at so the first occurrence wins.
type latestKey struct {
	Kind    vr.TupleKind
	Subject string
}

func latestPerTuple(records []vr.Record) map[latestKey]vr.Record {
	out := map[latestKey]vr.Record{}
	for _, r := range records {
		k := latestKey{Kind: r.TupleKind, Subject: r.SubjectID}
		if existing, ok := out[k]; !ok || r.EndedAt.After(existing.EndedAt) {
			out[k] = r
		}
	}
	return out
}

func sortedTuples(m map[latestKey]vr.Record) []vr.Record {
	out := make([]vr.Record, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TupleKind != out[j].TupleKind {
			return out[i].TupleKind < out[j].TupleKind
		}
		return out[i].SubjectID < out[j].SubjectID
	})
	return out
}
