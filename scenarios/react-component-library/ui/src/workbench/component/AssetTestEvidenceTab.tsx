import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleAlert, Play, RotateCw } from "lucide-react";

import {
  getComponentTestReport,
  runComponentTest,
  type ComponentTestReport,
} from "../../api/componentTests";
import { useTranslation } from "../../i18n";
import { formatRelative, timestampToDate, wbKeys, type ComponentBundle } from "../data";
import { Banner, Chip, Empty, WbButton } from "../primitives";

export function AssetTestEvidenceTab({
  bundle,
  version,
  selectedReportID,
  onSelectReport,
}: {
  bundle: ComponentBundle;
  version: string;
  selectedReportID: string | null;
  onSelectReport?: (reportID: string | null) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const selectedInHistory = bundle.testReports.find((report) => report.id === selectedReportID);
  const selectedQuery = useQuery({
    queryKey: ["wb", "component-test-report", selectedReportID],
    queryFn: () => getComponentTestReport(selectedReportID as string),
    enabled: Boolean(selectedReportID) && !selectedInHistory,
    staleTime: 60_000,
    retry: 0,
  });
  const selectedReport = selectedInHistory ?? selectedQuery.data;
  const run = useMutation({
    mutationFn: () =>
      runComponentTest({ componentId: bundle.asset.id, version, includeClosure: false }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(bundle.asset.id) });
    },
  });
  const rerun = useMutation({
    mutationFn: (report: ComponentTestReport) =>
      runComponentTest({
        componentId: report.rootLibraryId,
        version: report.rootVersion,
        includeClosure: report.includeClosure,
      }),
    onSuccess: (report) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.component(bundle.asset.id) });
      onSelectReport?.(report.id);
    },
  });

  return (
    <div className="wb-stack">
      <div className="wb-row-between">
        <span className="wb-caption">
          {t("workbench.component.testReports", { defaultValue: "Component test evidence" })}
        </span>
        <WbButton variant="ghost" size="sm" onClick={() => run.mutate()} disabled={run.isPending}>
          {run.isPending ? <RotateCw aria-hidden size={13} /> : <Play aria-hidden size={13} />}
          {run.isPending
            ? t("workbench.component.running", { defaultValue: "Running…" })
            : t("workbench.component.runTest", { defaultValue: "Run test" })}
        </WbButton>
      </div>
      {run.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {run.error instanceof Error ? run.error.message : String(run.error)}
        </Banner>
      ) : null}
      {rerun.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
          {rerun.error instanceof Error ? rerun.error.message : String(rerun.error)}
        </Banner>
      ) : null}
      {selectedReportID && selectedQuery.isError ? (
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
          {t("workbench.component.reportUnavailable", {
            defaultValue: "This report could not be loaded. Its ID is preserved in the URL.",
          })}
        </Banner>
      ) : null}
      {selectedReport ? (
        <section
          className="wb-panel"
          aria-label={t("workbench.component.selectedReport", {
            defaultValue: "Selected test report",
          })}
          style={{ padding: 12 }}
        >
          <div className="wb-row-between">
            <b className="wb-mono wb-small">{selectedReport.id}</b>
            <Chip
              tone={
                selectedReport.verdict === "passed"
                  ? "green"
                  : selectedReport.verdict === "failed"
                    ? "red"
                    : "amber"
              }
            >
              {selectedReport.verdict}
            </Chip>
          </div>
          <p className="wb-small wb-muted-2">
            {selectedReport.rootLibraryId}@{selectedReport.rootVersion}
            {selectedReport.includeClosure ? " · dependency closure" : " · focused component"}
          </p>
          <WbButton
            variant="ghost"
            size="sm"
            aria-label={`${t("workbench.component.rerunReport", { defaultValue: "Rerun this report" })} ${selectedReport.id}`}
            onClick={() => rerun.mutate(selectedReport)}
            disabled={rerun.isPending}
          >
            {rerun.isPending ? <RotateCw aria-hidden size={13} /> : <Play aria-hidden size={13} />}
            {rerun.isPending
              ? t("workbench.component.rerunningReport", { defaultValue: "Rerunning report…" })
              : t("workbench.component.rerunReport", { defaultValue: "Rerun this report" })}
          </WbButton>
          <div className="wb-stack tight" style={{ marginTop: 10 }}>
            {selectedReport.results.map((result, index) => (
              <div
                key={`${result.stage}:${result.subject}:${index}`}
                className="wb-panel"
                style={{ padding: 10 }}
              >
                <div className="wb-row-between">
                  <b>
                    {result.stage} · {result.subject}
                  </b>
                  <Chip
                    tone={
                      result.verdict === "passed"
                        ? "green"
                        : result.verdict === "failed"
                          ? "red"
                          : "amber"
                    }
                  >
                    {result.verdict}
                  </Chip>
                </div>
                {result.message ? <p className="wb-small">{result.message}</p> : null}
                {result.remediation ? (
                  <p className="wb-small">
                    <b>{t("workbench.component.remediation", { defaultValue: "Remediation" })}:</b>{" "}
                    {result.remediation}
                  </p>
                ) : null}
              </div>
            ))}
          </div>
        </section>
      ) : null}
      {bundle.testReports.length === 0 && !selectedReport ? (
        <Empty
          title={t("workbench.component.noTests", {
            defaultValue: "No component test evidence yet",
          })}
        >
          {t("workbench.component.noTestsBody", {
            defaultValue:
              "A release without a current test report is not proven. Run the focused component test to produce one.",
          })}
        </Empty>
      ) : bundle.testReports.length > 0 ? (
        bundle.testReports.map((report) => (
          <div key={report.id} className="wb-panel" style={{ padding: 12 }}>
            <div className="wb-row-between">
              <span className="wb-mono wb-small">{report.id.slice(0, 12)}</span>
              <Chip
                tone={
                  report.verdict === "passed"
                    ? "green"
                    : report.verdict === "failed"
                      ? "red"
                      : "amber"
                }
              >
                {report.verdict}
              </Chip>
              <span className="wb-small wb-muted-2">
                {formatRelative(timestampToDate(report.createdAt))}
              </span>
            </div>
            <div className="wb-row" style={{ marginTop: 8 }}>
              <WbButton
                variant="ghost"
                size="sm"
                aria-label={`${t("workbench.component.viewReport", { defaultValue: "View report" })} ${report.id}`}
                onClick={() => onSelectReport?.(report.id)}
              >
                {t("workbench.component.viewReport", { defaultValue: "View report" })}
              </WbButton>
              <WbButton
                variant="ghost"
                size="sm"
                aria-label={`${t("workbench.component.rerunReport", { defaultValue: "Rerun this report" })} ${report.id}`}
                onClick={() => rerun.mutate(report)}
                disabled={rerun.isPending}
              >
                {t("workbench.component.rerunReport", { defaultValue: "Rerun this report" })}
              </WbButton>
            </div>
            <div className="wb-stack tight" style={{ marginTop: 8 }}>
              {report.results.slice(0, 8).map((result) => (
                <div key={`${result.stage}:${result.subject}`} className="wb-checkrow">
                  <span
                    style={{
                      color: result.verdict === "passed" ? "var(--wb-green)" : "var(--wb-red)",
                    }}
                  >
                    ●
                  </span>
                  <span>{result.stage}</span>
                  <b>{result.verdict}</b>
                </div>
              ))}
            </div>
          </div>
        ))
      ) : null}
    </div>
  );
}
