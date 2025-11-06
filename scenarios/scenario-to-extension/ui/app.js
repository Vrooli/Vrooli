/**
 * Browser Extension Generator UI
 * Main application logic for the scenario-to-extension web interface
 */

// API resolution is handled by api-resolver.js (loaded before this file)
// Functions available: resolveApiConfig(), applyApiResolution(), startApiResolutionWatcher()

let apiResolution = window.resolveApiConfig ? window.resolveApiConfig() : {
    base: 'http://127.0.0.1:3201/api/v1',
    root: 'http://127.0.0.1:3201',
    source: 'fallback',
    notes: 'API resolver not loaded'
};

if (typeof window !== 'undefined') {
    window.__scenarioToExtensionApi = apiResolution;
    window.__currentApiResolution = apiResolution;
}

// Configuration
const CONFIG = {
    API_BASE: apiResolution.base,
    API_ROOT: apiResolution.root,
    API_SOURCE: apiResolution.source,
    API_NOTES: apiResolution.notes,
    POLL_INTERVAL: 5000, // 5 seconds (build status polling)
    STATUS_REFRESH_INTERVAL: 30000, // 30 seconds (system health checks)
    NOTIFICATION_TIMEOUT: 5000,
    DEBUG: false
};

// Application state
const state = {
    currentTab: 'generator',
    selectedTemplate: 'full',
    builds: [],
    templates: [],
    systemStatus: null,
    isLoading: false
};

// ============================================
// Debug & Development Utilities
// ============================================

// Conditional logging - no-op in production, full logging in debug mode
const log = CONFIG.DEBUG
    ? (...args) => console.log('[ExtensionGenerator]', ...args)
    : () => {}; // No-op function - zero overhead in production

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// ============================================
// String Formatting Utilities
// ============================================

function formatDate(dateString) {
    return new Date(dateString).toLocaleString();
}

function formatDuration(startTime, endTime) {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const diff = end - start;

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
}

function formatStatusLabel(value) {
    if (typeof value !== 'string' || !value) {
        return 'Unknown';
    }

    return value
        .split(/[_\s-]+/)
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ');
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function formatDisplayValue(value) {
    if (typeof value === 'boolean') {
        return value ? 'Available' : 'Unavailable';
    }

    if (value === null || value === undefined) {
        return 'Unknown';
    }

    if (value instanceof Date) {
        return value.toLocaleString();
    }

    if (typeof value === 'object') {
        try {
            return JSON.stringify(value);
        } catch (error) {
            log('Failed to stringify value for display', error);
        }
    }

    return String(value);
}

// ============================================
// API Integration (resolution handled by api-resolver.js)
// ============================================

// Note: The following functions are now provided by api-resolver.js:
// - resolveApiConfig()
// - applyApiResolution()
// - startApiResolutionWatcher()
// This section only contains app-specific API context functions.

// ============================================
// Icon Rendering Utilities
// ============================================

function createIconMarkup(name, { className = '', size, title } = {}) {
    if (!name) {
        return '';
    }

    const classes = ['lucide-icon', className].filter(Boolean).join(' ');
    const styleAttr = size ? ` style="width:${size}px;height:${size}px"` : '';
    const titleAttr = title ? ` aria-label="${title}"` : ' aria-hidden="true"';
    const classAttr = classes ? ` class="${classes}"` : '';

    return `<span${classAttr} data-lucide="${name}"${titleAttr}${styleAttr}></span>`;
}

function renderLucideIcons(scope = document) {
    if (!window.lucide?.createIcons || !scope) {
        return;
    }

    const elements = [];

    if (scope !== document && scope.matches?.('[data-lucide]')) {
        elements.push(scope);
    }

    const descendants = scope.querySelectorAll?.('[data-lucide]') ?? [];
    descendants.forEach((el) => elements.push(el));

    if (elements.length > 0) {
        window.lucide.createIcons({ elements });
    }
}

// ============================================
// API Client
// ============================================

class APIClient {
    static async request(method, endpoint, data = null) {
        const url = `${CONFIG.API_BASE}${endpoint}`;
        
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };
        
        if (data) {
            options.body = JSON.stringify(data);
        }
        
        log(`API ${method} ${url}`, data);
        
        try {
            const response = await fetch(url, options);
            
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
            }
            
            const result = await response.json();
            log(`API Response:`, result);
            return result;
        } catch (error) {
            log('API Error:', error);
            throw error;
        }
    }
    
    static async get(endpoint) {
        return this.request('GET', endpoint);
    }
    
    static async post(endpoint, data) {
        return this.request('POST', endpoint, data);
    }
    
    static async put(endpoint, data) {
        return this.request('PUT', endpoint, data);
    }
    
    static async delete(endpoint) {
        return this.request('DELETE', endpoint);
    }
}

// ============================================
// UI Components
// ============================================

