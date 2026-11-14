import { useState } from "react";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SurfaceCatalog } from "../features/surfaces/SurfaceCatalog";
import { ChatWorkspace } from "../features/chat/ChatWorkspace";
import { ModeIndicator } from "../features/integrations/ModeIndicator";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const [compactTab, setCompactTab] = useState("chat");

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      data-compact-tab={compactTab}
      className="companion-dashboard flex flex-col gap-4"
    >
      <div className="companion-dashboard-intro flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0">
          <h2 id="dashboard-heading" className="text-2xl font-semibold">
            {t(strings.pages.dashboard.title)}
          </h2>
          <p className="max-w-3xl text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
        </div>
        <ModeIndicator />
      </div>
      <div className="companion-tabs" role="group" aria-label={t(strings.companion.workspace)}>
        <button type="button" aria-pressed={compactTab === "chat"} onClick={() => setCompactTab("chat")}>{t(strings.companion.chat)}</button>
        <button type="button" aria-pressed={compactTab === "desktop"} onClick={() => setCompactTab("desktop")}>{t(strings.companion.desktop)}</button>
      </div>
      <div className="companion-desktop-workspace"><SurfaceCatalog /></div>
      <div className="companion-chat-workspace"><ChatWorkspace /></div>
    </section>
  );
}
