import { useTranslation } from "../../i18n";
import { VERDICT_ICON } from "../verdicts";
import type { ContextCheckGroup, ContextCheckRow } from "./contextChecks";
import type { Verdict } from "../verdicts";
import "./incontext.css";

const VERDICT_LABEL: Record<Verdict, string> = {
  pass: "PASS",
  fail: "FAIL",
  warn: "WARN",
  unmeasured: "N/M",
};

function CheckRow({ row, onRunEvidence }: { row: ContextCheckRow; onRunEvidence?: () => void }) {
  const { t } = useTranslation();
  const Icon = VERDICT_ICON[row.verdict];
  const runnable = row.group === "evidence" && onRunEvidence;
  return (
    <div className={`ic-check is-${row.verdict}`}>
      <span className="ic-check-icon" aria-hidden>
        <Icon size={12} />
      </span>
      <div className="ic-check-text">
        <b>{row.label}</b>
        <span>{row.detail}</span>
        {runnable ? (
          <button type="button" className="ic-check-action" onClick={onRunEvidence}>
            {t("workbench.component.ctxRunEvidence", { defaultValue: "Run evidence →" })}
          </button>
        ) : null}
      </div>
      <span className="ic-check-verdict">{VERDICT_LABEL[row.verdict]}</span>
    </div>
  );
}

export interface ChecksPanelProps {
  rows: ContextCheckRow[];
  /** Fixtures has fit rows plus footnotes; In context adds the secondary disclosure. */
  variant?: "context" | "fixture";
  evidenceNote?: string;
  templateHref?: string;
  templateName?: string;
  onRunEvidence?: () => void;
  onOpenContext?: () => void;
}

/**
 * One check panel for both tabs.
 *
 * The component fit is the lead. Page, evidence and human rows are one
 * disclosure away in context; fixtures shows the same fit rows with footnotes
 * instead of a second full panel.
 */
export function InContextChecksPanel({
  rows,
  variant = "context",
  evidenceNote,
  templateHref,
  templateName,
  onRunEvidence,
  onOpenContext,
}: ChecksPanelProps) {
  const { t } = useTranslation();

  if (rows.length === 0) {
    return (
      <p className="ic-checks-empty" data-testid="checks-empty">
        {t("workbench.component.ctxChecksEmpty", {
          defaultValue: "Measurements appear once the frame has rendered.",
        })}
      </p>
    );
  }

  const groupLabel: Record<Exclude<ContextCheckGroup, "info">, string> = {
    fit: t("workbench.component.ctxGroupComponent", { defaultValue: "Component" }),
    page: t("workbench.component.ctxGroupPageShort", { defaultValue: "Page" }),
    evidence: t("workbench.component.ctxGroupEvidenceShort", { defaultValue: "Evidence" }),
    human: t("workbench.component.ctxGroupHumanShort", { defaultValue: "Human" }),
  };
  const fitRows = rows.filter((row) => (row.group ?? "fit") === "fit");
  const infoRows = rows.filter((row) => row.group === "info");
  const pageRows = rows.filter((row) => row.group === "page");
  const evidenceRows = rows.filter((row) => row.group === "evidence");
  const humanRows = rows.filter((row) => row.group === "human");
  const secondaryIssues = [...pageRows, ...evidenceRows].filter(
    (row) => row.verdict === "fail" || row.verdict === "warn",
  ).length;

  return (
    <div className="ic-checks" data-testid="checks-panel">
      {fitRows.length > 0 ? (
        <section className="ic-check-group" aria-label={groupLabel.fit}>
          <h3>{groupLabel.fit}</h3>
          {fitRows.map((row) => (
            <CheckRow key={row.id} row={row} />
          ))}
        </section>
      ) : null}

      {infoRows.map((row) => (
        <p key={row.id} className="ic-type-note">
          {row.detail}
        </p>
      ))}

      {variant === "context" ? (
        <details className="ic-secondary">
          <summary>
            <span className="ic-secondary-label">
              {t("workbench.component.ctxSecondary", {
                defaultValue: "Page & evidence",
              })}
              {secondaryIssues > 0
                ? ` · ${t("workbench.component.ctxSecondaryIssues", {
                    defaultValue: "{{count}} issue",
                    count: secondaryIssues,
                  })}`
                : ""}
            </span>
          </summary>
          {pageRows.length > 0 ? (
            <section className="ic-check-group" aria-label={groupLabel.page}>
              <h3>{groupLabel.page}</h3>
              {pageRows.map((row) => (
                <CheckRow key={row.id} row={row} />
              ))}
              {templateHref ? (
                <a className="ic-check-action" href={templateHref}>
                  {t("workbench.component.ctxOpenTemplate", {
                    defaultValue: "Open {{name}} template →",
                    name:
                      templateName ||
                      t("workbench.component.ctxTemplate", { defaultValue: "page" }),
                  })}
                </a>
              ) : null}
            </section>
          ) : null}
          {evidenceRows.length > 0 ? (
            <section className="ic-check-group" aria-label={groupLabel.evidence}>
              <h3>{groupLabel.evidence}</h3>
              {evidenceRows.map((row) => (
                <CheckRow key={row.id} row={row} onRunEvidence={onRunEvidence} />
              ))}
            </section>
          ) : null}
          {humanRows.length > 0 ? (
            <section className="ic-check-group" aria-label={groupLabel.human}>
              <h3>{groupLabel.human}</h3>
              {humanRows.map((row) => (
                <CheckRow key={row.id} row={row} />
              ))}
            </section>
          ) : null}
        </details>
      ) : (
        <>
          <p className="ic-note">
            {t("workbench.component.ctxEvidenceNote", {
              defaultValue: "Evidence: {{note}}",
              note:
                evidenceNote ||
                t("workbench.component.ctxEvidenceUnavailable", {
                  defaultValue: "unavailable",
                }),
            })}{" "}
            {onRunEvidence ? (
              <>
                <span className="ic-inline-sep" aria-hidden>
                  ·
                </span>
                <button type="button" className="ic-inline-action" onClick={onRunEvidence}>
                  {t("workbench.component.ctxRetry", { defaultValue: "Retry →" })}
                </button>
              </>
            ) : null}
          </p>
          <p className="ic-note">
            {t("workbench.component.ctxFixturePageNote", {
              defaultValue: "Page health is not checked here.",
            })}{" "}
            {onOpenContext ? (
              <button type="button" className="ic-inline-action" onClick={onOpenContext}>
                {t("workbench.component.ctxOpenContext", {
                  defaultValue: "Open in context →",
                })}
              </button>
            ) : null}
          </p>
        </>
      )}

      <p className="ic-checks-note" data-testid="checks-note">
        {t("workbench.component.ctxChecksNoteShort", { defaultValue: "N/M is not a pass." })}
      </p>
    </div>
  );
}
