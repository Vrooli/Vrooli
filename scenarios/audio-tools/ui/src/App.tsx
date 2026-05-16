import { AppShell } from "./components/AppShell";
import { HealthCard } from "./features/health/HealthCard";
import { ConfigurationConsole } from "./features/configuration/ConfigurationConsole";
import { DiagnosticsWorkbench } from "./features/diagnostics/DiagnosticsWorkbench";
import { UsageDashboard } from "./features/usage/UsageDashboard";
import { DocsViewer } from "./features/docs/DocsViewer";

/**
 * audio-tools UI composition. The four P0 archetype features are rendered
 * directly under the AppShell; future iterations replace this flat
 * composition with a router (sidebar + per-archetype pages).
 */
export default function App() {
  return (
    <AppShell>
      <HealthCard />
      <ConfigurationConsole />
      <DiagnosticsWorkbench />
      <UsageDashboard />
      <DocsViewer />
    </AppShell>
  );
}
