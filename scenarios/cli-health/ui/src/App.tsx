import { useState } from "react";

import { AppShell, type CliHealthTab } from "./components/AppShell";
import { SearchPanel } from "./features/search/SearchPanel";
import { StatusPanel } from "./features/status/StatusPanel";
import { ValidatePanel } from "./features/validate/ValidatePanel";

export default function App() {
  const [tab, setTab] = useState<CliHealthTab>("search");

  return (
    <AppShell activeTab={tab} onTabChange={setTab}>
      {tab === "search" && <SearchPanel />}
      {tab === "validate" && <ValidatePanel />}
      {tab === "status" && <StatusPanel />}
    </AppShell>
  );
}
