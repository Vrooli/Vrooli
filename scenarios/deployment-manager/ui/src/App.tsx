import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { Layout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { Profiles } from "./pages/Profiles";
import { NewProfile } from "./pages/NewProfile";
import { ProfileDetail } from "./pages/ProfileDetail";
import { Analyze } from "./pages/Analyze";
import { Deployments } from "./pages/Deployments";
import { BundleTelemetry } from "./pages/BundleTelemetry";
import { Approvals } from "./pages/Approvals";

function getRouterBasename(): string {
  const proxyInfo = getProxyInfo();
  const proxyPath = proxyInfo?.primary?.path ?? proxyInfo?.basePath;
  if (proxyPath) {
    return proxyPath.replace(/\/+$/, "");
  }
  return "";
}

export default function App() {
  const basename = getRouterBasename();

  return (
    <BrowserRouter basename={basename}>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/profiles" element={<Profiles />} />
          <Route path="/profiles/new" element={<NewProfile />} />
          <Route path="/profiles/:id" element={<ProfileDetail />} />
          <Route path="/analyze" element={<Analyze />} />
          <Route path="/telemetry" element={<BundleTelemetry />} />
          <Route path="/deployments" element={<Deployments />} />
          <Route path="/deployments/:id" element={<Deployments />} />
          <Route path="/approvals" element={<Approvals />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}
