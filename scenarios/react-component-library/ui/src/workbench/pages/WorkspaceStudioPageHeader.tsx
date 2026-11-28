import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { forkWorkspace } from "../../api/workspaces";
import type { Workspace } from "../../api/workspaces";
import type { DesignFrame } from "../design-types";
import { useTranslation } from "../../i18n";
import { workspacePath } from "../../routes";
import { WorkspaceTopBar } from "../studio/WorkspaceTopBar";
import type { LoadedWorkspaceStudio } from "./workspaceStudioPageTypes";
import type { WorkspaceStudioPageActionSet } from "./workspaceStudioPageActions";

interface WorkspaceStudioPageHeaderProps {
  studio: LoadedWorkspaceStudio;
  workspace: Workspace;
  frame: DesignFrame;
  actions: WorkspaceStudioPageActionSet;
  onImportHtml: () => void;
  onImportPatch: () => void;
  updateURL: (next: Record<string, string | undefined>) => void;
  onPresent: () => void;
  flash: (message: string) => void;
}

export function WorkspaceStudioPageHeader({
  studio,
  workspace,
  frame,
  actions,
  onImportHtml,
  onImportPatch,
  updateURL,
  onPresent,
  flash,
}: WorkspaceStudioPageHeaderProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [forking, setForking] = useState(false);
  const [forkError, setForkError] = useState<string | null>(null);
  return (
    <>
      <WorkspaceTopBar
        workspaceName={workspace.name}
        pages={workspace.pages}
        activePageId={studio.page.id}
        onSelectPage={(id) => {
          studio.selectPage(id);
          updateURL({ page: id, display: undefined, state: undefined });
        }}
        onAddPage={() => {
          studio.addPage();
          flash(t("workbench.workspace.pageAdded", { defaultValue: "Added a page." }));
        }}
        onDuplicatePage={() => studio.duplicatePage()}
        onImportHtml={onImportHtml}
        onImportPatch={onImportPatch}
        onRemovePage={() => {
          studio.removePage();
          flash(t("workbench.workspace.pageRemoved", { defaultValue: "Removed the page." }));
        }}
        canUndo={studio.history.index > 0}
        canRedo={studio.history.index < studio.history.entries.length}
        onUndo={studio.undo}
        onRedo={studio.redo}
        dirty={studio.dirty}
        saving={studio.saving}
        saveFailed={studio.saveFailed}
        onSave={() => void studio.save()}
        device={frame.device}
        onDevice={actions.handleDevice}
        onPresent={onPresent}
        onShare={actions.share}
        readOnly={studio.readOnly}
      />

      {workspace.tags.includes("ex9-capture") ? (
        <div
          className="studio-template-bar"
          role="status"
          aria-label={t("workbench.workspace.captureRevisionBadge", {
            defaultValue: "EX9 revision",
          })}
        >
          <span className="studio-template-badge">
            {t("workbench.workspace.captureRevisionBadge", { defaultValue: "EX9 revision" })}
          </span>
          <p>
            <b>{workspace.summary}</b>
            {" · "}
            {t("workbench.workspace.captureRevisionEvidence", {
              defaultValue:
                "The L0 screenshot is captured evidence. These editable layers were mapped from the L1 frozen page and are a draft, not a composited screenshot.",
            })}
          </p>
        </div>
      ) : null}

      {workspace.tags.includes("design-brief") ? (
        <div
          className="studio-template-bar"
          role="status"
          aria-label={t("workbench.workspace.designBriefBadge", {
            defaultValue: "Design brief",
          })}
        >
          <span className="studio-template-badge">
            {t("workbench.workspace.designBriefBadge", { defaultValue: "Design brief" })}
          </span>
          <p>
            <b>{workspace.summary}</b>
            {" · "}
            {t("workbench.workspace.designBriefHint", {
              defaultValue:
                "Keep the purpose, target user, and main action in view while you shape the page.",
            })}
          </p>
        </div>
      ) : null}

      {studio.readOnly ? (
        <div
          className="studio-template-bar"
          role="status"
          aria-label={t("workbench.workspace.template", { defaultValue: "Curated template" })}
        >
          <span className="studio-template-badge">
            {t("workbench.workspace.templateChip", { defaultValue: "Template" })}
          </span>
          <p>
            <b>{t("workbench.workspace.templateTitle", { defaultValue: "Curated template" })}</b>
            {" · "}
            {t("workbench.workspace.templateBodyShort", {
              defaultValue: "A reviewed snapshot. Fork it to make it yours.",
            })}
          </p>
          {forkError ? <span role="alert">{forkError}</span> : null}
          <button
            type="button"
            className="wb-btn primary sm"
            disabled={forking}
            onClick={() => {
              setForking(true);
              setForkError(null);
              void forkWorkspace(workspace.id)
                .then((forked) => navigate(workspacePath(forked.id), { replace: true }))
                .catch((cause: unknown) =>
                  setForkError(
                    cause instanceof Error
                      ? cause.message
                      : t("workbench.workspace.forkFailed", {
                          defaultValue: "The template could not be forked.",
                        }),
                  ),
                )
                .finally(() => setForking(false));
            }}
          >
            {forking
              ? t("workbench.workspace.forking", { defaultValue: "Forking…" })
              : t("workbench.workspace.forkAction", { defaultValue: "Fork to edit" })}
          </button>
        </div>
      ) : null}
    </>
  );
}
