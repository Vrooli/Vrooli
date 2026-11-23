package ai

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/blobstore"
	"image-tools/internal/aigateway"
	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/capacity"
	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

var routeEditOps = []string{"text_to_image", "image_to_image", "edit_instruct"}

// cloudFake stands in for the AI Gateway backend.
type cloudFake struct{ fakeProvider }

func (*cloudFake) IsCloud() bool    { return true }
func (*cloudFake) GPUCapable() bool { return false }

type fakeJobSource struct {
	finished []internaljobs.Job
	active   []internaljobs.Job
	refused  []internaljobs.Job
}

func (f *fakeJobSource) ListRouteRefusals(context.Context, int) ([]internaljobs.Job, error) {
	return f.refused, nil
}

func (f *fakeJobSource) ListCostRecords(context.Context, int) ([]internaljobs.Job, error) {
	return f.finished, nil
}
func (f *fakeJobSource) ListActive(context.Context) ([]internaljobs.Job, error) { return f.active, nil }

var (
	routeNow     = time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	gpuHost      = capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8, GPUs: []capabilities.GPU{{Index: 0, UUID: "GPU-test", VRAMBytes: 16 << 30}}}
	noGPUHost    = capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}
	costProvened = func(usd string) map[string]string {
		return map[string]string{aigateway.MetaCostUSD: usd, aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}
	}
)

type routeRig struct {
	engine *Engine
	local  *fakeProvider
	cloud  *cloudFake
	jobs   *fakeJobSource
}

func newRouteRig(t *testing.T, host capabilities.Host, mutate func(*Deps)) *routeRig {
	t.Helper()
	registry, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	rig := &routeRig{
		local: &fakeProvider{name: "diffusers", ops: routeEditOps},
		cloud: &cloudFake{fakeProvider{name: "openrouter", ops: append(slices.Clone(routeEditOps), "caption")}},
		jobs:  &fakeJobSource{},
	}
	backendReg := backends.New()
	for _, p := range []backends.Provider{rig.local, rig.cloud} {
		if err := backendReg.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	deps := Deps{
		Registry:        registry,
		Backends:        backendReg,
		Probe:           capabilities.FakeProbe{Host: host},
		Store:           storage.NewWithBlobStore(blobstore.NewMemoryBlobStore(), t.TempDir()),
		RightsEvaluator: allowRightsForTest,
		ModelInstalled:  func(string) bool { return true },
		ModelsRoot:      t.TempDir(),
		Disk:            &fakeDiskAdmission{},
		Route:           DefaultRouteConfig(),
		Jobs:            rig.jobs,
		Now:             func() time.Time { return routeNow },
	}
	if mutate != nil {
		mutate(&deps)
	}
	if rig.engine, err = NewEngine(deps); err != nil {
		t.Fatal(err)
	}
	return rig
}

func (r *routeRig) plan(t *testing.T, req PlanRequest) Plan {
	t.Helper()
	if req.Operation == "" {
		req.Operation = "edit_instruct"
	}
	plan, err := r.engine.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("plan %+v: %v", req, err)
	}
	return plan
}

const (
	hatPrompt   = "the same guinea pig wearing a small knitted hat"
	animePrompt = "redraw this guinea pig as a cel-shaded anime illustration"
)

func TestChooseCloudRoleSeparatesIdentityEditsFromStyledImages(t *testing.T) {
	cases := []struct {
		name                          string
		op, role, style, quality, say string
		want                          string
	}{
		{name: "an object added to the subject keeps identity", op: "edit_instruct", say: hatPrompt, want: RoleEditIdentity},
		{name: "a background change keeps identity", op: "edit_instruct", say: "put it on grass in a sunny park, remove the cage", want: RoleEditIdentity},
		{name: "anime restyle is stylised", op: "edit_instruct", say: animePrompt, want: RoleEditDefault},
		{name: "watercolor restyle is stylised", op: "image_to_image", say: "a watercolour painting of this scene", want: RoleEditDefault},
		{name: "explicit identity beats the restyle words", op: "edit_instruct", style: "identity", say: animePrompt, want: RoleEditIdentity},
		{name: "explicit stylized beats a plain edit", op: "edit_instruct", style: "stylized", say: hatPrompt, want: RoleEditDefault},
		{name: "no prompt falls back to identity", op: "image_to_image", want: RoleEditIdentity},
		{name: "a caller role always wins", op: "edit_instruct", role: "image.vector.default", say: hatPrompt, want: "image.vector.default"},
		{name: "generation defaults", op: "text_to_image", say: "a treehouse", want: RoleGenerateDefault},
		{name: "generation under the quality policy", op: "text_to_image", quality: "quality", say: "a treehouse", want: RoleGenerateQuality},
		{name: "a restyle word does not move generation off its roles", op: "text_to_image", say: "an anime treehouse", want: RoleGenerateDefault},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			role, reason := ChooseCloudRole(tc.op, tc.role, tc.style, tc.quality, tc.say)
			if role != tc.want || reason == "" {
				t.Fatalf("role = %q (%q), want %q with a stated reason", role, reason, tc.want)
			}
		})
	}
}