class NotificationManager {
    static show(type, title, message, timeout = CONFIG.NOTIFICATION_TIMEOUT) {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        const iconName = {
            success: 'check-circle-2',
            warning: 'alert-triangle',
            error: 'circle-x',
            info: 'info'
        }[type] || 'bell';
        notification.innerHTML = `
            <div class="notification-header">
                <div class="notification-title">
                    ${createIconMarkup(iconName, { className: 'notification-icon' })}
                    <span>${title}</span>
                </div>
                <button class="notification-close" type="button" aria-label="Dismiss notification">
                    ${createIconMarkup('x', { className: 'notification-close-icon' })}
                </button>
            </div>
            <div class="notification-message">${message}</div>
        `;
        
        const container = document.getElementById('notifications');
        container.appendChild(notification);

        renderLucideIcons(notification);
        
        // Close button handler
        notification.querySelector('.notification-close').addEventListener('click', () => {
            notification.remove();
        });
        
        // Auto-remove after timeout
        if (timeout > 0) {
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.remove();
                }
            }, timeout);
        }
        
        return notification;
    }
    
    static success(title, message) {
        return this.show('success', title, message);
    }
    
    static warning(title, message) {
        return this.show('warning', title, message);
    }
    
    static error(title, message) {
        return this.show('error', title, message, 8000);
    }
    
    static info(title, message) {
        return this.show('info', title, message);
    }
}

class LoadingManager {
    static show(title = 'Processing...', message = 'This may take a moment...') {
        const overlay = document.getElementById('loading-overlay');
        document.getElementById('loading-title').textContent = title;
        document.getElementById('loading-message').textContent = message;
        overlay.classList.remove('hidden');
    }
    
    static hide() {
        document.getElementById('loading-overlay').classList.add('hidden');
    }
    
    static setButtonLoading(buttonId, loading = true) {
        const button = document.getElementById(buttonId);
        const spinner = button.querySelector('.spinner');
        
        if (loading) {
            button.disabled = true;
            if (spinner) spinner.classList.remove('hidden');
        } else {
            button.disabled = false;
            if (spinner) spinner.classList.add('hidden');
        }
    }
}

// ============================================
// Tab Management
// ============================================

function initializeTabs() {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const targetTab = button.getAttribute('data-tab');
            switchTab(targetTab);
        });
    });
}

function switchTab(tabName) {
    state.currentTab = tabName;
    
    // Update buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.toggle('active', btn.getAttribute('data-tab') === tabName);
    });
    
    // Update content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `tab-${tabName}`);
    });
    
    // Load tab-specific data
    switch (tabName) {
        case 'builds':
            loadBuilds();
            break;
        case 'templates':
            loadTemplates();
            break;
        case 'testing':
            // Testing tab is ready by default
            break;
    }
}

// ============================================
// System Status
// ============================================

async function refreshStatus() {
    const indicator = document.getElementById('status-indicator');
    if (indicator) {
        indicator.dataset.status = 'loading';
        indicator.setAttribute('aria-label', 'Loading system status...');
    }

    try {
        const status = await APIClient.get('/health');
        state.systemStatus = status;
        updateStatusIndicator(status);
    } catch (error) {
        log('Status check failed:', error);
        const fallbackStatus = {
            status: 'error',
            message: error?.message || 'Status check failed'
        };
        state.systemStatus = fallbackStatus;
        updateStatusIndicator(fallbackStatus);
    }
}

function updateStatusIndicator(status) {
    const indicator = document.getElementById('status-indicator');
    const popup = document.getElementById('status-popup');
    const dot = indicator?.querySelector('.status-dot');
    const stateValue = status?.status || 'error';

    if (!indicator) {
        return;
    }

    indicator.dataset.status = stateValue;

    const statusLabel = formatStatusLabel(stateValue);
    const timestamp = status?.timestamp ? formatDate(status.timestamp) : null;
    const ariaParts = [`${statusLabel} status`];
    if (timestamp) {
        ariaParts.push(`Checked ${timestamp}`);
    }

    indicator.setAttribute('aria-label', `${ariaParts.join('. ')}. Click for details.`);
    indicator.setAttribute('aria-expanded', popup?.classList.contains('is-open') ? 'true' : 'false');

    if (dot) {
        dot.classList.toggle('is-loading', stateValue === 'loading' || stateValue === 'building');
    }

    renderStatusPopupContent(status);
}

function syncApiEndpointField(previousRoot) {
    const apiInput = document.getElementById('api-endpoint');
    if (!apiInput) {
        return;
    }

    const resolvedRoot = (CONFIG.API_ROOT || '').trim();
    if (!resolvedRoot) {
        return;
    }

    const currentValue = (apiInput.value || '').trim();
    const autoManaged =
        !currentValue ||
        currentValue === previousRoot ||
        apiInput.dataset.prefilled === 'true' ||
        apiInput.dataset.autoGenerated === 'true';

    if (autoManaged) {
        apiInput.value = resolvedRoot;
        apiInput.placeholder = resolvedRoot;
        apiInput.dataset.prefilled = 'true';
        apiInput.dataset.autoGenerated = 'true';
    }
}

function applyApiContext() {
    syncApiEndpointField();

    const healthLink = document.getElementById('api-health-link');
    if (healthLink && CONFIG.API_BASE) {
        healthLink.href = `${CONFIG.API_BASE}/health`;
    }

    renderStatusPopupContent();
}

