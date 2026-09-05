import { ShieldAlert } from "lucide-react";

import {
  APPLY_CONVERGENCE_NOTE,
  APPLY_KIND_ACTIONS,
  APPLY_NO_REMOVAL_NOTE,
  type ApplyPlanItem,
  groupByKind,
  kindLabel,
  summarizeApplyPlan,
} from "../../lib/applyPlan";

/**
 * The operator's disclosure before consenting to host changes.
 *
 * It replaces a flat "kind: name · required" list and a hedge that said
 * safeguards "may require privilege". The API already reports which items are
 * privileged and which are present on this host, so both are stated as facts
 * rather than left for the operator to guess. This mirrors the wizard CLI
 * renderer; the two surfaces share lib/applyPlan.ts so they cannot drift.
 */
export function ApplyPlanDisclosure({ items }: { items: ApplyPlanItem[] }) {
  if (items.length === 0) {
    return (
      <p data-testid="apply-plan" role="note" className="mt-2 text-sm text-muted">
        Current selection has no consented host changes.
      </p>
    );
  }

  const summary = summarizeApplyPlan(items);

  return (
    <div data-testid="apply-plan">
      <p data-testid="apply-plan-summary" className="mt-2 text-sm text-foreground">
        {summary.total} selected item{summary.total === 1 ? "" : "s"}: {summary.pending.length} not yet in
        place, {summary.satisfied.length} already in place, {summary.unknown.length} not sampled.
      </p>

      <details data-testid="apply-plan-effects" className="mt-3 rounded-lg border border-border bg-surface p-3 text-sm">
        <summary className="cursor-pointer font-medium text-foreground">What &quot;apply&quot; does to this host</summary>
        <ul role="list" className="mt-2 space-y-1 text-muted">
          {APPLY_KIND_ACTIONS.map((action) => (
            <li key={action.kind}>
              <span className="font-medium text-foreground">{action.kind}</span> — <code>{action.command}</code> — {action.effect}
            </li>
          ))}
        </ul>
        <p className="mt-2 text-muted">{APPLY_CONVERGENCE_NOTE}</p>
        <p className="mt-1 text-muted">{APPLY_NO_REMOVAL_NOTE}</p>
      </details>

      {summary.pending.length > 0 && (
        <PlanSection
          testID="apply-plan-pending"
          heading={`Not yet in place — these change this host (${summary.pending.length}${summary.elevatedPending > 0 ? `, ${summary.elevatedPending} elevated` : ""})`}
          items={summary.pending}
          tone="warning"
          detailed
        />
      )}
      {summary.satisfied.length > 0 && (
        <PlanSection
          testID="apply-plan-satisfied"
          heading={`Already in place — verified present on this host (${summary.satisfied.length})`}
          items={summary.satisfied}
          tone="muted"
        />
      )}
      {summary.unknown.length > 0 && (
        <PlanSection
          testID="apply-plan-unknown"
          heading={`Not sampled — checking these costs a control-plane round trip each, so this plan did not (${summary.unknown.length})`}
          items={summary.unknown}
          tone="muted"
        />
      )}

      {summary.elevatedTotal > 0 ? (
        <p data-testid="privilege-warning" role="note" className="mt-3 flex items-start gap-2 text-sm text-warning">
          <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>
            {summary.elevatedTotal} item{summary.elevatedTotal === 1 ? "" : "s"} run with elevated privilege and
            change host state. No undeclared host action is performed.
          </span>
        </p>
      ) : (
        <p data-testid="privilege-warning" role="note" className="mt-3 text-sm text-muted">
          No item in this plan requires elevated privilege. No undeclared host action is performed.
        </p>
      )}
    </div>
  );
}

/**
 * Detailed lines are reserved for the group that changes the host. The rest are
 * summarized one line per kind so a long already-satisfied list cannot bury the
 * part the operator is actually deciding about.
 */
function PlanSection({
  testID,
  heading,
  items,
  tone,
  detailed = false,
}: {
  testID: string;
  heading: string;
  items: ApplyPlanItem[];
  tone: "warning" | "muted";
  detailed?: boolean;
}) {
  return (
    <section data-testid={testID} className="mt-3" aria-label={heading}>
      <h3 className={`text-sm font-medium ${tone === "warning" ? "text-warning" : "text-muted"}`}>{heading}</h3>
      <ul role="list" className="mt-1 space-y-1 text-sm text-muted">
        {groupByKind(items).map((group) => (
          <li key={group.kind}>
            <span className="font-medium text-foreground">{kindLabel(group.kind, group.items.length)} ({group.items.length})</span>
            {detailed ? (
              <ul role="list" className="mt-1 space-y-0.5 pl-4">
                {group.items.map((item) => (
                  <li key={item.id} data-testid={`apply-plan-item-${item.id}`}>
                    {item.privileged && <span aria-label="elevated" className="mr-1 text-warning">!</span>}
                    {item.name}
                    <span className="text-muted"> ({item.required ? "required" : "optional"}{item.privileged ? ", elevated" : ""})</span>
                  </li>
                ))}
              </ul>
            ) : (
              <span className="ml-1">{group.items.map((item) => item.name).join(", ")}</span>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