func TestRouteConfigFromEnv(t *testing.T) {
	cfg, err := RouteConfigFromEnv(func(string) string { return "" })
	if err != nil || !cfg.Active || !cfg.Enabled || cfg.WaitBudget != 10*time.Minute || cfg.MaxJobUSD != 0.25 || cfg.MaxDailyUSD != 3 {
		t.Fatalf("defaults must be the operator-approved ones: %+v err=%v", cfg, err)
	}
	env := map[string]string{
		EnvCloudFallback: "off", EnvCloudWaitBudget: "90s", EnvCloudMaxJobUSD: "0.5", EnvCloudMaxDailyUSD: "7.25",
		EnvCloudCostEstimates: "image.edit.identity=0.2, custom.role=0.01",
	}
	cfg, err = RouteConfigFromEnv(func(k string) string { return env[k] })
	if err != nil || cfg.Enabled || cfg.WaitBudget != 90*time.Second || cfg.MaxJobUSD != 0.5 || cfg.MaxDailyUSD != 7.25 ||
		cfg.CostEstimates[RoleEditIdentity] != 0.2 || cfg.CostEstimates["custom.role"] != 0.01 || cfg.CostEstimates[RoleEditDefault] == 0 {
		t.Fatalf("overrides not applied: %+v err=%v", cfg, err)
	}
	for name, value := range map[string]string{
		EnvCloudFallback: "maybe", EnvCloudWaitBudget: "soon", EnvCloudMaxJobUSD: "-1", EnvCloudMaxDailyUSD: "lots", EnvCloudCostEstimates: "image.edit.identity",
	} {
		if _, err := RouteConfigFromEnv(func(k string) string {
			if k == name {
				return value
			}
			return ""
		}); err == nil {
			t.Fatalf("%s=%q was accepted; a mistyped spending limit must not become a different limit", name, value)
		}
	}
}

func TestPlanOnAGPUlessHostPrefersTheCloudOverWeakCPUModels(t *testing.T) {
	for _, tc := range []struct {
		op, prompt string
		role       string
	}{
		{"edit_instruct", hatPrompt, RoleEditIdentity},
		{"edit_instruct", animePrompt, RoleEditDefault},
		{"image_to_image", "turn this into a loose watercolor painting", RoleEditDefault},
		{"text_to_image", "a cozy reading nook in a treehouse", RoleGenerateDefault},
	} {
		t.Run(tc.op+"/"+tc.role, func(t *testing.T) {
			rig := newRouteRig(t, noGPUHost, nil)
			plan := rig.plan(t, PlanRequest{Operation: tc.op, Prompt: tc.prompt})
			if plan.ModelID != "openrouter-image" || plan.Tier != backends.TierBYOK.String() || !plan.AllowBYOK {
				t.Fatalf("plan = %+v, want the cloud", plan)
			}
			if plan.Route == nil || plan.Route.Choice != "cloud" || plan.Route.Reason != routeReasonNoGPU || plan.Route.Role != tc.role {
				t.Fatalf("route = %+v", plan.Route)
			}
			if plan.Route.LocalModel == "" || plan.Route.LocalTier != backends.TierLocalCPU.String() {
				t.Fatalf("the original local choice must be recorded: %+v", plan.Route)
			}
			if plan.Params["openrouter_role"] != tc.role {
				t.Fatalf("the chosen role must reach the backend params: %v", plan.Params)
			}
			if !strings.Contains(strings.Join(plan.Warnings, "\n"), "no usable GPU") {
				t.Fatalf("the choice must be explained to the caller: %v", plan.Warnings)
			}
			if lane := LaneForTier(plan.Tier); lane != internaljobs.LaneNetwork {
				t.Fatalf("lane = %s", lane)
			}
		})
	}
}

func TestPlanOnAGPUlessHostStaysLocalWhenTheCallerOrTheSituationSaysSo(t *testing.T) {
	zero := 0.0
	cases := []struct {
		name string
		req  PlanRequest
		cfg  func(*Deps)
	}{
		{name: "local_only policy", req: PlanRequest{FallbackPolicy: "local_only"}},
		{name: "force cpu", req: PlanRequest{ForceCPU: true}},
		{name: "an explicitly named local model", req: PlanRequest{ModelOverride: "instruct-pix2pix"}},
		{name: "a zero spend cap", req: PlanRequest{MaxCostUSD: &zero}},
		{name: "a cap below the estimate", req: PlanRequest{MaxCostUSD: ptr(0.01)}},
		{name: "deployment switch off", cfg: func(d *Deps) { d.Route.Enabled = false }},
		{name: "policy inactive", cfg: func(d *Deps) { d.Route = RouteConfig{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rig := newRouteRig(t, noGPUHost, tc.cfg)
			tc.req.Prompt = hatPrompt
			plan := rig.plan(t, tc.req)
			if plan.Tier == backends.TierBYOK.String() || plan.ModelID == "openrouter-image" {
				t.Fatalf("plan went to the cloud: %+v", plan)
			}
		})
	}
}

// A hosted vision row asks the gateway role its own row names, and is priced as
// a text answer: a caption must not be planned, capped or counted against the
// daily budget as an image edit ($0.14).
func TestPlanPricesAHostedVisionRowByItsOwnRole(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	plan := rig.plan(t, PlanRequest{Operation: "caption", ModelOverride: "gemini-flash-lite-vision", AllowBYOK: true})
	if plan.Tier != backends.TierBYOK.String() || plan.Route == nil {
		t.Fatalf("the hosted vision row should run on the cloud tier: %+v", plan)
	}
	if plan.Route.Role != RoleVisionDefault || plan.Route.EstimatedCostUSD != DefaultRouteConfig().CostEstimates[RoleVisionDefault] {
		t.Errorf("route role %q estimate %v, want the vision role and its text price", plan.Route.Role, plan.Route.EstimatedCostUSD)
	}
}

func TestPlanLetsACallerAllowTheCloudPastADeploymentSwitchButNeverPastLocalOnly(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, func(d *Deps) { d.Route.Enabled = false })
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt, AllowBYOK: true})
	if plan.Tier != backends.TierBYOK.String() {
		t.Fatalf("an explicit allow_byok should permit the cloud: %+v", plan)
	}
	plan = rig.plan(t, PlanRequest{Prompt: hatPrompt, AllowBYOK: true, FallbackPolicy: "local_only"})
	if plan.Tier == backends.TierBYOK.String() {
		t.Fatalf("local_only must win over allow_byok: %+v", plan)
	}
}

