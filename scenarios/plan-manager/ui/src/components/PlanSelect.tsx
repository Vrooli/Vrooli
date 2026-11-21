import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { usePlansList } from "../features/plans/usePlans";

/**
 * Shared plan picker used by the Execution, Validation, and Velocity boards.
 * Backed by the same `usePlansList` query (so it shares the cache) and emits the
 * selected plan id. Renders a labelled native <select> for full keyboard support.
 */
export function PlanSelect({
  value,
  onChange,
  label,
  testId,
}: {
  value: string;
  onChange: (planId: string) => void;
  label: string;
  testId: string;
}) {
  const { t } = useTranslation();
  const plans = usePlansList();

  return (
    <label className="flex flex-col gap-1 text-sm">
      <span className="text-xs font-medium text-app-muted-foreground">{label}</span>
      <select
        data-testid={testId}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-10 rounded-control border border-app-border bg-app-surface px-3 text-app-foreground"
      >
        <option value="">{t(strings.common.selectPlaceholder)}</option>
        {(plans.data ?? []).map((plan) => (
          <option key={plan.id} value={plan.id}>
            {plan.title}
          </option>
        ))}
      </select>
    </label>
  );
}