function renderStatusPopupContent(status = state.systemStatus) {
    const popupContent = document.getElementById('status-popup-content');
    if (!popupContent) {
        return;
    }

    const effectiveStatus = status || state.systemStatus;

    if (!effectiveStatus) {
        popupContent.innerHTML = '<p class="status-popup-empty">Status information not available yet.</p>';
        return;
    }

    const statusState = effectiveStatus.status || 'error';
    const statusLabel = formatStatusLabel(statusState);
    const service = effectiveStatus.service || 'Scenario API';
    const scenarioName = effectiveStatus.scenario || 'Unknown';
    const version = effectiveStatus.version || 'Unknown';
    const timestamp = effectiveStatus.timestamp ? formatDate(effectiveStatus.timestamp) : null;
    const message = effectiveStatus.message;
    const resources = effectiveStatus.resources || {};
    const stats = effectiveStatus.stats || {};

    const resourcesMarkup = Object.keys(resources).length > 0
        ? Object.entries(resources)
            .map(([key, value]) => {
                const label = formatStatusLabel(key);
                const isBoolean = typeof value === 'boolean';
                const badgeClass = value === true
                    ? 'status-popup-badge status-popup-badge--ok'
                    : value === false
                        ? 'status-popup-badge status-popup-badge--error'
                        : 'status-popup-badge';
                const displayValue = isBoolean ? (value ? 'Available' : 'Unavailable') : formatDisplayValue(value);
                return `
                    <li>
                        <span>${escapeHtml(label)}</span>
                        <span class="${badgeClass}">${escapeHtml(displayValue)}</span>
                    </li>
                `;
            })
            .join('')
        : '<li class="status-popup-empty">No resource data</li>';

    const statsMarkup = Object.keys(stats).length > 0
        ? Object.entries(stats)
            .map(([key, value]) => `
                <li>
                    <span>${escapeHtml(formatStatusLabel(key))}</span>
                    <span class="status-popup-value-strong">${escapeHtml(formatDisplayValue(value))}</span>
                </li>
            `)
            .join('')
        : '<li class="status-popup-empty">No stats available</li>';

    const apiSourceLabel = CONFIG.API_SOURCE ? formatStatusLabel(CONFIG.API_SOURCE) : '';
    const apiBase = CONFIG.API_BASE || '';

    popupContent.innerHTML = `
        <div class="status-popup-header">
            <span class="status-popup-dot" data-status="${escapeHtml(statusState)}"></span>
            <div>
                <div class="status-popup-title">${escapeHtml(statusLabel)}</div>
                <div class="status-popup-subtitle">${escapeHtml(service)}</div>
            </div>
        </div>
        <div class="status-popup-section">
            <div class="status-popup-section-title">Summary</div>
            <ul class="status-popup-list">
                <li><span>Scenario</span><span>${escapeHtml(scenarioName)}</span></li>
                <li><span>Version</span><span>${escapeHtml(version)}</span></li>
                ${timestamp ? `<li><span>Checked</span><span>${escapeHtml(timestamp)}</span></li>` : ''}
                ${message ? `<li><span>Message</span><span>${escapeHtml(message)}</span></li>` : ''}
            </ul>
        </div>
        <div class="status-popup-section">
            <div class="status-popup-section-title">Resources</div>
            <ul class="status-popup-list">${resourcesMarkup}</ul>
        </div>
        <div class="status-popup-section">
            <div class="status-popup-section-title">Stats</div>
            <ul class="status-popup-list">${statsMarkup}</ul>
        </div>
        <div class="status-popup-footer">
            <div class="status-popup-footer-info">
                ${apiSourceLabel ? `<span>${escapeHtml(apiSourceLabel)}</span>` : ''}
                ${apiBase ? `<span class="status-popup-footer-url">${escapeHtml(apiBase)}</span>` : ''}
            </div>
            ${apiBase ? `<a href="${escapeHtml(`${apiBase}/health`)}" target="_blank" rel="noopener">Open JSON</a>` : ''}
        </div>
    `;
}

function closeStatusPopup() {
    const popup = document.getElementById('status-popup');
    const trigger = document.getElementById('status-indicator');
    if (!popup || !trigger) {
        return;
    }

    popup.classList.remove('is-open');
    popup.setAttribute('aria-hidden', 'true');
    trigger.setAttribute('aria-expanded', 'false');
}

function toggleStatusPopup() {
    const popup = document.getElementById('status-popup');
    const trigger = document.getElementById('status-indicator');
    if (!popup || !trigger) {
        return;
    }

    const isOpen = popup.classList.toggle('is-open');
    popup.setAttribute('aria-hidden', isOpen ? 'false' : 'true');
    trigger.setAttribute('aria-expanded', isOpen ? 'true' : 'false');

    if (isOpen) {
        renderStatusPopupContent();
    }
}

function initializeStatusPopover() {
    const popup = document.getElementById('status-popup');
    const trigger = document.getElementById('status-indicator');

    if (!popup || !trigger) {
        return;
    }

    popup.setAttribute('aria-hidden', 'true');

    trigger.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        toggleStatusPopup();
    });

    document.addEventListener('click', (event) => {
        if (popup.contains(event.target) || event.target === trigger) {
            return;
        }
        closeStatusPopup();
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
            closeStatusPopup();
        }
    });
}

