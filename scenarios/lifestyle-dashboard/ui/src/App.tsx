/**
 * Lifestyle Dashboard - Main Application
 *
 * Unified personal lifestyle intelligence dashboard providing shared data model,
 * correlation engine, and analytics layer for domain-specific health scenarios.
 *
 * Architecture: Hash-based routing with component-based design.
 * Pages are extracted into src/pages/ for maintainability.
 * Components are extracted into src/components/ for reusability.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Unified dashboard UI with routing
 */
import { HashRouter, Routes, Route } from "react-router-dom";

import { Layout } from "./components/Layout";
import {
  DashboardPage,
  DomainsPage,
  DomainDetailPage,
  EventsPage,
  SettingsPage,
  BriefsPage,
} from "./pages";

export default function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<DashboardPage />} />
          <Route path="domains" element={<DomainsPage />} />
          <Route path="domains/:name" element={<DomainDetailPage />} />
          <Route path="events" element={<EventsPage />} />
          <Route path="briefs" element={<BriefsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </HashRouter>
  );
}
