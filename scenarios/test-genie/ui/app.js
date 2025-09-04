// Test Genie Dashboard JavaScript
class TestGenieApp {
    constructor() {
        this.apiBaseUrl = '/api/v1';
        this.activePage = 'dashboard';
        this.currentData = {
            suites: [],
            executions: [],
            metrics: {}
        };
        
        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.setupNavigation();
        await this.loadInitialData();
        this.startPeriodicUpdates();
    }

    setupEventListeners() {
        // Navigation
        document.addEventListener('click', (e) => {
            if (e.target.closest('.nav-item')) {
                const navItem = e.target.closest('.nav-item');
                const page = navItem.getAttribute('data-page');
                if (page) {
                    this.navigateTo(page);
                }
            }
        });

        // Form submissions
        document.getElementById('generate-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleGenerateSubmit();
        });

        document.getElementById('coverage-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCoverageSubmit();
        });

        document.getElementById('vault-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleVaultSubmit();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch(e.key) {
                    case '1':
                        e.preventDefault();
                        this.navigateTo('dashboard');
                        break;
                    case '2':
                        e.preventDefault();
                        this.navigateTo('generate');
                        break;
                    case '3':
                        e.preventDefault();
                        this.navigateTo('suites');
                        break;
                    case 'r':
                        e.preventDefault();
                        this.refreshCurrentPage();
                        break;
                }
            }
        });
    }

    setupNavigation() {
        // Handle browser back/forward
        window.addEventListener('popstate', (e) => {
            const page = e.state?.page || 'dashboard';
            this.navigateTo(page, false);
        });

        // Set initial state
        const initialPage = window.location.hash.replace('#', '') || 'dashboard';
        this.navigateTo(initialPage, false);
    }

    navigateTo(page, pushState = true) {
        // Update active nav item
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`.nav-item[data-page="${page}"]`)?.classList.add('active');

        // Update active page
        document.querySelectorAll('.page').forEach(p => {
            p.classList.remove('active');
        });
        document.getElementById(page)?.classList.add('active');

        // Update URL
        if (pushState) {
            history.pushState({ page }, '', `#${page}`);
        }

        this.activePage = page;

        // Load page-specific data
        this.loadPageData(page);
    }

    async loadInitialData() {
        try {
            await this.checkApiHealth();
            await this.loadDashboardData();
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showError('Failed to connect to Test Genie API');
        }
    }

    async checkApiHealth() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/health`);
            const data = await response.json();
            
            if (data.status === 'healthy') {
                this.updateSystemStatus(true);
            } else {
                this.updateSystemStatus(false);
            }
        } catch (error) {
            console.error('Health check failed:', error);
            this.updateSystemStatus(false);
        }
    }

    updateSystemStatus(healthy) {
        const statusDot = document.querySelector('.status-dot');
        const statusText = document.querySelector('.system-status span');
        
        if (healthy) {
            statusDot.style.background = '#39ff14';
            statusDot.style.boxShadow = '0 0 10px #39ff14';
            statusText.textContent = 'System Healthy';
        } else {
            statusDot.style.background = '#ff0040';
            statusDot.style.boxShadow = '0 0 10px #ff0040';
            statusText.textContent = 'System Offline';
        }
    }

    async loadDashboardData() {
        // Update header stats (mock data for now)
        this.updateHeaderStats({
            activeSuites: Math.floor(Math.random() * 20) + 5,
            runningTests: Math.floor(Math.random() * 10),
            avgCoverage: Math.floor(Math.random() * 20) + 80
        });

        // Update dashboard metrics (mock data for now)
        this.updateDashboardMetrics({
            totalSuites: Math.floor(Math.random() * 100) + 50,
            testsGenerated: Math.floor(Math.random() * 1000) + 500,
            avgCoverage: Math.floor(Math.random() * 20) + 80,
            failedTests: Math.floor(Math.random() * 10)
        });

        // Load recent executions
        await this.loadRecentExecutions();
    }

    updateHeaderStats(stats) {
        document.getElementById('active-suites').textContent = stats.activeSuites;
        document.getElementById('running-tests').textContent = stats.runningTests;
        document.getElementById('avg-coverage').textContent = `${stats.avgCoverage}%`;
    }

    updateDashboardMetrics(metrics) {
        document.getElementById('metric-total-suites').textContent = metrics.totalSuites;
        document.getElementById('metric-tests-generated').textContent = metrics.testsGenerated;
        document.getElementById('metric-avg-coverage').textContent = `${metrics.avgCoverage}%`;
        document.getElementById('metric-failed-tests').textContent = metrics.failedTests;
    }

    async loadRecentExecutions() {
        const container = document.getElementById('recent-executions');
        
        // Mock recent executions data
        const executions = [
            {
                id: 'exec-001',
                suiteName: 'document-manager',
                status: 'completed',
                duration: 120.5,
                passed: 28,
                failed: 2,
                timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString()
            },
            {
                id: 'exec-002',
                suiteName: 'personal-digital-twin',
                status: 'running',
                duration: 45.2,
                passed: 15,
                failed: 0,
                timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString()
            },
            {
                id: 'exec-003',
                suiteName: 'test-genie',
                status: 'completed',
                duration: 89.7,
                passed: 42,
                failed: 1,
                timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString()
            }
        ];

        container.innerHTML = this.renderExecutionsTable(executions);
    }

    renderExecutionsTable(executions) {
        if (!executions || executions.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No recent executions found</p>';
        }

        return `
            <table class="table">
                <thead>
                    <tr>
                        <th>Suite Name</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Passed</th>
                        <th>Failed</th>
                        <th>Time</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${executions.map(exec => `
                        <tr>
                            <td><strong>${exec.suiteName}</strong></td>
                            <td><span class="status ${this.getStatusClass(exec.status)}">${exec.status}</span></td>
                            <td>${exec.duration}s</td>
                            <td style="color: var(--accent-success);">${exec.passed}</td>
                            <td style="color: ${exec.failed > 0 ? 'var(--accent-error)' : 'var(--text-muted)'};">${exec.failed}</td>
                            <td>${this.formatTimestamp(exec.timestamp)}</td>
                            <td>
                                <button class="btn" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onclick="app.viewExecution('${exec.id}')">
                                    📊 View
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    getStatusClass(status) {
        switch (status) {
            case 'completed': return 'success';
            case 'running': return 'info';
            case 'failed': return 'error';
            default: return 'warning';
        }
    }

    formatTimestamp(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / (1000 * 60));
        const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
        
        if (diffMins < 60) {
            return `${diffMins}m ago`;
        } else if (diffHours < 24) {
            return `${diffHours}h ago`;
        } else {
            return date.toLocaleDateString();
        }
    }

    async loadPageData(page) {
        switch (page) {
            case 'suites':
                await this.loadTestSuites();
                break;
            case 'executions':
                await this.loadExecutions();
                break;
            case 'dashboard':
                await this.loadDashboardData();
                break;
        }
    }

    async loadTestSuites() {
        const container = document.getElementById('suites-table');
        container.innerHTML = '<div class="loading"><div class="spinner"></div>Loading test suites...</div>';

        // Mock test suites data
        setTimeout(() => {
            const suites = [
                {
                    id: 'suite-001',
                    scenarioName: 'document-manager',
                    suiteType: 'unit,integration',
                    testsCount: 45,
                    coverage: 92.3,
                    status: 'active',
                    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 2).toISOString()
                },
                {
                    id: 'suite-002',
                    scenarioName: 'personal-digital-twin',
                    suiteType: 'unit,integration,performance',
                    testsCount: 67,
                    coverage: 88.7,
                    status: 'active',
                    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString()
                },
                {
                    id: 'suite-003',
                    scenarioName: 'test-genie',
                    suiteType: 'unit,integration,vault',
                    testsCount: 89,
                    coverage: 95.1,
                    status: 'active',
                    createdAt: new Date(Date.now() - 1000 * 60 * 60 * 12).toISOString()
                }
            ];

            container.innerHTML = this.renderSuitesTable(suites);
        }, 1000);
    }

    renderSuitesTable(suites) {
        if (!suites || suites.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test suites found</p>';
        }

        return `
            <table class="table">
                <thead>
                    <tr>
                        <th>Scenario</th>
                        <th>Types</th>
                        <th>Tests</th>
                        <th>Coverage</th>
                        <th>Status</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${suites.map(suite => `
                        <tr>
                            <td><strong>${suite.scenarioName}</strong></td>
                            <td>${suite.suiteType}</td>
                            <td>${suite.testsCount}</td>
                            <td>
                                <div class="progress" style="width: 80px; margin-right: 0.5rem; display: inline-block;">
                                    <div class="progress-bar" style="width: ${suite.coverage}%"></div>
                                </div>
                                ${suite.coverage}%
                            </td>
                            <td><span class="status ${this.getStatusClass(suite.status)}">${suite.status}</span></td>
                            <td>${this.formatTimestamp(suite.createdAt)}</td>
                            <td>
                                <button class="btn" style="padding: 0.25rem 0.5rem; font-size: 0.75rem; margin-right: 0.5rem;" onclick="app.executeSuite('${suite.id}')">
                                    🚀 Execute
                                </button>
                                <button class="btn secondary" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onclick="app.viewSuite('${suite.id}')">
                                    👁️ View
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    async loadExecutions() {
        const container = document.getElementById('executions-table');
        container.innerHTML = '<div class="loading"><div class="spinner"></div>Loading test executions...</div>';

        // Use the same mock data as recent executions but with more entries
        setTimeout(() => {
            const executions = [
                {
                    id: 'exec-001',
                    suiteName: 'document-manager',
                    status: 'completed',
                    duration: 120.5,
                    passed: 28,
                    failed: 2,
                    timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString()
                },
                {
                    id: 'exec-002',
                    suiteName: 'personal-digital-twin',
                    status: 'running',
                    duration: 45.2,
                    passed: 15,
                    failed: 0,
                    timestamp: new Date(Date.now() - 1000 * 60 * 5).toISOString()
                },
                {
                    id: 'exec-003',
                    suiteName: 'test-genie',
                    status: 'completed',
                    duration: 89.7,
                    passed: 42,
                    failed: 1,
                    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2).toISOString()
                },
                {
                    id: 'exec-004',
                    suiteName: 'app-monitor',
                    status: 'failed',
                    duration: 25.3,
                    passed: 8,
                    failed: 5,
                    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 6).toISOString()
                },
                {
                    id: 'exec-005',
                    suiteName: 'scenario-improver',
                    status: 'completed',
                    duration: 156.8,
                    passed: 67,
                    failed: 0,
                    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 24).toISOString()
                }
            ];

            container.innerHTML = this.renderExecutionsTable(executions);
        }, 1000);
    }

    async handleGenerateSubmit() {
        const form = document.getElementById('generate-form');
        const btn = document.getElementById('generate-btn');
        const resultDiv = document.getElementById('generation-result');

        const formData = new FormData(form);
        const scenarioName = document.getElementById('scenario-name').value;
        const testTypes = Array.from(document.getElementById('test-types').selectedOptions).map(option => option.value);
        const coverageTarget = parseInt(document.getElementById('coverage-target').value);
        const includePerformance = document.getElementById('include-performance').checked;
        const includeSecurity = document.getElementById('include-security').checked;

        if (!scenarioName || testTypes.length === 0) {
            this.showError('Please fill in all required fields');
            return;
        }

        btn.disabled = true;
        btn.innerHTML = '<div class="spinner"></div> Generating...';
        resultDiv.style.display = 'none';

        try {
            const requestData = {
                scenario_name: scenarioName,
                test_types: testTypes,
                coverage_target: coverageTarget,
                options: {
                    include_performance_tests: includePerformance,
                    include_security_tests: includeSecurity,
                    custom_test_patterns: [],
                    execution_timeout: 300
                }
            };

            const response = await fetch(`${this.apiBaseUrl}/test-suite/generate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            
            resultDiv.innerHTML = `✅ Test Suite Generated Successfully!

Suite ID: ${result.suite_id}
Generated Tests: ${result.generated_tests}
Estimated Coverage: ${result.estimated_coverage}%
Generation Time: ${result.generation_time}s

Test Files Generated:
${Object.entries(result.test_files || {}).map(([type, files]) => 
    `  ${type}: ${files.length} files`
).join('\n')}`;

            resultDiv.style.display = 'block';
            
            // Show success notification
            this.showSuccess(`Test suite generated successfully! ${result.generated_tests} tests created.`);

        } catch (error) {
            console.error('Test generation failed:', error);
            this.showError(`Test generation failed: ${error.message}`);
        } finally {
            btn.disabled = false;
            btn.innerHTML = '⚡ Generate Test Suite';
        }
    }

    async handleCoverageSubmit() {
        const scenarioName = document.getElementById('coverage-scenario').value;
        const analysisDepth = document.getElementById('analysis-depth').value;
        const resultDiv = document.getElementById('coverage-result');

        if (!scenarioName) {
            this.showError('Please enter a scenario name');
            return;
        }

        resultDiv.style.display = 'none';

        try {
            const requestData = {
                scenario_name: scenarioName,
                source_code_paths: ['./api', './cli', './ui'],
                existing_test_paths: ['./test'],
                analysis_depth: analysisDepth
            };

            const response = await fetch(`${this.apiBaseUrl}/test-analysis/coverage`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(requestData)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            
            resultDiv.innerHTML = `📊 Coverage Analysis Complete!

Overall Coverage: ${result.overall_coverage}%

Coverage by File:
${Object.entries(result.coverage_by_file || {}).map(([file, coverage]) => 
    `  ${file}: ${coverage}%`
).join('\n')}

Coverage Gaps Found:
  Untested Functions: ${result.coverage_gaps?.untested_functions?.length || 0}
  Untested Branches: ${result.coverage_gaps?.untested_branches?.length || 0}
  Untested Edge Cases: ${result.coverage_gaps?.untested_edge_cases?.length || 0}

Priority Areas:
${(result.priority_areas || []).map(area => `  🎯 ${area}`).join('\n')}

Improvement Suggestions:
${(result.improvement_suggestions || []).map(suggestion => `  • ${suggestion}`).join('\n')}`;

            resultDiv.style.display = 'block';
            this.showSuccess('Coverage analysis completed successfully!');

        } catch (error) {
            console.error('Coverage analysis failed:', error);
            this.showError(`Coverage analysis failed: ${error.message}`);
        }
    }

    async handleVaultSubmit() {
        const scenarioName = document.getElementById('vault-scenario').value;
        const phases = Array.from(document.getElementById('vault-phases').selectedOptions).map(option => option.value);
        const timeout = parseInt(document.getElementById('vault-timeout').value);
        const resultDiv = document.getElementById('vault-result');

        if (!scenarioName || phases.length === 0) {
            this.showError('Please fill in all required fields');
            return;
        }

        resultDiv.style.display = 'none';

        // Simulate vault creation (this would use CLI in real implementation)
        setTimeout(() => {
            const vaultId = `vault-${Date.now().toString(36)}`;
            
            resultDiv.innerHTML = `🏛️ Test Vault Created Successfully!

Vault ID: ${vaultId}
Scenario: ${scenarioName}
Phases: ${phases.join(', ')}
Phase Timeout: ${timeout}s
Total Estimated Time: ${timeout * phases.length}s

Vault Structure Created:
  📁 test-vault-${scenarioName}/
    📄 vault.yaml
    📄 run_vault.sh
    📁 phases/
${phases.map(phase => `      📄 ${phase}.yaml`).join('\n')}

To execute the vault:
  cd test-vault-${scenarioName}
  ./run_vault.sh`;

            resultDiv.style.display = 'block';
            this.showSuccess(`Test vault created successfully with ${phases.length} phases!`);
        }, 1500);
    }

    // Action handlers
    async executeSuite(suiteId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/test-suite/${suiteId}/execute`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    execution_type: 'full',
                    environment: 'local',
                    parallel_execution: false,
                    timeout_seconds: 300,
                    notification_settings: {
                        on_completion: true,
                        on_failure: true
                    }
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            this.showSuccess(`Test execution started! Execution ID: ${result.execution_id}`);
            
            // Navigate to executions page to monitor progress
            this.navigateTo('executions');

        } catch (error) {
            console.error('Test execution failed:', error);
            this.showError(`Test execution failed: ${error.message}`);
        }
    }

    viewSuite(suiteId) {
        this.showInfo(`Viewing test suite ${suiteId}. Detailed suite view coming soon!`);
    }

    viewExecution(executionId) {
        this.showInfo(`Viewing execution ${executionId}. Detailed execution view coming soon!`);
    }

    // Utility methods
    async refreshCurrentPage() {
        await this.loadPageData(this.activePage);
        this.showSuccess('Page refreshed!');
    }

    refreshDashboard() {
        this.loadDashboardData();
    }

    startPeriodicUpdates() {
        // Update system status every 30 seconds
        setInterval(() => {
            this.checkApiHealth();
        }, 30000);

        // Update dashboard metrics every 60 seconds
        setInterval(() => {
            if (this.activePage === 'dashboard') {
                this.loadDashboardData();
            }
        }, 60000);
    }

    // Notification methods
    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showInfo(message) {
        this.showNotification(message, 'info');
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 1rem 1.5rem;
            background: ${type === 'success' ? 'var(--accent-success)' : 
                         type === 'error' ? 'var(--accent-error)' : 
                         'var(--accent-secondary)'};
            color: var(--bg-primary);
            border-radius: var(--panel-border-radius);
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
            z-index: 10000;
            font-weight: 500;
            max-width: 400px;
            transform: translateX(100%);
            transition: transform 0.3s ease;
        `;
        
        notification.textContent = message;
        document.body.appendChild(notification);

        // Animate in
        setTimeout(() => {
            notification.style.transform = 'translateX(0)';
        }, 100);

        // Remove after 5 seconds
        setTimeout(() => {
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 5000);
    }
}

// Initialize the app
const app = new TestGenieApp();