// ============================================
// Extension Generation
// ============================================

function initializeGenerator() {
    const apiEndpointInput = document.getElementById('api-endpoint');
    if (apiEndpointInput && CONFIG.API_ROOT) {
        apiEndpointInput.value = CONFIG.API_ROOT;
        apiEndpointInput.placeholder = CONFIG.API_ROOT;
        apiEndpointInput.dataset.prefilled = 'true';
        apiEndpointInput.dataset.autoGenerated = 'true';
    }

    // Template selection
    const templateCards = document.querySelectorAll('.template-card');
    templateCards.forEach(card => {
        card.addEventListener('click', () => {
            templateCards.forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            state.selectedTemplate = card.getAttribute('data-template');
        });
    });

    // Form submission
    const form = document.getElementById('extension-form');
    form.addEventListener('submit', handleExtensionGeneration);

    // Reset form button
    const resetBtn = document.getElementById('reset-form-btn');
    if (resetBtn) {
        resetBtn.addEventListener('click', resetForm);
    }

    // Auto-populate app name from scenario name
    const scenarioNameInput = document.getElementById('scenario-name');
    const appNameInput = document.getElementById('app-name');

    scenarioNameInput.addEventListener('input', debounce(() => {
        if (!appNameInput.value || appNameInput.dataset.autoGenerated) {
            const scenarioName = scenarioNameInput.value;
            const appName = scenarioName
                .split('-')
                .map(word => word.charAt(0).toUpperCase() + word.slice(1))
                .join(' ') + ' Extension';
            appNameInput.value = appName;
            appNameInput.dataset.autoGenerated = 'true';
        }
    }, 300));

    appNameInput.addEventListener('input', () => {
        delete appNameInput.dataset.autoGenerated;
    });
}

async function handleExtensionGeneration(event) {
    event.preventDefault();
    
    LoadingManager.setButtonLoading('generate-btn', true);
    
    try {
        // Collect form data
        const formData = new FormData(event.target);
        const permissions = Array.from(document.getElementById('permissions').selectedOptions)
            .map(option => option.value);
        const hostPermissions = document.getElementById('host-permissions').value
            .split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0);
        
        const requestData = {
            scenario_name: formData.get('scenario_name'),
            template_type: state.selectedTemplate,
            config: {
                app_name: formData.get('app_name'),
                app_description: formData.get('app_description'),
                api_endpoint: formData.get('api_endpoint'),
                permissions: permissions,
                host_permissions: hostPermissions,
                version: formData.get('version') || '1.0.0',
                author_name: formData.get('author') || 'Vrooli Scenario Generator',
                license: 'MIT',
                debug_mode: formData.get('debug_mode') === 'on'
            }
        };
        
        log('Generating extension with config:', requestData);
        
        // Make API request
        const result = await APIClient.post('/extension/generate', requestData);
        
        NotificationManager.success(
            'Extension Generation Started',
            `Build ${result.build_id} is now processing. Check the Builds tab for progress.`
        );
        
        // Switch to builds tab to show progress
        setTimeout(() => {
            switchTab('builds');
            startBuildPolling(result.build_id);
        }, 1000);
        
    } catch (error) {
        log('Extension generation failed:', error);
        NotificationManager.error(
            'Generation Failed',
            error.message || 'Unknown error occurred during extension generation'
        );
    } finally {
        LoadingManager.setButtonLoading('generate-btn', false);
    }
}

function resetForm() {
    document.getElementById('extension-form').reset();
    const apiInput = document.getElementById('api-endpoint');
    if (apiInput) {
        const nextValue = CONFIG.API_ROOT || '';
        apiInput.value = nextValue;
        apiInput.placeholder = nextValue;
        apiInput.dataset.prefilled = 'true';
        apiInput.dataset.autoGenerated = 'true';
    }
    document.getElementById('version').value = '1.0.0';
    document.getElementById('author').value = 'Vrooli Scenario Generator';
    
    // Reset template selection
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.remove('active');
    });
    document.querySelector('[data-template="full"]').classList.add('active');
    state.selectedTemplate = 'full';
    
    // Reset permissions
    const permissionsSelect = document.getElementById('permissions');
    Array.from(permissionsSelect.options).forEach(option => {
        option.selected = ['storage', 'activeTab'].includes(option.value);
    });
}

// ============================================
// Builds Management
// ============================================

async function loadBuilds() {
    const container = document.getElementById('builds-container');
    const loading = document.getElementById('builds-loading');
    const empty = document.getElementById('builds-empty');
    const list = document.getElementById('builds-list');
    
    try {
        loading.classList.remove('hidden');
        empty.classList.add('hidden');
        list.innerHTML = '';
        
        const response = await APIClient.get('/extension/builds');
        state.builds = response.builds || [];
        
        if (state.builds.length === 0) {
            loading.classList.add('hidden');
            empty.classList.remove('hidden');
        } else {
            loading.classList.add('hidden');
            renderBuilds();
        }
        
    } catch (error) {
        log('Failed to load builds:', error);
        loading.classList.add('hidden');
        NotificationManager.error('Load Failed', 'Failed to load extension builds');
    }
}

