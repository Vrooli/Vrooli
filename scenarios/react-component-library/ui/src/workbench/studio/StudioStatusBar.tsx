import { useTranslation } from "../../i18n";
import { formatRelative } from "../data";
import type { DesignDocument } from "../designs";
import type { Coverage } from "./rcl";
import { paintedShare, type FrameAuditMap } from "./checks";
import type { SavedState } from "./useStudio";
import { Dot } from "../primitives";

export function StudioStatusBar({
  doc,
  coverage,
  audits,
  saved,
  dirty,
  saving,
  saveFailed,
  onRetrySave,
  unavailable,
  fallback,
  openCount,
  openUnit = "design",
  issueCount,
  zoom,
  readOnly = false,
}: {
  doc: DesignDocument;
  coverage: Coverage;
  audits: FrameAuditMap;
  saved: SavedState | undefined;
  dirty: boolean;
  saving: boolean;
  saveFailed: boolean;
  onRetrySave: () => void;
  unavailable: number;
  fallback: boolean;
  openCount: number;
  /** What the count names: designs in a document shelf, pages in a workspace. */
  openUnit?: "design" | "page";
  issueCount: number;
  zoom: number;
  /** A curated template is never saved; the bar says so instead of "Saved". */
  readOnly?: boolean;
}) {
  const { t } = useTranslation();
  const state = readOnly
    ? "template"
    : saveFailed
    ? "failed"
    : saving
      ? "saving"
      : dirty
        ? "draft"
        : saved?.local
          ? "local"
          : "saved";
  const area = paintedShare(audits);

  return (
    <footer className="studio-status">
      <span className="studio-status-state">
        <Dot
          tone={state === "saved" ? "green" : state === "failed" ? "red" : state === "template" ? "ice" : "amber"}
        />
        <b>
          {state === "template"
            ? t("workbench.studio.templateRevision", {
                defaultValue: "Template · revision {{revision}}",
                revision: doc.revision,
              })
            : state === "saved"
            ? t("workbench.studio.savedRevision", {
                defaultValue: "Saved · revision {{revision}}",
                revision: saved?.revision ?? doc.revision,
              })
            : state === "saving"
              ? t("workbench.studio.saving", { defaultValue: "Saving…" })
              : state === "failed"
                ? t("workbench.studio.saveFailed", { defaultValue: "Save failed" })
                : state === "local"
                  ? t("workbench.studio.savedLocally", {
                      defaultValue: "Local draft · revision {{revision}}",
                      revision: saved?.revision ?? doc.revision,
                    })
                  : t("workbench.studio.draft", { defaultValue: "Draft" })}
        </b>
        <span>
          {readOnly
            ? t("workbench.workspace.readOnly", { defaultValue: "read-only" })
            : saved
              ? formatRelative(new Date(saved.at))
              : t("workbench.studio.notSaved", { defaultValue: "not saved yet" })}
        </span>
        {state === "failed" || state === "local" ? (
          <button type="button" className="studio-status-retry" onClick={onRetrySave}>
            {t("workbench.studio.retrySave", { defaultValue: "Retry" })}
          </button>
        ) : null}
      </span>

      <span className="studio-status-sep" aria-hidden>
        ·
      </span>
      <span className="studio-status-item studio-status-open">
        {openUnit === "page"
          ? openCount === 1
            ? t("workbench.studio.pageCountOne", { defaultValue: "1 page" })
            : t("workbench.studio.pageCountMany", {
                defaultValue: "{{count}} pages",
                count: openCount,
              })
          : openCount === 1
            ? t("workbench.studio.openCountOne", { defaultValue: "1 design open" })
            : t("workbench.studio.openCountMany", {
                defaultValue: "{{count}} designs open",
                count: openCount,
              })}
      </span>

      {fallback ? (
        <>
          <span className="studio-status-sep" aria-hidden>
            ·
          </span>
          <span className="studio-status-item studio-status-warn" role="status">
            {t("workbench.studio.localDraft", {
              defaultValue: "Local draft · API unavailable",
            })}
          </span>
        </>
      ) : null}

      {unavailable > 0 ? (
        <>
          <span className="studio-status-sep" aria-hidden>
            ·
          </span>
          <span className="studio-status-item studio-status-warn" role="status">
            {t("workbench.studio.unavailable", {
              defaultValue: "{{count}} designs could not be loaded",
              count: unavailable,
            })}
          </span>
        </>
      ) : null}

      <span className="studio-status-spacer" />

      <span className="studio-status-item studio-status-rcl">
        {t("workbench.studio.rclStatus", {
          defaultValue: "RCL {{percent}}% · {{rcl}}/{{total}} layers",
          percent: Math.round(coverage.ratio * 100),
          rcl: coverage.rcl,
          total: coverage.total,
        })}
        {area
          ? ` · ${t("workbench.studio.rclArea", {
              defaultValue: "{{percent}}% painted area",
              percent: Math.round(area.ratio * 100),
            })}`
          : null}
      </span>
      <span className="studio-status-sep" aria-hidden>
        ·
      </span>
      <span className="studio-status-item studio-status-issue">
        {t("workbench.studio.issueCount", {
          defaultValue: "{{count}} checks need review",
          count: issueCount,
        })}
      </span>
      <span className="studio-status-sep" aria-hidden>
        ·
      </span>
      <span className="studio-status-item studio-status-zoom">{Math.round(zoom * 100)}%</span>
    </footer>
  );
}
