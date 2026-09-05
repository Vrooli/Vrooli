import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { Layout } from "./features/shared/Layout";
import { Dashboard } from "./features/shared/Dashboard";
import { Profiles } from "./features/profiles/Profiles";
import { NewProfile } from "./features/profiles/NewProfile";
import { ProfileDetail } from "./features/profiles/ProfileDetail";
import { Analyze } from "./features/dependencies/Analyze";
import { Deployments } from "./features/deployments/Deployments";
import { BundleTelemetry } from "./features/telemetry/BundleTelemetry";
import { Approvals } from "./features/releases/Approvals";
import { Releases } from "./features/releases/Releases";
import { EvidenceReview } from "./features/evidence/EvidenceReview";

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
          <Route path="/releases" element={<Releases />} />
          <Route path="/evidence" element={<EvidenceReview />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}
