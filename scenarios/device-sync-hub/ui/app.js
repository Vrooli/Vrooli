// Device Sync Hub - Main Application JavaScript

class DeviceSyncApp {
  constructor() {
    this.apiUrl = 'http://localhost:3300';
    this.wsUrl = 'ws://localhost:3300';
    this.authToken = localStorage.getItem('auth_token');
    this.user = null;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.syncItems = [];
    this.currentFilter = 'all';
    this.settings = {
      defaultExpiry: 24,
      maxFileSize: 10485760 // 10MB
    };

    this.init();
  }

  async init() {
    this.setupEventListeners();
    await this.checkAuth();
  }

  // Authentication
  async checkAuth() {
    this.showScreen('loading');
    
    if (!this.authToken) {
      this.showScreen('auth');
      return;
    }

    try {
      const response = await fetch(`http://localhost:3250/api/v1/auth/validate`, {
        headers: {
          'Authorization': `Bearer ${this.authToken}`
        }
      });

      const result = await response.json();
      
      if (result.valid) {
        this.user = {
          id: result.user_id,
          email: result.email,
          roles: result.roles || []
        };
        await this.initApp();
      } else {
        localStorage.removeItem('auth_token');
        this.authToken = null;
        this.showScreen('auth');
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      this.showToast('Authentication service unavailable', 'error');
      this.showScreen('auth');
    }
  }

  async login(email, password) {
    try {
      const response = await fetch(`http://localhost:3250/api/v1/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ email, password })
      });

      const result = await response.json();
      
      if (response.ok) {
        this.authToken = result.token;
        localStorage.setItem('auth_token', this.authToken);
        this.user = {
          id: result.user_id,
          email: result.email,
          roles: result.roles || []
        };
        await this.initApp();
      } else {
        this.showAuthError(result.error || 'Login failed');
      }
    } catch (error) {
      console.error('Login error:', error);
      this.showAuthError('Connection failed. Please try again.');
    }
  }

  logout() {
    localStorage.removeItem('auth_token');
    this.authToken = null;
    this.user = null;
    if (this.ws) {
      this.ws.close();
    }
    this.showScreen('auth');
  }

  // App initialization
  async initApp() {
    this.showScreen('main');
    await this.loadSettings();
    await this.loadSyncItems();
    this.initWebSocket();
    this.setupPeriodicRefresh();
  }

  // WebSocket connection
  initWebSocket() {
    if (!this.authToken) return;

    this.updateConnectionStatus('connecting');
    
    this.ws = new WebSocket(this.wsUrl + '/api/v1/sync/websocket');
    
    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.updateConnectionStatus('online');
      this.reconnectAttempts = 0;
      
      // Authenticate the WebSocket connection
      this.ws.send(JSON.stringify({
        type: 'auth',
        token: this.authToken,
        device_info: this.getDeviceInfo()
      }));
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.handleWebSocketMessage(message);
      } catch (error) {
        console.error('WebSocket message parse error:', error);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.updateConnectionStatus('offline');
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.updateConnectionStatus('offline');
    };
  }

  handleWebSocketMessage(message) {
    switch (message.type) {
      case 'auth_success':
        console.log('WebSocket authenticated');
        break;
      case 'auth_error':
        console.error('WebSocket auth failed:', message.message);
        this.showToast('Connection authentication failed', 'error');
        break;
      case 'item_added':
        this.handleItemAdded(message.item);
        break;
      case 'item_deleted':
        this.handleItemDeleted(message.item_id);
        break;
    }
  }

  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('Max reconnect attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    setTimeout(() => this.initWebSocket(), delay);
  }

  getDeviceInfo() {
    return {
      platform: navigator.platform,
      userAgent: navigator.userAgent,
      language: navigator.language,
      screen: {
        width: screen.width,
        height: screen.height
      },
      timestamp: new Date().toISOString()
    };
  }

  // Settings
  async loadSettings() {
    try {
      const response = await this.apiRequest('/api/v1/sync/settings');
      if (response.ok) {
        const settings = await response.json();
        this.settings = {
          ...this.settings,
          maxFileSize: settings.max_file_size,
          defaultExpiry: settings.default_expiry_hours
        };
        this.updateSettingsUI();
      }
    } catch (error) {
      console.error('Failed to load settings:', error);
    }
  }

  updateSettingsUI() {
    const maxFileSizeEl = document.getElementById('max-file-size');
    if (maxFileSizeEl) {
      maxFileSizeEl.textContent = this.formatFileSize(this.settings.maxFileSize);
    }

    const defaultExpiryEl = document.getElementById('default-expiry');
    if (defaultExpiryEl) {
      defaultExpiryEl.value = this.settings.defaultExpiry.toString();
    }
  }

  // Sync items management
  async loadSyncItems() {
    try {
      const response = await this.apiRequest('/api/v1/sync/items');
      if (response.ok) {
        const data = await response.json();
        this.syncItems = data.items;
        this.renderSyncItems();
      }
    } catch (error) {
      console.error('Failed to load sync items:', error);
      this.showToast('Failed to load items', 'error');
    }
  }

  async uploadFile(file, contentType = 'file') {
    if (file.size > this.settings.maxFileSize) {
      this.showToast(`File too large. Maximum size is ${this.formatFileSize(this.settings.maxFileSize)}`, 'error');
      return;
    }

    const formData = new FormData();
    formData.append('file', file);
    formData.append('content_type', contentType);
    formData.append('expires_in', this.settings.defaultExpiry.toString());

    try {
      this.showUploadProgress(true);
      
      const response = await this.apiRequest('/api/v1/sync/upload', {
        method: 'POST',
        body: formData
      });

      if (response.ok) {
        const result = await response.json();
        this.showToast(`${file.name} uploaded successfully`, 'success');
        await this.loadSyncItems(); // Refresh the list
      } else {
        const error = await response.json();
        this.showToast(error.error || 'Upload failed', 'error');
      }
    } catch (error) {
      console.error('Upload error:', error);
      this.showToast('Upload failed', 'error');
    } finally {
      this.showUploadProgress(false);
    }
  }

  async uploadText(text, contentType = 'text') {
    if (!text.trim()) {
      this.showToast('Please enter some text', 'error');
      return;
    }

    try {
      const response = await this.apiRequest('/api/v1/sync/upload', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          text: text,
          content_type: contentType,
          expires_in: this.settings.defaultExpiry
        })
      });

      if (response.ok) {
        const result = await response.json();
        this.showToast('Text shared successfully', 'success');
        await this.loadSyncItems();
        
        // Clear text input
        const textInput = document.getElementById('text-content');
        if (textInput) textInput.value = '';
        
        // Hide clipboard preview
        this.hideClipboardPreview();
      } else {
        const error = await response.json();
        this.showToast(error.error || 'Share failed', 'error');
      }
    } catch (error) {
      console.error('Text upload error:', error);
      this.showToast('Share failed', 'error');
    }
  }

  async deleteItem(itemId) {
    try {
      const response = await this.apiRequest(`/api/v1/sync/items/${itemId}`, {
        method: 'DELETE'
      });

      if (response.ok) {
        this.showToast('Item deleted', 'success');
        await this.loadSyncItems();
      } else {
        const error = await response.json();
        this.showToast(error.error || 'Delete failed', 'error');
      }
    } catch (error) {
      console.error('Delete error:', error);
      this.showToast('Delete failed', 'error');
    }
  }

  downloadItem(itemId, filename) {
    const url = `${this.apiUrl}/api/v1/sync/items/${itemId}/download`;
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    
    // Add authentication header by creating a temporary form
    const form = document.createElement('form');
    form.method = 'GET';
    form.action = url;
    form.style.display = 'none';
    
    const authInput = document.createElement('input');
    authInput.type = 'hidden';
    authInput.name = 'token';
    authInput.value = this.authToken;
    
    form.appendChild(authInput);
    document.body.appendChild(form);
    form.submit();
    document.body.removeChild(form);
  }

  // WebSocket event handlers
  handleItemAdded(item) {
    this.syncItems.unshift(item);
    this.renderSyncItems();
    this.showToast('New item synced from another device', 'success');
  }

  handleItemDeleted(itemId) {
    this.syncItems = this.syncItems.filter(item => item.id !== itemId);
    this.renderSyncItems();
    this.showToast('Item deleted on another device', 'warning');
  }

  // UI rendering
  renderSyncItems() {
    const container = document.getElementById('items-list');
    const emptyState = document.getElementById('empty-state');
    
    if (!container || !emptyState) return;

    const filteredItems = this.filterItems(this.syncItems);
    
    if (filteredItems.length === 0) {
      container.innerHTML = '';
      container.appendChild(emptyState);
      emptyState.classList.remove('hidden');
      return;
    }
    
    emptyState.classList.add('hidden');
    
    const itemsHtml = filteredItems.map(item => this.renderSyncItem(item)).join('');
    container.innerHTML = itemsHtml;
    
    // Add event listeners for action buttons
    container.querySelectorAll('.download-btn').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const itemId = btn.dataset.itemId;
        const filename = btn.dataset.filename;
        this.downloadItem(itemId, filename);
      });
    });
    
    container.querySelectorAll('.delete-btn').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const itemId = btn.dataset.itemId;
        if (confirm('Delete this item?')) {
          this.deleteItem(itemId);
        }
      });
    });

    container.querySelectorAll('.copy-btn').forEach(btn => {
      btn.addEventListener('click', async (e) => {
        e.stopPropagation();
        const text = btn.dataset.text;
        try {
          await navigator.clipboard.writeText(text);
          this.showToast('Copied to clipboard', 'success');
        } catch (error) {
          console.error('Copy failed:', error);
          this.showToast('Copy failed', 'error');
        }
      });
    });
  }

  renderSyncItem(item) {
    const expiresAt = new Date(item.expires_at);
    const now = new Date();
    const hoursUntilExpiry = (expiresAt - now) / (1000 * 60 * 60);
    
    let expiryStatus = 'active';
    let expiryText = this.formatTimeRemaining(expiresAt);
    
    if (hoursUntilExpiry <= 1) {
      expiryStatus = 'expires-soon';
      expiryText = 'Expires soon';
    }

    const thumbnail = this.renderItemThumbnail(item);
    const actions = this.renderItemActions(item);
    const fileSize = item.file_size ? this.formatFileSize(item.file_size) : '';

    return `
      <div class="item-card" data-item-id="${item.id}">
        ${thumbnail}
        <div class="item-info">
          <div class="item-name">${this.escapeHtml(item.filename)}</div>
          <div class="item-meta">
            <span class="expiry-badge ${expiryStatus}">${expiryText}</span>
            ${fileSize ? `<span class="separator">•</span><span>${fileSize}</span>` : ''}
            <span class="separator">•</span>
            <span>${this.formatDate(item.created_at)}</span>
          </div>
        </div>
        <div class="item-actions">
          ${actions}
        </div>
      </div>
    `;
  }

  renderItemThumbnail(item) {
    if (item.thumbnail_url) {
      return `
        <div class="item-thumbnail">
          <img src="${this.apiUrl}${item.thumbnail_url}" alt="Thumbnail" loading="lazy">
        </div>
      `;
    }

    const icon = this.getFileTypeIcon(item.mime_type, item.content_type);
    return `
      <div class="item-thumbnail">
        ${icon}
      </div>
    `;
  }

  renderItemActions(item) {
    let actions = [];
    
    if (item.content_type === 'text' || item.content_type === 'clipboard') {
      actions.push(`
        <button class="btn btn-secondary copy-btn" data-text="${this.escapeHtml(item.metadata?.text_preview || '')}" title="Copy to clipboard">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
          </svg>
        </button>
      `);
    }
    
    actions.push(`
      <button class="btn btn-secondary download-btn" data-item-id="${item.id}" data-filename="${this.escapeHtml(item.filename)}" title="Download">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2z"/>
        </svg>
      </button>
    `);
    
    actions.push(`
      <button class="btn btn-secondary delete-btn" data-item-id="${item.id}" title="Delete">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
        </svg>
      </button>
    `);
    
    return actions.join('');
  }

  getFileTypeIcon(mimeType, contentType) {
    if (contentType === 'clipboard') {
      return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>`;
    }
    
    if (contentType === 'text') {
      return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>`;
    }
    
    if (mimeType.startsWith('image/')) {
      return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>`;
    }
    
    if (mimeType.startsWith('video/')) {
      return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>`;
    }
    
    if (mimeType.startsWith('audio/')) {
      return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"/></svg>`;
    }
    
    return `<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>`;
  }

  // Event listeners
  setupEventListeners() {
    // Auth form
    const loginBtn = document.getElementById('login-btn');
    const emailInput = document.getElementById('email');
    const passwordInput = document.getElementById('password');
    
    if (loginBtn && emailInput && passwordInput) {
      loginBtn.addEventListener('click', () => {
        const email = emailInput.value.trim();
        const password = passwordInput.value;
        
        if (!email || !password) {
          this.showAuthError('Please enter both email and password');
          return;
        }
        
        this.login(email, password);
      });
      
      // Enter key support
      [emailInput, passwordInput].forEach(input => {
        input.addEventListener('keypress', (e) => {
          if (e.key === 'Enter') {
            loginBtn.click();
          }
        });
      });
    }

    // Logout button
    const logoutBtn = document.getElementById('logout-btn');
    if (logoutBtn) {
      logoutBtn.addEventListener('click', () => this.logout());
    }

    // Tab switching
    const tabBtns = document.querySelectorAll('.tab-btn');
    tabBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        const tabName = btn.dataset.tab;
        this.switchTab(tabName);
      });
    });

    // File upload
    const fileInput = document.getElementById('file-input');
    const fileUploadArea = document.getElementById('file-upload-area');
    
    if (fileInput && fileUploadArea) {
      fileInput.addEventListener('change', (e) => {
        const files = Array.from(e.target.files);
        files.forEach(file => this.uploadFile(file));
        fileInput.value = ''; // Clear input
      });

      // Drag and drop
      fileUploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        fileUploadArea.classList.add('dragover');
      });

      fileUploadArea.addEventListener('dragleave', (e) => {
        e.preventDefault();
        fileUploadArea.classList.remove('dragover');
      });

      fileUploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        fileUploadArea.classList.remove('dragover');
        
        const files = Array.from(e.dataTransfer.files);
        files.forEach(file => this.uploadFile(file));
      });
    }

    // Text upload
    const uploadTextBtn = document.getElementById('upload-text-btn');
    const textContent = document.getElementById('text-content');
    
    if (uploadTextBtn && textContent) {
      uploadTextBtn.addEventListener('click', () => {
        const text = textContent.value.trim();
        this.uploadText(text, 'text');
      });
    }

    // Clipboard functionality
    const pasteBtn = document.getElementById('paste-btn');
    if (pasteBtn) {
      pasteBtn.addEventListener('click', async () => {
        try {
          const text = await navigator.clipboard.readText();
          this.showClipboardPreview(text);
        } catch (error) {
          console.error('Clipboard read failed:', error);
          this.showToast('Clipboard access denied or empty', 'error');
        }
      });
    }

    const shareClipboardBtn = document.getElementById('share-clipboard-btn');
    if (shareClipboardBtn) {
      shareClipboardBtn.addEventListener('click', () => {
        const previewEl = document.querySelector('#clipboard-preview .preview-content');
        if (previewEl) {
          const text = previewEl.textContent;
          this.uploadText(text, 'clipboard');
        }
      });
    }

    const clearClipboardBtn = document.getElementById('clear-clipboard-btn');
    if (clearClipboardBtn) {
      clearClipboardBtn.addEventListener('click', () => {
        this.hideClipboardPreview();
      });
    }

    // Settings modal
    const settingsBtn = document.getElementById('settings-btn');
    const settingsModal = document.getElementById('settings-modal');
    const closeSettings = document.getElementById('close-settings');
    
    if (settingsBtn && settingsModal) {
      settingsBtn.addEventListener('click', () => {
        settingsModal.classList.remove('hidden');
      });
    }
    
    if (closeSettings && settingsModal) {
      closeSettings.addEventListener('click', () => {
        settingsModal.classList.add('hidden');
      });
    }
    
    if (settingsModal) {
      settingsModal.addEventListener('click', (e) => {
        if (e.target === settingsModal) {
          settingsModal.classList.add('hidden');
        }
      });
    }

    // Refresh button
    const refreshBtn = document.getElementById('refresh-btn');
    if (refreshBtn) {
      refreshBtn.addEventListener('click', () => this.loadSyncItems());
    }

    // Content filter
    const contentFilter = document.getElementById('content-filter');
    if (contentFilter) {
      contentFilter.addEventListener('change', (e) => {
        this.currentFilter = e.target.value;
        this.renderSyncItems();
      });
    }

    // Default expiry setting
    const defaultExpiry = document.getElementById('default-expiry');
    if (defaultExpiry) {
      defaultExpiry.addEventListener('change', (e) => {
        this.settings.defaultExpiry = parseInt(e.target.value);
        localStorage.setItem('sync_hub_settings', JSON.stringify(this.settings));
      });
    }
  }

  // UI helper methods
  switchTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tab === tabName);
    });

    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => {
      content.classList.toggle('active', content.dataset.tab === tabName);
    });
  }

  showScreen(screenName) {
    const screens = ['loading', 'auth', 'main'];
    screens.forEach(screen => {
      const element = document.getElementById(`${screen}-screen`);
      if (screen === 'main') {
        const element = document.getElementById('main-app');
        if (element) {
          element.classList.toggle('hidden', screenName !== 'main');
        }
      } else {
        if (element) {
          element.classList.toggle('hidden', screenName !== screen);
        }
      }
    });
  }

  showAuthError(message) {
    const errorEl = document.getElementById('auth-error');
    if (errorEl) {
      errorEl.textContent = message;
      errorEl.classList.remove('hidden');
    }
  }

  showUploadProgress(show) {
    const progressEl = document.getElementById('upload-progress');
    if (progressEl) {
      progressEl.classList.toggle('hidden', !show);
    }
  }

  showClipboardPreview(text) {
    const previewEl = document.getElementById('clipboard-preview');
    const contentEl = previewEl?.querySelector('.preview-content');
    
    if (previewEl && contentEl) {
      contentEl.textContent = text.substring(0, 500); // Limit preview
      previewEl.classList.remove('hidden');
    }
  }

  hideClipboardPreview() {
    const previewEl = document.getElementById('clipboard-preview');
    if (previewEl) {
      previewEl.classList.add('hidden');
    }
  }

  updateConnectionStatus(status) {
    const statusEl = document.getElementById('connection-status');
    const indicator = statusEl?.querySelector('.status-indicator');
    const text = statusEl?.querySelector('span');
    
    if (indicator && text) {
      indicator.className = `status-indicator ${status}`;
      
      switch (status) {
        case 'online':
          text.textContent = 'Connected';
          break;
        case 'connecting':
          text.textContent = 'Connecting...';
          break;
        case 'offline':
          text.textContent = 'Offline';
          break;
      }
    }

    // Update settings modal status
    const wsStatusEl = document.getElementById('ws-status');
    if (wsStatusEl) {
      wsStatusEl.textContent = status === 'online' ? 'Connected' : 'Disconnected';
      wsStatusEl.className = `status-badge ${status === 'online' ? 'success' : 'error'}`;
    }
  }

  showToast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;

    container.appendChild(toast);

    // Auto remove after 4 seconds
    setTimeout(() => {
      if (toast.parentNode) {
        toast.parentNode.removeChild(toast);
      }
    }, 4000);
  }

  // Utility methods
  async apiRequest(endpoint, options = {}) {
    const url = this.apiUrl + endpoint;
    const defaultOptions = {
      headers: {
        'Authorization': `Bearer ${this.authToken}`,
        ...options.headers
      }
    };

    return fetch(url, { ...defaultOptions, ...options });
  }

  filterItems(items) {
    if (this.currentFilter === 'all') return items;
    return items.filter(item => item.content_type === this.currentFilter);
  }

  formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString();
  }

  formatTimeRemaining(expiresAt) {
    const now = new Date();
    const diffMs = expiresAt - now;
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMs < 0) return 'Expired';
    if (diffHours < 1) return 'Expires soon';
    if (diffHours < 24) return `${diffHours}h remaining`;
    if (diffDays < 7) return `${diffDays}d remaining`;
    
    return expiresAt.toLocaleDateString();
  }

  escapeHtml(unsafe) {
    return unsafe
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  setupPeriodicRefresh() {
    // Refresh items every 30 seconds
    setInterval(() => {
      if (this.user && document.visibilityState === 'visible') {
        this.loadSyncItems();
      }
    }, 30000);
  }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.syncApp = new DeviceSyncApp();
});