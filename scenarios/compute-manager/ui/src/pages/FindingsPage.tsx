import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.4.2";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useEffect, useState } from "react";

import { fetchOpenFindings } from "../api/compute";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";

type Finding = Awaited<ReturnType<typeof fetchOpenFindings>>["findings"][number];

export function FindingsPage() {
  const { t } = useTranslation();
  const [rows, setRows] = useState<Finding[]>([]);
  const [status, setStatus] = useState<"loading" | "success" | "request-error">("loading");
  useEffect(() => {
    fetchOpenFindings().then((response) => { setRows(response.findings); setStatus("success"); }).catch(() => setStatus("request-error"));
  }, []);
  const columns: DataTableColumn<Finding>[] = [
    { id: "kind", header: t(strings.pages.findings.kind), accessor: (row) => row.kind },
    { id: "provider", header: t(strings.pages.findings.providerInstance), accessor: (row) => row.providerInstanceId },
    { id: "detail", header: t(strings.pages.findings.detail), accessor: (row) => row.detail },
  ];
  return <section data-testid={selectors.pages.findings} className="flex flex-col gap-space-md" aria-labelledby="findings-heading"><PageHeader headingId="findings-heading" title={t(strings.pages.findings.title)} description={t(strings.pages.findings.description)} /><DataTable rows={rows} columns={columns} getRowKey={(row) => row.id} caption={t(strings.pages.findings.title)} status={status} errorMessage={t(strings.pages.findings.error)} emptyMessage={t(strings.pages.findings.empty)} /></section>;
}
