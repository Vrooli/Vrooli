// Package taste exposes durable preference, elicitation, and recommendation
// operations over exact catalog tracks. It never accesses audio or models.
package taste

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	tastev1 "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste"
	tastev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste/taste_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"music-library/internal/catalog"
	"music-library/internal/decomposition"
	"music-library/internal/module"
	"music-library/internal/preference"
)

type catalogLister interface {
	List(context.Context, catalog.Filter) ([]catalog.Track, error)
}

type decompositionSourceLister interface {
	ListSources(context.Context, []string) (map[string]decomposition.SourceMetadata, error)
	ListTasteFeatures(context.Context, []string) (map[string][]decomposition.TasteFeature, error)
}

type handler struct {
	store          *preference.Store
	catalog        catalogLister
	decompositions decompositionSourceLister
}

type rankedCandidate struct {
	candidate preference.Candidate
	score     preference.Score
	selection float64
	bonus     float64
}

func Module(db *database.RoutedDB) module.Module {
	h := &handler{store: preference.NewStore(db), catalog: catalog.NewStore(db), decompositions: decomposition.NewStore(db)}
	path, server := tastev1connect.NewTasteServiceHandler(h)
	return module.Module{Name: "taste", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: server})
	}, Endpoints: Endpoints}
}

func Schema() string { return preference.Schema() }

func mapError(err error) error {
	if errors.Is(err, preference.ErrInvalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, preference.ErrJudgmentNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, catalog.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *handler) GetProfile(ctx context.Context, req *connect.Request[tastev1.GetProfileRequest]) (*connect.Response[tastev1.GetProfileResponse], error) {
	p, err := h.store.Get(ctx, req.Msg.Context)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&tastev1.GetProfileResponse{Profile: profileProto(p)}), nil
}

func (h *handler) SetControl(ctx context.Context, req *connect.Request[tastev1.SetControlRequest]) (*connect.Response[tastev1.SetControlResponse], error) {
	if req.Msg.Control == nil {
		return nil, mapError(preference.ErrInvalid)
	}
	c := req.Msg.Control
	p, err := h.store.SetControl(ctx, req.Msg.Context, preference.Control{Attribute: c.Attribute, Mode: preference.ControlMode(c.Mode), Value: c.Value})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&tastev1.SetControlResponse{Profile: profileProto(p)}), nil
}

func (h *handler) SelectPair(ctx context.Context, req *connect.Request[tastev1.SelectPairRequest]) (*connect.Response[tastev1.SelectPairResponse], error) {
	p, err := h.store.Get(ctx, req.Msg.Context)
	if err != nil {
		return nil, mapError(err)
	}
	candidates, err := h.candidates(ctx, false)
	if err != nil {
		return nil, mapError(err)
	}
	seen := make(map[string]bool, len(p.Judgments))
	for _, j := range p.Judgments {
		seen[canonicalPair(j.LeftID, j.RightID)] = true
	}
	left, right, info, ok := selectPair(p, candidates, seen, req.Msg.RandomBaseline, req.Msg.RandomSeed)
	if !ok {
		return connect.NewResponse(&tastev1.SelectPairResponse{AvailableCandidateCount: int64(len(candidates)), InsufficientEvidence: true}), nil
	}
	return connect.NewResponse(&tastev1.SelectPairResponse{Left: candidateProto(left), Right: candidateProto(right), ExpectedInformation: info, AvailableCandidateCount: int64(len(candidates)), InsufficientEvidence: informativeJudgmentCount(p.Judgments) == 0}), nil
}

func (h *handler) RecordJudgment(ctx context.Context, req *connect.Request[tastev1.RecordJudgmentRequest]) (*connect.Response[tastev1.RecordJudgmentResponse], error) {
	all, err := h.candidates(ctx, true)
	if err != nil {
		return nil, mapError(err)
	}
	byID := make(map[string]preference.Candidate, len(all))
	for _, c := range all {
		byID[c.ID] = c
	}
	left, lok := byID[req.Msg.LeftTrackSha256]
	right, rok := byID[req.Msg.RightTrackSha256]
	if !lok || !rok || left.ID == right.ID {
		return nil, connect.NewError(connect.CodeInvalidArgument, preference.ErrInvalid)
	}
	j := preference.Judgment{
		LeftID: left.ID, RightID: right.ID, Choice: preference.Choice(req.Msg.Choice),
		Reason: req.Msg.Reason, AttributeScope: req.Msg.AttributeScope, Reverses: req.Msg.ReversesJudgmentId,
		LeftEvidence: left.Features, RightEvidence: right.Features,
	}
	p, err := h.store.RecordJudgment(ctx, req.Msg.Context, j)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&tastev1.RecordJudgmentResponse{Profile: profileProto(p)}), nil
}

