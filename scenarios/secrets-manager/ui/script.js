// Dark Chrome Security Dashboard - Interactive Script
// Secrets Manager UI Controller

class SecretsManager {
    constructor() {
        this.apiUrl = this.getApiUrl();
        this.refreshInterval = null;
        this.vaultStatus = null;
        this.securityScanResult = null;
        this.complianceData = null;
        this.isConnected = false;
        this.currentView = 'vault'; // Track current view: 'vault', 'critical', 'high', 'medium', 'low', 'configured'
        this.activeStatCard = null; // Track which stat card is active
        
        this.init();
    }
    
    getApiUrl() {
        // Use same-origin URL for tunnel compatibility
        return `${window.location.protocol}//${window.location.host}`;
    }
    
    init() {
        this.updateClock();
        setInterval(() => this.updateClock(), 1000);
        
        // Show skeleton loader on initial load
        this.showTableSkeleton();
        
        this.bindEvents();
        this.checkApiConnection();
        this.startAutoRefresh();
        
        // Initialize with vault status check, compliance data, and security scan
        setTimeout(async () => {
            await this.loadVaultStatus();
            await this.loadComplianceData();
            await this.performSecurityScan();
        }, 1000);
    }
    
    bindEvents() {
        // Refresh button - now checks vault status and performs security scan
        const scanBtn = document.getElementById('scan-all-btn');
        if (scanBtn) {
            scanBtn.addEventListener('click', async () => {
                this.loadVaultStatus();
                this.loadComplianceData();
                // Also perform security scan to get scan metrics
                await this.performSecurityScan();
                // After refresh, restore the current view if it's not vault
                if (this.currentView !== 'vault') {
                    setTimeout(() => {
                        this.restoreCurrentView();
                    }, 1000);
                }
            });
        }
        
        // Vault info button - shows status details
        const vaultInfoBtn = document.getElementById('vault-info-btn');
        if (vaultInfoBtn) {
            vaultInfoBtn.addEventListener('click', () => {
                this.showVaultStatusInfo();
            });
        }
        
        // Open vault UI button
        const openVaultBtn = document.getElementById('open-vault-btn');
        if (openVaultBtn) {
            openVaultBtn.addEventListener('click', () => {
                // Vault typically runs on port 8200
                window.open('http://localhost:8200', '_blank');
            });
        }
        
        // Fix selected vulnerabilities button
        const fixSelectedBtn = document.getElementById('fix-selected-btn');
        if (fixSelectedBtn) {
            fixSelectedBtn.addEventListener('click', () => {
                this.fixSelectedVulnerabilities();
            });
        }
        
        // Select all checkbox
        const selectAllCheckbox = document.getElementById('select-all');
        if (selectAllCheckbox) {
            selectAllCheckbox.addEventListener('change', (e) => {
                this.selectAll(e.target.checked);
            });
        }
        
        // Component filter
        const componentFilter = document.getElementById('component-filter');
        if (componentFilter) {
            componentFilter.addEventListener('change', (e) => {
                this.filterComponents(e.target.value);
            });
        }
        
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
        
        // Make stat boxes clickable and add visual feedback
        document.querySelectorAll('.stat-card').forEach(card => {
            card.style.cursor = 'pointer';
            card.style.transition = 'all 0.3s ease';
            
            // Add hover effect
            card.addEventListener('mouseenter', () => {
                card.style.transform = 'translateY(-2px)';
                card.style.boxShadow = '0 4px 8px rgba(0,255,0,0.3)';
            });
            
            card.addEventListener('mouseleave', () => {
                if (!card.classList.contains('active')) {
                    card.style.transform = 'translateY(0)';
                    card.style.boxShadow = '';
                }
            });
            
            card.addEventListener('click', () => {
                // Remove active class from all cards
                document.querySelectorAll('.stat-card').forEach(c => {
                    c.classList.remove('active');
                    c.style.transform = 'translateY(0)';
                    c.style.boxShadow = '';
                    c.style.borderColor = '';
                });
                
                // Add active class to clicked card
                card.classList.add('active');
                card.style.transform = 'translateY(-2px)';
                card.style.boxShadow = '0 4px 8px rgba(0,255,0,0.3)';
                card.style.borderColor = '#00ff00';
                
                // Store active card reference
                this.activeStatCard = card;
                
                // Get the type from the card
                const label = card.querySelector('.stat-label')?.textContent || '';
                this.filterTableByType(label);
            });
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
        // Status indicator removed - just update endpoint
        const endpointElement = document.getElementById('api-endpoint');
        
        if (healthData) {
            endpointElement.textContent = this.apiUrl;
        } else {
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
    
    async loadVaultStatus() {
        if (!this.isConnected) {
            this.showNotification('API connection required', 'error');
            return;
        }
        
        this.showNotification('Checking vault secrets status...', 'info');
        this.showScanningState(true);
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/vault/secrets/status`);
            
            if (!response.ok) {
                throw new Error(`Failed to get vault status: ${response.statusText}`);
            }
            
            this.vaultStatus = await response.json();
            this.updateVaultSecretsTable();
            this.updateResourceFilter();
            
            // Update last scan time
            document.getElementById('last-scan').textContent = new Date().toLocaleTimeString();
            
            const missingCount = this.vaultStatus.missing_secrets ? this.vaultStatus.missing_secrets.length : 0;
            const message = `Vault check complete: ${this.vaultStatus.configured_resources}/${this.vaultStatus.total_resources} resources configured`;
            
            this.showNotification(message, missingCount > 0 ? 'warning' : 'success');
            
        } catch (error) {
            console.error('Vault status error:', error);
            this.showNotification('Failed to check vault status: ' + error.message, 'error');
        } finally {
            this.showScanningState(false);
        }
    }
    
    async performSecurityScan() {
        if (!this.isConnected) {
            this.showNotification('API connection required', 'error');
            return;
        }
        
        this.showNotification('Scanning for security vulnerabilities...', 'info');
        this.showTableSkeleton();
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/security/scan`);
            
            if (!response.ok) {
                throw new Error(`Security scan failed: ${response.statusText}`);
            }
            
            this.securityScanResult = await response.json();
            this.updateSecurityVulnerabilities();
            this.updateScanMetrics();
            
            const vulnCount = this.securityScanResult.vulnerabilities ? this.securityScanResult.vulnerabilities.length : 0;
            const riskScore = this.securityScanResult.risk_score || 0;
            const scanMetrics = this.securityScanResult.scan_metrics || {};
            
            // Update last scan time
            document.getElementById('last-scan').textContent = new Date().toLocaleTimeString('en-US', { hour12: false }) + ' UTC';
            
            let message = `Security scan complete: ${vulnCount} vulnerabilities found (Risk: ${riskScore}/100)`;
            
            // Add scan performance info
            if (scanMetrics.total_scan_time_ms) {
                message += ` - Scanned ${scanMetrics.files_scanned || 0} files in ${scanMetrics.total_scan_time_ms}ms`;
            }
            
            // Show scan errors if any
            if (scanMetrics.scan_errors && scanMetrics.scan_errors.length > 0) {
                this.showNotification(`Scan completed with ${scanMetrics.scan_errors.length} errors`, 'warning');
                // Show first few errors as additional notifications
                scanMetrics.scan_errors.slice(0, 3).forEach(error => {
                    setTimeout(() => this.showNotification(`Scan error: ${error}`, 'error'), 500);
                });
            }
            
            // Show timeout warning if occurred
            if (scanMetrics.timeout_occurred) {
                this.showNotification('Scan timeout occurred - results may be incomplete', 'warning');
            }
            
            this.showNotification(message, riskScore > 50 ? 'error' : (riskScore > 20 ? 'warning' : 'success'));
            
        } catch (error) {
            console.error('Security scan error:', error);
            this.showNotification('Security scan failed: ' + error.message, 'error');
        }
    }
    
    async loadComplianceData() {
        if (!this.isConnected) {
            return;
        }
        
        try {
            const response = await fetch(`${this.apiUrl}/api/v1/security/compliance`);
            
            if (!response.ok) {
                throw new Error(`Failed to get compliance data: ${response.statusText}`);
            }
            
            this.complianceData = await response.json();
            this.updateHealthStats(this.complianceData);
            this.updateSecurityAlerts(this.complianceData);
            
        } catch (error) {
            console.error('Compliance data error:', error);
        }
    }
    
    async refreshSecretData() {
        // Silently refresh data without user notification
        try {
            // Refresh vault status
            const vaultResponse = await fetch(`${this.apiUrl}/api/v1/vault/secrets/status`);
            if (vaultResponse.ok) {
                this.vaultStatus = await vaultResponse.json();
                // Only update table if we're in vault view, otherwise preserve current view
                if (this.currentView === 'vault') {
                    this.updateVaultSecretsTable();
                }
            }
            
            // Refresh compliance data
            const complianceResponse = await fetch(`${this.apiUrl}/api/v1/security/compliance`);
            if (complianceResponse.ok) {
                this.complianceData = await complianceResponse.json();
                this.updateHealthStats(this.complianceData);
                this.updateSecurityAlerts(this.complianceData);
                
                // Refresh current vulnerability view if showing vulnerabilities
                if (this.currentView !== 'vault' && this.currentView !== 'configured' && this.currentView !== 'total') {
                    this.showVulnerabilitiesBySeverity(this.currentView);
                }
            }
        } catch (error) {
            console.warn('Auto-refresh failed:', error);
        }
    }
    
    updateHealthStats(data) {
        // Update stats from compliance data - using new component-based approach
        if (data.configured_components !== undefined) {
            document.getElementById('configured-components').textContent = data.configured_components || 0;
        }
        if (data.vulnerability_summary) {
            // Store vulnerability counts for later use
            this.vulnerabilityCounts = {
                critical: data.vulnerability_summary.critical || 0,
                high: data.vulnerability_summary.high || 0,
                medium: data.vulnerability_summary.medium || 0,
                low: data.vulnerability_summary.low || 0
            };
            
            // Update individual vulnerability stat boxes
            document.getElementById('critical-vulns').textContent = this.vulnerabilityCounts.critical;
            document.getElementById('high-vulns').textContent = this.vulnerabilityCounts.high;
            document.getElementById('medium-vulns').textContent = this.vulnerabilityCounts.medium;
            document.getElementById('low-vulns').textContent = this.vulnerabilityCounts.low;
        }
        
        // Update vault and database status with proper coloring
        const vaultStatusEl = document.getElementById('vault-status');
        const vaultHealth = data.vault_secrets_health;
        
        // Remove all status classes
        vaultStatusEl.classList.remove('critical', 'warning');
        
        if (vaultHealth > 75) {
            vaultStatusEl.textContent = 'Operational';
            // Default green color, no extra class needed
        } else if (vaultHealth > 50) {
            vaultStatusEl.textContent = 'Degraded';
            vaultStatusEl.classList.add('warning');
        } else {
            vaultStatusEl.textContent = 'Critical';
            vaultStatusEl.classList.add('critical');
        }
        
        // Store health info for info button
        this.vaultHealthInfo = {
            status: vaultStatusEl.textContent,
            health: vaultHealth,
            totalSecrets: data.total_secrets || 0,
            configuredSecrets: data.configured_secrets || 0,
            missingSecrets: data.missing_secrets || 0
        };
        
        document.getElementById('db-status').textContent = 'Connected';
    }
    
    updateVaultSecretsTable() {
        const tbody = document.getElementById('secrets-table-body');
        if (!tbody) {
            console.error('Table body element not found');
            this.showNotification('UI elements not found - page may not be loaded correctly', 'error');
            return;
        }
        
        tbody.innerHTML = '';
        
        if (!this.vaultStatus || !this.vaultStatus.resource_statuses) {
            tbody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="6">
                        <div class="loading-animation">
                            <span>NO VAULT DATA - CLICK SCAN TO CHECK VAULT STATUS</span>
                        </div>
                    </td>
                </tr>
            `;
            return;
        }
        
        // Show vault status for each resource
        this.vaultStatus.resource_statuses.forEach(resource => {
            const statusBadge = resource.health_status === 'healthy' ? 'valid' :
                               resource.health_status === 'degraded' ? 'discovered' : 'missing';
            
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <input type="checkbox" class="row-checkbox">
                </td>
                <td><span class="component-badge resource">RES</span> ${resource.resource_name}</td>
                <td><code>${resource.secrets_found}/${resource.secrets_total} secrets</code></td>
                <td>VAULT STATUS</td>
                <td><span class="status-badge ${statusBadge}">${resource.health_status.toUpperCase()}</span></td>
                <td>
                    ${resource.secrets_missing > 0 ? 
                        `<button class="chrome-btn provision-btn" data-resource="${resource.resource_name}">
                            PROVISION
                        </button>` : 
                        `<span class="chrome-text">✓ CONFIGURED</span>`
                    }
                </td>
            `;
            
            // Add provision event listener if button exists
            const provisionBtn = row.querySelector('.provision-btn');
            if (provisionBtn) {
                provisionBtn.addEventListener('click', () => {
                    this.showProvisionModal(resource.resource_name, true);
                });
            }
            
            // Add checkbox event listener
            const checkbox = row.querySelector('.row-checkbox');
            if (checkbox) {
                checkbox.addEventListener('change', () => {
                    if (checkbox.checked) {
                        row.classList.add('selected');
                    } else {
                        row.classList.remove('selected');
                    }
                    this.updateFixButtonState();
                });
            }
            
            tbody.appendChild(row);
        });
        
        // Add missing secrets details if any
        if (this.vaultStatus.missing_secrets && this.vaultStatus.missing_secrets.length > 0) {
            this.vaultStatus.missing_secrets.forEach(missing => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>
                        <input type="checkbox" class="row-checkbox">
                    </td>
                    <td><span class="component-badge resource">RES</span> ${missing.resource_name}</td>
                    <td><code>${missing.secret_name}</code></td>
                    <td>MISSING SECRET</td>
                    <td><span class="status-badge missing">${missing.required ? 'REQUIRED' : 'OPTIONAL'}</span></td>
                    <td>
                        <button class="chrome-btn provision-btn" data-resource="${missing.resource_name}" data-secret="${missing.secret_name}">
                            SET
                        </button>
                    </td>
                `;
                
                row.querySelector('.provision-btn').addEventListener('click', () => {
                    this.showProvisionModal(missing.resource_name, true);
                });
                
                // Add checkbox event listener
                const checkbox = row.querySelector('.row-checkbox');
                if (checkbox) {
                    checkbox.addEventListener('change', () => {
                        if (checkbox.checked) {
                            row.classList.add('selected');
                        } else {
                            row.classList.remove('selected');
                        }
                        this.updateFixButtonState();
                    });
                }
                
                tbody.appendChild(row);
            });
        }
    }
    
    updateSecurityVulnerabilities() {
        // Store vulnerabilities for later display
        if (this.securityScanResult && this.securityScanResult.vulnerabilities) {
            this.vulnerabilities = this.securityScanResult.vulnerabilities;
            console.log('Security vulnerabilities found:', this.vulnerabilities.length);
        }
    }
    
    updateScanMetrics() {
        if (!this.securityScanResult || !this.securityScanResult.scan_metrics) {
            return;
        }
        
        const metrics = this.securityScanResult.scan_metrics;
        
        // Update scan time
        const scanTime = metrics.total_scan_time_ms || 0;
        document.getElementById('scan-time').textContent = `${scanTime}ms`;
        if (scanTime > 30000) { // Highlight slow scans (>30s)
            document.getElementById('scan-time').style.color = '#ffaa00';
        } else {
            document.getElementById('scan-time').style.color = '';
        }
        
        // Update files scanned
        const filesScanned = metrics.files_scanned || 0;
        document.getElementById('files-scanned').textContent = filesScanned.toString();
        
        // Update scan errors with color coding
        const errorCount = metrics.scan_errors ? metrics.scan_errors.length : 0;
        const errorElement = document.getElementById('scan-errors');
        const errorInfoBtn = document.getElementById('scan-errors-info-btn');
        errorElement.textContent = errorCount.toString();
        
        let errorTooltip;
        if (errorCount > 0) {
            errorElement.style.color = '#ff4444';
            errorTooltip = `${errorCount} scan errors occurred:\n${metrics.scan_errors.slice(0, 3).join('\n')}${errorCount > 3 ? '\n... and more' : ''}`;
        } else {
            errorElement.style.color = '#00ff00';
            errorTooltip = 'No scan errors';
        }
        
        // Set tooltip on both the value and info button
        errorElement.title = errorTooltip;
        if (errorInfoBtn) {
            errorInfoBtn.title = errorTooltip;
        }
        
        // Update other metrics if available
        if (metrics.timeout_occurred) {
            document.getElementById('scan-time').title = 'Scan timed out - results may be incomplete';
            document.getElementById('scan-time').style.color = '#ff4444';
        }
        
        console.log('Scan metrics updated:', {
            scanTime: scanTime + 'ms',
            filesScanned,
            errorCount,
            timeoutOccurred: metrics.timeout_occurred,
            largeFilesSkipped: metrics.large_files_skipped || 0
        });
    }
    
    filterTableByType(label) {
        // Show skeleton immediately for smooth transition
        this.showTableSkeleton();
        
        // Update the section title based on what's being shown
        const sectionTitle = document.querySelector('.secrets-panel .panel-header h2');
        
        // Small delay to show the skeleton animation
        setTimeout(() => {
            if (label.includes('Configured')) {
                // Show configured resources
                this.currentView = 'configured';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> CONFIGURED COMPONENTS';
                this.showConfiguredComponents();
            } else if (label.includes('Critical')) {
                // Show critical vulnerabilities
                this.currentView = 'critical';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> CRITICAL VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('critical');
            } else if (label.includes('High')) {
                // Show high vulnerabilities
                this.currentView = 'high';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> HIGH SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('high');
            } else if (label.includes('Medium')) {
                // Show medium vulnerabilities
                this.currentView = 'medium';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> MEDIUM SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('medium');
            } else if (label.includes('Low')) {
                // Show low vulnerabilities
                this.currentView = 'low';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> LOW SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('low');
            } else {
                // Default to showing vault status
                this.currentView = 'vault';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> RESOURCE VAULT STATUS';
                this.updateVaultSecretsTable();
            }
        }, 300); // 300ms delay to show skeleton animation
    }
    
    filterComponents(componentType) {
        // Filter table by component type (resource, scenario, or all)
        this.showTableSkeleton();
        
        const sectionTitle = document.querySelector('.secrets-panel .panel-header h2');
        
        setTimeout(() => {
            if (componentType === 'resource') {
                this.currentView = 'resources';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> RESOURCE COMPONENTS';
                this.showComponentsByType('resource');
            } else if (componentType === 'scenario') {
                this.currentView = 'scenarios';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> SCENARIO COMPONENTS';
                this.showComponentsByType('scenario');
            } else {
                this.currentView = 'vault';
                if (sectionTitle) sectionTitle.innerHTML = '<span class="matrix-green">●</span> COMPONENT SECURITY STATUS';
                this.updateVaultSecretsTable();
            }
        }, 300);
    }
    
    restoreCurrentView() {
        // Restore the current view and active stat card after refresh
        if (this.activeStatCard) {
            // Re-trigger the click on the active stat card
            this.activeStatCard.click();
        }
    }
    
    showConfiguredComponents() {
        const tbody = document.getElementById('secrets-table-body');
        
        if (!this.vaultStatus || !this.vaultStatus.resource_statuses) {
            tbody.innerHTML = `
                <tr><td colspan="6" class="chrome-text">No vault data available. Click "CHECK VAULT" to scan.</td></tr>
            `;
            return;
        }
        
        // Filter for healthy/configured resources only
        const configuredResources = this.vaultStatus.resource_statuses.filter(r => r.health_status === 'healthy');
        
        if (configuredResources.length === 0) {
            tbody.innerHTML = `
                <tr><td colspan="6" class="chrome-text">No fully configured resources found.</td></tr>
            `;
            return;
        }
        
        configuredResources.forEach(resource => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>
                    <input type="checkbox" class="row-checkbox">
                </td>
                <td><span class="component-badge resource">RES</span> ${resource.resource_name}</td>
                <td><code>${resource.secrets_found}/${resource.secrets_total} secrets</code></td>
                <td>VAULT STATUS</td>
                <td><span class="status-badge valid">CONFIGURED</span></td>
                <td><span class="chrome-text">✓ ALL SECRETS SET</span></td>
            `;
            // Add checkbox event listener
            const checkbox = row.querySelector('.row-checkbox');
            if (checkbox) {
                checkbox.addEventListener('change', () => {
                    if (checkbox.checked) {
                        row.classList.add('selected');
                    } else {
                        row.classList.remove('selected');
                    }
                    this.updateFixButtonState();
                });
            }

            tbody.appendChild(row);
        });
    }
    
    async showComponentsByType(componentType) {
        const tbody = document.getElementById('secrets-table-body');
        
        try {
            // Fetch vulnerabilities filtered by component type
            const response = await fetch(`/api/v1/vulnerabilities?component_type=${componentType}`);
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.message || 'Failed to fetch components');
            }
            
            const vulnerabilities = data.vulnerabilities || [];
            
            if (vulnerabilities.length === 0) {
                tbody.innerHTML = `
                    <tr><td colspan="6" class="chrome-text">No ${componentType} vulnerabilities found.</td></tr>
                `;
                return;
            }
            
            // Clear table and show each component with vulnerabilities
            tbody.innerHTML = '';
            vulnerabilities.forEach(vuln => {
                const row = document.createElement('tr');
                const componentBadge = vuln.component_type === 'resource' ? 
                    '<span class="component-badge resource">RES</span>' : 
                    '<span class="component-badge scenario">APP</span>';
                    
                const severityBadge = vuln.severity === 'critical' ? 'missing' :
                                      vuln.severity === 'high' ? 'invalid' :
                                      vuln.severity === 'medium' ? 'discovered' : 'valid';
                
                row.innerHTML = `
                    <td>
                        <input type="checkbox" class="row-checkbox">
                    </td>
                    <td>${componentBadge} ${vuln.component_name}</td>
                    <td style="font-size: 0.9em;">
                        <strong>${vuln.title}</strong><br>
                        <code style="font-size: 0.85em;">${vuln.file_path}:${vuln.line_number}</code>
                    </td>
                    <td>${vuln.type.replace(/_/g, ' ').toUpperCase()}</td>
                    <td><span class="status-badge ${severityBadge}">${vuln.severity.toUpperCase()}</span></td>
                    <td>
                        <button class="chrome-btn" style="padding: 2px 8px;" onclick="window.secretsManager.showVulnerabilityDetails('${vuln.id}')">
                            DETAILS
                        </button>
                    </td>
                `;
                
                // Add checkbox event listener
                const checkbox = row.querySelector('.row-checkbox');
                if (checkbox) {
                    checkbox.addEventListener('change', () => {
                        if (checkbox.checked) {
                            row.classList.add('selected');
                        } else {
                            row.classList.remove('selected');
                        }
                        this.updateFixButtonState();
                    });
                }
                
                // Add click handler for vulnerability code viewing
                this.addVulnerabilityRowClickHandler(row, vuln);
                
                tbody.appendChild(row);
            });
            
        } catch (error) {
            console.error('Error fetching components:', error);
            tbody.innerHTML = `
                <tr><td colspan="6" class="chrome-text">Error loading ${componentType} components: ${error.message}</td></tr>
            `;
        }
    }
    
    async showVulnerabilitiesBySeverity(severity) {
        const tbody = document.getElementById('secrets-table-body');
        
        try {
            // Fetch vulnerabilities from API
            const response = await fetch(`/api/v1/vulnerabilities?severity=${severity}`);
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.message || 'Failed to fetch vulnerabilities');
            }
            
            const vulnerabilities = data.vulnerabilities || [];
            
            if (vulnerabilities.length === 0) {
                tbody.innerHTML = `
                    <tr><td colspan="6" class="chrome-text">No ${severity} vulnerabilities found.</td></tr>
                `;
                return;
            }
            
            // Clear table and show each vulnerability
            tbody.innerHTML = '';
            vulnerabilities.forEach(vuln => {
                const row = document.createElement('tr');
                const componentBadge = vuln.component_type === 'resource' ? 
                    '<span class="component-badge resource">RES</span>' : 
                    '<span class="component-badge scenario">APP</span>';
                const severityBadge = vuln.severity === 'critical' ? 'missing' :
                                      vuln.severity === 'high' ? 'invalid' :
                                      vuln.severity === 'medium' ? 'discovered' : 'valid';
                
                row.innerHTML = `
                    <td>
                        <input type="checkbox" class="row-checkbox">
                    </td>
                    <td>${componentBadge} ${vuln.component_name || vuln.scenario_name || 'Unknown'}</td>
                    <td style="font-size: 0.9em;">
                        <strong>${vuln.title}</strong><br>
                        <code style="font-size: 0.85em;">${vuln.file_path}:${vuln.line_number}</code>
                    </td>
                    <td>${vuln.type.replace(/_/g, ' ').toUpperCase()}</td>
                    <td><span class="status-badge ${severityBadge}">${vuln.severity.toUpperCase()}</span></td>
                    <td>
                        <button class="chrome-btn" style="padding: 2px 8px;" onclick="window.secretsManager.showVulnerabilityDetails('${vuln.id}')">
                            DETAILS
                        </button>
                    </td>
                `;
                // Add checkbox event listener
                const checkbox = row.querySelector('.row-checkbox');
                if (checkbox) {
                    checkbox.addEventListener('change', () => {
                        if (checkbox.checked) {
                            row.classList.add('selected');
                        } else {
                            row.classList.remove('selected');
                        }
                        this.updateFixButtonState();
                    });
                }

                // Add click handler for vulnerability code viewing
                this.addVulnerabilityRowClickHandler(row, vuln);
                
                tbody.appendChild(row);
            });
            
        } catch (error) {
            console.error('Error fetching vulnerabilities:', error);
            
            // Fallback to showing summary from compliance data
            const count = this.vulnerabilityCounts ? this.vulnerabilityCounts[severity] : 0;
            
            if (count === 0) {
                tbody.innerHTML = `
                    <tr><td colspan="6" class="chrome-text">No ${severity} vulnerabilities detected.</td></tr>
                `;
                return;
            }
            
            // Show summary message with call to action
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" class="chrome-text" style="text-align: center;">
                        <div style="padding: 20px;">
                            <div style="font-size: 1.2em; color: ${severity === 'critical' ? '#ff0000' : severity === 'high' ? '#ff8800' : severity === 'medium' ? '#ffaa00' : '#ffff00'};">
                                ${count} ${severity.toUpperCase()} VULNERABILITIES DETECTED
                            </div>
                            <div style="margin-top: 10px; opacity: 0.8;">
                                Security compliance scan identified ${count} ${severity} severity issues across scenarios.
                                <br><em>Error loading details: ${error.message}</em>
                            </div>
                            <div style="margin-top: 15px;">
                                <button class="chrome-btn" onclick="window.secretsManager.performSecurityScan()" style="background: rgba(255,0,0,0.2); border-color: #ff0000;">
                                    RUN DETAILED SCAN
                                </button>
                                <button class="chrome-btn" onclick="window.secretsManager.resetToVaultView()" style="margin-left: 10px;">
                                    BACK TO VAULT
                                </button>
                            </div>
                        </div>
                    </td>
                </tr>
            `;
        }
    }
    
    showVulnerabilityDetails(vulnId) {
        const vuln = this.vulnerabilities.find(v => v.id === vulnId);
        if (!vuln) return;
        
        // Show vulnerability details in modal-like format
        const modal = document.createElement('div');
        modal.className = 'vulnerability-modal';
        modal.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: linear-gradient(135deg, rgba(0,0,0,0.95), rgba(20,20,20,0.95));
            border: 2px solid rgba(0,255,0,0.3);
            border-radius: 8px;
            padding: 20px;
            max-width: 600px;
            max-height: 80vh;
            overflow-y: auto;
            z-index: 10000;
            color: #00ff00;
            box-shadow: 0 0 30px rgba(0,255,0,0.3);
        `;
        
        modal.innerHTML = `
            <div style="margin-bottom: 15px; border-bottom: 1px solid rgba(0,255,0,0.2); padding-bottom: 10px;">
                <h3 style="margin: 0; color: #00ff00;">${vuln.title}</h3>
                <small style="color: #888;">${vuln.file_path}:${vuln.line_number}</small>
            </div>
            
            <div style="margin-bottom: 15px;">
                <strong>Severity:</strong> <span style="color: ${vuln.severity === 'critical' ? '#ff0000' : vuln.severity === 'high' ? '#ff8800' : '#ffff00'}">
                    ${vuln.severity.toUpperCase()}
                </span>
            </div>
            
            <div style="margin-bottom: 15px;">
                <strong>Type:</strong> ${vuln.type}
            </div>
            
            <div style="margin-bottom: 15px;">
                <strong>Description:</strong><br>
                ${vuln.description}
            </div>
            
            ${vuln.code ? `
            <div style="margin-bottom: 15px;">
                <strong>Code:</strong><br>
                <pre style="background: rgba(0,0,0,0.5); padding: 10px; border-radius: 4px; overflow-x: auto;">
${vuln.code}</pre>
            </div>
            ` : ''}
            
            <div style="margin-bottom: 15px;">
                <strong>Recommendation:</strong><br>
                ${vuln.recommendation}
            </div>
            
            <div style="text-align: right;">
                <button class="chrome-btn" onclick="this.parentElement.parentElement.remove()">
                    CLOSE
                </button>
                ${vuln.can_auto_fix ? `
                <button class="chrome-btn" style="margin-left: 10px; background: rgba(0,255,0,0.2);" 
                        onclick="window.secretsManager.autoFixVulnerability('${vuln.id}')">
                    AUTO-FIX
                </button>
                ` : ''}
            </div>
        `;
        
        document.body.appendChild(modal);
    }
    
    autoFixVulnerability(vulnId) {
        // Placeholder for auto-fix functionality
        this.showNotification('Auto-fix feature coming soon', 'info');
        document.querySelector('.vulnerability-modal')?.remove();
    }
    
    resetToVaultView() {
        // Reset section title
        const sectionTitle = document.querySelector('.secrets-panel .panel-header h2');
        if (sectionTitle) {
            sectionTitle.innerHTML = '<span class="matrix-green">●</span> RESOURCE VAULT STATUS';
        }
        
        // Remove active class from all stat cards
        document.querySelectorAll('.stat-card').forEach(card => {
            card.classList.remove('active');
            card.style.transform = 'translateY(0)';
            card.style.boxShadow = '';
            card.style.borderColor = '';
        });
        
        // Show vault secrets table
        this.updateVaultSecretsTable();
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
        const select = document.getElementById('component-filter');
        if (!select) {
            console.error('Component filter element not found');
            return;
        }
        
        // Get resources from vault status
        const resources = this.vaultStatus && this.vaultStatus.resource_statuses ? 
            this.vaultStatus.resource_statuses.map(r => r.resource_name) : [];
        
        // Clear existing options except "ALL COMPONENTS"
        select.innerHTML = '<option value="">ALL COMPONENTS</option><option value="resource">RESOURCES ONLY</option><option value="scenario">SCENARIOS ONLY</option>';
        
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
        // Alerts section removed - vulnerabilities shown in stat boxes
        return;
        
        /* Legacy code preserved for reference
        const alertsContainer = document.getElementById('alerts-container');
        const alertCount = document.getElementById('alert-count');
        
        const alerts = [];
        
        // Add alerts based on compliance data
        if (data.vulnerability_summary) {
            if (data.vulnerability_summary.critical > 0) {
                alerts.push({
                    level: 'error',
                    message: `${data.vulnerability_summary.critical} CRITICAL security vulnerabilities detected`,
                    details: 'Run security scan for details'
                });
            }
            
            if (data.vulnerability_summary.high > 0) {
                alerts.push({
                    level: 'warning',
                    message: `${data.vulnerability_summary.high} HIGH severity vulnerabilities found`,
                    details: 'Review and remediate high-priority issues'
                });
            }
        }
        
        // Add vault secrets alerts
        if (this.vaultStatus && this.vaultStatus.missing_secrets && this.vaultStatus.missing_secrets.length > 0) {
            const requiredMissing = this.vaultStatus.missing_secrets.filter(s => s.required).length;
            if (requiredMissing > 0) {
                alerts.push({
                    level: 'error',
                    message: `${requiredMissing} required vault secrets are missing`,
                    details: this.vaultStatus.missing_secrets
                        .filter(s => s.required)
                        .map(s => `${s.resource_name}:${s.secret_name}`)
                        .slice(0, 3)
                        .join(', ') + (requiredMissing > 3 ? '...' : '')
                });
            }
        }
        
        // Add overall compliance alert
        if (data.overall_score !== undefined && data.overall_score < 50) {
            alerts.push({
                level: 'warning',
                message: `Overall security compliance: ${data.overall_score}%`,
                details: 'Critical security improvements needed'
            });
        }
        
        alertCount.textContent = alerts.length;
        
        if (alerts.length === 0) {
            alertsContainer.innerHTML = `
                <div class="no-alerts">
                    <span class="chrome-text">✓ SECURITY POSTURE: HEALTHY</span>
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
        */
    }
    
    showProvisionModal(resourceOrKey, isResource = false) {
        if (isResource) {
            // For resource-based provisioning, show resource name
            document.getElementById('secret-key').value = `Resource: ${resourceOrKey}`;
            document.getElementById('secret-key').dataset.resource = resourceOrKey;
        } else {
            document.getElementById('secret-key').value = resourceOrKey;
        }
        document.getElementById('secret-value').value = '';
        document.getElementById('storage-method').value = 'vault';
        document.getElementById('provision-modal').classList.add('show');
    }
    
    hideModal() {
        document.getElementById('provision-modal').classList.remove('show');
    }
    
    async provisionSecret() {
        const secretKeyElement = document.getElementById('secret-key');
        const secretValue = document.getElementById('secret-value').value;
        const storageMethod = document.getElementById('storage-method').value;
        
        if (!secretValue.trim()) {
            this.showNotification('Secret value is required', 'error');
            return;
        }
        
        // Check if this is a resource-based provision
        const resourceName = secretKeyElement.dataset.resource;
        
        try {
            let response;
            if (resourceName) {
                // Use vault provision endpoint for resources
                response = await fetch(`${this.apiUrl}/api/v1/vault/secrets/provision`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        resource_name: resourceName,
                        secrets: {
                            [secretKeyElement.value.replace(`Resource: ${resourceName}`, '').trim() || 'default']: secretValue
                        }
                    })
                });
            } else {
                // Legacy provision for individual secrets
                response = await fetch(`${this.apiUrl}/api/v1/secrets/provision`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        secret_key: secretKeyElement.value,
                        secret_value: secretValue,
                        storage_method: storageMethod
                    })
                });
            }
            
            if (!response.ok) {
                throw new Error(`Provisioning failed: ${response.statusText}`);
            }
            
            const data = await response.json();
            
            if (data.success) {
                this.showNotification(`Secret provisioned successfully`, 'success');
                this.hideModal();
                // Clear resource data
                delete secretKeyElement.dataset.resource;
                // Refresh vault status after provisioning
                setTimeout(() => this.loadVaultStatus(), 1000);
            } else {
                this.showNotification('Provisioning failed: ' + (data.message || 'Unknown error'), 'error');
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
            scanBtn.innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="rotating">
                    <polyline points="23 4 23 10 17 10"></polyline>
                    <polyline points="1 20 1 14 7 14"></polyline>
                    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
                </svg>`;
            this.showTableSkeleton();
        } else {
            scanBtn.disabled = false;
            scanBtn.innerHTML = `
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="23 4 23 10 17 10"></polyline>
                    <polyline points="1 20 1 14 7 14"></polyline>
                    <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
                </svg>`;
        }
    }
    
    showVaultStatusInfo() {
        if (!this.vaultHealthInfo) {
            this.showNotification('Loading vault status...', 'info');
            return;
        }
        
        const info = this.vaultHealthInfo;
        const message = `Vault Status: ${info.status}\n` +
                       `Health Score: ${info.health}%\n` +
                       `Total Secrets: ${info.totalSecrets}\n` +
                       `Configured: ${info.configuredSecrets}\n` +
                       `Missing: ${info.missingSecrets}\n\n` +
                       `Status Thresholds:\n` +
                       `• Operational: >75% health\n` +
                       `• Degraded: 50-75% health\n` +
                       `• Critical: <50% health`;
        
        alert(message);
    }
    
    showTableSkeleton() {
        const tbody = document.getElementById('secrets-table-body');
        if (!tbody) {
            console.error('Table body element not found for skeleton');
            return;
        }
        
        tbody.innerHTML = '';
        
        // Create skeleton rows
        for (let i = 0; i < 5; i++) {
            const row = document.createElement('tr');
            row.className = 'skeleton-row';
            row.innerHTML = `
                <td>
                    <div class="skeleton-line" style="width: 16px; height: 16px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
                <td>
                    <div class="skeleton-line" style="width: 80%; height: 14px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
                <td>
                    <div class="skeleton-line" style="width: 60%; height: 14px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                    <div class="skeleton-line" style="width: 90%; height: 12px; margin-top: 4px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
                <td>
                    <div class="skeleton-line" style="width: 70%; height: 14px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
                <td>
                    <div class="skeleton-badge" style="width: 80px; height: 24px; border-radius: 4px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
                <td>
                    <div class="skeleton-button" style="width: 70px; height: 28px; border-radius: 4px; background: linear-gradient(90deg, rgba(0,255,0,0.1) 25%, rgba(0,255,0,0.2) 50%, rgba(0,255,0,0.1) 75%); background-size: 200% 100%; animation: shimmer 1.5s infinite;"></div>
                </td>
            `;
            tbody.appendChild(row);
        }
        
        // Add shimmer animation if not already present
        if (!document.getElementById('shimmer-animation')) {
            const style = document.createElement('style');
            style.id = 'shimmer-animation';
            style.textContent = `
                @keyframes shimmer {
                    0% { background-position: -200% 0; }
                    100% { background-position: 200% 0; }
                }
                .skeleton-row td {
                    padding: 12px;
                }
                .skeleton-line, .skeleton-badge, .skeleton-button {
                    border-radius: 4px;
                }
            `;
            document.head.appendChild(style);
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

    // Bulk selection functionality
    selectAll(checked) {
        const checkboxes = document.querySelectorAll('#secrets-table tbody input[type="checkbox"]');
        checkboxes.forEach(checkbox => {
            checkbox.checked = checked;
            const row = checkbox.closest('tr');
            if (row) {
                if (checked) {
                    row.classList.add('selected');
                } else {
                    row.classList.remove('selected');
                }
            }
        });
        this.updateFixButtonState();
    }

    updateFixButtonState() {
        const selectedCheckboxes = document.querySelectorAll('#secrets-table tbody input[type="checkbox"]:checked');
        const fixButton = document.getElementById('fix-selected-btn');
        
        if (selectedCheckboxes.length > 0) {
            fixButton.disabled = false;
            fixButton.textContent = `FIX SELECTED (${selectedCheckboxes.length})`;
        } else {
            fixButton.disabled = true;
            fixButton.textContent = 'FIX SELECTED';
        }
    }

    async fixSelectedVulnerabilities() {
        const selectedCheckboxes = document.querySelectorAll('#secrets-table tbody input[type="checkbox"]:checked');
        
        if (selectedCheckboxes.length === 0) {
            this.showNotification('No vulnerabilities selected', 'error');
            return;
        }

        // Collect selected vulnerabilities
        const selectedVulnerabilities = [];
        selectedCheckboxes.forEach(checkbox => {
            const row = checkbox.closest('tr');
            if (row) {
                const resource = row.querySelector('td:nth-child(2)').textContent;
                const secretKey = row.querySelector('td:nth-child(3)').textContent;
                const type = row.querySelector('td:nth-child(4)').textContent;
                const status = row.querySelector('td:nth-child(5)').textContent;
                
                // Build vulnerability object
                const vulnerability = {
                    id: `${resource}:${secretKey}`,
                    resource_name: resource,
                    secret_key: secretKey,
                    secret_type: type.toLowerCase(),
                    severity: status.includes('MISSING') || status.includes('CRITICAL') ? 'critical' : 
                             status.includes('INVALID') || status.includes('HIGH') ? 'high' : 'medium',
                    description: `Missing or misconfigured secret: ${secretKey} in resource ${resource}`,
                    recommendation: `Store ${secretKey} securely in HashiCorp Vault and update resource configuration`,
                    category: 'secrets_management'
                };
                
                selectedVulnerabilities.push(vulnerability);
            }
        });

        if (selectedVulnerabilities.length === 0) {
            this.showNotification('Could not extract vulnerability details', 'error');
            return;
        }

        this.showNotification(`Spawning Claude Code agent to fix ${selectedVulnerabilities.length} vulnerabilities...`, 'info');

        try {
            const response = await fetch(`${this.apiUrl}/api/v1/vulnerabilities/fix`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    vulnerabilities: selectedVulnerabilities
                })
            });

            if (!response.ok) {
                throw new Error(`Fix request failed: ${response.statusText}`);
            }

            const data = await response.json();
            
            this.showNotification(
                `Claude Code vulnerability fixer agent has been spawned! Request ID: ${data.fix_request_id}`, 
                'success'
            );

            // Clear selections
            this.selectAll(false);
            document.getElementById('select-all').checked = false;

            // Show success notification with details
            setTimeout(() => {
                this.showNotification(
                    'The agent is working in the background to fix your selected vulnerabilities. Check the console for progress updates.',
                    'info'
                );
            }, 3000);

        } catch (error) {
            console.error('Fix vulnerabilities error:', error);
            this.showNotification('Failed to spawn vulnerability fixer: ' + error.message, 'error');
        }
    }

    // Helper function to add click handlers to vulnerability rows
    addVulnerabilityRowClickHandler(row, vulnerability) {
        if (!vulnerability || !vulnerability.file_path) {
            // Not a vulnerability row, skip click handler
            console.log('Skipping click handler - no vulnerability or file_path:', vulnerability);
            return;
        }

        // Add row click handler to open code viewer
        row.style.cursor = 'pointer';
        row.addEventListener('click', (e) => {
            console.log('Row clicked! Vulnerability:', vulnerability.id, vulnerability.file_path);
            // Don't trigger if clicking on checkbox or button
            if (e.target.type === 'checkbox' || e.target.tagName === 'BUTTON' || e.target.closest('button')) {
                console.log('Click on checkbox or button, ignoring');
                return;
            }
            console.log('Opening code viewer for:', vulnerability.file_path);
            this.openCodeViewer(vulnerability);
        });
        console.log('Added click handler for vulnerability:', vulnerability.id, vulnerability.file_path);
    }

    // Code viewer functionality
    async openCodeViewer(vulnerability) {
        console.log('openCodeViewer called with:', vulnerability);
        const modal = document.getElementById('code-viewer-modal');
        if (!modal) {
            console.error('Code viewer modal not found');
            return;
        }

        console.log('Modal found, showing...');
        // Show loading state
        this.showCodeViewerLoading();
        modal.classList.add('show');
        console.log('Modal show class added');

        try {
            // Fetch file content
            const response = await fetch(`/api/v1/files/content?path=${encodeURIComponent(vulnerability.file_path)}`);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || 'Failed to fetch file content');
            }

            // Populate modal with content
            this.displayCodeInViewer(vulnerability, data);

        } catch (error) {
            console.error('Error fetching file content:', error);
            this.showCodeViewerError(error.message);
        }
    }

    showCodeViewerLoading() {
        document.getElementById('code-file-path').textContent = 'Loading...';
        document.getElementById('code-language').textContent = 'Unknown';
        document.getElementById('vuln-type-display').textContent = 'Loading...';
        document.getElementById('vuln-severity-display').textContent = 'Unknown';
        document.getElementById('vuln-description-display').textContent = 'Loading vulnerability details...';
        document.getElementById('vuln-line-number').textContent = '?';
        document.getElementById('code-content').textContent = 'Loading code...';
        document.getElementById('code-content').className = 'language-text';
    }

    showCodeViewerError(message) {
        document.getElementById('code-file-path').textContent = 'Error loading file';
        document.getElementById('code-language').textContent = 'Error';
        document.getElementById('vuln-type-display').textContent = 'Error';
        document.getElementById('vuln-severity-display').textContent = 'Error';
        document.getElementById('vuln-description-display').textContent = `Error: ${message}`;
        document.getElementById('vuln-line-number').textContent = '?';
        document.getElementById('code-content').textContent = `Error loading file: ${message}`;
    }

    displayCodeInViewer(vulnerability, fileData) {
        // Update header info
        document.getElementById('code-file-path').textContent = vulnerability.file_path;
        document.getElementById('code-language').textContent = fileData.language;
        
        // Update vulnerability info
        document.getElementById('vuln-type-display').textContent = vulnerability.type.replace(/_/g, ' ').toUpperCase();
        const severityElement = document.getElementById('vuln-severity-display');
        severityElement.textContent = vulnerability.severity.toUpperCase();
        severityElement.setAttribute('data-severity', vulnerability.severity.toUpperCase());
        document.getElementById('vuln-description-display').textContent = vulnerability.description || vulnerability.title || 'No description available';
        document.getElementById('vuln-line-number').textContent = vulnerability.line_number || '?';

        // Update code content
        const codeElement = document.getElementById('code-content');
        const preElement = codeElement.parentElement;
        
        // Clear any existing line highlighting
        preElement.removeAttribute('data-line');
        
        // Set the code content and language
        codeElement.textContent = fileData.content;
        codeElement.className = `language-${fileData.language}`;
        
        // Ensure line-numbers class is present
        preElement.className = 'code-block line-numbers';
        preElement.style.position = 'relative';
        preElement.style.paddingLeft = '3.8em';

        // Apply syntax highlighting
        if (window.Prism) {
            console.log('Prism available, highlighting code...');
            console.log('Pre element classes:', preElement.className);
            console.log('Code element classes:', codeElement.className);
            
            // First highlight the code
            Prism.highlightElement(codeElement);
            
            // Check if line numbers exist after highlighting
            let lineNumbersExist = preElement.querySelector('.line-numbers-rows');
            console.log('Line numbers exist after Prism highlight:', !!lineNumbersExist);
            
            // Force create line numbers every time for debugging
            if (lineNumbersExist) {
                lineNumbersExist.remove();
                console.log('Removed existing line numbers');
            }
            
            // Create line numbers manually
            const lines = (codeElement.textContent.match(/\n/g) || []).length + 1;
            console.log('Creating line numbers for', lines, 'lines');
            
            const lineNumbersWrapper = document.createElement('span');
            lineNumbersWrapper.className = 'line-numbers-rows';
            lineNumbersWrapper.setAttribute('aria-hidden', 'true');
            lineNumbersWrapper.style.cssText = 'position: absolute; top: 1em; left: 0; width: 3.8em; border-right: 1px solid #00cc33; background: #1a1a1a; color: #808080; z-index: 10;';
            
            for (let i = 0; i < lines; i++) {
                const span = document.createElement('span');
                span.style.cssText = 'display: block; counter-increment: linenumber; text-align: right; padding-right: 0.8em; line-height: 1.5;';
                span.setAttribute('data-line', i + 1);
                lineNumbersWrapper.appendChild(span);
            }
            
            preElement.appendChild(lineNumbersWrapper);
            console.log('Line numbers wrapper added:', preElement.querySelector('.line-numbers-rows'));
            
            // Set the vulnerable line to highlight AFTER Prism processing
            if (vulnerability.line_number) {
                preElement.setAttribute('data-line', vulnerability.line_number.toString());
                // Re-run line highlight plugin
                if (Prism.plugins && Prism.plugins.lineHighlight) {
                    Prism.plugins.lineHighlight.highlightLines(preElement)();
                }
                this.scrollToVulnerableLine(vulnerability.line_number);
            }
        }
    }

    scrollToVulnerableLine(lineNumber) {
        // Scroll to the vulnerable line after Prism has rendered
        setTimeout(() => {
            // Find the highlighted line or line number element
            const highlightedLine = document.querySelector('.code-block .line-highlight');
            if (highlightedLine) {
                highlightedLine.scrollIntoView({ behavior: 'smooth', block: 'center' });
            } else {
                // Fallback: scroll based on line height estimation
                const codeBlock = document.querySelector('.code-block');
                if (codeBlock) {
                    const lineHeight = 24; // Approximate line height in pixels
                    const scrollPosition = (lineNumber - 5) * lineHeight; // Scroll to 5 lines before the target
                    codeBlock.scrollTop = Math.max(0, scrollPosition);
                }
            }
        }, 100); // Small delay to ensure Prism has finished rendering
    }

    setupCodeViewerEventListeners() {
        // Close modal handlers
        const modal = document.getElementById('code-viewer-modal');
        const closeBtn = document.getElementById('code-modal-close');
        const copyBtn = document.getElementById('copy-code-btn');

        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                modal.classList.remove('show');
            });
        }

        // Click outside to close
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.classList.remove('show');
                }
            });
        }

        // Copy code functionality
        if (copyBtn) {
            copyBtn.addEventListener('click', () => {
                const codeContent = document.getElementById('code-content').textContent;
                navigator.clipboard.writeText(codeContent).then(() => {
                    this.showNotification('Code copied to clipboard', 'success');
                }).catch(() => {
                    this.showNotification('Failed to copy code', 'error');
                });
            });
        }

        // ESC key to close
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && modal.classList.contains('show')) {
                modal.classList.remove('show');
            }
        });
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
    window.secretsManager.setupCodeViewerEventListeners();
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