function renderBuilds() {
    const list = document.getElementById('builds-list');
    list.innerHTML = '';
    
    // Sort builds by creation date (newest first)
    const sortedBuilds = [...state.builds].sort((a, b) => 
        new Date(b.created_at) - new Date(a.created_at)
    );
    
    sortedBuilds.forEach(build => {
        const buildElement = createBuildElement(build);
        list.appendChild(buildElement);
    });

    renderLucideIcons(list);
}

function createBuildElement(build) {
    const div = document.createElement('div');
    div.className = 'build-item';
    
    const statusState = (build.status || 'unknown').toLowerCase();
    const statusMeta = {
        building: {
            icon: 'loader-2',
            label: 'Building',
            badgeClass: 'is-building',
            iconClass: 'is-spinning'
        },
        ready: {
            icon: 'check-circle-2',
            label: 'Ready',
            badgeClass: 'is-ready'
        },
        failed: {
            icon: 'circle-x',
            label: 'Failed',
            badgeClass: 'is-failed'
        }
    }[statusState] || {
        icon: 'circle-help',
        label: formatStatusLabel(build.status || 'Pending'),
        badgeClass: 'is-unknown'
    };

    const statusClasses = ['build-status', statusMeta.badgeClass, `status-${statusState}`]
        .filter(Boolean)
        .join(' ');
    const statusIconMarkup = createIconMarkup(statusMeta.icon, {
        className: ['status-icon', statusMeta.iconClass].filter(Boolean).join(' ')
    });
    
    div.innerHTML = `
        <div class="build-header">
            <div>
                <h4>${build.scenario_name}</h4>
                <div class="build-id">${build.build_id}</div>
            </div>
            <div class="${statusClasses}">
                ${statusIconMarkup}
                <span>${statusMeta.label}</span>
            </div>
        </div>
        
        <div class="build-info">
            <div class="build-info-item">
                <div class="build-info-label">Template</div>
                <div>${build.template_type}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Created</div>
                <div>${formatDate(build.created_at)}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Duration</div>
                <div>${formatDuration(build.created_at, build.completed_at)}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Path</div>
                <div class="text-muted">${build.extension_path}</div>
            </div>
        </div>
        
        ${build.error_log && build.error_log.length > 0 ? `
            <div class="build-errors">
                <h5>Errors:</h5>
                <ul>
                    ${build.error_log.map(error => `<li>${error}</li>`).join('')}
                </ul>
            </div>
        ` : ''}
        
        <div class="build-actions">
            ${build.status === 'ready' ? `
                <button class="btn btn-outline" data-action="test" data-build-id="${build.build_id}">
                    ${createIconMarkup('beaker', { className: 'btn-icon' })}
                    <span>Test Extension</span>
                </button>
                <button class="btn btn-outline" data-action="download" data-build-id="${build.build_id}">
                    ${createIconMarkup('download', { className: 'btn-icon' })}
                    <span>Download</span>
                </button>
            ` : ''}
            <button class="btn btn-outline" data-action="view-details" data-build-id="${build.build_id}">
                ${createIconMarkup('eye', { className: 'btn-icon' })}
                <span>View Details</span>
            </button>
        </div>
    `;
    
    renderLucideIcons(div);
    return div;
}

function startBuildPolling(buildId) {
    const pollBuild = async () => {
        try {
            const build = await APIClient.get(`/extension/status/${buildId}`);
            
            // Update the build in state
            const index = state.builds.findIndex(b => b.build_id === buildId);
            if (index >= 0) {
                state.builds[index] = build;
            } else {
                state.builds.unshift(build);
            }
            
            // Re-render builds
            if (state.currentTab === 'builds') {
                renderBuilds();
            }
            
            // Continue polling if still building
            if (build.status === 'building') {
                setTimeout(pollBuild, CONFIG.POLL_INTERVAL);
            } else {
                // Show completion notification
                if (build.status === 'ready') {
                    NotificationManager.success(
                        'Extension Ready',
                        `${build.scenario_name} extension has been generated successfully!`
                    );
                } else if (build.status === 'failed') {
                    NotificationManager.error(
                        'Generation Failed',
                        `${build.scenario_name} extension generation failed. Check build details for errors.`
                    );
                }
            }
            
        } catch (error) {
            log('Build polling error:', error);
            // Don't spam error notifications during polling
        }
    };
    
    pollBuild();
}

function testBuild(buildId) {
    const build = state.builds.find(b => b.build_id === buildId);
    if (build) {
        // Switch to testing tab and populate the extension path
        switchTab('testing');
        document.getElementById('test-extension-path').value = build.extension_path;
    }
}

