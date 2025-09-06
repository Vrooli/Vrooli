/**
 * Home Automation Intelligence - Frontend Application
 * Self-evolving home automation with AI-generated automations
 */

class HomeAutomationApp {
    constructor() {
        this.apiBaseUrl = '/api';
        this.wsUrl = `ws://${window.location.host}`;
        this.ws = null;
        this.currentUser = null;
        this.devices = new Map();
        this.automations = new Map();
        this.scenes = new Map();
        
        this.init();
    }
    
    init() {
        console.log('🏠 Initializing Home Automation Intelligence...');
        
        // Initialize UI components
        this.initializeTabNavigation();
        this.initializeWebSocket();
        this.initializeEventListeners();
        
        // Load initial data
        this.loadUserProfile();
        this.loadDevices();
        this.loadScenes();
        this.loadAutomations();
        
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
                console.log('🔌 WebSocket connected');
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
                console.log('🔌 WebSocket disconnected');
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
    
    // API Calls
    async apiCall(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': this.getAuthToken(),
                    ...options.headers
                },
                ...options
            });
            
            if (!response.ok) {
                throw new Error(`API call failed: ${response.status} ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            console.error(`API call to ${endpoint} failed:`, error);
            this.showToast(`API Error: ${error.message}`, 'error');
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
            
            console.log('👤 User profile loaded:', this.currentUser);
        } catch (error) {
            console.error('Failed to load user profile:', error);
            this.showToast('Failed to load user profile', 'error');
        }
    }
    
    // Devices Management
    async loadDevices() {
        try {
            this.showLoading('Loading devices...');
            
            // Mock device data for development
            // In implementation, this would call the home automation API
            const mockDevices = [
                {
                    id: 'light.living_room',
                    name: 'Living Room Lights',
                    type: 'lights',
                    state: { on: true, brightness: 80 },
                    available: true,
                    user_can_control: true
                },
                {
                    id: 'sensor.temperature',
                    name: 'Temperature Sensor',
                    type: 'sensors',
                    state: { temperature: 72, unit: '°F' },
                    available: true,
                    user_can_control: false
                },
                {
                    id: 'switch.coffee_maker',
                    name: 'Coffee Maker',
                    type: 'switches',
                    state: { on: false },
                    available: true,
                    user_can_control: true
                }
            ];
            
            this.devices.clear();
            mockDevices.forEach(device => {
                this.devices.set(device.id, device);
            });
            
            this.renderDevices();
            this.hideLoading();
            
            console.log('🔌 Devices loaded:', this.devices.size);
        } catch (error) {
            console.error('Failed to load devices:', error);
            this.showToast('Failed to load devices', 'error');
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
                    <div class="placeholder-icon">🔍</div>
                    <p>No devices found</p>
                    <small>Check Home Assistant connection</small>
                </div>
            `;
            return;
        }
        
        this.devices.forEach(device => {
            const deviceCard = this.createDeviceCard(device);
            devicesGrid.appendChild(deviceCard);
        });
    }
    
    createDeviceCard(device) {
        const card = document.createElement('div');
        card.className = 'device-card';
        card.dataset.deviceId = device.id;
        
        const statusClass = device.available 
            ? (device.state.on ? 'on' : 'off')
            : 'unavailable';
            
        const controls = device.user_can_control 
            ? this.createDeviceControls(device)
            : '<p style="color: var(--text-tertiary); font-size: 0.8rem;">Read-only</p>';
        
        card.innerHTML = `
            <div class="device-header">
                <div class="device-info">
                    <h3>${device.name}</h3>
                    <div class="device-type">${device.type}</div>
                </div>
                <div class="device-status ${statusClass}"></div>
            </div>
            <div class="device-state">
                ${this.formatDeviceState(device.state)}
            </div>
            <div class="device-controls">
                ${controls}
            </div>
        `;
        
        return card;
    }
    
    createDeviceControls(device) {
        if (device.type === 'lights' || device.type === 'switches') {
            const isOn = device.state.on;
            return `
                <button class="btn device-control ${isOn ? 'btn-secondary' : 'btn-primary'}" 
                        onclick="app.toggleDevice('${device.id}')">
                    ${isOn ? '🌙 Turn Off' : '💡 Turn On'}
                </button>
            `;
        }
        
        if (device.type === 'sensors') {
            return `
                <button class="btn device-control btn-secondary" 
                        onclick="app.refreshDevice('${device.id}')">
                    🔄 Refresh
                </button>
            `;
        }
        
        return '<small style="color: var(--text-tertiary);">No controls</small>';
    }
    
    formatDeviceState(state) {
        if (state.temperature !== undefined) {
            return `<div style="color: var(--accent-secondary); font-weight: 600;">${state.temperature}${state.unit}</div>`;
        }
        
        if (state.brightness !== undefined) {
            return `<div style="color: var(--text-secondary); font-size: 0.875rem;">Brightness: ${state.brightness}%</div>`;
        }
        
        if (state.on !== undefined) {
            return `<div style="color: ${state.on ? 'var(--device-on)' : 'var(--device-off)'}; font-weight: 500;">
                ${state.on ? 'On' : 'Off'}
            </div>`;
        }
        
        return '<div style="color: var(--text-tertiary); font-size: 0.875rem;">Ready</div>';
    }
    
    async toggleDevice(deviceId) {
        try {
            const device = this.devices.get(deviceId);
            if (!device) return;
            
            this.showLoading(`Controlling ${device.name}...`);
            
            // Mock device control - in implementation, this would call the API
            device.state.on = !device.state.on;
            
            setTimeout(() => {
                this.renderDevices();
                this.hideLoading();
                this.showToast(`${device.name} ${device.state.on ? 'turned on' : 'turned off'}`, 'success');
            }, 500);
            
        } catch (error) {
            console.error('Failed to toggle device:', error);
            this.showToast('Failed to control device', 'error');
            this.hideLoading();
        }
    }
    
    async refreshDevice(deviceId) {
        const device = this.devices.get(deviceId);
        if (device) {
            this.showToast(`${device.name} refreshed`, 'success');
        }
    }
    
    filterDevices(filter) {
        const deviceCards = document.querySelectorAll('.device-card');
        
        deviceCards.forEach(card => {
            const deviceId = card.dataset.deviceId;
            const device = this.devices.get(deviceId);
            
            if (filter === 'all' || device.type === filter) {
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
            console.log('🎬 Scenes loaded');
        } catch (error) {
            console.error('Failed to load scenes:', error);
        }
    }
    
    // Automations Management
    async loadAutomations() {
        try {
            // Mock automations - in implementation, this would load from API
            this.automations.clear();
            console.log('🤖 Automations loaded');
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
                this.showToast('🧠 AI automation generated successfully!', 'success');
                
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
        console.log('📨 WebSocket message:', data);
        
        switch (data.type) {
            case 'device_state_changed':
                this.handleDeviceStateUpdate(data);
                break;
            case 'automation_executed':
                this.handleAutomationExecution(data);
                break;
            case 'connection':
                console.log('🔌 WebSocket connection confirmed');
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
            console.log('🔄 Device state updated:', data.device_id);
        }
    }
    
    handleAutomationExecution(data) {
        this.showToast(`🤖 Automation executed: ${data.automation_id}`, 'info');
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
        toast.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>${message}</span>
                <button class="btn-close" onclick="this.parentElement.parentElement.remove()">✕</button>
            </div>
        `;
        
        container.appendChild(toast);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (toast.parentElement) {
                toast.remove();
            }
        }, 5000);
    }
    
    onTabChanged(tabName) {
        console.log(`📱 Switched to ${tabName} tab`);
        
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
        console.log('⚡ Loading energy data...');
        
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
        // Mock system status check
        console.log('🔧 Checking system status...');
        
        const haStatus = document.getElementById('haStatus');
        if (haStatus) {
            // Mock Home Assistant status
            haStatus.textContent = 'Online';
            haStatus.className = 'status-badge online';
        }
    }
    
    getAuthToken() {
        // In implementation, this would get JWT token from scenario-authenticator
        return 'Bearer mock-jwt-token';
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new HomeAutomationApp();
});