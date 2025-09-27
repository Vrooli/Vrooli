// Test Genie Dashboard JavaScript
const MODEL_STORAGE_KEY = 'testGenie.selectedModel';
const AGENT_POLL_INTERVAL = 8000;

class TestGenieApp {
    constructor() {
        this.apiBaseUrl = '/api/v1';
        this.activePage = 'dashboard';
        this.currentData = {
            suites: [],
            executions: [],
            metrics: {}
        };
        this.suiteDetailsCache = new Map();
        this.refreshInterval = null;
        this.wsConnection = null;
        this.availableModels = [];
        this.defaultModel = 'openrouter/x-ai/grok-code-fast-1';
        this.selectedModel = null;
        this.agentInterval = null;
        this.agentMenuOpen = false;
        this.mobileBreakpoint = 768;
        this.sidebarToggleButton = null;
        this.sidebarOverlay = null;
        this.sidebarCloseButton = null;
        this.sidebarElement = null;
        this.suiteDetailOverlay = null;
        this.suiteDetailContent = null;
        this.suiteDetailCloseButton = null;
        this.suiteDetailExecuteButton = null;
        this.suiteDetailTitle = null;
        this.suiteDetailId = null;
        this.suiteDetailSubtitle = null;
        this.activeSuiteDetailId = null;
        this.lastSuiteDetailTrigger = null;

        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.setupNavigation();
        this.setupSuiteDetailPanel();
        await this.loadModels();
        await this.loadInitialData();
        this.startPeriodicUpdates();
        this.startAgentPolling();
    }

