import { BrowserRouter, Routes, Route, useNavigate } from "react-router-dom";
import { AppShell } from "./components/layout/AppShell";
import Dashboard from "./pages/Dashboard";
import ScenarioDetail from "./pages/ScenarioDetail";
import FileDetail from "./pages/FileDetail";
import IssuesView from "./pages/IssuesView";
import CampaignsView from "./pages/CampaignsView";
import SettingsView from "./pages/SettingsView";
import { KeyboardShortcuts } from "./components/ui/keyboard-shortcuts";

function AppContent() {
  const navigate = useNavigate();

  return (
    <>
      <AppShell>
        <main role="main" aria-label="Tidiness Manager Application">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/scenario/:scenarioName" element={<ScenarioDetail />} />
            <Route path="/scenario/:scenarioName/file/*" element={<FileDetail />} />
            <Route path="/scenario/:scenarioName/issues" element={<IssuesView />} />
            <Route path="/campaigns" element={<CampaignsView />} />
            <Route path="/settings" element={<SettingsView />} />
          </Routes>
        </main>
      </AppShell>
      <KeyboardShortcuts onNavigate={navigate} />
    </>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  );
}
