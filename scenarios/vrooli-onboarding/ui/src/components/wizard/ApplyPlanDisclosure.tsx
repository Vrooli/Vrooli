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
import { i18n } from "../../i18n";

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
      <>
        <p data-testid="apply-plan" role="note" className="mt-2 text-sm text-muted">
          {i18n.t("onboarding.applyDisclosure.noChanges")}
        </p>
        <p data-testid="privilege-warning" role="note" className="mt-3 text-sm text-muted">
          {i18n.t("onboarding.applyDisclosure.noPrivilege")}
        </p>
      </>
    );
  }

  const summary = summarizeApplyPlan(items);

  return (
    <div data-testid="apply-plan">
      <p data-testid="apply-plan-summary" className="mt-2 text-sm text-foreground">
        {i18n.t("onboarding.applyDisclosure.summary", { total: summary.total, plural: summary.total === 1 ? "" : "s", pending: summary.pending.length, satisfied: summary.satisfied.length, unknown: summary.unknown.length })}
      </p>

      <details data-testid="apply-plan-effects" className="mt-3 rounded-lg border border-border bg-surface p-3 text-sm">
        <summary className="cursor-pointer font-medium text-foreground">{i18n.t("onboarding.applyDisclosure.effects")}</summary>
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
          heading={i18n.t("onboarding.applyDisclosure.pending", { count: summary.pending.length, elevated: summary.elevatedPending > 0 ? i18n.t("onboarding.applyDisclosure.elevated", { count: summary.elevatedPending }) : "" })}
          items={summary.pending}
          tone="warning"
          detailed
        />
      )}
      {summary.satisfied.length > 0 && (
        <PlanSection
          testID="apply-plan-satisfied"
          heading={i18n.t("onboarding.applyDisclosure.satisfied", { count: summary.satisfied.length })}
          items={summary.satisfied}
          tone="muted"
        />
      )}
      {summary.unknown.length > 0 && (
        <PlanSection
          testID="apply-plan-unknown"
          heading={i18n.t("onboarding.applyDisclosure.unknown", { count: summary.unknown.length })}
          items={summary.unknown}
          tone="muted"
        />
      )}

      {summary.elevatedTotal > 0 ? (
        <p data-testid="privilege-warning" role="note" className="mt-3 flex items-start gap-2 text-sm text-warning">
          <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>
            {i18n.t("onboarding.applyDisclosure.elevatedWarning", { count: summary.elevatedTotal, plural: summary.elevatedTotal === 1 ? "" : "s" })}
          </span>
        </p>
      ) : (
        <p data-testid="privilege-warning" role="note" className="mt-3 text-sm text-muted">
          {i18n.t("onboarding.applyDisclosure.noPrivilege")}
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
                    {item.privileged && <span aria-label={i18n.t("onboarding.applyDisclosure.elevatedLabel")} className="mr-1 text-warning">!</span>}
                    {item.name}
                    <span className="text-muted"> ({item.required ? i18n.t("onboarding.applyDisclosure.required") : i18n.t("onboarding.applyDisclosure.optional")}{item.privileged ? `, ${i18n.t("onboarding.applyDisclosure.elevatedLabel")}` : ""})</span>
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
