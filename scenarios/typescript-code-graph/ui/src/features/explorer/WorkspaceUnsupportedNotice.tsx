import { Info } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Informational notice shown when Extract returns the typed
 * `workspace_unsupported` outcome (Connect CodeUnimplemented). pnpm/yarn
 * workspaces are intentionally out of scope for v1 — this is designed
 * behavior, so it is presented as information (an Info badge, not a danger
 * error state) with a pointer to the tracked limitation.
 */
export function WorkspaceUnsupportedNotice() {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.workbench.status.workspaceUnsupported}
      role="status"
      className="flex items-start gap-3 rounded-panel border border-app-border bg-app-surface p-4 backdrop-blur-sm"
    >
      <Info aria-hidden="true" className="mt-0.5 h-5 w-5 text-app-primary" />
      <div className="flex flex-col gap-1">
        <p className="text-sm font-semibold text-app-foreground">
          {t(strings.workbench.workspaceUnsupported.title)}
        </p>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.workbench.workspaceUnsupported.description)}
        </p>
      </div>
    </div>
  );
}
