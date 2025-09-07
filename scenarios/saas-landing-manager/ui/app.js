// SaaS Landing Manager Dashboard JavaScript

class SaaSLandingManager {
    constructor() {
        this.apiBaseUrl = '/api/v1';
        this.currentTab = 'scenarios';
        this.scenarios = [];
        this.templates = [];
        this.deployments = [];
        
        this.init();
    }

    // Initialize the application
    async init() {
        this.setupEventListeners();
        this.setupTabNavigation();
        await this.checkApiHealth();
        this.loadInitialData();
    }

    // Setup event listeners
    setupEventListeners() {
        // Header actions
        document.getElementById('refreshBtn').addEventListener('click', () => this.refreshCurrentTab());

        // Scenarios tab
        document.getElementById('scanScenarios').addEventListener('click', () => this.scanScenarios());
        document.getElementById('typeFilter').addEventListener('change', () => this.filterScenarios());
        document.getElementById('searchFilter').addEventListener('input', () => this.filterScenarios());

        // Templates tab
        document.getElementById('createTemplate').addEventListener('click', () => this.showCreateTemplate());

        // Generator tab
        document.getElementById('generateForm').addEventListener('submit', (e) => this.handleGenerateForm(e));
        document.getElementById('scenarioSelect').addEventListener('change', () => this.updateGeneratorPreview());
        document.getElementById('templateSelect').addEventListener('change', () => this.updateGeneratorPreview());

        // Deployments tab
        document.getElementById('deployForm').addEventListener('submit', (e) => this.handleDeployForm(e));

        // Analytics tab
        document.getElementById('timeframeSelect').addEventListener('change', () => this.loadAnalytics());

        // Modal events
        document.querySelector('.modal-close').addEventListener('click', () => this.hideModal());
        document.getElementById('modalCancel').addEventListener('click', () => this.hideModal());
        document.getElementById('modal').addEventListener('click', (e) => {
            if (e.target.id === 'modal') this.hideModal();
        });
    }