func TestPlanOnAGPUHostIsLocalFirstAndArmsTheWaitBudget(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	if plan.ModelID == "openrouter-image" || plan.Tier != backends.TierLocalGPU.String() {
		t.Fatalf("a GPU host runs local first: %+v", plan)
	}
	if plan.AllowBYOK {
		t.Fatal("arming the fallback must not widen the pinned provider's own BYOK permission")
	}
	route := plan.Route
	if route == nil || route.Choice != "local" || !route.Fallback || route.WaitBudgetSeconds != 600 || route.MaxCostUSD != 0.25 || route.Role != RoleEditIdentity || route.EstimatedCostUSD != 0.14 {
		t.Fatalf("route = %+v", route)
	}
	if route.LocalModel != plan.ModelID || route.LocalTier != plan.Tier {
		t.Fatalf("the local choice must be recorded for a later re-route: %+v", route)
	}

	wait := 45
	plan = rig.plan(t, PlanRequest{Prompt: hatPrompt, MaxWaitSeconds: &wait, MaxCostUSD: ptr(0.2)})
	if plan.Route.WaitBudgetSeconds != 45 || plan.Route.MaxCostUSD != 0.2 {
		t.Fatalf("per-request limits not honoured: %+v", plan.Route)
	}

	forever := 0
	if plan = rig.plan(t, PlanRequest{Prompt: hatPrompt, MaxWaitSeconds: &forever}); plan.Route.Fallback {
		t.Fatalf("--max-wait 0 means wait locally without limit: %+v", plan.Route)
	}
	for name, req := range map[string]PlanRequest{
		"local_only":  {FallbackPolicy: "local_only"},
		"require gpu": {RequireGPU: true},
		"pinned":      {ModelOverride: "flux-2-klein-4b"},
		"zero cap":    {MaxCostUSD: ptr(0.0)},
	} {
		req.Prompt = hatPrompt
		if p := rig.plan(t, req); p.Route != nil && p.Route.Fallback {
			t.Fatalf("%s must not arm a cloud fallback: %+v", name, p.Route)
		}
	}
}

func TestSpendLedgerEnforcesTheRollingDailyCapFromRecordedCost(t *testing.T) {
	finished := func(id string, age time.Duration, cost string) internaljobs.Job {
		at := routeNow.Add(-age)
		return internaljobs.Job{ID: id, State: internaljobs.StateSucceeded, FinishedAt: &at, Meta: costProvened(cost)}
	}
	t.Run("recorded cost plus the estimate over the cap refuses the cloud", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		rig.jobs.finished = []internaljobs.Job{finished("a", time.Hour, "1.5"), finished("b", 23*time.Hour, "1.4")}
		plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
		if plan.Tier == backends.TierBYOK.String() {
			t.Fatalf("$2.90 recorded + $0.14 > $3: %+v", plan)
		}
		if !strings.Contains(strings.Join(plan.Warnings, "\n"), "rolling daily cap") {
			t.Fatalf("the refusal must name the cap: %v", plan.Warnings)
		}
	})
	t.Run("cost older than the window no longer counts", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		rig.jobs.finished = []internaljobs.Job{finished("old", 25*time.Hour, "2.9")}
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier != backends.TierBYOK.String() {
			t.Fatalf("a day-old spend must not block today: %+v", plan)
		}
	})
	t.Run("an explicit cloud request is held to the cap with an error", func(t *testing.T) {
		rig := newRouteRig(t, gpuHost, nil)
		rig.jobs.finished = []internaljobs.Job{finished("a", time.Hour, "2.95")}
		_, err := rig.engine.Plan(context.Background(), PlanRequest{Operation: "edit_instruct", Prompt: hatPrompt, ModelOverride: "openrouter-image", AllowBYOK: true})
		if !errors.Is(err, ErrSpendCap) {
			t.Fatalf("err = %v, want ErrSpendCap", err)
		}
	})
	t.Run("per-job cap", func(t *testing.T) {
		rig := newRouteRig(t, gpuHost, nil)
		_, err := rig.engine.Plan(context.Background(), PlanRequest{Operation: "edit_instruct", Prompt: hatPrompt, ModelOverride: "openrouter-image", AllowBYOK: true, MaxCostUSD: ptr(0.05)})
		if !errors.Is(err, ErrSpendCap) || !strings.Contains(err.Error(), "per-job cap") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("in-flight cloud jobs are committed, but only those created earlier", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		cloud, _ := json.Marshal(Payload{Tier: backends.TierBYOK.String(), Route: &RouteState{EstimatedCostUSD: 2.9}})
		rig.jobs.active = []internaljobs.Job{{ID: "in-flight", CreatedAt: routeNow.Add(-time.Minute), Payload: cloud}}
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier == backends.TierBYOK.String() {
			t.Fatalf("$2.90 already committed in flight: %+v", plan)
		}
		recorded, committed, err := rig.engine.cloudSpend(context.Background(), "me", routeNow.Add(-2*time.Minute))
		if err != nil || recorded != 0 || committed != 0 {
			t.Fatalf("a later job must not hold back an earlier one: %v %v %v", recorded, committed, err)
		}
	})
	t.Run("failed jobs count the attempt costs the gateway reported", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		evidence, _ := json.Marshal(aigateway.Evidence{SchemaVersion: 1, Variations: []aigateway.EvidenceVariation{{Index: 0, Outcome: "failed", CostState: "known", CostUSD: ptr(2.95)}}})
		at := routeNow.Add(-time.Hour)
		rig.jobs.finished = []internaljobs.Job{{ID: "failed", State: internaljobs.StateFailed, FinishedAt: &at, Meta: map[string]string{internaljobs.MetaGatewayExecutionEvidence: string(evidence)}}}
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier == backends.TierBYOK.String() {
			t.Fatalf("a billed failed attempt must count toward the cap: %+v", plan)
		}
	})
	t.Run("an unreadable ledger fails closed", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, func(d *Deps) { d.Jobs = nil })
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier == backends.TierBYOK.String() {
			t.Fatalf("a cap that cannot be verified is not a cap: %+v", plan)
		}
	})
}

