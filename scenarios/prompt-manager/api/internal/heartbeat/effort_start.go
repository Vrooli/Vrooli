package heartbeat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/effortauthority"
)

// StartEffort is the separately authenticated external client route. A signed
// task label is not trusted task attestation. The native owner still qualifies
// each dispatch; this operation cannot configure/enable a team or mint grants.
func (f *FiniteLeaderRuntime) StartEffort(ctx context.Context, team, member, encoded string) (any, error) {
	if f.Efforts == nil || f.Efforts.Authority == nil || len(encoded) > 24000 {
		return nil, effortauthority.ErrRefused
	}
	raw, e := base64.RawURLEncoding.DecodeString(encoded)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var proof effortauthority.Proof
	if json.Unmarshal(raw, &proof) != nil {
		return nil, effortauthority.ErrRefused
	}
	configured, ok := f.Efforts.Bindings[team+"/"+member]
	if !ok {
		return nil, effortauthority.ErrRefused
	}
	b, e := f.Efforts.Authority.CheckProof(ctx, proof)
	if e != nil || b != configured || proof.Intent.Endpoint != EffortStartEndpoint(team, member) || proof.Intent.Team != team || proof.Intent.Member != member || proof.Intent.Effect != "run.create" || proof.Intent.InputDigest != effortauthority.Digest(struct{ Team, Member string }{team, member}) {
		return nil, effortauthority.ErrRefused
	}
	// A verified same-key retry is observation of its original protected
	// receipt. It never enters Tick/relaunch, even if the current leader ended
	// or Planner lost its local state after the atomic preparation commit.
	receipt, exists, err := f.Efforts.Authority.ObserveIngress(ctx, proof)
	if err != nil {
		return nil, effortauthority.ErrRefused
	}
	if exists {
		return receipt, nil
	}
	cfg, e := f.Executor.teamStore.GetHeartbeatConfig(ctx, team, member)
	if e != nil || cfg == nil || cfg.FiniteLeader == nil || cfg.ProfileKey != proof.Intent.Profile {
		return nil, effortauthority.ErrRefused
	}
	if _, e = f.prepareFiniteCaller(ctx, team, member, cfg); e != nil {
		return nil, e
	}
	if e = f.eligible(ctx, team, member, cfg); e != nil {
		return nil, e
	}
	// An external key selects the exact new leader ID. Its nonce and aggregate
	// preparation reservation are committed together before Tick's first write.
	ctx = context.WithValue(ctx, ingressProofKey{}, &proof)
	return f.Tick(ctx, team, member)
}
func EffortStartEndpoint(team, member string) string {
	return "/api/v1/teams/" + team + "/members/" + member + "/finite/start"
}

// EffortStartHandler is registered only at an explicitly approved setup. No
// new runtime route is automatically enabled by adding this implementation.
func (f *FiniteLeaderRuntime) EffortStartHandler(team, member string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != EffortStartEndpoint(team, member) || len(r.Header.Values(effortauthority.Header)) != 1 || strings.TrimSpace(r.Header.Get(effortauthority.Header)) == "" || len(r.Header.Values("Authorization")) != 0 || len(r.Header.Values("X-Agent-Identity-Token")) != 0 {
			http.Error(w, "finite caller refused", http.StatusUnauthorized)
			return
		}
		if r.ContentLength != 0 {
			http.Error(w, "finite start takes no body", http.StatusBadRequest)
			return
		}
		result, e := f.StartEffort(r.Context(), team, member, r.Header.Get(effortauthority.Header))
		if e != nil {
			http.Error(w, "finite caller refused", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
