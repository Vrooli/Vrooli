// Device Sync Hub - Main Application (Refactored)

class DeviceSyncApp {
  constructor() {
    // Configuration from meta tags
    this.config = this.loadConfiguration();
    
    // Initialize managers
    this.authManager = new AuthManager(this.config);
    this.deviceManager = new DeviceManager();
    this.apiClient = new SyncApiClient(this.config.apiUrl, this.authManager);
    this.wsManager = new WebSocketManager(this.config, this.authManager, this.deviceManager);
    
    // App state
    this.syncItems = [];
    this.currentFilter = 'all';
    this.settings = {
      defaultExpiry: 24,
      maxFileSize: 10485760 // 10MB
    };

    // UI elements
    this.elements = {};
    
    this.init();
  }

  // Load configuration from meta tags
  loadConfiguration() {
    const getMetaContent = (name) => {
      const meta = document.querySelector(`meta[name="${name}"]`);
      return meta ? meta.content : null;
    };

    return {
      apiUrl: getMetaContent('api-url') || '/api',
      authUrl: getMetaContent('auth-url') || '/auth',
      apiPort: getMetaContent('api-port'),
      authPort: getMetaContent('auth-port')
    };
  }

  // Initialize the application
  async init() {
    this.cacheElements();
    this.setupEventListeners();
    this.setupWebSocketHandlers();
    await this.checkAuth();
  }

  // Cache DOM elements
  cacheElements() {
    this.elements = {
      // Screens
      loadingScreen: document.getElementById('loading-screen'),
      authScreen: document.getElementById('auth-screen'),
      mainApp: document.getElementById('main-app'),
      
      // Auth form
      emailInput: document.getElementById('email'),
      passwordInput: document.getElementById('password'),
      loginBtn: document.getElementById('login-btn'),
      authError: document.getElementById('auth-error'),
      
      // Main app
      connectionStatus: document.getElementById('connection-status'),
      logoutBtn: document.getElementById('logout-btn'),
      settingsBtn: document.getElementById('settings-btn'),
      
      // Upload tabs
      tabBtns: document.querySelectorAll('.tab-btn'),
      tabContents: document.querySelectorAll('.tab-content'),
      
      // File upload
      fileInput: document.getElementById('file-input'),
      fileUploadArea: document.getElementById('file-upload-area'),
      uploadProgress: document.getElementById('upload-progress'),
      
      // Text upload
      textContent: document.getElementById('text-content'),
      uploadTextBtn: document.getElementById('upload-text-btn'),
      
      // Clipboard
      pasteBtn: document.getElementById('paste-btn'),
      clipboardPreview: document.getElementById('clipboard-preview'),
      shareClipboardBtn: document.getElementById('share-clipboard-btn'),
      clearClipboardBtn: document.getElementById('clear-clipboard-btn'),
      
      // Items list
      itemsList: document.getElementById('items-list'),
      emptyState: document.getElementById('empty-state'),
      refreshBtn: document.getElementById('refresh-btn'),
      contentFilter: document.getElementById('content-filter'),
      
      // Settings modal
      settingsModal: document.getElementById('settings-modal'),
      closeSettings: document.getElementById('close-settings'),
      
      // Toast container
      toastContainer: document.getElementById('toast-container')
    };
  }