func (h *handler) Recommend(ctx context.Context, req *connect.Request[tastev1.RecommendRequest]) (*connect.Response[tastev1.RecommendResponse], error) {
	if !hashRE.MatchString(req.Msg.SeedTrackSha256) {
		return nil, connect.NewError(connect.CodeInvalidArgument, preference.ErrInvalid)
	}
	p, err := h.store.Get(ctx, req.Msg.Context)
	if err != nil {
		return nil, mapError(err)
	}
	all, err := h.candidates(ctx, true)
	if err != nil {
		return nil, mapError(err)
	}
	var seed *preference.Candidate
	for i := range all {
		if all[i].ID == req.Msg.SeedTrackSha256 {
			seed = &all[i]
			break
		}
	}
	if seed == nil {
		return nil, connect.NewError(connect.CodeNotFound, catalog.ErrNotFound)
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 10 {
		limit = 10
	}
	items := make([]rankedCandidate, 0, len(all))
	for _, c := range all {
		if c.ID == seed.ID {
			continue
		}
		s := preference.ScoreCandidate(p, c)
		if !s.Eligible {
			continue
		}
		bonus := 0.0
		if s.Explore {
			bonus = 0.15 * s.Uncertainty
		}
		items = append(items, rankedCandidate{candidate: c, score: s, selection: s.Mean + bonus, bonus: bonus})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].selection != items[j].selection {
			return items[i].selection > items[j].selection
		}
		return items[i].candidate.ID < items[j].candidate.ID
	})
	items = diversifyCandidates(items, seed.Artist, limit)
	response := &tastev1.RecommendResponse{Seed: candidateProto(*seed)}
	chosen, avoided := map[string]bool{}, map[string]bool{}
	for _, item := range items {
		rec := &tastev1.TasteRecommendation{
			Candidate: candidateProto(item.candidate), Rank: int32(len(response.Recommendations) + 1),
			PredictedPreference: item.score.Mean, Uncertainty: item.score.Uncertainty, Exploration: item.score.Explore,
			SelectionScore: item.selection, ExplorationBonus: item.bonus,
		}
		for _, part := range item.score.Contributions {
			rec.Contributions = append(rec.Contributions, &tastev1.TasteContribution{Attribute: part.Attribute, FeatureValue: part.Value, FeatureConfidence: confidenceFor(item.candidate, part.Attribute), LearnedWeight: part.Weight, Contribution: part.Contribution, EvidenceSource: part.EvidenceSource})
			if part.Contribution > 0.05 {
				chosen[part.Attribute] = true
			}
			if part.Contribution < -0.05 {
				avoided[part.Attribute] = true
			}
		}
		response.Recommendations = append(response.Recommendations, rec)
	}
	response.ChosenFeatures = sortedKeys(chosen)
	response.AvoidedFeatures = sortedKeys(avoided)
	for _, c := range response.Recommendations {
		for _, f := range c.Candidate.Features {
			response.Provenance = append(response.Provenance, f.EvidenceSource)
		}
		if c.Candidate.Artist != "" {
			response.Provenance = append(response.Provenance, "decomposition-source:"+c.Candidate.TrackSha256)
		}
	}
	response.Provenance = uniqueSorted(response.Provenance)
	if informativeJudgmentCount(p.Judgments) == 0 {
		response.EvidenceState = "sparse_prior_exploration"
	} else {
		response.EvidenceState = "operator_pairwise_evidence"
	}
	response.Mq2Brief = buildMQ2Brief(*seed, response.Recommendations, response.ChosenFeatures, response.AvoidedFeatures, response.Provenance, p)
	return connect.NewResponse(response), nil
}

