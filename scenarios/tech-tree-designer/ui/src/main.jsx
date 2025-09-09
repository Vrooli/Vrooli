import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

// Update health status
if (typeof window !== 'undefined') {
    window.__TECH_TREE_HEALTH__ = {
        status: 'healthy',
        timestamp: Date.now(),
        service: 'tech-tree-designer-ui',
        version: __APP_VERSION__
    };
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)