func armedLocalJob(t *testing.T, rig *routeRig, req PlanRequest, createdAgo time.Duration) internaljobs.Job {
	t.Helper()
	plan := rig.plan(t, req)
	pl := Payload{Operation: "edit_instruct", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, GPU: plan.GPUViable, AutoAcquire: true,
		Params: map[string]string{"prompt": req.Prompt}, Route: plan.Route}
	raw, err := json.Marshal(pl)
	if err != nil {
		t.Fatal(err)
	}
	return internaljobs.Job{ID: "job-1", Operation: "edit_instruct", Lane: internaljobs.LaneGPU, State: internaljobs.StateQueued, CreatedAt: routeNow.Add(-createdAgo), Payload: raw}
}

func TestRerouteToCloudMovesAJobThatWaitedPastItsBudget(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	job := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 11*time.Minute)
	move, err := rig.engine.RerouteToCloud(context.Background(), job)
	if err != nil || move == nil {
		t.Fatalf("reroute = %+v err=%v", move, err)
	}
	if move.Lane != internaljobs.LaneNetwork || move.EstimatedSeconds != 30 || !strings.Contains(move.Message, "re-routed to the cloud") {
		t.Fatalf("move = %+v", move)
	}
	var pl Payload
	if err := json.Unmarshal(move.Payload, &pl); err != nil {
		t.Fatal(err)
	}
	if pl.ModelID != "openrouter-image" || pl.Backend != "openrouter" || pl.Tier != backends.TierBYOK.String() || !pl.AllowBYOK || pl.GPU || pl.AutoAcquire {
		t.Fatalf("payload not moved to the cloud: %+v", pl)
	}
	if pl.Params["openrouter_role"] != RoleEditIdentity || pl.Params["prompt"] != hatPrompt {
		t.Fatalf("params = %v", pl.Params)
	}
	r := pl.Route
	if r == nil || !r.Rerouted || r.Choice != "cloud" || r.Reason != routeReasonWait || r.Fallback || r.WaitedSeconds != 660 || r.LocalModel == "" || r.LocalTier != backends.TierLocalGPU.String() {
		t.Fatalf("route record = %+v", r)
	}
	meta := routeMeta(r, rig.engine.deps.Route)
	for key, want := range map[string]string{"route_choice": "cloud", "route_reason": routeReasonWait, "route_rerouted": "true", "route_waited_seconds": "660", "route_local_tier": "local-gpu", "route_role": RoleEditIdentity, "route_max_cost_usd": "0.25", "route_daily_cap_usd": "3"} {
		if meta[key] != want {
			t.Fatalf("meta[%s] = %q, want %q (%v)", key, meta[key], want, meta)
		}
	}
	if meta["route_local_model"] != r.LocalModel {
		t.Fatalf("the original local model must be in the meta: %v", meta)
	}

	// It never moves twice: the moved payload is not offered again.
	moved := job
	moved.Payload = move.Payload
	if again, err := rig.engine.RerouteToCloud(context.Background(), moved); again != nil || err != nil {
		t.Fatalf("a rerouted job was offered again: %+v %v", again, err)
	}
}

func TestRerouteToCloudLeavesJobsAloneUnlessEverythingAllowsIt(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	if move, err := rig.engine.RerouteToCloud(context.Background(), armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 9*time.Minute)); move != nil || err != nil {
		t.Fatalf("inside the budget the job stays local: %+v %v", move, err)
	}
	one := 1
	short := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt, MaxWaitSeconds: &one}, 2*time.Second)
	if move, err := rig.engine.RerouteToCloud(context.Background(), short); move == nil || err != nil {
		t.Fatalf("a caller's tiny budget must re-route: %+v %v", move, err)
	}
	zero := 0
	noLimit := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt, MaxWaitSeconds: &zero}, 5*time.Hour)
	if move, err := rig.engine.RerouteToCloud(context.Background(), noLimit); move != nil || err != nil {
		t.Fatalf("--max-wait 0 waits locally without limit: %+v %v", move, err)
	}
	local := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt, FallbackPolicy: "local_only"}, 5*time.Hour)
	if move, err := rig.engine.RerouteToCloud(context.Background(), local); move != nil || err != nil {
		t.Fatalf("local_only never re-routes: %+v %v", move, err)
	}

	// The daily cap is re-read at the moment of the move, not at submit.
	late := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 11*time.Minute)
	at := routeNow.Add(-time.Hour)
	rig.jobs.finished = []internaljobs.Job{{ID: "spent", State: internaljobs.StateSucceeded, FinishedAt: &at, Meta: costProvened("2.95")}}
	move, err := rig.engine.RerouteToCloud(context.Background(), late)
	if move != nil || !errors.Is(err, ErrSpendCap) {
		t.Fatalf("a spent day must decline the move: %+v %v", move, err)
	}
}

func TestAdmitRefusesACloudJobTheCapsNoLongerAllowBeforeItCanBeBilled(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, nil)
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	pl := Payload{Operation: "edit_instruct", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true, Params: plan.Params, Route: plan.Route}
	raw, _ := json.Marshal(pl)
	job := internaljobs.Job{ID: "cloud-1", Operation: "edit_instruct", CreatedAt: routeNow, Payload: raw}

	admission, err := rig.engine.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if err != nil {
		t.Fatalf("an affordable cloud job must be admitted: %v", err)
	}

	at := routeNow.Add(-time.Hour)
	rig.jobs.finished = []internaljobs.Job{{ID: "spent", State: internaljobs.StateSucceeded, FinishedAt: &at, Meta: costProvened("2.95")}}
	admission, err = rig.engine.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if !errors.Is(err, ErrSpendCap) || errors.Is(err, internaljobs.ErrCapacityBusy) {
		t.Fatalf("err = %v, want a terminal ErrSpendCap (not a retryable busy)", err)
	}
	if rig.cloud.execN != 0 {
		t.Fatal("the cloud provider was called")
	}
}