    // Setup tab navigation
    setupTabNavigation() {
        const tabs = document.querySelectorAll('.nav-tab');
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const tabName = tab.getAttribute('data-tab');
                this.switchTab(tabName);
            });
        });
    }

    // Switch between tabs
    switchTab(tabName) {
        // Update navigation
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.classList.remove('active');
        });
        document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

        // Update content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(tabName).classList.add('active');

        this.currentTab = tabName;

        // Load tab-specific data
        this.loadTabData(tabName);
    }

    // Load data for specific tab
    async loadTabData(tabName) {
        switch (tabName) {
            case 'scenarios':
                await this.loadScenarios();
                break;
            case 'templates':
                await this.loadTemplates();
                break;
            case 'generator':
                await this.populateGeneratorDropdowns();
                break;
            case 'deployments':
                await this.loadDeployments();
                break;
            case 'analytics':
                await this.loadAnalytics();
                break;
        }
    }

    // API utility methods
    async apiCall(method, endpoint, data = null) {
        try {
            const options = {
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                }
            };

            if (data) {
                options.body = JSON.stringify(data);
            }

            const response = await fetch(this.apiBaseUrl + endpoint, options);
            
            if (!response.ok) {
                throw new Error(`API error: ${response.status} ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API call failed:', error);
            this.showToast('API request failed: ' + error.message, 'error');
            throw error;
        }
    }

    // Check API health
    async checkApiHealth() {
        try {
            await fetch('/health');
            this.updateApiStatus('healthy');
        } catch (error) {
            this.updateApiStatus('unhealthy');
        }
    }

    // Update API status indicator
    updateApiStatus(status) {
        const indicator = document.getElementById('apiStatus');
        const dot = indicator.querySelector('.dot');
        const text = indicator.querySelector('.text');

        if (status === 'healthy') {
            dot.style.background = 'var(--secondary)';
            text.textContent = 'API Healthy';
        } else {
            dot.style.background = 'var(--error)';
            text.textContent = 'API Unavailable';
        }
    }

    // Load initial data
    async loadInitialData() {
        await this.loadTabData(this.currentTab);
    }

    // Refresh current tab data
    async refreshCurrentTab() {
        const refreshBtn = document.getElementById('refreshBtn');
        refreshBtn.disabled = true;
        refreshBtn.innerHTML = '<span class="icon">⏳</span> Refreshing...';

        try {
            await this.loadTabData(this.currentTab);
            this.showToast('Data refreshed successfully', 'success');
        } catch (error) {
            this.showToast('Failed to refresh data', 'error');
        } finally {
            refreshBtn.disabled = false;
            refreshBtn.innerHTML = '<span class="icon">🔄</span> Refresh';
        }
    }

    // Scenarios functionality
    async scanScenarios() {
        const scanBtn = document.getElementById('scanScenarios');
        scanBtn.disabled = true;
        scanBtn.innerHTML = '<span class="icon">⏳</span> Scanning...';

        try {
            const data = await this.apiCall('POST', '/scenarios/scan', {
                force_rescan: true
            });

            this.updateScanStats(data);
            this.scenarios = data.scenarios || [];
            this.renderScenariosTable();
            this.showToast(`Found ${data.saas_scenarios} SaaS scenarios (${data.newly_detected} new)`, 'success');
        } catch (error) {
            this.showToast('Failed to scan scenarios', 'error');
        } finally {
            scanBtn.disabled = false;
            scanBtn.innerHTML = '<span class="icon">🔍</span> Scan Scenarios';
        }
    }

    // Update scan statistics
    updateScanStats(data) {
        document.getElementById('totalScenarios').textContent = data.total_scenarios || 0;
        document.getElementById('saasScenarios').textContent = data.saas_scenarios || 0;
        document.getElementById('newlyDetected').textContent = data.newly_detected || 0;
        
        const withLandingPages = (data.scenarios || []).filter(s => s.has_landing_page).length;
        document.getElementById('withLandingPages').textContent = withLandingPages;
    }

    // Load scenarios data
    async loadScenarios() {
        const container = document.getElementById('scenariosTable');
        container.innerHTML = '<div class="loading">Loading scenarios...</div>';

        try {
            // For now, trigger a scan to get data
            // In a real implementation, we'd have a separate endpoint to just get existing data
            await this.scanScenarios();
        } catch (error) {
            container.innerHTML = '<div class="text-center">Failed to load scenarios. Click "Scan Scenarios" to start.</div>';
        }
    }

    // Render scenarios table
    renderScenariosTable() {
        const container = document.getElementById('scenariosTable');
        
        if (!this.scenarios || this.scenarios.length === 0) {
            container.innerHTML = '<div class="text-center">No SaaS scenarios detected. Click "Scan Scenarios" to search.</div>';
            return;
        }

        const table = document.createElement('table');
        table.className = 'table';
        table.innerHTML = `
            <thead>
                <tr>
                    <th>Scenario</th>
                    <th>Type</th>
                    <th>Industry</th>
                    <th>Confidence</th>
                    <th>Revenue</th>
                    <th>Landing Page</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${this.scenarios.map(scenario => `
                    <tr>
                        <td>
                            <strong>${scenario.display_name || scenario.scenario_name}</strong>
                            <br>
                            <small class="text-gray-500">${scenario.scenario_name}</small>
                        </td>
                        <td><span class="badge">${scenario.saas_type || 'Unknown'}</span></td>
                        <td>${scenario.industry || '-'}</td>
                        <td>
                            <div class="confidence-bar">
                                <div class="confidence-fill" style="width: ${(scenario.confidence_score || 0) * 100}%"></div>
                                <span class="confidence-text">${Math.round((scenario.confidence_score || 0) * 100)}%</span>
                            </div>
                        </td>
                        <td>${scenario.revenue_potential || '-'}</td>
                        <td>
                            ${scenario.has_landing_page 
                                ? '<span class="status-badge success">✓ Yes</span>' 
                                : '<span class="status-badge pending">⏳ No</span>'
                            }
                        </td>
                        <td>
                            <div class="action-buttons">
                                <button class="btn btn-sm btn-primary" onclick="app.generateLandingPage('${scenario.scenario_name}')">
                                    Generate
                                </button>
                                ${scenario.has_landing_page ? 
                                    `<button class="btn btn-sm btn-outline" onclick="app.viewLandingPage('${scenario.scenario_name}')">View</button>` 
                                    : ''
                                }
                            </div>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        `;

        container.innerHTML = '';
        container.appendChild(table);
        this.applyTableStyles();
    }

    // Apply additional table styles
    applyTableStyles() {
        const style = document.createElement('style');
        style.textContent = `
            .badge {
                display: inline-block;
                padding: 0.25rem 0.5rem;
                background: var(--primary-light);
                color: white;
                border-radius: var(--radius-sm);
                font-size: 0.75rem;
                font-weight: 500;
                text-transform: uppercase;
            }
            
            .confidence-bar {
                position: relative;
                height: 20px;
                background: var(--gray-200);
                border-radius: var(--radius-sm);
                overflow: hidden;
            }
            
            .confidence-fill {
                height: 100%;
                background: linear-gradient(90deg, var(--error), var(--warning), var(--secondary));
                transition: width 0.3s ease;
            }
            
            .confidence-text {
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                font-size: 0.75rem;
                font-weight: 600;
                color: var(--gray-700);
            }
            
            .status-badge {
                padding: 0.25rem 0.5rem;
                border-radius: var(--radius-sm);
                font-size: 0.75rem;
                font-weight: 500;
            }
            
            .status-badge.success {
                background: rgba(16, 185, 129, 0.1);
                color: var(--secondary-dark);
            }
            
            .status-badge.pending {
                background: rgba(245, 158, 11, 0.1);
                color: var(--accent-dark);
            }
            
            .action-buttons {
                display: flex;
                gap: 0.5rem;
            }
            
            .btn-sm {
                padding: 0.375rem 0.75rem;
                font-size: 0.75rem;
            }
        `;
        
        if (!document.getElementById('dynamic-styles')) {
            style.id = 'dynamic-styles';
            document.head.appendChild(style);
        }
    }

    // Filter scenarios
    filterScenarios() {
        const typeFilter = document.getElementById('typeFilter').value.toLowerCase();
        const searchFilter = document.getElementById('searchFilter').value.toLowerCase();

        const filteredScenarios = this.scenarios.filter(scenario => {
            const matchesType = !typeFilter || (scenario.saas_type || '').toLowerCase().includes(typeFilter);
            const matchesSearch = !searchFilter || 
                (scenario.scenario_name || '').toLowerCase().includes(searchFilter) ||
                (scenario.display_name || '').toLowerCase().includes(searchFilter) ||
                (scenario.industry || '').toLowerCase().includes(searchFilter);

            return matchesType && matchesSearch;
        });

        // Temporarily replace scenarios for rendering
        const originalScenarios = this.scenarios;
        this.scenarios = filteredScenarios;
        this.renderScenariosTable();
        this.scenarios = originalScenarios;
    }

    // Templates functionality
    async loadTemplates() {
        const container = document.getElementById('templatesGrid');
        container.innerHTML = '<div class="loading">Loading templates...</div>';

        try {
            const data = await this.apiCall('GET', '/templates');
            this.templates = data.templates || [];
            this.renderTemplatesGrid();
        } catch (error) {
            container.innerHTML = '<div class="text-center">Failed to load templates</div>';
        }
    }

    // Render templates grid
    renderTemplatesGrid() {
        const container = document.getElementById('templatesGrid');
        
        if (!this.templates || this.templates.length === 0) {
            container.innerHTML = '<div class="text-center">No templates available</div>';
            return;
        }

        container.innerHTML = this.templates.map(template => `
            <div class="template-card">
                <div class="template-preview">
                    🎨
                </div>
                <div class="template-info">
                    <h3>${template.name}</h3>
                    <div class="template-meta">
                        <span>${template.category}</span>
                        <span>${template.usage_count} uses</span>
                    </div>
                    <p class="text-gray-600">${template.saas_type} • ${template.industry}</p>
                    <div class="template-actions">
                        <button class="btn btn-sm btn-primary" onclick="app.previewTemplate('${template.id}')">
                            Preview
                        </button>
                        <button class="btn btn-sm btn-outline" onclick="app.editTemplate('${template.id}')">
                            Edit
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
    }

    // Generator functionality
    async populateGeneratorDropdowns() {
        // Populate scenario dropdown
        const scenarioSelect = document.getElementById('scenarioSelect');
        scenarioSelect.innerHTML = '<option value="">Choose a scenario...</option>';
        
        if (this.scenarios.length === 0) {
            scenarioSelect.innerHTML += '<option value="" disabled>No scenarios available - scan scenarios first</option>';
        } else {
            this.scenarios.forEach(scenario => {
                scenarioSelect.innerHTML += `<option value="${scenario.scenario_name}">${scenario.display_name || scenario.scenario_name}</option>`;
            });
        }

        // Populate template dropdown
        const templateSelect = document.getElementById('templateSelect');
        templateSelect.innerHTML = '<option value="">Auto-select best template</option>';
        
        this.templates.forEach(template => {
            templateSelect.innerHTML += `<option value="${template.id}">${template.name} (${template.saas_type})</option>`;
        });
    }

    // Handle generate form submission
    async handleGenerateForm(event) {
        event.preventDefault();
        
        const formData = new FormData(event.target);
        const data = {
            scenario_id: formData.get('scenario_id'),
            template_id: formData.get('template_id') || undefined,
            enable_ab_testing: formData.has('enable_ab_testing')
        };

        const submitBtn = event.target.querySelector('button[type="submit"]');
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="icon">⏳</span> Generating...';

        try {
            const result = await this.apiCall('POST', '/landing-pages/generate', data);
            
            this.showToast('Landing page generated successfully!', 'success');
            this.showModal('Landing Page Generated', `
                <div class="space-y-4">
                    <p><strong>Landing Page ID:</strong> ${result.landing_page_id}</p>
                    <p><strong>Status:</strong> ${result.deployment_status}</p>
                    <p><strong>Preview URL:</strong> <a href="${result.preview_url}" target="_blank">${result.preview_url}</a></p>
                    ${result.ab_test_variants.length > 0 ? 
                        `<p><strong>A/B Test Variants:</strong> ${result.ab_test_variants.join(', ')}</p>` 
                        : ''
                    }
                    <div class="form-actions">
                        <button class="btn btn-primary" onclick="app.deployLandingPage('${result.landing_page_id}', '${data.scenario_id}')">
                            Deploy Now
                        </button>
                    </div>
                </div>
            `);

        } catch (error) {
            this.showToast('Failed to generate landing page', 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span class="icon">✨</span> Generate Landing Page';
        }
    }

    // Update generator preview
    updateGeneratorPreview() {
        const scenarioId = document.getElementById('scenarioSelect').value;
        const templateId = document.getElementById('templateSelect').value;
        const previewContainer = document.getElementById('generatorPreview');

        if (!scenarioId) {
            previewContainer.innerHTML = `
                <div class="preview-placeholder">
                    <div class="icon">👁️</div>
                    <p>Select a scenario to see preview</p>
                </div>
            `;
            return;
        }

        const scenario = this.scenarios.find(s => s.scenario_name === scenarioId);
        const template = this.templates.find(t => t.id === templateId);

        previewContainer.innerHTML = `
            <div class="preview-content">
                <h4>Preview for ${scenario?.display_name || scenarioId}</h4>
                <div class="preview-meta">
                    <p><strong>Type:</strong> ${scenario?.saas_type || 'Unknown'}</p>
                    <p><strong>Template:</strong> ${template?.name || 'Auto-selected'}</p>
                    <p><strong>Revenue Potential:</strong> ${scenario?.revenue_potential || 'Unknown'}</p>
                </div>
                <div class="preview-mockup">
                    <div style="background: linear-gradient(135deg, var(--primary-light), var(--secondary)); height: 200px; border-radius: var(--radius-md); display: flex; align-items: center; justify-content: center; color: white; font-size: 1.5rem;">
                        ${scenario?.display_name || scenarioId} Landing Page
                    </div>
                </div>
            </div>
        `;
    }

    // Deployments functionality
    async loadDeployments() {
        const container = document.getElementById('deploymentsList');
        container.innerHTML = '<div class="loading">Loading deployments...</div>';

        // Mock deployment data for now
        setTimeout(() => {
            const mockDeployments = [
                {
                    id: '1',
                    landing_page_id: 'lp-001',
                    target_scenario: 'funnel-builder',
                    status: 'completed',
                    deployed_at: '2 hours ago'
                },
                {
                    id: '2',
                    landing_page_id: 'lp-002',
                    target_scenario: 'invoice-generator',
                    status: 'pending',
                    deployed_at: '5 minutes ago'
                }
            ];

            this.renderDeploymentsList(mockDeployments);
        }, 1000);

        // Also populate target scenario dropdown
        this.populateTargetScenarioDropdown();
    }

    // Render deployments list
    renderDeploymentsList(deployments) {
        const container = document.getElementById('deploymentsList');
        
        container.innerHTML = deployments.map(deployment => `
            <div class="deployment-item">
                <div>
                    <strong>${deployment.target_scenario}</strong>
                    <br>
                    <small class="text-gray-500">LP: ${deployment.landing_page_id}</small>
                </div>
                <div>
                    <div class="deployment-status ${deployment.status}">${deployment.status}</div>
                    <small class="text-gray-500">${deployment.deployed_at}</small>
                </div>
            </div>
        `).join('');
    }

    // Populate target scenario dropdown
    populateTargetScenarioDropdown() {
        const select = document.getElementById('targetScenario');
        select.innerHTML = '<option value="">Choose target scenario...</option>';
        
        this.scenarios.forEach(scenario => {
            select.innerHTML += `<option value="${scenario.scenario_name}">${scenario.display_name || scenario.scenario_name}</option>`;
        });
    }

    // Handle deploy form submission
    async handleDeployForm(event) {
        event.preventDefault();
        
        const formData = new FormData(event.target);
        const landingPageId = formData.get('landing_page_id');
        const data = {
            target_scenario: formData.get('target_scenario'),
            deployment_method: formData.get('deployment_method'),
            backup_existing: formData.has('backup_existing')
        };

        const submitBtn = event.target.querySelector('button[type="submit"]');
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="icon">⏳</span> Deploying...';

        try {
            const result = await this.apiCall('POST', `/landing-pages/${landingPageId}/deploy`, data);
            
            this.showToast('Deployment started successfully!', 'success');
            this.showModal('Deployment Started', `
                <div class="space-y-4">
                    <p><strong>Deployment ID:</strong> ${result.deployment_id}</p>
                    <p><strong>Status:</strong> ${result.status}</p>
                    ${result.agent_session_id ? 
                        `<p><strong>Agent Session:</strong> ${result.agent_session_id}</p>` 
                        : ''
                    }
                    <p><strong>Estimated Completion:</strong> ${result.estimated_completion}</p>
                </div>
            `);

            // Reload deployments
            this.loadDeployments();

        } catch (error) {
            this.showToast('Failed to start deployment', 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span class="icon">🚀</span> Deploy Landing Page';
        }
    }

    // Analytics functionality
    async loadAnalytics() {
        try {
            const data = await this.apiCall('GET', '/analytics/dashboard');
            this.updateAnalyticsCards(data);
        } catch (error) {
            this.showToast('Failed to load analytics', 'error');
        }
    }

    // Update analytics cards
    updateAnalyticsCards(data) {
        document.getElementById('totalLandingPages').textContent = data.total_pages || 0;
        document.getElementById('activeABTests').textContent = data.active_ab_tests || 0;
        document.getElementById('avgConversionRate').textContent = 
            ((data.average_conversion_rate || 0) * 100).toFixed(1) + '%';
        document.getElementById('totalConversions').textContent = data.total_conversions || 0;
    }

    // Action methods called from HTML
    async generateLandingPage(scenarioName) {
        this.switchTab('generator');
        document.getElementById('scenarioSelect').value = scenarioName;
        this.updateGeneratorPreview();
    }

    async viewLandingPage(scenarioName) {
        const scenario = this.scenarios.find(s => s.scenario_name === scenarioName);
        if (scenario && scenario.landing_page_url) {
            window.open(scenario.landing_page_url, '_blank');
        } else {
            this.showToast('Landing page URL not available', 'warning');
        }
    }

    async deployLandingPage(landingPageId, targetScenario) {
        this.hideModal();
        this.switchTab('deployments');
        document.getElementById('landingPageId').value = landingPageId;
        document.getElementById('targetScenario').value = targetScenario;
    }

    previewTemplate(templateId) {
        const template = this.templates.find(t => t.id === templateId);
        if (template) {
            this.showModal(`Template Preview: ${template.name}`, `
                <div class="template-preview-modal">
                    <div class="preview-info">
                        <p><strong>Category:</strong> ${template.category}</p>
                        <p><strong>SaaS Type:</strong> ${template.saas_type}</p>
                        <p><strong>Industry:</strong> ${template.industry}</p>
                        <p><strong>Usage Count:</strong> ${template.usage_count}</p>
                        <p><strong>Rating:</strong> ${template.rating}/5.0</p>
                    </div>
                    <div class="preview-mockup" style="height: 300px; background: linear-gradient(135deg, var(--primary-light), var(--secondary)); border-radius: var(--radius-md); display: flex; align-items: center; justify-content: center; color: white; font-size: 2rem; margin-top: 1rem;">
                        ${template.name} Preview
                    </div>
                </div>
            `);
        }
    }

    editTemplate(templateId) {
        this.showToast('Template editing coming soon!', 'info');
    }

    showCreateTemplate() {
        this.showToast('Template creation coming soon!', 'info');
    }

    // UI utility methods
    showToast(message, type = 'info') {
        const container = document.getElementById('toastContainer');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <div class="toast-content">
                <strong>${type.charAt(0).toUpperCase() + type.slice(1)}</strong>
                <p>${message}</p>
            </div>
        `;

        container.appendChild(toast);

        // Remove toast after 5 seconds
        setTimeout(() => {
            if (container.contains(toast)) {
                container.removeChild(toast);
            }
        }, 5000);
    }

    showModal(title, content) {
        const modal = document.getElementById('modal');
        const modalTitle = document.getElementById('modalTitle');
        const modalBody = document.getElementById('modalBody');

        modalTitle.textContent = title;
        modalBody.innerHTML = content;
        modal.classList.add('active');
    }

    hideModal() {
        const modal = document.getElementById('modal');
        modal.classList.remove('active');
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new SaaSLandingManager();
});

// Add some additional CSS for dynamic elements
document.addEventListener('DOMContentLoaded', () => {
    const additionalStyles = `
        <style>
            .preview-content {
                text-align: center;
            }
            
            .preview-meta {
                background: var(--gray-50);
                padding: 1rem;
                border-radius: var(--radius-md);
                margin: 1rem 0;
                text-align: left;
            }
            
            .preview-meta p {
                margin-bottom: 0.5rem;
            }
            
            .preview-meta p:last-child {
                margin-bottom: 0;
            }
            
            .preview-mockup {
                margin-top: 1rem;
            }
            
            .space-y-4 > * + * {
                margin-top: 1rem;
            }
            
            .template-preview-modal .preview-info {
                background: var(--gray-50);
                padding: 1rem;
                border-radius: var(--radius-md);
                margin-bottom: 1rem;
            }
            
            .template-preview-modal .preview-info p {
                margin-bottom: 0.5rem;
            }
            
            .template-preview-modal .preview-info p:last-child {
                margin-bottom: 0;
            }
            
            .toast-content {
                display: flex;
                flex-direction: column;
                gap: 0.25rem;
            }
            
            .toast-content p {
                margin: 0;
                font-size: 0.875rem;
                color: var(--gray-600);
            }
            
            .text-gray-500 {
                color: var(--gray-500);
            }
            
            .text-gray-600 {
                color: var(--gray-600);
            }
        </style>
    `;
    
    document.head.insertAdjacentHTML('beforeend', additionalStyles);
});