function downloadBuild(buildId) {
    const build = state.builds.find(b => b.build_id === buildId);
    if (!build) {
        NotificationManager.error('Download Failed', 'Build not found');
        return;
    }

    if (build.status !== 'ready') {
        NotificationManager.warning('Not Ready', 'Build must be completed before downloading');
        return;
    }

    // Create download URL
    const downloadUrl = `${CONFIG.API_BASE}/extension/download/${buildId}`;

    // Create temporary link and trigger download
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = `${build.scenario_name}-v${build.config.version || '1.0.0'}.zip`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    NotificationManager.success(
        'Download Started',
        `Downloading ${build.scenario_name} extension as ZIP file`
    );
}

function viewBuildDetails(buildId) {
    const build = state.builds.find(b => b.build_id === buildId);
    if (!build) {
        NotificationManager.error('Error', 'Build not found');
        return;
    }

    const modal = document.getElementById('build-details-modal');
    const content = document.getElementById('build-details-content');

    // Format duration
    const duration = build.completed_at
        ? formatDuration(build.created_at, build.completed_at)
        : formatDuration(build.created_at, new Date());

    // Build the modal content
    content.innerHTML = `
        <div class="detail-section">
            <h3>
                ${createIconMarkup('info', { className: 'section-icon' })}
                Build Information
            </h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">Build ID</span>
                    <span class="detail-value">${escapeHtml(build.build_id)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Scenario</span>
                    <span class="detail-value">${escapeHtml(build.scenario_name)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Template</span>
                    <span class="detail-value">${escapeHtml(build.template_type)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Status</span>
                    <span class="detail-value">${formatStatusLabel(build.status)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Created</span>
                    <span class="detail-value">${formatDate(build.created_at)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Duration</span>
                    <span class="detail-value">${duration}</span>
                </div>
            </div>
        </div>

        <div class="detail-section">
            <h3>
                ${createIconMarkup('settings', { className: 'section-icon' })}
                Configuration
            </h3>
            <div class="detail-grid">
                <div class="detail-item">
                    <span class="detail-label">App Name</span>
                    <span class="detail-value">${escapeHtml(build.config.app_name)}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Version</span>
                    <span class="detail-value">${escapeHtml(build.config.version || '1.0.0')}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Author</span>
                    <span class="detail-value">${escapeHtml(build.config.author_name || 'N/A')}</span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">API Endpoint</span>
                    <span class="detail-value" style="word-break: break-all; font-size: 0.9rem;">
                        ${escapeHtml(build.config.api_endpoint)}
                    </span>
                </div>
                <div class="detail-item">
                    <span class="detail-label">Extension Path</span>
                    <span class="detail-value" style="word-break: break-all; font-size: 0.85rem; color: var(--text-muted);">
                        ${escapeHtml(build.extension_path)}
                    </span>
                </div>
            </div>
        </div>

        ${build.build_log && build.build_log.length > 0 ? `
            <div class="detail-section">
                <h3>
                    ${createIconMarkup('terminal', { className: 'section-icon' })}
                    Build Log (${build.build_log.length} entries)
                </h3>
                <div class="log-container">
                    ${build.build_log.map(entry => {
                        const logClass = entry.toLowerCase().includes('error') ? 'log-error'
                            : entry.toLowerCase().includes('warning') ? 'log-warning'
                            : entry.toLowerCase().includes('success') ? 'log-success'
                            : 'log-info';
                        return `<div class="log-entry ${logClass}">${escapeHtml(entry)}</div>`;
                    }).join('')}
                </div>
            </div>
        ` : '<div class="empty-message">No build log entries</div>'}

        ${build.error_log && build.error_log.length > 0 ? `
            <div class="detail-section">
                <h3>
                    ${createIconMarkup('alert-circle', { className: 'section-icon' })}
                    Error Log (${build.error_log.length} errors)
                </h3>
                <div class="log-container">
                    ${build.error_log.map(error =>
                        `<div class="log-entry log-error">${escapeHtml(error)}</div>`
                    ).join('')}
                </div>
            </div>
        ` : ''}
    `;

    // Render icons in the modal content
    renderLucideIcons(content);

    // Show modal
    modal.classList.remove('hidden');
    modal.setAttribute('aria-hidden', 'false');

    // Setup close handlers
    const closeButton = modal.querySelector('.modal-close');
    const backdrop = modal.querySelector('.modal-backdrop');

    const closeModal = () => {
        modal.classList.add('hidden');
        modal.setAttribute('aria-hidden', 'true');
    };

    closeButton.onclick = closeModal;
    backdrop.onclick = closeModal;

    // Close on Escape key
    const handleEscape = (e) => {
        if (e.key === 'Escape') {
            closeModal();
            document.removeEventListener('keydown', handleEscape);
        }
    };
    document.addEventListener('keydown', handleEscape);
}

// ============================================
// Templates Management
// ============================================

async function loadTemplates() {
    const container = document.getElementById('templates-container');
    const loading = document.getElementById('templates-loading');
    const list = document.getElementById('templates-list');
    
    try {
        loading.classList.remove('hidden');
        list.innerHTML = '';
        
        const response = await APIClient.get('/extension/templates');
        state.templates = response.templates || [];
        
        loading.classList.add('hidden');
        renderTemplates();
        
    } catch (error) {
        log('Failed to load templates:', error);
        loading.classList.add('hidden');
        NotificationManager.error('Load Failed', 'Failed to load extension templates');
    }
}

