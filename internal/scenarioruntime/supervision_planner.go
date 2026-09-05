package scenarioruntime

import (
	"sort"
	"time"
)

const defaultSupervisionBatchSize = 100

type SupervisionPlanInput struct {
	Now              time.Time
	CurrentBootID    string
	SupervisorID     string
	Instances        []Instance
	ClaimsByInstance map[string][]PortClaim
	RefsByInstance   map[string][]ProcessRef
	Processes        map[string]ProcessEvidence
	Listeners        map[int]ListenerEvidence
	HealthByInstance map[string]HealthSnapshot
	HealthInterval   time.Duration
	BatchSize        int
}

type PlannedInstance struct {
	Instance       Instance
	Classification ReconcileClassification
	Reason         string
}

type SupervisionPlan struct {
	RenewalBatches [][]SupervisionClaim
	Unverified     []PlannedInstance
	Expire         []PlannedInstance
	HealthProbes   []string
}

func PlanSupervision(in SupervisionPlanInput) SupervisionPlan {
	if in.Now.IsZero() {
		in.Now = time.Now().UTC()
	}
	batchSize := in.BatchSize
	if batchSize <= 0 {
		batchSize = defaultSupervisionBatchSize
	}

	instances := append([]Instance(nil), in.Instances...)
	sort.SliceStable(instances, func(i, j int) bool {
		if instances[i].Scenario == instances[j].Scenario {
			return instances[i].Generation > instances[j].Generation
		}
		return instances[i].Scenario < instances[j].Scenario
	})

	plan := SupervisionPlan{}
	for _, instance := range instances {
		if instance.Status != StatusRunning {
			continue
		}
		claims := in.ClaimsByInstance[instance.InstanceID]
		reconcileInstance := instance
		if in.SupervisorID != "" {
			reconcileInstance.SupervisorID = ""
		}
		result := ReconcileRuntime(ReconcileInput{
			Now:           in.Now,
			CurrentBootID: in.CurrentBootID,
			Instance:      reconcileInstance,
			Claims:        claims,
			ProcessRefs:   in.RefsByInstance[instance.InstanceID],
			Processes:     in.Processes,
			Listeners:     in.Listeners,
		})
		if !result.Authoritative {
			item := PlannedInstance{Instance: instance, Classification: result.Classification, Reason: result.Reason}
			switch result.Classification {
			case ReconcileStaleInstance:
				plan.Expire = append(plan.Expire, item)
			default:
				plan.Unverified = append(plan.Unverified, item)
			}
			continue
		}
		supervisorID := instance.SupervisorID
		if in.SupervisorID != "" && supervisorID != in.SupervisorID {
			supervisorID = in.SupervisorID
		}
		if supervisorID == "" {
			plan.Unverified = append(plan.Unverified, PlannedInstance{
				Instance:       instance,
				Classification: ReconcileUnverified,
				Reason:         "no supervisor ownership is available",
			})
			continue
		}
		plan.addRenewal(batchSize, SupervisionClaim{
			InstanceID:   instance.InstanceID,
			Generation:   instance.Generation,
			SupervisorID: supervisorID,
		})
		if shouldProbeHealth(in.HealthByInstance[instance.InstanceID], in.Now, in.HealthInterval) {
			plan.HealthProbes = append(plan.HealthProbes, instance.InstanceID)
		}
	}
	return plan
}

func (p *SupervisionPlan) addRenewal(batchSize int, claim SupervisionClaim) {
	if batchSize <= 0 {
		batchSize = defaultSupervisionBatchSize
	}
	last := len(p.RenewalBatches) - 1
	if last < 0 || len(p.RenewalBatches[last]) >= batchSize {
		p.RenewalBatches = append(p.RenewalBatches, []SupervisionClaim{})
		last++
	}
	p.RenewalBatches[last] = append(p.RenewalBatches[last], claim)
}

func shouldProbeHealth(snapshot HealthSnapshot, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}
	if snapshot.CheckedAt == nil {
		return true
	}
	return !snapshot.CheckedAt.Add(interval).After(now)
}
