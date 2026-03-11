import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { Tasks } from "./pages/Tasks";
import { Projects } from "./pages/Projects";
import { ErrorBoundary } from "./components/ErrorBoundary";

/**
 * Root application component with routing
 *
 * Routes:
 * - / : Dashboard overview
 * - /tasks : Task list and management
 * - /projects : Project list and management
 *
 * [REQ:P0-006a] React component architecture with proper routing
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
      <BrowserRouter basename=".">
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/tasks" element={<Tasks />} />
            <Route path="/projects" element={<Projects />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
