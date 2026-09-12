package hostpressure

import "context"

// ReclaimCandidate is the control-plane identity of a managed service, not a
// PID. Recycling is delegated to the lifecycle owner; this package never
// signals a process and never changes swap policy.
type ReclaimCandidate struct {
	Service string
	Process Process
}
type ReclaimDecision struct {
	Selected   *ReclaimCandidate
	HeldReason string
}
type ReclaimPolicy struct {
	SwapToResident float64
	MinimumSwapped uint64
	Saturated      func(context.Context) (bool, error)
	Serving        func(context.Context, string) (bool, error)
	Managed        func(context.Context, string) (bool, error)
	Recycle        func(context.Context, string) error
}

func ReclaimOne(ctx context.Context, processes []Process, candidates []ReclaimCandidate, policy ReclaimPolicy) (ReclaimDecision, error) {
	if policy.Saturated != nil {
		s, err := policy.Saturated(ctx)
		if err != nil {
			return ReclaimDecision{}, err
		}
		if s {
			return ReclaimDecision{HeldReason: "host saturated; lifecycle recovery is braked"}, nil
		}
	}
	stranded := Stranded(processes, policy.SwapToResident)
	for _, p := range stranded {
		if p.Swapped < policy.MinimumSwapped {
			continue
		}
		for i := range candidates {
			if candidates[i].Process.PID != p.PID {
				continue
			}
			managed := true
			if policy.Managed != nil {
				var err error
				managed, err = policy.Managed(ctx, candidates[i].Service)
				if err != nil {
					return ReclaimDecision{}, err
				}
			}
			if !managed {
				continue
			}
			serving := false
			if policy.Serving != nil {
				var err error
				serving, err = policy.Serving(ctx, candidates[i].Service)
				if err != nil {
					return ReclaimDecision{}, err
				}
			}
			if serving {
				continue
			}
			if policy.Recycle == nil {
				return ReclaimDecision{HeldReason: "lifecycle recycle action unavailable"}, nil
			}
			if err := policy.Recycle(ctx, candidates[i].Service); err != nil {
				return ReclaimDecision{}, err
			}
			return ReclaimDecision{Selected: &candidates[i]}, nil
		}
	}
	return ReclaimDecision{HeldReason: "no idle managed stranded service met the reclaim bar"}, nil
}
