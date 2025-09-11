import { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import './Layout.css'

interface LayoutProps {
  children: ReactNode
}

function Layout({ children }: LayoutProps) {
  const location = useLocation()
  
  const isActive = (path: string) => location.pathname === path
  
  return (
    <div className="layout">
      <nav className="navbar">
        <div className="navbar-brand">
          <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
            <circle cx="8" cy="8" r="3" fill="var(--primary-color)" />
            <circle cx="24" cy="8" r="3" fill="var(--secondary-color)" />
            <circle cx="16" cy="24" r="3" fill="var(--accent-color)" />
            <path d="M8 8 L24 8 M8 8 L16 24 M24 8 L16 24" stroke="var(--text-secondary)" strokeWidth="1.5" />
          </svg>
          <span>Graph Studio</span>
        </div>
        
        <ul className="navbar-menu">
          <li>
            <Link to="/dashboard" className={isActive('/dashboard') ? 'active' : ''}>
              Dashboard
            </Link>
          </li>
          <li>
            <Link to="/graphs" className={isActive('/graphs') ? 'active' : ''}>
              Graphs
            </Link>
          </li>
          <li>
            <Link to="/plugins" className={isActive('/plugins') ? 'active' : ''}>
              Plugins
            </Link>
          </li>
          <li>
            <Link to="/settings" className={isActive('/settings') ? 'active' : ''}>
              Settings
            </Link>
          </li>
        </ul>
        
        <div className="navbar-actions">
          <button className="btn btn-primary">
            New Graph
          </button>
        </div>
      </nav>
      
      <main className="main-content">
        {children}
      </main>
      
      <footer className="footer">
        <div className="footer-content">
          <span>Graph Studio v1.0.0</span>
          <span>•</span>
          <span>Powered by Vrooli</span>
        </div>
      </footer>
    </div>
  )
}

export default Layout