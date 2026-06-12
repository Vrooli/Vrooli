import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { SeverityBadge } from "../../components/SeverityBadge";
import { ErrorState } from "../../components/ErrorState";
import { EmptyState } from "../../components/EmptyState";
import {
  makeFileMoveOp,
  makeImportRewriteOp,
  OperationStatus,
  type Operation,
} from "../../api/rewrite";
import { useRewritePlan, useRewriteApply } from "./controllers/useRewrite";

type OpRow =
  | { readonly kind: "fileMove"; from: string; to: string }
  | { readonly kind: "importRewrite"; old: string; new: string };

let ROW_KEY = 0;
const newKey = () => `op-${++ROW_KEY}`;

export interface RewriteTabProps {
  projectPath: string;
}

function rowToOperation(row: OpRow): Operation | null {
  if (row.kind === "fileMove") {
    if (row.from.trim().length === 0 || row.to.trim().length === 0) return null;
    return makeFileMoveOp(row.from.trim(), row.to.trim());
  }
  if (row.old.trim().length === 0 || row.new.trim().length === 0) return null;
  return makeImportRewriteOp(row.old.trim(), row.new.trim());
}

/**
 * Rewrite workbench. Compose typed operation rows (FileMove / ImportRewrite),
 * preview a dry-run plan (green/red pseudo-diff), then apply behind a confirm
 * dialog. The producer's own guardrail (dry-run first, then explicit apply) is
 * surfaced directly in the UI.
 */
