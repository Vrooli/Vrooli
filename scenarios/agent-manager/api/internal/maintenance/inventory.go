package maintenance

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	platform "github.com/vrooli/platform-go"
)

// Inventory is a bounded read projection, not a scheduler or process remediator.
// A terminal database status never overrides positive physical executor evidence.
type Inventory struct {
	Remaining       *int               `json:"remaining"`
	Work            []WorkRef          `json:"work"`
	Executors       []ExecutorEvidence `json:"executors"`
	Unknown         []string           `json:"unknown"`
	ControlPlaneGap string             `json:"controlPlaneGap"`
}

type WorkRef struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type ExecutorRef struct {
	WorkRef
	PID  int `json:"pid"`
	PGID int `json:"pgid"`
}

type ExecutorEvidence struct {
	ExecutorRef
	Alive       bool  `json:"alive"`
	HasChildren bool  `json:"hasChildren"`
	PIDs        []int `json:"pids,omitempty"`
}

type InventoryDatabase interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type InventoryReader struct {
	db            InventoryDatabase
	physical      func(context.Context, ExecutorRef) (ExecutorEvidence, error)
	batchPhysical func(context.Context, []ExecutorRef) ([]ExecutorEvidence, error)
}

func NewInventory(db InventoryDatabase, physical func(context.Context, ExecutorRef) (ExecutorEvidence, error)) *InventoryReader {
	return &InventoryReader{db: db, physical: physical}
}

func NewBatchedInventory(db InventoryDatabase, physical func(context.Context, []ExecutorRef) ([]ExecutorEvidence, error)) *InventoryReader {
	return &InventoryReader{db: db, batchPhysical: physical}
}

// RecordedExecutor reads shared platform evidence only. A missing recorded root
// does not exclude an orphaned process group. Only the control plane can supply
// complete scope evidence for that case; no /proc walker or shell repair belongs
// here. This conservative adapter must not be represented as host-wide inventory.
func RecordedExecutor(ctx context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
	result := ExecutorEvidence{ExecutorRef: ref}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if ref.PID <= 0 {
		return result, fmt.Errorf("recorded executor root is unavailable; control-plane scope proof required")
	}
	_, err := platform.ProcessName(ref.PID)
	if os.IsNotExist(err) {
		return result, fmt.Errorf("root %d exited; control-plane descendant/group exclusion required", ref.PID)
	}
	if err != nil {
		return result, err
	}
	result.Alive = true
	result.HasChildren, err = platform.ProcessHasChildren(ref.PID)
	return result, err
}

func (r *InventoryReader) Remaining(ctx context.Context) (int, error) {
	state, err := r.Observe(ctx)
	if err != nil || state.Remaining == nil {
		if err == nil {
			err = fmt.Errorf("executor inventory is incomplete")
		}
		return -1, err
	}
	return *state.Remaining, nil
}

