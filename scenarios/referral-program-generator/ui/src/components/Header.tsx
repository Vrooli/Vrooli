import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Target, BarChart3, Settings, Zap } from 'lucide-react';
import './Header.css';

export function Header() {
  const location = useLocation();
  
  const navigation = [
    { name: 'Dashboard', href: '/', icon: BarChart3 },
    { name: 'Analyze', href: '/analyze', icon: Target },
    { name: 'Generate', href: '/generate', icon: Zap },
    { name: 'Implement', href: '/implement', icon: Settings },
    { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  ];

  return (
    <header className="header">
      <div className="header-container">
        <div className="header-brand">
          <Link to="/" className="brand-link">
            <div className="brand-icon">
              <Target size={28} />
            </div>
            <div className="brand-text">
              <h1 className="brand-title">Referral Program Generator</h1>
              <p className="brand-subtitle">Automated affiliate systems for Vrooli scenarios</p>
            </div>
          </Link>
        </div>
        
        <nav className="header-nav">
          <ul className="nav-list">
            {navigation.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.href;
              
              return (
                <li key={item.name}>
                  <Link
                    to={item.href}
                    className={`nav-link ${isActive ? 'nav-link-active' : ''}`}
                  >
                    <Icon size={18} />
                    <span>{item.name}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
        
        <div className="header-actions">
          <div className="status-indicator status-success">
            <div className="status-dot"></div>
            <span>Online</span>
          </div>
        </div>
      </div>
    </header>
  );
}