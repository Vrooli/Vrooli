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
        
        // Initialize with vault status check and compliance data
        setTimeout(async () => {
            await this.loadVaultStatus();
            await this.loadComplianceData();
        }, 1000);
    }
    
    bindEvents() {
        // Scan button - now checks vault status
        document.getElementById('scan-all-btn').addEventListener('click', () => {
            this.loadVaultStatus();
        });
        
        // Validate button - now runs security scan
        document.getElementById('validate-btn').addEventListener('click', () => {
            this.performSecurityScan();
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
            
            const vulnCount = this.securityScanResult.vulnerabilities ? this.securityScanResult.vulnerabilities.length : 0;
            const riskScore = this.securityScanResult.risk_score || 0;
            
            const message = `Security scan complete: ${vulnCount} vulnerabilities found (Risk: ${riskScore}/100)`;
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
                this.updateVaultSecretsTable();
            }
            
            // Refresh compliance data
            const complianceResponse = await fetch(`${this.apiUrl}/api/v1/security/compliance`);
            if (complianceResponse.ok) {
                this.complianceData = await complianceResponse.json();
                this.updateHealthStats(this.complianceData);
                this.updateSecurityAlerts(this.complianceData);
            }
        } catch (error) {
            console.warn('Auto-refresh failed:', error);
        }
    }
    
    updateHealthStats(data) {
        // Update stats from compliance data
        if (data.configured_resources !== undefined) {
            document.getElementById('valid-secrets').textContent = data.configured_resources || 0;
        }
        if (data.total_resources !== undefined) {
            document.getElementById('total-secrets').textContent = data.total_resources || 0;
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
        
        // Update vault and database status
        document.getElementById('vault-status').textContent = 
            data.vault_secrets_health > 75 ? 'Operational' : 
            data.vault_secrets_health > 50 ? 'Degraded' : 'Critical';
        document.getElementById('db-status').textContent = 'Connected';
    }
    
    updateVaultSecretsTable() {
        const tbody = document.getElementById('secrets-table-body');
        tbody.innerHTML = '';
        
        if (!this.vaultStatus || !this.vaultStatus.resource_statuses) {
            tbody.innerHTML = `
                <tr class="loading-row">
                    <td colspan="5">
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
                <td>${resource.resource_name}</td>
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
            
            tbody.appendChild(row);
        });
        
        // Add missing secrets details if any
        if (this.vaultStatus.missing_secrets && this.vaultStatus.missing_secrets.length > 0) {
            this.vaultStatus.missing_secrets.forEach(missing => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${missing.resource_name}</td>
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
    
    filterTableByType(label) {
        // Show skeleton immediately for smooth transition
        this.showTableSkeleton();
        
        // Update the section title based on what's being shown
        const sectionTitle = document.querySelector('.secrets-panel .panel-header h2');
        
        // Small delay to show the skeleton animation
        setTimeout(() => {
            if (label.includes('Configured')) {
                // Show configured resources
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> CONFIGURED RESOURCES';
                this.showConfiguredResources();
            } else if (label.includes('Total')) {
                // Show all resources
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> ALL RESOURCES';
                this.updateVaultSecretsTable();
            } else if (label.includes('Critical')) {
                // Show critical vulnerabilities
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> CRITICAL VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('critical');
            } else if (label.includes('High')) {
                // Show high vulnerabilities
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> HIGH SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('high');
            } else if (label.includes('Medium')) {
                // Show medium vulnerabilities
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> MEDIUM SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('medium');
            } else if (label.includes('Low')) {
                // Show low vulnerabilities
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> LOW SEVERITY VULNERABILITIES';
                this.showVulnerabilitiesBySeverity('low');
            } else {
                // Default to showing vault status
                sectionTitle.innerHTML = '<span class="matrix-green">●</span> RESOURCE VAULT STATUS';
                this.updateVaultSecretsTable();
            }
        }, 300); // 300ms delay to show skeleton animation
    }
    
    showConfiguredResources() {
        const tbody = document.getElementById('secrets-table-body');
        
        if (!this.vaultStatus || !this.vaultStatus.resource_statuses) {
            tbody.innerHTML = `
                <tr><td colspan="5" class="chrome-text">No vault data available. Click "CHECK VAULT" to scan.</td></tr>
            `;
            return;
        }
        
        // Filter for healthy/configured resources only
        const configuredResources = this.vaultStatus.resource_statuses.filter(r => r.health_status === 'healthy');
        
        if (configuredResources.length === 0) {
            tbody.innerHTML = `
                <tr><td colspan="5" class="chrome-text">No fully configured resources found.</td></tr>
            `;
            return;
        }
        
        configuredResources.forEach(resource => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${resource.resource_name}</td>
                <td><code>${resource.secrets_found}/${resource.secrets_total} secrets</code></td>
                <td>VAULT STATUS</td>
                <td><span class="status-badge valid">CONFIGURED</span></td>
                <td><span class="chrome-text">✓ ALL SECRETS SET</span></td>
            `;
            tbody.appendChild(row);
        });
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
                    <tr><td colspan="5" class="chrome-text">No ${severity} vulnerabilities found.</td></tr>
                `;
                return;
            }
            
            // Clear table and show each vulnerability
            tbody.innerHTML = '';
            vulnerabilities.forEach(vuln => {
                const row = document.createElement('tr');
                const severityBadge = vuln.severity === 'critical' ? 'missing' :
                                      vuln.severity === 'high' ? 'invalid' :
                                      vuln.severity === 'medium' ? 'discovered' : 'valid';
                
                row.innerHTML = `
                    <td>${vuln.scenario_name || 'Unknown'}</td>
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
                tbody.appendChild(row);
            });
            
        } catch (error) {
            console.error('Error fetching vulnerabilities:', error);
            
            // Fallback to showing summary from compliance data
            const count = this.vulnerabilityCounts ? this.vulnerabilityCounts[severity] : 0;
            
            if (count === 0) {
                tbody.innerHTML = `
                    <tr><td colspan="5" class="chrome-text">No ${severity} vulnerabilities detected.</td></tr>
                `;
                return;
            }
            
            // Show summary message with call to action
            tbody.innerHTML = `
                <tr>
                    <td colspan="5" class="chrome-text" style="text-align: center;">
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
        const sectionTitle = document.querySelector('.section-header h2');
        if (sectionTitle) {
            sectionTitle.textContent = 'RESOURCE VAULT STATUS';
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
        const select = document.getElementById('resource-filter');
        
        // Get resources from vault status
        const resources = this.vaultStatus && this.vaultStatus.resource_statuses ? 
            this.vaultStatus.resource_statuses.map(r => r.resource_name) : [];
        
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
            scanBtn.innerHTML = '<span class="btn-glow"></span>SCANNING...';
            this.showTableSkeleton();
        } else {
            scanBtn.disabled = false;
            scanBtn.innerHTML = '<span class="btn-glow"></span>CHECK VAULT';
        }
    }
    
    showTableSkeleton() {
        const tbody = document.getElementById('secrets-table-body');
        tbody.innerHTML = '';
        
        // Create skeleton rows
        for (let i = 0; i < 5; i++) {
            const row = document.createElement('tr');
            row.className = 'skeleton-row';
            row.innerHTML = `
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