func runRouted(t *testing.T, rig *routeRig, pl Payload) (internaljobs.Result, error) {
	t.Helper()
	raw, err := json.Marshal(pl)
	if err != nil {
		t.Fatal(err)
	}
	return rig.engine.BuildRunners()["edit_instruct"](context.Background(), internaljobs.Job{ID: "run-1", Operation: "edit_instruct", CreatedAt: routeNow, Payload: raw}, func(int, string) {})
}

func putInput(t *testing.T, rig *routeRig) {
	t.Helper()
	storeInput(t, rig.engine.deps.Store.(*storage.Store), "input/pig.png", []byte("source"))
}

func TestRunRecordsTheRouteAndTheEffectivePromptInMeta(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, nil)
	putInput(t, rig)
	rig.cloud.meta = map[string]string{"backend": "openrouter", aigateway.MetaCostUSD: "0.138", aigateway.MetaCostProvenance: aigateway.ProviderReportedCostProvenance}
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	pl := Payload{Operation: "edit_instruct", InputKey: "input/pig.png", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true,
		Params: map[string]string{"prompt": hatPrompt, "openrouter_role": plan.Params["openrouter_role"]}, Route: plan.Route}
	result, err := runRouted(t, rig, pl)
	if err != nil {
		t.Fatal(err)
	}
	if rig.cloud.execN != 1 || rig.local.execN != 0 {
		t.Fatalf("cloud=%d local=%d, want exactly one cloud run", rig.cloud.execN, rig.local.execN)
	}
	if got := rig.cloud.lastReq.Params["prompt"]; !strings.HasPrefix(got, hatPrompt) || !strings.Contains(got, "Preserve the subject's identity") {
		t.Fatalf("the backend must receive the identity-preserving prompt: %q", got)
	}
	if got := rig.cloud.lastReq.Params[aigateway.ParamMaxCostUSD]; got != "0.25" {
		t.Fatalf("the job's cloud cap must reach the backend so the provider bounds its reservation: max_cost_usd=%q", got)
	}
	meta := result.Meta
	for key, want := range map[string]string{
		"route_choice": "cloud", "route_reason": routeReasonNoGPU, "route_role": RoleEditIdentity, "route_max_cost_usd": "0.25",
		"identity_preservation": "applied", "prompt_original": hatPrompt, aigateway.MetaCostUSD: "0.138",
	} {
		if meta[key] != want {
			t.Fatalf("meta[%s] = %q, want %q (%v)", key, meta[key], want, meta)
		}
	}
	if meta["effective_prompt"] != rig.cloud.lastReq.Params["prompt"] || meta["route_local_model"] == "" {
		t.Fatalf("effective prompt / original local choice missing: %v", meta)
	}
	if meta["route_cost_over_cap"] != "" {
		t.Fatalf("0.138 is under the cap: %v", meta)
	}
}

func TestRunFlagsACostAboveTheJobCap(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, nil)
	putInput(t, rig)
	rig.cloud.meta = costProvened("0.9")
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	pl := Payload{Operation: "edit_instruct", InputKey: "input/pig.png", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true, Params: map[string]string{"prompt": hatPrompt}, Route: plan.Route}
	result, err := runRouted(t, rig, pl)
	if err != nil || result.Meta["route_cost_over_cap"] != "true" {
		t.Fatalf("an overshoot of the cap must be visible: %v %v", result.Meta, err)
	}
}

func TestFailedJobStillReportsWhereItWasRoutedAndWhatItMayCost(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, nil)
	putInput(t, rig)
	rig.cloud.failWith = errors.New("gateway refused")
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	pl := Payload{Operation: "edit_instruct", InputKey: "input/pig.png", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true, Params: map[string]string{"prompt": hatPrompt}, Route: plan.Route}
	result, err := runRouted(t, rig, pl)
	if err == nil {
		t.Fatal("expected the provider failure")
	}
	if result.Meta["route_choice"] != "cloud" || result.Meta["route_max_cost_usd"] != "0.25" {
		t.Fatalf("failed job lost its route: %v", result.Meta)
	}
	if result.Ref != "" || result.Meta[aigateway.MetaCostUSD] != "" {
		t.Fatalf("failed job leaked output or cost: %+v", result)
	}
}

