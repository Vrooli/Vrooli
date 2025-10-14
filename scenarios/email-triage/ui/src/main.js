import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

// Email Triage Dashboard JavaScript
// AI-powered multi-tenant email management system

const BRIDGE_STATE_KEY = '__emailTriageBridgeInitialized';

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[EmailTriage] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'email-triage' });
    window[BRIDGE_STATE_KEY] = true;
}


// Configuration - dynamically determine API URL
function getApiBaseUrl() {
    // Check for explicit API_BASE_URL in window (for testing/config)
    if (window.API_BASE_URL) {
        return window.API_BASE_URL;
    }

    // Use the same origin but with API port from environment or default
    const protocol = window.location.protocol;
    const hostname = window.location.hostname;
    // API runs on different port - try to get from config or use relative path
    // For production, API should be accessible via proxy or same origin
    return `${protocol}//${hostname}:${window.API_PORT || '19544'}`;
}

const API_BASE_URL = getApiBaseUrl();
const AUTH_TOKEN = localStorage.getItem('email-triage-auth-token') || '';

// Global state
let currentTab = 'dashboard';
let dashboardData = {};
let accounts = [];
let rules = [];

// Utility functions
function showAlert(message, type = 'info') {
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type}`;
    alertDiv.innerHTML = message;
    
    const container = document.querySelector('.tab-content');
    const activeTab = document.querySelector('.tab-pane.active');
    activeTab.insertBefore(alertDiv, activeTab.firstChild);
    
    // Remove alert after 5 seconds
    setTimeout(() => {
        if (alertDiv.parentNode) {
            alertDiv.parentNode.removeChild(alertDiv);
        }
    }, 5000);
}

function showLoading(containerId) {
    const container = document.getElementById(containerId);
    container.innerHTML = `
        <div class="loading">
            <div class="spinner"></div>
            Loading...
        </div>
    `;
}

function hideLoading(containerId) {
    const container = document.getElementById(containerId);
    const loading = container.querySelector('.loading');
    if (loading) {
        loading.style.display = 'none';
    }
}

// API functions
async function apiRequest(method, endpoint, data = null) {
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        },
    };
    
    if (AUTH_TOKEN) {
        options.headers.Authorization = `Bearer ${AUTH_TOKEN}`;
    }
    
    if (data) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
        const result = await response.json();
        
        if (!response.ok) {
            throw new Error(result.error || result.message || 'API request failed');
        }
        
        return result;
    } catch (error) {
        console.error('API Error:', error);
        throw error;
    }
}

// Tab management
function showTab(tabName) {
    // Update nav tabs
    document.querySelectorAll('.nav-tab').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelector(`[onclick="showTab('${tabName}')"]`).classList.add('active');
    
    // Update tab content
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
    });
    document.getElementById(`${tabName}-tab`).classList.add('active');
    
    currentTab = tabName;
    
    // Load tab-specific data
    switch (tabName) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'accounts':
            loadAccounts();
            break;
        case 'rules':
            loadRules();
            break;
        case 'settings':
            loadSystemStatus();
            break;
    }
}

// Dashboard functions
async function loadDashboard() {
    try {
        const data = await apiRequest('GET', '/api/v1/analytics/dashboard');
        dashboardData = data;
        updateDashboardStats(data);
        updateDashboardContent(data);
    } catch (error) {
        // Show placeholder data if API unavailable
        const placeholderData = {
            emails_processed: 1247,
            rules_active: 8,
            actions_automated: 623,
            average_process_time_ms: 150.5
        };
        updateDashboardStats(placeholderData);
        
        document.getElementById('dashboard-content').innerHTML = `
            <div class="alert alert-info">
                <strong>Demo Mode:</strong> API connection not available. Showing placeholder data. 
                Start the email-triage API server to see live data.
            </div>
            <div class="alert alert-success">
                <strong>System Architecture:</strong>
                <ul style="margin-top: 10px;">
                    <li>✅ Multi-tenant authentication via scenario-authenticator</li>
                    <li>✅ AI-powered rule generation with Ollama</li>
                    <li>✅ Semantic search with Qdrant vector database</li>
                    <li>✅ Email server integration via mail-in-a-box</li>
                    <li>✅ Real-time notifications via notification-hub</li>
                </ul>
            </div>
        `;
    }
}

function updateDashboardStats(data) {
    document.getElementById('emails-processed').textContent = data.emails_processed || 0;
    document.getElementById('rules-active').textContent = data.rules_active || 0;
    document.getElementById('actions-automated').textContent = data.actions_automated || 0;
    document.getElementById('avg-process-time').textContent = `${data.average_process_time_ms || 0}ms`;
}

function updateDashboardContent(data) {
    const content = document.getElementById('dashboard-content');
    
    let html = `
        <h3>Recent Activity</h3>
        <div class="email-list">
    `;
    
    if (data.recent_activity && data.recent_activity.length > 0) {
        data.recent_activity.forEach(activity => {
            html += `
                <div class="email-item">
                    <div class="email-subject">${activity.description}</div>
                    <div class="email-sender">${activity.type} • ${new Date(activity.timestamp).toLocaleString()}</div>
                </div>
            `;
        });
    } else {
        html += `
            <div class="email-item">
                <div class="email-subject">Welcome to Email Triage!</div>
                <div class="email-sender">Getting started • Add an email account and create your first rule</div>
            </div>
        `;
    }
    
    html += '</div>';
    content.innerHTML = html;
}

// Account management functions
async function loadAccounts() {
    showLoading('accounts-list');
    
    try {
        const data = await apiRequest('GET', '/api/v1/accounts');
        accounts = data.accounts || [];
        displayAccounts();
    } catch (error) {
        document.getElementById('accounts-list').innerHTML = `
            <div class="alert alert-info">
                <strong>No accounts connected yet.</strong> Add your first email account to start processing emails with AI.
            </div>
        `;
    }
}

function displayAccounts() {
    const container = document.getElementById('accounts-list');
    
    if (accounts.length === 0) {
        container.innerHTML = `
            <div class="alert alert-info">
                <strong>No email accounts connected.</strong> Add your first account to get started.
            </div>
        `;
        return;
    }
    
    let html = '<div class="email-list">';
    
    accounts.forEach(account => {
        html += `
            <div class="email-item">
                <div class="email-subject">${account.email_address}</div>
                <div class="email-sender">
                    Status: ${account.status || 'Active'} • 
                    Last sync: ${account.last_sync ? new Date(account.last_sync).toLocaleString() : 'Never'} •
                    <button class="btn btn-primary" onclick="syncAccount('${account.id}')" style="margin-left: 10px; padding: 5px 10px; font-size: 12px;">Sync Now</button>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

