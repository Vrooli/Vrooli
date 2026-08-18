package catalogcoverage

import (
	"sort"
	"strings"

	"react-component-library/internal/assetrung"
)

// Bucket is which side of the join an entry landed on.
type Bucket string

const (
	// BucketPlannedBuilt is a catalog asset with an implementation.
	BucketPlannedBuilt Bucket = "planned-built"
	// BucketPlannedUnbuilt is a catalog asset with no implementation for a
	// declared target. The normal state for most of the catalog.
	BucketPlannedUnbuilt Bucket = "planned-unbuilt"
	// BucketSupplemental is an implementation with no catalog entry. Legitimate
	// — the library may exceed the target list — and worth counting separately
	// so it is a deliberate choice rather than an accident.
	BucketSupplemental Bucket = "supplemental"
)

// AchievedRung is the maturity actually supported by evidence. Missing is
// intentionally achieved-only: catalog targets may not ask for it.
type AchievedRung string

const (
	RungMissing         AchievedRung = "missing"
	RungScaffolded      AchievedRung = "scaffolded"
	RungImplemented     AchievedRung = "implemented"
	RungVerified        AchievedRung = "verified"
	RungProductionReady AchievedRung = "production-ready"
)

// GateEvidence is one observed result for an asset/target gate.
// Result is pass, fail, or not-run. Unknown and empty results are treated as
// not-run so a new runner cannot accidentally inflate coverage.
type GateEvidence struct {
	AssetID        string
	Target         string
	Gate           string
	Version        string
	Result         string
	SourceRevision string
	RecordedAt     string
}

// GateDefinition is the config projection needed by the coverage engine.
type GateDefinition struct {
	ID                         string
	Rung                       AchievedRung
	Blocking                   bool
	Attribution                string
	Runner                     map[string]string `json:"runner"`
	AppliesTo                  []string
	ExperienceClaimTypes       []string `json:"x-experience-claim-types"`
	ExperienceMinimumViewports int      `json:"x-experience-min-viewports"`
	ExperienceRequiresCapture  bool     `json:"x-experience-requires-capture"`
}

// Row is one joined entry.
type Row struct {
	AssetID     string
	Name        string
	Domain      string
	Rung        assetrung.Rung
	RungName    string
	DomainOrder int
	Kind        string
	Priority    string
	Bucket      Bucket
	Target      string
	Platform    string
	Achieved    AchievedRung
	// Implementation is the on-disk directory name when one exists.
	Implementation string
	// BlocksDownstream is how many other catalog assets transitively require
	// this one. It is the work-ordering signal: an unbuilt asset blocking forty
	// others is worth more than one blocking none.
	BlocksDownstream    int
	AssetScore          float64
	Weight              float64
	PassedGates         []string
	FailedGates         []string
	NearestBlockingGate string
	NewestEvidence      string
	VisualEvidence      bool
}

// Report is the joined view.
type Report struct {
	Rows []Row
	// Totals count rows per bucket.
	Totals map[Bucket]int
	// ByDomain and ByPriority count only planned assets, since supplemental
	// entries have no declared domain or priority to roll up under.
	ByDomain   map[string]DomainCount
	ByPriority map[string]DomainCount
	// MaturityCoverage is the number of declared asset/target rows at or above
	// their target, grouped both as a total and by the same catalog rollups.
	Maturity               MaturityCoverage
	Score                  float64
	ScoreWeightNumerator   float64
	ScoreWeightDenominator float64
	ByGate                 map[string]ScoreBreakdown
	ByRungScore            map[AchievedRung]ScoreBreakdown
	Corpus                 []CorpusStatus
}

type ScoreBreakdown struct {
	Passed     int
	Applicable int
	Score      float64
}

type CorpusStatus struct {
	Gate             string
	Result           string
	FindingCount     int
	RunnerErrorCount int
}

type MaturityCoverage struct {
	AtOrAboveTarget         int
	Total                   int
	ByDomain                map[string]DomainCount
	ByPriority              map[string]DomainCount
	ByRung                  map[AchievedRung]int
	CatalogCompletion       CoverageMetric
	MandatoryGateCoverage   CoverageMetric
	WeightedQuality         CoverageMetric
	ProductionReadyCoverage CoverageMetric
}

// CoverageMetric keeps a ratio auditable. A percentage without its
// numerator and denominator is not a trustworthy coverage report.
type CoverageMetric struct {
	Numerator   int
	Denominator int
	Ratio       float64
}

// DomainCount is a planned/built pair for one grouping.
type DomainCount struct {
	Planned int
	Built   int
}

// Compute joins catalog assets against implementations by catalogId. It is
// retained as the compatibility entrypoint for callers that do not yet have
// persisted gate evidence; a linked implementation is then scaffolded.
func Compute(assets []Asset, impls []Implementation) Report {
	return ComputeWithEvidence(assets, impls, nil, nil)
}

