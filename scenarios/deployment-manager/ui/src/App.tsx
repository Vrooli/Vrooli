import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Layout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { Profiles } from "./pages/Profiles";
import { NewProfile } from "./pages/NewProfile";
import { ProfileDetail } from "./pages/ProfileDetail";
import { Analyze } from "./pages/Analyze";
import { Deployments } from "./pages/Deployments";

export default function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/profiles" element={<Profiles />} />
          <Route path="/profiles/new" element={<NewProfile />} />
          <Route path="/profiles/:id" element={<ProfileDetail />} />
          <Route path="/analyze" element={<Analyze />} />
          <Route path="/deployments" element={<Deployments />} />
          <Route path="/deployments/:id" element={<Deployments />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}
