import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import FunnelBuilder from './pages/FunnelBuilder'
import FunnelPreview from './pages/FunnelPreview'
import Analytics from './pages/Analytics'
import Templates from './pages/Templates'

function App() {
  return (
    <Routes>
      <Route path="/health" element={<div>OK</div>} />
      <Route path="/" element={<Layout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="builder/:id?" element={<FunnelBuilder />} />
        <Route path="preview/:id" element={<FunnelPreview />} />
        <Route path="analytics/:id" element={<Analytics />} />
        <Route path="templates" element={<Templates />} />
      </Route>
    </Routes>
  )
}

export default App