// ComputeWithEvidence derives maturity per declared asset target. Identity
// (catalogId) and quality (gate evidence) are deliberately independent.
func ComputeWithEvidence(assets []Asset, impls []Implementation, evidence []GateEvidence, gates []GateDefinition) Report {
	rep := Report{
		Totals:     map[Bucket]int{},
		ByDomain:   map[string]DomainCount{},
		ByPriority: map[string]DomainCount{},
		Maturity:   MaturityCoverage{ByDomain: map[string]DomainCount{}, ByPriority: map[string]DomainCount{}, ByRung: map[AchievedRung]int{}},
		ByGate:     map[string]ScoreBreakdown{}, ByRungScore: map[AchievedRung]ScoreBreakdown{},
	}
	for _, gate := range gates {
		if gate.Attribution != "corpus" {
			continue
		}
		status := CorpusStatus{Gate: gate.ID, Result: "not-run"}
		for _, item := range evidence {
			if item.AssetID == "__corpus__" && item.Gate == gate.ID && item.Result != "" {
				status.Result = item.Result
				break
			}
		}
		rep.Corpus = append(rep.Corpus, status)
	}
	byCatalogID := map[string]Implementation{}
	claimed := map[string]bool{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			byCatalogID[impl.CatalogID] = impl
		}
	}
	downstream := blockedCounts(assets)

	for _, asset := range assets {
		targets := asset.Targets
		if len(targets) == 0 {
			targets = []string{"react-vite"}
		}
		for _, target := range targets {
			row := Row{
				AssetID: asset.ID, Name: asset.Name, Domain: asset.Domain,
				Kind: asset.Kind, Rung: asset.Rung, RungName: asset.RungName, DomainOrder: asset.DomainOrder,
				Priority: asset.Priority, Target: asset.Maturity, Platform: target,
				Achieved: RungMissing, BlocksDownstream: downstream[asset.ID],
			}
			if impl, ok := byCatalogID[asset.ID]; ok {
				row.Bucket = BucketPlannedBuilt
				row.Implementation = impl.Name
				row.Achieved = achieved(asset, target, impl, gates, evidence)
				row.AssetScore, row.PassedGates, row.FailedGates, row.NearestBlockingGate, row.NewestEvidence, row.VisualEvidence = scoreAsset(asset, target, impl, gates, evidence)
				baseWeight := asset.PinnedWeight
				if baseWeight <= 0 {
					baseWeight = 1
				}
				row.Weight = baseWeight + float64(row.BlocksDownstream)
				claimed[impl.Name] = true
			} else {
				row.Bucket = BucketPlannedUnbuilt
			}
			rep.Rows = append(rep.Rows, row)
			rep.Totals[row.Bucket]++

			d := rep.ByDomain[asset.Domain]
			d.Planned++
			if row.Bucket == BucketPlannedBuilt {
				d.Built++
			}
			rep.ByDomain[asset.Domain] = d
			p := rep.ByPriority[asset.Priority]
			p.Planned++
			if row.Bucket == BucketPlannedBuilt {
				p.Built++
			}
			rep.ByPriority[asset.Priority] = p
			rep.Maturity.Total++
			rep.Maturity.ByRung[row.Achieved]++
			if atOrAbove(row.Achieved, AchievedRung(asset.Maturity)) {
				rep.Maturity.AtOrAboveTarget++
				md := rep.Maturity.ByDomain[asset.Domain]
				md.Built++
				rep.Maturity.ByDomain[asset.Domain] = md
				mp := rep.Maturity.ByPriority[asset.Priority]
				mp.Built++
				rep.Maturity.ByPriority[asset.Priority] = mp
			}
		}
	}
	rep.Maturity.CatalogCompletion = metric(rep.Totals[BucketPlannedBuilt], rep.Maturity.Total)
	for _, row := range rep.Rows {
		if row.Bucket == BucketSupplemental {
			continue
		}
		for _, gate := range gates {
			if !gate.Blocking || !isAttributable(gate) || !contains(gate.AppliesTo, row.Kind) {
				continue
			}
			rep.Maturity.MandatoryGateCoverage.Denominator++
			if hasEvidence(evidence, row.AssetID, row.Platform, gate.ID) {
				rep.Maturity.MandatoryGateCoverage.Numerator++
			}
		}
		if row.Bucket == BucketPlannedBuilt {
			rep.ScoreWeightNumerator += row.AssetScore * row.Weight
			rep.ScoreWeightDenominator += row.Weight
		}
		for _, gate := range gates {
			if !gate.Blocking || !isAttributable(gate) || !contains(gate.AppliesTo, row.Kind) || row.Bucket != BucketPlannedBuilt {
				continue
			}
			breakdown := rep.ByGate[gate.ID]
			breakdown.Applicable++
			if contains(row.PassedGates, gate.ID) {
				breakdown.Passed++
			}
			breakdown.Score = ratio(breakdown.Passed, breakdown.Applicable)
			rep.ByGate[gate.ID] = breakdown
		}
		if row.Bucket == BucketPlannedBuilt {
			breakdown := rep.ByRungScore[row.Achieved]
			breakdown.Applicable++
			if row.AssetScore >= 1 {
				breakdown.Passed++
			}
			breakdown.Score = ratio(breakdown.Passed, breakdown.Applicable)
			rep.ByRungScore[row.Achieved] = breakdown
		}
		weight := priorityWeight(row.Priority)
		rep.Maturity.WeightedQuality.Denominator += weight * rungRank(RungProductionReady)
		rep.Maturity.WeightedQuality.Numerator += weight * rungRank(row.Achieved)
		if row.Achieved == RungProductionReady {
			rep.Maturity.ProductionReadyCoverage.Numerator++
		}
		rep.Maturity.ProductionReadyCoverage.Denominator++
	}
	rep.Maturity.MandatoryGateCoverage.Ratio = ratio(rep.Maturity.MandatoryGateCoverage.Numerator, rep.Maturity.MandatoryGateCoverage.Denominator)
	rep.Maturity.WeightedQuality.Ratio = ratio(rep.Maturity.WeightedQuality.Numerator, rep.Maturity.WeightedQuality.Denominator)
	rep.Maturity.ProductionReadyCoverage.Ratio = ratio(rep.Maturity.ProductionReadyCoverage.Numerator, rep.Maturity.ProductionReadyCoverage.Denominator)
	if rep.ScoreWeightDenominator > 0 {
		rep.Score = rep.ScoreWeightNumerator / rep.ScoreWeightDenominator * 100
	}

	for _, impl := range impls {
		if impl.CatalogID != "" && claimed[impl.Name] {
			continue
		}
		if impl.CatalogID != "" {
			// Points at a catalog id that does not exist. A dangling reference
			// is a real defect rather than a supplemental asset, so it is
			// surfaced with its intended id still attached.
			rep.Rows = append(rep.Rows, Row{
				AssetID: impl.CatalogID, Name: impl.Name, Bucket: BucketSupplemental,
				Implementation: impl.Name, Domain: "(dangling catalogId)",
			})
			rep.Totals[BucketSupplemental]++
			continue
		}
		rep.Rows = append(rep.Rows, Row{
			Name: impl.Name, Bucket: BucketSupplemental,
			Implementation: impl.Name, Domain: "(" + impl.Root + ")",
		})
		rep.Totals[BucketSupplemental]++
	}
	return rep
}

