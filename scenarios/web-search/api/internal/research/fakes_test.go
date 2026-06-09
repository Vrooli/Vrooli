package research_test

import (
	"context"

	"web-search/internal/research"
	"web-search/internal/research/agentmanager"
)

// fakeSearcher returns canned candidates and records the requested topN.
type fakeSearcher struct {
	candidates []research.Candidate
	err        error
	gotTopN    int
	gotQuery   string
}

func (f *fakeSearcher) Candidates(_ context.Context, query string, topN int) ([]research.Candidate, error) {
	f.gotQuery = query
	f.gotTopN = topN
	if f.err != nil {
		return nil, f.err
	}
	out := f.candidates
	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}
	return out, nil
}

// fakeFetcher returns canned text per URL; URLs not in the map return failErr
// (so per-page fetch-failure tolerance is exercisable).
type fakeFetcher struct {
	textByURL map[string]string
	failErr   error
	fetched   []string
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (string, error) {
	f.fetched = append(f.fetched, url)
	if txt, ok := f.textByURL[url]; ok {
		return txt, nil
	}
	if f.failErr != nil {
		return "", f.failErr
	}
	return "", nil
}

// fakeSynthesizer returns a canned synthesis and records the docs it saw.
type fakeSynthesizer struct {
	result   research.Synthesis
	err      error
	gotDocs  []research.Document
	gotQuery string
}

func (f *fakeSynthesizer) Synthesize(_ context.Context, query string, docs []research.Document) (research.Synthesis, error) {
	f.gotQuery = query
	f.gotDocs = docs
	if f.err != nil {
		return research.Synthesis{}, f.err
	}
	return f.result, nil
}

// fakeAgentManager returns canned spawn/poll results and records calls.
type fakeAgentManager struct {
	spawnResult agentmanager.RunResult
	spawnErr    error
	stateByID   map[string]agentmanager.RunState
	stateErr    error
	spawnedReq  agentmanager.SpawnRequest
}

func (f *fakeAgentManager) Spawn(_ context.Context, req agentmanager.SpawnRequest) (agentmanager.RunResult, error) {
	f.spawnedReq = req
	if f.spawnErr != nil {
		return agentmanager.RunResult{}, f.spawnErr
	}
	return f.spawnResult, nil
}

func (f *fakeAgentManager) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if f.stateErr != nil {
		return agentmanager.RunState{}, f.stateErr
	}
	return f.stateByID[runID], nil
}
