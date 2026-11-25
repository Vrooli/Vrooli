import { useTranslation } from "../../i18n";
import { VERDICT_ICON } from "../verdicts";
import type { ContextCheckGroup, ContextCheckRow, ContextVerdict } from "./contextChecks";
import "./incontext.css";

/**
 * The same four verdicts the fixtures panel uses. "unmeasured" is never a
 * pass; the abbreviation keeps the narrow inspector column readable.
 */
const VERDICT_LABEL: Record<ContextVerdict, string> = {
  pass: "PASS",
  fail: "FAIL",
  warn: "WARN",
  unmeasured: "N/M",
};

/**
 * The inspector's checks panel. Every row carries its evidence sentence, so a
 * bare verdict can never be read as a measurement it does not have.
 */
export function InContextChecksPanel({ rows }: { rows: ContextCheckRow[] }) {
  const { t } = useTranslation();

  if (rows.length === 0) {
    return (
      <p className="ic-checks-empty">
        {t("workbench.component.ctxChecksEmpty", {
          defaultValue: "Measurements appear once the frame has rendered.",
        })}
      </p>
    );
  }

  const groupOrder: ContextCheckGroup[] = ["fit", "page", "evidence", "human"];
  const groupLabel: Record<ContextCheckGroup, string> = {
    fit: t("workbench.component.ctxGroupFit", { defaultValue: "Component fit" }),
    page: t("workbench.component.ctxGroupPage", { defaultValue: "Page health" }),
    evidence: t("workbench.component.ctxGroupEvidence", { defaultValue: "State evidence" }),
    human: t("workbench.component.ctxGroupHuman", { defaultValue: "Human check" }),
  };

  return (
    <div className="ic-checks">
      {groupOrder.map((group) => {
        const groupRows = rows.filter((row) => (row.group ?? "fit") === group);
        if (groupRows.length === 0) return null;
        return (
          <section key={group} className="ic-check-group" aria-label={groupLabel[group]}>
            <h3>{groupLabel[group]}</h3>
            {groupRows.map((row) => {
              const Icon = VERDICT_ICON[row.verdict];
              return (
                <div key={row.id} className={`ic-check is-${row.verdict}`}>
                  <span className="ic-check-icon" aria-hidden>
                    <Icon size={12} />
                  </span>
                  <div className="ic-check-text">
                    <b>{row.label}</b>
                    <span>{row.detail}</span>
                  </div>
                  <span className="ic-check-verdict">
                    {VERDICT_LABEL[row.verdict]}
                    {row.evidence === "measured" ? (
                      <em>{t("workbench.component.ctxMeasured", { defaultValue: "measured" })}</em>
                    ) : row.evidence === "declared" ? (
                      <em>{t("workbench.component.ctxDeclared", { defaultValue: "declared" })}</em>
                    ) : null}
                  </span>
                </div>
              );
            })}
          </section>
        );
      })}
      <p className="ic-checks-note">
        {t("workbench.component.ctxChecksNote", {
          defaultValue:
            "N/M means the instrument has not run. It is never counted as a pass.",
        })}
      </p>
    </div>
  );
}
