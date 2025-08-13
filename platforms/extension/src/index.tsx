/**
 * Popup UI for {{APP_NAME}} Extension
 * 
 * This is the interface shown when users click the extension icon.
 * It should provide quick access to the most common features.
 * 
 * AGENT NOTE: Customize this based on the app's main functionality.
 * You can replace React with vanilla JS if the app doesn't need it.
 */

import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom/client';

// ============================================
// Types
// ============================================

interface User {
    id: string;
    name: string;
    email: string;
    // Add user fields as needed
}

interface AppState {
    isAuthenticated: boolean;
    user: User | null;
    isLoading: boolean;
    error: string | null;
    // Add app-specific state
}

// ============================================
// Main App Component
// ============================================

function App() {
    const [state, setState] = useState<AppState>({
        isAuthenticated: false,
        user: null,
        isLoading: true,
        error: null,
    });
    
    // Check authentication on mount
    useEffect(() => {
        checkAuth();
    }, []);
    
    // ============================================
    // Core Functions
    // ============================================
    
    async function checkAuth() {
        try {
            const response = await sendMessage({ type: 'AUTH_CHECK' });
            setState({
                isAuthenticated: response.isAuthenticated,
                user: response.user,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setState(prev => ({ 
                ...prev, 
                isLoading: false, 
                error: 'Failed to check authentication' 
            }));
        }
    }
    
    async function handleLogin(e: React.FormEvent) {
        e.preventDefault();
        const form = e.target as HTMLFormElement;
        const formData = new FormData(form);
        
        setState(prev => ({ ...prev, isLoading: true, error: null }));
        
        try {
            const response = await sendMessage({
                type: 'LOGIN',
                credentials: {
                    email: formData.get('email'),
                    password: formData.get('password'),
                },
            });
            
            if (response.success) {
                setState({
                    isAuthenticated: true,
                    user: response.user,
                    isLoading: false,
                    error: null,
                });
            } else {
                setState(prev => ({ 
                    ...prev, 
                    isLoading: false, 
                    error: response.error 
                }));
            }
        } catch (error) {
            setState(prev => ({ 
                ...prev, 
                isLoading: false, 
                error: 'Login failed' 
            }));
        }
    }
    
    async function handleLogout() {
        setState(prev => ({ ...prev, isLoading: true }));
        
        try {
            await sendMessage({ type: 'LOGOUT' });
            setState({
                isAuthenticated: false,
                user: null,
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setState(prev => ({ 
                ...prev, 
                isLoading: false, 
                error: 'Logout failed' 
            }));
        }
    }
    
    async function handle{{PrimaryAction}}() {
        setState(prev => ({ ...prev, isLoading: true }));
        
        try {
            // Get current tab or other context
            const tabInfo = await sendMessage({ type: 'GET_CURRENT_TAB_INFO' });
            
            // Execute primary action
            const result = await sendMessage({
                type: '{{PRIMARY_ACTION_MESSAGE}}',
                data: {
                    url: tabInfo.tab?.url,
                    title: tabInfo.tab?.title,
                    // Add action-specific data
                },
            });
            
            console.log('Action completed:', result);
            setState(prev => ({ ...prev, isLoading: false }));
            
            // Show success feedback
            showNotification('{{PRIMARY_ACTION}} completed!');
            
        } catch (error) {
            setState(prev => ({ 
                ...prev, 
                isLoading: false, 
                error: 'Action failed' 
            }));
        }
    }
    
    // ============================================
    // Helper Functions
    // ============================================
    
    function sendMessage(message: any): Promise<any> {
        return new Promise((resolve, reject) => {
            chrome.runtime.sendMessage(message, (response) => {
                if (chrome.runtime.lastError) {
                    reject(chrome.runtime.lastError);
                } else if (response?.error) {
                    reject(new Error(response.error));
                } else {
                    resolve(response);
                }
            });
        });
    }
    
    function showNotification(message: string) {
        // Simple notification - could be replaced with a toast library
        const notification = document.createElement('div');
        notification.className = 'notification';
        notification.textContent = message;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
    
    function openOptionsPage() {
        chrome.runtime.openOptionsPage();
    }
    
    function openAppWebsite() {
        chrome.tabs.create({ url: 'https://{{APP_DOMAIN}}' });
    }
    
    // ============================================
    // Render UI
    // ============================================
    
    if (state.isLoading) {
        return (
            <div className="container loading">
                <div className="spinner"></div>
                <p>Loading...</p>
            </div>
        );
    }
    
    if (!state.isAuthenticated) {
        return (
            <div className="container auth-container">
                <header>
                    <img src="/icons/icon-48.png" alt="{{APP_NAME}}" />
                    <h1>{{APP_NAME}}</h1>
                </header>
                
                <form onSubmit={handleLogin} className="login-form">
                    <h2>Sign In</h2>
                    
                    {state.error && (
                        <div className="error-message">{state.error}</div>
                    )}
                    
                    <input
                        type="email"
                        name="email"
                        placeholder="Email"
                        required
                        autoFocus
                    />
                    
                    <input
                        type="password"
                        name="password"
                        placeholder="Password"
                        required
                    />
                    
                    <button type="submit" disabled={state.isLoading}>
                        Sign In
                    </button>
                    
                    <div className="form-footer">
                        <a href="#" onClick={openAppWebsite}>
                            Create Account
                        </a>
                    </div>
                </form>
            </div>
        );
    }
    
    return (
        <div className="container main-container">
            <header>
                <div className="user-info">
                    <img src="/icons/icon-32.png" alt="{{APP_NAME}}" />
                    <div>
                        <h3>{state.user?.name || 'User'}</h3>
                        <small>{state.user?.email}</small>
                    </div>
                </div>
                
                <button className="icon-button" onClick={openOptionsPage} title="Settings">
                    ⚙️
                </button>
            </header>
            
            <main>
                {state.error && (
                    <div className="error-message">{state.error}</div>
                )}
                
                <section className="actions">
                    <button 
                        className="primary-action"
                        onClick={handle{{PrimaryAction}}}
                        disabled={state.isLoading}
                    >
                        {{PRIMARY_ACTION}}
                    </button>
                    
                    {/* Add more action buttons as needed */}
                    {/*
                    <button className="secondary-action">
                        Secondary Action
                    </button>
                    */}
                </section>
                
                <section className="quick-stats">
                    <h4>Quick Stats</h4>
                    <div className="stats-grid">
                        <div className="stat">
                            <span className="stat-value">0</span>
                            <span className="stat-label">Items</span>
                        </div>
                        <div className="stat">
                            <span className="stat-value">0</span>
                            <span className="stat-label">Actions</span>
                        </div>
                    </div>
                </section>
                
                {/* Add app-specific UI sections here */}
            </main>
            
            <footer>
                <button className="link-button" onClick={openAppWebsite}>
                    Open {{APP_NAME}}
                </button>
                <button className="link-button" onClick={handleLogout}>
                    Sign Out
                </button>
            </footer>
        </div>
    );
}

// ============================================
// Styles (inline for simplicity)
// ============================================

const styles = `
    * {
        box-sizing: border-box;
    }
    
    .container {
        width: 350px;
        min-height: 200px;
        padding: 16px;
        background: #ffffff;
    }
    
    .loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 200px;
    }
    
    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid #f0f0f0;
        border-top: 3px solid #3b82f6;
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }
    
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
    
    header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid #e5e7eb;
    }
    
    header img {
        width: 32px;
        height: 32px;
        margin-right: 8px;
    }
    
    header h1 {
        margin: 0;
        font-size: 18px;
        font-weight: 600;
    }
    
    .user-info {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    
    .user-info h3 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
    }
    
    .user-info small {
        color: #6b7280;
        font-size: 12px;
    }
    
    .icon-button {
        background: none;
        border: none;
        cursor: pointer;
        font-size: 18px;
        padding: 4px;
        opacity: 0.7;
        transition: opacity 0.2s;
    }
    
    .icon-button:hover {
        opacity: 1;
    }
    
    main {
        margin-bottom: 16px;
    }
    
    .error-message {
        background: #fee;
        color: #c00;
        padding: 8px 12px;
        border-radius: 4px;
        margin-bottom: 12px;
        font-size: 13px;
    }
    
    .actions {
        margin-bottom: 20px;
    }
    
    .primary-action {
        width: 100%;
        padding: 12px;
        background: #3b82f6;
        color: white;
        border: none;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: background 0.2s;
    }
    
    .primary-action:hover:not(:disabled) {
        background: #2563eb;
    }
    
    .primary-action:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .secondary-action {
        width: 100%;
        padding: 10px;
        background: #f3f4f6;
        color: #374151;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        cursor: pointer;
        margin-top: 8px;
        transition: background 0.2s;
    }
    
    .secondary-action:hover {
        background: #e5e7eb;
    }
    
    .quick-stats h4 {
        margin: 0 0 12px 0;
        font-size: 13px;
        font-weight: 600;
        color: #6b7280;
        text-transform: uppercase;
    }
    
    .stats-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }
    
    .stat {
        background: #f9fafb;
        padding: 12px;
        border-radius: 6px;
        text-align: center;
    }
    
    .stat-value {
        display: block;
        font-size: 24px;
        font-weight: 600;
        color: #111827;
    }
    
    .stat-label {
        display: block;
        font-size: 12px;
        color: #6b7280;
        margin-top: 4px;
    }
    
    footer {
        display: flex;
        justify-content: space-between;
        padding-top: 12px;
        border-top: 1px solid #e5e7eb;
    }
    
    .link-button {
        background: none;
        border: none;
        color: #3b82f6;
        cursor: pointer;
        font-size: 13px;
        padding: 4px 8px;
        transition: color 0.2s;
    }
    
    .link-button:hover {
        color: #2563eb;
        text-decoration: underline;
    }
    
    /* Login Form Styles */
    .auth-container header {
        flex-direction: column;
        text-align: center;
        border: none;
        margin-bottom: 24px;
    }
    
    .auth-container header img {
        width: 48px;
        height: 48px;
        margin: 0 0 8px 0;
    }
    
    .login-form h2 {
        margin: 0 0 16px 0;
        font-size: 16px;
        font-weight: 600;
        text-align: center;
    }
    
    .login-form input {
        width: 100%;
        padding: 10px 12px;
        margin-bottom: 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
    }
    
    .login-form input:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
    }
    
    .login-form button {
        width: 100%;
        padding: 10px;
        background: #3b82f6;
        color: white;
        border: none;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: background 0.2s;
    }
    
    .login-form button:hover:not(:disabled) {
        background: #2563eb;
    }
    
    .form-footer {
        text-align: center;
        margin-top: 16px;
    }
    
    .form-footer a {
        color: #3b82f6;
        font-size: 13px;
        text-decoration: none;
    }
    
    .form-footer a:hover {
        text-decoration: underline;
    }
    
    /* Notification Styles */
    .notification {
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        background: #10b981;
        color: white;
        padding: 12px 20px;
        border-radius: 6px;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        animation: slideUp 0.3s ease;
    }
    
    @keyframes slideUp {
        from {
            transform: translateX(-50%) translateY(100%);
            opacity: 0;
        }
        to {
            transform: translateX(-50%) translateY(0);
            opacity: 1;
        }
    }
`;

// ============================================
// Mount React App
// ============================================

const root = document.getElementById('root');
if (root) {
    // Remove loading state
    root.innerHTML = '';
    
    // Inject styles
    const styleSheet = document.createElement('style');
    styleSheet.textContent = styles;
    document.head.appendChild(styleSheet);
    
    // Mount React
    ReactDOM.createRoot(root).render(
        <React.StrictMode>
            <App />
        </React.StrictMode>
    );
}
