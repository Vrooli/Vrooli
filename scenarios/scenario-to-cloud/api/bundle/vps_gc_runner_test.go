package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// fakeOwner is an in-memory target owner: it answers `release list` from its
// store and honours `release prune` the way the real owner does (active and
// previous are refused whatever the request names). It records every verb
// so tests can prove nothing but typed verbs reached the target.
type fakeOwner struct {
	releases map[string]TargetRelease
	active   string
	previous string
	verbs    []string
	prunes   int
}

var _ reach.Reach = (*fakeOwner)(nil)

func (f *fakeOwner) Exec(_ context.Context, _ identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if err := reach.ValidateCommand(cmd); err != nil {
		return reach.Result{}, err
	}
	f.verbs = append(f.verbs, strings.Join(cmd.Argv(), " "))
	switch cmd.Verb {
	case "cloud-target release list":
		listing := TargetReleaseListing{DeploymentID: "dep"}
		if f.active != "" {
			listing.Active = &struct {
				ActiveRelease   string `json:"active_release"`
				PreviousRelease string `json:"previous_release"`
			}{ActiveRelease: f.active, PreviousRelease: f.previous}
		}
		for _, rel := range f.releases {
			rel.Role = "staged"
			switch rel.Digest {
			case f.active:
				rel.Role = "active"
			case f.previous:
				rel.Role = "previous"
			}
			listing.Releases = append(listing.Releases, rel)
		}
		sort.Slice(listing.Releases, func(i, j int) bool { return listing.Releases[i].Digest < listing.Releases[j].Digest })
		raw, _ := json.Marshal(listing)
		return reach.Result{Stdout: string(raw)}, nil
	case "cloud-target release prune":
		if !cmd.Effectful {
			return reach.Result{}, fmt.Errorf("prune must be effectful")
		}
		f.prunes++
		report := PruneReport{Deleted: []string{}, Refused: map[string]string{}}
		for i, a := range cmd.Args {
			if a != "--release" || i+1 >= len(cmd.Args) {
				continue
			}
			d := cmd.Args[i+1]
			switch d {
			case f.active:
				report.Refused[d] = "active"
			case f.previous:
				report.Refused[d] = "previous"
			default:
				if rel, ok := f.releases[d]; ok {
					report.ReclaimedBytes += rel.SizeBytes
					delete(f.releases, d)
				}
				report.Deleted = append(report.Deleted, d)
			}
		}
		raw, _ := json.Marshal(map[string]any{"report": report})
		return reach.Result{Stdout: string(raw)}, nil
	}
	return reach.Result{ExitCode: 2, Stdout: `{"error":{"code":"unknown_verb","message":"` + cmd.Verb + `"}}`}, nil
}

func (f *fakeOwner) Deliver(context.Context, identity.TargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	return reach.DeliveryReceipt{}, nil
}

func (f *fakeOwner) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return reach.Capabilities{Online: true, NativeCLI: true}, nil
}

func digestN(n int) string { return fmt.Sprintf("%064x", n) }

func ownerWith(n int) *fakeOwner {
	f := &fakeOwner{releases: map[string]TargetRelease{}}
	for i := 0; i < n; i++ {
		d := digestN(i)
		f.releases[d] = TargetRelease{Digest: d, State: "complete", SizeBytes: 10, ModTime: fmt.Sprintf("2026-02-%02dT03:00:00Z", i+1), BundleSHA256: fmt.Sprintf("%064x", 1000+i)}
	}
	return f
}

func gcTarget() identity.TargetRef {
	return identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", Workdir: "/root/Vrooli"}}
}

// [REQ:STC-P0-026] A dry run reads the owner's listing, reports the plan and
// asks the owner to prune nothing.
func TestGCTargetReleases_DryRunReturnsPlanAndDoesNotPrune(t *testing.T) {
	owner := ownerWith(3)
	resp := GCTargetReleases(context.Background(), owner, gcTarget(), "dep", "app", domain.VPSBundleGCRequest{ScenarioID: "app", KeepLatest: 2, DryRun: true})
	if !resp.OK || !resp.DryRun {
		t.Fatalf("expected ok dry-run, got ok=%v dry=%v err=%q", resp.OK, resp.DryRun, resp.Error)
	}
	if resp.DeletedCount != 1 || owner.prunes != 0 || len(owner.releases) != 3 {
		t.Fatalf("dry run must plan one deletion and prune nothing: count=%d prunes=%d releases=%d", resp.DeletedCount, owner.prunes, len(owner.releases))
	}
	for _, v := range owner.verbs {
		if !strings.HasPrefix(v, "vrooli cloud-target release list") {
			t.Fatalf("dry run issued a non-list verb: %s", v)
		}
	}
}

