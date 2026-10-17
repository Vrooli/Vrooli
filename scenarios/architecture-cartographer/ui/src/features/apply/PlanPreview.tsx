import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { OperationKind, type Operation, type Plan } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

function kindLabelKey(kind: OperationKind) {
  switch (kind) {
    case OperationKind.MOVE_FILE:
      return strings.apply.operationKind.move_file;
    case OperationKind.REWRITE_IMPORT:
      return strings.apply.operationKind.rewrite_import;
    case OperationKind.DELETE_FILE:
      return strings.apply.operationKind.delete_file;
    case OperationKind.CREATE_FILE:
      return strings.apply.operationKind.create_file;
    default:
      return strings.apply.operationKind.unspecified;
  }
}

export interface PlanPreviewProps {
  plan: Plan | undefined;
}

export function PlanPreview({ plan }: PlanPreviewProps) {
  const { t } = useTranslation();

  if (!plan) {
    return (
      <div data-testid={selectors.features.apply.plan.empty}>
        <EmptyState title={t(strings.pages.targetApply.noPlan)} />
      </div>
    );
  }

  if (plan.operations.length === 0) {
    return (
      <div data-testid={selectors.features.apply.plan.empty}>
        <EmptyState title={t(strings.pages.targetApply.noOperations)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<Operation>> = [
    {
      key: "kind",
      header: t(strings.pages.targetApply.operationsHeading),
      cell: (row) => <span className="text-sm">{t(kindLabelKey(row.kind))}</span>,
    },
    {
      key: "from",
      header: "From",
      cell: (row) => <span className="font-mono text-xs">{row.fromPath || "—"}</span>,
    },
    {
      key: "to",
      header: "To",
      cell: (row) => <span className="font-mono text-xs">{row.toPath || "—"}</span>,
    },
  ];

  return (
    <div data-testid={selectors.features.apply.plan.root}>
      <DataTable
        rows={plan.operations}
        getRowId={(op) => op.id || `${op.kind}-${op.fromPath}-${op.toPath}`}
        columns={columns}
        emptyMessage={t(strings.pages.targetApply.noOperations)}
      />
    </div>
  );
}