export function RewriteTab({ projectPath }: RewriteTabProps) {
  const { t } = useTranslation();
  const [rows, setRows] = React.useState<Array<OpRow & { key: string }>>([]);
  const [confirming, setConfirming] = React.useState(false);

  const plan = useRewritePlan();
  const apply = useRewriteApply();

  const operations = React.useMemo(
    () => rows.map(rowToOperation).filter((op): op is Operation => op !== null),
    [rows],
  );

  const addFileMove = () => {
    setRows((prev) => [...prev, { key: newKey(), kind: "fileMove", from: "", to: "" }]);
    plan.reset();
    apply.reset();
  };
  const addImportRewrite = () => {
    setRows((prev) => [...prev, { key: newKey(), kind: "importRewrite", old: "", new: "" }]);
    plan.reset();
    apply.reset();
  };
  const removeRow = (key: string) => {
    setRows((prev) => prev.filter((r) => r.key !== key));
    plan.reset();
    apply.reset();
  };
  const updateRow = (key: string, patch: Partial<OpRow>) => {
    setRows((prev) =>
      prev.map((r) => (r.key === key ? ({ ...r, ...patch } as OpRow & { key: string }) : r)),
    );
    plan.reset();
    apply.reset();
  };

  const runPlan = () => {
    apply.reset();
    setConfirming(false);
    plan.mutate({ projectPath, operations });
  };
  const runApply = () => {
    const planId = plan.data?.planId;
    if (planId === undefined) return;
    setConfirming(false);
    apply.mutate({ projectPath, planId });
  };

  const planId = plan.data?.planId;
  const canPlan = projectPath.length > 0 && operations.length > 0 && !plan.isPending;
  const canApply = planId !== undefined && planId.length > 0 && !apply.isPending;

  return (
    <div data-testid={selectors.features.rewrite.root} className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h3 className="text-lg font-semibold">{t(strings.rewrite.title)}</h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.rewrite.description)}</p>
      </div>

      {/* Operations editor */}
      <section
        data-testid={selectors.features.rewrite.opsEditor.root}
        className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
      >
        <div className="flex flex-wrap items-center gap-2">
          <h4 className="text-sm font-semibold">{t(strings.rewrite.ops.title)}</h4>
          <div className="ms-auto flex gap-2">
            <Button
              variant="outline"
              size="sm"
              data-testid={selectors.features.rewrite.opsEditor.addFileMove}
              onClick={addFileMove}
            >
              {t(strings.rewrite.ops.addFileMove)}
            </Button>
            <Button
              variant="outline"
              size="sm"
              data-testid={selectors.features.rewrite.opsEditor.addImportRewrite}
              onClick={addImportRewrite}
            >
              {t(strings.rewrite.ops.addImportRewrite)}
            </Button>
          </div>
        </div>

        {rows.length === 0 ? (
          <p
            data-testid={selectors.features.rewrite.opsEditor.empty}
            className="text-xs text-app-muted-foreground"
          >
            {t(strings.rewrite.ops.empty)}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {rows.map((row, index) => (
              <li
                key={row.key}
                data-testid={selectors.features.rewrite.opRow({ index })}
                className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-2 sm:flex-row sm:items-center"
              >
                <span className="w-32 shrink-0 text-xs font-medium uppercase text-app-muted-foreground">
                  {row.kind === "fileMove"
                    ? t(strings.rewrite.ops.fileMove)
                    : t(strings.rewrite.ops.importRewrite)}
                </span>
                {row.kind === "fileMove" ? (
                  <>
                    <Input
                      aria-label={t(strings.rewrite.ops.fromPath)}
                      placeholder={t(strings.rewrite.ops.fromPath)}
                      value={row.from}
                      onChange={(e) => updateRow(row.key, { from: e.target.value })}
                    />
                    <Input
                      aria-label={t(strings.rewrite.ops.toPath)}
                      placeholder={t(strings.rewrite.ops.toPath)}
                      value={row.to}
                      onChange={(e) => updateRow(row.key, { to: e.target.value })}
                    />
                  </>
                ) : (
                  <>
                    <Input
                      aria-label={t(strings.rewrite.ops.oldImport)}
                      placeholder={t(strings.rewrite.ops.oldImport)}
                      value={row.old}
                      onChange={(e) => updateRow(row.key, { old: e.target.value })}
                    />
                    <Input
                      aria-label={t(strings.rewrite.ops.newImport)}
                      placeholder={t(strings.rewrite.ops.newImport)}
                      value={row.new}
                      onChange={(e) => updateRow(row.key, { new: e.target.value })}
                    />
                  </>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  aria-label={t(strings.rewrite.ops.remove)}
                  onClick={() => removeRow(row.key)}
                >
                  ✕
                </Button>
              </li>
            ))}
          </ul>
        )}

        <div>
          <Button
            data-testid={selectors.features.rewrite.plan.button}
            disabled={!canPlan}
            onClick={runPlan}
          >
            {plan.isPending ? t(strings.rewrite.plan.planning) : t(strings.rewrite.plan.button)}
          </Button>
        </div>
      </section>

      {/* Plan preview */}
      <section className="flex flex-col gap-2">
        {plan.isError ? (
          <ErrorState
            title={t(strings.shared.error.title)}
            message={plan.error.message}
            retryLabel={t(strings.shared.error.retry)}
            onRetry={runPlan}
          />
        ) : plan.data ? (
          <div
            data-testid={selectors.features.rewrite.plan.result}
            className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
          >
            <div className="flex flex-wrap items-center gap-2">
              <h4 className="text-sm font-semibold">{t(strings.rewrite.plan.title)}</h4>
              <span className="text-xs text-app-muted-foreground">
                {t(strings.rewrite.plan.count, { count: plan.data.normalizedOperations.length })}
              </span>
              <span className="ms-auto font-mono text-xs text-app-muted-foreground">
                {t(strings.rewrite.plan.planIdLabel)} {plan.data.planId}
              </span>
            </div>
            <ul className="flex flex-col gap-1 font-mono text-xs">
              {plan.data.normalizedOperations.map((op, index) => {
                const removed =
                  op.op.case === "fileMove"
                    ? op.op.value.fromPath
                    : op.op.case === "importRewrite"
                      ? op.op.value.oldPath
                      : "";
                const added =
                  op.op.case === "fileMove"
                    ? op.op.value.toPath
                    : op.op.case === "importRewrite"
                      ? op.op.value.newPath
                      : "";
                return (
                  <li key={index} className="flex flex-col">
                    <span className="text-app-danger">- {removed}</span>
                    <span className="text-app-success">+ {added}</span>
                  </li>
                );
              })}
            </ul>
            <div>
              <Button
                data-testid={selectors.features.rewrite.apply.button}
                disabled={!canApply}
                onClick={() => setConfirming(true)}
              >
                {apply.isPending
                  ? t(strings.rewrite.apply.applying)
                  : t(strings.rewrite.apply.button)}
              </Button>
            </div>
          </div>
        ) : (
          <div data-testid={selectors.features.rewrite.plan.empty}>
            <EmptyState title={t(strings.rewrite.plan.empty)} />
          </div>
        )}
      </section>

      {/* Apply confirm dialog */}
      {confirming ? (
        <div
          data-testid={selectors.features.rewrite.apply.confirmDialog.root}
          role="alertdialog"
          aria-label={t(strings.rewrite.apply.confirmTitle)}
          className="flex flex-col gap-3 rounded-panel border border-app-danger/40 bg-app-danger/10 p-4"
        >
          <p className="text-sm font-semibold text-app-danger">
            {t(strings.rewrite.apply.confirmTitle)}
          </p>
          <p className="text-sm text-app-foreground">
            {t(strings.rewrite.apply.confirmMessage, { target: projectPath })}
          </p>
          <div className="flex gap-2">
            <Button
              data-testid={selectors.features.rewrite.apply.confirmDialog.confirm}
              onClick={runApply}
            >
              {t(strings.rewrite.apply.confirm)}
            </Button>
            <Button
              variant="outline"
              data-testid={selectors.features.rewrite.apply.confirmDialog.cancel}
              onClick={() => setConfirming(false)}
            >
              {t(strings.rewrite.apply.cancel)}
            </Button>
          </div>
        </div>
      ) : null}

      {/* Apply results */}
      {apply.isError ? (
        <ErrorState
          title={t(strings.shared.error.title)}
          message={apply.error.message}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => setConfirming(true)}
        />
      ) : apply.data ? (
        <div
          data-testid={selectors.features.rewrite.apply.result}
          className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
        >
          <div className="flex items-center gap-2">
            <h4 className="text-sm font-semibold">{t(strings.rewrite.apply.resultTitle)}</h4>
            {apply.data.dryRun ? (
              <span className="text-xs text-app-muted-foreground">
                {t(strings.rewrite.apply.dryRunNote)}
              </span>
            ) : null}
          </div>
          <ul className="flex flex-col gap-1">
            {apply.data.results.map((result, index) => {
              const ok = result.status === OperationStatus.OK;
              return (
                <li
                  key={index}
                  data-testid={selectors.features.rewrite.opResult({ index })}
                  className="flex items-center gap-2 text-sm"
                >
                  <SeverityBadge
                    level={ok ? "info" : "high"}
                    label={
                      ok ? t(strings.rewrite.apply.status.ok) : t(strings.rewrite.apply.status.failed)
                    }
                  />
                  <span className={cn("text-xs", ok ? "text-app-muted-foreground" : "text-app-danger")}>
                    {result.message}
                  </span>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
