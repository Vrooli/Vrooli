/**
 * App.tsx - Main application entry point.
 *
 * Provides:
 * - Error boundaries for graceful error handling
 * - Theme context for light/dark mode support
 *
 * Note: QueryClientProvider is already configured in main.tsx.
 * Route-first navigation keeps world, graph, and detail surfaces in browser
 * history instead of treating detail surfaces like transient dialogs.
 */

import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { getProxyInfo } from '@vrooli/api-base'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ThemeProvider } from './hooks/use-theme'
import { SkillManagerLayout } from './components/layout/SkillManagerLayout'
import { Toaster } from './components/ui/toaster'

function getRouterBasename(): string {
	const proxyInfo = getProxyInfo()
	const proxyPath = proxyInfo ? proxyInfo.primary.path ?? proxyInfo.basePath : undefined
  return proxyPath ? proxyPath.replace(/\/+$/, '') : ''
}

export default function App() {
  const basename = getRouterBasename()

  return (
    <ErrorBoundary>
      <ThemeProvider>
        <BrowserRouter basename={basename}>
          <Routes>
            <Route path="/" element={<Navigate to="/world" replace />} />
            <Route path="/world" element={<SkillManagerLayout />} />
            <Route path="/graph" element={<SkillManagerLayout />} />
            <Route path="/skills/:skillId" element={<SkillManagerLayout />} />
            <Route path="/agents/:agentId" element={<SkillManagerLayout />} />
            <Route path="/teams/:teamId" element={<SkillManagerLayout />} />
            <Route path="/runs/:runId" element={<SkillManagerLayout />} />
            <Route path="/topics/new" element={<SkillManagerLayout />} />
            <Route path="/topics/:topicId" element={<SkillManagerLayout />} />
            <Route path="/actions/:actionId" element={<SkillManagerLayout />} />
            <Route path="*" element={<Navigate to="/world" replace />} />
          </Routes>
        </BrowserRouter>
        <Toaster />
      </ThemeProvider>
    </ErrorBoundary>
  )
}
