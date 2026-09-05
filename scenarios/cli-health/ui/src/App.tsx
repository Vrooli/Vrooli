import { useState } from "react";

import { AppShell } from "./components/AppShell";
import { selectors } from "./consts/selectors";
import { strings } from "./consts/strings";
import { useTranslation } from "./i18n";
import { SearchPanel } from "./features/search/SearchPanel";
import { StatusPanel } from "./features/status/StatusPanel";
import { ValidatePanel } from "./features/validate/ValidatePanel";

type Tab = "search" | "validate" | "status";

export default function App() {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>("search");

  const tabButton = (id: Tab, testId: string, label: string) => (
    <button
      key={id}
      type="button"
      data-testid={testId}
      onClick={() => setTab(id)}
      aria-pressed={tab === id}
      className={
        tab === id
          ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
          : "rounded-control border border-app-border px-3 py-1 text-sm text-app-muted-foreground hover:text-app-foreground"
      }
    >
      {label}
    </button>
  );

  return (
    <AppShell>
      <div
        role="tablist"
        data-testid={selectors.nav.root}
        className="mt-6 flex items-center gap-2"
      >
        {tabButton("search", selectors.nav.tabSearch, t(strings.nav.search))}
        {tabButton("validate", selectors.nav.tabValidate, t(strings.nav.validate))}
        {tabButton("status", selectors.nav.tabStatus, t(strings.nav.status))}
      </div>
      {tab === "search" && <SearchPanel />}
      {tab === "validate" && <ValidatePanel />}
      {tab === "status" && <StatusPanel />}
    </AppShell>
  );
}