func (r *InventoryReader) Observe(parent context.Context) (Inventory, error) {
	state := Inventory{Work: []WorkRef{}, Executors: []ExecutorEvidence{}, Unknown: []string{}, ControlPlaneGap: "AM admission drain is not host lifecycle authorization. A runner or descendant can restart another scenario whose dependencies include AM or Plan Manager. The control plane must own dependency-impact exclusion and complete executor-scope evidence; this handler never stops or repairs host processes."}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if r == nil || r.db == nil || (r.physical == nil && r.batchPhysical == nil) {
		return state, fmt.Errorf("durable and physical inventory observers are required")
	}
	// Resting review is not execution: its future continuation must acquire the
	// admission fence. Physical executors and active finalization still count.
	// Unknown/new statuses remain conservative. Historical accounting is retained,
	// not replayed or deleted to establish an empty execution inventory.
	// Imported mode is the owner's explicit read-only transcript projection, not
	// an admitted executor, even when historical status is unknown or running.
	// Retained imported PIDs describe foreign history, not AM-owned execution.
	// Active finalization contradicts that ownership and stays explicit unknown.
	type candidate struct {
		ref                  ExecutorRef
		logical              bool
		importedFinalization bool
	}
	rows, err := r.db.QueryContext(ctx, `WITH candidates(id,kind,status,pid,pgid,logical,imported_finalization) AS (
 SELECT id,'run',COALESCE(status,'unknown'),COALESCE(runner_pid,0),COALESCE(runner_pgid,0),
 CASE WHEN (status IN ('complete','failed','cancelled','needs_review') OR execution_mode='imported') AND COALESCE(finalization_status,'none') NOT IN ('pending','running') THEN 0 ELSE 1 END,
 COALESCE(execution_mode,'')='imported' AND COALESCE(finalization_status,'none') IN ('pending','running')
 FROM runs WHERE (COALESCE(execution_mode,'') <> 'imported' AND (status IS NULL OR status NOT IN ('complete','failed','cancelled','needs_review') OR runner_pid>0 OR runner_pgid>0)) OR finalization_status IN ('pending','running')
 UNION ALL SELECT id,'workflow',status,0,0,1,0 FROM workflow_executions WHERE status NOT IN ('succeeded','blocked','abstained','budget_exhausted','failed','cancelled')
 ) SELECT id,kind,status,pid,pgid,logical,imported_finalization FROM candidates ORDER BY kind,id`)
	if err != nil {
		return state, fmt.Errorf("durable drain inventory: %w", err)
	}
	defer rows.Close()
	// Stream one SQL result under the attachment deadline. Repeating a sorted
	// union for each small page rescans history and can consume the entire budget
	// before the host read. Only bounded references and response samples are kept.
	// Close the SQL cursor before identity hydration or any physical observer.
	count := 0
	var physicalRefs []ExecutorRef
	logicalRefs := map[string]bool{}
	physicalCount := 0
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ref.ID, &c.ref.Kind, &c.ref.Status, &c.ref.PID, &c.ref.PGID, &c.logical, &c.importedFinalization); err != nil {
			return state, err
		}
		if err := ctx.Err(); err != nil {
			return state, err
		}
		if c.logical {
			count++
			if len(state.Work) < 64 {
				state.Work = append(state.Work, c.ref.WorkRef)
			}
		}
		if c.importedFinalization && len(state.Unknown) < 64 {
			state.Unknown = append(state.Unknown, c.ref.ID+": imported read-only run claims active finalization; ownership reconciliation required")
		}
		if !c.importedFinalization && (c.ref.PID > 0 || c.ref.PGID > 0) {
			physicalCount++
			if len(physicalRefs) < executorScopeMaxReferences {
				physicalRefs = append(physicalRefs, c.ref)
				logicalRefs[c.ref.ID] = c.logical
			}
		}
	}
	if err := rows.Err(); err != nil {
		return state, err
	}
	if err := rows.Close(); err != nil {
		return state, err
	}
	// One host snapshot per inventory attachment, not one scan per row/page.
	// A genuine protocol capacity limit is explicit and cannot become empty proof.
	if physicalCount > executorScopeMaxReferences {
		message := fmt.Sprintf("%d physical references exceed executor-scope capacity %d; control-plane batched capacity is required", physicalCount, executorScopeMaxReferences)
		state.Unknown = append(state.Unknown, message)
		return state, fmt.Errorf("%s", message)
	}
	if len(physicalRefs) > 0 {
		var evidence []ExecutorEvidence
		var err error
		if r.batchPhysical != nil {
			evidence, err = r.batchPhysical(ctx, physicalRefs)
		} else {
			// Isolated single-ref observer seam; production always uses one batch.
			for _, ref := range physicalRefs {
				item, readErr := r.physical(ctx, ref)
				item.ExecutorRef = ref
				evidence = append(evidence, item)
				if readErr != nil && len(state.Unknown) < 64 {
					state.Unknown = append(state.Unknown, ref.ID+": "+readErr.Error())
				}
			}
		}
		batch := map[string]ExecutorEvidence{}
		for _, item := range evidence {
			batch[item.ID] = item
		}
		for _, ref := range physicalRefs {
			item, ok := batch[ref.ID]
			item.ExecutorRef = ref
			if item.Alive {
				if !logicalRefs[ref.ID] {
					count++
				}
				if len(state.Executors) < 64 {
					state.Executors = append(state.Executors, item)
				}
			}
			if !ok && len(state.Unknown) < 64 {
				state.Unknown = append(state.Unknown, ref.ID+": control-plane executor evidence missing")
			}
		}
		if err != nil {
			state.Unknown = append(state.Unknown, err.Error())
		}
	}
	if err := ctx.Err(); err != nil {
		return state, err
	}
	if len(state.Unknown) != 0 {
		return state, fmt.Errorf("physical executor inventory is incomplete")
	}
	state.Remaining = &count
	return state, nil
}
