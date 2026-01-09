import { StrictMode, lazy, Suspense } from 'react'
import ReactDOM from 'react-dom/client'
import { HashRouter, Routes, Route, Navigate, useParams } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { ConfirmDialogProvider } from './utils/confirmDialog'
import { ReportIssueProvider } from './components/issues/ReportIssueProvider'
import { KeyboardShortcutsDialog } from './components/ui/keyboard-shortcuts-dialog'
import './styles/global.css'
import './styles/legacy.css'

// Lazy load pages for code splitting (reduces initial bundle size)
const Orientation = lazy(() => import('./pages/Orientation'))
const Catalog = lazy(() => import('./pages/Catalog'))
const Drafts = lazy(() => import('./pages/Drafts'))
const Backlog = lazy(() => import('./pages/Backlog'))
const RequirementsRegistry = lazy(() => import('./pages/RequirementsRegistry'))
const ScenarioControlCenter = lazy(() => import('./pages/ScenarioControlCenter'))
const QualityScanner = lazy(() => import('./pages/QualityScanner'))

// Loading fallback component with better visual feedback
const PageLoader = () => (
  <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-gradient-to-br from-violet-50/30 via-white to-slate-50/30">
    <div className="flex items-center justify-center gap-3">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-violet-200 border-t-violet-600"></div>
      <span className="text-lg font-medium text-slate-700">Loading PRD Control Tower...</span>
    </div>
    <div className="text-sm text-slate-500">Preparing your workspace</div>
  </div>
)

const LegacyScenarioRedirect = () => {
  const { entityType = '', entityName = '' } = useParams()
  return <Navigate to={`/scenario/${entityType}/${entityName}`} replace />
}

declare global {
  interface Window {
    __prdControlTowerBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window) {
  if (!window.__prdControlTowerBridgeInitialized) {
    let parentOrigin: string | undefined
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin
      }
    } catch {
      // Unable to determine parent origin; iframe bridge will use fallback behavior
    }

    try {
      initIframeBridgeChild({
        parentOrigin,
        appId: 'prd-control-tower',
        captureLogs: { enabled: true },
        captureNetwork: { enabled: true },
      })
      window.__prdControlTowerBridgeInitialized = true
    } catch (error) {
      console.warn('[prd-control-tower] Unable to initialize iframe bridge', error)
    }
  }
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ConfirmDialogProvider>
      <ReportIssueProvider>
        <HashRouter>
          <Suspense fallback={<PageLoader />}>
            <Routes>
              <Route path="/" element={<Orientation />} />
              <Route path="/catalog" element={<Catalog />} />
              <Route path="/drafts" element={<Drafts />} />
              <Route path="/draft/:entityType/:entityName" element={<Drafts />} />
              <Route path="/backlog" element={<Backlog />} />
              <Route path="/requirements" element={<RequirementsRegistry />} />
              <Route path="/requirements-registry" element={<RequirementsRegistry />} />
              <Route path="/scenario/:entityType/:entityName" element={<ScenarioControlCenter />} />
              {/* Legacy routes redirect to Scenario Control Center */}
              <Route path="/prd/:entityType/:entityName" element={<LegacyScenarioRedirect />} />
              <Route path="/requirements-dashboard/:entityType/:entityName" element={<LegacyScenarioRedirect />} />
              <Route path="/requirements/:entityType/:entityName" element={<LegacyScenarioRedirect />} />
              <Route path="/targets/:entityType/:entityName" element={<LegacyScenarioRedirect />} />
              <Route path="/quality-scanner" element={<QualityScanner />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Suspense>
          <KeyboardShortcutsDialog />
        </HashRouter>
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: '#fff',
              color: '#1a202c',
              padding: '16px',
              borderRadius: '8px',
              boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
            },
            success: {
              iconTheme: {
                primary: '#10b981',
                secondary: '#fff',
              },
            },
            error: {
              iconTheme: {
                primary: '#ef4444',
                secondary: '#fff',
              },
            },
          }}
        />
      </ReportIssueProvider>
    </ConfirmDialogProvider>
  </StrictMode>,
)