function renderTemplates() {
    const list = document.getElementById('templates-list');
    list.innerHTML = '';
    
    state.templates.forEach(template => {
        const templateElement = createTemplateElement(template);
        list.appendChild(templateElement);
    });

    renderLucideIcons(list);
}

function createTemplateElement(template) {
    const div = document.createElement('div');
    div.className = 'template-detail';
    
    div.innerHTML = `
        <h4>${template.display_name || template.name}</h4>
        <p>${template.description}</p>
        
        <div class="template-files">
            <h5>Generated Files:</h5>
            <div class="file-list">
                ${template.files.map(file => `<span class="file-tag">${file}</span>`).join('')}
            </div>
        </div>
    `;
    
    return div;
}

// ============================================
// Extension Testing
// ============================================

function initializeTesting() {
    const form = document.getElementById('testing-form');
    form.addEventListener('submit', handleExtensionTesting);

    // Browse extension button
    const browseBtn = document.getElementById('browse-extension-btn');
    if (browseBtn) {
        browseBtn.addEventListener('click', async () => {
            // Load latest builds if not already loaded
            if (state.builds.length === 0) {
                try {
                    const response = await APIClient.get('/extension/builds');
                    state.builds = response.builds || [];
                } catch (error) {
                    log('Failed to load builds for browser:', error);
                }
            }
            browseExtension();
        });
    }
}

async function handleExtensionTesting(event) {
    event.preventDefault();
    
    LoadingManager.setButtonLoading('test-btn', true);
    
    try {
        const formData = new FormData(event.target);
        const testSites = document.getElementById('test-sites').value
            .split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0);
        
        const requestData = {
            extension_path: formData.get('extension_path'),
            test_sites: testSites.length > 0 ? testSites : ['https://example.com'],
            screenshot: formData.get('screenshot') === 'on',
            headless: formData.get('headless') === 'on'
        };
        
        log('Testing extension with config:', requestData);
        
        // Make API request
        const result = await APIClient.post('/extension/test', requestData);
        
        // Show results
        displayTestResults(result);
        
        if (result.success) {
            NotificationManager.success(
                'Tests Passed',
                `All ${result.summary.passed} tests completed successfully!`
            );
        } else {
            NotificationManager.warning(
                'Some Tests Failed',
                `${result.summary.failed} out of ${result.summary.total_tests} tests failed.`
            );
        }
        
    } catch (error) {
        log('Extension testing failed:', error);
        NotificationManager.error(
            'Testing Failed',
            error.message || 'Unknown error occurred during extension testing'
        );
    } finally {
        LoadingManager.setButtonLoading('test-btn', false);
    }
}

function displayTestResults(results) {
    const resultsContainer = document.getElementById('test-results');
    const resultsContent = document.getElementById('test-results-content');
    
    const { summary, test_results } = results;
    
    resultsContent.innerHTML = `
        <div class="test-summary">
            <div class="test-stat">
                <div class="test-stat-value">${summary.total_tests}</div>
                <div class="test-stat-label">Total Tests</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value" style="color: var(--success-color)">${summary.passed}</div>
                <div class="test-stat-label">Passed</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value" style="color: var(--error-color)">${summary.failed}</div>
                <div class="test-stat-label">Failed</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value">${summary.success_rate.toFixed(1)}%</div>
                <div class="test-stat-label">Success Rate</div>
            </div>
        </div>
        
        <div class="test-details">
            <h4>Test Results by Site</h4>
            ${test_results.map(result => `
                <div class="test-site-result">
                    <div class="test-site-info">
                        <h5>${new URL(result.site).hostname}</h5>
                        <div class="test-site-url">${result.site}</div>
                        ${result.errors && result.errors.length > 0 ? `
                            <div class="test-errors">
                                ${result.errors.map(error => `<div class="error-text">${error}</div>`).join('')}
                            </div>
                        ` : ''}
                    </div>
                    <div class="test-result-status">
                        ${createIconMarkup(result.loaded ? 'check-circle-2' : 'circle-x', {
                            className: 'test-result-icon'
                        })}
                        <span>${result.loaded ? 'PASS' : 'FAIL'}</span>
                        ${result.load_time_ms ? `<span class="text-muted">(${result.load_time_ms}ms)</span>` : ''}
                    </div>
                </div>
            `).join('')}
        </div>
    `;
    
    resultsContainer.classList.remove('hidden');
    renderLucideIcons(resultsContent);
}

