package gates

import (
	"context"
	"fmt"
	"strings"

	"react-component-library/internal/graphreconcile"
)

func ValidateGraphReconciled(scope Scope) (Result, error) {
	root := scope.Root
	ctx := scope.Context
	if ctx == nil {
		ctx = context.Background()
	}
	var report graphreconcile.Report
	var err error
	if len(scope.Assets) > 0 {
		report, err = graphreconcile.ReconcileScoped(ctx, root, scope.Assets)
	} else {
		report, err = graphreconcile.Reconcile(ctx, root)
	}
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(report.Assets)}
	for _, row := range report.Assets {
		if row.Verdict != graphreconcile.ImportsUnavailable {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.graph_reconciled_unavailable", AssetID: "__corpus__.graph-reconciled",
			Message:     fmt.Sprintf("imports-unavailable: %s", row.Cause),
			Remediation: "Start typescript-code-graph and confirm it has indexed scenarios/react-component-library; the gate is blocking because an unavailable import graph cannot prove dependency reconciliation.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
		return result, nil
	}
	for _, row := range report.Assets {
		// A not-implemented production asset is a census result, not graph
		// drift. Calibration deliberately plants that same state, however, and
		// must remain observable so the gate proves it can discriminate it.
		if row.Verdict == graphreconcile.Reconciled || (row.Verdict == graphreconcile.NotImplemented && !strings.HasPrefix(row.AssetID, "calibration.")) {
			continue
		}
		remediation := "Bring the three dependency views into agreement: the requires edges in catalog/assets/, the dependencies[] pins in the asset's library/**/component.json, and the imports the source actually makes. Whichever two agree usually identifies the stale one. The gate reports and never edits library/ on your behalf."
		switch row.Verdict {
		case graphreconcile.ImportsUnavailable:
			remediation = "The reconciler could not obtain an import graph from typescript-code-graph, so the source-import view is missing and no reconciliation verdict is possible. Start the graph service and confirm it has indexed scenarios/react-component-library."
		case graphreconcile.NotImplemented:
			// The catalog is desired state, so this is the expected resting
			// verdict for anything not built yet. It is reported for census,
			// not as drift, and carries no reconciliation advice: there is only
			// one dependency view to read.
			remediation = "No library implementation exists for this catalog asset, so there is nothing to reconcile against its requires edges. This is a construction gap rather than dependency drift; use `react-component-library catalog next` to decide whether this asset is worth building."
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.graph_reconciled", AssetID: row.AssetID,
			Message:     fmt.Sprintf("%s: %s", row.Verdict, row.Cause),
			Remediation: remediation,
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
	}
	return nonEmpty(result, "graph-reconciled"), nil
}

// ValidateReleaseProvenance rejects release directories that did not pass
// through the draft publisher. Historical releases carry an explicit
// backfilled marker; new releases must name their draft and publication time.
