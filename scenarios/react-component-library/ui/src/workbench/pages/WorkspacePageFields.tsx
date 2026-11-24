import { useTranslation } from "../../i18n";

export function PageFields({
  pageName,
  route,
  pageTypeId,
  stateName,
  canRemoveState,
  onPageName,
  onRoute,
  onPageType,
  onStateName,
  onRemoveState,
}: {
  pageName: string;
  route: string;
  pageTypeId: string;
  stateName: string;
  canRemoveState: boolean;
  onPageName: (value: string) => void;
  onRoute: (value: string) => void;
  onPageType: (value: string) => void;
  onStateName: (value: string) => void;
  onRemoveState: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="studio-page-fields">
      <span className="wb-caption">
        {t("workbench.workspace.pageContext", { defaultValue: "Page" })}
      </span>
      <label>
        <span className="wb-label">
          {t("workbench.workspace.pageName", { defaultValue: "Name" })}
        </span>
        <input
          className="wb-input"
          value={pageName}
          onChange={(event) => onPageName(event.target.value)}
        />
      </label>
      <label>
        <span className="wb-label">
          {t("workbench.workspace.pageRoute", { defaultValue: "Route" })}
        </span>
        <input
          className="wb-input"
          value={route}
          placeholder={t("workbench.workspace.routePlaceholder", {
            defaultValue: "/onboarding/welcome",
          })}
          onChange={(event) => onRoute(event.target.value)}
        />
      </label>
      <label>
        <span className="wb-label">
          {t("workbench.workspace.pageType", { defaultValue: "Page type" })}
        </span>
        <input
          className="wb-input"
          value={pageTypeId}
          placeholder={t("workbench.workspace.pageTypePlaceholder", {
            defaultValue: "templates.landing-page",
          })}
          onChange={(event) => onPageType(event.target.value)}
        />
      </label>
      <span className="wb-caption">
        {t("workbench.workspace.stateSection", { defaultValue: "State" })}
      </span>
      <label>
        <span className="wb-label">
          {t("workbench.workspace.stateName", { defaultValue: "Name" })}
        </span>
        <input
          className="wb-input"
          value={stateName}
          onChange={(event) => onStateName(event.target.value)}
        />
      </label>
      <div className="studio-shelf-actions">
        <button
          type="button"
          className="studio-chip-btn is-danger"
          onClick={onRemoveState}
          disabled={!canRemoveState}
        >
          {t("workbench.workspace.removeState", { defaultValue: "Remove state" })}
        </button>
      </div>
    </div>
  );
}