func TestIdentityPreservationIsOnByDefaultAndOptOutIsHonoured(t *testing.T) {
	const bare = "the same guinea pig sitting on the grass in a sunny park"
	t.Run("default adds the instruction once", func(t *testing.T) {
		got, meta := editPromptParams(map[string]string{"prompt": bare, "seed": "42"}, nil)
		if !strings.HasPrefix(got["prompt"], bare+". Preserve the subject's identity: keep its exact pose, face and distinguishing markings") || got["seed"] != "42" {
			t.Fatalf("params = %v", got)
		}
		if meta["identity_preservation"] != identityApplied || meta["prompt_original"] != bare || meta["effective_prompt"] != got["prompt"] {
			t.Fatalf("meta = %v", meta)
		}
	})
	t.Run("opt-out sends the prompt exactly as written", func(t *testing.T) {
		off := false
		got, meta := editPromptParams(map[string]string{"prompt": bare}, &off)
		if got["prompt"] != bare || meta["identity_preservation"] != identityOff || meta["effective_prompt"] != bare || meta["prompt_original"] != "" {
			t.Fatalf("params=%v meta=%v", got, meta)
		}
	})
	t.Run("a prompt that already says it is not stacked", func(t *testing.T) {
		for _, said := range []string{
			"place the same guinea pig, keeping its exact pose, face and fur markings, on the grass",
			"put a hat on it. Preserve the identity of the animal.",
			"swap the sky, keep the exact same face",
		} {
			got, meta := editPromptParams(map[string]string{"prompt": said}, nil)
			if got["prompt"] != said || meta["identity_preservation"] != identityAlreadyPresent {
				t.Fatalf("%q was rewritten: %v %v", said, got, meta)
			}
		}
	})
	t.Run("the caller's params are not mutated and an empty prompt is left alone", func(t *testing.T) {
		in := map[string]string{"prompt": bare}
		editPromptParams(in, nil)
		if in["prompt"] != bare {
			t.Fatal("input map was modified")
		}
		if got, meta := editPromptParams(map[string]string{"prompt": "  "}, nil); strings.TrimSpace(got["prompt"]) != "" || meta["identity_preservation"] != identityOff {
			t.Fatalf("empty prompt invented text: %v %v", got, meta)
		}
	})
}

func ptr[T any](v T) *T { return &v }

// A refused admission still owns whatever it acquired before the refusal, and
// releasing that must survive the same contended capacity ledger that a
// finished job's release does.
func TestARefusedAdmissionStillDeclaresItsReleaseRetryable(t *testing.T) {
	disk := &fakeDiskAdmission{}
	rig := newRouteRig(t, gpuHost, func(d *Deps) {
		d.Disk = disk
		d.CPU = func(context.Context, string, int64) (capacity.MemoryLease, error) {
			return nil, errors.New("cpu ledger unavailable")
		}
	})
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	raw, _ := json.Marshal(Payload{Operation: "edit_instruct", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, GPU: plan.GPUViable, Params: map[string]string{"prompt": hatPrompt}})
	admission, err := rig.engine.Admit(context.Background(), internaljobs.Job{ID: "refused", Operation: "edit_instruct", Payload: raw})
	if err == nil || !strings.Contains(err.Error(), "cpu ledger unavailable") {
		t.Fatalf("err = %v", err)
	}
	if !admission.ReleaseRetryable || admission.Release == nil {
		t.Fatalf("a partial admission must be retry-releasable: %+v", admission)
	}
	if err := admission.Release(context.Background()); err != nil || len(disk.leases) != 1 || disk.leases[0].released != 1 {
		t.Fatalf("the disk lease acquired before the refusal was not released: %v leases=%d", err, len(disk.leases))
	}
}

// The per-job cap bounds the whole job: each variation is a separate paid call,
// so the estimate scales with them, and admission must not count it twice.
func TestCloudEstimateScalesWithVariationsExactlyOnce(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	req := PlanRequest{Operation: "edit_instruct", Prompt: hatPrompt, ModelOverride: "openrouter-image", AllowBYOK: true, Variations: 4}
	if _, err := rig.engine.Plan(context.Background(), req); !errors.Is(err, ErrSpendCap) {
		t.Fatalf("4 x $0.14 exceeds the $0.25 cap: %v", err)
	}
	req.MaxCostUSD = ptr(0.6)
	plan, err := rig.engine.Plan(context.Background(), req)
	if err != nil || plan.Route == nil || plan.Route.EstimatedCostUSD < 0.559 || plan.Route.EstimatedCostUSD > 0.561 {
		t.Fatalf("plan = %+v err=%v", plan.Route, err)
	}
	raw, _ := json.Marshal(Payload{Operation: "edit_instruct", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true, Variations: 4, Params: plan.Params, Route: plan.Route})
	admission, err := rig.engine.Admit(context.Background(), internaljobs.Job{ID: "four", Operation: "edit_instruct", CreatedAt: routeNow, Payload: raw})
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	if err != nil {
		t.Fatalf("admission must accept what the plan accepted, without counting the variations twice: %v", err)
	}
}

// A job that could have used the cloud but stayed local says why in its meta,
// so "why did this not go to the cloud" never needs the service log.
func TestLocalJobsRecordWhyTheyStayedLocal(t *testing.T) {
	cases := []struct {
		name string
		req  PlanRequest
		want string
	}{
		{"zero spend cap", PlanRequest{MaxCostUSD: ptr(0.0)}, routeStayCap},
		{"local_only", PlanRequest{FallbackPolicy: "local_only"}, routeStayLocalReq},
		{"require gpu", PlanRequest{RequireGPU: true}, routeStayLocalReq},
		{"pinned model", PlanRequest{ModelOverride: "flux-2-klein-4b"}, routeStayPinned},
		{"no wait budget", PlanRequest{MaxWaitSeconds: ptr(0)}, routeStayNoWait},
	}
	rig := newRouteRig(t, gpuHost, nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.Prompt = hatPrompt
			plan := rig.plan(t, tc.req)
			if plan.Route == nil || plan.Route.Choice != "local" || plan.Route.Reason != tc.want || plan.Route.Fallback {
				t.Fatalf("route = %+v, want local/%s unarmed", plan.Route, tc.want)
			}
			meta := routeMeta(plan.Route, rig.engine.deps.Route)
			if meta["route_choice"] != "local" || meta["route_reason"] != tc.want || meta["route_fallback"] != "" {
				t.Fatalf("meta = %v", meta)
			}
		})
	}
}

const noCreditsError = `AI Gateway media execution "media-1" ended in MEDIA_EXECUTION_STATUS_FAILED: VROOLI_PROVIDER_ERROR {"code":"insufficient_credits","http_status":402,"message":"This request's maximum cost exceeds your available credits."}`