function showAddAccountForm() {
    document.getElementById('add-account-form').style.display = 'block';
}

function hideAddAccountForm() {
    document.getElementById('add-account-form').style.display = 'none';
    document.getElementById('email-address').value = '';
    document.getElementById('email-password').value = '';
}

async function addEmailAccount() {
    const email = document.getElementById('email-address').value;
    const password = document.getElementById('email-password').value;
    
    if (!email || !password) {
        showAlert('Please fill in all fields', 'danger');
        return;
    }
    
    try {
        const data = await apiRequest('POST', '/api/v1/accounts', {
            email_address: email,
            password: password,
            imap_server: 'localhost',
            imap_port: 143,
            smtp_server: 'localhost',
            smtp_port: 25,
            use_tls: false
        });
        
        showAlert('Email account connected successfully!', 'success');
        hideAddAccountForm();
        loadAccounts();
    } catch (error) {
        showAlert(`Failed to connect account: ${error.message}`, 'danger');
    }
}

async function syncAccount(accountId) {
    try {
        await apiRequest('POST', `/api/v1/accounts/${accountId}/sync`);
        showAlert('Email sync started successfully!', 'success');
        
        // Refresh accounts list after a short delay
        setTimeout(() => {
            loadAccounts();
        }, 2000);
    } catch (error) {
        showAlert(`Sync failed: ${error.message}`, 'danger');
    }
}

// Rules management functions
async function loadRules() {
    showLoading('rules-list');
    
    try {
        const data = await apiRequest('GET', '/api/v1/rules');
        rules = data.rules || [];
        displayRules();
    } catch (error) {
        document.getElementById('rules-list').innerHTML = `
            <div class="alert alert-info">
                <strong>No triage rules created yet.</strong> Create your first AI-powered rule to start automating email management.
            </div>
        `;
    }
}

