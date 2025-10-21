/**
 * Home Automation Intelligence - Frontend Application
 * Self-evolving home automation with AI-generated automations
 */

class HomeAutomationApp {
    constructor() {
        this.apiBaseUrl = this.resolveApiBaseUrl();
        this.wsUrl = this.resolveWebSocketUrl();
        this.ws = null;
        this.currentUser = null;
        this.devices = new Map();
        this.automations = new Map();
        this.scenes = new Map();
        this.iconRetryAttempts = 0;
        this.pendingIconRefresh = null;
        
        this.init();
    }
    
    init() {
        console.log('Initializing Home Automation Intelligence...');
        
        // Initialize UI components
        this.initializeTabNavigation();
        this.initializeWebSocket();
        this.initializeEventListeners();
        
        // Load initial data
        this.loadUserProfile();
        this.loadDevices();
        this.loadScenes();
        this.loadAutomations();
        
        this.refreshIcons();

        console.log('✅ Home Automation App initialized');
    }
    
    // Tab Navigation
    initializeTabNavigation() {
        const tabButtons = document.querySelectorAll('.tab-button');
        const tabContents = document.querySelectorAll('.tab-content');
        
        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                const targetTab = button.dataset.tab;
                
                // Update active tab button
                tabButtons.forEach(btn => btn.classList.remove('active'));
                button.classList.add('active');
                
                // Update active tab content
                tabContents.forEach(content => {
                    content.classList.remove('active');
                    if (content.id === `${targetTab}-tab`) {
                        content.classList.add('active');
                    }
                });
                
                // Trigger tab-specific loading
                this.onTabChanged(targetTab);
            });
        });
    }
    
    // WebSocket Connection
    initializeWebSocket() {
        try {
            this.ws = new WebSocket(this.wsUrl);
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.updateConnectionStatus('online');
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleWebSocketMessage(data);
                } catch (error) {
                    console.error('WebSocket message parsing error:', error);
                }
            };
            
            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.updateConnectionStatus('offline');
                // Attempt to reconnect after 5 seconds
                setTimeout(() => this.initializeWebSocket(), 5000);
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('offline');
            };
        } catch (error) {
            console.error('Failed to initialize WebSocket:', error);
            this.updateConnectionStatus('offline');
        }
    }
    
    // Event Listeners
    initializeEventListeners() {
        // Refresh devices
        document.getElementById('refreshDevices')?.addEventListener('click', () => {
            this.loadDevices();
        });
        
        // Device filter
        document.getElementById('deviceFilter')?.addEventListener('change', (e) => {
            this.filterDevices(e.target.value);
        });
        
        // Create automation
        document.getElementById('createAutomation')?.addEventListener('click', () => {
            this.showAutomationCreator();
        });
        
        // Close automation creator
        document.getElementById('closeCreator')?.addEventListener('click', () => {
            this.hideAutomationCreator();
        });
        
        // Cancel automation creation
        document.getElementById('cancelCreation')?.addEventListener('click', () => {
            this.hideAutomationCreator();
        });
        
        // Generate automation
        document.getElementById('generateAutomation')?.addEventListener('click', () => {
            this.generateAutomation();
        });
        
        // Settings buttons
        document.getElementById('manageProfiles')?.addEventListener('click', () => {
            this.showToast('Profile management coming soon', 'info');
        });
        
        document.getElementById('securitySettings')?.addEventListener('click', () => {
            this.showToast('Security settings coming soon', 'info');
        });
        
        document.getElementById('calendarSettings')?.addEventListener('click', () => {
            this.showToast('Calendar integration coming soon', 'info');
        });
    }
    
    resolveApiBaseUrl() {
        const ensureVersionedBase = (candidate) => {
            if (typeof candidate !== 'string') {
                return '';
            }

            const trimmed = candidate.trim();
            if (!trimmed) {
                return '';
            }

            const normalized = trimmed.replace(/\/+$/, '');
            if (/\/api$/i.test(normalized)) {
                return `${normalized}/v1`;
            }

            return normalized;
        };

        if (typeof window !== 'undefined') {
            const injected = window.__HOME_AUTOMATION_API_BASE__;
            if (typeof injected === 'string' && injected.trim()) {
                return ensureVersionedBase(injected);
            }
        }

        const meta = document.querySelector('meta[name="home-automation-api-base"]');
        if (meta?.content?.trim()) {
            const metaContent = meta.content.trim();
            try {
                return ensureVersionedBase(new URL(metaContent, window.location.href).toString());
            } catch (error) {
                console.warn('Invalid meta API base URL, falling back to relative base:', error);
                return ensureVersionedBase(metaContent);
            }
        }

        return ensureVersionedBase(new URL('./api/v1/', window.location.href).toString());
    }

    resolveWebSocketUrl() {
        if (typeof window !== 'undefined') {
            const injected = window.__HOME_AUTOMATION_WS_BASE__;
            if (typeof injected === 'string' && injected.trim()) {
                return injected.trim();
            }
        }

        const meta = document.querySelector('meta[name="home-automation-ws-base"]');
        if (meta?.content?.trim()) {
            const metaContent = meta.content.trim();
            try {
                return new URL(metaContent, window.location.href).toString();
            } catch (error) {
                console.warn('Invalid meta WS base URL, falling back to location-derived base:', error);
                return metaContent;
            }
        }

        try {
            const apiUrl = new URL(`${this.apiBaseUrl}/`);
            apiUrl.protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:';
            if (apiUrl.pathname.endsWith('/api/')) {
                apiUrl.pathname = apiUrl.pathname.replace(/\/api\/$/, '/');
            }
            return apiUrl.toString();
        } catch (error) {
            const locationUrl = new URL(window.location.href);
            locationUrl.protocol = locationUrl.protocol === 'https:' ? 'wss:' : 'ws:';
            return locationUrl.toString();
        }
    }

    buildApiUrl(endpoint = '/') {
        const sanitizedEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
        try {
            const base = new URL(`${this.apiBaseUrl}/`);
            return new URL(sanitizedEndpoint, base).toString();
        } catch (error) {
            console.error('Failed to build API URL, falling back to location-derived path:', error);
            return new URL(sanitizedEndpoint, window.location.href).toString();
        }
    }

    // API Calls
    async apiCall(endpoint, options = {}) {
        try {
            const response = await fetch(this.buildApiUrl(endpoint), {
                ...options,
                headers: {
                    ...this.buildApiHeaders(),
                    ...(options.headers || {})
                }
            });

            if (!response.ok) {
                throw new Error(`API call failed: ${response.status} ${response.statusText}`);
            }

            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('content-type') || '';
            if (contentType.includes('application/json')) {
                return await response.json();
            }

            return await response.text();
        } catch (error) {
            console.error(`API call to ${endpoint} failed:`, error);
            throw error;
        }
    }
    
    // User Profile
    async loadUserProfile() {
        try {
            // For now, use a mock user profile
            // In implementation, this would call scenario-authenticator
            this.currentUser = {
                id: 'mock-user-id',
                name: 'Demo User',
                permissions: {
                    devices: ['*'],
                    scenes: ['*'],
                    automation_create: true,
                    admin_access: true
                }
            };
            
            document.querySelector('.profile-name').textContent = this.currentUser.name;
            
            console.log('User profile loaded:', this.currentUser);
        } catch (error) {
            console.error('Failed to load user profile:', error);
            this.showToast('Failed to load user profile', 'error');
        }
    }
    
    // Devices Management
    async loadDevices() {
        try {
            this.showLoading('Loading devices...');

            const payload = await this.apiCall('/devices');
            const apiDevices = Array.isArray(payload.devices) ? payload.devices : [];

            this.devices.clear();
            apiDevices.forEach(rawDevice => {
                const normalized = this.normalizeDevice(rawDevice);
                if (normalized) {
                    this.devices.set(normalized.id, normalized);
                }
            });

            this.renderDevices();
            this.hideLoading();

            console.log('Devices loaded:', this.devices.size);
        } catch (error) {
            console.error('Failed to load devices:', error);
            this.showToast(`Failed to load devices: ${error.message}`, 'error');
            this.hideLoading();
        }
    }
    
    renderDevices() {
        const devicesGrid = document.getElementById('devicesGrid');
        if (!devicesGrid) return;
        
        devicesGrid.innerHTML = '';
        
        if (this.devices.size === 0) {
            devicesGrid.innerHTML = `
                <div class="device-placeholder">
                    <div class="placeholder-icon"><span class="icon" data-lucide="search"></span></div>
                    <p>No devices found</p>
                    <small>Check Home Assistant connection</small>
                </div>
            `;
            this.refreshIcons();
            return;
        }
        
        this.devices.forEach(device => {
            const deviceCard = this.createDeviceCard(device);
            devicesGrid.appendChild(deviceCard);
        });

        this.refreshIcons();
    }

    buildApiHeaders() {
        const headers = {
            'Content-Type': 'application/json'
        };

        const token = this.getAuthToken();
        if (token) {
            headers['Authorization'] = token;
        }

        return headers;
    }

    normalizeDevice(raw) {
        if (!raw) {
            return null;
        }

        const id = raw.device_id || raw.id;
        if (!id) {
            return null;
        }

        const domain = (raw.type || (id.includes('.') ? id.split('.')[0] : 'unknown')).toLowerCase();
        const state = this.normalizeDeviceState(domain, raw.state || {});

        return {
            id,
            name: raw.name || id,
            type: domain,
            domain,
            state,
            rawState: raw.state || {},
            available: raw.available !== false,
            lastUpdated: raw.last_updated || raw.lastUpdated || null,
            user_can_control: this.canControlDomain(domain)
        };
    }

    normalizeDeviceState(domain, state) {
        const normalized = { ...state };

        const rawState = state.raw_state ?? state.state;

        if (typeof normalized.on !== 'boolean' && rawState) {
            normalized.on = rawState === 'on' || rawState === 'heat' || rawState === 'cool';
        }

        if (typeof normalized.active !== 'boolean') {
            normalized.active = normalized.on === true || normalized.locked === true || rawState === 'locked';
        }

        if (domain === 'light' || domain === 'switch' || domain === 'fan') {
            if (typeof normalized.brightness === 'number' && typeof normalized.brightness_pct !== 'number') {
                normalized.brightness_pct = Math.round((normalized.brightness / 255) * 100);
            }
        }

        if (domain === 'sensor' || domain === 'binary_sensor') {
            if (normalized.value === undefined && rawState !== undefined) {
                normalized.value = rawState;
            }
            if (!normalized.unit && state.unit_of_measurement) {
                normalized.unit = state.unit_of_measurement;
            }
        }

        if (domain === 'climate') {
            normalized.mode = normalized.mode || rawState;
            normalized.temperature = normalized.temperature ?? state.temperature ?? state.current_temperature;
            normalized.target_temperature = normalized.target_temperature ?? normalized.target ?? state.target_temperature ?? state.target_temp;
        }

        if (domain === 'lock') {
            normalized.locked = normalized.locked ?? rawState === 'locked';
        }

        normalized.raw_state = rawState;
        return normalized;
    }

    canControlDomain(domain) {
        const controllable = new Set(['light', 'switch', 'climate', 'fan', 'lock', 'cover', 'humidifier']);
        return controllable.has(domain);
    }
    
    createDeviceCard(device) {
        const card = document.createElement('div');
        card.className = 'device-card';
        card.dataset.deviceId = device.id;
        
        const isActive = device.state?.active ?? device.state?.on ?? false;
        const statusClass = device.available
            ? (isActive ? 'on' : 'off')
            : 'unavailable';

        const controls = device.user_can_control 
            ? this.createDeviceControls(device)
            : '<p style="color: var(--text-tertiary); font-size: 0.8rem;">Read-only</p>';

        card.innerHTML = `
            <div class="device-header">
                <div class="device-info">
                    <h3>${device.name}</h3>
                    <div class="device-type">${device.domain || device.type}</div>
                </div>
                <div class="device-status ${statusClass}"></div>
            </div>
            <div class="device-state">
                ${this.formatDeviceState(device.state, device)}
            </div>
            <div class="device-controls">
                ${controls}
            </div>
        `;
        
        return card;
    }
    
    createDeviceControls(device) {
        const domain = device.domain || device.type;

        if (!this.canControlDomain(domain)) {
            return `
                <button class="btn device-control btn-secondary" 
                        onclick="app.refreshDevice('${device.id}')">
                    <span class="icon" data-lucide="refresh-cw"></span>
                    <span>Refresh</span>
                </button>
            `;
        }

        const isOn = device.state?.on === true || device.state?.active === true;
        const icon = isOn ? 'power-off' : 'power';
        const label = isOn ? 'Turn Off' : 'Turn On';

        return `
            <button class="btn device-control ${isOn ? 'btn-secondary' : 'btn-primary'}" 
                    onclick="app.toggleDevice('${device.id}')">
                <span class="icon" data-lucide="${icon}"></span>
                <span>${label}</span>
            </button>
        `;
    }

    formatDeviceState(state, device) {
        const domain = device?.domain || device?.type;

        if (domain === 'climate') {
            const temp = state?.temperature ?? state?.target_temperature;
            const mode = state?.mode ?? state?.raw_state;
            return `
                <div style="display:flex; flex-direction:column; gap:4px;">
                    <div style="color: var(--accent-secondary); font-weight:600;">${temp ?? '--'}°</div>
                    <div style="color: var(--text-tertiary); font-size:0.8rem; text-transform:capitalize;">${mode || 'off'}</div>
                </div>
            `;
        }

        if (domain === 'sensor' || domain === 'binary_sensor') {
            const value = state?.value ?? state?.raw_state;
            const unit = state?.unit ? ` ${state.unit}` : '';
            return `<div style="color: var(--accent-secondary); font-weight:600;">${value ?? '--'}${unit}</div>`;
        }

        if (state?.brightness_pct !== undefined) {
            return `<div style="color: var(--text-secondary); font-size: 0.875rem;">Brightness: ${Math.round(state.brightness_pct)}%</div>`;
        }

        if (state?.on !== undefined) {
            return `<div style="color: ${state.on ? 'var(--device-on)' : 'var(--device-off)'}; font-weight: 500;">
                ${state.on ? 'On' : 'Off'}
            </div>`;
        }

        if (state?.locked !== undefined) {
            return `<div style="color: ${state.locked ? 'var(--device-on)' : 'var(--device-off)'}; font-weight: 500;">
                ${state.locked ? 'Locked' : 'Unlocked'}
            </div>`;
        }

        return '<div style="color: var(--text-tertiary); font-size: 0.875rem;">Ready</div>';
    }
    
    async toggleDevice(deviceId) {
        const device = this.devices.get(deviceId);
        if (!device) return;

        try {
            this.showLoading(`Updating ${device.name}...`);

            const { action, parameters } = this.getToggleCommand(device);

            const result = await this.apiCall('/devices/control', {
                method: 'POST',
                body: JSON.stringify({
                    device_id: device.id,
                    action,
                    parameters: parameters || {}
                })
            });
            const updatedState = result.device_state || {};
            const normalized = this.normalizeDevice({ ...device, state: updatedState, available: result.success !== false });

            this.devices.set(device.id, normalized);
            this.renderDevices();
            this.showToast(`${device.name} updated successfully`, 'success');
        } catch (error) {
            console.error('Failed to toggle device:', error);
            this.showToast(`Failed to control ${device.name}: ${error.message}`, 'error');
        } finally {
            this.hideLoading();
        }
    }

    getToggleCommand(device) {
        const domain = device.domain || device.type;
        const state = device.state || {};

        switch (domain) {
            case 'light':
            case 'switch':
            case 'fan':
            case 'humidifier':
                return { action: state.on ? 'turn_off' : 'turn_on' };
            case 'lock':
                return { action: state.locked ? 'turn_off' : 'turn_on' };
            case 'cover':
                return { action: state.open ? 'turn_off' : 'turn_on' };
            case 'climate': {
                const isOn = state.mode && state.mode !== 'off';
                const nextMode = isOn ? 'off' : (state.mode && state.mode !== 'off' ? state.mode : 'cool');
                return {
                    action: 'set_mode',
                    parameters: { mode: nextMode }
                };
            }
            default:
                return { action: state.on ? 'turn_off' : 'turn_on' };
        }
    }
    
    async refreshDevice(deviceId) {
        const device = this.devices.get(deviceId);
        if (!device) return;

        try {
            this.showLoading(`Refreshing ${device.name}...`);

            const status = await this.apiCall(`/devices/${encodeURIComponent(deviceId)}/status`);
            const normalized = this.normalizeDevice(status);
            if (normalized) {
                this.devices.set(normalized.id, normalized);
                this.renderDevices();
            }

            this.showToast(`${device.name} refreshed`, 'success');
        } catch (error) {
            console.error('Failed to refresh device:', error);
            this.showToast(`Failed to refresh ${device.name}: ${error.message}`, 'error');
        } finally {
            this.hideLoading();
        }
    }
    
    filterDevices(filter) {
        const deviceCards = document.querySelectorAll('.device-card');
        
        const filterMap = {
            lights: 'light',
            sensors: 'sensor',
            switches: 'switch',
            climate: 'climate'
        };

        const normalizedFilter = filterMap[filter] || filter;

        deviceCards.forEach(card => {
            const deviceId = card.dataset.deviceId;
            const device = this.devices.get(deviceId);

            if (filter === 'all') {
                card.style.display = 'block';
                return;
            }

            const domain = device?.domain || device?.type;

            if (normalizedFilter === domain) {
                card.style.display = 'block';
            } else {
                card.style.display = 'none';
            }
        });
    }
    
    // Scenes Management
    async loadScenes() {
        try {
            // Mock scenes - in implementation, this would load from API
            this.scenes.clear();
            console.log('Scenes loaded');
        } catch (error) {
            console.error('Failed to load scenes:', error);
        }
    }
    
    // Automations Management
    async loadAutomations() {
        try {
            // Mock automations - in implementation, this would load from API
            this.automations.clear();
            console.log('Automations loaded');
        } catch (error) {
            console.error('Failed to load automations:', error);
        }
    }
    
    showAutomationCreator() {
        const creator = document.getElementById('automationCreator');
        if (creator) {
            creator.style.display = 'block';
            document.getElementById('automationDescription')?.focus();
        }
    }
    
    hideAutomationCreator() {
        const creator = document.getElementById('automationCreator');
        if (creator) {
            creator.style.display = 'none';
            document.getElementById('automationDescription').value = '';
        }
    }
    
    async generateAutomation() {
        try {
            const description = document.getElementById('automationDescription')?.value?.trim();
            if (!description) {
                this.showToast('Please describe the automation you want to create', 'warning');
                return;
            }
            
            this.showLoading('Generating automation with AI...');
            
            // Mock AI generation - in implementation, this would call Claude Code
            setTimeout(() => {
                this.hideLoading();
                this.hideAutomationCreator();
                this.showToast('AI automation generated successfully!', 'success');
                
                // In implementation, would show generated code for review
                console.log('Generated automation for:', description);
            }, 3000);
            
        } catch (error) {
            console.error('Failed to generate automation:', error);
            this.showToast('Failed to generate automation', 'error');
            this.hideLoading();
        }
    }
    
    // WebSocket Message Handling
    handleWebSocketMessage(data) {
        console.log('WebSocket message:', data);
        
        switch (data.type) {
            case 'device_state_changed':
                this.handleDeviceStateUpdate(data);
                break;
            case 'automation_executed':
                this.handleAutomationExecution(data);
                break;
            case 'connection':
                console.log('WebSocket connection confirmed');
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }
    
    handleDeviceStateUpdate(data) {
        const device = this.devices.get(data.device_id);
        if (device) {
            device.state = { ...device.state, ...data.new_state };
            this.renderDevices();
            console.log('Device state updated:', data.device_id);
        }
    }
    
    handleAutomationExecution(data) {
        this.showToast(`Automation executed: ${data.automation_id}`, 'info');
    }
    
    // UI Utilities
    updateConnectionStatus(status) {
        const statusIndicator = document.querySelector('.status-indicator');
        const statusText = statusIndicator?.nextElementSibling;
        
        if (statusIndicator) {
            statusIndicator.className = `status-indicator ${status}`;
        }
        
        if (statusText) {
            switch (status) {
                case 'online':
                    statusText.textContent = 'Connected';
                    break;
                case 'offline':
                    statusText.textContent = 'Disconnected';
                    break;
                case 'connecting':
                    statusText.textContent = 'Connecting...';
                    break;
                default:
                    statusText.textContent = 'Unknown';
            }
        }
    }
    
    showLoading(message = 'Loading...') {
        const overlay = document.getElementById('loadingOverlay');
        const messageEl = document.getElementById('loadingMessage');
        
        if (overlay) {
            overlay.style.display = 'flex';
        }
        
        if (messageEl) {
            messageEl.textContent = message;
        }
    }
    
    hideLoading() {
        const overlay = document.getElementById('loadingOverlay');
        if (overlay) {
            overlay.style.display = 'none';
        }
    }
    
    showToast(message, type = 'info') {
        const container = document.getElementById('toastContainer');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        const content = document.createElement('div');
        content.className = 'toast-content';

        const iconWrapper = document.createElement('div');
        iconWrapper.className = 'toast-icon';
        const iconName = {
            success: 'check-circle-2',
            error: 'alert-triangle',
            warning: 'alert-circle',
            info: 'info'
        }[type] || 'info';
        const iconSpan = document.createElement('span');
        iconSpan.className = 'icon';
        iconSpan.setAttribute('data-lucide', iconName);
        iconWrapper.appendChild(iconSpan);

        const messageSpan = document.createElement('span');
        messageSpan.className = 'toast-message';
        messageSpan.textContent = message;

        const dismissBtn = document.createElement('button');
        dismissBtn.className = 'btn-close toast-dismiss';
        dismissBtn.type = 'button';
        dismissBtn.setAttribute('aria-label', 'Dismiss notification');
        dismissBtn.innerHTML = '<span class="icon" data-lucide="x"></span>';

        dismissBtn.addEventListener('click', () => {
            toast.remove();
        });

        content.appendChild(iconWrapper);
        content.appendChild(messageSpan);
        content.appendChild(dismissBtn);
        toast.appendChild(content);
        container.appendChild(toast);

        this.refreshIcons();

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (toast.parentElement) {
                toast.remove();
            }
        }, 5000);
    }

    refreshIcons() {
        if (window.lucide?.createIcons) {
            window.lucide.createIcons();
            this.iconRetryAttempts = 0;
            if (this.pendingIconRefresh) {
                clearTimeout(this.pendingIconRefresh);
                this.pendingIconRefresh = null;
            }
            return;
        }

        if (this.iconRetryAttempts >= 5 || this.pendingIconRefresh) {
            return;
        }

        this.iconRetryAttempts += 1;
        this.pendingIconRefresh = setTimeout(() => {
            this.pendingIconRefresh = null;
            this.refreshIcons();
        }, 100 * this.iconRetryAttempts);
    }
    
    onTabChanged(tabName) {
        console.log(`Switched to ${tabName} tab`);
        
        // Tab-specific logic can be added here
        switch (tabName) {
            case 'devices':
                // Refresh devices when tab is viewed
                if (this.devices.size === 0) {
                    this.loadDevices();
                }
                break;
            case 'energy':
                // Load energy data
                this.loadEnergyData();
                break;
            case 'settings':
                // Check system status
                this.checkSystemStatus();
                break;
        }
    }
    
    async loadEnergyData() {
        // Mock energy data loading
        console.log('Loading energy data...');
        
        // Update energy stats with mock data
        const stats = {
            current: '2.4 kW',
            today: '18.5 kWh',
            monthly: '547 kWh'
        };
        
        document.getElementById('currentUsage').textContent = stats.current;
        document.getElementById('todayUsage').textContent = stats.today;
        document.getElementById('monthlyUsage').textContent = stats.monthly;
    }
    
    async checkSystemStatus() {
        const haStatus = document.getElementById('haStatus');
        if (!haStatus) return;

        try {
            const health = await this.apiCall('/health');
            const dependency = health?.dependencies?.home_assistant || {};
            const status = dependency.status || 'unknown';

            if (status === 'healthy') {
                haStatus.textContent = 'Online';
                haStatus.className = 'status-badge online';
            } else {
                haStatus.textContent = 'Degraded';
                haStatus.className = 'status-badge offline';
            }
        } catch (error) {
            console.error('Health check failed:', error);
            haStatus.textContent = 'Offline';
            haStatus.className = 'status-badge offline';
        }
    }

    getAuthToken() {
        // Token integration can be added via scenario-authenticator when available
        return '';
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new HomeAutomationApp();
});