    setupEventListeners() {
        // Navigation
        document.addEventListener('click', (e) => {
            const navItem = e.target.closest('.nav-item');
            if (navItem) {
                const page = navItem.getAttribute('data-page');
                if (page) {
                    this.navigateTo(page);
                    if (window.innerWidth <= this.mobileBreakpoint) {
                        this.closeMobileSidebar();
                    }
                }
                this.hideAgentDropdown();
                return;
            }

            if (!e.target.closest('.agent-status')) {
                this.hideAgentDropdown();
            }
        });

        this.sidebarToggleButton = document.getElementById('sidebar-toggle');
        this.sidebarOverlay = document.getElementById('sidebar-overlay');
        this.sidebarCloseButton = document.getElementById('sidebar-close');
        this.sidebarElement = document.getElementById('sidebar');

        if (this.sidebarToggleButton) {
            this.sidebarToggleButton.addEventListener('click', (e) => {
                e.preventDefault();
                this.toggleSidebar();
            });
        }

        if (this.sidebarOverlay) {
            this.sidebarOverlay.addEventListener('click', () => {
                this.closeMobileSidebar();
            });
        }

        if (this.sidebarCloseButton) {
            this.sidebarCloseButton.addEventListener('click', (e) => {
                e.preventDefault();
                this.closeMobileSidebar();
            });
        }

        const agentToggle = document.getElementById('agent-toggle');
        if (agentToggle) {
            agentToggle.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.toggleAgentDropdown();
            });
        }

        const agentList = document.getElementById('agent-list');
        if (agentList) {
            agentList.addEventListener('click', (e) => {
                const stopButton = e.target.closest('.agent-stop-btn');
                if (stopButton) {
                    const agentId = stopButton.dataset.agentId;
                    if (agentId) {
                        this.stopAgent(agentId, stopButton);
                    }
                }
            });
        }

        const modelSelect = document.getElementById('model-select');
        if (modelSelect) {
            modelSelect.addEventListener('change', (event) => {
                this.updateSelectedModel(event.target.value);
            });
        }

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
            if (e.key === 'Escape') {
                if (this.isSuiteDetailOpen()) {
                    e.preventDefault();
                    this.closeSuiteDetail();
                    return;
                }

                if (window.innerWidth <= this.mobileBreakpoint) {
                    this.closeMobileSidebar();
                }
            }

            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
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

        window.addEventListener('resize', () => this.handleResize());

        this.handleResize();
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

    setupSuiteDetailPanel() {
        this.suiteDetailOverlay = document.getElementById('suite-detail-overlay');
        if (!this.suiteDetailOverlay) {
            return;
        }

        this.suiteDetailContent = document.getElementById('suite-detail-content');
        this.suiteDetailCloseButton = document.getElementById('suite-detail-close');
        this.suiteDetailExecuteButton = document.getElementById('suite-detail-execute');
        this.suiteDetailTitle = document.getElementById('suite-detail-title');
        this.suiteDetailId = document.getElementById('suite-detail-id');
        this.suiteDetailSubtitle = document.getElementById('suite-detail-subtitle');

        this.suiteDetailOverlay.addEventListener('click', (event) => {
            if (event.target === this.suiteDetailOverlay) {
                this.closeSuiteDetail();
            }
        });

        if (this.suiteDetailCloseButton) {
            this.suiteDetailCloseButton.addEventListener('click', () => this.closeSuiteDetail());
        }

        if (this.suiteDetailExecuteButton) {
            this.suiteDetailExecuteButton.addEventListener('click', () => {
                if (this.activeSuiteDetailId) {
                    this.executeSuite(this.activeSuiteDetailId);
                }
            });
        }
    }

    toggleSidebar() {
        const isMobile = window.innerWidth <= this.mobileBreakpoint;

        if (isMobile) {
            if (document.body.classList.contains('sidebar-open')) {
                this.closeMobileSidebar();
            } else {
                document.body.classList.add('sidebar-open');
                document.body.classList.remove('sidebar-collapsed');
                this.updateSidebarAccessibility(true);
                this.focusElement(this.sidebarCloseButton);
            }
            return;
        }

        const isCollapsed = document.body.classList.toggle('sidebar-collapsed');
        this.updateSidebarAccessibility(!isCollapsed);
    }

    closeMobileSidebar() {
        if (!document.body.classList.contains('sidebar-open')) {
            return;
        }

        document.body.classList.remove('sidebar-open');
        this.updateSidebarAccessibility(false);
        this.focusElement(this.sidebarToggleButton);
    }

    handleResize() {
        const isMobile = window.innerWidth <= this.mobileBreakpoint;

        if (isMobile) {
            document.body.classList.remove('sidebar-collapsed');
            const isExpanded = document.body.classList.contains('sidebar-open');
            this.updateSidebarAccessibility(isExpanded);
        } else {
            document.body.classList.remove('sidebar-open');
            const isExpanded = !document.body.classList.contains('sidebar-collapsed');
            this.updateSidebarAccessibility(isExpanded);
        }
    }

    updateSidebarAccessibility(isExpanded) {
        if (this.sidebarToggleButton) {
            this.sidebarToggleButton.setAttribute('aria-expanded', isExpanded ? 'true' : 'false');
            this.sidebarToggleButton.setAttribute('aria-label', isExpanded ? 'Hide navigation' : 'Show navigation');
        }

        if (this.sidebarElement) {
            this.sidebarElement.setAttribute('aria-hidden', isExpanded ? 'false' : 'true');
        }
    }

    focusElement(element) {
        if (!element || typeof element.focus !== 'function') {
            return;
        }

        try {
            element.focus({ preventScroll: true });
        } catch (error) {
            element.focus();
        }
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

        if (page !== 'suites') {
            this.closeSuiteDetail();
        }

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
                this.bindExecutionsTableActions(container);
                this.refreshIcons();
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
            <table class="table executions-table">
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
                            <td class="cell-scenario"><strong>${exec.suiteName}</strong></td>
                            <td class="cell-status"><span class="status ${this.getStatusClass(exec.status)}">${exec.status}</span></td>
                            <td>${exec.duration}s</td>
                            <td class="text-success">${exec.passed}</td>
                            <td class="${exec.failed > 0 ? 'text-error' : 'text-muted'}">${exec.failed}</td>
                            <td class="cell-created">${this.formatTimestamp(exec.timestamp)}</td>
                            <td class="cell-actions">
                                <button class="btn icon-btn" type="button" data-action="view-execution" data-execution-id="${exec.id}" aria-label="View execution details">
                                    <i data-lucide="bar-chart-3"></i>
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    }

    getStatusClass(status) {
        const normalized = (status || '').toString().toLowerCase();

        if (['completed', 'active', 'success', 'ready', 'passed'].includes(normalized)) {
            return 'success';
        }

        if (['running', 'in_progress', 'queued', 'pending', 'scheduled'].includes(normalized)) {
            return 'info';
        }

        if (['failed', 'error', 'blocked', 'cancelled', 'aborted'].includes(normalized)) {
            return 'error';
        }

        return 'warning';
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
                this.bindSuitesTableActions(container);
                this.refreshIcons();
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
            <table class="table suites-table">
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
                    ${suites.map(suite => {
                        const rawCoverage = Number(suite.coverage);
                        const coverageValue = Math.max(0, Math.min(100, Number.isFinite(rawCoverage) ? Math.round(rawCoverage) : 0));
                        return `
                        <tr>
                            <td class="cell-scenario"><strong>${suite.scenarioName}</strong></td>
                            <td class="cell-types">${suite.suiteType || '—'}</td>
                            <td class="cell-count">${suite.testsCount}</td>
                            <td class="cell-coverage">
                                <div class="coverage-meter">
                                    <div class="progress">
                                        <div class="progress-bar" style="width: ${coverageValue}%"></div>
                                    </div>
                                    <span>${coverageValue}%</span>
                                </div>
                            </td>
                            <td class="cell-status"><span class="status ${this.getStatusClass(suite.status)}">${suite.status}</span></td>
                            <td class="cell-created">${this.formatTimestamp(suite.createdAt)}</td>
                            <td class="cell-actions">
                                <button class="btn icon-btn" type="button" data-action="execute" data-suite-id="${suite.id}" aria-label="Execute test suite">
                                    <i data-lucide="play"></i>
                                </button>
                                <button class="btn icon-btn secondary" type="button" data-action="view" data-suite-id="${suite.id}" aria-label="View test suite">
                                    <i data-lucide="eye"></i>
                                </button>
                            </td>
                        </tr>
                        `;
                    }).join('')}
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
                this.bindExecutionsTableActions(container);
                this.refreshIcons();
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
                model: this.getSelectedModel(),
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

    viewSuite(suiteId, triggerElement = null) {
        this.openSuiteDetail(suiteId, triggerElement);
    }

    async openSuiteDetail(suiteId, triggerElement = null) {
        if (!this.suiteDetailOverlay) {
            this.showInfo(`Test suite ${suiteId}`);
            return;
        }

        this.activeSuiteDetailId = suiteId;
        this.lastSuiteDetailTrigger = triggerElement;

        document.body.classList.add('suite-detail-open');
        this.suiteDetailOverlay.classList.add('active');
        this.suiteDetailOverlay.setAttribute('aria-hidden', 'false');

        if (this.suiteDetailContent) {
            this.suiteDetailContent.scrollTop = 0;
        }

        if (this.suiteDetailExecuteButton) {
            this.suiteDetailExecuteButton.disabled = true;
            this.suiteDetailExecuteButton.dataset.suiteId = suiteId;
        }

        this.focusElement(this.suiteDetailCloseButton);

        let suite = this.suiteDetailsCache.get(suiteId);
        if (!suite && this.suiteDetailContent) {
            this.suiteDetailContent.innerHTML = '<div class="loading"><div class="spinner"></div>Loading suite details...</div>';
        }

        if (!suite) {
            suite = await this.fetchWithErrorHandling(`/test-suite/${suiteId}`);
            if (suite) {
                this.suiteDetailsCache.set(suiteId, suite);
            }
        }

        if (suite) {
            this.renderSuiteDetail(suite);
            if (this.suiteDetailExecuteButton) {
                this.suiteDetailExecuteButton.disabled = false;
            }
            this.refreshIcons();
        } else if (this.suiteDetailContent) {
            this.suiteDetailContent.innerHTML = '<div class="suite-detail-empty" role="alert">Failed to load suite details. Please try again.</div>';
        }
    }

    closeSuiteDetail() {
        if (!this.suiteDetailOverlay) {
            return;
        }

        this.suiteDetailOverlay.classList.remove('active');
        this.suiteDetailOverlay.setAttribute('aria-hidden', 'true');
        document.body.classList.remove('suite-detail-open');
        this.activeSuiteDetailId = null;

        if (this.suiteDetailExecuteButton) {
            this.suiteDetailExecuteButton.disabled = false;
            delete this.suiteDetailExecuteButton.dataset.suiteId;
        }

        if (this.lastSuiteDetailTrigger) {
            this.focusElement(this.lastSuiteDetailTrigger);
            this.lastSuiteDetailTrigger = null;
        }
    }

    isSuiteDetailOpen() {
        return this.suiteDetailOverlay ? this.suiteDetailOverlay.classList.contains('active') : false;
    }

    renderSuiteDetail(suite) {
        if (!this.suiteDetailContent) {
            return;
        }

        const testCases = Array.isArray(suite.test_cases) ? suite.test_cases : [];
        const suiteName = suite.scenario_name || suite.name || 'Test Suite';
        const rawSuiteId = suite.id || this.activeSuiteDetailId || '';
        const suiteId = rawSuiteId ? String(rawSuiteId) : '';
        const createdAt = suite.generated_at || suite.created_at;
        const lastExecutedAt = suite.last_executed || suite.lastExecuted;

        if (this.suiteDetailTitle) {
            this.suiteDetailTitle.textContent = suiteName;
        }

        if (this.suiteDetailId) {
            this.suiteDetailId.textContent = suiteId ? `Suite ID: ${suiteId}` : '';
        }

        if (this.suiteDetailSubtitle) {
            const createdText = createdAt ? `Generated ${this.formatDetailedTimestamp(createdAt)}` : 'Generated date unavailable';
            const executedText = lastExecutedAt ? `Last executed ${this.formatDetailedTimestamp(lastExecutedAt)}` : 'Not executed yet';
            this.suiteDetailSubtitle.textContent = `${createdText} • ${executedText}`;
        }

        if (this.suiteDetailExecuteButton) {
            this.suiteDetailExecuteButton.disabled = false;
            this.suiteDetailExecuteButton.dataset.suiteId = suiteId;
        }

        const sections = [
            this.buildSuiteSummary(suite, testCases),
            this.buildCoverageSection(suite),
            this.buildTestCasesSection(testCases)
        ];

        this.suiteDetailContent.innerHTML = sections.join('');
    }

    buildSuiteSummary(suite, testCases) {
        const totalTests = testCases.length;
        const statusLabel = this.formatLabel(suite.status || 'unknown');
        const statusClass = this.getStatusClass(suite.status);
        const lastExecutedAt = suite.last_executed || suite.lastExecuted;
        const lastExecutedSummary = lastExecutedAt ? `Ran ${this.formatRelativeTime(lastExecutedAt)}` : 'No executions yet';

        const suiteTypes = this.extractSuiteTypes(suite, testCases);
        const suiteTypeSummary = suiteTypes.length ? suiteTypes.map(type => this.formatLabel(type)).join(', ') : 'No types recorded';

        const priorityCounts = this.countBy(testCases.map(testCase => (testCase.priority || 'unspecified')));
        const prioritySummary = this.summarizeCounts(priorityCounts, 'priority');

        const timeouts = testCases
            .map(testCase => Number(testCase.execution_timeout))
            .filter(Number.isFinite);
        const averageTimeout = timeouts.length ? Math.round(timeouts.reduce((sum, value) => sum + value, 0) / timeouts.length) : null;

        const uniqueTags = new Set();
        const uniqueDependencies = new Set();
        testCases.forEach(testCase => {
            if (Array.isArray(testCase.tags)) {
                testCase.tags.filter(Boolean).forEach(tag => uniqueTags.add(tag));
            }
            if (Array.isArray(testCase.dependencies)) {
                testCase.dependencies.filter(Boolean).forEach(dep => uniqueDependencies.add(dep));
            }
        });

        const metadataPieces = [
            `Tags: ${uniqueTags.size}`,
            averageTimeout !== null ? `Avg timeout: ${averageTimeout}s` : null,
            `Dependencies: ${uniqueDependencies.size}`
        ].filter(Boolean);

        return `
            <div class="suite-detail-section">
                <h3>Suite Summary</h3>
                <div class="suite-detail-grid">
                    <div class="suite-detail-card">
                        <span class="label">Status</span>
                        <span class="value"><span class="status ${statusClass}">${this.escapeHtml(statusLabel)}</span></span>
                        <span class="value-small">${this.escapeHtml(lastExecutedSummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Test Cases</span>
                        <span class="value">${totalTests}</span>
                        <span class="value-small">${this.escapeHtml(prioritySummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Suite Types</span>
                        <span class="value-small">${this.escapeHtml(suiteTypeSummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Metadata</span>
                        <span class="value-small">${this.escapeHtml(metadataPieces.length ? metadataPieces.join(' • ') : 'No metadata captured')}</span>
                    </div>
                </div>
            </div>
        `;
    }

    buildCoverageSection(suite) {
        const metrics = suite.coverage_metrics || {};
        const entries = [
            { label: 'Code Coverage', value: metrics.code_coverage ?? suite.coverage },
            { label: 'Branch Coverage', value: metrics.branch_coverage },
            { label: 'Function Coverage', value: metrics.function_coverage }
        ].filter(entry => Number.isFinite(Number(entry.value)));

        if (entries.length === 0) {
            return `
                <div class="suite-detail-section">
                    <h3>Coverage Metrics</h3>
                    <div class="suite-detail-empty">No coverage metrics available for this suite.</div>
                </div>
            `;
        }

        const bars = entries.map(entry => {
            const percent = this.clampPercentage(Number(entry.value));
            return `
                <div class="coverage-item">
                    <div class="coverage-item-header">
                        <span>${this.escapeHtml(entry.label)}</span>
                        <span>${percent}%</span>
                    </div>
                    <div class="progress">
                        <div class="progress-bar" style="width: ${percent}%"></div>
                    </div>
                </div>
            `;
        }).join('');

        return `
            <div class="suite-detail-section">
                <h3>Coverage Metrics</h3>
                <div class="coverage-bars">
                    ${bars}
                </div>
            </div>
        `;
    }

    buildTestCasesSection(testCases) {
        if (!testCases.length) {
            return `
                <div class="suite-detail-section">
                    <h3>Test Cases</h3>
                    <div class="suite-detail-empty">No test cases have been generated for this suite yet.</div>
                </div>
            `;
        }

        const rows = testCases.map((testCase, index) => {
            const name = this.escapeHtml(testCase.name || `Test ${index + 1}`);
            const description = testCase.description ? this.escapeHtml(testCase.description) : '';
            const testType = this.escapeHtml(this.formatLabel(testCase.test_type || 'unspecified'));
            const priority = this.escapeHtml(this.formatLabel(testCase.priority || 'unspecified'));
            const timeout = Number(testCase.execution_timeout);
            const timeoutText = Number.isFinite(timeout) ? `${timeout}s` : '—';
            const expected = this.escapeHtml(testCase.expected_result || '—');

            const tags = Array.isArray(testCase.tags) && testCase.tags.length > 0
                ? testCase.tags.filter(Boolean).map(tag => `<span class="suite-detail-tag">${this.escapeHtml(tag)}</span>`).join('')
                : '<span class="suite-detail-subtitle">None</span>';

            const dependencies = Array.isArray(testCase.dependencies) && testCase.dependencies.length > 0
                ? testCase.dependencies.filter(Boolean).map(dep => `<span class="suite-detail-tag">${this.escapeHtml(dep)}</span>`).join('')
                : '<span class="suite-detail-subtitle">None</span>';

            const codeBlock = testCase.test_code
                ? `<details><summary>View Generated Test</summary><pre class="suite-detail-code">${this.escapeHtml(testCase.test_code)}</pre></details>`
                : '';

            return `
                <tr>
                    <td>
                        <div><strong>${name}</strong></div>
                        ${description ? `<div class="suite-detail-subtitle">${description}</div>` : ''}
                        ${codeBlock}
                    </td>
                    <td>
                        <div class="suite-detail-tags"><span class="suite-detail-tag">${testType}</span></div>
                        <div class="suite-detail-subtitle">Priority: ${priority}</div>
                        <div class="suite-detail-subtitle">Timeout: ${timeoutText}</div>
                    </td>
                    <td>
                        <div class="suite-detail-subtitle">Tags</div>
                        <div class="suite-detail-tags">${tags}</div>
                        <div class="suite-detail-subtitle" style="margin-top: 0.5rem;">Dependencies</div>
                        <div class="suite-detail-tags">${dependencies}</div>
                    </td>
                    <td>${expected}</td>
                </tr>
            `;
        }).join('');

        return `
            <div class="suite-detail-section">
                <h3>Test Cases</h3>
                <div class="suite-detail-tests">
                    <table>
                        <thead>
                            <tr>
                                <th style="width: 30%;">Test Case</th>
                                <th style="width: 20%;">Type & Priority</th>
                                <th style="width: 25%;">Tags & Dependencies</th>
                                <th style="width: 25%;">Expected Result</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${rows}
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }

    viewExecution(executionId) {
        this.showInfo(`Viewing execution ${executionId}. Detailed execution view coming soon!`);
    }

    bindSuitesTableActions(container) {
        const table = container.querySelector('.suites-table');
        if (!table || table.dataset.bound === 'true') {
            return;
        }

        table.addEventListener('click', event => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            const { action, suiteId } = button.dataset;

            if (action === 'execute' && suiteId) {
                this.executeSuite(suiteId);
            }

            if (action === 'view' && suiteId) {
                this.viewSuite(suiteId, button);
            }
        });

        table.dataset.bound = 'true';
    }

    bindExecutionsTableActions(container) {
        const table = container.querySelector('.executions-table');
        if (!table || table.dataset.bound === 'true') {
            return;
        }

        table.addEventListener('click', event => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            if (button.dataset.action === 'view-execution' && button.dataset.executionId) {
                this.viewExecution(button.dataset.executionId);
            }
        });

        table.dataset.bound = 'true';
    }

    refreshIcons() {
        if (typeof lucide !== 'undefined' && typeof lucide.createIcons === 'function') {
            lucide.createIcons();
        }
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

    // Model selection helpers
    async loadModels() {
        try {
            const response = await this.fetchWithErrorHandling('/models');
            if (response && Array.isArray(response.models)) {
                this.availableModels = response.models;
                if (response.default_model) {
                    this.defaultModel = response.default_model;
                }
            } else {
                this.availableModels = [];
            }
        } catch (error) {
            console.error('Failed to load model catalog:', error);
            this.availableModels = [];
        }

        const stored = localStorage.getItem(MODEL_STORAGE_KEY);
        if (stored && (this.availableModels.length === 0 || this.availableModels.some(model => model.id === stored))) {
            this.selectedModel = stored;
        } else if (this.availableModels.length > 0) {
            const defaultMatch = this.availableModels.find(model => model.id === this.defaultModel);
            this.selectedModel = defaultMatch ? defaultMatch.id : this.availableModels[0].id;
        } else {
            this.selectedModel = this.defaultModel;
        }

        if (this.selectedModel) {
            localStorage.setItem(MODEL_STORAGE_KEY, this.selectedModel);
        }

        this.updateModelSelector();
    }

    getSelectedModel() {
        return this.selectedModel || this.defaultModel;
    }

    updateSelectedModel(modelId) {
        if (!modelId) {
            return;
        }

        const previous = this.selectedModel;
        this.selectedModel = modelId;
        localStorage.setItem(MODEL_STORAGE_KEY, this.selectedModel);
        this.updateModelSelector();

        if (previous !== this.selectedModel) {
            this.showSuccess(`Default AI model set to ${this.selectedModel}`);
        }
    }

    updateModelSelector() {
        const select = document.getElementById('model-select');
        const helper = document.getElementById('model-select-helper');
        if (!select) {
            return;
        }

        select.innerHTML = '';
        const models = this.availableModels || [];
        const selected = this.getSelectedModel();

        if (models.length === 0) {
            const option = document.createElement('option');
            option.value = selected;
            option.textContent = selected;
            select.appendChild(option);
            select.disabled = true;
            if (helper) {
                helper.textContent = 'Using fallback model. Configure resource-opencode to unlock provider models.';
            }
            return;
        }

        models.forEach(model => {
            const option = document.createElement('option');
            option.value = model.id;
            option.textContent = this.formatModelLabel(model);
            select.appendChild(option);
        });

        if (!models.some(model => model.id === selected)) {
            const option = document.createElement('option');
            option.value = selected;
            option.textContent = selected;
            select.appendChild(option);
        }

        select.value = selected;
        select.disabled = false;

        if (helper) {
            const modelMeta = models.find(model => model.id === selected);
            helper.textContent = modelMeta
                ? `Selected model: ${modelMeta.id}`
                : `Selected model: ${selected}`;
        }
    }

    formatModelLabel(model) {
        if (!model) {
            return '';
        }
        const label = model.label && model.label !== model.id
            ? `${model.label} — ${model.id}`
            : model.id;
        const provider = model.provider ? `${model.provider} • ` : '';
        const suffix = model.free ? ' (free tier)' : '';
        return `${provider}${label}${suffix}`;
    }

    // Agent management helpers
    startAgentPolling() {
        const toggle = document.getElementById('agent-toggle');
        if (!toggle) {
            return;
        }

        if (this.agentInterval) {
            clearInterval(this.agentInterval);
        }

        this.refreshAgentStatus();
        this.agentInterval = setInterval(() => this.refreshAgentStatus(), AGENT_POLL_INTERVAL);
    }

    async refreshAgentStatus() {
        const data = await this.fetchWithErrorHandling('/agents');
        if (!data) {
            this.updateAgentIndicator({ count: 0, agents: [] });
            return;
        }
        this.updateAgentIndicator(data);
    }

    updateAgentIndicator(agentData) {
        const button = document.getElementById('agent-toggle');
        const countEl = document.getElementById('agent-count');
        const listEl = document.getElementById('agent-list');
        const dropdown = document.getElementById('agent-dropdown');

        if (!button || !countEl || !listEl || !dropdown) {
            return;
        }

        const count = agentData?.count || 0;
        const agents = Array.isArray(agentData?.agents) ? agentData.agents : [];

        button.classList.toggle('inactive', count === 0);
        button.classList.toggle('active', count > 0);
        countEl.textContent = count > 0
            ? `${count} Active Agent${count === 1 ? '' : 's'}`
            : 'No Active Agents';

        listEl.innerHTML = '';

        if (agents.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'agent-empty';
            empty.textContent = 'No agents currently running';
            listEl.appendChild(empty);
            this.hideAgentDropdown();
            return;
        }

        agents.forEach(agent => {
            const item = document.createElement('div');
            item.className = 'agent-item';

            const title = document.createElement('h4');
            title.textContent = agent.label || agent.name || agent.id;
            item.appendChild(title);

            const actionRow = document.createElement('div');
            actionRow.className = 'agent-meta';
            const actionSpan = document.createElement('span');
            actionSpan.textContent = this.describeAgentAction(agent.action);
            const durationSpan = document.createElement('span');
            durationSpan.textContent = this.formatAgentDuration(agent);
            actionRow.appendChild(actionSpan);
            actionRow.appendChild(durationSpan);
            item.appendChild(actionRow);

            const modelRow = document.createElement('div');
            modelRow.className = 'agent-meta';
            const modelSpan = document.createElement('span');
            modelSpan.textContent = agent.model || 'Model not specified';
            const statusSpan = document.createElement('span');
            statusSpan.textContent = (agent.status || 'running').toUpperCase();
            modelRow.appendChild(modelSpan);
            modelRow.appendChild(statusSpan);
            item.appendChild(modelRow);

            const actions = document.createElement('div');
            actions.className = 'agent-actions';
            const stopBtn = document.createElement('button');
            stopBtn.type = 'button';
            stopBtn.className = 'agent-stop-btn';
            stopBtn.dataset.agentId = agent.id;
            stopBtn.textContent = 'Stop Agent';
            actions.appendChild(stopBtn);
            item.appendChild(actions);

            listEl.appendChild(item);
        });
    }

    toggleAgentDropdown() {
        const dropdown = document.getElementById('agent-dropdown');
        if (!dropdown) {
            return;
        }
        const hasAgents = Boolean(document.querySelector('#agent-list .agent-item'));
        if (!hasAgents) {
            this.hideAgentDropdown();
            return;
        }
        this.agentMenuOpen = !this.agentMenuOpen;
        dropdown.classList.toggle('visible', this.agentMenuOpen);
    }

    hideAgentDropdown() {
        const dropdown = document.getElementById('agent-dropdown');
        if (!dropdown) {
            return;
        }
        if (this.agentMenuOpen) {
            this.agentMenuOpen = false;
            dropdown.classList.remove('visible');
        }
    }

    async stopAgent(agentId, button) {
        if (!agentId) {
            return;
        }

        if (button) {
            button.disabled = true;
            button.textContent = 'Stopping…';
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}/agents/${agentId}/stop`, {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            this.showSuccess('Agent stop requested');
        } catch (error) {
            console.error('Failed to stop agent', error);
            this.showError('Failed to stop agent');
        } finally {
            if (button) {
                button.disabled = false;
                button.textContent = 'Stop Agent';
            }
            this.refreshAgentStatus();
        }
    }

    describeAgentAction(action) {
        if (!action) {
            return 'Running task';
        }
        const map = {
            generate_unit_tests: 'Generating unit tests',
            generate_integration_tests: 'Generating integration tests',
            generate_performance_tests: 'Generating performance tests'
        };
        if (map[action]) {
            return map[action];
        }
        const formatted = action.replace(/_/g, ' ');
        return formatted.charAt(0).toUpperCase() + formatted.slice(1);
    }

    formatAgentDuration(agent) {
        if (!agent) {
            return '';
        }
        let seconds = agent.duration_seconds;
        if ((!seconds || seconds <= 0) && agent.started_at) {
            const start = new Date(agent.started_at);
            seconds = Math.max(0, Math.floor((Date.now() - start.getTime()) / 1000));
        }
        if (!seconds || seconds <= 0) {
            return '0s';
        }
        const minutes = Math.floor(seconds / 60);
        const secs = seconds % 60;
        if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        }
        return `${secs}s`;
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

    formatLabel(text) {
        if (!text) {
            return 'Unknown';
        }

        return String(text)
            .replace(/[_-]+/g, ' ')
            .split(' ')
            .filter(Boolean)
            .map(word => word.charAt(0).toUpperCase() + word.slice(1))
            .join(' ');
    }

    formatRelativeTime(timestamp) {
        if (!timestamp) {
            return '';
        }

        const reference = timestamp instanceof Date ? timestamp : new Date(timestamp);
        if (Number.isNaN(reference.getTime())) {
            return '';
        }

        const now = Date.now();
        const diffMs = reference.getTime() - now;
        const absDiff = Math.abs(diffMs);
        const minute = 60 * 1000;
        const hour = 60 * minute;
        const day = 24 * hour;
        const isFuture = diffMs > 0;

        if (absDiff < minute) {
            return isFuture ? 'in <1m' : 'just now';
        }
        if (absDiff < hour) {
            const mins = Math.round(absDiff / minute);
            return `${mins}m ${isFuture ? 'from now' : 'ago'}`;
        }
        if (absDiff < day) {
            const hours = Math.round(absDiff / hour);
            return `${hours}h ${isFuture ? 'from now' : 'ago'}`;
        }
        if (absDiff < day * 7) {
            const days = Math.round(absDiff / day);
            return `${days}d ${isFuture ? 'from now' : 'ago'}`;
        }

        return reference.toLocaleDateString();
    }

    formatDetailedTimestamp(timestamp) {
        if (!timestamp) {
            return 'unknown time';
        }

        const reference = timestamp instanceof Date ? timestamp : new Date(timestamp);
        if (Number.isNaN(reference.getTime())) {
            return 'unknown time';
        }

        const absolute = reference.toLocaleString(undefined, {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });

        const relative = this.formatRelativeTime(reference);
        return relative ? `${absolute} (${relative})` : absolute;
    }

    clampPercentage(value) {
        if (!Number.isFinite(value)) {
            return 0;
        }
        const bounded = Math.max(0, Math.min(100, value));
        return Math.round(bounded);
    }

    extractSuiteTypes(suite, testCases) {
        const types = new Set();

        if (Array.isArray(suite.test_types)) {
            suite.test_types.filter(Boolean).forEach(type => types.add(type));
        }

        const rawSuiteType = suite.suite_type || suite.suiteType;
        if (typeof rawSuiteType === 'string') {
            rawSuiteType.split(',').map(token => token.trim()).filter(Boolean).forEach(type => types.add(type));
        }

        testCases.forEach(testCase => {
            if (testCase && testCase.test_type) {
                types.add(testCase.test_type);
            }
        });

        return Array.from(types);
    }

    countBy(items) {
        const result = {};
        items.forEach(item => {
            const key = (item || 'unspecified').toString().toLowerCase();
            if (!result[key]) {
                result[key] = 0;
            }
            result[key] += 1;
        });
        return result;
    }

    summarizeCounts(counts, context = 'items') {
        const entries = Object.entries(counts || {}).filter(([, value]) => value > 0);

        if (entries.length === 0) {
            return `No ${context} data`;
        }

        entries.sort((a, b) => b[1] - a[1]);
        return entries.map(([key, value]) => `${this.formatLabel(key)} (${value})`).join(' • ');
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
