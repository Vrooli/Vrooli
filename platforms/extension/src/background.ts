/**
 * Background Service Worker for {{APP_NAME}} Extension
 * 
 * This runs persistently in the background and handles:
 * - API communication
 * - Long-running tasks
 * - Cross-component messaging
 * - Browser event handling
 * 
 * AGENT NOTE: Adapt this file based on the app's needs.
 * Common patterns are provided below - remove unused ones.
 */

// ============================================
// Configuration
// ============================================

const CONFIG = {
    API_BASE: '{{API_ENDPOINT}}',
    APP_DOMAIN: '{{APP_DOMAIN}}',
    SYNC_INTERVAL: 5 * 60 * 1000, // 5 minutes
};

// ============================================
// State Management
// ============================================

interface ExtensionState {
    isAuthenticated: boolean;
    user: any;
    lastSync: number;
    // Add app-specific state here
}

let state: ExtensionState = {
    isAuthenticated: false,
    user: null,
    lastSync: 0,
};

// ============================================
// Lifecycle Events
// ============================================

// When extension is installed or updated
chrome.runtime.onInstalled.addListener((details) => {
    console.log(`{{APP_NAME}} Extension ${details.reason}`);
    
    if (details.reason === 'install') {
        // First time setup
        initializeExtension();
        
        // Open welcome page (optional)
        // chrome.tabs.create({ url: 'https://{{APP_DOMAIN}}/welcome' });
    } else if (details.reason === 'update') {
        // Handle updates
        handleUpdate(details.previousVersion);
    }
});

// When browser starts up
chrome.runtime.onStartup.addListener(() => {
    console.log('{{APP_NAME}} Extension started');
    initializeExtension();
});

// ============================================
// Message Handling (Popup <-> Background communication)
// ============================================

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    console.log('Message received:', message);
    
    // Handle async responses
    (async () => {
        try {
            switch (message.type) {
                case 'AUTH_CHECK':
                    sendResponse({ isAuthenticated: state.isAuthenticated, user: state.user });
                    break;
                    
                case 'LOGIN':
                    const loginResult = await handleLogin(message.credentials);
                    sendResponse(loginResult);
                    break;
                    
                case 'LOGOUT':
                    await handleLogout();
                    sendResponse({ success: true });
                    break;
                    
                case '{{PRIMARY_ACTION_MESSAGE}}':
                    const result = await handle{{PrimaryAction}}(message.data);
                    sendResponse(result);
                    break;
                    
                case 'GET_CURRENT_TAB_INFO':
                    const tab = await getCurrentTab();
                    sendResponse({ tab });
                    break;
                    
                // Add more message handlers as needed
                default:
                    sendResponse({ error: 'Unknown message type' });
            }
        } catch (error) {
            console.error('Error handling message:', error);
            sendResponse({ error: error.message });
        }
    })();
    
    return true; // Keep message channel open for async response
});

// ============================================
// Keyboard Commands
// ============================================

chrome.commands.onCommand.addListener((command) => {
    console.log('Command triggered:', command);
    
    switch (command) {
        case '{{PRIMARY_ACTION_ID}}':
            execute{{PrimaryAction}}();
            break;
        // Add more commands as defined in manifest.json
    }
});

// ============================================
// Context Menu (Right-click menu)
// ============================================

// Uncomment if using context menus
/*
chrome.runtime.onInstalled.addListener(() => {
    chrome.contextMenus.create({
        id: 'send-to-{{APP_NAME}}',
        title: 'Send to {{APP_NAME}}',
        contexts: ['selection', 'link', 'image'],
    });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
    if (info.menuItemId === 'send-to-{{APP_NAME}}') {
        handleContextMenuAction(info, tab);
    }
});
*/

// ============================================
// Tab Management
// ============================================

// Listen for tab updates (useful for page-specific features)
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (changeInfo.status === 'complete' && tab.url) {
        // Check if this is a relevant URL for your app
        if (tab.url.includes('{{RELEVANT_DOMAIN}}')) {
            // Inject content script or update badge
            handleRelevantPage(tab);
        }
    }
});

// ============================================
// Alarms (Periodic Tasks)
// ============================================

// Set up periodic sync (if needed)
chrome.alarms.create('sync', { periodInMinutes: 5 });

chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'sync') {
        syncWithServer();
    }
});

// ============================================
// Storage Management
// ============================================

async function saveToStorage(key: string, value: any): Promise<void> {
    return new Promise((resolve) => {
        chrome.storage.local.set({ [key]: value }, resolve);
    });
}

