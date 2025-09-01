// Dark Chrome Security Dashboard - Interactive Script
// Secrets Manager UI Controller

class SecretsManager {
    constructor() {
        this.apiUrl = this.getApiUrl();
        this.refreshInterval = null;
        this.secrets = [];
        this.healthSummary = [];
        this.isConnected = false;
        
        this.init();
    }
    
    getApiUrl() {
        // Use same-origin URL for tunnel compatibility
        return `${window.location.protocol}//${window.location.host}`;
    }
    
    init() {
        this.updateClock();
        setInterval(() => this.updateClock(), 1000);
        
        this.bindEvents();
        this.checkApiConnection();
        this.startAutoRefresh();
        
        // Initialize with a scan
        setTimeout(() => this.performScan(), 1000);
    }
    
    bindEvents() {
        // Scan button
        document.getElementById('scan-all-btn').addEventListener('click', () => {
            this.performScan();
        });
        
        // Validate button
        document.getElementById('validate-btn').addEventListener('click', () => {
            this.performValidation();
        });
        
        // Resource filter
        document.getElementById('resource-filter').addEventListener('change', (e) => {
            this.filterSecrets(e.target.value);
        });
        
        // Modal controls
        document.getElementById('modal-close').addEventListener('click', () => {
            this.hideModal();
        });
        
        document.getElementById('cancel-provision').addEventListener('click', () => {
            this.hideModal();
        });
        
        document.getElementById('provision-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.provisionSecret();
        });
        