var rungOrder = []AchievedRung{RungScaffolded, RungImplemented, RungVerified, RungProductionReady}

func achieved(asset Asset, target string, impl Implementation, gates []GateDefinition, evidence []GateEvidence) AchievedRung {
	if impl.ExperienceStateKnown && (!impl.ExperienceRegistered || impl.ExperienceVacuous) {
		return RungScaffolded
	}
	if len(gates) == 0 {
		return RungScaffolded
	}
	results := map[string]string{}
	for _, item := range evidence {
		if item.AssetID == asset.ID && item.Target == target && (item.Version == "" || item.Version == impl.Latest) {
			results[item.Gate] = strings.ToLower(item.Result)
		}
	}
	current := RungScaffolded
	// The catalog maturity is the declared bar, not a ceiling. A component can
	// legitimately exceed its planned target when higher-rung gates have real
	// passing evidence; coverage must preserve that distinction for next-work
	// ranking and production-ready reporting.
	for _, rung := range rungOrder {
		for _, gate := range gates {
			if !gate.Blocking || !isAttributable(gate) || rungRank(gate.Rung) > rungRank(rung) || !contains(gate.AppliesTo, asset.Kind) {
				continue
			}
			if results[gate.ID] != "pass" {
				return current
			}
		}
		current = rung
	}
	return current
}

func scoreAsset(asset Asset, target string, impl Implementation, gates []GateDefinition, evidence []GateEvidence) (float64, []string, []string, string, string, bool) {
	if impl.CatalogID == "" {
		return 0, nil, nil, "", "", false
	}
	results := map[string]GateEvidence{}
	for _, item := range evidence {
		if item.AssetID == asset.ID && item.Target == target && (item.Version == "" || item.Version == impl.Latest) {
			if existing, ok := results[item.Gate]; !ok || item.RecordedAt > existing.RecordedAt {
				results[item.Gate] = item
			}
		}
	}
	var passed, applicable int
	var passedGates, failedGates []string
	nearest := ""
	var newest string
	visual := false
	for _, item := range results {
		if item.RecordedAt > newest {
			newest = item.RecordedAt
		}
	}
	for _, gate := range gates {
		if !gate.Blocking || !isAttributable(gate) || !contains(gate.AppliesTo, asset.Kind) {
			continue
		}
		applicable++
		if strings.EqualFold(results[gate.ID].Result, "pass") {
			passed++
			passedGates = append(passedGates, gate.ID)
		} else {
			failedGates = append(failedGates, gate.ID)
			if nearest == "" || rungRank(gate.Rung) < rungRank(gateByID(gates, nearest).Rung) {
				nearest = gate.ID
			}
		}
		if gate.ID == "visual" && strings.EqualFold(results[gate.ID].Result, "pass") {
			visual = true
		}
	}
	sort.Strings(passedGates)
	sort.Strings(failedGates)
	return ratio(passed, applicable), passedGates, failedGates, nearest, newest, visual
}