async function getFromStorage(key: string): Promise<any> {
    return new Promise((resolve) => {
        chrome.storage.local.get(key, (result) => {
            resolve(result[key]);
        });
    });
}

// ============================================
// API Communication
// ============================================

async function apiRequest(endpoint: string, options: RequestInit = {}): Promise<any> {
    const token = await getFromStorage('authToken');
    
    const response = await fetch(`${CONFIG.API_BASE}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': token ? `Bearer ${token}` : '',
            ...options.headers,
        },
    });
    
    if (!response.ok) {
        throw new Error(`API error: ${response.statusText}`);
    }
    
    return response.json();
}

// ============================================
// Core Functions
// ============================================

async function initializeExtension() {
    // Load saved state
    const savedState = await getFromStorage('extensionState');
    if (savedState) {
        state = savedState;
    }
    
    // Check authentication
    await checkAuthentication();
    
    // Update badge
    updateBadge();
}

async function handleLogin(credentials: any) {
    try {
        const response = await apiRequest('/auth/login', {
            method: 'POST',
            body: JSON.stringify(credentials),
        });
        
        state.isAuthenticated = true;
        state.user = response.user;
        
        await saveToStorage('authToken', response.token);
        await saveToStorage('extensionState', state);
        
        updateBadge();
        
        return { success: true, user: response.user };
    } catch (error) {
        return { success: false, error: error.message };
    }
}

async function handleLogout() {
    state.isAuthenticated = false;
    state.user = null;
    
    await chrome.storage.local.clear();
    updateBadge();
}

async function checkAuthentication() {
    const token = await getFromStorage('authToken');
    if (token) {
        try {
            const user = await apiRequest('/auth/me');
            state.isAuthenticated = true;
            state.user = user;
        } catch {
            state.isAuthenticated = false;
        }
    }
}

function updateBadge() {
    if (state.isAuthenticated) {
        chrome.action.setBadgeText({ text: '' });
        chrome.action.setBadgeBackgroundColor({ color: '#4CAF50' });
    } else {
        chrome.action.setBadgeText({ text: '!' });
        chrome.action.setBadgeBackgroundColor({ color: '#F44336' });
    }
}

async function getCurrentTab(): Promise<chrome.tabs.Tab | null> {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    return tab || null;
}

async function syncWithServer() {
    if (!state.isAuthenticated) return;
    
    try {
        // Implement sync logic here
        console.log('Syncing with server...');
        state.lastSync = Date.now();
        await saveToStorage('extensionState', state);
    } catch (error) {
        console.error('Sync failed:', error);
    }
}

// ============================================
// App-Specific Functions (CUSTOMIZE THESE)
// ============================================

async function handle{{PrimaryAction}}(data: any) {
    // Implement primary action logic
    console.log('Executing {{PRIMARY_ACTION}}:', data);
    
    // Example: Send data to API
    return apiRequest('/{{primary-endpoint}}', {
        method: 'POST',
        body: JSON.stringify(data),
    });
}

async function execute{{PrimaryAction}}() {
    // Get current tab or other context
    const tab = await getCurrentTab();
    if (!tab) return;
    
    // Execute action
    // This is triggered by keyboard shortcut
    console.log('Quick action triggered on:', tab.url);
}

async function handleRelevantPage(tab: chrome.tabs.Tab) {
    // Handle pages relevant to your app
    console.log('Relevant page detected:', tab.url);
    
    // Example: Inject content script
    // chrome.scripting.executeScript({
    //     target: { tabId: tab.id! },
    //     files: ['content.js']
    // });
    
    // Or: Update badge with page-specific info
    chrome.action.setBadgeText({ 
        text: '✓',
        tabId: tab.id 
    });
}

async function handleUpdate(previousVersion?: string) {
    console.log(`Updated from version ${previousVersion}`);
    
    // Handle any migration needed between versions
    // Example: Clear old cache, update storage schema, etc.
}

// Uncomment if using context menus
/*
async function handleContextMenuAction(info: chrome.contextMenus.OnClickData, tab?: chrome.tabs.Tab) {
    if (info.selectionText) {
        // Handle selected text
        console.log('Processing selection:', info.selectionText);
    } else if (info.linkUrl) {
        // Handle link
        console.log('Processing link:', info.linkUrl);
    } else if (info.srcUrl) {
        // Handle image
        console.log('Processing image:', info.srcUrl);
    }
    
    // Send to your API or store locally
}
*/

// ============================================
// Error Handling
// ============================================

// Global error handler
self.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
    // Optionally report to error tracking service
});

// Log for debugging
console.log('{{APP_NAME}} Extension background service worker loaded');
