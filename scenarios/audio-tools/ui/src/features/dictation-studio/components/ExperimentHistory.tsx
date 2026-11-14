import { useState } from "react";

import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { LoadingRows } from "../../../components/composites/LoadingRows";
import { Button } from "../../../components/ui/button";
import { Table, TBody, TD, TH, THead, TR } from "../../../components/ui/table";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { type ExperimentRow } from "../../../services/experiment";
import { isTerminal } from "../ExperimentLabFormat";
import { StatusBadge } from "../ExperimentLabShared";

export function ExperimentHistory({
  rows,
  pending,
  error,
  selectedId,
  compareSelected,
  onToggleCompare,
  onSelect,
  onWait,
  onCancel,
  onReport,
  onRetry,
  actionPending,
}: {
  rows: ExperimentRow[];
  pending: boolean;
  error: Error | null;
  selectedId: string;
  compareSelected: string[];
  onToggleCompare: (id: string) => void;
  onSelect: (id: string) => void;
  onWait: (id: string) => void;
  onCancel: (id: string) => void;
  onReport: (id: string) => void;
  onRetry: () => void;
  actionPending: boolean;
}) {
  const { t } = useTranslation();
  const [confirmCancelId, setConfirmCancelId] = useState("");
  if (pending) return <LoadingRows rows={3} label={t(strings.dictationStudio.historyTitle)} />;
  if (error) return <ApiErrorState error={error} title={t(strings.dictationStudio.historyError)} onRetry={onRetry} />;
  if (rows.length === 0) return <p className="text-sm text-app-muted-foreground">{t(strings.dictationStudio.historyEmpty)}</p>;

  const compareSet = new Set(compareSelected);

  return (
    <div className="overflow-x-auto">
      <Table>
        <THead>
          <TR>
            <TH>{t(strings.dictationStudio.compareSelectHeader)}</TH>
            <TH>{t(strings.dictationStudio.colName)}</TH>
            <TH>{t(strings.dictationStudio.colStatus)}</TH>
            <TH>{t(strings.dictationStudio.colRecipe)}</TH>
            <TH>{t(strings.dictationStudio.colActions)}</TH>
          </TR>
        </THead>
        <TBody>
          {rows.map((row) => (
            <TR
              key={row.id}
              data-testid={selectors.dictationStudio.experimentRow({ id: row.id })}
              className={row.id === selectedId ? "bg-app-surface-muted" : undefined}
            >
              <TD>
                <input
                  type="checkbox"
                  aria-label={t(strings.dictationStudio.compareSelectHeader)}
                  data-testid={selectors.dictationStudio.experimentCompare({ id: row.id })}
                  checked={compareSet.has(row.id)}
                  onChange={() => onToggleCompare(row.id)}
                />
              </TD>
              <TD>
                <button type="button" className="text-left font-medium text-app-foreground underline-offset-2 hover:underline" onClick={() => onSelect(row.id)}>
                  {row.name || row.id}
                </button>
                <div className="text-xs text-app-muted-foreground">{row.id}</div>
              </TD>
              <TD><StatusBadge status={row.status} /></TD>
              <TD className="text-xs text-app-muted-foreground">
                {row.recipe.strategies.join(", ") || t(strings.dictationStudio.recipeDefault)} · {row.recipe.longFormEnabled ? `${row.recipe.targetDurationSeconds}s` : t(strings.dictationStudio.recipeClips)}
              </TD>
              <TD>
                <div className="flex flex-wrap gap-1">
                  {!isTerminal(row.status) ? (
                    confirmCancelId === row.id ? (
                      <>
                        <span className="self-center text-xs text-app-muted-foreground">{t(strings.dictationStudio.cancelConfirmPrompt)}</span>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          data-testid={selectors.dictationStudio.experimentCancelConfirm({ id: row.id })}
                          disabled={actionPending}
                          onClick={() => {
                            onCancel(row.id);
                            setConfirmCancelId("");
                          }}
                        >
                          {t(strings.dictationStudio.cancelConfirm)}
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          data-testid={selectors.dictationStudio.experimentCancelDismiss({ id: row.id })}
                          onClick={() => setConfirmCancelId("")}
                        >
                          {t(strings.dictationStudio.cancelDismiss)}
                        </Button>
                      </>
                    ) : (
                      <>
                        <Button type="button" size="sm" variant="outline" data-testid={selectors.dictationStudio.experimentWait({ id: row.id })} disabled={actionPending} onClick={() => onWait(row.id)}>
                          {t(strings.dictationStudio.wait)}
                        </Button>
                        <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.experimentCancel({ id: row.id })} disabled={actionPending} onClick={() => setConfirmCancelId(row.id)}>
                          {t(strings.dictationStudio.cancel)}
                        </Button>
                      </>
                    )
                  ) : null}
                  <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.experimentReport({ id: row.id })} disabled={actionPending} onClick={() => onReport(row.id)}>
                    {t(strings.dictationStudio.report)}
                  </Button>
                </div>
              </TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </div>
  );
}