        // Close modal on overlay click
        document.getElementById('provision-modal').addEventListener('click', (e) => {
            if (e.target.id === 'provision-modal') {
                this.hideModal();
            }
        });
    }
    
    updateClock() {
        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', { 
            hour12: false,
            timeZone: 'UTC'
        }) + ' UTC';
        document.getElementById('current-time').textContent = timeString;
    }
    
    async checkApiConnection() {
        try {
            const response = await fetch(`${this.apiUrl}/health`);
            if (response.ok) {
                const data = await response.json();
                this.isConnected = true;
                this.updateConnectionStatus('SECURE CONNECTION ESTABLISHED', 'connected');
                this.updateApiStatus(data);
                return true;
            }
        } catch (error) {
            console.warn('API connection failed:', error);
        }
        
        this.isConnected = false;
        this.updateConnectionStatus('CONNECTION FAILED - RETRYING...', 'error');
        this.updateApiStatus(null);
        return false;
    }
    
    updateConnectionStatus(text, status) {
        const element = document.getElementById('connection-status');
        element.textContent = text;
        element.className = `status-text ${status}`;
    }
    
    updateApiStatus(healthData) {
        const statusElement = document.getElementById('api-status');
        const endpointElement = document.getElementById('api-endpoint');
        
        if (healthData) {
            statusElement.innerHTML = '<span class="pulse-dot"></span><span>OPERATIONAL</span>';
            statusElement.className = 'status-indicator connected';
            endpointElement.textContent = this.apiUrl;
        } else {
            statusElement.innerHTML = '<span class="pulse-dot error"></span><span>OFFLINE</span>';
            statusElement.className = 'status-indicator error';
            endpointElement.textContent = 'Connection Failed';
        }
    }
    
    startAutoRefresh() {
        // Refresh every 30 seconds
        this.refreshInterval = setInterval(async () => {
            if (await this.checkApiConnection()) {
                this.refreshSecretData();
            }
        }, 30000);
    }
    
    async performScan() {
        if (!this.isConnected) {
            this.showNotification('API connection required for scanning', 'error');
            return;
        }
        
        this.showNotification('Initiating security scan...', 'info');
        this.showScanningState(true);
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/secrets/scan`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ scan_type: 'full' })
            });
            
            if (!response.ok) {
                throw new Error(`Scan failed: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.secrets = data.discovered_secrets || [];
            this.updateSecretsTable();
            this.updateResourceFilter();
            
            // Update last scan time
            document.getElementById('last-scan').textContent = new Date().toLocaleTimeString();
            
            this.showNotification(`Scan completed: ${this.secrets.length} secrets discovered`, 'success');
            
            // Automatically trigger validation after scan
            setTimeout(() => this.performValidation(), 1000);
            
        } catch (error) {
            console.error('Scan error:', error);
            this.showNotification('Scan failed: ' + error.message, 'error');
        } finally {
            this.showScanningState(false);
        }
    }
    
    async performValidation() {
        if (!this.isConnected) {
            this.showNotification('API connection required for validation', 'error');
            return;
        }
        
        this.showNotification('Validating secret configuration...', 'info');
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/secrets/validate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({})
            });
            
            if (!response.ok) {
                throw new Error(`Validation failed: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.updateHealthStats(data);
            this.updateSecretValidationStatus(data);
            this.updateSecurityAlerts(data);
            
            const message = `Validation complete: ${data.valid_secrets}/${data.total_secrets} secrets valid`;
            this.showNotification(message, data.missing_secrets.length > 0 ? 'warning' : 'success');
            
        } catch (error) {
            console.error('Validation error:', error);
            this.showNotification('Validation failed: ' + error.message, 'error');
        }
    }
    
    async refreshSecretData() {
        // Silently refresh data without user notification
        try {
            const scanResponse = await fetch(`${this.apiUrl}/api/v1/secrets/scan`);
            if (scanResponse.ok) {
                const scanData = await scanResponse.json();
                this.secrets = scanData.discovered_secrets || [];
                this.updateSecretsTable();
            }
            
            const validateResponse = await fetch(`${this.apiUrl}/api/v1/secrets/validate`);
            if (validateResponse.ok) {
                const validateData = await validateResponse.json();
                this.updateHealthStats(validateData);
                this.updateSecretValidationStatus(validateData);
                this.updateSecurityAlerts(validateData);
            }
        } catch (error) {
            console.warn('Auto-refresh failed:', error);
        }
    }
    
    updateHealthStats(data) {
        document.getElementById('valid-secrets').textContent = data.valid_secrets || 0;
        document.getElementById('missing-secrets').textContent = (data.missing_secrets || []).length;
        document.getElementById('invalid-secrets').textContent = (data.invalid_secrets || []).length;
        document.getElementById('total-secrets').textContent = data.total_secrets || 0;
        
        // Update vault and database status based on validation results
        document.getElementById('vault-status').textContent = 
            data.valid_secrets > 0 ? 'Operational' : 'No Secrets Stored';
        document.getElementById('db-status').textContent = 'Connected';
    }
    
    updateSecretsTable() {
        const tbody = document.getElementById('secrets-table-body');
        tbody.innerHTML = '';
        
        if (this.secrets.length === 0) {
            tbody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="5">
                        <div class="loading-animation">
                            <span>NO SECRETS DISCOVERED - RUN SCAN</span>
                        </div>
                    </td>
                </tr>
            `;
            return;
        }
        
        this.secrets.forEach(secret => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${secret.resource_name}</td>
                <td><code>${secret.secret_key}</code></td>
                <td>${this.formatSecretType(secret.secret_type)}</td>
                <td><span class="status-badge discovered">DISCOVERED</span></td>
                <td>
                    <button class="chrome-btn provision-btn" data-secret="${secret.secret_key}">
                        PROVISION
                    </button>
                </td>
            `;
            
            // Add provision event listener
            row.querySelector('.provision-btn').addEventListener('click', () => {
                this.showProvisionModal(secret.secret_key);
            });
            
            tbody.appendChild(row);
        });
    }
    
    updateSecretValidationStatus(data) {
        // Update table with validation results
        const rows = document.querySelectorAll('#secrets-table-body tr');
        
        // Create a map of secret validation results
        const validationMap = new Map();
        
        if (data.missing_secrets) {
            data.missing_secrets.forEach(v => {
                const secret = this.secrets.find(s => s.id === v.resource_secret_id);
                if (secret) {
                    validationMap.set(secret.secret_key, { status: 'missing', ...v });
                }
            });
        }
        
        if (data.invalid_secrets) {
            data.invalid_secrets.forEach(v => {
                const secret = this.secrets.find(s => s.id === v.resource_secret_id);
                if (secret) {
                    validationMap.set(secret.secret_key, { status: 'invalid', ...v });
                }
            });
        }
        
        // Update table rows with validation status
        rows.forEach((row, index) => {
            const secret = this.secrets[index];
            if (secret && validationMap.has(secret.secret_key)) {
                const validation = validationMap.get(secret.secret_key);
                const statusCell = row.querySelector('td:nth-child(4)');
                statusCell.innerHTML = `<span class="status-badge ${validation.status}">${validation.status.toUpperCase()}</span>`;
            } else if (secret) {
                // Mark as valid if not in missing/invalid lists
                const statusCell = row.querySelector('td:nth-child(4)');
                statusCell.innerHTML = '<span class="status-badge valid">VALID</span>';
            }
        });
    }
    
    updateResourceFilter() {
        const select = document.getElementById('resource-filter');
        const resources = [...new Set(this.secrets.map(s => s.resource_name))];
        
        // Clear existing options except "ALL RESOURCES"
        select.innerHTML = '<option value="">ALL RESOURCES</option>';
        
        resources.forEach(resource => {
            const option = document.createElement('option');
            option.value = resource;
            option.textContent = resource.toUpperCase();
            select.appendChild(option);
        });
    }
    
    filterSecrets(resourceName) {
        const rows = document.querySelectorAll('#secrets-table-body tr');
        
        rows.forEach(row => {
            if (!resourceName) {
                row.style.display = '';
            } else {
                const resourceCell = row.querySelector('td:first-child');
                if (resourceCell && resourceCell.textContent === resourceName) {
                    row.style.display = '';
                } else {
                    row.style.display = 'none';
                }
            }
        });
    }
    
    updateSecurityAlerts(data) {
        const alertsContainer = document.getElementById('alerts-container');
        const alertCount = document.getElementById('alert-count');
        
        const alerts = [];
        
        if (data.missing_secrets && data.missing_secrets.length > 0) {
            alerts.push({
                level: 'error',
                message: `${data.missing_secrets.length} required secrets are missing`,
                details: data.missing_secrets.map(s => {
                    const secret = this.secrets.find(sec => sec.id === s.resource_secret_id);
                    return secret ? secret.secret_key : 'Unknown';
                }).join(', ')
            });
        }
        
        if (data.invalid_secrets && data.invalid_secrets.length > 0) {
            alerts.push({
                level: 'warning', 
                message: `${data.invalid_secrets.length} secrets have invalid values`,
                details: 'Check secret format and validation patterns'
            });
        }
        
        alertCount.textContent = alerts.length;
        
        if (alerts.length === 0) {
            alertsContainer.innerHTML = `
                <div class="no-alerts">
                    <span class="chrome-text">NO ACTIVE SECURITY THREATS</span>
                </div>
            `;
        } else {
            alertsContainer.innerHTML = alerts.map(alert => `
                <div class="alert ${alert.level}">
                    <div class="alert-message">${alert.message}</div>
                    <div class="alert-details">${alert.details}</div>
                </div>
            `).join('');
        }
    }
    
    showProvisionModal(secretKey) {
        document.getElementById('secret-key').value = secretKey;
        document.getElementById('secret-value').value = '';
        document.getElementById('storage-method').value = 'vault';
        document.getElementById('provision-modal').classList.add('show');
    }
    
    hideModal() {
        document.getElementById('provision-modal').classList.remove('show');
    }
    
    async provisionSecret() {
        const secretKey = document.getElementById('secret-key').value;
        const secretValue = document.getElementById('secret-value').value;
        const storageMethod = document.getElementById('storage-method').value;
        
        if (!secretValue.trim()) {
            this.showNotification('Secret value is required', 'error');
            return;
        }
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/secrets/provision`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    secret_key: secretKey,
                    secret_value: secretValue,
                    storage_method: storageMethod
                })
            });
            
            if (!response.ok) {
                throw new Error(`Provisioning failed: ${response.statusText}`);
            }
            
            const data = await response.json();
            
            if (data.success) {
                this.showNotification(`Secret provisioned successfully to ${data.storage_location}`, 'success');
                this.hideModal();
                // Refresh validation after provisioning
                setTimeout(() => this.performValidation(), 1000);
            } else {
                this.showNotification('Provisioning failed: ' + (data.validation_result.error_message || 'Unknown error'), 'error');
            }
            
        } catch (error) {
            console.error('Provisioning error:', error);
            this.showNotification('Provisioning failed: ' + error.message, 'error');
        }
    }
    
    showScanningState(scanning) {
        const scanBtn = document.getElementById('scan-all-btn');
        if (scanning) {
            scanBtn.disabled = true;
            scanBtn.innerHTML = '<span class="btn-glow"></span>SCANNING...';
        } else {
            scanBtn.disabled = false;
            scanBtn.innerHTML = '<span class="btn-glow"></span>FULL SCAN';
        }
    }
    
    formatSecretType(type) {
        const types = {
            'env_var': 'ENV VAR',
            'api_key': 'API KEY',
            'credential': 'CREDENTIAL',
            'token': 'TOKEN',
            'password': 'PASSWORD',
            'certificate': 'CERTIFICATE'
        };
        return types[type] || type.toUpperCase();
    }
    
    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <div class="notification-content">
                ${this.getNotificationIcon(type)} ${message}
            </div>
        `;
        
        document.getElementById('notifications').appendChild(notification);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            notification.style.animation = 'slide-out 0.3s ease forwards';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 5000);
    }
    
    getNotificationIcon(type) {
        const icons = {
            'success': '✅',
            'error': '❌', 
            'warning': '⚠️',
            'info': 'ℹ️'
        };
        return icons[type] || 'ℹ️';
    }
}

// CSS animation for slide-out
const slideOutKeyframes = `
@keyframes slide-out {
    to { transform: translateX(100%); opacity: 0; }
}
`;

// Add the keyframes to the document
if (!document.getElementById('slide-out-styles')) {
    const style = document.createElement('style');
    style.id = 'slide-out-styles';
    style.textContent = slideOutKeyframes;
    document.head.appendChild(style);
}

// Initialize the application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.secretsManager = new SecretsManager();
});

// Add some cyberpunk keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Ctrl+S for scan
    if (e.ctrlKey && e.key === 's') {
        e.preventDefault();
        window.secretsManager?.performScan();
    }
    
    // Ctrl+V for validate
    if (e.ctrlKey && e.key === 'v') {
        e.preventDefault();
        window.secretsManager?.performValidation();
    }
    
    // Escape to close modal
    if (e.key === 'Escape') {
        window.secretsManager?.hideModal();
    }
});

// Add console easter egg for the cyberpunk theme
console.log(`
    ██████╗ ██████╗  ██████╗ ██████╗ ██╗     ██╗
    ██╔══██╗██╔══██╗██╔═══██╗██╔═══██╗██║     ██║
    ██████╔╝██████╔╝██║   ██║██║   ██║██║     ██║
    ██╔═══╝ ██╔══██╗██║   ██║██║   ██║██║     ██║
    ██║     ██║  ██║╚██████╔╝╚██████╔╝███████╗██║
    ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚══════╝╚═╝
                                                  
    🔐 SECRETS MANAGER - Dark Chrome Terminal
    Shortcuts:
      Ctrl+S: Full Scan
      Ctrl+V: Validate All
      ESC: Close Modal
`);