func (h *handler) candidates(ctx context.Context, includeRejected bool) ([]preference.Candidate, error) {
	tracks, err := h.catalog.List(ctx, catalog.Filter{})
	if err != nil {
		return nil, err
	}
	out := make([]preference.Candidate, 0, len(tracks))
	var sources map[string]decomposition.SourceMetadata
	var acousticFeatures map[string][]decomposition.TasteFeature
	if h.decompositions != nil {
		hashes := make([]string, 0, len(tracks))
		for _, track := range tracks {
			hashes = append(hashes, track.SHA256)
		}
		var sourceErr error
		sources, sourceErr = h.decompositions.ListSources(ctx, hashes)
		if sourceErr != nil {
			return nil, sourceErr
		}
		acousticFeatures, sourceErr = h.decompositions.ListTasteFeatures(ctx, hashes)
		if sourceErr != nil {
			return nil, sourceErr
		}
	}
	for _, track := range tracks {
		if !includeRejected && track.Approval == "rejected" {
			continue
		}
		features := make([]preference.Feature, 0, len(track.Tags))
		genre := ""
		for _, tag := range track.Tags {
			name := strings.ToLower(strings.TrimSpace(tag))
			if name == "" {
				continue
			}
			attribute := "tag:" + name
			if strings.HasPrefix(name, "genre:") {
				genre = strings.TrimSpace(strings.TrimPrefix(name, "genre:"))
				if genre != "" {
					attribute = "genre:" + genre
				}
			}
			features = append(features, preference.Feature{Attribute: attribute, Value: 1, Confidence: 1, Source: "catalog:" + track.SHA256})
		}
		for _, feature := range acousticFeatures[track.SHA256] {
			features = append(features, preference.Feature{Attribute: feature.Attribute, Value: feature.Value, Confidence: feature.Confidence, Source: feature.Source})
		}
		sort.Slice(features, func(i, j int) bool { return features[i].Attribute < features[j].Attribute })
		candidate := preference.Candidate{ID: track.SHA256, Title: track.Title, Genre: genre, Features: features}
		if source, ok := sources[track.SHA256]; ok {
			candidate.Artist = strings.TrimSpace(source.Artist)
		}
		out = append(out, candidate)
	}
	return out, nil
}

func candidateProto(c preference.Candidate) *tastev1.TasteCandidate {
	out := &tastev1.TasteCandidate{TrackSha256: c.ID, Title: c.Title, Artist: c.Artist, Genre: c.Genre}
	for _, f := range c.Features {
		out.Features = append(out.Features, &tastev1.TasteFeature{Attribute: f.Attribute, Value: f.Value, Confidence: f.Confidence, EvidenceSource: f.Source})
		if strings.HasPrefix(f.Attribute, "tag:") {
			out.Tags = append(out.Tags, strings.TrimPrefix(f.Attribute, "tag:"))
		} else if strings.HasPrefix(f.Attribute, "genre:") {
			out.Tags = append(out.Tags, "genre:"+strings.TrimPrefix(f.Attribute, "genre:"))
		}
	}
	return out
}

func profileProto(p preference.Profile) *tastev1.TasteProfile {
	out := &tastev1.TasteProfile{Context: p.Context, ModelVersion: p.Version, FeatureSpace: p.FeatureSpace, Revision: p.Revision}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(p.UpdatedAt)
	}
	for _, c := range p.Controls {
		out.Controls = append(out.Controls, &tastev1.TasteControl{Attribute: c.Attribute, Mode: string(c.Mode), Value: c.Value})
	}
	attributes := make([]string, 0, len(p.Weights))
	for attribute := range p.Weights {
		attributes = append(attributes, attribute)
	}
	sort.Strings(attributes)
	counts := activeEvidenceCounts(p.Judgments)
	for _, attribute := range attributes {
		w := p.Weights[attribute]
		out.Weights = append(out.Weights, &tastev1.TasteWeight{Attribute: attribute, Mean: w.Mean, Variance: w.Variance, EvidenceCount: counts[attribute]})
	}
	for _, j := range p.Judgments {
		out.Judgments = append(out.Judgments, judgmentProto(j))
	}
	return out
}

