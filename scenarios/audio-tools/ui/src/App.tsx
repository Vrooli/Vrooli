import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { AppShell } from "./components/AppShell";
import { Toaster } from "./components/ui/toast";
import { PreferencesProvider } from "./hooks/usePreferences";
import { useTranslation } from "./i18n";
import { strings } from "./consts/strings";
import { OverviewPage } from "./features/overview/OverviewPage";

const DiagnosticsPage = lazy(() => import("./features/diagnostics/DiagnosticsPage").then((m) => ({ default: m.DiagnosticsPage })));
const StatusPage = lazy(() => import("./features/status/StatusPage").then((m) => ({ default: m.StatusPage })));
const ConfigurationPage = lazy(() => import("./features/configuration/ConfigurationPage").then((m) => ({ default: m.ConfigurationPage })));
const VoicesPage = lazy(() => import("./features/voices/VoicesPage").then((m) => ({ default: m.VoicesPage })));
const UsagePage = lazy(() => import("./features/usage/UsagePage").then((m) => ({ default: m.UsagePage })));
const DocsPage = lazy(() => import("./features/docs/DocsPage").then((m) => ({ default: m.DocsPage })));
const DocViewerPage = lazy(() => import("./features/docs/DocViewerPage").then((m) => ({ default: m.DocViewerPage })));
const SpeakerVerificationPage = lazy(() => import("./features/admin/SpeakerVerificationPage").then((m) => ({ default: m.SpeakerVerificationPage })));
const WakeWordPage = lazy(() => import("./features/admin/WakeWordPage").then((m) => ({ default: m.WakeWordPage })));
const StreamConfigPage = lazy(() => import("./features/admin/StreamConfigPage").then((m) => ({ default: m.StreamConfigPage })));
const DictationStudioPage = lazy(() => import("./features/dictation-studio/DictationStudioPage").then((m) => ({ default: m.DictationStudioPage })));
const NotFoundPage = lazy(() => import("./features/not-found/NotFoundPage").then((m) => ({ default: m.NotFoundPage })));

function getRouterBasename(): string {
  const info = getProxyInfo();
  const path = info?.primary.path ?? info?.basePath;
  if (!path) return "";
  return path.replace(/\/+$/, "");
}

export default function App() {
  const basename = getRouterBasename();
  return (
    <PreferencesProvider>
      <BrowserRouter basename={basename}>
        <Suspense fallback={<RouteFallback />}>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<OverviewPage />} />
              <Route path="diagnostics" element={<DiagnosticsPage />} />
              <Route path="dictation-studio" element={<DictationStudioPage />} />
              <Route path="status" element={<StatusPage />} />
              <Route path="configuration" element={<ConfigurationPage />} />
              <Route path="voices" element={<VoicesPage />} />
              <Route path="usage" element={<UsagePage />} />
              <Route path="docs" element={<DocsPage />} />
              <Route path="docs/*" element={<DocViewerPage />} />
              <Route path="admin/speaker-verification" element={<SpeakerVerificationPage />} />
              <Route path="admin/wake-word" element={<WakeWordPage />} />
              <Route path="admin/stream-config" element={<StreamConfigPage />} />
              <Route path="overview" element={<Navigate to="/" replace />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </Suspense>
      </BrowserRouter>
      <Toaster />
    </PreferencesProvider>
  );
}

function RouteFallback() {
  const { t } = useTranslation();
  return (
    <div
      role="status"
      aria-label={t(strings.common.loading)}
      className="flex h-full items-center justify-center text-sm text-app-muted-foreground"
    >
      <span
        aria-hidden="true"
        className="h-6 w-6 animate-spin rounded-full border-2 border-app-border border-t-app-primary"
      />
    </div>
  );
}
