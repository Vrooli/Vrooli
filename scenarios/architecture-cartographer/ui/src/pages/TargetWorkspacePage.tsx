import * as React from "react";
import { Navigate, Outlet } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { useRecentTargets } from "../features/targets/hooks/useRecentTargets";
import { WorkspaceSubNav } from "../layout/WorkspaceSubNav";

/**
 * TargetWorkspacePage — shell for everything that operates against a single
 * target scenario. Renders the per-target sub-nav (Graph/Manifest/Conflicts/
 * Apply/Analytics) and an `<Outlet />` for the selected section.
 *
 * Side-effect: opening a target records it in the recent-targets list so it
 * surfaces on the overview the next time the user lands.
 */
export function TargetWorkspacePage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const { record } = useRecentTargets();

  React.useEffect(() => {
    if (scenario !== null) record(scenario);
    // record() is intentionally not in deps — we only want to write once per
    // scenario per mount. Including it would loop on every storage update.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scenario]);

  if (scenario === null) {
    return <Navigate to="/" replace />;
  }

  return (
    <section
      data-testid={selectors.pages.targetWorkspace}
      aria-labelledby="target-workspace-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="target-workspace-heading" className="text-2xl font-semibold">
          {t(strings.pages.targetWorkspace.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetWorkspace.scenarioLabel)}{" "}
          <span className="font-mono text-app-foreground">{scenario}</span>
        </p>
      </header>
      <WorkspaceSubNav scenario={scenario} />
      <Outlet />
    </section>
  );
}