func judgmentProto(j preference.Judgment) *tastev1.PairwiseJudgment {
	out := &tastev1.PairwiseJudgment{Id: j.ID, Context: j.Context, LeftTrackSha256: j.LeftID, RightTrackSha256: j.RightID, Choice: string(j.Choice), Reason: j.Reason, AttributeScope: j.AttributeScope, Source: j.Source, ReversesJudgmentId: j.Reverses}
	if parsed, err := time.Parse(time.RFC3339Nano, j.At); err == nil {
		out.JudgedAt = timestamppb.New(parsed)
	}
	for _, f := range j.LeftEvidence {
		out.LeftEvidence = append(out.LeftEvidence, featureProto(f))
	}
	for _, f := range j.RightEvidence {
		out.RightEvidence = append(out.RightEvidence, featureProto(f))
	}
	return out
}

func featureProto(f preference.Feature) *tastev1.TasteFeature {
	return &tastev1.TasteFeature{Attribute: f.Attribute, Value: f.Value, Confidence: f.Confidence, EvidenceSource: f.Source}
}

func activeEvidenceCounts(judgments []preference.Judgment) map[string]int64 {
	reversed := map[string]bool{}
	for _, j := range judgments {
		if j.Reverses != "" {
			reversed[j.Reverses] = true
		}
	}
	out := map[string]int64{}
	for _, j := range judgments {
		if !informativeChoice(j.Choice) || (j.ID != "" && reversed[j.ID]) {
			continue
		}
		seen := map[string]bool{}
		for _, f := range append(append([]preference.Feature(nil), j.LeftEvidence...), j.RightEvidence...) {
			if !seen[f.Attribute] {
				out[f.Attribute]++
				seen[f.Attribute] = true
			}
		}
	}
	return out
}

func informativeJudgmentCount(judgments []preference.Judgment) int {
	reversed := map[string]bool{}
	for _, j := range judgments {
		if j.Reverses != "" {
			reversed[j.Reverses] = true
		}
	}
	count := 0
	for _, j := range judgments {
		if informativeChoice(j.Choice) && (j.ID == "" || !reversed[j.ID]) {
			count++
		}
	}
	return count
}

func informativeChoice(choice preference.Choice) bool {
	return choice == preference.ChooseLeft || choice == preference.ChooseRight || choice == preference.ChooseTie
}

func confidenceFor(c preference.Candidate, attribute string) float64 {
	for _, f := range c.Features {
		if f.Attribute == attribute {
			return f.Confidence
		}
	}
	return 0
}