// [REQ:STC-P0-026] Execution prunes only the digests the plan named, in
// one typed verb, and the owner keeps active and previous even when the
// cloud names them.
func TestGCTargetReleases_PrunesThroughTheOwnerAndHonoursOwnerRefusals(t *testing.T) {
	owner := ownerWith(120)
	resp := GCTargetReleases(context.Background(), owner, gcTarget(), "dep", "app", domain.VPSBundleGCRequest{ScenarioID: "app", KeepLatest: 2})
	if !resp.OK {
		t.Fatalf("expected ok, got %q", resp.Error)
	}
	if resp.DeletedCount != 118 || len(owner.releases) != 2 || owner.prunes != 1 {
		t.Fatalf("deleted=%d remaining=%d prunes=%d", resp.DeletedCount, len(owner.releases), owner.prunes)
	}
	if resp.DeletedBytes != 1180 {
		t.Fatalf("reclaimed bytes must come from the owner's report, got %d", resp.DeletedBytes)
	}

	// Active and previous survive even a keep_latest=1 plan that would name
	// them by age: the planner keeps them by role and the owner refuses them
	// if ever asked.
	owner = ownerWith(4)
	owner.active = digestN(0)
	owner.previous = digestN(1)
	resp = GCTargetReleases(context.Background(), owner, gcTarget(), "dep", "app", domain.VPSBundleGCRequest{ScenarioID: "app", KeepLatest: 1})
	if !resp.OK {
		t.Fatalf("expected ok, got %q", resp.Error)
	}
	if _, ok := owner.releases[digestN(0)]; !ok {
		t.Fatal("active release was pruned")
	}
	if _, ok := owner.releases[digestN(1)]; !ok {
		t.Fatal("previous release was pruned")
	}
	if len(owner.releases) != 3 || resp.DeletedCount != 1 {
		t.Fatalf("expected one staged release pruned, remaining=%d deleted=%d", len(owner.releases), resp.DeletedCount)
	}

	// An owner refusal is reported, not hidden.
	owner = ownerWith(3)
	owner.active = digestN(0)
	kept, deleted, _ := PlanVPSBundleGC([]domain.VPSBundleInfo{{Filename: digestN(0), ScenarioID: "app", ModTime: "2026-01-01T00:00:00Z"}, {Filename: digestN(1), ScenarioID: "app", ModTime: "2026-01-02T00:00:00Z"}, {Filename: digestN(2), ScenarioID: "app", ModTime: "2026-01-03T00:00:00Z"}}, "app", 1, nil)
	if len(kept) != 1 || len(deleted) != 2 {
		t.Fatalf("planner without roles: kept=%d deleted=%d", len(kept), len(deleted))
	}
	report, err := PruneTargetReleases(context.Background(), owner, gcTarget(), "dep", []string{digestN(0), digestN(1)})
	if err != nil || report.Refused[digestN(0)] != "active" || len(report.Deleted) != 1 {
		t.Fatalf("owner refusal must surface: %+v err=%v", report, err)
	}
}

// [REQ:STC-P0-024] Only well-formed digests reach the owner: the cloud
// refuses to compose a prune with anything else, before any transport call.
func TestPruneTargetReleasesRefusesMalformedDigests(t *testing.T) {
	owner := ownerWith(1)
	if _, err := PruneTargetReleases(context.Background(), owner, gcTarget(), "dep", []string{"../etc/passwd"}); err == nil {
		t.Fatal("malformed digest must be refused")
	}
	if len(owner.verbs) != 0 {
		t.Fatalf("refused prune reached the target: %v", owner.verbs)
	}
}