func refusedAt(ago time.Duration) internaljobs.Job {
	return internaljobs.Job{ID: "refused", State: internaljobs.StateQueued, Meta: map[string]string{MetaCloudRefusedAt: routeNow.Add(-ago).UTC().Format(time.RFC3339), MetaCloudRefusal: "insufficient_credits"}}
}

// reroutedPayload returns the cloud payload a wait-budget re-route produces, with
// the local route it left recorded inside it.
func reroutedPayload(t *testing.T, rig *routeRig) (internaljobs.Job, Payload) {
	t.Helper()
	job := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 11*time.Minute)
	job.EstimatedSeconds = 45
	move, err := rig.engine.RerouteToCloud(context.Background(), job)
	if err != nil || move == nil {
		t.Fatalf("reroute = %+v err=%v", move, err)
	}
	var pl Payload
	if err := json.Unmarshal(move.Payload, &pl); err != nil {
		t.Fatal(err)
	}
	job.Payload = move.Payload
	return job, pl
}

func TestCloudRefusalOnlyRecognisesRefusalsThatCannotHaveBilled(t *testing.T) {
	for text, want := range map[string]string{
		noCreditsError: "insufficient_credits",
		`openrouter: VROOLI_PROVIDER_ERROR {"http_status":401,"message":"bad key"}`:                     "http_401",
		`VROOLI_PROVIDER_ERROR {"code":"provider_unavailable","http_status":503}`:                       "provider_unavailable",
		`dial tcp 127.0.0.1:17999: connect: connection refused`:                                         "provider_unreachable",
		`VROOLI_PROVIDER_ERROR {"code":"rate_limited","http_status":429}`:                               "rate_limited",
		`VROOLI_PROVIDER_ERROR {"code":"x","http_status":402,"message":"unknown code, payment status"}`: "http_402",
	} {
		if code, ok := cloudRefusal(errors.New(text)); !ok || code != want {
			t.Fatalf("%q: got %q %v, want %q", text, code, ok, want)
		}
	}
	for _, text := range []string{
		"generation failed: the model returned no image",
		"context deadline exceeded",
		`VROOLI_PROVIDER_ERROR {"code":"content_policy","http_status":400}`,
		`gateway returned HTTP 500 after the image was generated`,
	} {
		if code, ok := cloudRefusal(errors.New(text)); ok {
			t.Fatalf("%q was treated as a pre-billing refusal (%s); it may have been billed", text, code)
		}
	}
	if _, ok := cloudRefusal(nil); ok {
		t.Fatal("nil error is not a refusal")
	}
}

func TestACloudRefusalReturnsAReroutedJobToItsLocalRoute(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	putInput(t, rig)
	job, pl := reroutedPayload(t, rig)
	pl.InputKey = "input/pig.png"
	rig.cloud.failWith = errors.New(noCreditsError)
	local := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 11*time.Minute)

	_, err := runRouted(t, rig, pl)
	var back *internaljobs.RouteBackError
	if !errors.As(err, &back) {
		t.Fatalf("a refused cloud attempt must return the job to its local route, got %v", err)
	}
	if rig.cloud.execN != 1 || rig.local.execN != 0 {
		t.Fatalf("cloud=%d local=%d", rig.cloud.execN, rig.local.execN)
	}
	var restored Payload
	if err := json.Unmarshal(back.Move.Payload, &restored); err != nil {
		t.Fatal(err)
	}
	var original Payload
	_ = json.Unmarshal(local.Payload, &original)
	if restored.ModelID != original.ModelID || restored.Backend != original.Backend || restored.Tier != backends.TierLocalGPU.String() || restored.AllowBYOK || restored.Params["prompt"] != hatPrompt {
		t.Fatalf("restored payload = %+v, want the original local route %+v", restored, original)
	}
	r := restored.Route
	if r == nil || r.Choice != "local" || r.Reason != routeReasonFellBack || r.Fallback || !r.Rerouted || r.WaitedSeconds != 660 || r.LocalPayload != nil || !strings.Contains(r.CloudRefusal, "insufficient_credits") {
		t.Fatalf("route = %+v", r)
	}
	if back.Move.Lane != internaljobs.LaneGPU || back.Move.EstimatedSeconds != job.EstimatedSeconds {
		t.Fatalf("move = lane %s eta %d", back.Move.Lane, back.Move.EstimatedSeconds)
	}
	for key, want := range map[string]string{"route_choice": "local", "route_reason": routeReasonFellBack, "route_rerouted": "true", "route_local_model": original.ModelID} {
		if back.Move.Meta[key] != want {
			t.Fatalf("meta[%s] = %q, want %q (%v)", key, back.Move.Meta[key], want, back.Move.Meta)
		}
	}
	if !strings.Contains(back.Move.Meta[MetaCloudRefusal], "insufficient_credits") || back.Move.Meta[MetaCloudRefusedAt] != routeNow.UTC().Format(time.RFC3339) {
		t.Fatalf("the provider error and time must be in the meta: %v", back.Move.Meta)
	}

	// It never goes back to the cloud: the restored job is not armed.
	rig.jobs.refused = nil
	again := local
	again.Payload = back.Move.Payload
	if move, err := rig.engine.RerouteToCloud(context.Background(), again); move != nil || err != nil {
		t.Fatalf("a fallen-back job was offered to the cloud again: %+v %v", move, err)
	}
}