  // Setup event listeners
  setupEventListeners() {
    // Auth form
    this.elements.loginBtn.addEventListener('click', () => this.handleLogin());
    this.elements.emailInput.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') this.handleLogin();
    });
    this.elements.passwordInput.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') this.handleLogin();
    });

    // Navigation
    this.elements.logoutBtn.addEventListener('click', () => this.handleLogout());
    this.elements.settingsBtn.addEventListener('click', () => this.toggleSettings());
    this.elements.closeSettings.addEventListener('click', () => this.toggleSettings());

    // Upload tabs
    this.elements.tabBtns.forEach(btn => {
      btn.addEventListener('click', () => this.switchTab(btn.dataset.tab));
    });

    // File upload
    this.elements.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
    this.elements.fileUploadArea.addEventListener('dragover', (e) => this.handleDragOver(e));
    this.elements.fileUploadArea.addEventListener('drop', (e) => this.handleDrop(e));
    this.elements.fileUploadArea.addEventListener('click', () => this.elements.fileInput.click());

    // Text upload
    this.elements.uploadTextBtn.addEventListener('click', () => this.handleTextUpload());

    // Clipboard
    this.elements.pasteBtn.addEventListener('click', () => this.handleClipboardPaste());
    this.elements.shareClipboardBtn.addEventListener('click', () => this.handleClipboardShare());
    this.elements.clearClipboardBtn.addEventListener('click', () => this.handleClipboardClear());

    // Items list
    this.elements.refreshBtn.addEventListener('click', () => this.refreshItems());
    this.elements.contentFilter.addEventListener('change', (e) => this.handleFilterChange(e));

    // Settings modal backdrop
    this.elements.settingsModal.addEventListener('click', (e) => {
      if (e.target === this.elements.settingsModal) {
        this.toggleSettings();
      }
    });
  }

  // Setup WebSocket message handlers
  setupWebSocketHandlers() {
    this.wsManager.setStatusChangeHandler((status) => {
      this.updateConnectionStatus(status);
    });

    this.wsManager.addMessageHandler('item_added', (message) => {
      this.handleItemAdded(message.item);
    });

    this.wsManager.addMessageHandler('item_deleted', (message) => {
      this.handleItemDeleted(message.item_id);
    });

    this.wsManager.addMessageHandler('item_updated', (message) => {
      this.handleItemUpdated(message.item);
    });
  }

  // Authentication
  async checkAuth() {
    this.showScreen('loading');
    
    const isAuthenticated = await this.authManager.checkAuth();
    
    if (isAuthenticated) {
      await this.initApp();
    } else {
      this.showScreen('auth');
    }
  }

  async handleLogin() {
    const email = this.elements.emailInput.value.trim();
    const password = this.elements.passwordInput.value;
    
    if (!email || !password) {
      this.showAuthError('Please enter both email and password');
      return;
    }

    this.elements.loginBtn.disabled = true;
    this.elements.loginBtn.textContent = 'Signing in...';
    
    try {
      const result = await this.authManager.login(email, password);
      
      if (result.success) {
        await this.initApp();
      } else {
        this.showAuthError(result.error);
      }
    } catch (error) {
      this.showAuthError(Utils.getErrorMessage(error));
    } finally {
      this.elements.loginBtn.disabled = false;
      this.elements.loginBtn.textContent = 'Sign In';
    }
  }

  handleLogout() {
    this.wsManager.disconnect();
    this.authManager.logout();
    this.showScreen('auth');
    this.clearAuthError();
  }

  // App initialization
  async initApp() {
    this.showScreen('main');
    this.clearAuthError();
    
    await this.loadSettings();
    await this.refreshItems();
    
    // Connect WebSocket
    this.wsManager.connect();
    
    // Register device
    try {
      await this.deviceManager.registerDevice(this.config.apiUrl, this.authManager.getToken());
    } catch (error) {
      console.warn('Device registration failed:', error);
    }
    
    // Setup periodic refresh
    this.setupPeriodicRefresh();
  }

  // Load settings from server
  async loadSettings() {
    try {
      const health = await this.apiClient.getHealthStatus();
      if (health.metrics) {
        this.settings.maxFileSize = (health.metrics.max_file_size_mb || 10) * 1024 * 1024;
        this.settings.defaultExpiry = health.metrics.default_expiry_hours || 24;
      }
    } catch (error) {
      console.warn('Failed to load settings:', error);
    }
  }

  // Setup periodic refresh of items
  setupPeriodicRefresh() {
    // Refresh items every 30 seconds
    setInterval(() => {
      if (this.authManager.isAuthenticated() && this.wsManager.getStatus() === 'online') {
        this.refreshItems();
      }
    }, 30000);
  }

  // Screen management
  showScreen(screenName) {
    this.elements.loadingScreen.classList.toggle('hidden', screenName !== 'loading');
    this.elements.authScreen.classList.toggle('hidden', screenName !== 'auth');
    this.elements.mainApp.classList.toggle('hidden', screenName !== 'main');
  }

  showAuthError(message) {
    this.elements.authError.textContent = message;
    this.elements.authError.classList.remove('hidden');
  }

  clearAuthError() {
    this.elements.authError.textContent = '';
    this.elements.authError.classList.add('hidden');
  }

  // Connection status
  updateConnectionStatus(status) {
    const indicator = this.elements.connectionStatus.querySelector('.status-indicator');
    const text = this.elements.connectionStatus.querySelector('span');
    
    indicator.className = 'status-indicator';
    
    switch (status) {
      case 'online':
        indicator.classList.add('online');
        text.textContent = 'Connected';
        break;
      case 'connecting':
        indicator.classList.add('connecting');
        text.textContent = 'Connecting...';
        break;
      case 'offline':
        indicator.classList.add('offline');
        text.textContent = 'Offline';
        break;
      case 'error':
        indicator.classList.add('error');
        text.textContent = 'Connection Error';
        break;
      default:
        indicator.classList.add('offline');
        text.textContent = 'Disconnected';
    }
  }

  // Tab management
  switchTab(tabName) {
    this.elements.tabBtns.forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tab === tabName);
    });
    
    this.elements.tabContents.forEach(content => {
      content.classList.toggle('active', content.dataset.tab === tabName);
    });
  }

  // Show toast notification
  showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    
    this.elements.toastContainer.appendChild(toast);
    
    // Animate in
    setTimeout(() => toast.classList.add('show'), 100);
    
    // Remove after 3 seconds
    setTimeout(() => {
      toast.classList.remove('show');
      setTimeout(() => {
        if (toast.parentNode) {
          this.elements.toastContainer.removeChild(toast);
        }
      }, 300);
    }, 3000);
  }

  // Settings modal
  toggleSettings() {
    this.elements.settingsModal.classList.toggle('hidden');
  }

  // Continue with implementation in next part...
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.app = new DeviceSyncApp();
});

// Export for module use
window.DeviceSyncApp = DeviceSyncApp;