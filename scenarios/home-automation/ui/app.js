/**
 * Home Automation Intelligence - Frontend Application
 * Self-evolving home automation with AI-generated automations
 */

import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__homeAutomationBridgeInitialized';

// Common suffix patterns that typically represent accessory entities rather than primary devices
const DEVICE_GROUP_SUFFIX_PATTERNS = [
    'battery',
    'battery_level',
    'connected',
    'consumption',
    'duration',
    'energy',
    'filter',
    'filter_remaining',
    'humidity',
    'illuminance',
    'last_changed',
    'last_seen',
    'last_update',
    'last_updated',
    'lighting',
    'next_dawn',
    'next_dusk',
    'next_midnight',
    'next_noon',
    'next_rising',
    'next_setting',
    'power',
    'pressure',
    'schedule',
    'schedule_turn_off',
    'schedule_turn_on',
    'signal',
    'speed',
    'state',
    'status',
    'temperature',
    'voltage'
];

function initializeIframeBridge() {
    if (typeof window === 'undefined') {
        return;
    }

    if (window.parent === window) {
        return;
    }

    if (window[BRIDGE_FLAG]) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[HomeAutomation] Unable to parse parent origin for iframe bridge', error);
    }

    try {
        initIframeBridgeChild({
            appId: 'home-automation',
            parentOrigin,
            captureLogs: { enabled: true },
            captureNetwork: { enabled: true }
        });
        window[BRIDGE_FLAG] = true;
    } catch (error) {
        console.warn('[HomeAutomation] Failed to initialize iframe bridge', error);
    }
}

initializeIframeBridge();

class HomeAutomationApp {
    constructor() {
        this.apiBaseUrl = this.resolveApiBaseUrl();
        this.wsUrl = this.resolveWebSocketUrl();
        this.ws = null;
        this.currentUser = null;
        this.devices = new Map();
        this.deviceGroups = [];
        this.automations = new Map();
        this.scenes = new Map();
        this.iconRetryAttempts = 0;
        this.hiddenDeviceGroupIds = new Set();
        this.hiddenLinkedEntityIds = new Set();
        this.hiddenDeviceStorageKey = 'homeAutomationHiddenDeviceGroups';
        this.hiddenLinkedStorageKey = 'homeAutomationHiddenLinkedEntities';
        this.currentDeviceFilter = 'visible';
        this.currentLinkedEntitiesFilter = 'visible';
        this.pendingIconRefresh = null;
        this.deviceDataSource = 'unknown';
        this.usingMockDevices = false;
        this.homeAssistantConfig = {
            baseUrl: '',
            tokenConfigured: false,
            mockMode: false,
            status: 'unknown'
        };

        this.deviceModal = null;
        this.deviceModalContent = null;
        this.deviceModalTitle = null;
        this.deviceModalSubtitle = null;
        this.modalReturnFocusTo = null;
        this.boundModalKeyListener = null;
        this.activeModalDeviceId = null;
        this.linkedEntitiesSectionElement = null;
        this.boundLinkedEntitiesHandler = null;
        this.activeLinkedGroupId = null;

        this.init();
    }

