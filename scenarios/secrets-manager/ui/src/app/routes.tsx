import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { PasswordManagerPage } from "../pages/PasswordManagerPage";
import { normalizeRouterBasename } from "./routerUtils";

export function AppRouter() {
  return <BrowserRouter basename={normalizeRouterBasename(import.meta.env.BASE_URL)}><Routes>
    <Route path="/" element={<PasswordManagerPage page="vault" />} />
    <Route path="/vault" element={<PasswordManagerPage page="vault" />} />
    <Route path="/access" element={<PasswordManagerPage page="access" />} />
    <Route path="/activity" element={<PasswordManagerPage page="activity" />} />
    <Route path="/recovery" element={<PasswordManagerPage page="recovery" />} />
    <Route path="/sources" element={<PasswordManagerPage page="sources" />} />
    <Route path="/settings" element={<PasswordManagerPage page="settings" />} />
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes></BrowserRouter>;
}
