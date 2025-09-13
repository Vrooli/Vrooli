import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import Dashboard from './components/Dashboard';
import ChatbotList from './components/ChatbotList';
import ChatbotEditor from './components/ChatbotEditor';
import Analytics from './components/Analytics';
import TestChat from './components/TestChat';
import apiClient from './utils/api';
import './App.css';

function App() {
  const [apiStatus, setApiStatus] = useState('checking');

  useEffect(() => {
    checkApiHealth();
    const interval = setInterval(checkApiHealth, 30000); // Check every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const checkApiHealth = async () => {
    try {
      const response = await apiClient.get('/health');
      if (response.ok) {
        setApiStatus('healthy');
      } else {
        setApiStatus('error');
      }
    } catch (error) {
      setApiStatus('error');
    }
  };

  if (apiStatus === 'checking') {
    return (
      <div className="app-loading">
        <div className="loading-spinner"></div>
        <p>Connecting to AI Chatbot Manager...</p>
      </div>
    );
  }

  if (apiStatus === 'error') {
    return (
      <div className="app-error">
        <div className="error-icon">⚠️</div>
        <h2>API Server Unavailable</h2>
        <p>The AI Chatbot Manager API is not responding.</p>
        <p>Please make sure the server is running:</p>
        <code>vrooli scenario run ai-chatbot-manager</code>
        <button onClick={checkApiHealth} className="retry-button">
          Retry Connection
        </button>
      </div>
    );
  }

  return (
    <Router>
      <div className="App">
        <Navigation />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/chatbots" element={<ChatbotList />} />
            <Route path="/chatbots/new" element={<ChatbotEditor />} />
            <Route path="/chatbots/:id/edit" element={<ChatbotEditor />} />
            <Route path="/chatbots/:id/analytics" element={<Analytics />} />
            <Route path="/chatbots/:id/test" element={<TestChat />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

function Navigation() {
  const location = useLocation();

  const navItems = [
    { path: '/', label: 'Dashboard', icon: '📊' },
    { path: '/chatbots', label: 'Chatbots', icon: '🤖' },
    { path: '/chatbots/new', label: 'New Chatbot', icon: '➕' },
  ];

  return (
    <nav className="navigation">
      <div className="nav-header">
        <h1 className="nav-title">
          <span className="nav-icon">🚀</span>
          AI Chatbot Manager
        </h1>
      </div>
      <ul className="nav-menu">
        {navItems.map((item) => (
          <li key={item.path} className="nav-item">
            <Link
              to={item.path}
              className={`nav-link ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-item-icon">{item.icon}</span>
              {item.label}
            </Link>
          </li>
        ))}
      </ul>
      <div className="nav-footer">
        <div className="status-indicator healthy">
          <span className="status-dot"></span>
          API Connected
        </div>
      </div>
    </nav>
  );
}

export default App;