    init() {
        console.log('Initializing Home Automation Intelligence...');
        
        // Initialize UI components
        this.initializeTabNavigation();
        this.initializeWebSocket();
        this.initializeEventListeners();
        this.initializeDeviceDetailsModal();
        this.initializeHomeAssistantConfigForm();
        this.loadHiddenDevicePreferences();
        this.loadHiddenLinkedEntityPreferences();
        this.syncInitialDeviceFilter();

        // Load initial data
        this.loadUserProfile();
        this.loadDevices();
        this.loadScenes();
        this.loadAutomations();
        this.loadHomeAssistantConfig();

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

    loadHiddenDevicePreferences() {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            const raw = window.localStorage.getItem(this.hiddenDeviceStorageKey);
            if (!raw) {
                return;
            }

            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) {
                const normalized = parsed
                    .map((value) => this.normalizeGroupId(value))
                    .filter((value) => Boolean(value));
                this.hiddenDeviceGroupIds = new Set(normalized);
            }
        } catch (error) {
            console.warn('[HomeAutomation] Failed to load hidden device preferences', error);
            this.hiddenDeviceGroupIds = new Set();
        }
    }

    loadHiddenLinkedEntityPreferences() {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            const raw = window.localStorage.getItem(this.hiddenLinkedStorageKey);
            if (!raw) {
                return;
            }

            const parsed = JSON.parse(raw);
            if (Array.isArray(parsed)) {
                const normalized = parsed
                    .map((value) => this.normalizeEntityId(value))
                    .filter(Boolean);
                this.hiddenLinkedEntityIds = new Set(normalized);
            }
        } catch (error) {
            console.warn('[HomeAutomation] Failed to load hidden linked entity preferences', error);
            this.hiddenLinkedEntityIds = new Set();
        }
    }

    syncInitialDeviceFilter() {
        if (typeof document === 'undefined') {
            this.currentDeviceFilter = 'visible';
            return;
        }

        const select = document.getElementById('deviceFilter');
        const normalized = this.normalizeDeviceFilter(select?.value);
        this.currentDeviceFilter = normalized;

        if (select && select.value !== normalized) {
            select.value = normalized;
        }
    }

    persistHiddenDevicePreferences() {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            const serialized = JSON.stringify(Array.from(this.hiddenDeviceGroupIds));
            window.localStorage.setItem(this.hiddenDeviceStorageKey, serialized);
        } catch (error) {
            console.warn('[HomeAutomation] Failed to persist hidden device preferences', error);
        }
    }


    persistHiddenLinkedPreferences() {
        if (typeof window === 'undefined') {
            return;
        }

        try {
            const serialized = JSON.stringify(Array.from(this.hiddenLinkedEntityIds));
            window.localStorage.setItem(this.hiddenLinkedStorageKey, serialized);
        } catch (error) {
            console.warn('[HomeAutomation] Failed to persist hidden linked entity preferences', error);
        }
    }


    initializeDeviceDetailsModal() {
        if (typeof document === 'undefined') {
            return;
        }

        this.deviceModal = document.getElementById('deviceDetailModal');
        this.deviceModalContent = document.getElementById('deviceModalContent');
        this.deviceModalTitle = document.getElementById('deviceModalTitle');
        this.deviceModalSubtitle = document.getElementById('deviceModalSubtitle');

        if (!this.deviceModal || !this.deviceModalContent || !this.deviceModalTitle) {
            console.warn('Device detail modal elements not found; device inspector disabled.');
            return;
        }

        const dismissElements = this.deviceModal.querySelectorAll('[data-modal-dismiss]');
        dismissElements.forEach((element) => {
            element.addEventListener('click', (event) => {
                event.preventDefault();
                this.closeDeviceModal();
            }, { passive: true });
        });

        this.deviceModal.addEventListener('click', (event) => {
            if (event.target && event.target.dataset && Object.prototype.hasOwnProperty.call(event.target.dataset, 'modalDismiss')) {
                this.closeDeviceModal();
            }
        });

        this.boundModalKeyListener = (event) => {
            if (event.key === 'Escape' && this.isDeviceModalOpen()) {
                this.closeDeviceModal();
            }
        };

        document.addEventListener('keydown', this.boundModalKeyListener);
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
                const contentType = response.headers.get('content-type') || '';
                if (contentType.includes('application/json')) {
                    const errorPayload = await response.json().catch(() => null);
                    const message = errorPayload?.error || errorPayload?.message;
                    throw new Error(message || `API call failed: ${response.status} ${response.statusText}`);
                }

                const errorText = (await response.text()).trim();
                throw new Error(errorText || `API call failed: ${response.status} ${response.statusText}`);
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
                profileId: 'profile-admin',
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
            const dataSource = typeof payload?.data_source === 'string' ? payload.data_source : 'unknown';
            const usingMockData = payload?.mock_data === true || dataSource === 'mock';

            this.devices.clear();
            apiDevices.forEach(rawDevice => {
                const normalized = this.normalizeDevice(rawDevice);
                if (normalized) {
                    this.devices.set(normalized.id, normalized);
                }
            });

            this.deviceDataSource = dataSource;
            this.usingMockDevices = usingMockData;
            this.updateDeviceDataSourceBanner();

            this.renderDevices();
            this.hideLoading();

            console.log('Devices loaded:', this.devices.size);
        } catch (error) {
            console.error('Failed to load devices:', error);
            this.showToast(`Failed to load devices: ${error.message}`, 'error');
            this.hideLoading();
            this.usingMockDevices = false;
            this.deviceDataSource = 'unknown';
            this.updateDeviceDataSourceBanner();
        }
    }
    
    renderDevices() {
        const devicesGrid = document.getElementById('devicesGrid');
        if (!devicesGrid) {
            return;
        }

        devicesGrid.innerHTML = '';

        const deviceGroups = this.buildDeviceGroups();
        this.deviceGroups = deviceGroups;
        this.pruneHiddenDeviceGroups(deviceGroups);
        this.pruneHiddenLinkedEntities(deviceGroups);

        const normalizedFilter = this.normalizeDeviceFilter(this.currentDeviceFilter);
        this.currentDeviceFilter = normalizedFilter;

        const select = document.getElementById('deviceFilter');
        if (select && select.value !== normalizedFilter) {
            select.value = normalizedFilter;
        }

        const filteredGroups = this.getFilteredDeviceGroups(deviceGroups, normalizedFilter);

        if (filteredGroups.length === 0) {
            devicesGrid.innerHTML = this.buildDeviceGridEmptyState(normalizedFilter, deviceGroups.length);
            this.refreshIcons();
            return;
        }

        filteredGroups.forEach((group) => {
            const groupCard = this.createDeviceGroupCard(group);
            devicesGrid.appendChild(groupCard);
        });

        this.refreshIcons();
    }
    normalizeDeviceFilter(filter) {
        const allowed = new Set(['visible', 'hidden', 'all', 'lights', 'sensors', 'switches', 'climate']);
        if (!filter || !allowed.has(filter)) {
            return 'visible';
        }
        return filter;
    }

    normalizeGroupId(value) {
        if (typeof value === 'string') {
            return value;
        }

        if (typeof value === 'number') {
            return String(value);
        }

        return null;
    }

    normalizeEntityId(value) {
        if (value === null || value === undefined) {
            return null;
        }

        if (typeof value === 'object' && value && Object.prototype.hasOwnProperty.call(value, 'id')) {
            return this.normalizeEntityId(value.id);
        }

        if (typeof value === 'string') {
            return value;
        }

        if (typeof value === 'number') {
            return String(value);
        }

        return null;
    }

    isDeviceGroupHidden(groupId) {
        const normalized = this.normalizeGroupId(groupId);
        if (!normalized) {
            return false;
        }

        return this.hiddenDeviceGroupIds.has(normalized);
    }

    isLinkedEntityHidden(entityId) {
        const normalized = this.normalizeEntityId(entityId);
        if (!normalized) {
            return false;
        }

        return this.hiddenLinkedEntityIds.has(normalized);
    }

    pruneHiddenDeviceGroups(deviceGroups) {
        if (!Array.isArray(deviceGroups) || deviceGroups.length === 0) {
            return;
        }

        const availableIds = new Set(deviceGroups.map((group) => this.normalizeGroupId(group.id)).filter(Boolean));
        let changed = false;

        Array.from(this.hiddenDeviceGroupIds).forEach((groupId) => {
            if (!availableIds.has(groupId)) {
                this.hiddenDeviceGroupIds.delete(groupId);
                changed = true;
            }
        });

        if (changed) {
            this.persistHiddenDevicePreferences();
        }
    }

    pruneHiddenLinkedEntities(deviceGroups) {
        if (!Array.isArray(deviceGroups) || deviceGroups.length === 0) {
            return;
        }

        const availableIds = new Set();
        deviceGroups.forEach((group) => {
            if (!group || !Array.isArray(group.others)) {
                return;
            }

            group.others.forEach((entity) => {
                const identifier = this.normalizeEntityId(entity?.id);
                if (identifier) {
                    availableIds.add(identifier);
                }
            });
        });

        let changed = false;
        Array.from(this.hiddenLinkedEntityIds).forEach((entityId) => {
            if (!availableIds.has(entityId)) {
                this.hiddenLinkedEntityIds.delete(entityId);
                changed = true;
            }
        });

        if (changed) {
            this.persistHiddenLinkedPreferences();
        }
    }

    getFilteredDeviceGroups(deviceGroups, filter) {
        const normalizedFilter = this.normalizeDeviceFilter(filter);
        const domainMap = {
            lights: ['light'],
            sensors: ['sensor', 'binary_sensor'],
            switches: ['switch'],
            climate: ['climate']
        };

        if (normalizedFilter === 'hidden') {
            return deviceGroups.filter((group) => this.isDeviceGroupHidden(group.id));
        }

        if (normalizedFilter === 'all') {
            return deviceGroups;
        }

        const visibleGroups = deviceGroups.filter((group) => !this.isDeviceGroupHidden(group.id));
        const domains = domainMap[normalizedFilter];

        if (!domains || normalizedFilter === 'visible') {
            return visibleGroups;
        }

        return visibleGroups.filter((group) => this.groupMatchesDomainFilter(group, domains));
    }

    groupMatchesDomainFilter(group, domain) {
        if (!group || !domain) {
            return false;
        }

        const domainList = Array.isArray(domain) ? domain : [domain];
        const normalizedDomains = domainList
            .map((value) => (value || '').toLowerCase())
            .filter(Boolean);

        if (normalizedDomains.length === 0) {
            return false;
        }

        const primaryDomain = (group.primary?.domain || group.primary?.type || '').toLowerCase();
        if (normalizedDomains.includes(primaryDomain)) {
            return true;
        }

        return Array.isArray(group.domains) && group.domains.some((value) => normalizedDomains.includes((value || '').toLowerCase()));
    }

    buildDeviceGridEmptyState(filter, totalGroups) {
        let title = 'No devices found';
        let message = 'Check Home Assistant connection';
        let icon = 'search';

        if (filter === 'hidden') {
            title = 'No hidden devices';
            message = 'Hide a device from the main list to manage it here.';
            icon = 'eye';
        } else if (totalGroups > 0) {
            title = 'No devices match the filter';
            message = 'Try a different filter or show all devices.';
            icon = 'filter';
        }

        return `
            <div class="device-placeholder">
                <div class="placeholder-icon"><span class="icon" data-lucide="${icon}"></span></div>
                <p>${title}</p>
                <small>${message}</small>
            </div>
        `;
    }

    hideDeviceGroup(group) {
        const identifier = this.normalizeGroupId(typeof group === 'object' ? group?.id : group);
        if (!identifier) {
            return;
        }

        if (this.hiddenDeviceGroupIds.has(identifier)) {
            return;
        }

        this.hiddenDeviceGroupIds.add(identifier);
        this.persistHiddenDevicePreferences();

        const label = typeof group === 'object' ? (group?.name || group?.primary?.name || identifier) : identifier;
        this.showToast(`${label} hidden. View Hidden Devices to manage it.`, 'info');
        this.renderDevices();
    }

    unhideDeviceGroup(group) {
        const identifier = this.normalizeGroupId(typeof group === 'object' ? group?.id : group);
        if (!identifier || !this.hiddenDeviceGroupIds.has(identifier)) {
            return;
        }

        this.hiddenDeviceGroupIds.delete(identifier);
        this.persistHiddenDevicePreferences();

        const label = typeof group === 'object' ? (group?.name || group?.primary?.name || identifier) : identifier;
        this.showToast(`${label} returned to the device list.`, 'success');
        this.renderDevices();
    }

    hideLinkedEntity(entity, context = {}) {
        const identifier = this.normalizeEntityId(typeof entity === 'object' ? entity?.id : entity);
        if (!identifier || this.hiddenLinkedEntityIds.has(identifier)) {
            return;
        }

        this.hiddenLinkedEntityIds.add(identifier);
        this.persistHiddenLinkedPreferences();

        const entityName = typeof entity === 'object' ? (entity?.name || entity?.id || identifier) : identifier;
        const groupName = context?.group?.name || context?.group?.primary?.name;
        const label = groupName ? `${entityName} hidden on ${groupName}.` : `${entityName} hidden.`;

        this.showToast(`${label} Switch to Hidden to manage it.`, 'info');
        this.renderDevices();
    }

    unhideLinkedEntity(entity, context = {}) {
        const identifier = this.normalizeEntityId(typeof entity === 'object' ? entity?.id : entity);
        if (!identifier || !this.hiddenLinkedEntityIds.has(identifier)) {
            return;
        }

        this.hiddenLinkedEntityIds.delete(identifier);
        this.persistHiddenLinkedPreferences();

        const entityName = typeof entity === 'object' ? (entity?.name || entity?.id || identifier) : identifier;
        const groupName = context?.group?.name || context?.group?.primary?.name;
        const label = groupName ? `${entityName} is visible on ${groupName} again.` : `${entityName} is visible again.`;

        this.showToast(label, 'success');
        this.renderDevices();
    }


    buildDeviceGroups() {
        if (this.devices.size === 0) {
            return [];
        }

        const groups = new Map();

        this.devices.forEach(device => {
            const groupId = device.groupId || device.id;
            if (!groups.has(groupId)) {
                groups.set(groupId, {
                    id: groupId,
                    devices: [],
                    domains: new Set(),
                    primary: null,
                    primaryScore: Number.POSITIVE_INFINITY,
                    name: null
                });
            }

            const group = groups.get(groupId);
            group.devices.push(device);

            const domain = (device.domain || device.type || 'unknown').toLowerCase();
            if (domain) {
                group.domains.add(domain);
            }

            const score = this.getDomainPriority(domain, device);
            if (!group.primary || score < group.primaryScore || (score === group.primaryScore && (device.name || '').localeCompare(group.primary.name || '', undefined, { sensitivity: 'base' }) < 0)) {
                group.primary = device;
                group.primaryScore = score;
                group.name = this.computeGroupName(device);
            }
        });

        const sortedGroups = Array.from(groups.values()).map(group => {
            const others = group.devices.filter(device => !group.primary || device.id !== group.primary.id);
            const orderedOthers = others.sort((a, b) => {
                const diff = this.getDomainPriority(a.domain || a.type, a) - this.getDomainPriority(b.domain || b.type, b);
                if (diff !== 0) {
                    return diff;
                }
                return (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' });
            });

            return {
                id: group.id,
                name: group.name || group.primary?.name || group.id,
                primary: group.primary || group.devices[0],
                others: orderedOthers,
                domains: Array.from(group.domains),
                size: group.devices.length
            };
        });

        sortedGroups.sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }));

        return sortedGroups;
    }

    computeGroupName(device) {
        if (!device) {
            return 'Unnamed device';
        }

        const attributes = device.attributes || {};

        if (attributes.device && typeof attributes.device === 'object' && attributes.device.name) {
            return attributes.device.name;
        }

        if (attributes.friendly_name && typeof attributes.friendly_name === 'string') {
            return attributes.friendly_name;
        }

        return device.name || device.id;
    }

    getDomainPriority(domain, device) {
        const normalized = (domain || '').toLowerCase();
        const priorityMap = {
            'climate': 0,
            'lock': 1,
            'cover': 2,
            'light': 3,
            'fan': 4,
            'switch': 5,
            'humidifier': 6,
            'binary_sensor': 8,
            'sensor': 9,
        };

        const basePriority = Object.prototype.hasOwnProperty.call(priorityMap, normalized) ? priorityMap[normalized] : 20;
        const accessoryPenalty = device?.isAccessory ? 20 : 0;

        return basePriority + accessoryPenalty;
    }

    createDeviceGroupCard(group) {
        const primaryDevice = group.primary;
        const normalizedGroupId = this.normalizeGroupId(group.id) || group.id;
        const isHidden = this.isDeviceGroupHidden(group.id);
        const card = this.createDeviceCard(primaryDevice, { groupId: normalizedGroupId, hidden: isHidden });

        card.dataset.groupId = normalizedGroupId;
        card.dataset.domains = group.domains.join(',');
        card.dataset.primaryDomain = primaryDevice?.domain || primaryDevice?.type || '';
        card.dataset.groupSize = String(group.size || 1);
        card.dataset.hidden = String(isHidden);
        card.dataset.visibilityState = isHidden ? 'hidden' : 'visible';

        if (isHidden) {
            card.classList.add('device-card--hidden');
            const hiddenBanner = document.createElement('div');
            hiddenBanner.className = 'device-hidden-banner';
            hiddenBanner.innerHTML = `
                <span class="icon" data-lucide="eye-off"></span>
                <span>Hidden Device</span>
            `;
            card.insertBefore(hiddenBanner, card.firstChild);
        }

        const allLinkedEntities = Array.isArray(group.others) ? group.others : [];
        const visibleLinkedEntities = allLinkedEntities.filter((linked) => !this.isLinkedEntityHidden(linked.id));
        const hiddenLinkedCount = allLinkedEntities.length - visibleLinkedEntities.length;

        if (allLinkedEntities.length > 0) {
            card.classList.add('device-card--group');

            const headerInfo = card.querySelector('.device-info');
            if (headerInfo) {
                const badge = document.createElement('div');
                badge.className = 'device-badge device-badge--hub';
                const visibleCount = visibleLinkedEntities.length;
                const baseLabel = `${visibleCount} linked ${visibleCount === 1 ? 'entity' : 'entities'}`;
                const badgeLabel = hiddenLinkedCount > 0 ? `${baseLabel} (+${hiddenLinkedCount} hidden)` : baseLabel;
                badge.innerHTML = `
                    <span class="icon" data-lucide="git-branch"></span>
                    <span>${this.escapeHtml(badgeLabel)}</span>
                `;
                headerInfo.appendChild(badge);
            }

            const summary = document.createElement('div');
            summary.className = 'device-group-summary';

            const header = document.createElement('div');
            header.className = 'device-group-summary-header';

            const headerIcon = document.createElement('span');
            headerIcon.className = 'icon';
            headerIcon.dataset.lucide = 'layers';

            const headerLabel = document.createElement('span');
            headerLabel.textContent = 'Linked controls & sensors';

            header.appendChild(headerIcon);
            header.appendChild(headerLabel);

            summary.appendChild(header);

            const controlsCount = visibleLinkedEntities.filter(device => this.canControlDomain(device.domain)).length;
            const sensorsCount = visibleLinkedEntities.length - controlsCount;

            const stats = document.createElement('div');
            stats.className = 'device-group-summary-stats';
            if (visibleLinkedEntities.length > 0) {
                stats.textContent = `${controlsCount} control${controlsCount === 1 ? '' : 's'} • ${sensorsCount} sensor${sensorsCount === 1 ? '' : 's'}`;
            } else if (hiddenLinkedCount > 0) {
                stats.textContent = 'All linked entities are hidden';
            } else {
                stats.textContent = 'No linked entities available';
            }
            summary.appendChild(stats);

            const names = visibleLinkedEntities
                .map(device => this.buildLinkedSummaryLabel(group, device))
                .filter(Boolean);
            const summaryList = this.formatLinkedNames(names);

            if (summaryList) {
                const listEl = document.createElement('div');
                listEl.className = 'device-group-summary-list';
                listEl.textContent = summaryList;
                summary.appendChild(listEl);
            } else if (hiddenLinkedCount > 0) {
                const hintEl = document.createElement('div');
                hintEl.className = 'device-group-summary-list';
                hintEl.textContent = 'Hidden linked entities can be managed in device details.';
                summary.appendChild(hintEl);
            }

            const subEntitiesContainer = document.createElement('div');
            subEntitiesContainer.className = 'device-sub-entities';

            visibleLinkedEntities.forEach(device => {
                subEntitiesContainer.appendChild(this.createSubEntityRow(device, group.name));
            });

            if (visibleLinkedEntities.length === 0 && hiddenLinkedCount > 0) {
                const placeholder = document.createElement('div');
                placeholder.className = 'sub-entity-row sub-entity-row--empty';
                placeholder.innerHTML = `
                    <span class="icon" data-lucide="eye-off"></span>
                    <span>All linked entities are hidden. Open details to review them.</span>
                `;
                subEntitiesContainer.appendChild(placeholder);
            }

            card.appendChild(summary);
            card.appendChild(subEntitiesContainer);
        }

        this.attachVisibilityToggle(card, group);
        this.attachDeviceCardInteractions(card, group);

        return card;
    }

    attachDeviceCardInteractions(card, group) {
        if (!card) {
            return;
        }

        const deviceId = card.dataset.deviceId;
        if (!deviceId) {
            return;
        }

        const inferredLabel = group?.name || this.devices.get(deviceId)?.name || 'device';

        card.setAttribute('role', 'button');
        card.setAttribute('tabindex', '0');
        card.setAttribute('aria-label', `View details for ${inferredLabel}`);

        const openModal = () => {
            this.openDeviceModal(deviceId, { group, triggerElement: card });
        };

        card.addEventListener('click', (event) => {
            if (event.target.closest('.device-control') || event.target.closest('.sub-entity-controls') || event.target.closest('button')) {
                return;
            }

            openModal();
        });

        card.addEventListener('keydown', (event) => {
            if (event.key !== 'Enter' && event.key !== ' ') {
                return;
            }

            if (event.target !== card && event.target.closest('.device-control')) {
                return;
            }

            event.preventDefault();
            openModal();
        });
    }

    attachVisibilityToggle(card, group) {
        if (!card || !group?.id) {
            return;
        }

        const toggle = card.querySelector('.device-visibility-toggle');
        if (!toggle) {
            return;
        }

        toggle.addEventListener('click', (event) => {
            event.preventDefault();
            event.stopPropagation();

            if (this.isDeviceGroupHidden(group.id)) {
                this.unhideDeviceGroup(group);
            } else {
                this.hideDeviceGroup(group);
            }
        });
    }

    openDeviceModal(deviceId, options = {}) {
        if (!this.deviceModal || !this.deviceModalContent || !deviceId) {
            return;
        }

        const device = this.devices.get(deviceId);
        if (!device) {
            console.warn('Device not found for modal view:', deviceId);
            return;
        }

        const group = options.group || this.findDeviceGroup(device);

        if (this.deviceModalTitle) {
            this.deviceModalTitle.textContent = device.name || device.id;
        }

        if (this.deviceModalSubtitle) {
            const subtitleParts = [];
            if (device.id) {
                subtitleParts.push(`ID: ${device.id}`);
            }
            if (group && typeof group.size === 'number') {
                subtitleParts.push(group.size === 1 ? 'Single entity' : `${group.size} linked entities`);
            }
            if (device.available === true) {
                subtitleParts.push('Online');
            } else if (device.available === false) {
                subtitleParts.push('Offline');
            }
            this.deviceModalSubtitle.textContent = subtitleParts.join(' • ');
        }

        this.currentLinkedEntitiesFilter = 'visible';
        this.activeLinkedGroupId = this.normalizeGroupId(group?.id) || this.normalizeGroupId(device.groupId) || null;

        this.deviceModalContent.innerHTML = this.buildDeviceModalContent(device, group);
        this.deviceModalContent.scrollTop = 0;

        this.initializeLinkedEntitiesSection(group);
        this.refreshIcons();

        this.deviceModal.classList.add('is-open');
        this.deviceModal.setAttribute('aria-hidden', 'false');
        document.body.classList.add('modal-open');
        this.activeModalDeviceId = deviceId;

        if (options.triggerElement instanceof HTMLElement) {
            this.modalReturnFocusTo = options.triggerElement;
        } else {
            this.modalReturnFocusTo = null;
        }

        const dialog = this.deviceModal.querySelector('.modal__dialog');
        if (dialog && typeof dialog.focus === 'function') {
            setTimeout(() => dialog.focus({ preventScroll: true }), 0);
        }
    }

    closeDeviceModal() {
        if (!this.deviceModal) {
            return;
        }

        this.deviceModal.classList.remove('is-open');
        this.deviceModal.setAttribute('aria-hidden', 'true');
        document.body.classList.remove('modal-open');
        this.activeModalDeviceId = null;

        if (this.linkedEntitiesSectionElement && this.boundLinkedEntitiesHandler) {
            this.linkedEntitiesSectionElement.removeEventListener('click', this.boundLinkedEntitiesHandler);
        }

        this.linkedEntitiesSectionElement = null;
        this.boundLinkedEntitiesHandler = null;
        this.activeLinkedGroupId = null;
        this.currentLinkedEntitiesFilter = 'visible';

        if (this.deviceModalContent) {
            this.deviceModalContent.innerHTML = '';
        }

        if (this.deviceModalSubtitle) {
            this.deviceModalSubtitle.textContent = '';
        }

        const returnTarget = this.modalReturnFocusTo;
        this.modalReturnFocusTo = null;

        if (returnTarget && typeof returnTarget.focus === 'function') {
            setTimeout(() => returnTarget.focus(), 0);
        }
    }

    isDeviceModalOpen() {
        return Boolean(this.deviceModal && this.deviceModal.classList.contains('is-open'));
    }

    findDeviceGroup(device) {
        if (!device) {
            return null;
        }

        const deviceId = device.id;
        const groupId = device.groupId;

        return this.deviceGroups.find((group) => {
            if (!group) {
                return false;
            }

            if (group.id && groupId && group.id === groupId) {
                return true;
            }

            if (group.primary?.id === deviceId) {
                return true;
            }

            return Array.isArray(group.others) && group.others.some((item) => item.id === deviceId);
        }) || null;
    }

    buildDeviceModalContent(device, group) {
        const overviewRows = this.buildDeviceSummaryRows(device, group);
        const stateSection = this.buildDetailSection(device.state);
        const attributesSection = this.buildDetailSection(device.attributes);
        const rawSection = this.buildRawStateSection(device.rawState);

        return `
            <section class="modal__section">
                <h4 class="modal__section-title">Overview</h4>
                <div class="device-detail-grid">
                    ${overviewRows}
                </div>
            </section>
            <section class="modal__section">
                <h4 class="modal__section-title">Current State</h4>
                ${stateSection}
            </section>
            <section class="modal__section">
                <h4 class="modal__section-title">Attributes</h4>
                ${attributesSection}
            </section>
            <section class="modal__section modal__section--linked">
                <div class="modal__section-heading modal__section-heading--linked">
                    <h4 class="modal__section-title">Linked Entities</h4>
                    <div class="linked-entities__toolbar" role="group" aria-label="Linked entities filter"></div>
                </div>
                <div class="linked-entities__body"></div>
            </section>
            <section class="modal__section">
                <h4 class="modal__section-title">Raw Payload</h4>
                ${rawSection}
                <span class="modal__hint">Use the raw payload to debug integrations or build advanced automations.</span>
            </section>
        `;
    }

    initializeLinkedEntitiesSection(group) {
        if (!this.deviceModalContent) {
            return;
        }

        const section = this.deviceModalContent.querySelector('.modal__section--linked');
        if (!section) {
            return;
        }

        if (this.linkedEntitiesSectionElement && this.boundLinkedEntitiesHandler) {
            this.linkedEntitiesSectionElement.removeEventListener('click', this.boundLinkedEntitiesHandler);
        }

        this.linkedEntitiesSectionElement = section;
        this.boundLinkedEntitiesHandler = (event) => {
            this.handleLinkedEntitiesSectionClick(event);
        };

        section.addEventListener('click', this.boundLinkedEntitiesHandler, false);
        this.renderLinkedEntitiesSection(group);
    }

    renderLinkedEntitiesSection(group) {
        const section = this.linkedEntitiesSectionElement || this.deviceModalContent?.querySelector('.modal__section--linked');
        if (!section) {
            return;
        }

        const toolbar = section.querySelector('.linked-entities__toolbar');
        const body = section.querySelector('.linked-entities__body');
        if (!toolbar || !body) {
            return;
        }

        const normalizedGroupId = this.normalizeGroupId(group?.id);
        if (normalizedGroupId) {
            section.dataset.groupId = normalizedGroupId;
            this.activeLinkedGroupId = normalizedGroupId;
        } else {
            delete section.dataset.groupId;
            this.activeLinkedGroupId = null;
        }

        const linkedEntities = Array.isArray(group?.others) ? group.others : [];
        const visibleEntities = linkedEntities.filter((entity) => !this.isLinkedEntityHidden(entity.id));
        const hiddenEntities = linkedEntities.filter((entity) => this.isLinkedEntityHidden(entity.id));
        const activeFilter = this.currentLinkedEntitiesFilter === 'hidden' ? 'hidden' : 'visible';

        toolbar.innerHTML = this.buildLinkedEntitiesToolbar({
            activeFilter,
            visibleCount: visibleEntities.length,
            hiddenCount: hiddenEntities.length
        });

        const entitiesToRender = activeFilter === 'hidden' ? hiddenEntities : visibleEntities;
        let contentHtml;

        if (entitiesToRender.length === 0) {
            let message;
            if (linkedEntities.length === 0) {
                message = 'No linked entities available';
            } else if (activeFilter === 'hidden') {
                message = 'No hidden linked entities';
            } else if (hiddenEntities.length > 0) {
                message = 'No visible linked entities. Switch to Hidden to manage them.';
            } else {
                message = 'No visible linked entities';
            }
            contentHtml = `<span class="detail-empty">${this.escapeHtml(message)}</span>`;
        } else {
            const items = entitiesToRender
                .map((entity) => this.buildLinkedEntityItem(entity, { hidden: this.isLinkedEntityHidden(entity.id) }))
                .join('');
            contentHtml = `<div class="linked-entities linked-entities--${activeFilter}">${items}</div>`;
        }

        body.innerHTML = contentHtml;
        this.refreshIcons();
    }

    buildLinkedEntitiesToolbar({ activeFilter, visibleCount, hiddenCount }) {
        const normalizedFilter = activeFilter === 'hidden' ? 'hidden' : 'visible';
        return `
            <button type="button" class="linked-entities__filter-button ${normalizedFilter === 'visible' ? 'is-active' : ''}" data-linked-filter="visible">
                <span class="icon" data-lucide="eye"></span>
                <span>Visible (${visibleCount})</span>
            </button>
            <button type="button" class="linked-entities__filter-button ${normalizedFilter === 'hidden' ? 'is-active' : ''}" data-linked-filter="hidden">
                <span class="icon" data-lucide="eye-off"></span>
                <span>Hidden (${hiddenCount})</span>
            </button>
        `;
    }

    buildLinkedEntityItem(entity, options = {}) {
        if (!entity) {
            return '';
        }

        const normalizedId = this.normalizeEntityId(entity.id);
        if (!normalizedId) {
            return '';
        }

        const isHidden = options?.hidden === true;
        const stateHtml = this.formatDeviceState(entity.state, entity);
        const toggleAction = isHidden ? 'unhide' : 'hide';
        const toggleLabel = isHidden ? 'Unhide' : 'Hide';
        const toggleIcon = isHidden ? 'eye' : 'eye-off';

        return `
            <div class="linked-entity ${isHidden ? 'linked-entity--hidden' : ''}" data-entity-id="${this.escapeHtml(normalizedId)}" data-visibility="${isHidden ? 'hidden' : 'visible'}">
                <div class="linked-entity__meta">
                    <span class="icon" data-lucide="${this.escapeHtml(this.iconForDomain(entity.domain))}"></span>
                    <div class="linked-entity__text">
                        <span class="linked-entity__name">${this.escapeHtml(entity.name || entity.id)}</span>
                        <span class="linked-entity__type">${this.escapeHtml(this.titleCase(entity.domain || entity.type || 'Unknown'))}</span>
                    </div>
                </div>
                <div class="linked-entity__state">${stateHtml}</div>
                <div class="linked-entity__actions">
                    <button type="button" class="linked-entity__visibility-toggle" data-entity-id="${this.escapeHtml(normalizedId)}" data-action="${toggleAction}" aria-label="${this.escapeHtml(`${toggleLabel} ${entity.name || entity.id}`)}">
                        <span class="icon" data-lucide="${toggleIcon}"></span>
                        <span>${toggleLabel}</span>
                    </button>
                </div>
            </div>
        `;
    }

    handleLinkedEntitiesSectionClick(event) {
        const target = event.target instanceof HTMLElement ? event.target : null;
        if (!target) {
            return;
        }

        const filterButton = target.closest('[data-linked-filter]');
        if (filterButton) {
            event.preventDefault();
            const requestedFilter = filterButton.dataset.linkedFilter;
            if (requestedFilter && requestedFilter !== this.currentLinkedEntitiesFilter) {
                this.currentLinkedEntitiesFilter = requestedFilter === 'hidden' ? 'hidden' : 'visible';
                const group = this.getActiveLinkedEntityGroup();
                this.renderLinkedEntitiesSection(group);
            }
            return;
        }

        const toggleButton = target.closest('.linked-entity__visibility-toggle');
        if (!toggleButton) {
            return;
        }

        event.preventDefault();

        const entityId = this.normalizeEntityId(toggleButton.dataset.entityId);
        const action = toggleButton.dataset.action;
        const group = this.getActiveLinkedEntityGroup();
        if (!entityId || !group) {
            return;
        }

        const entity = Array.isArray(group.others)
            ? group.others.find((item) => this.normalizeEntityId(item.id) === entityId)
            : null;

        if (action === 'hide') {
            this.hideLinkedEntity(entity || entityId, { group });
        } else if (action === 'unhide') {
            this.unhideLinkedEntity(entity || entityId, { group });
        }

        const updatedGroup = this.getActiveLinkedEntityGroup();
        this.renderLinkedEntitiesSection(updatedGroup);
    }

    getActiveLinkedEntityGroup() {
        const normalizedGroupId = this.activeLinkedGroupId ? this.normalizeGroupId(this.activeLinkedGroupId) : null;
        if (normalizedGroupId) {
            const byId = this.deviceGroups.find((group) => this.normalizeGroupId(group.id) === normalizedGroupId);
            if (byId) {
                return byId;
            }
        }

        if (this.activeModalDeviceId) {
            const device = this.devices.get(this.activeModalDeviceId);
            if (device) {
                return this.findDeviceGroup(device);
            }
        }

        return null;
    }

    buildDeviceSummaryRows(device, group) {
        const rows = [];

        rows.push({
            label: 'Domain',
            value: this.escapeHtml(this.titleCase(device.domain || device.type || 'Unknown'))
        });

        rows.push({
            label: 'Entity ID',
            value: this.escapeHtml(device.id)
        });

        if (group?.name && group.name !== device.name) {
            rows.push({
                label: 'Group',
                value: this.escapeHtml(group.name)
            });
        }

        if (group) {
            const linkedEntities = Array.isArray(group.others) ? group.others : [];
            const hiddenLinked = linkedEntities.filter((entity) => this.isLinkedEntityHidden(entity.id)).length;
            const visibleLinked = linkedEntities.length - hiddenLinked;

            let summaryValue;
            if (linkedEntities.length === 0) {
                summaryValue = 'None';
            } else if (hiddenLinked > 0) {
                summaryValue = `${visibleLinked} visible • ${hiddenLinked} hidden`;
            } else {
                summaryValue = `${visibleLinked} visible`;
            }

            rows.push({
                label: 'Linked Entities',
                value: this.escapeHtml(summaryValue)
            });
        }

        if (device.entityCategory) {
            rows.push({
                label: 'Entity Category',
                value: this.escapeHtml(this.titleCase(device.entityCategory))
            });
        }

        rows.push({
            label: 'Availability',
            value: this.buildAvailabilityPill(device)
        });

        if (typeof device.user_can_control === 'boolean') {
            rows.push({
                label: 'User Control',
                value: this.formatDetailValue(device.user_can_control)
            });
        }

        if (device.lastUpdated) {
            rows.push({
                label: 'Last Updated',
                value: this.escapeHtml(this.formatTimestamp(device.lastUpdated))
            });
        }

        if (this.deviceDataSource && this.deviceDataSource !== 'unknown') {
            rows.push({
                label: 'Data Source',
                value: this.escapeHtml(this.formatDataSourceLabel(this.deviceDataSource))
            });
        }

        return rows
            .filter((row) => row && row.value)
            .map((row) => `
                <div class="device-detail-row">
                    <span class="device-detail-label">${this.escapeHtml(row.label)}</span>
                    <span class="device-detail-value">${row.value}</span>
                </div>
            `)
            .join('');
    }

    buildDetailSection(data) {
        if (!data || typeof data !== 'object') {
            return '<span class="detail-empty">No details available</span>';
        }

        const entries = Object.entries(data).filter(([key, value]) => typeof value !== 'function' && value !== undefined);
        if (entries.length === 0) {
            return '<span class="detail-empty">No details available</span>';
        }

        const rows = entries.map(([key, value]) => `
            <div class="device-detail-row">
                <span class="device-detail-label">${this.escapeHtml(this.titleCase(key))}</span>
                <span class="device-detail-value">${this.formatDetailValue(value)}</span>
            </div>
        `).join('');

        return `<div class="device-detail-grid">${rows}</div>`;
    }

    buildRawStateSection(rawState) {
        if (!rawState || typeof rawState !== 'object' || Object.keys(rawState).length === 0) {
            return '<span class="detail-empty">No raw payload available</span>';
        }

        return `<pre class="device-detail-json">${this.escapeHtml(JSON.stringify(rawState, null, 2))}</pre>`;
    }

    buildAvailabilityPill(device) {
        if (device.available === true) {
            return '<span class="status-pill is-online">Online</span>';
        }

        if (device.available === false) {
            return '<span class="status-pill is-offline">Offline</span>';
        }

        return '<span class="status-pill is-unknown">Unknown</span>';
    }

    formatDetailValue(value) {
        if (value === null || value === undefined) {
            return '<span class="detail-empty">Not available</span>';
        }

        if (typeof value === 'boolean') {
            return `<span class="status-pill ${value ? 'is-positive' : 'is-negative'}">${value ? 'True' : 'False'}</span>`;
        }

        if (typeof value === 'number') {
            return this.escapeHtml(String(value));
        }

        if (typeof value === 'string') {
            const trimmed = value.trim();
            return trimmed ? this.escapeHtml(trimmed) : '<span class="detail-empty">Empty</span>';
        }

        if (Array.isArray(value)) {
            if (value.length === 0) {
                return '<span class="detail-empty">Empty</span>';
            }
            return `<pre class="device-detail-json">${this.escapeHtml(JSON.stringify(value, null, 2))}</pre>`;
        }

        if (typeof value === 'object') {
            const keys = Object.keys(value);
            if (keys.length === 0) {
                return '<span class="detail-empty">Empty</span>';
            }
            return `<pre class="device-detail-json">${this.escapeHtml(JSON.stringify(value, null, 2))}</pre>`;
        }

        return this.escapeHtml(String(value));
    }

    formatDataSourceLabel(value) {
        if (!value) {
            return '';
        }
        return this.titleCase(String(value).replace(/[_\-]+/g, ' '));
    }

    titleCase(value) {
        if (!value && value !== 0) {
            return '';
        }

        return String(value)
            .replace(/[_\-]+/g, ' ')
            .split(' ')
            .filter(Boolean)
            .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
            .join(' ');
    }

    escapeHtml(value) {
        if (value === null || value === undefined) {
            return '';
        }

        return String(value)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    }

    formatTimestamp(value) {
        if (!value) {
            return '';
        }

        try {
            const date = new Date(value);
            if (!Number.isNaN(date.getTime())) {
                return date.toLocaleString(undefined, { hour12: false });
            }
        } catch (error) {
            console.warn('Failed to format timestamp', value, error);
        }

        return String(value);
    }

    formatLinkedNames(names) {
        if (!names || names.length === 0) {
            return '';
        }

        const joined = names.join(', ');

        if (joined.length <= 80) {
            return joined;
        }

        return `${joined.slice(0, 77)}...`;
    }

    createSubEntityRow(device, groupName) {
        const row = document.createElement('div');
        row.className = 'sub-entity-row';

        const domain = device.domain || device.type || 'unknown';
        const icon = this.iconForDomain(domain);

        const meta = document.createElement('div');
        meta.className = 'sub-entity-meta';

        const iconSpan = document.createElement('span');
        iconSpan.className = 'icon';
        iconSpan.dataset.lucide = icon;

        const textWrapper = document.createElement('div');
        textWrapper.className = 'sub-entity-text';

        const nameEl = document.createElement('div');
        nameEl.className = 'sub-entity-name';
        nameEl.textContent = this.simplifyLinkedName(groupName, device.name || device.id);

        const typeEl = document.createElement('div');
        typeEl.className = 'sub-entity-type';
        typeEl.textContent = this.describeLinkedEntity(domain, device);

        textWrapper.appendChild(nameEl);
        textWrapper.appendChild(typeEl);

        meta.appendChild(iconSpan);
        meta.appendChild(textWrapper);

        const stateEl = document.createElement('div');
        stateEl.className = 'sub-entity-state';
        stateEl.innerHTML = this.formatDeviceState(device.state, device);

        const controlsEl = document.createElement('div');
        controlsEl.className = 'sub-entity-controls';
        controlsEl.innerHTML = this.createSubEntityControls(device);

        row.appendChild(meta);
        row.appendChild(stateEl);
        row.appendChild(controlsEl);

        return row;
    }

    createSubEntityControls(device) {
        const domain = device.domain || device.type;
        if (!this.canControlDomain(domain) || !device.user_can_control) {
            return '<span class="sub-entity-readonly">Read-only</span>';
        }

        const isOn = device.state?.on === true || device.state?.active === true;
        const icon = isOn ? 'power-off' : 'power';
        const label = isOn ? 'Off' : 'On';

        return `
            <button class="btn device-control btn-compact ${isOn ? 'btn-secondary' : 'btn-primary'}" 
                    onclick="event.stopPropagation(); app.toggleDevice('${device.id}')">
                <span class="icon" data-lucide="${icon}"></span>
                <span>${label}</span>
            </button>
        `;
    }

    iconForDomain(domain) {
        switch ((domain || '').toLowerCase()) {
            case 'climate':
                return 'thermometer';
            case 'light':
                return 'lightbulb';
            case 'switch':
                return 'toggle-left';
            case 'fan':
                return 'fan';
            case 'lock':
                return 'lock';
            case 'cover':
                return 'app-window';
            case 'humidifier':
                return 'droplet';
            case 'sensor':
                return 'activity';
            case 'binary_sensor':
                return 'scan';
            default:
                return 'circle';
        }
    }

    updateDeviceDataSourceBanner() {
        const banner = document.getElementById('devicesDataSourceBanner');
        if (!banner) {
            return;
        }

        if (this.usingMockDevices) {
            banner.innerHTML = `
                <span class="icon" data-lucide="beaker"></span>
                <div class="banner-copy">
                    <strong>Demo devices in use</strong>
                    <p>Connect your Home Assistant hub to see live device data.</p>
                </div>
            `;
            banner.classList.add('is-visible', 'is-warning');
            console.warn('Home Automation UI is showing mock device data. Connect Home Assistant to view live devices.');
        } else {
            banner.innerHTML = '';
            banner.classList.remove('is-visible', 'is-warning');
        }
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

    deriveDeviceGroupId(raw, fallbackId) {
        const attributes = (raw && typeof raw === 'object' && raw.attributes && typeof raw.attributes === 'object')
            ? raw.attributes
            : {};

        if (raw?.group_id) {
            return String(raw.group_id);
        }

        if (attributes.device_id) {
            return String(attributes.device_id);
        }

        if (attributes.device && typeof attributes.device === 'object') {
            const deviceMeta = attributes.device;
            const nestedId = deviceMeta.id || deviceMeta.device_id || deviceMeta.identifier;
            if (nestedId) {
                return String(nestedId);
            }

            if (Array.isArray(deviceMeta.identifiers) && deviceMeta.identifiers.length > 0) {
                const identifier = deviceMeta.identifiers[0];
                if (Array.isArray(identifier)) {
                    return identifier.map(part => String(part)).join(':');
                }
                if (identifier) {
                    return String(identifier);
                }
            }

            if (deviceMeta.name) {
                const slug = this.slugifyName(deviceMeta.name);
                if (slug) {
                    return `ha:${slug}`;
                }
            }
        }

        const friendlyName = (raw?.name || attributes.friendly_name || '').trim();
        const friendlySlug = this.slugifyName(friendlyName);

        const entityId = raw?.device_id || raw?.id || fallbackId || '';
        const entitySlug = this.slugifyName(entityId.includes('.') ? entityId.split('.')[1] : entityId);

        const candidates = [];
        if (entitySlug) {
            const trimmed = this.stripAccessorySuffix(entitySlug);
            if (trimmed) {
                candidates.push(trimmed);
            }
        }

        if (friendlySlug) {
            const trimmed = this.stripAccessorySuffix(friendlySlug);
            if (trimmed) {
                candidates.push(trimmed);
            }
        }

        const base = candidates.find(Boolean) || entitySlug || friendlySlug;
        if (base) {
            return `ha:${base}`;
        }

        return entityId || fallbackId || null;
    }

    stripAccessorySuffix(value) {
        if (!value) {
            return value;
        }

        let sanitized = value;
        let changed = true;

        while (changed && sanitized) {
            changed = false;
            sanitized = sanitized.replace(/_[0-9]+$/, '');

            for (const suffix of DEVICE_GROUP_SUFFIX_PATTERNS) {
                if (sanitized === suffix) {
                    continue;
                }

                if (sanitized.endsWith(`_${suffix}`)) {
                    sanitized = sanitized.slice(0, sanitized.length - suffix.length - 1);
                    sanitized = sanitized.replace(/_[0-9]+$/, '');
                    changed = true;
                }
            }
        }

        return sanitized;
    }

    slugifyName(value) {
        if (!value || typeof value !== 'string') {
            return '';
        }

        return value
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '_')
            .replace(/^_+|_+$/g, '')
            .replace(/_+/g, '_');
    }

    describeLinkedEntity(domain, device) {
        const normalizedDomain = (domain || '').toLowerCase();
        const baseLabel = normalizedDomain
            ? normalizedDomain.replace(/_/g, ' ')
            : 'entity';

        const kindLabel = () => {
            if (['sensor', 'binary_sensor'].includes(normalizedDomain)) {
                return 'sensor';
            }
            if (this.canControlDomain(normalizedDomain)) {
                return 'control';
            }
            return 'setting';
        };

        const category = device?.entityCategory
            ? device.entityCategory.replace(/_/g, ' ')
            : null;

        let label = baseLabel;
        if (category) {
            label = `${label} • ${category}`;
        } else {
            const kind = kindLabel();
            if (kind && !label.includes(kind)) {
                label = `${label} ${kind}`;
            }
        }

        return label.replace(/\b([a-z])/g, (_, char) => char.toUpperCase());
    }

    simplifyLinkedName(parentName, childName) {
        if (!childName) {
            return childName;
        }

        if (!parentName) {
            return childName;
        }

        const normalizedParent = parentName.trim().toLowerCase();
        const normalizedChild = childName.trim().toLowerCase();

        if (normalizedChild.startsWith(normalizedParent)) {
            let trimmed = childName.slice(parentName.length).trim();
            trimmed = trimmed.replace(/^[-:•\s]+/, '');
            return trimmed || childName;
        }

        return childName;
    }

    buildLinkedSummaryLabel(group, device) {
        if (!device) {
            return '';
        }

        const groupName = group?.name || group?.primary?.name || '';
        const friendlyName = device.name || device.id;
        const shortName = this.simplifyLinkedName(groupName, friendlyName);
        const descriptor = this.describeLinkedEntity(device.domain || device.type, device);

        if (!shortName || shortName.toLowerCase() === groupName.toLowerCase()) {
            return descriptor;
        }

        return `${shortName} (${descriptor})`;
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
        const attributes = (raw.attributes && typeof raw.attributes === 'object') ? { ...raw.attributes } : {};
        const entityCategory = raw.entity_category || attributes.entity_category || null;
        const groupId = this.deriveDeviceGroupId(raw, id);
        const friendlyName = raw.name || attributes.friendly_name || id;
        let isAccessory = entityCategory && ['diagnostic', 'config', 'auxiliary'].includes((entityCategory || '').toLowerCase());

        if (!isAccessory && groupId && groupId !== id) {
            isAccessory = ['sensor', 'binary_sensor'].includes(domain);
        }

        return {
            id,
            name: friendlyName,
            type: domain,
            domain,
            state,
            rawState: raw.state || {},
            attributes,
            groupId,
            entityCategory,
            isAccessory,
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
    
    createDeviceCard(device, options = {}) {
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

        const groupId = options?.groupId ? this.normalizeGroupId(options.groupId) : null;
        const isHidden = options?.hidden === true;
        const visibilityButton = groupId ? `
            <button type="button" class="device-visibility-toggle" data-group-id="${groupId}" data-state="${isHidden ? 'hidden' : 'visible'}" aria-label="${isHidden ? 'Unhide device' : 'Hide device'}" title="${isHidden ? 'Unhide device' : 'Hide device'}">
                <span class="icon" data-lucide="${isHidden ? 'eye' : 'eye-off'}"></span>
            </button>
        ` : '';

        card.innerHTML = `
            <div class="device-header">
                <div class="device-info">
                    <h3>${device.name}</h3>
                    <div class="device-type">${device.domain || device.type}</div>
                </div>
                <div class="device-actions">
                    ${visibilityButton}
                    <div class="device-status ${statusClass}"></div>
                </div>
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
                        onclick="event.stopPropagation(); app.refreshDevice('${device.id}')">
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
                    onclick="event.stopPropagation(); app.toggleDevice('${device.id}')">
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
            const commandParameters = parameters || {};

            const controlPayload = {
                device_id: device.id,
                action,
                parameters: commandParameters
            };

            const userId = this.currentUser?.id?.trim();
            if (userId) {
                controlPayload.user_id = userId;
            }

            const profileId = this.currentUser?.profileId?.trim();
            if (profileId) {
                controlPayload.profile_id = profileId;
            }

            const result = await this.apiCall('/devices/control', {
                method: 'POST',
                body: JSON.stringify(controlPayload)
            });

            if (result?.success === false) {
                const message = result?.error || result?.message || 'Unknown error';
                throw new Error(message);
            }

            const mergedState = this.mergeDeviceControlResult(
                device,
                action,
                commandParameters,
                result?.device_state
            );

            const normalized = this.normalizeDevice({
                ...device,
                state: mergedState,
                available: result?.success !== false,
                last_updated: new Date().toISOString()
            });

            this.devices.set(normalized.id, normalized);
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

    mergeDeviceControlResult(device, action, parameters, apiState) {
        const domain = (device?.domain || device?.type || '').toLowerCase();
        const previousState = {
            ...(typeof device?.rawState === 'object' && device.rawState !== null ? device.rawState : {}),
            ...(typeof device?.state === 'object' && device.state !== null ? device.state : {})
        };

        const mergedState = { ...previousState };

        if (apiState && typeof apiState === 'object') {
            Object.keys(apiState).forEach((key) => {
                mergedState[key] = apiState[key];
            });
        }

        this.applyOptimisticState(domain, mergedState, previousState, action, parameters);

        if (mergedState.raw_state === undefined) {
            if (domain === 'climate' && typeof mergedState.mode === 'string') {
                mergedState.raw_state = mergedState.mode;
            } else if (typeof mergedState.on === 'boolean') {
                mergedState.raw_state = mergedState.on ? 'on' : 'off';
            } else if (domain === 'lock' && typeof mergedState.locked === 'boolean') {
                mergedState.raw_state = mergedState.locked ? 'locked' : 'unlocked';
            } else if (domain === 'cover' && typeof mergedState.open === 'boolean') {
                mergedState.raw_state = mergedState.open ? 'open' : 'closed';
            }
        }

        if (mergedState.active === undefined && typeof mergedState.on === 'boolean') {
            mergedState.active = mergedState.on;
        }

        return mergedState;
    }

    applyOptimisticState(domain, targetState, previousState, action, parameters = {}) {
        const assign = (key, value) => {
            if (value === undefined || value === null) {
                return;
            }
            targetState[key] = value;
        };

        const assignOnState = (isOn) => {
            if (typeof isOn !== 'boolean') {
                return;
            }
            assign('on', isOn);
            assign('active', isOn);
            assign('raw_state', isOn ? 'on' : 'off');
        };

        const previousOn = this.resolveBooleanDeviceState(previousState, domain);
        const currentOn = this.resolveBooleanDeviceState(targetState, domain);

        const coalesceToggle = () => {
            if (typeof currentOn === 'boolean') {
                return !currentOn;
            }
            if (typeof previousOn === 'boolean') {
                return !previousOn;
            }
            return !(targetState.raw_state === 'on');
        };

        switch (domain) {
            case 'light':
            case 'switch':
            case 'fan':
            case 'humidifier': {
                if (action === 'turn_on') {
                    assignOnState(true);
                } else if (action === 'turn_off') {
                    assignOnState(false);
                } else if (action === 'toggle') {
                    assignOnState(coalesceToggle());
                } else if (action === 'set_brightness') {
                    const hasPct = typeof parameters.brightness_pct === 'number';
                    const normalizedPct = hasPct
                        ? Math.max(0, Math.min(100, Number(parameters.brightness_pct)))
                        : undefined;
                    const normalizedValue = typeof parameters.brightness === 'number'
                        ? Math.max(0, Math.min(255, Number(parameters.brightness)))
                        : (normalizedPct !== undefined ? Math.round((normalizedPct / 100) * 255) : undefined);

                    assign('brightness_pct', normalizedPct);
                    assign('brightness', normalizedValue);
                    if (normalizedPct !== undefined) {
                        assignOnState(normalizedPct > 0);
                    } else if (normalizedValue !== undefined) {
                        assignOnState(normalizedValue > 0);
                    } else {
                        assignOnState(true);
                    }
                } else if (action === 'set_mode' && domain === 'fan') {
                    assign('preset_mode', parameters.mode);
                    if (typeof parameters.mode === 'string') {
                        assignOnState(parameters.mode.toLowerCase() !== 'off');
                    }
                }
                break;
            }
            case 'climate': {
                if (action === 'set_mode') {
                    const desiredMode = typeof parameters.mode === 'string'
                        ? parameters.mode
                        : (targetState.mode || previousState.mode || 'auto');
                    assign('mode', desiredMode);
                    assign('raw_state', desiredMode);
                    if (typeof desiredMode === 'string') {
                        const lowered = desiredMode.toLowerCase();
                        const isActive = lowered !== 'off';
                        assign('on', isActive);
                        assign('active', isActive);
                    }
                } else if (action === 'turn_off') {
                    assign('mode', 'off');
                    assign('raw_state', 'off');
                    assign('on', false);
                    assign('active', false);
                } else if (action === 'turn_on') {
                    const fallbackMode = previousState.mode && previousState.mode !== 'off'
                        ? previousState.mode
                        : (targetState.mode && targetState.mode !== 'off' ? targetState.mode : 'cool');
                    assign('mode', fallbackMode);
                    assign('raw_state', fallbackMode);
                    assign('on', true);
                    assign('active', true);
                } else if (action === 'set_temperature') {
                    const temp = typeof parameters.temperature === 'number'
                        ? parameters.temperature
                        : (typeof parameters.target_temperature === 'number' ? parameters.target_temperature : undefined);
                    assign('target_temperature', temp);
                }
                break;
            }
            case 'lock': {
                const prevLocked = this.resolveLockedState(previousState, targetState);
                if (action === 'turn_on') {
                    assign('locked', false);
                    assign('raw_state', 'unlocked');
                    assign('active', false);
                } else if (action === 'turn_off') {
                    assign('locked', true);
                    assign('raw_state', 'locked');
                    assign('active', true);
                } else if (action === 'toggle') {
                    const nextLocked = !(typeof prevLocked === 'boolean' ? prevLocked : targetState.locked === true);
                    assign('locked', nextLocked);
                    assign('raw_state', nextLocked ? 'locked' : 'unlocked');
                    assign('active', nextLocked);
                }
                break;
            }
            case 'cover': {
                if (action === 'turn_on' || action === 'open_cover') {
                    assign('open', true);
                    assign('raw_state', 'open');
                    assign('active', true);
                } else if (action === 'turn_off' || action === 'close_cover') {
                    assign('open', false);
                    assign('raw_state', 'closed');
                    assign('active', false);
                } else if (action === 'toggle') {
                    const currentlyOpen = typeof targetState.open === 'boolean' ? targetState.open : targetState.raw_state === 'open';
                    const nextOpen = !currentlyOpen;
                    assign('open', nextOpen);
                    assign('raw_state', nextOpen ? 'open' : 'closed');
                    assign('active', nextOpen);
                }

                if (typeof parameters.position === 'number') {
                    assign('position', Math.max(0, Math.min(100, Number(parameters.position))));
                }
                break;
            }
            default: {
                if (action === 'turn_on') {
                    assignOnState(true);
                } else if (action === 'turn_off') {
                    assignOnState(false);
                } else if (action === 'toggle') {
                    assignOnState(coalesceToggle());
                }
            }
        }
    }

    resolveBooleanDeviceState(state, domain) {
        if (!state || typeof state !== 'object') {
            return undefined;
        }

        if (typeof state.on === 'boolean') {
            return state.on;
        }

        if (typeof state.active === 'boolean') {
            return state.active;
        }

        if (typeof state.raw_state === 'string') {
            const raw = state.raw_state.toLowerCase();
            if (['on', 'open', 'cool', 'heat', 'heating', 'cooling', 'unlocked'].includes(raw)) {
                return true;
            }
            if (['off', 'closed', 'locked', 'idle'].includes(raw)) {
                return false;
            }
        }

        if (domain === 'climate' && typeof state.mode === 'string') {
            return state.mode.toLowerCase() !== 'off';
        }

        if (domain === 'cover' && typeof state.open === 'boolean') {
            return state.open;
        }

        if (domain === 'lock' && typeof state.locked === 'boolean') {
            return !state.locked;
        }

        return undefined;
    }

    resolveLockedState(previousState, targetState) {
        if (previousState && typeof previousState.locked === 'boolean') {
            return previousState.locked;
        }

        if (targetState && typeof targetState.locked === 'boolean') {
            return targetState.locked;
        }

        const raw = (previousState && previousState.raw_state) || (targetState && targetState.raw_state);
        if (typeof raw === 'string') {
            const lowered = raw.toLowerCase();
            if (lowered === 'locked') {
                return true;
            }
            if (lowered === 'unlocked') {
                return false;
            }
        }

        return undefined;
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
        const normalized = this.normalizeDeviceFilter(filter);
        this.currentDeviceFilter = normalized;
        this.renderDevices();
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
                this.loadHomeAssistantConfig();
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
        const haStatusDetail = document.getElementById('haStatusDetail');
        if (!haStatus) return;

        const setBadgeTitle = (message) => {
            if (!message) {
                haStatus.removeAttribute('title');
                haStatus.removeAttribute('aria-label');
                return;
            }

            haStatus.setAttribute('title', message);
            haStatus.setAttribute('aria-label', `Home Assistant status ${haStatus.textContent || ''}. ${message}`.trim());
        };

        const hideDetail = () => {
            if (haStatusDetail) {
                haStatusDetail.textContent = '';
                haStatusDetail.className = 'status-detail';
            }
            setBadgeTitle('');
        };

        const showDetail = (message, variant = 'warning') => {
            if (haStatusDetail) {
                haStatusDetail.textContent = message;
                haStatusDetail.className = `status-detail is-visible status-detail--${variant}`;
            }
            setBadgeTitle(message);
        };

        try {
            const health = await this.apiCall('/health');
            const dependency = health?.dependencies?.home_assistant || {};
            const status = (dependency.status || 'unknown').toLowerCase();

            const detailParts = [];
            if (dependency.message) {
                detailParts.push(dependency.message);
            }
            if (dependency.error && dependency.error !== dependency.message) {
                detailParts.push(dependency.error);
            }
            if (dependency.action_required) {
                detailParts.push(dependency.action_required);
            }

            const defaultDetail = 'No diagnostic details provided. Review Home Assistant credentials and API logs.';
            const detailText = (detailParts.length ? detailParts : [defaultDetail])
                .filter(Boolean)
                .join(' • ');

            if (status === 'healthy') {
                haStatus.textContent = 'Online';
                haStatus.className = 'status-badge online';
                hideDetail();
            } else if (status === 'degraded' || status === 'warn' || status === 'warning') {
                haStatus.textContent = 'Degraded';
                haStatus.className = 'status-badge degraded';
                showDetail(detailText, 'warning');
            } else {
                haStatus.textContent = 'Offline';
                haStatus.className = 'status-badge offline';
                showDetail(detailText, 'error');
            }
        } catch (error) {
            console.error('Health check failed:', error);
            haStatus.textContent = 'Offline';
            haStatus.className = 'status-badge offline';
            const fallbackDetail = 'Unable to reach Home Automation API health endpoint. Verify the scenario is running and reachable.';
            showDetail(fallbackDetail, 'error');
        }
    }

    initializeHomeAssistantConfigForm() {
        const form = document.getElementById('homeAssistantConfigForm');
        if (form) {
            form.addEventListener('submit', (event) => {
                event.preventDefault();
                this.saveHomeAssistantConfig();
            });
        }

        const testButton = document.getElementById('haTestConnection');
        if (testButton) {
            testButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.testHomeAssistantConfig();
            });
        }

        const clearToken = document.getElementById('haClearToken');
        const tokenInput = document.getElementById('haToken');
        if (clearToken) {
            clearToken.addEventListener('change', () => {
                if (clearToken.checked && tokenInput) {
                    tokenInput.value = '';
                }
            });
        }

        if (tokenInput && clearToken) {
            tokenInput.addEventListener('input', () => {
                if (tokenInput.value.trim().length > 0) {
                    clearToken.checked = false;
                }
            });
        }

        const autoProvisionButton = document.getElementById('haAutoProvision');
        if (autoProvisionButton) {
            autoProvisionButton.addEventListener('click', () => {
                this.autoProvisionHomeAssistant();
            });
        }

        const openDashboardButton = document.getElementById('haOpenDashboard');
        if (openDashboardButton) {
            openDashboardButton.addEventListener('click', () => {
                this.openHomeAssistantDashboard();
            });
        }

        const showGuideButton = document.getElementById('haShowGuide');
        if (showGuideButton) {
            showGuideButton.addEventListener('click', () => {
                const details = document.getElementById('haSetupDetails');
                if (details) {
                    details.open = true;
                    details.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                }
            });
        }
    }

    async loadHomeAssistantConfig() {
        try {
            const config = await this.apiCall('/integrations/home-assistant/config');
            if (!config || typeof config !== 'object') {
                throw new Error('Invalid configuration response');
            }

            this.homeAssistantConfig = {
                baseUrl: typeof config.base_url === 'string' ? config.base_url : '',
                tokenConfigured: config.token_configured === true,
                mockMode: config.mock_mode === true,
                status: typeof config.status === 'string' ? config.status : 'unknown',
                message: typeof config.message === 'string' ? config.message : '',
                actionRequired: typeof config.action_required === 'string' ? config.action_required : '',
                error: typeof config.error === 'string' ? config.error : ''
            };

            this.populateHomeAssistantConfigForm(config);
            this.updateHomeAssistantStatus(config);
        } catch (error) {
            console.error('Failed to load Home Assistant configuration:', error);
            this.showHomeAssistantFeedback('Failed to load configuration. Check API connectivity.', 'error');
        }
    }

    populateHomeAssistantConfigForm(config = {}) {
        const baseInput = document.getElementById('haBaseUrl');
        if (baseInput) {
            baseInput.value = typeof config.base_url === 'string'
                ? config.base_url
                : (this.homeAssistantConfig.baseUrl || '');
        }

        const tokenInput = document.getElementById('haToken');
        if (tokenInput) {
            tokenInput.value = '';
            const hasToken = config.token_configured === true || this.homeAssistantConfig.tokenConfigured;
            tokenInput.placeholder = hasToken
                ? 'Token configured (leave blank to keep existing)'
                : 'Enter Home Assistant long-lived token';
        }

        const mockCheckbox = document.getElementById('haMockMode');
        if (mockCheckbox) {
            mockCheckbox.checked = config.mock_mode === true || this.homeAssistantConfig.mockMode === true;
        }

        const clearToken = document.getElementById('haClearToken');
        if (clearToken) {
            clearToken.checked = false;
        }

        this.showHomeAssistantFeedback('', 'info');
    }

    updateHomeAssistantStatus(config = {}) {
        const haStatus = document.getElementById('haStatus');
        const haStatusDetail = document.getElementById('haStatusDetail');
        if (!haStatus) {
            return;
        }

        const status = String(config.status || 'unknown').toLowerCase();
        let badgeText = 'Unknown';
        let badgeClass = 'status-badge';

        if (status === 'healthy') {
            badgeText = 'Online';
            badgeClass += ' online';
        } else if (status === 'degraded' || status === 'unconfigured') {
            badgeText = status === 'unconfigured' ? 'Needs Setup' : 'Degraded';
            badgeClass += ' degraded';
        } else {
            badgeText = 'Offline';
            badgeClass += ' offline';
        }

        haStatus.textContent = badgeText;
        haStatus.className = badgeClass;

        let detailText = '';
        if (haStatusDetail) {
            const detailParts = [];
            if (config.message) {
                detailParts.push(config.message);
            }
            if (config.action_required && detailParts.indexOf(config.action_required) === -1) {
                detailParts.push(config.action_required);
            }
            if (config.error) {
                detailParts.push(config.error);
            }

            if (detailParts.length > 0) {
                detailText = detailParts.join(' • ');
                const variant = status === 'healthy'
                    ? ''
                    : (status === 'degraded' || status === 'unconfigured')
                        ? 'status-detail--warning'
                        : 'status-detail--error';
                haStatusDetail.textContent = detailText;
                haStatusDetail.className = ['status-detail', 'is-visible', variant].filter(Boolean).join(' ');
            } else {
                haStatusDetail.textContent = '';
                haStatusDetail.className = 'status-detail';
            }
        }

        const ariaParts = [
            `Home Assistant status ${badgeText}`,
            detailText
        ].filter(Boolean);
        haStatus.setAttribute('title', detailText || badgeText);
        haStatus.setAttribute('aria-label', ariaParts.join('. '));

        const statusCard = document.getElementById('homeAssistantCard');
        if (statusCard) {
            statusCard.dataset.status = status;
        }
    }

    showHomeAssistantFeedback(message, variant = 'info') {
        const feedback = document.getElementById('haConfigFeedback');
        if (!feedback) {
            return;
        }

        const classes = {
            success: 'form-feedback is-success',
            error: 'form-feedback is-error',
            warning: 'form-feedback is-warning',
            info: 'form-feedback'
        };

        feedback.className = classes[variant] || classes.info;
        feedback.textContent = message || '';
    }

    collectHomeAssistantPayload({ requireToken = false, testOnly = false } = {}) {
        const baseInput = document.getElementById('haBaseUrl');
        const tokenInput = document.getElementById('haToken');
        const mockCheckbox = document.getElementById('haMockMode');
        const clearToken = document.getElementById('haClearToken');

        const payload = {
            base_url: baseInput?.value?.trim() || '',
            mock_mode: !!(mockCheckbox && mockCheckbox.checked)
        };

        const tokenValue = tokenInput?.value?.trim() || '';
        const clearTokenChecked = !!(clearToken && clearToken.checked);
        if (clearTokenChecked) {
            payload.clear_token = true;
        }
        if (tokenValue) {
            payload.token = tokenValue;
        }
        if (testOnly) {
            payload.test_only = true;
        }

        const errors = [];
        if (!payload.base_url) {
            errors.push('Home Assistant URL is required.');
        }
        const tokenMissing = !clearTokenChecked && !tokenValue && !this.homeAssistantConfig.tokenConfigured;
        if (requireToken && tokenMissing) {
            errors.push('Provide a Home Assistant token or clear the stored token.');
        }

        return { payload, errors };
    }

    async saveHomeAssistantConfig() {
        const { payload, errors } = this.collectHomeAssistantPayload({ requireToken: true, testOnly: false });
        if (errors.length) {
            this.showHomeAssistantFeedback(errors.join(' '), 'error');
            return;
        }

        this.showHomeAssistantFeedback('Saving configuration...', 'info');

        try {
            const response = await this.apiCall('/integrations/home-assistant/config', {
                method: 'POST',
                body: JSON.stringify(payload)
            });

            this.homeAssistantConfig = {
                baseUrl: response?.base_url || payload.base_url,
                tokenConfigured: response?.token_configured === true,
                mockMode: response?.mock_mode === true,
                status: response?.status || 'unknown',
                message: response?.message || '',
                actionRequired: response?.action_required || '',
                error: response?.error || ''
            };

            this.populateHomeAssistantConfigForm(response);
            this.updateHomeAssistantStatus(response);

            const feedbackMessage = response?.message
                || (response?.status === 'healthy'
                    ? 'Configuration saved and Home Assistant is reachable.'
                    : 'Configuration saved. Review the status message for next steps.');
            const feedbackVariant = response?.status === 'healthy' ? 'success' : 'warning';
            this.showHomeAssistantFeedback(feedbackMessage, feedbackVariant);
        } catch (error) {
            console.error('Failed to save Home Assistant configuration:', error);
            this.showHomeAssistantFeedback(`Failed to save configuration: ${error.message}`, 'error');
        }
    }

    async testHomeAssistantConfig() {
        const { payload, errors } = this.collectHomeAssistantPayload({ requireToken: false, testOnly: true });
        if (errors.length) {
            this.showHomeAssistantFeedback(errors.join(' '), 'error');
            return;
        }

        this.showHomeAssistantFeedback('Testing connection...', 'info');

        try {
            const response = await this.apiCall('/integrations/home-assistant/config', {
                method: 'POST',
                body: JSON.stringify(payload)
            });

            this.updateHomeAssistantStatus(response);

            const variant = response?.status === 'healthy' ? 'success' : 'warning';
            const summary = response?.message
                || (response?.status === 'healthy' ? 'Home Assistant connection verified.' : 'Connection test completed. Review status details.');
            this.showHomeAssistantFeedback(summary, variant);
        } catch (error) {
            console.error('Home Assistant connection test failed:', error);
            this.showHomeAssistantFeedback(`Connection test failed: ${error.message}`, 'error');
        }
    }

    openHomeAssistantDashboard() {
        let target = this.homeAssistantConfig?.baseUrl || '';
        const baseInput = document.getElementById('haBaseUrl');
        if (!target && baseInput) {
            target = baseInput.value.trim();
        }

        if (!target) {
            target = 'http://localhost:8123';
        }

        try {
            const parsed = new URL(target, window.location.href);
            window.open(parsed.toString(), '_blank', 'noopener');
        } catch (error) {
            const fallback = target.startsWith('http') ? target : `http://${target}`;
            window.open(fallback, '_blank', 'noopener');
        }
    }

    async autoProvisionHomeAssistant() {
        const button = document.getElementById('haAutoProvision');
        if (button) {
            button.disabled = true;
        }

        const baseInput = document.getElementById('haBaseUrl');
        const baseUrl = baseInput?.value?.trim() || this.homeAssistantConfig.baseUrl || '';

        this.showHomeAssistantFeedback('Attempting automatic Home Assistant provisioning...', 'info');

        try {
            const response = await this.apiCall('/integrations/home-assistant/provision', {
                method: 'POST',
                body: JSON.stringify({
                    base_url: baseUrl,
                    client_name: 'Home Automation Intelligence',
                    lifespan_days: 3650
                })
            });

            this.homeAssistantConfig = {
                baseUrl: response?.base_url || baseUrl,
                tokenConfigured: response?.token_configured === true,
                mockMode: response?.mock_mode === true,
                status: response?.status || 'unknown',
                message: response?.message || '',
                actionRequired: response?.action_required || '',
                error: response?.error || ''
            };

            this.populateHomeAssistantConfigForm(response);
            this.updateHomeAssistantStatus(response);

            const variant = response?.status === 'healthy' ? 'success' : 'warning';
            const summary = response?.message || 'Home Assistant connection established.';
            this.showHomeAssistantFeedback(summary, variant);

            const details = document.getElementById('haSetupDetails');
            if (details) {
                details.open = false;
            }
        } catch (error) {
            console.error('Home Assistant auto-provision failed:', error);
            this.showHomeAssistantFeedback(`Automatic provisioning failed: ${error.message}`, 'error');
        } finally {
            if (button) {
                button.disabled = false;
            }
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
