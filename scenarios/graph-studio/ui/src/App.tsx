import { Routes, Route, Navigate } from 'react-router-dom'
import { Suspense, lazy } from 'react'
import Layout from './components/Layout'
import LoadingSpinner from './components/LoadingSpinner'

// Lazy load pages for better performance
const Dashboard = lazy(() => import('./pages/Dashboard'))
const GraphEditor = lazy(() => import('./pages/GraphEditor'))
const PluginGallery = lazy(() => import('./pages/PluginGallery'))
const GraphList = lazy(() => import('./pages/GraphList'))
const Settings = lazy(() => import('./pages/Settings'))

function App() {
  return (
    <Layout>
      <Suspense fallback={<LoadingSpinner />}>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/graphs" element={<GraphList />} />
          <Route path="/graphs/:id" element={<GraphEditor />} />
          <Route path="/plugins" element={<PluginGallery />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Suspense>
    </Layout>
  )
}

export default App