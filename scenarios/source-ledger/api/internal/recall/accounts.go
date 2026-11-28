package recall

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"source-ledger/internal/policy"
)

const accountPrefix = "account:"

const accountPublications = `
SELECT e.id, e.kind, COALESCE(e.import_key,''), metadata.evidence,
       NOT EXISTS (
         SELECT 1
         FROM json_each(CASE WHEN json_valid(metadata.evidence) THEN metadata.evidence ELSE '{}' END, '$.evidence_ids') ref
         WHERE ref.type<>'text' OR NOT EXISTS (
           SELECT 1 FROM entries original WHERE original.id=ref.value AND original.scope=e.scope AND original.id<>e.id
         )
       )
FROM journal_observation_context metadata
JOIN entries e ON e.id=metadata.entry_id
WHERE metadata.routing_hint='defect-account' AND e.scope=?`

// An account is a published observation with a stable case ID and links to
// source observations. Scope-specific status lives alongside the authored
// summary so recall can show the current account while zoom retains history.
type accountEvidence struct {
	CaseID        string   `json:"case_id"`
	Revision      int      `json:"revision"`
	EvidenceIDs   []string `json:"evidence_ids"`
	ReconciledIDs []string `json:"reconciled_ids"`
	States        []struct {
		Scope  string `json:"scope"`
		Status string `json:"status"`
	} `json:"states"`
}

func decodeAccountEvidence(raw string) (accountEvidence, bool) {
	var evidence accountEvidence
	err := json.Unmarshal([]byte(raw), &evidence)
	for _, id := range evidence.ReconciledIDs {
		if !slices.Contains(evidence.EvidenceIDs, id) {
			return accountEvidence{}, false
		}
	}
	return evidence, err == nil && strings.HasPrefix(evidence.CaseID, "defect-") &&
		!strings.Contains(evidence.CaseID, "/") && evidence.Revision > 0 &&
		len(evidence.EvidenceIDs) > 0 && len(evidence.States) > 0
}

// Validate the publishing operation as well as the routing hint. Ordinary
// observations can carry arbitrary evidence without taking the place of an
// investigator account. The journal enforces one immutable write per key.
func (a accountEvidence) publicationKey(kind, key string) bool {
	switch kind {
	case "observation":
		return key == fmt.Sprintf("request:remember/scenario-qa/%s/%d/0", a.CaseID, a.Revision)
	case "legacy-bug-disposition":
		reportID, legacy := strings.CutPrefix(a.CaseID, "defect-legacy-")
		return legacy && key == fmt.Sprintf("request:legacy-scenario-qa-bugs/v1/closed/%s/%d", reportID, a.Revision)
	default:
		return false
	}
}

func (a accountEvidence) open() bool {
	if len(a.States) == 0 {
		return true
	}
	for _, state := range a.States {
		switch state.Status {
		case "fixed", "resolved", "closed", "dismissed":
		default:
			return true
		}
	}
	return false
}

// currentAccounts reads only explicit publications in this ledger scope.
// Revisions are selected by the investigator's sequence, not capture time.
func (s *SQLiteSource) currentAccounts(ctx context.Context) (map[string]accountEvidence, map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, accountPublications+" ORDER BY e.created_at DESC,e.id DESC", policy.ScopeFromContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	type account struct {
		id       string
		evidence accountEvidence
	}
	byCase := make(map[string]account)
	publications := make(map[string]bool)
	for rows.Next() {
		var id, kind, key, raw string
		var scopedEvidence bool
		if err := rows.Scan(&id, &kind, &key, &raw, &scopedEvidence); err != nil {
			return nil, nil, err
		}
		evidence, valid := decodeAccountEvidence(raw)
		if valid && scopedEvidence && evidence.publicationKey(kind, key) {
			// The sealed legacy importer only linked records from its own
			// defect history. New accounts explicitly identify stale claims.
			if kind == "legacy-bug-disposition" && len(evidence.ReconciledIDs) == 0 {
				evidence.ReconciledIDs = evidence.EvidenceIDs
			}
			publications[id] = true
			if evidence.Revision > byCase[evidence.CaseID].evidence.Revision {
				byCase[evidence.CaseID] = account{id, evidence}
			}
		}
	}
	latest := make(map[string]accountEvidence, len(byCase))
	for _, record := range byCase {
		latest[record.id] = record.evidence
	}
	return latest, publications, rows.Err()
}

func (s *SQLiteSource) AccountEvidence(ctx context.Context, id string) ([]Node, error) {
	entryID := strings.TrimPrefix(id, accountPrefix)
	var kind, key, raw string
	var scopedEvidence bool
	scope := policy.ScopeFromContext(ctx)
	err := s.db.QueryRowContext(ctx, accountPublications+" AND e.id=?", scope, entryID).
		Scan(&entryID, &kind, &key, &raw, &scopedEvidence)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	evidence, valid := decodeAccountEvidence(raw)
	if !valid || !scopedEvidence || !evidence.publicationKey(kind, key) {
		return nil, nil
	}
	ids, err := json.Marshal(append(evidence.EvidenceIDs, entryID))
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id,e.facet_id,e.body,e.created_at,COALESCE(context.evidence,''),COALESCE(context.routing_hint,''),e.actor_id,e.run_id,e.source_runtime
FROM entries e
LEFT JOIN journal_observation_context context ON context.entry_id=e.id
WHERE e.scope=? AND e.id IN (SELECT value FROM json_each(?))
ORDER BY e.created_at,e.id`, scope, string(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var created string
		if err := rows.Scan(&n.ID, &n.FacetID, &n.Text, &created, &n.Evidence, &n.RoutingHint, &n.ActorID, &n.RunID, &n.SourceRuntime); err != nil {
			return nil, err
		}
		n.EntryID = n.ID
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (n Node) contains(other Node) bool {
	return slices.Contains(n.ReconciledIDs, other.EntryID)
}

func (n *Node) applyAccount(account accountEvidence) {
	n.ID = accountPrefix + n.EntryID
	n.ParentID = ""
	n.Summary = true
	n.Span = len(account.EvidenceIDs)
	n.CaseID = account.CaseID
	n.Open = account.open()
	n.EvidenceIDs = account.EvidenceIDs
	n.ReconciledIDs = account.ReconciledIDs
}

// Historical evidence stays addressable in zoom. Current accounts replace its
// older claims in both category and general recall, regardless of labeling.
func historicalAccountNodes(nodes []Node, parents map[string]string) map[string]bool {
	linked := make(map[string]bool)
	for _, node := range nodes {
		for _, id := range node.ReconciledIDs {
			linked[id] = true
		}
	}
	if len(linked) == 0 {
		return nil
	}
	historical := make(map[string]bool)
	for _, node := range nodes {
		if node.CaseID != "" || !linked[node.EntryID] {
			continue
		}
		for id := node.ID; id != "" && !historical[id]; id = parents[id] {
			historical[id] = true
		}
	}
	return historical
}

// Let alternate wording in historical evidence help find the current account.
func accountRelevance(nodes []Node, scores []float64) map[string]float64 {
	const evidenceDiscount = 0.95
	byEntry := make(map[string]float64, len(nodes))
	for i, node := range nodes {
		byEntry[node.EntryID] = scores[i]
	}
	relevance := make(map[string]float64)
	for _, node := range nodes {
		for _, id := range node.ReconciledIDs {
			relevance[node.ID] = max(relevance[node.ID], byEntry[id]*evidenceDiscount)
		}
	}
	return relevance
}
