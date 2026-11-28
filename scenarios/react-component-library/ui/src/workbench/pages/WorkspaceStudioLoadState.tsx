import { Link } from "react-router-dom";
import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import { Skeleton } from "../primitives";

export function WorkspaceStudioLoadState({
  loadError,
  ready,
}: {
  loadError: string | null;
  ready: boolean;
}) {
  const { t } = useTranslation();
  if (loadError === "not-found") {
    return (
      <div className="wb-page" data-testid="workspace-studio">
        <div className="wb-empty">
          <b className="wb-empty-title">
            {t("workbench.workspace.unavailable", { defaultValue: "Workspace unavailable" })}
          </b>
          <p>{t("workbench.workspace.notFound", { defaultValue: "Workspace not found." })}</p>
          <Link className="wb-btn quiet" to={appRoutes.design}>
            {t("workbench.workspace.back", { defaultValue: "Back to workspaces" })}
          </Link>
        </div>
      </div>
    );
  }
  if (loadError) {
    return (
      <div className="wb-page" data-testid="workspace-studio">
        <div className="wb-empty">
          <b className="wb-empty-title">
            {t("workbench.workspace.loadFailed", { defaultValue: "The workspace could not load" })}
          </b>
          <p>{loadError}</p>
          <Link className="wb-btn quiet" to={appRoutes.design}>
            {t("workbench.workspace.back", { defaultValue: "Back to workspaces" })}
          </Link>
        </div>
      </div>
    );
  }
  if (!ready) {
    return (
      <div className="studio" data-testid="workspace-studio" aria-busy="true">
        <div className="studio-loading">
          <Skeleton height={24} />
          <Skeleton height={240} />
        </div>
      </div>
    );
  }
  return null;
}
