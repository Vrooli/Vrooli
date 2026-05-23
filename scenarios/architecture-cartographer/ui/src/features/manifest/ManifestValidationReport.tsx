import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { Badge } from "../../components/ui/badge";
import {
  DiagnosticSeverity,
  type Diagnostic,
} from "@vrooli/proto-types/architecture-cartographer/v1/manifest/manifest_pb";

function severityLabelKey(sev: DiagnosticSeverity) {
  switch (sev) {
    case DiagnosticSeverity.WARN:
      return strings.manifest.severity.warn;
    case DiagnosticSeverity.ERROR:
      return strings.manifest.severity.error;
    default:
      return strings.manifest.severity.info;
  }
}

const SEVERITY_VARIANT: Record<DiagnosticSeverity, "info" | "warning" | "danger" | "default"> = {
  [DiagnosticSeverity.UNSPECIFIED]: "default",
  [DiagnosticSeverity.INFO]: "info",
  [DiagnosticSeverity.WARN]: "warning",
  [DiagnosticSeverity.ERROR]: "danger",
};

export interface ManifestValidationReportProps {
  diagnostics: readonly Diagnostic[];
  valid: boolean;
}

export function ManifestValidationReport({ diagnostics, valid }: ManifestValidationReportProps) {
  const { t } = useTranslation();

  if (diagnostics.length === 0) {
    return (
      <div data-testid={selectors.features.manifest.validation.root} className="flex flex-col gap-2">
        <div data-testid={selectors.features.manifest.validation.validBanner}>
          <Badge variant="success">{t(strings.pages.targetManifest.validHeading)}</Badge>
        </div>
        <EmptyState title={t(strings.pages.targetManifest.noDiagnostics)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<Diagnostic & { _index: number }>> = [
    {
      key: "severity",
      header: t(strings.pages.targetManifest.columns.severity),
      cell: (row) => (
        <Badge variant={SEVERITY_VARIANT[row.severity]}>
          {t(severityLabelKey(row.severity))}
        </Badge>
      ),
    },
    {
      key: "code",
      header: t(strings.pages.targetManifest.columns.code),
      cell: (row) => <span className="font-mono text-xs">{row.code || "—"}</span>,
    },
    {
      key: "path",
      header: t(strings.pages.targetManifest.columns.path),
      cell: (row) => <span className="font-mono text-xs">{row.path || "—"}</span>,
    },
    {
      key: "location",
      header: t(strings.pages.targetManifest.columns.location),
      cell: (row) => (
        <span className="font-mono text-xs">
          {row.line || row.column ? `${row.line}:${row.column}` : "—"}
        </span>
      ),
    },
    {
      key: "message",
      header: t(strings.pages.targetManifest.columns.message),
      cell: (row) => <span className="text-sm">{row.message}</span>,
    },
  ];

  const rows = diagnostics.map((d, i) => Object.assign({}, d, { _index: i }));

  return (
    <div data-testid={selectors.features.manifest.validation.root} className="flex flex-col gap-2">
      <div data-testid={selectors.features.manifest.validation.invalidBanner}>
        <Badge variant={valid ? "warning" : "danger"}>
          {t(valid ? strings.pages.targetManifest.validHeading : strings.pages.targetManifest.invalidHeading)}
        </Badge>
      </div>
      <DataTable
        rows={rows}
        getRowId={(d) => `${d.code}-${d._index}`}
        columns={columns}
        emptyMessage={t(strings.pages.targetManifest.noDiagnostics)}
      />
    </div>
  );
}
