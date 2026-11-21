package backup

import (
	"context"
	"sort"
	"time"

	"scenario-to-cloud/domain"
)

// References are the live holders that protect recovery points. A point is
// protected while the release it was captured under is active or retained
// (rollback eligibility depends on it) or while a non-terminal operation
// names it (a restore or forward repair in flight).
type References struct {
	ActiveReleases        []string `json:"active_releases"`
	RetainedReleases      []string `json:"retained_releases"`
	NonTerminalOperations []string `json:"non_terminal_operations"`
	// PinnedRecoveryPoints are ids an operator or a rollback plan pinned.
	PinnedRecoveryPoints []string `json:"pinned_recovery_points"`
}

// Protection reports whether refs protect rp and who holds it.
func Protection(rp domain.RecoveryPoint, refs References) (bool, []string) {
	holders := []string{}
	for _, digest := range refs.ActiveReleases {
		if digest != "" && digest == rp.ReleaseDigest {
			holders = append(holders, "release:active:"+digest)
		}
	}
	for _, digest := range refs.RetainedReleases {
		if digest != "" && digest == rp.ReleaseDigest {
			holders = append(holders, "release:retained:"+digest)
		}
	}
	for _, op := range refs.NonTerminalOperations {
		if op != "" && op == rp.OperationID {
			holders = append(holders, "operation:"+op)
		}
	}
	for _, id := range refs.PinnedRecoveryPoints {
		if id == rp.ID {
			holders = append(holders, "pinned:"+id)
		}
	}
	sort.Strings(holders)
	return len(holders) > 0, holders
}

// Policy is the retention policy retention applies to unprotected points.
// Zero values mean "no limit" for that dimension.
type Policy struct {
	// KeepLast keeps this many newest unprotected points per deployment.
	KeepLast int `json:"keep_last"`
	// MaxAge expires unprotected points older than this.
	MaxAge time.Duration `json:"max_age_ns"`
}

// PrunePlan is what retention decided. Protected points are listed with
// their holders and are never in Delete.
type PrunePlan struct {
	Keep      []string            `json:"keep"`
	Delete    []string            `json:"delete"`
	Protected map[string][]string `json:"protected"`
}

// Plan computes retention over points without touching storage.
func Plan(points []domain.RecoveryPoint, refs References, policy Policy, now time.Time) PrunePlan {
	plan := PrunePlan{Keep: []string{}, Delete: []string{}, Protected: map[string][]string{}}
	unprotected := make([]domain.RecoveryPoint, 0, len(points))
	for _, p := range points {
		if protected, holders := Protection(p, refs); protected {
			plan.Protected[p.ID] = holders
			plan.Keep = append(plan.Keep, p.ID)
			continue
		}
		unprotected = append(unprotected, p)
	}
	sort.Slice(unprotected, func(i, j int) bool { return unprotected[i].CapturedAt.After(unprotected[j].CapturedAt) })
	for i, p := range unprotected {
		expired := policy.MaxAge > 0 && now.Sub(p.CapturedAt) > policy.MaxAge
		overflow := policy.KeepLast > 0 && i >= policy.KeepLast
		if expired || overflow {
			plan.Delete = append(plan.Delete, p.ID)
			continue
		}
		plan.Keep = append(plan.Keep, p.ID)
	}
	sort.Strings(plan.Keep)
	sort.Strings(plan.Delete)
	return plan
}

// Prune records protection on every stored point of the deployment and
// deletes only the unprotected points the policy retires. A protected point
// is never deleted; the repository refuses even a direct delete of one.
func (s *Service) Prune(ctx context.Context, deploymentID string, refs References, policy Policy) (PrunePlan, error) {
	points, err := s.Repo.ListRecoveryPoints(ctx, deploymentID)
	if err != nil {
		return PrunePlan{}, err
	}
	for _, p := range points {
		protected, holders := Protection(p, refs)
		if p.Protected != protected || !equalStrings(p.ProtectedBy, holders) {
			if _, err := s.Repo.SetRecoveryPointProtection(ctx, p.ID, protected, holders); err != nil {
				return PrunePlan{}, err
			}
		}
	}
	plan := Plan(points, refs, policy, s.now())
	for _, id := range plan.Delete {
		if err := s.Repo.DeleteRecoveryPoint(ctx, id); err != nil {
			return plan, err
		}
		s.removeRecoveryPointDir(deploymentID, id)
	}
	return plan, nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
