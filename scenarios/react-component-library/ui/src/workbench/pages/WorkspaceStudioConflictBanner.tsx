import { useTranslation } from "../../i18n";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";

export function WorkspaceStudioConflictBanner({
  studio,
}: {
  studio: ReturnType<typeof useWorkspaceStudio>;
}) {
  const { t } = useTranslation();
  return studio.conflict ? (
    <div className="studio-conflict" role="alert">
      <div>
        <b>
          {t("workbench.workspace.conflictTitle", {
            defaultValue: "This workspace changed elsewhere",
          })}
        </b>
        <span>
          {t("workbench.workspace.conflictBody", {
            defaultValue:
              "Saving is paused so neither version is overwritten. Choose which one to keep.",
          })}
        </span>
      </div>
      <div className="studio-conflict-actions">
        <button
          type="button"
          className="studio-conflict-btn"
          onClick={() => studio.resolveConflict("take-theirs")}
        >
          {t("workbench.studio.conflictTheirs", { defaultValue: "Load the other version" })}
        </button>
        <button
          type="button"
          className="studio-conflict-btn is-primary"
          onClick={() => studio.resolveConflict("keep-mine")}
        >
          {t("workbench.studio.conflictMine", { defaultValue: "Keep mine" })}
        </button>
      </div>
    </div>
  ) : null;
}
