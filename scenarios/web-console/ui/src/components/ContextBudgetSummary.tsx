import { useTranslation } from "react-i18next";
import { contextLimitRecord, type CompactCandidate } from "../api/usageMode";
import { strings } from "../consts/strings";

function kTokens(value: number | undefined): number {
  return Math.round((value ?? 0) / 1000);
}

export function ContextBudgetSummary({ row }: { row: CompactCandidate }) {
  const { t } = useTranslation();
  const budget = row.contextBudget;
  if (!budget) return <p className="text-xs text-wc-text-muted">{row.contextTokens > 0 ? t(strings.contextCompaction.tokens, { count: kTokens(row.contextTokens) }) : t(strings.contextCompaction.unknownSize)}</p>;
  const limitRecord = contextLimitRecord({
    model: budget.model,
    modelSource: budget.modelSource,
    contextWindowTokens: budget.modelWindowTokens,
    contextWindowSource: budget.modelWindowSource,
    contextLimitTokens: budget.compactionPointTokens,
    contextLimitSource: budget.compactionPointSource,
  });
  const shown = (tokens: number | null) => tokens !== null ? `${kTokens(tokens)}K` : t(strings.launcher.contextWindowUnknown);
  return (
    <div className="min-w-0 space-y-0.5 break-words text-xs text-wc-text-muted" data-testid="context-budget" data-forecast={budget.forecastState}
      data-window-tokens={limitRecord.modelWindowTokens ?? "unknown"} data-window-source={limitRecord.modelWindowSource}
      data-compaction-tokens={limitRecord.compactionPointTokens ?? "unknown"} data-compaction-source={limitRecord.compactionPointSource}>
      {limitRecord.modelWindowTokens !== null && budget.progressPercent !== null ? <div role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.min(100, budget.progressPercent)} aria-label={`${budget.model ?? row.harness} context window usage`} className="h-1.5 overflow-hidden rounded bg-wc-default">
        <div className="h-full rounded bg-wc-accent" style={{ width: `${Math.min(100, budget.progressPercent)}%` }} />
      </div> : null}
      <p>{limitRecord.model ? `${limitRecord.model}${limitRecord.modelSource ? ` · ${limitRecord.modelSource}` : ""} · ` : ""}{budget.usedTokens !== null ? `${t(strings.launcher.contextBudgetUsed)} ${shown(budget.usedTokens)} · ` : ""}{shown(limitRecord.modelWindowTokens)} {limitRecord.modelWindowSource !== "unknown" ? `· ${limitRecord.modelWindowSource}` : ""}</p>
      <p>{limitRecord.compactionPointTokens !== null ? `${t(strings.launcher.contextBudgetCompactsAt, { count: kTokens(limitRecord.compactionPointTokens) })}K` : t(strings.launcher.contextBudgetCompactionUnknown)}
        {budget.tokensUntilCompaction !== null ? ` · ${shown(budget.tokensUntilCompaction)} ${t(strings.launcher.contextBudgetRemaining)}` : ""}</p>
      {budget.forecastState === "projected" && budget.secondsUntilCompaction != null ? <p data-testid="context-budget-forecast">{t(strings.launcher.contextBudgetForecast, { minutes: Math.max(1, Math.ceil(budget.secondsUntilCompaction / 60)) })}</p> : null}
    </div>
  );
}

export default ContextBudgetSummary;
