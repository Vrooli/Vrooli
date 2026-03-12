import { BrowserRouter, Routes, Route } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import ReferenceDetail from "./pages/ReferenceDetail";

// ─────────────────────────────────────────────────────────────────────────────
// App Router
// [REQ:P0-001] Reference Scenario Registry - Navigation structure
// ─────────────────────────────────────────────────────────────────────────────
//
// Navigation Model:
//   / (Dashboard) → List all references with connection counts
//   /references/:slug → Reference detail view with full connection info
//
// The router uses BrowserRouter for clean URLs. All pages share the same
// layout structure (header/main/footer) to maintain visual consistency.
//
// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: BrowserRouter basename                    ║
// ║                                                              ║
// ║  The basename prop is set to "/" for standalone operation.   ║
// ║  When embedded in an iframe with a subpath, the host should  ║
// ║  configure the iframe src with the appropriate path, or the  ║
// ║  scenario's vite.config.ts base should be adjusted.          ║
// ║                                                              ║
// ║  This explicit "/" ensures the router works correctly when   ║
// ║  served from the scenario's root path.                       ║
// ╚══════════════════════════════════════════════════════════════╝
// ─────────────────────────────────────────────────────────────────────────────

export default function App() {
  return (
    <BrowserRouter basename="/">
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/references/:slug" element={<ReferenceDetail />} />
      </Routes>
    </BrowserRouter>
  );
}