function browseExtension() {
    // Show a helpful modal with available builds to choose from
    const pathInput = document.getElementById('test-extension-path');

    if (state.builds.length === 0) {
        NotificationManager.info(
            'No Builds Available',
            'Generate an extension first to have build paths available. Or enter a path manually.'
        );
        return;
    }

    // Filter to only completed builds
    const completedBuilds = state.builds.filter(b => b.status === 'ready');

    if (completedBuilds.length === 0) {
        NotificationManager.warning(
            'No Completed Builds',
            'Wait for a build to complete, or enter a path manually.'
        );
        return;
    }

    // Create a quick selection dialog using native select element
    const selectHtml = `
        <div style="margin-bottom: 16px;">
            <label style="display: block; margin-bottom: 8px; font-weight: 500; color: var(--text-primary);">
                Select a recent build:
            </label>
            <select id="build-path-selector" style="width: 100%; padding: 10px; border: 1px solid var(--border-color); border-radius: 8px; font-size: 1rem;">
                ${completedBuilds.map(build => `
                    <option value="${escapeHtml(build.extension_path)}">
                        ${escapeHtml(build.scenario_name)} (${escapeHtml(build.config.app_name)}) - v${escapeHtml(build.config.version || '1.0.0')}
                    </option>
                `).join('')}
            </select>
        </div>
        <div style="display: flex; gap: 12px; justify-content: flex-end;">
            <button id="cancel-path-btn" class="btn btn-outline" style="padding: 10px 20px;">
                Cancel
            </button>
            <button id="select-path-btn" class="btn btn-primary" style="padding: 10px 20px;">
                ${createIconMarkup('check', { className: 'btn-icon' })}
                <span>Select Path</span>
            </button>
        </div>
    `;

    // Create temporary modal overlay
    const overlay = document.createElement('div');
    overlay.className = 'modal';
    overlay.innerHTML = `
        <div class="modal-backdrop"></div>
        <div class="modal-container" style="max-width: 600px;">
            <div class="modal-header">
                <h2>Select Extension Path</h2>
                <button class="modal-close" type="button" aria-label="Close">
                    ${createIconMarkup('x')}
                </button>
            </div>
            <div class="modal-content">
                ${selectHtml}
            </div>
        </div>
    `;

    document.body.appendChild(overlay);
    renderLucideIcons(overlay);

    // Show modal
    requestAnimationFrame(() => {
        overlay.classList.remove('hidden');
    });

    const selector = overlay.querySelector('#build-path-selector');
    const selectBtn = overlay.querySelector('#select-path-btn');
    const cancelBtn = overlay.querySelector('#cancel-path-btn');
    const closeBtn = overlay.querySelector('.modal-close');
    const backdrop = overlay.querySelector('.modal-backdrop');

    const closeOverlay = () => {
        overlay.classList.add('hidden');
        setTimeout(() => {
            document.body.removeChild(overlay);
        }, 300);
    };

    selectBtn.onclick = () => {
        pathInput.value = selector.value;
        NotificationManager.success('Path Selected', 'Extension path has been populated');
        closeOverlay();
    };

    cancelBtn.onclick = closeOverlay;
    closeBtn.onclick = closeOverlay;
    backdrop.onclick = closeOverlay;

    // Close on Escape
    const handleEscape = (e) => {
        if (e.key === 'Escape') {
            closeOverlay();
            document.removeEventListener('keydown', handleEscape);
        }
    };
    document.addEventListener('keydown', handleEscape);
}

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', () => {
    log('Extension Generator UI initializing...');

    applyApiContext();
    renderLucideIcons();

    // Initialize components
    initializeTabs();
    initializeGenerator();
    initializeTesting();
    initializeStatusPopover();

    // Additional UI event listeners
    const refreshBuildsBtn = document.getElementById('refresh-builds-btn');
    if (refreshBuildsBtn) {
        refreshBuildsBtn.addEventListener('click', loadBuilds);
    }

    const gotoGeneratorBtn = document.getElementById('goto-generator-btn');
    if (gotoGeneratorBtn) {
        gotoGeneratorBtn.addEventListener('click', () => switchTab('generator'));
    }

    // Event delegation for dynamically generated build action buttons
    const buildsContainer = document.getElementById('builds-list');
    if (buildsContainer) {
        buildsContainer.addEventListener('click', (event) => {
            const button = event.target.closest('[data-action]');
            if (!button) return;

            const action = button.dataset.action;
            const buildId = button.dataset.buildId;

            switch (action) {
                case 'test':
                    testBuild(buildId);
                    break;
                case 'download':
                    downloadBuild(buildId);
                    break;
                case 'view-details':
                    viewBuildDetails(buildId);
                    break;
            }
        });
    }

    // Initial status check
    refreshStatus();

    // Set up periodic status updates
    setInterval(refreshStatus, CONFIG.STATUS_REFRESH_INTERVAL);

    log('Extension Generator UI initialized');
});

// ============================================
// Global Functions (exposed for api-resolver.js integration)
// ============================================

// Consolidate all app functions into a single namespace to reduce global pollution
window.ScenarioToExtensionApp = {
    refreshStatus,
    switchTab,
    resetForm,
    loadBuilds,
    loadTemplates,
    testBuild,
    downloadBuild,
    viewBuildDetails,
    browseExtension,
    syncApiEndpointField,
    applyApiContext
};

// Bootstrap API resolution from api-resolver.js
if (typeof window.applyApiResolution === 'function') {
    window.applyApiResolution(apiResolution, { force: true, silent: true });
}
if (typeof window.startApiResolutionWatcher === 'function') {
    window.startApiResolutionWatcher();
}
