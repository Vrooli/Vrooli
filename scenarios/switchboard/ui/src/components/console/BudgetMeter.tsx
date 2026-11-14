import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";

import type { ThreadBudget } from "../../api/console";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatNumber } from "../../i18n/format";

export function budgetPressure(budget: ThreadBudget): { ratio: number; tone: "neutral" | "success" | "warning" | "danger" } {
  const turnRatio = budget.turn_budget > 0 ? budget.used / budget.turn_budget : 0;
  const spendRatio = budget.spend_cap_cents > 0 ? budget.spent_cents / budget.spend_cap_cents : 0;
  const ratio = Math.min(1, Math.max(turnRatio, spendRatio));
  if (budget.exhausted || ratio >= 1) return { ratio: 1, tone: "danger" };
  if (ratio >= 0.7) return { ratio, tone: "warning" };
  return { ratio, tone: ratio === 0 ? "neutral" : "success" };
}

interface BudgetMeterProps {
  budget: ThreadBudget;
  /** Compact omits the description line. */
  compact?: boolean;
  testId?: string;
  className?: string;
}

/** Turn budget and spend cap for one thread, as one meter with a stated remainder. */
export function BudgetMeter({ budget, compact, testId, className }: BudgetMeterProps) {
  const { t } = useTranslation();
  const { tone } = budgetPressure(budget);
  const unlimited = budget.turn_budget <= 0;
  const remaining = Math.max(0, budget.turn_budget - budget.used);
  const spend =
    budget.spend_cap_cents > 0
      ? t(strings.console.budget.spend, {
          spent: formatNumber(budget.spent_cents / 100, { style: "currency", currency: "USD" }),
          cap: formatNumber(budget.spend_cap_cents / 100, { style: "currency", currency: "USD" }),
        })
      : t(strings.console.budget.noSpendCap);
  return (
    <BoundedMeter
      testId={testId}
      className={className}
      label={t(strings.console.budget.turnsThisHour)}
      ariaLabel={t(strings.console.budget.turnsThisHour)}
      value={unlimited ? 0 : budget.used}
      min={0}
      max={unlimited ? 1 : budget.turn_budget}
      tone={tone}
      valueText={unlimited ? t(strings.console.budget.unlimited) : t(strings.console.budget.remaining, { remaining, total: budget.turn_budget })}
      status={budget.exhausted ? t(strings.console.budget.exhausted) : undefined}
      description={compact ? undefined : spend}
    />
  );
}
