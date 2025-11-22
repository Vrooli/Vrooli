import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AppShell } from "./components/layout/AppShell";
import Dashboard from "./pages/Dashboard";
import ScenarioDetail from "./pages/ScenarioDetail";
import FileDetail from "./pages/FileDetail";
import IssuesView from "./pages/IssuesView";
import CampaignsView from "./pages/CampaignsView";
import SettingsView from "./pages/SettingsView";

export default function App() {
  return (
    <BrowserRouter>
      <AppShell>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/scenario/:scenarioName" element={<ScenarioDetail />} />
          <Route path="/scenario/:scenarioName/file/*" element={<FileDetail />} />
          <Route path="/scenario/:scenarioName/issues" element={<IssuesView />} />
          <Route path="/campaigns" element={<CampaignsView />} />
          <Route path="/settings" element={<SettingsView />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}