func buildMQ2Brief(seed preference.Candidate, recommendations []*tastev1.TasteRecommendation, chosen, avoided, provenance []string, p preference.Profile) string {
	evidence := "sparse prior; no informative pairwise judgments have been recorded"
	if count := informativeJudgmentCount(p.Judgments); count > 0 {
		evidence = fmt.Sprintf("%d informative operator pairwise judgments exist; no held-out accuracy claim is available", count)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Preference context: %s. Source track SHA-256: %s. Use only evidence-backed features: %s. Avoid only evidence-backed features: %s. Preference evidence: %s. Scores use hash-bound catalog tags (binary presence), explicit genre tags (typed category), and only operator-confirmed acoustic descriptors. Duration value is clamp(log2(seconds/180)/log2(600/180), -1, 1); tempo is clamp(log2(BPM/120)/log2(300/120), -1, 1). `key:<tonic>_<mode>` is one-hot. Source artist metadata is used for diversity, and unreviewed acoustic dimensions are omitted. Diversity excludes the seed artist and caps known artist/genre groups at two; unknown metadata is not inferred.\n", p.Context, seed.ID, strings.Join(chosen, ", "), strings.Join(avoided, ", "), evidence)
	for _, rec := range recommendations {
		if rec == nil || rec.Candidate == nil {
			continue
		}
		fmt.Fprintf(&b, "%d. track_sha256=%s expected_preference=%.3f uncertainty=%.3f\n", rec.Rank, rec.Candidate.TrackSha256, rec.PredictedPreference, rec.Uncertainty)
		for _, contribution := range rec.Contributions {
			fmt.Fprintf(&b, "   score_feature=%s value=%.3f confidence=%.3f contribution=%.3f evidence=%s\n", contribution.Attribute, contribution.FeatureValue, contribution.FeatureConfidence, contribution.Contribution, contribution.EvidenceSource)
		}
	}
	fmt.Fprintf(&b, "Candidate/source provenance: %s. Explore original material; do not imitate source audio or treat this brief as operator approval.", strings.Join(provenance, ", "))
	return b.String()
}

// diversifyCandidates keeps taste score order, excludes the seed artist, and
// caps known artist and explicit genre groups. Unknown metadata remains
// ungrouped instead of being inferred from titles or generic tags.
func diversifyCandidates(items []rankedCandidate, seedArtist string, limit int) []rankedCandidate {
	if limit < 1 || len(items) == 0 {
		return nil
	}
	seedArtist = strings.ToLower(strings.TrimSpace(seedArtist))
	artistCounts := make(map[string]int)
	genreCounts := make(map[string]int)
	chosen := make([]rankedCandidate, 0, min(limit, len(items)))
	for _, item := range items {
		artist := strings.ToLower(strings.TrimSpace(item.candidate.Artist))
		if artist != "" && artist == seedArtist {
			continue
		}
		genre := strings.ToLower(strings.TrimSpace(item.candidate.Genre))
		if artist != "" && artistCounts[artist] >= 2 {
			continue
		}
		if genre != "" && genreCounts[genre] >= 2 {
			continue
		}
		chosen = append(chosen, item)
		if artist != "" {
			artistCounts[artist]++
		}
		if genre != "" {
			genreCounts[genre]++
		}
		if len(chosen) == limit {
			break
		}
	}
	return chosen
}

func selectPair(p preference.Profile, candidates []preference.Candidate, seen map[string]bool, randomBaseline bool, seed int64) (preference.Candidate, preference.Candidate, float64, bool) {
	if randomBaseline {
		pairs := make([][2]preference.Candidate, 0)
		for i := 0; i < len(candidates); i++ {
			for j := i + 1; j < len(candidates); j++ {
				if candidates[i].ID != candidates[j].ID && preference.AreComparable(candidates[i], candidates[j]) && !seen[canonicalPair(candidates[i].ID, candidates[j].ID)] {
					pairs = append(pairs, [2]preference.Candidate{candidates[i], candidates[j]})
				}
			}
		}
		if len(pairs) == 0 {
			return preference.Candidate{}, preference.Candidate{}, 0, false
		}
		pair := pairs[rand.New(rand.NewSource(seed)).Intn(len(pairs))]
		return pair[0], pair[1], information(p, pair[0], pair[1]), true
	}
	a, b, ok := preference.SelectPair(p, candidates, seen)
	if !ok {
		return a, b, 0, false
	}
	return a, b, information(p, a, b), true
}

func information(p preference.Profile, a, b preference.Candidate) float64 {
	return preference.PairInformation(p, a, b)
}

func canonicalPair(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func uniqueSorted(in []string) []string {
	m := map[string]bool{}
	out := []string{}
	for _, x := range in {
		if !m[x] {
			m[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

var hashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

var Endpoints = []module.EndpointDescriptor{
	{ID: "taste_profile_get", Path: tastev1connect.TasteServiceGetProfileProcedure, Method: "POST", Summary: "Read the editable taste profile", Description: "Return versioned operator controls, learned estimates, uncertainty and provenance-bound pairwise judgments for one context.", Category: "taste"},
	{ID: "taste_control_set", Path: tastev1connect.TasteServiceSetControlProcedure, Method: "POST", Summary: "Set an explicit taste control", Description: "Append a profile revision with an operator-authored boost, suppression, pin or ban separate from learned weights.", Category: "taste"},
	{ID: "taste_pair_select", Path: tastev1connect.TasteServiceSelectPairProcedure, Method: "POST", Summary: "Select an informative pair", Description: "Offer a fresh catalog comparison using uncertainty-aware selection or a measurable random baseline.", Category: "taste"},
	{ID: "taste_judgment_record", Path: tastev1connect.TasteServiceRecordJudgmentProcedure, Method: "POST", Summary: "Record a provenance-bound comparison", Description: "Append an operator choice against two exact catalog track hashes with their source-tag evidence snapshots.", Category: "taste"},
	{ID: "taste_recommend", Path: tastev1connect.TasteServiceRecommendProcedure, Method: "POST", Summary: "Find recommendations from a liked track", Description: "Rank candidates from hash-bound catalog tags, explicit genre tags, and only exact-job acoustic descriptors confirmed by the operator; use matched decomposition artist metadata for diversity, and emit score-derived explanations, uncertainty, and an MQ2 planning brief.", Category: "taste"},
}