function displayRules() {
    const container = document.getElementById('rules-list');
    
    if (rules.length === 0) {
        container.innerHTML = `
            <div class="alert alert-info">
                <strong>No triage rules created.</strong> Create your first AI-powered rule to automate email processing.
            </div>
        `;
        return;
    }
    
    let html = '<div class="email-list">';
    
    rules.forEach(rule => {
        const confidence = rule.ai_confidence ? `${Math.round(rule.ai_confidence * 100)}%` : 'Manual';
        const status = rule.enabled ? '✅ Active' : '❌ Disabled';
        
        html += `
            <div class="email-item">
                <div class="email-subject">${rule.name}</div>
                <div class="email-sender">
                    ${rule.description} • 
                    AI Confidence: ${confidence} • 
                    Matches: ${rule.match_count || 0} • 
                    ${status}
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

function showAddRuleForm() {
    document.getElementById('add-rule-form').style.display = 'block';
}

function hideAddRuleForm() {
    document.getElementById('add-rule-form').style.display = 'none';
    document.getElementById('rule-description').value = '';
}

async function createTriageRule() {
    const description = document.getElementById('rule-description').value;
    
    if (!description) {
        showAlert('Please describe what you want to do with emails', 'danger');
        return;
    }
    
    try {
        const data = await apiRequest('POST', '/api/v1/rules', {
            description: description,
            use_ai_generation: true,
            actions: [
                { type: 'archive', parameters: {} }
            ]
        });
        
        const confidence = data.ai_confidence ? `${Math.round(data.ai_confidence * 100)}%` : 'N/A';
        showAlert(`Rule created successfully! AI confidence: ${confidence}`, 'success');
        hideAddRuleForm();
        loadRules();
    } catch (error) {
        showAlert(`Failed to create rule: ${error.message}`, 'danger');
    }
}

// Search functions
async function searchEmails() {
    const query = document.getElementById('search-query').value;
    
    if (!query) {
        showAlert('Please enter a search query', 'danger');
        return;
    }
    
    const resultsContainer = document.getElementById('search-results');
    showLoading('search-results');
    
    try {
        const data = await apiRequest('GET', `/api/v1/emails/search?q=${encodeURIComponent(query)}&limit=20`);
        
        let html = `<h3>Search Results (${data.total} found in ${data.query_time_ms}ms)</h3>`;
        
        if (data.results && data.results.length > 0) {
            html += '<div class="email-list">';
            
            data.results.forEach(result => {
                const similarity = Math.round(result.similarity_score * 100);
                html += `
                    <div class="email-item">
                        <div class="email-subject">${result.subject}</div>
                        <div class="email-sender">
                            From: ${result.sender} • 
                            Similarity: ${similarity}% • 
                            ${new Date(result.processed_at).toLocaleDateString()}
                        </div>
                    </div>
                `;
            });
            
            html += '</div>';
        } else {
            html += `
                <div class="alert alert-info">
                    No emails found matching your query. Try different keywords or phrases.
                </div>
            `;
        }
        
        resultsContainer.innerHTML = html;
    } catch (error) {
        resultsContainer.innerHTML = `
            <div class="alert alert-danger">
                Search failed: ${error.message}
            </div>
            <div class="alert alert-info">
                <strong>Demo Search Results:</strong> In a working system, this would show emails related to "${query}" using AI-powered semantic search.
            </div>
        `;
    }
}

// System status functions
async function loadSystemStatus() {
    showLoading('system-status');
    
    try {
        const data = await apiRequest('GET', '/health');
        
        const statusHtml = `
            <div class="alert alert-success">
                <strong>System Status: ${data.status}</strong><br>
                Last checked: ${new Date(data.timestamp).toLocaleString()}
            </div>
            <div class="stats-grid">
                ${Object.entries(data.services || {}).map(([service, status]) => `
                    <div class="stat-card">
                        <div class="stat-value">${status === 'connected' || status === 'available' ? '✅' : '❌'}</div>
                        <div class="stat-label">${service}</div>
                    </div>
                `).join('')}
            </div>
        `;
        
        document.getElementById('system-status').innerHTML = statusHtml;
    } catch (error) {
        document.getElementById('system-status').innerHTML = `
            <div class="alert alert-danger">
                <strong>System Status: Offline</strong><br>
                Could not connect to API server at ${API_BASE_URL}. Please ensure the email-triage API is running.
            </div>
            <div class="alert alert-info">
                <strong>Quick Start:</strong>
                <ol style="margin-top: 10px;">
                    <li>Start required resources: <code>vrooli resource start postgres qdrant</code></li>
                    <li>Start scenario: <code>vrooli scenario run email-triage</code></li>
                    <li>Access dashboard at: <code>${window.location.origin}</code></li>
                </ol>
            </div>
        `;
    }
}

// Initialize the application
function initializeApp() {
    console.log('Email Triage Dashboard v1.0.0');
    console.log('AI-powered multi-tenant email management system');
    
    // Load initial tab
    loadDashboard();
    
    // Set up event listeners
    document.getElementById('search-query').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            searchEmails();
        }
    });
    
    // Check authentication status
    if (!AUTH_TOKEN) {
        showAlert(`
            <strong>Authentication Required:</strong> 
            To use this system, you need to authenticate with scenario-authenticator. 
            This demo shows the interface - in production, users would log in first.
        `, 'info');
    }
}

// Start the application when DOM is loaded
document.addEventListener('DOMContentLoaded', initializeApp);

Object.assign(window, {
    showTab,
    showAddAccountForm,
    hideAddAccountForm,
    addEmailAccount,
    createTriageRule,
    hideAddRuleForm,
    searchEmails,
    syncAccount,
});
