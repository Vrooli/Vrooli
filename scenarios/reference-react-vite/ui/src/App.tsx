import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { RouteGate } from "./components/RouteGate";
import { Dashboard } from "./pages/Dashboard";
import { Tasks } from "./pages/Tasks";
import { TaskDetail } from "./pages/TaskDetail";
import { Projects } from "./pages/Projects";
import { ProjectDetail } from "./pages/ProjectDetail";
import { Login } from "./pages/Login";
import { ForgotPassword } from "./pages/ForgotPassword";
import { Settings } from "./pages/Settings";
import { SettingsDisplay } from "./pages/settings/Display";
import { SettingsNotifications } from "./pages/settings/Notifications";
import { SettingsAbout } from "./pages/settings/About";
import { Beta } from "./pages/Beta";
import { AdminUsers } from "./pages/admin/Users";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { AppContextProvider } from "./contexts/AppContext";
import { ROUTE_PATTERNS } from "./routes.generated";

/**
 * Root application component. Routes are sourced from
 * ui/flow/navigation.json via routes.generated.ts so that the URL
 * surface stays in lock-step with the navigation spec.
 */
export default function App() {
  // ╔══════════════════════════════════════════════════════════════╗
  // ║  INTEROP-CRITICAL: Relative basename for proxy contexts      ║
  // ║                                                              ║
  // ║  When served through app-monitor's proxy at                  ║
  // ║  /apps/<name>/proxy/, the basename must match the proxy      ║
  // ║  path. Using "." (relative) works in all contexts:           ║
  // ║  - Local dev: http://localhost:35000/                        ║
  // ║  - Proxy: https://domain/apps/ref/proxy/                     ║
  // ║  - Tunnel: https://tunnel-domain/                            ║
  // ║                                                              ║
  // ║  DO NOT change to "/" or hardcoded paths.                    ║
  // ╚══════════════════════════════════════════════════════════════╝
  return (
    <ErrorBoundary>
      <AppContextProvider>
        <BrowserRouter basename=".">
          <Routes>
            <Route element={<Layout />}>
              <Route
                path={ROUTE_PATTERNS.login}
                element={
                  <RouteGate redirectLoggedIn>
                    <Login />
                  </RouteGate>
                }
              />
              <Route
                path={ROUTE_PATTERNS.forgotPassword}
                element={
                  <RouteGate redirectLoggedIn>
                    <ForgotPassword />
                  </RouteGate>
                }
              />
              <Route element={<RouteGate requireAuth />}>
                <Route path={ROUTE_PATTERNS.dashboard} element={<Dashboard />} />
                <Route path={ROUTE_PATTERNS.tasksIndex} element={<Tasks />} />
                <Route path={ROUTE_PATTERNS.taskDetail} element={<TaskDetail />} />
                <Route path={ROUTE_PATTERNS.projectsIndex} element={<Projects />} />
                <Route path={ROUTE_PATTERNS.projectDetail} element={<ProjectDetail />} />
                <Route path={ROUTE_PATTERNS.settingsIndex} element={<Settings />} />
                <Route path={ROUTE_PATTERNS.settingsDisplay} element={<SettingsDisplay />} />
                <Route path={ROUTE_PATTERNS.settingsNotifications} element={<SettingsNotifications />} />
                <Route path={ROUTE_PATTERNS.settingsAbout} element={<SettingsAbout />} />
                <Route
                  path={ROUTE_PATTERNS.adminUsers}
                  element={
                    <RouteGate requireAuth requireRole="admin">
                      <AdminUsers />
                    </RouteGate>
                  }
                />
                <Route
                  path={ROUTE_PATTERNS.betaDashboard}
                  element={
                    <RouteGate requireAuth requireBeta>
                      <Beta />
                    </RouteGate>
                  }
                />
              </Route>
            </Route>
          </Routes>
        </BrowserRouter>
      </AppContextProvider>
    </ErrorBoundary>
  );
}