func gateByID(gates []GateDefinition, id string) GateDefinition {
	for _, gate := range gates {
		if gate.ID == id {
			return gate
		}
	}
	return GateDefinition{}
}

func isAttributable(gate GateDefinition) bool {
	// Empty is accepted for in-memory compatibility fixtures. Authored config
	// is schema-required and always supplies one of the two explicit values.
	return gate.Attribution == "" || gate.Attribution == "attributable"
}

func hasEvidence(evidence []GateEvidence, assetID, target, gate string) bool {
	for _, item := range evidence {
		if item.AssetID == assetID && item.Target == target && item.Gate == gate && strings.EqualFold(item.Result, "pass") {
			return true
		}
	}
	return false
}

func priorityWeight(priority string) int {
	switch strings.ToUpper(priority) {
	case "P0":
		return 3
	case "P1":
		return 2
	default:
		return 1
	}
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func metric(numerator, denominator int) CoverageMetric {
	return CoverageMetric{Numerator: numerator, Denominator: denominator, Ratio: ratio(numerator, denominator)}
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func rungRank(r AchievedRung) int {
	for i, item := range []AchievedRung{RungMissing, RungScaffolded, RungImplemented, RungVerified, RungProductionReady} {
		if r == item {
			return i
		}
	}
	return 0
}

func atOrAbove(achieved, target AchievedRung) bool {
	return target != "" && rungRank(achieved) >= rungRank(target)
}

// NextWork returns every planned row below target, ordered by maturity gap,
// leverage, priority, then identity. A weakly implemented asset therefore
// beats a lower-leverage asset that has not started.
func NextWork(rep Report, limit int) []Row {
	out := nextWorkRows(rep, false)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// NextWorkLanes keeps promotion work separate from catalog construction.
// Promotion is the default lane because it turns an existing implementation
// into a useful, evidenced capability before expanding the backlog.
func NextWorkLanes(rep Report, limit int) (promote, build []Row) {
	promote = nextWorkRows(rep, false)
	build = nextWorkRows(rep, true)
	if limit > 0 {
		if len(promote) > limit {
			promote = promote[:limit]
		}
		if len(build) > limit {
			build = build[:limit]
		}
	}
	return promote, build
}

func nextWorkRows(rep Report, buildOnly bool) []Row {
	rank := map[string]int{"P0": 0, "P1": 1, "P2": 2}
	var out []Row
	for _, row := range rep.Rows {
		if row.Bucket == BucketSupplemental {
			continue
		}
		unbuilt := row.Bucket == BucketPlannedUnbuilt
		belowTarget := !atOrAbove(row.Achieved, AchievedRung(row.Target))
		if (buildOnly && unbuilt) || (!buildOnly && !unbuilt && belowTarget) {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		gi := rungRank(AchievedRung(out[i].Target)) - rungRank(out[i].Achieved)
		gj := rungRank(AchievedRung(out[j].Target)) - rungRank(out[j].Achieved)
		if gi != gj {
			return gi > gj
		}
		if out[i].BlocksDownstream != out[j].BlocksDownstream {
			return out[i].BlocksDownstream > out[j].BlocksDownstream
		}
		ri, rj := rank[out[i].Priority], rank[out[j].Priority]
		if ri != rj {
			return ri < rj
		}
		return out[i].AssetID < out[j].AssetID
	})
	return out
}

// blockedCounts computes, for each asset, how many distinct other assets
// transitively require it. Only `requires` edges count: a `suggests` edge is
// never copied and therefore never blocks anything.
func blockedCounts(assets []Asset) map[string]int {
	requires := make(map[string][]string, len(assets))
	for _, a := range assets {
		requires[a.ID] = a.Requires
	}
	counts := map[string]int{}
	for _, a := range assets {
		seen := map[string]bool{}
		var walk func(string)
		walk = func(id string) {
			for _, dep := range requires[id] {
				if seen[dep] {
					continue
				}
				seen[dep] = true
				walk(dep)
			}
		}
		walk(a.ID)
		for dep := range seen {
			counts[dep]++
		}
	}
	return counts
}
