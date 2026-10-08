import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DiffView, type DiffLine } from "../../components/DiffView";
import { EmptyState } from "../../components/EmptyState";
import { OperationKind, type Plan } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

/**
 * Derive a deterministic pseudo-diff from a Plan's operations. v0.1's
 * RunApply is unimplemented, so we don't have real per-file diffs; what we
 * can show is the operation list rendered as a green/red mock-diff so the
 * surface shape is in place when the backend lands.
 */
function planToDiffLines(plan: Plan): readonly DiffLine[] {
  const lines: DiffLine[] = [];
  for (const op of plan.operations) {
    if (op.kind === OperationKind.MOVE_FILE) {
      if (op.fromPath) lines.push({ kind: "removed", text: op.fromPath });
      if (op.toPath) lines.push({ kind: "added", text: op.toPath });
    } else if (op.kind === OperationKind.CREATE_FILE) {
      if (op.toPath) lines.push({ kind: "added", text: op.toPath });
    } else if (op.kind === OperationKind.DELETE_FILE) {
      if (op.fromPath) lines.push({ kind: "removed", text: op.fromPath });
    } else if (op.kind === OperationKind.REWRITE_IMPORT) {
      const target = op.toPath || op.fromPath;
      if (target) lines.push({ kind: "context", text: `rewrite imports in ${target}` });
    }
  }
  return lines;
}

export interface DryRunDiffProps {
  plan: Plan | undefined;
}

export function DryRunDiff({ plan }: DryRunDiffProps) {
  const { t } = useTranslation();
  if (!plan || plan.operations.length === 0) {
    return (
      <div data-testid={selectors.features.apply.dryRun.empty}>
        <EmptyState title={t(strings.pages.targetApply.noOperations)} />
      </div>
    );
  }
  const lines = planToDiffLines(plan);
  return (
    <div data-testid={selectors.features.apply.dryRun.root}>
      <DiffView
        lines={lines}
        addedLabel={t(strings.shared.diff.added)}
        removedLabel={t(strings.shared.diff.removed)}
      />
    </div>
  );
}
