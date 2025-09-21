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
        this.refreshInterval = null;
        this.wsConnection = null;
        
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
        try {
            // Load system metrics from API
            const [systemMetrics, testSuitesData, executionsData] = await Promise.all([
                this.fetchWithErrorHandling('/system/metrics'),
                this.fetchWithErrorHandling('/test-suites'),
                this.fetchWithErrorHandling('/test-executions')
            ]);

            const suites = this.normalizeCollection(testSuitesData, 'test_suites');
            const executions = this.normalizeCollection(executionsData, 'executions');

            this.currentData.suites = suites;
            this.currentData.executions = executions;

            // Calculate dashboard metrics from real data
            const activeSuites = suites.filter(s => (s.status || '').toLowerCase() === 'active').length;
            const runningExecutions = executions.filter(e => (e.status || '').toLowerCase() === 'running').length;
            
            let avgCoverage = 0;
            if (suites.length > 0) {
                const totalCoverage = suites.reduce((sum, suite) => {
                    const coverage = suite.coverage_metrics?.code_coverage ?? suite.coverage ?? 0;
                    return sum + Number(coverage);
                }, 0);
                avgCoverage = Math.round(totalCoverage / suites.length);
            }

            // Update header stats with real data
            this.updateHeaderStats({
                activeSuites,
                runningTests: runningExecutions,
                avgCoverage
            });

            // Calculate additional metrics
            const totalSuites = suites.length;
            const totalTestCases = suites.reduce((sum, suite) => {
                if (typeof suite.total_tests === 'number') {
                    return sum + suite.total_tests;
                }
                if (Array.isArray(suite.test_cases)) {
                    return sum + suite.test_cases.length;
                }
                return sum;
            }, 0);
            const failedTests = executions.filter(e => (e.status || '').toLowerCase() === 'failed').length;

            this.updateDashboardMetrics({
                totalSuites,
                testsGenerated: totalTestCases,
                avgCoverage,
                failedTests
            });

            // Load recent executions with real data
            await this.loadRecentExecutions();
            
        } catch (error) {
            console.error('Failed to load dashboard data:', error);
            this.showError('Failed to load dashboard data');
        }
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
        
        try {
            const executionsResponse = await this.fetchWithErrorHandling('/test-executions?limit=5&sort=desc');
            const executions = this.normalizeCollection(executionsResponse, 'executions');

            if (executions && executions.length > 0) {
                // Transform API data to match our display format
                const formattedExecutions = executions.map(exec => ({
                    id: exec.id,
                    suiteName: this.lookupSuiteName(exec.suite_id) || exec.suite_name || 'Unknown Suite',
                    status: exec.status,
                    duration: this.calculateDuration(exec.start_time, exec.end_time),
                    passed: exec.passed_tests || 0,
                    failed: exec.failed_tests || 0,
                    timestamp: exec.start_time
                }));
                
                container.innerHTML = this.renderExecutionsTable(formattedExecutions);
            } else {
                container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No recent executions found</p>';
            }
        } catch (error) {
            console.error('Failed to load recent executions:', error);
            container.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load recent executions</p>';
        }
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

        try {
            const suitesResponse = await this.fetchWithErrorHandling('/test-suites');
            const suites = this.normalizeCollection(suitesResponse, 'test_suites');
            this.currentData.suites = suites;

            if (suites && suites.length > 0) {
                // Transform API data to match our display format
                const formattedSuites = suites.map(suite => ({
                    id: suite.id,
                    scenarioName: suite.scenario_name,
                    suiteType: Array.isArray(suite.test_types) ? suite.test_types.join(',') : (suite.suite_type || 'unknown'),
                    testsCount: typeof suite.total_tests === 'number' ? suite.total_tests : (Array.isArray(suite.test_cases) ? suite.test_cases.length : 0),
                    coverage: Number(suite.coverage_metrics?.code_coverage ?? suite.coverage ?? 0),
                    status: suite.status || 'unknown',
                    createdAt: suite.generated_at || suite.created_at
                }));
                
                container.innerHTML = this.renderSuitesTable(formattedSuites);
            } else {
                container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test suites found</p>';
            }
        } catch (error) {
            console.error('Failed to load test suites:', error);
            container.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test suites</p>';
        }
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

        try {
            const executionsResponse = await this.fetchWithErrorHandling('/test-executions');
            const executions = this.normalizeCollection(executionsResponse, 'executions');
            this.currentData.executions = executions;

            if (executions && executions.length > 0) {
                // Transform API data to match our display format
                const formattedExecutions = executions.map(exec => ({
                    id: exec.id,
                    suiteName: this.lookupSuiteName(exec.suite_id) || exec.suite_name || 'Unknown Suite',
                    status: exec.status,
                    duration: this.calculateDuration(exec.start_time, exec.end_time),
                    passed: exec.passed_tests || 0,
                    failed: exec.failed_tests || 0,
                    timestamp: exec.start_time
                }));
                
                container.innerHTML = this.renderExecutionsTable(formattedExecutions);
            } else {
                container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
            }
        } catch (error) {
            console.error('Failed to load test executions:', error);
            container.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test executions</p>';
        }
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

        try {
            const requestData = {
                scenario_name: scenarioName,
                vault_name: `${scenarioName}-vault-${Date.now()}`,
                phases: phases,
                phase_configurations: phases.reduce((config, phase) => {
                    config[phase] = {
                        timeout: timeout,
                        retry_count: 3,
                        parallel_execution: false
                    };
                    return config;
                }, {}),
                success_criteria: {
                    minimum_pass_rate: 90,
                    maximum_duration: timeout * phases.length,
                    required_phases: phases
                },
                total_timeout: timeout * phases.length
            };

            const response = await fetch(`${this.apiBaseUrl}/test-vault/create`, {
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
            
            resultDiv.innerHTML = `🏛️ Test Vault Created Successfully!

Vault ID: ${result.vault_id}
Scenario: ${scenarioName}
Phases: ${phases.join(', ')}
Phase Timeout: ${timeout}s
Total Estimated Time: ${timeout * phases.length}s

Vault Configuration:
  Total Timeout: ${result.total_timeout}s
  Success Criteria: ${result.success_criteria?.minimum_pass_rate || 90}% pass rate
  Phases: ${result.phases?.length || phases.length} configured

To execute the vault:
  POST /api/v1/test-vault/${result.vault_id}/execute`;

            resultDiv.style.display = 'block';
            this.showSuccess(`Test vault created successfully with ${phases.length} phases!`);
        } catch (error) {
            console.error('Test vault creation failed:', error);
            this.showError(`Test vault creation failed: ${error.message}`);
        }
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

    // Utility methods for API calls
    async fetchWithErrorHandling(endpoint) {
        try {
            const response = await fetch(`${this.apiBaseUrl}${endpoint}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`API call failed for ${endpoint}:`, error);
            return null;
        }
    }

    normalizeCollection(data, primaryKey) {
        if (!data) return [];
        if (Array.isArray(data)) {
            return data;
        }
        if (primaryKey && Array.isArray(data[primaryKey])) {
            return data[primaryKey];
        }
        if (Array.isArray(data.items)) {
            return data.items;
        }
        if (Array.isArray(data.data)) {
            return data.data;
        }
        return [];
    }

    lookupSuiteName(suiteId) {
        if (!suiteId || !this.currentData.suites) {
            return null;
        }
        const match = this.currentData.suites.find(suite => suite.id === suiteId);
        return match ? match.scenario_name || match.name : null;
    }

    calculateDuration(startTime, endTime) {
        if (!startTime) return 0;
        if (!endTime) {
            // For running executions, calculate current duration
            const start = new Date(startTime);
            const now = new Date();
            return Math.round((now - start) / 1000 * 10) / 10; // Round to 1 decimal
        }
        const start = new Date(startTime);
        const end = new Date(endTime);
        return Math.round((end - start) / 1000 * 10) / 10; // Round to 1 decimal
    }

    // WebSocket connection for real-time updates
    initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        try {
            this.wsConnection = new WebSocket(wsUrl);
            
            this.wsConnection.onopen = () => {
                console.log('WebSocket connection established');
                
                // Subscribe to relevant topics
                this.wsConnection.send(JSON.stringify({
                    type: 'subscribe',
                    payload: {
                        topics: ['system_status', 'executions', 'test_completion']
                    }
                }));
            };
            
            this.wsConnection.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleWebSocketMessage(data);
            };
            
            this.wsConnection.onclose = () => {
                console.log('WebSocket connection closed');
                // Reconnect after 5 seconds
                setTimeout(() => this.initWebSocket(), 5000);
            };
            
            this.wsConnection.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
        } catch (error) {
            console.error('Failed to initialize WebSocket:', error);
        }
    }

    handleWebSocketMessage(data) {
        switch (data.type) {
            case 'execution_update':
                this.handleExecutionUpdate(data.payload);
                break;
            case 'test_completion':
                this.handleTestCompletion(data.payload);
                break;
            case 'system_status':
                this.updateSystemStatus(data.payload.healthy);
                break;
            default:
                console.log('Unknown WebSocket message type:', data.type);
        }
    }

    handleExecutionUpdate(payload) {
        // Update execution status in real-time
        if (this.activePage === 'executions' || this.activePage === 'dashboard') {
            this.loadPageData(this.activePage);
        }
        
        // Show notification for execution status changes
        if (payload.status === 'completed') {
            this.showSuccess(`Test execution ${payload.execution_id} completed successfully`);
        } else if (payload.status === 'failed') {
            this.showError(`Test execution ${payload.execution_id} failed`);
        }
    }

    handleTestCompletion(payload) {
        // Refresh dashboard metrics
        if (this.activePage === 'dashboard') {
            this.loadDashboardData();
        }
        
        this.showInfo(`Test ${payload.test_name} completed with status: ${payload.status}`);
    }

    // Enhanced periodic updates with better error handling
    startPeriodicUpdates() {
        // Update system status every 30 seconds
        setInterval(async () => {
            await this.checkApiHealth();
        }, 30000);

        // Update current page data every 60 seconds
        this.refreshInterval = setInterval(async () => {
            if (this.activePage === 'dashboard') {
                await this.loadDashboardData();
            } else if (this.activePage === 'executions') {
                await this.loadExecutions();
            } else if (this.activePage === 'suites') {
                await this.loadTestSuites();
            }
        }, 60000);

        // Initialize WebSocket for real-time updates
        this.initWebSocket();
    }

    // Clean up resources
    destroy() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
        if (this.wsConnection) {
            this.wsConnection.close();
        }
    }
}

// Initialize the app
const app = new TestGenieApp();

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    app.destroy();
});