func TestACloudRefusalFromACallerWhoDemandedTheCloudIsAnHonestError(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, nil)
	putInput(t, rig)
	rig.cloud.failWith = errors.New(noCreditsError)
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt, AllowBYOK: true})
	pl := Payload{Operation: "edit_instruct", InputKey: "input/pig.png", ModelID: plan.ModelID, Backend: plan.Provider, Tier: plan.Tier, AllowBYOK: true, Params: map[string]string{"prompt": hatPrompt}, Route: plan.Route}
	result, err := runRouted(t, rig, pl)
	var back *internaljobs.RouteBackError
	if err == nil || errors.As(err, &back) || !strings.Contains(err.Error(), "insufficient_credits") {
		t.Fatalf("an explicit cloud job has no local route to return to: %v", err)
	}
	if result.Meta[MetaCloudRefusedAt] == "" || !strings.Contains(result.Meta[MetaCloudRefusal], "insufficient_credits") || result.Meta["route_choice"] != "cloud" {
		t.Fatalf("the failed job must record the refusal so the cloud pauses: %v", result.Meta)
	}
}

func TestAnUnrecognisedCloudFailureStaysAFailure(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	putInput(t, rig)
	_, pl := reroutedPayload(t, rig)
	pl.InputKey = "input/pig.png"
	rig.cloud.failWith = errors.New("generation failed after the image was produced")
	_, err := runRouted(t, rig, pl)
	var back *internaljobs.RouteBackError
	if err == nil || errors.As(err, &back) {
		t.Fatalf("a failure that may have been billed must not be retried locally: %v", err)
	}
}

func TestRefusedCloudPausesAutomaticRoutingForTheCooldownOnly(t *testing.T) {
	t.Run("a recent refusal keeps a GPU-less host on its local model", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		rig.jobs.refused = []internaljobs.Job{refusedAt(5 * time.Minute)}
		plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
		if plan.Tier == backends.TierBYOK.String() || plan.Route == nil || plan.Route.Reason != routeStayCooldown {
			t.Fatalf("plan = %+v route=%+v", plan, plan.Route)
		}
		if !strings.Contains(strings.Join(plan.Warnings, "\n"), "paused") {
			t.Fatalf("the pause must be explained: %v", plan.Warnings)
		}
	})
	t.Run("it is not armed and does not re-route while paused", func(t *testing.T) {
		rig := newRouteRig(t, gpuHost, nil)
		job := armedLocalJob(t, rig, PlanRequest{Prompt: hatPrompt}, 11*time.Minute)
		rig.jobs.refused = []internaljobs.Job{refusedAt(time.Minute)}
		if move, err := rig.engine.RerouteToCloud(context.Background(), job); move != nil || !errors.Is(err, ErrCloudCooldown) {
			t.Fatalf("a paused cloud must not take the job: %+v %v", move, err)
		}
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Route.Fallback {
			t.Fatalf("a new job must not arm a fallback while paused: %+v", plan.Route)
		}
	})
	t.Run("a caller who demands the cloud is not held back", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		rig.jobs.refused = []internaljobs.Job{refusedAt(time.Minute)}
		for name, req := range map[string]PlanRequest{"byok": {AllowBYOK: true}, "named role": {Role: RoleEditDefault}} {
			req.Prompt = hatPrompt
			if plan := rig.plan(t, req); plan.Tier != backends.TierBYOK.String() {
				t.Fatalf("%s: plan = %+v", name, plan)
			}
		}
	})
	t.Run("the pause ends after the cooldown and can be disabled", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, nil)
		rig.jobs.refused = []internaljobs.Job{refusedAt(31 * time.Minute)}
		if plan := rig.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier != backends.TierBYOK.String() {
			t.Fatalf("a refusal older than the cooldown must not pause: %+v", plan)
		}
		off := newRouteRig(t, noGPUHost, func(d *Deps) { d.Route.Cooldown = 0 })
		off.jobs.refused = []internaljobs.Job{refusedAt(time.Second)}
		if plan := off.plan(t, PlanRequest{Prompt: hatPrompt}); plan.Tier != backends.TierBYOK.String() {
			t.Fatalf("cooldown 0 disables the pause: %+v", plan)
		}
	})
	t.Run("an unreadable history pauses rather than guessing", func(t *testing.T) {
		rig := newRouteRig(t, noGPUHost, func(d *Deps) { d.Jobs = nil })
		if err := rig.engine.checkCooldown(context.Background()); !errors.Is(err, ErrCloudCooldown) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a job already re-routed returns to local at admission instead of calling the paused cloud", func(t *testing.T) {
		rig := newRouteRig(t, gpuHost, nil)
		job, _ := reroutedPayload(t, rig)
		rig.jobs.refused = []internaljobs.Job{refusedAt(time.Minute)}
		admission, err := rig.engine.Admit(context.Background(), job)
		if admission.Release != nil {
			_ = admission.Release(context.Background())
		}
		var back *internaljobs.RouteBackError
		if !errors.As(err, &back) || rig.cloud.execN != 0 {
			t.Fatalf("err = %v, cloud calls = %d", err, rig.cloud.execN)
		}
		var restored Payload
		_ = json.Unmarshal(back.Move.Payload, &restored)
		if restored.Tier != backends.TierLocalGPU.String() || restored.Route.Reason != routeReasonFellBack {
			t.Fatalf("restored = %+v / %+v", restored, restored.Route)
		}
	})
}

func TestAReroutedJobReturnsLocalWhenTheDailyCapFillsBeforeTheCall(t *testing.T) {
	rig := newRouteRig(t, gpuHost, nil)
	job, _ := reroutedPayload(t, rig)
	at := routeNow.Add(-time.Hour)
	rig.jobs.finished = []internaljobs.Job{{ID: "spent", State: internaljobs.StateSucceeded, FinishedAt: &at, Meta: costProvened("2.95")}}
	admission, err := rig.engine.Admit(context.Background(), job)
	if admission.Release != nil {
		_ = admission.Release(context.Background())
	}
	var back *internaljobs.RouteBackError
	if !errors.As(err, &back) || !errors.Is(err, ErrSpendCap) {
		t.Fatalf("err = %v, want a return to the local route caused by the spend cap", err)
	}
}
