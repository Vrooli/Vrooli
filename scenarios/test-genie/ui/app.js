// Test Genie Dashboard JavaScript
const MODEL_STORAGE_KEY = 'testGenie.selectedModel';
const AGENT_POLL_INTERVAL = 8000;
const DEFAULT_VAULT_PHASES = ['setup', 'develop', 'test'];
const VAULT_PHASE_DEFINITIONS = {
    setup: {
        label: 'Setup',
        description: 'Provision dependencies, seed data, confirm configuration drift.',
        defaultTimeout: 600
    },
    develop: {
        label: 'Develop',
        description: 'Run developer-focused checks: lint, unit, integration, feature toggles.',
        defaultTimeout: 900
    },
    test: {
        label: 'Test',
        description: 'Execute regression and smoke suites that guard critical flows.',
        defaultTimeout: 1200
    },
    deploy: {
        label: 'Deploy',
        description: 'Validate rollout scripts, migrations, and packaging steps.',
        defaultTimeout: 900
    },
    monitor: {
        label: 'Monitor',
        description: 'Observe health metrics, load, and canary checks post-deploy.',
        defaultTimeout: 900
    }
};

class TestGenieApp {
    constructor() {
        this.apiBaseUrl = '/api/v1';
        this.activePage = 'dashboard';
        this.currentData = {
            suites: [],
            executions: [],
            metrics: {},
            coverage: [],
            scenarios: [],
            reports: {
                overview: null,
                trends: null,
                insights: null
            }
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
        this.selectedSuiteIds = new Set();
        this.selectedExecutionIds = new Set();
        this.runSelectedSuitesButton = null;
        this.runSelectedSuitesLabel = null;
        this.deleteSelectedExecutionsButton = null;
        this.suiteSelectAllCheckbox = null;
        this.executionSelectAllCheckbox = null;
        this.activeExecutionDetailId = null;
        this.lastExecutionDetailTrigger = null;
        this.executionDetailOverlay = null;
        this.executionDetailContent = null;
        this.executionDetailCloseButton = null;
        this.executionDetailTitle = null;
        this.executionDetailId = null;
        this.executionDetailSubtitle = null;
        this.executionDetailStatus = null;
        this.lastGenerateDialogTrigger = null;
        this.lastCoverageDialogTrigger = null;
        this.lastVaultDialogTrigger = null;
        this.coverageTableContainer = null;
        this.coverageDetailOverlay = null;
        this.coverageDetailContent = null;
        this.coverageDetailTitle = null;
        this.coverageDetailCloseButton = null;
        this.coverageRefreshButton = null;
        this.generateDialogOverlay = null;
        this.generateDialogCloseButton = null;
        this.vaultDialogOverlay = null;
        this.vaultDialogCloseButton = null;
        this.vaultInitialized = false;
        this.vaultSelectedPhases = new Set();
        this.vaultPhaseDrafts = new Map();
        this.activeVaultId = null;
        this.vaultScenarioOptionsLoaded = false;
        this.vaultIndexById = new Map();
        this.renderedSuiteIds = [];
        this.reportsWindowDays = 30;
        this.reportsTrendCanvas = null;
        this.reportsTrendCtx = null;
        this.reportsIsLoading = false;
        this.handleWindowResize = this.handleWindowResize.bind(this);

        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.setupNavigation();
        this.setupSuiteDetailPanel();
        this.setupExecutionDetailPanel();
        this.setupCoverageDetailDialog();
        this.setupGenerateDialog();
        this.setupVaultDialog();
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

        document.getElementById('vault-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleVaultSubmit();
        });

        this.runSelectedSuitesButton = document.getElementById('suites-run-selected-btn');
        this.runSelectedSuitesLabel = document.getElementById('suites-run-selected-label');
        if (this.runSelectedSuitesButton) {
            this.runSelectedSuitesButton.addEventListener('click', async (event) => {
                event.preventDefault();
                await this.runSelectedSuites();
            });
        }

        const suitesRefreshButton = document.getElementById('suites-refresh-btn');
        if (suitesRefreshButton) {
            suitesRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                const previousDisabled = suitesRefreshButton.disabled;
                suitesRefreshButton.disabled = true;
                try {
                    await this.loadTestSuites();
                    this.showSuccess('Test suites refreshed');
                } catch (error) {
                    console.error('Failed to refresh test suites:', error);
                    this.showError('Failed to refresh test suites');
                } finally {
                    suitesRefreshButton.disabled = previousDisabled;
                }
            });
        }

        const executionsRefreshButton = document.getElementById('executions-refresh-btn');
        if (executionsRefreshButton) {
            executionsRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                const previousDisabled = executionsRefreshButton.disabled;
                executionsRefreshButton.disabled = true;
                try {
                    await this.loadExecutions();
                    await this.loadRecentExecutions();
                    this.showSuccess('Test executions refreshed');
                } catch (error) {
                    console.error('Failed to refresh executions:', error);
                    this.showError('Failed to refresh executions');
                } finally {
                    executionsRefreshButton.disabled = previousDisabled;
                }
            });
        }

        const reportsRefreshButton = document.getElementById('reports-refresh');
        if (reportsRefreshButton) {
            reportsRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                if (this.reportsIsLoading) {
                    return;
                }
                await this.loadReports();
            });
        }

        const reportsWindowSelect = document.getElementById('reports-window-select');
        if (reportsWindowSelect) {
            reportsWindowSelect.addEventListener('change', async (event) => {
                const value = parseInt(event.target.value, 10);
                if (Number.isNaN(value)) {
                    return;
                }
                this.reportsWindowDays = value;
                if (this.reportsIsLoading) {
                    return;
                }
                await this.loadReports();
            });
        }

        window.addEventListener('resize', this.handleWindowResize);

        const executionsClearButton = document.getElementById('executions-clear-btn');
        if (executionsClearButton) {
            executionsClearButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.clearAllExecutions(executionsClearButton);
            });
        }

        this.deleteSelectedExecutionsButton = document.getElementById('executions-delete-selected-btn');
        if (this.deleteSelectedExecutionsButton) {
            this.deleteSelectedExecutionsButton.addEventListener('click', async (event) => {
                event.preventDefault();
                await this.deleteSelectedExecutions();
            });
        }

        this.coverageTableContainer = document.getElementById('coverage-table');
        this.coverageRefreshButton = document.getElementById('coverage-refresh-btn');
        if (this.coverageRefreshButton) {
            this.coverageRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                const previousDisabled = this.coverageRefreshButton.disabled;
                this.coverageRefreshButton.disabled = true;
                try {
                    await this.loadCoverageSummaries();
                    this.showSuccess('Coverage summaries refreshed');
                } catch (error) {
                    console.error('Failed to refresh coverage summaries:', error);
                    this.showError('Failed to refresh coverage summaries');
                } finally {
                    this.coverageRefreshButton.disabled = previousDisabled;
                }
            });
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (this.isSuiteDetailOpen()) {
                    e.preventDefault();
                    this.closeSuiteDetail();
                    return;
                }

                if (this.isExecutionDetailOpen()) {
                    e.preventDefault();
                    this.closeExecutionDetail();
                    return;
                }

                if (this.isOverlayActive(this.coverageDetailOverlay)) {
                    e.preventDefault();
                    this.closeCoverageDetailDialog();
                    return;
                }

                if (this.isOverlayActive(this.generateDialogOverlay)) {
                    e.preventDefault();
                    this.closeGenerateDialog();
                    return;
                }

                if (this.isOverlayActive(this.vaultDialogOverlay)) {
                    e.preventDefault();
                    this.closeVaultDialog();
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
                        this.navigateTo('suites');
                        break;
                    case '3':
                        e.preventDefault();
                        this.navigateTo('executions');
                        break;
                    case '4':
                        e.preventDefault();
                        this.navigateTo('coverage');
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

    setupExecutionDetailPanel() {
        this.executionDetailOverlay = document.getElementById('execution-detail-overlay');
        if (!this.executionDetailOverlay) {
            return;
        }

        this.executionDetailContent = document.getElementById('execution-detail-content');
        this.executionDetailCloseButton = document.getElementById('execution-detail-close');
        this.executionDetailTitle = document.getElementById('execution-detail-title');
        this.executionDetailId = document.getElementById('execution-detail-id');
        this.executionDetailSubtitle = document.getElementById('execution-detail-subtitle');
        this.executionDetailStatus = document.getElementById('execution-detail-status');

        this.executionDetailOverlay.addEventListener('click', (event) => {
            if (event.target === this.executionDetailOverlay) {
                this.closeExecutionDetail();
            }
        });

        if (this.executionDetailCloseButton) {
            this.executionDetailCloseButton.addEventListener('click', () => this.closeExecutionDetail());
        }
    }

    setupCoverageDetailDialog() {
        this.coverageDetailOverlay = document.getElementById('coverage-detail-overlay');
        if (!this.coverageDetailOverlay) {
            return;
        }

        this.coverageDetailContent = document.getElementById('coverage-detail-content');
        this.coverageDetailTitle = document.getElementById('coverage-detail-title');
        this.coverageDetailCloseButton = document.getElementById('coverage-detail-close');

        this.coverageDetailOverlay.addEventListener('click', (event) => {
            if (event.target === this.coverageDetailOverlay) {
                this.closeCoverageDetailDialog();
            }
        });

        if (this.coverageDetailCloseButton) {
            this.coverageDetailCloseButton.addEventListener('click', () => this.closeCoverageDetailDialog());
        }
    }

    setupGenerateDialog() {
        this.generateDialogOverlay = document.getElementById('generate-dialog-overlay');
        if (!this.generateDialogOverlay) {
            return;
        }

        this.generateDialogCloseButton = document.getElementById('generate-dialog-close');

        this.generateDialogOverlay.addEventListener('click', (event) => {
            if (event.target === this.generateDialogOverlay) {
                this.closeGenerateDialog();
            }
        });

        if (this.generateDialogCloseButton) {
            this.generateDialogCloseButton.addEventListener('click', () => this.closeGenerateDialog());
        }
    }

    setupVaultDialog() {
        this.vaultDialogOverlay = document.getElementById('vault-dialog-overlay');
        if (!this.vaultDialogOverlay) {
            return;
        }

        this.vaultDialogCloseButton = document.getElementById('vault-dialog-close');

        this.vaultDialogOverlay.addEventListener('click', (event) => {
            if (event.target === this.vaultDialogOverlay) {
                this.closeVaultDialog();
            }
        });

        if (this.vaultDialogCloseButton) {
            this.vaultDialogCloseButton.addEventListener('click', () => this.closeVaultDialog());
        }

        const openButton = document.getElementById('vault-dialog-open');
        if (openButton) {
            openButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.openVaultDialog(openButton);
            });
        }
    }

    openCoverageDetailDialog(triggerElement = null, scenarioName = '') {
        if (!this.coverageDetailOverlay) {
            return;
        }

        this.lastCoverageDialogTrigger = triggerElement || null;
        this.coverageDetailOverlay.classList.add('active');
        this.coverageDetailOverlay.setAttribute('aria-hidden', 'false');
        this.lockDialogScroll();

        if (this.coverageDetailTitle) {
            this.coverageDetailTitle.textContent = scenarioName || 'Coverage Analysis';
        }
    }

    closeCoverageDetailDialog() {
        if (!this.coverageDetailOverlay) {
            return;
        }

        this.coverageDetailOverlay.classList.remove('active');
        this.coverageDetailOverlay.setAttribute('aria-hidden', 'true');
        this.unlockDialogScrollIfIdle();

        if (this.lastCoverageDialogTrigger) {
            this.focusElement(this.lastCoverageDialogTrigger);
            this.lastCoverageDialogTrigger = null;
        }
    }

    openGenerateDialog(triggerElement = null, scenarioName = '') {
        if (!this.generateDialogOverlay) {
            return;
        }

        this.lastGenerateDialogTrigger = triggerElement || null;
        this.generateDialogOverlay.classList.add('active');
        this.generateDialogOverlay.setAttribute('aria-hidden', 'false');
        this.lockDialogScroll();

        const scenarioInput = document.getElementById('scenario-name');
        if (scenarioInput) {
            if (scenarioName) {
                scenarioInput.value = scenarioName;
            }
            this.focusElement(scenarioInput);
            scenarioInput.select?.();
        }

        const resultDiv = document.getElementById('generation-result');
        if (resultDiv) {
            resultDiv.style.display = 'none';
            resultDiv.textContent = '';
        }
    }

    closeGenerateDialog() {
        if (!this.generateDialogOverlay) {
            return;
        }

        this.generateDialogOverlay.classList.remove('active');
        this.generateDialogOverlay.setAttribute('aria-hidden', 'true');
        this.unlockDialogScrollIfIdle();

        if (this.lastGenerateDialogTrigger) {
            this.focusElement(this.lastGenerateDialogTrigger);
            this.lastGenerateDialogTrigger = null;
        }
    }

    openVaultDialog(triggerElement = null) {
        if (!this.vaultDialogOverlay) {
            return;
        }

        this.lastVaultDialogTrigger = triggerElement || null;
        this.vaultDialogOverlay.classList.add('active');
        this.vaultDialogOverlay.setAttribute('aria-hidden', 'false');
        this.lockDialogScroll();

        const scenarioInput = document.getElementById('vault-scenario');
        if (scenarioInput) {
            this.focusElement(scenarioInput);
        }

        if (this.vaultResultCard) {
            this.vaultResultCard.classList.remove('visible');
            this.vaultResultCard.innerHTML = '';
        }
    }

    closeVaultDialog() {
        if (!this.vaultDialogOverlay) {
            return;
        }

        this.vaultDialogOverlay.classList.remove('active');
        this.vaultDialogOverlay.setAttribute('aria-hidden', 'true');
        this.unlockDialogScrollIfIdle();

        if (this.lastVaultDialogTrigger) {
            this.focusElement(this.lastVaultDialogTrigger);
            this.lastVaultDialogTrigger = null;
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
        let resolvedPage = page;
        if (!document.getElementById(resolvedPage)) {
            resolvedPage = 'dashboard';
        }
		// Update active nav item
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`.nav-item[data-page="${resolvedPage}"]`)?.classList.add('active');

        // Update active page
        document.querySelectorAll('.page').forEach(p => {
            p.classList.remove('active');
        });
        document.getElementById(resolvedPage)?.classList.add('active');

        // Update URL
        if (pushState) {
            history.pushState({ page: resolvedPage }, '', `#${resolvedPage}`);
        }

        this.activePage = resolvedPage;

        if (resolvedPage !== 'suites') {
            this.closeSuiteDetail();
        }

        if (resolvedPage !== 'executions') {
            this.closeExecutionDetail();
        }

        // Load page-specific data
        this.loadPageData(resolvedPage);
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
                
                container.innerHTML = this.renderExecutionsTable(formattedExecutions, { selectable: false });
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

    renderExecutionsTable(executions, options = {}) {
        if (!executions || executions.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No recent executions found</p>';
        }

        const { selectable = true } = options;
        const selectionHeader = selectable
            ? '<th class="cell-select"><input type="checkbox" data-execution-select-all aria-label="Select all test executions"></th>'
            : '';

        return `
            <table class="table executions-table" data-selectable="${selectable}">
                <thead>
                    <tr>
                        ${selectionHeader}
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
                    ${executions.map(exec => {
                        const executionId = this.normalizeId(exec.id);
                        const isChecked = selectable && executionId && this.selectedExecutionIds.has(executionId);
                        const failedClass = exec.failed > 0 ? 'text-error' : 'text-muted';
                        const durationLabel = Number.isFinite(exec.duration) ? `${exec.duration}s` : '—';
                        const selectionCell = selectable
                            ? `<td class="cell-select"><input type="checkbox" data-execution-id="${executionId}" ${isChecked ? 'checked' : ''} aria-label="Select execution ${this.escapeHtml(exec.suiteName || executionId || '')}"></td>`
                            : '';
                        return `
                        <tr>
                            ${selectionCell}
                            <td class="cell-scenario"><strong>${exec.suiteName}</strong></td>
                            <td class="cell-status"><span class="status ${this.getStatusClass(exec.status)}">${exec.status}</span></td>
                            <td>${durationLabel}</td>
                            <td class="text-success">${exec.passed}</td>
                            <td class="${failedClass}">${exec.failed}</td>
                            <td class="cell-created">${this.formatTimestamp(exec.timestamp)}</td>
                            <td class="cell-actions">
                                <button class="btn icon-btn" type="button" data-action="view-execution" data-execution-id="${executionId}" aria-label="View execution details">
                                    <i data-lucide="bar-chart-3"></i>
                                </button>
                                <button class="btn icon-btn warning" type="button" data-action="delete-execution" data-execution-id="${executionId}" aria-label="Delete execution">
                                    <i data-lucide="trash-2"></i>
                                </button>
                            </td>
                        </tr>
                        `;
                    }).join('')}
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

    formatDateTime(value) {
        const date = this.parseDate(value);
        if (!date) {
            return '—';
        }
        return date.toLocaleString();
    }

    formatDateRange(startValue, endValue) {
        const start = this.parseDate(startValue);
        const end = this.parseDate(endValue);

        if (!start || !end) {
            return '';
        }

        const sameDay = start.toDateString() === end.toDateString();
        const options = { month: 'short', day: 'numeric' };
        const startLabel = start.toLocaleDateString(undefined, options);
        const endLabel = end.toLocaleDateString(undefined, options);
        const startYear = start.getFullYear();
        const endYear = end.getFullYear();

        if (sameDay) {
            return `${startLabel} ${startYear}`;
        }

        if (startYear === endYear) {
            return `${startLabel} → ${endLabel} ${endYear}`;
        }

        return `${startLabel} ${startYear} → ${endLabel} ${endYear}`;
    }

    formatDurationSeconds(totalSeconds) {
        if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) {
            return '—';
        }
        const seconds = Math.round(totalSeconds);
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const remainingSeconds = seconds % 60;
        const parts = [];
        if (hours) parts.push(`${hours}h`);
        if (minutes) parts.push(`${minutes}m`);
        if (!hours && (remainingSeconds || parts.length === 0)) {
            parts.push(`${remainingSeconds}s`);
        }
        return parts.join(' ');
    }

    formatPercent(value, digits = 1) {
        if (!Number.isFinite(value)) {
            return '0%';
        }
        return `${value.toFixed(digits)}%`;
    }

    formatPhaseLabel(phase) {
        if (!phase) {
            return 'Phase';
        }
        return phase
            .toString()
            .replace(/[_-]+/g, ' ')
            .split(' ')
            .filter(Boolean)
            .map(part => part.charAt(0).toUpperCase() + part.slice(1))
            .join(' ');
    }

    parseDate(value) {
        if (!value) {
            return null;
        }
        const date = new Date(value);
        return Number.isNaN(date.getTime()) ? null : date;
    }

    getStatusDescriptor(status) {
        const normalized = (status || '').toString().toLowerCase();

        if ([
            'completed',
            'active',
            'success',
            'ready',
            'passed',
            'done'
        ].includes(normalized)) {
            return { key: 'completed', label: 'Completed', icon: 'check' };
        }

        if ([
            'running',
            'in_progress',
            'active_run',
            'ongoing',
            'processing'
        ].includes(normalized)) {
            return { key: 'running', label: 'In Progress', icon: 'zap' };
        }

        if ([
            'failed',
            'error',
            'blocked',
            'cancelled',
            'aborted'
        ].includes(normalized)) {
            return { key: 'failed', label: 'Failed', icon: 'x' };
        }

        return { key: 'pending', label: 'Pending', icon: 'clock' };
    }

    ensurePhaseOrder(configuredPhases = [], executionDetails = null, latestExecution = null) {
        const order = [];
        const seen = new Set();

        const pushPhase = (phase) => {
            if (!phase) return;
            const phaseName = String(phase);
            const key = phaseName.toLowerCase();
            if (!seen.has(key)) {
                seen.add(key);
                order.push(phaseName);
            }
        };

        configuredPhases.forEach(pushPhase);

        const phaseResults = executionDetails?.phase_results || {};
        Object.keys(phaseResults).forEach(pushPhase);

        (executionDetails?.completed_phases || latestExecution?.completed_phases || []).forEach(pushPhase);
        (executionDetails?.failed_phases || latestExecution?.failed_phases || []).forEach(pushPhase);
        pushPhase(executionDetails?.current_phase || latestExecution?.current_phase);

        return order;
    }

    setupVaultPage() {
        if (this.vaultInitialized) {
            return;
        }

        this.vaultForm = document.getElementById('vault-form');
        this.vaultPhaseConfigContainer = document.getElementById('vault-phase-configurations');
        this.vaultResultCard = document.getElementById('vault-result');
        this.vaultRefreshButton = document.getElementById('vault-refresh-btn');
        this.vaultListContainer = document.getElementById('vault-list');
        this.vaultTimelineContainer = document.getElementById('execution-timeline');
        this.vaultTimelineContent = document.getElementById('timeline-phases');
        this.vaultHistoryContainer = document.getElementById('vault-history');
        this.vaultHistoryContent = document.getElementById('vault-history-content');
        this.vaultPhaseCheckboxes = Array.from(document.querySelectorAll('#vault-phase-selector [data-vault-phase]'));

        this.vaultSelectedPhases = new Set();
        this.vaultPhaseDrafts = new Map();

        this.vaultPhaseCheckboxes.forEach((checkbox) => {
            const phase = checkbox.dataset.vaultPhase;
            if (!phase) {
                return;
            }
            if (checkbox.checked) {
                this.vaultSelectedPhases.add(phase);
            }
            checkbox.addEventListener('change', () => {
                if (checkbox.checked) {
                    this.vaultSelectedPhases.add(phase);
                } else {
                    this.vaultSelectedPhases.delete(phase);
                }
                this.updateVaultPhaseConfigurations();
            });
        });

        if (this.vaultForm) {
            this.vaultForm.addEventListener('reset', () => {
                this.vaultPhaseDrafts.clear();
                window.setTimeout(() => this.resetVaultPhaseSelection(), 0);
            });
        }

        if (this.vaultRefreshButton) {
            this.vaultRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                await this.loadVaultList();
            });
        }

        this.resetVaultPhaseSelection();
        this.hideVaultDetails('Select a vault to inspect execution details.');

        this.vaultInitialized = true;
    }

    resetVaultPhaseSelection() {
        this.vaultSelectedPhases.clear();
        this.vaultPhaseDrafts.clear();

        if (Array.isArray(this.vaultPhaseCheckboxes)) {
            this.vaultPhaseCheckboxes.forEach((checkbox) => {
                const phase = checkbox.dataset.vaultPhase;
                if (!phase) {
                    return;
                }
                const shouldSelect = DEFAULT_VAULT_PHASES.includes(phase);
                checkbox.checked = shouldSelect;
                if (shouldSelect) {
                    this.vaultSelectedPhases.add(phase);
                }
            });
        }

        this.updateVaultPhaseConfigurations();
    }

    updateVaultPhaseConfigurations() {
        if (!this.vaultPhaseConfigContainer) {
            return;
        }

        // Capture current drafts before rebuilding
        this.vaultPhaseConfigContainer.querySelectorAll('.phase-config').forEach((config) => {
            const timeoutInput = config.querySelector('[data-vault-timeout]');
            if (!timeoutInput) {
                return;
            }
            const phase = timeoutInput.dataset.vaultTimeout;
            if (!phase) {
                return;
            }
            const notesField = config.querySelector('[data-vault-desc]');
            this.vaultPhaseDrafts.set(phase, {
                timeout: timeoutInput.value,
                description: notesField?.value || ''
            });
        });

        this.vaultPhaseConfigContainer.innerHTML = '';

        if (this.vaultSelectedPhases.size === 0) {
            const helper = document.createElement('div');
            helper.className = 'form-helper';
            helper.textContent = 'Select at least one phase above to configure timeouts and intent.';
            this.vaultPhaseConfigContainer.appendChild(helper);
            return;
        }

        const selected = Array.from(this.vaultSelectedPhases);
        const orderedPhases = [
            ...DEFAULT_VAULT_PHASES.filter((phase) => this.vaultSelectedPhases.has(phase)),
            ...selected.filter((phase) => !DEFAULT_VAULT_PHASES.includes(phase)).sort()
        ];

        orderedPhases.forEach((phase) => {
            const definition = VAULT_PHASE_DEFINITIONS[phase] || {
                label: this.formatPhaseLabel(phase),
                description: '',
                defaultTimeout: 600
            };
            const draft = this.vaultPhaseDrafts.get(phase) || {};
            const timeoutValue = Number(draft.timeout) || definition.defaultTimeout;
            const notesValue = draft.description || '';

            const config = document.createElement('div');
            config.className = 'phase-config';
            config.innerHTML = `
                <div class="phase-config-header">
                    <span class="phase-name">${this.escapeHtml(definition.label)}</span>
                    <div class="phase-timeout">
                        <label>Timeout:</label>
                        <input type="number" class="timeout-input" data-vault-timeout="${phase}" value="${timeoutValue}" min="30">
                        <span>seconds</span>
                    </div>
                </div>
                <textarea class="form-textarea" placeholder="${this.escapeHtml(definition.description)}" data-vault-desc="${phase}"></textarea>
            `;
            this.vaultPhaseConfigContainer.appendChild(config);

            const timeoutInput = config.querySelector('[data-vault-timeout]');
            const notesField = config.querySelector('[data-vault-desc]');

            if (timeoutInput) {
                timeoutInput.value = timeoutValue;
            }
            if (notesField && notesValue) {
                notesField.value = notesValue;
            }

            const persistDraft = () => {
                this.vaultPhaseDrafts.set(phase, {
                    timeout: timeoutInput?.value || definition.defaultTimeout,
                    description: notesField?.value || ''
                });
            };

            timeoutInput?.addEventListener('input', persistDraft);
            notesField?.addEventListener('input', persistDraft);
        });
    }

    async loadVaultScenarioOptions(forceRefresh = false) {
        const datalist = document.getElementById('vault-scenario-options');
        if (!datalist) {
            return;
        }

        if (this.vaultScenarioOptionsLoaded && !forceRefresh) {
            return;
        }

        const scenariosResponse = await this.fetchWithErrorHandling('/scenarios');
        if (!scenariosResponse) {
            return;
        }

        const scenarios = this.normalizeCollection(scenariosResponse, 'scenarios');
        const names = Array.from(new Set((scenarios || [])
            .map((scenario) => scenario?.name || scenario?.scenario_name)
            .filter(Boolean)))
            .sort((a, b) => a.localeCompare(b));

        if (names.length) {
            datalist.innerHTML = names.map((name) => `<option value="${this.escapeHtml(name)}"></option>`).join('');
            this.vaultScenarioOptionsLoaded = true;
        }
    }

    async loadVaultList() {
        if (!this.vaultListContainer) {
            return;
        }

        this.vaultListContainer.innerHTML = '<div class="loading-message">Loading vaults...</div>';

        try {
            const response = await this.fetchWithErrorHandling('/test-vaults');
            const vaults = this.normalizeCollection(response, 'vaults');
            this.currentData.vaults = Array.isArray(vaults) ? vaults : [];

            if (!this.currentData.vaults.length) {
                this.vaultListContainer.innerHTML = '<div class="empty-message">No vaults created yet</div>';
                this.vaultIndexById.clear();
                this.hideVaultDetails('Create a vault to populate execution data.');
                return;
            }

            this.renderVaultList(this.currentData.vaults);
        } catch (error) {
            console.error('Failed to load vaults:', error);
            this.vaultListContainer.innerHTML = '<div class="error-message">Failed to load vaults</div>';
            this.hideVaultDetails('Unable to load vault details.');
        }
    }

    renderVaultList(vaults) {
        if (!this.vaultListContainer) {
            return;
        }

        if (!Array.isArray(vaults) || vaults.length === 0) {
            this.vaultListContainer.innerHTML = '<div class="empty-message">No vaults created yet</div>';
            this.hideVaultDetails('Create a vault to populate execution data.');
            return;
        }

        const fragments = [];
        this.vaultIndexById = new Map();

        vaults.forEach((vault) => {
            const id = this.normalizeId(vault.id);
            this.vaultIndexById.set(id, vault);
            const descriptor = this.getStatusDescriptor(vault.status);
            const phaseCount = Array.isArray(vault.phases) ? vault.phases.length : 0;

            fragments.push(`
                <div class="vault-item" data-vault-id="${this.escapeHtml(id)}">
                    <div class="vault-item-header">
                        <span class="vault-item-name">${this.escapeHtml(vault.vault_name || id)}</span>
                        <span class="status-pill" data-status="${descriptor.key}"><i data-lucide="${descriptor.icon}"></i>${descriptor.label}</span>
                    </div>
                    <div class="vault-item-meta">
                        <span><i data-lucide="package"></i> ${this.escapeHtml(vault.scenario_name || 'Unknown')}</span>
                        <span><i data-lucide="layers"></i> ${phaseCount} phase${phaseCount === 1 ? '' : 's'}</span>
                        <span><i data-lucide="clock"></i> ${this.formatTimestamp(vault.created_at || vault.updated_at || Date.now())}</span>
                    </div>
                </div>
            `);
        });

        this.vaultListContainer.innerHTML = fragments.join('');
        this.refreshIcons();

        this.vaultListContainer.querySelectorAll('.vault-item').forEach((item) => {
            item.addEventListener('click', () => {
                const id = item.dataset.vaultId;
                this.setActiveVaultItem(id);
                this.showVaultExecution(id);
            });
        });

        let targetId = this.activeVaultId && this.vaultIndexById.has(this.activeVaultId)
            ? this.activeVaultId
            : null;
        if (!targetId && vaults.length > 0) {
            targetId = this.normalizeId(vaults[0].id);
        }

        if (targetId) {
            this.setActiveVaultItem(targetId);
            this.showVaultExecution(targetId);
        } else {
            this.hideVaultDetails('Select a vault to inspect execution details.');
        }
    }

    setActiveVaultItem(vaultId) {
        this.activeVaultId = vaultId;
        if (!this.vaultListContainer) {
            return;
        }
        this.vaultListContainer.querySelectorAll('.vault-item').forEach((item) => {
            item.classList.toggle('is-active', item.dataset.vaultId === vaultId);
        });
    }

    hideVaultDetails(message = '') {
        if (this.vaultTimelineContainer && this.vaultTimelineContent) {
            if (message) {
                this.vaultTimelineContainer.style.display = 'block';
                this.vaultTimelineContent.innerHTML = `<div class="history-empty">${this.escapeHtml(message)}</div>`;
            } else {
                this.vaultTimelineContainer.style.display = 'none';
                this.vaultTimelineContent.innerHTML = '';
            }
        }

        if (this.vaultHistoryContainer && this.vaultHistoryContent) {
            if (message) {
                this.vaultHistoryContainer.style.display = 'block';
                this.vaultHistoryContent.innerHTML = `<div class="history-empty">${this.escapeHtml(message)}</div>`;
            } else {
                this.vaultHistoryContainer.style.display = 'none';
                this.vaultHistoryContent.innerHTML = '';
            }
        }
    }

    async showVaultExecution(vaultId) {
        if (!vaultId || !this.vaultTimelineContent || !this.vaultHistoryContent) {
            return;
        }

        this.vaultTimelineContainer.style.display = 'block';
        this.vaultHistoryContainer.style.display = 'block';
        this.vaultTimelineContent.innerHTML = '<div class="loading-message">Loading execution details...</div>';
        this.vaultHistoryContent.innerHTML = '<div class="loading-message">Loading vault history...</div>';

        try {
            const [vaultDetails, executionsResponse] = await Promise.all([
                this.fetchWithErrorHandling(`/test-vault/${vaultId}`),
                this.fetchWithErrorHandling(`/vault-executions?vault_id=${encodeURIComponent(vaultId)}`)
            ]);

            const vault = vaultDetails || this.vaultIndexById.get(vaultId) || null;
            const configuredPhases = Array.isArray(vault?.phases) ? vault.phases : [];

            const executionsRaw = this.normalizeCollection(executionsResponse, 'executions');
            let executions = Array.isArray(executionsRaw) ? executionsRaw : [];

            if (executions.length) {
                const normalizedId = vaultId.toLowerCase();
                const filtered = executions.filter((execution) => this.normalizeId(execution.vault_id).toLowerCase() === normalizedId);
                if (filtered.length) {
                    executions = filtered;
                }
            }

            if (!executions.length) {
                this.vaultTimelineContent.innerHTML = '<div class="history-empty">No vault executions yet. Trigger the vault to see progress.</div>';
                this.vaultHistoryContent.innerHTML = '<div class="history-empty">No execution history available.</div>';
                this.refreshIcons();
                return;
            }

            executions.sort((a, b) => {
                const aDate = this.parseDate(a.start_time) || new Date(0);
                const bDate = this.parseDate(b.start_time) || new Date(0);
                return bDate.getTime() - aDate.getTime();
            });

            const latestExecution = executions[0];
            let executionDetails = null;
            const latestExecutionId = this.normalizeId(latestExecution.id);
            if (latestExecutionId) {
                executionDetails = await this.fetchWithErrorHandling(`/vault-execution/${latestExecutionId}/results`);
            }

            this.renderVaultTimeline(configuredPhases, executionDetails, latestExecution);
            this.renderVaultHistory(executions);
        } catch (error) {
            console.error('Failed to load vault execution data:', error);
            this.vaultTimelineContent.innerHTML = '<div class="error-message">Failed to load execution details</div>';
            this.vaultHistoryContent.innerHTML = '<div class="error-message">Failed to load execution history</div>';
        } finally {
            this.refreshIcons();
        }
    }

    renderVaultTimeline(phases, executionDetails, latestExecution) {
        if (!this.vaultTimelineContent) {
            return;
        }

        const phaseResultsRaw = executionDetails?.phase_results || {};
        const phaseResults = {};
        Object.entries(phaseResultsRaw).forEach(([key, value]) => {
            phaseResults[key.toLowerCase()] = value;
        });

        const completedSet = new Set((executionDetails?.completed_phases || latestExecution?.completed_phases || []).map((phase) => phase.toLowerCase()));
        const failedSet = new Set((executionDetails?.failed_phases || latestExecution?.failed_phases || []).map((phase) => phase.toLowerCase()));
        const currentPhase = (executionDetails?.current_phase || latestExecution?.current_phase || '').toLowerCase();

        const orderedPhases = this.ensurePhaseOrder(phases, executionDetails, latestExecution);

        if (!orderedPhases.length) {
            this.vaultTimelineContent.innerHTML = '<div class="history-empty">No phase data available yet. Run the vault to populate results.</div>';
            return;
        }

        const timelineHtml = orderedPhases.map((phaseName) => {
            const lowerPhase = phaseName.toLowerCase();
            let descriptor = this.getStatusDescriptor(phaseResults[lowerPhase]?.status);

            if (failedSet.has(lowerPhase)) {
                descriptor = { key: 'failed', label: 'Failed', icon: 'x' };
            } else if (completedSet.has(lowerPhase)) {
                descriptor = { key: 'completed', label: 'Completed', icon: 'check' };
            } else if (currentPhase === lowerPhase) {
                descriptor = { key: 'running', label: 'In Progress', icon: 'zap' };
            }

            const result = phaseResults[lowerPhase];
            const detailParts = [];
            const duration = result?.duration;
            if (Number.isFinite(duration)) {
                detailParts.push(`Duration: ${this.formatDurationSeconds(duration)}`);
            }
            let testCount = result?.test_count;
            if (!Number.isFinite(testCount) && Array.isArray(result?.test_results)) {
                testCount = result.test_results.length;
            }
            if (Number.isFinite(testCount)) {
                detailParts.push(`${testCount} tests`);
            }
            if (!detailParts.length) {
                if (descriptor.key === 'pending') {
                    detailParts.push('Waiting to start');
                } else {
                    detailParts.push(descriptor.label);
                }
            }

            const iconHtml = descriptor.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';
            const label = this.formatPhaseLabel(phaseName);

            return `
                <div class="timeline-phase">
                    <div class="timeline-indicator ${descriptor.key}">${iconHtml}</div>
                    <div class="timeline-content">
                        <div class="timeline-phase-name">${this.escapeHtml(label)}</div>
                        <div class="timeline-phase-status">${detailParts.map((part) => this.escapeHtml(part)).join(' • ')}</div>
                    </div>
                </div>
            `;
        }).join('');

        this.vaultTimelineContent.innerHTML = timelineHtml;
    }

    renderVaultHistory(executions) {
        if (!this.vaultHistoryContent) {
            return;
        }

        if (!Array.isArray(executions) || executions.length === 0) {
            this.vaultHistoryContent.innerHTML = '<div class="history-empty">No execution history available.</div>';
            return;
        }

        const rows = executions.map((execution, index) => {
            const descriptor = this.getStatusDescriptor(execution.status);
            const statusHtml = `<span class="status-pill" data-status="${descriptor.key}"><i data-lucide="${descriptor.icon}"></i>${descriptor.label}</span>`;

            let durationSeconds = Number(execution.duration);
            if (!Number.isFinite(durationSeconds)) {
                const start = this.parseDate(execution.start_time);
                const end = this.parseDate(execution.end_time);
                if (start && end) {
                    durationSeconds = (end.getTime() - start.getTime()) / 1000;
                }
            }

            const completedCount = Array.isArray(execution.completed_phases) ? execution.completed_phases.length : 0;
            const failedCount = Array.isArray(execution.failed_phases) ? execution.failed_phases.length : 0;

            return `
                <tr ${index === 0 ? 'data-latest="true"' : ''}>
                    <td>${this.escapeHtml(this.formatDateTime(execution.start_time))}</td>
                    <td>${statusHtml}</td>
                    <td>${completedCount}</td>
                    <td>${failedCount}</td>
                    <td>${this.escapeHtml(this.formatDurationSeconds(durationSeconds))}</td>
                </tr>
            `;
        }).join('');

        this.vaultHistoryContent.innerHTML = `
            <table class="history-table">
                <thead>
                    <tr>
                        <th>Started</th>
                        <th>Status</th>
                        <th>Completed</th>
                        <th>Failed</th>
                        <th>Duration</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        `;

        this.refreshIcons();
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
            case 'coverage':
                await this.loadCoverageSummaries();
                break;
            case 'vault':
                this.setupVaultPage();
                await Promise.all([
                    this.loadVaultScenarioOptions(),
                    this.loadVaultList()
                ]);
                break;
            case 'reports':
                await this.loadReports();
                break;
        }
    }

    async loadTestSuites() {
        const container = document.getElementById('suites-table');
        container.innerHTML = '<div class="loading"><div class="spinner"></div>Loading test suites...</div>';

        try {
            const [scenariosResponse, suitesResponse] = await Promise.all([
                this.fetchWithErrorHandling('/scenarios'),
                this.fetchWithErrorHandling('/test-suites')
            ]);

            const scenarios = this.normalizeCollection(scenariosResponse, 'scenarios');
            const suites = this.normalizeCollection(suitesResponse, 'test_suites');

            this.currentData.scenarios = scenarios;
            this.currentData.suites = suites;

            const suitesByScenario = new Map();
            if (Array.isArray(suites)) {
                suites.forEach((suite) => {
                    const scenarioName = suite?.scenario_name || suite?.scenarioName;
                    if (!scenarioName) {
                        return;
                    }

                    const normalized = scenarioName.trim();
                    if (!suitesByScenario.has(normalized)) {
                        suitesByScenario.set(normalized, []);
                    }
                    suitesByScenario.get(normalized).push(suite);
                });
            }

            const seenIds = new Set();
            const scenarioRows = [];

            const buildRowFromScenario = (scenario, attachedSuites = []) => {
                const rawName = scenario?.name || scenario?.scenario_name;
                if (!rawName) {
                    return;
                }

                const scenarioName = String(rawName).trim();
                if (!scenarioName) {
                    return;
                }

                const suiteCount = Number(scenario?.suite_count ?? attachedSuites.length ?? 0) || 0;
                const suiteArray = attachedSuites.length ? attachedSuites : (suitesByScenario.get(scenarioName) || []);
                const testCaseCount = Number(scenario?.test_case_count ?? 0) || suiteArray.reduce((sum, suite) => {
                    if (Array.isArray(suite?.test_cases) && suite.test_cases.length) {
                        return sum + suite.test_cases.length;
                    }
                    const total = Number(suite?.total_tests);
                    if (Number.isFinite(total) && total > 0) {
                        return sum + total;
                    }
                    return sum;
                }, 0);

                const typeSet = new Set(Array.isArray(scenario?.suite_types) ? scenario.suite_types : []);
                if (!typeSet.size && suiteArray.length) {
                    suiteArray.forEach((suite) => {
                        const rawType = Array.isArray(suite?.test_types)
                            ? suite.test_types
                            : (suite?.suite_type ? String(suite.suite_type).split(',') : []);
                        rawType.forEach((type) => {
                            const trimmed = String(type || '').trim().toLowerCase();
                            if (trimmed) {
                                typeSet.add(trimmed);
                            }
                        });
                    });
                }

                const latestSuiteId = scenario?.latest_suite_id
                    || suiteArray.reduce((latest, suite) => {
                        const id = this.normalizeId(suite?.id);
                        if (!id) {
                            return latest;
                        }
                        if (!latest) {
                            return id;
                        }

                        const currentTime = suite?.generated_at || suite?.created_at;
                        const latestSuite = suiteArray.find((item) => this.normalizeId(item?.id) === latest);
                        const latestTime = latestSuite?.generated_at || latestSuite?.created_at;

                        if (!latestTime) {
                            return id;
                        }

                        if (!currentTime) {
                            return latest;
                        }

                        return new Date(currentTime).getTime() > new Date(latestTime).getTime() ? id : latest;
                    }, '');

                const latestGeneratedAt = scenario?.latest_suite_generated_at
                    || (latestSuiteId
                        ? (suiteArray.find((suite) => this.normalizeId(suite?.id) === latestSuiteId)?.generated_at
                            || suiteArray.find((suite) => this.normalizeId(suite?.id) === latestSuiteId)?.created_at)
                        : null);

                const latestStatus = scenario?.latest_suite_status
                    || (suiteArray.find((suite) => this.normalizeId(suite?.id) === latestSuiteId)?.status)
                    || (suiteCount > 0 ? 'unknown' : 'missing');

                const latestCoverage = Number(scenario?.latest_suite_coverage ?? 0)
                    || (() => {
                        const suite = suiteArray.find((item) => this.normalizeId(item?.id) === latestSuiteId) || suiteArray[0];
                        if (!suite) {
                            return 0;
                        }
                        const coverage = Number(suite?.coverage_metrics?.code_coverage ?? suite?.coverage ?? 0);
                        return Number.isFinite(coverage) ? coverage : 0;
                    })();

                const phases = Array.from(typeSet).map((type) => this.formatLabel(type));

                const row = {
                    scenarioName,
                    phases,
                    testsCount: testCaseCount,
                    coverage: latestCoverage,
                    status: latestStatus,
                    createdAt: latestGeneratedAt || null,
                    latestSuiteId: this.normalizeId(latestSuiteId),
                    suiteCount,
                    hasTests: Boolean(scenario?.has_tests || suiteCount > 0),
                    hasTestDirectory: Boolean(scenario?.has_test_directory),
                    isMissing: !(scenario?.has_tests || suiteCount > 0),
                };

                scenarioRows.push(row);

                if (row.latestSuiteId) {
                    seenIds.add(row.latestSuiteId);
                }
            };

            if (Array.isArray(scenarios) && scenarios.length) {
                scenarios.forEach((scenario) => buildRowFromScenario(scenario));
            }

            suitesByScenario.forEach((suiteArray, scenarioName) => {
                const alreadyListed = Array.isArray(scenarios) && scenarios.some((scenario) => scenario?.name === scenarioName || scenario?.scenario_name === scenarioName);
                if (!alreadyListed) {
                const fallback = {
                    name: scenarioName,
                    has_tests: suiteArray.length > 0,
                    suite_count: suiteArray.length,
                    test_case_count: suiteArray.reduce((sum, suite) => {
                            if (Array.isArray(suite?.test_cases)) {
                                return sum + suite.test_cases.length;
                            }
                            const total = Number(suite?.total_tests);
                            if (Number.isFinite(total) && total > 0) {
                                return sum + total;
                            }
                            return sum;
                        }, 0),
                    };
                    buildRowFromScenario(fallback, suiteArray);
                }
            });

            const renderedIds = Array.from(seenIds);
            this.pruneSuiteSelection(renderedIds);
            this.renderedSuiteIds = renderedIds;

            if (scenarioRows.length > 0) {
                scenarioRows.sort((a, b) => a.scenarioName.localeCompare(b.scenarioName));
                container.innerHTML = this.renderSuitesTable(scenarioRows);
                this.bindSuitesTableActions(container);
                this.refreshIcons();
                this.updateSuiteSelectionUI();
            } else {
                container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No scenarios found</p>';
                this.selectedSuiteIds.clear();
                this.updateSuiteSelectionUI();
                this.renderedSuiteIds = [];
            }
        } catch (error) {
            console.error('Failed to load test suites:', error);
            container.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test suites</p>';
            this.renderedSuiteIds = [];
            this.updateSuiteSelectionUI();
        }
    }

    renderSuitesTable(scenarios) {
        if (!scenarios || scenarios.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No scenarios found</p>';
        }

        return `
            <table class="table suites-table" data-selectable="true">
                <thead>
                    <tr>
                        <th class="cell-select">
                            <input type="checkbox" data-suite-select-all aria-label="Select all suites">
                        </th>
                        <th>Scenario</th>
                        <th>Phases</th>
                        <th>Coverage</th>
                        <th>Status</th>
                        <th>Latest Suite</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${scenarios.map((scenario) => {
                        const suiteId = scenario.latestSuiteId ? this.normalizeId(scenario.latestSuiteId) : '';
                        const isChecked = suiteId && this.selectedSuiteIds.has(suiteId);
                        const rawCoverage = Number(scenario.coverage ?? 0);
                        const coverageValue = Math.max(0, Math.min(100, Number.isFinite(rawCoverage) ? Math.round(rawCoverage) : 0));
                        const hasSuite = Boolean(suiteId);
                        const isMissing = Boolean(scenario.isMissing);
                        const phases = Array.isArray(scenario.phases) && scenario.phases.length
                            ? scenario.phases.join(', ')
                            : (hasSuite ? '—' : 'None yet');
                        const statusRaw = isMissing ? 'missing' : (scenario.status || 'unknown');
                        const statusLabel = this.formatLabel(statusRaw);
                        const statusClass = this.getStatusClass(statusRaw);
                        const rowClass = isMissing ? 'scenario-row missing-scenario' : 'scenario-row has-suite';
                        const createdLabel = scenario.createdAt ? this.formatTimestamp(scenario.createdAt) : '—';

                        const coverageCell = hasSuite
                            ? `
                                <div class="coverage-meter">
                                    <div class="progress">
                                        <div class="progress-bar" style="width: ${coverageValue}%"></div>
                                    </div>
                                    <span>${coverageValue}%</span>
                                </div>
                              `
                            : '<span class="coverage-empty">—</span>';

        return `
                        <tr class="${rowClass}">
                            <td class="cell-select">
                                ${hasSuite ? `<input type="checkbox" data-suite-id="${suiteId}" ${isChecked ? 'checked' : ''} aria-label="Select suite for ${this.escapeHtml(scenario.scenarioName)}">` : ''}
                            </td>
                            <td class="cell-scenario">
                                <strong>${this.escapeHtml(scenario.scenarioName)}</strong>
                            </td>
                            <td class="cell-phases">${phases}</td>
                            <td class="cell-coverage">${coverageCell}</td>
                            <td class="cell-status"><span class="status ${statusClass}">${statusLabel}</span></td>
                            <td class="cell-created">${createdLabel}</td>
                            <td class="cell-actions">
                                ${hasSuite ? `
                                    <button class="btn icon-btn" type="button" data-action="execute" data-suite-id="${suiteId}" aria-label="Execute latest suite for ${this.escapeHtml(scenario.scenarioName)}">
                                        <i data-lucide="play"></i>
                                    </button>
                                    <button class="btn icon-btn secondary" type="button" data-action="view" data-suite-id="${suiteId}" aria-label="View latest suite for ${this.escapeHtml(scenario.scenarioName)}">
                                        <i data-lucide="eye"></i>
                                    </button>
                                ` : ''}
                                ${!hasSuite ? `
                                    <button class="btn icon-btn highlighted" type="button" data-action="generate" data-scenario="${this.escapeHtml(scenario.scenarioName)}" aria-label="Generate tests for ${this.escapeHtml(scenario.scenarioName)}">
                                        <i data-lucide="sparkles"></i>
                                    </button>
                                ` : ''}
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
            this.pruneExecutionSelection(executions);

            if (executions && executions.length > 0) {
                // Transform API data to match our display format
                const formattedExecutions = executions.map(exec => ({
                    id: this.normalizeId(exec.id),
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
                this.updateExecutionSelectionUI();
            } else {
                container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
                this.selectedExecutionIds.clear();
                this.updateExecutionSelectionUI();
            }
        } catch (error) {
            console.error('Failed to load test executions:', error);
            container.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test executions</p>';
            this.updateExecutionSelectionUI();
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

            await Promise.all([
                this.loadTestSuites(),
                this.loadVaultScenarioOptions(true)
            ]);

        } catch (error) {
            console.error('Test generation failed:', error);
            this.showError(`Test generation failed: ${error.message}`);
        } finally {
            btn.disabled = false;
            btn.innerHTML = '⚡ Generate Test Suite';
        }
    }

    async loadCoverageSummaries() {
        if (!this.coverageTableContainer) {
            return;
        }

        this.coverageTableContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading coverage summaries...</div>';

        try {
            const response = await this.fetchWithErrorHandling('/test-analysis/coverage');
            const coverages = this.normalizeCollection(response, 'coverages');
            this.currentData.coverage = coverages || [];

            if (!coverages || coverages.length === 0) {
                this.coverageTableContainer.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No coverage analyses found. Run a scenario\'s unit tests to generate coverage data.</p>';
                return;
            }

            this.coverageTableContainer.innerHTML = this.renderCoverageTable(coverages);
            this.bindCoverageTableActions();
        } catch (error) {
            console.error('Failed to load coverage summaries:', error);
            this.coverageTableContainer.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load coverage summaries.</p>';
        }
    }

    async loadReports() {
        const scenariosContainer = document.getElementById('reports-scenarios');
        if (!scenariosContainer) {
            return;
        }

        const windowSelect = document.getElementById('reports-window-select');
        if (windowSelect && Number(windowSelect.value) !== this.reportsWindowDays) {
            windowSelect.value = String(this.reportsWindowDays);
        }

        this.reportsIsLoading = true;
        this.showReportsLoading();

        const query = `?window_days=${encodeURIComponent(this.reportsWindowDays)}`;

        try {
            const [overview, trends, insights] = await Promise.all([
                this.fetchWithErrorHandling(`/reports/overview${query}`),
                this.fetchWithErrorHandling(`/reports/trends${query}`),
                this.fetchWithErrorHandling(`/reports/insights${query}`)
            ]);

            if (!overview || !trends || !insights) {
                throw new Error('reports endpoints returned empty responses');
            }

            this.currentData.reports = { overview, trends, insights };
            this.updateReportsWindowMeta(overview);
            this.renderReportsOverview(overview);
            this.renderReportsTrends(trends);
            this.renderReportsScenarioTable(overview);
            this.renderReportsInsights(insights);
            this.refreshIcons();
        } catch (error) {
            console.error('Failed to load reports:', error);
            this.renderReportsError('Failed to load analytics. Please try again.');
        } finally {
            this.reportsIsLoading = false;
        }
    }

    showReportsLoading() {
        const metricValueIds = ['report-quality-index', 'report-regressions', 'report-coverage', 'report-vault'];
        const metricSubtitleIds = ['report-quality-subtitle', 'report-regressions-subtitle', 'report-coverage-subtitle', 'report-vault-subtitle'];

        metricValueIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = '--';
            }
        });

        metricSubtitleIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = 'Loading…';
            }
        });

        const meta = document.getElementById('reports-window-meta');
        if (meta) {
            meta.textContent = '';
        }

        const containers = [
            document.getElementById('reports-trend-body'),
            document.getElementById('reports-scenarios'),
            document.getElementById('reports-insights')
        ];

        containers.forEach((container) => {
            if (!container) {
                return;
            }
            container.innerHTML = '<div class="loading"><div class="spinner"></div>Loading analytics…</div>';
        });
    }

    updateReportsWindowMeta(overview) {
        const meta = document.getElementById('reports-window-meta');
        if (!meta || !overview) {
            return;
        }

        const rangeLabel = this.formatDateRange(overview.window_start, overview.window_end);
        if (rangeLabel) {
            meta.textContent = `Window: ${rangeLabel}`;
        } else {
            meta.textContent = '';
        }
    }

    renderReportsOverview(overview) {
        if (!overview) {
            return;
        }

        const scenarios = Array.isArray(overview.scenarios) ? overview.scenarios : [];
        const global = overview.global || {};
        const vaults = overview.vaults || {};
        const scenarioCount = scenarios.length;
        const averageHealth = scenarioCount
            ? scenarios.reduce((sum, item) => sum + (Number(item.health_score) || 0), 0) / scenarioCount
            : 0;

        const qualityValue = document.getElementById('report-quality-index');
        if (qualityValue) {
            qualityValue.textContent = Math.round(averageHealth);
        }

        const qualitySubtitle = document.getElementById('report-quality-subtitle');
        if (qualitySubtitle) {
            qualitySubtitle.textContent = `${scenarioCount} scenario${scenarioCount === 1 ? '' : 's'} tracked`;
        }

        const regressionsValue = document.getElementById('report-regressions');
        if (regressionsValue) {
            regressionsValue.textContent = Number(global.active_regressions || 0);
        }

        const regressionsSubtitle = document.getElementById('report-regressions-subtitle');
        if (regressionsSubtitle) {
            const running = Number(global.active_executions || 0);
            const runningLabel = running > 0 ? `${running} running execution${running === 1 ? '' : 's'}` : 'No active runs';
            regressionsSubtitle.textContent = runningLabel;
        }

        const avgCoverage = Number(global.average_coverage || 0);
        const coverageTarget = 95;
        const coverageValue = document.getElementById('report-coverage');
        if (coverageValue) {
            coverageValue.textContent = this.formatPercent(avgCoverage, 1);
        }

        const coverageSubtitle = document.getElementById('report-coverage-subtitle');
        if (coverageSubtitle) {
            const delta = avgCoverage - coverageTarget;
            const deltaLabel = `${delta >= 0 ? '+' : '-'}${Math.abs(delta).toFixed(1)}% vs target`;
            coverageSubtitle.textContent = `Target ${coverageTarget}% (${deltaLabel})`;
        }

        const vaultValue = document.getElementById('report-vault');
        if (vaultValue) {
            const successRate = Number(vaults.success_rate || 0);
            vaultValue.textContent = this.formatPercent(successRate, 1);
        }

        const vaultSubtitle = document.getElementById('report-vault-subtitle');
        if (vaultSubtitle) {
            const totalExec = Number(vaults.total_executions || 0);
            const failedExec = Number(vaults.failed_executions || 0);
            vaultSubtitle.textContent = totalExec
                ? `${totalExec} execution${totalExec === 1 ? '' : 's'} • ${failedExec} failure${failedExec === 1 ? '' : 's'}`
                : 'No vault executions in window';
        }
    }

    renderReportsTrends(trends) {
        const container = document.getElementById('reports-trend-body');
        if (!container) {
            return;
        }

        const series = trends && Array.isArray(trends.series) ? trends.series : [];
        if (series.length === 0) {
            container.innerHTML = '<div class="reports-empty">No trend data available yet. Execute test suites to build history.</div>';
            this.reportsTrendCanvas = null;
            this.reportsTrendCtx = null;
            return;
        }

        container.innerHTML = `
            <canvas id="reports-trend-canvas" height="220"></canvas>
            <div class="chart-legend">
                <span class="legend-item"><span class="legend-swatch pass"></span>Pass rate</span>
                <span class="legend-item"><span class="legend-swatch fail"></span>Failed executions</span>
            </div>
        `;

        this.reportsTrendCanvas = document.getElementById('reports-trend-canvas');
        if (!this.reportsTrendCanvas) {
            return;
        }

        this.drawReportsTrendChart(series);
    }

    drawReportsTrendChart(series) {
        if (!this.reportsTrendCanvas) {
            return;
        }

        const devicePixelRatio = window.devicePixelRatio || 1;
        const displayWidth = this.reportsTrendCanvas.clientWidth || 600;
        const displayHeight = this.reportsTrendCanvas.getAttribute('height') ? Number(this.reportsTrendCanvas.getAttribute('height')) : 220;

        this.reportsTrendCanvas.width = Math.max(displayWidth * devicePixelRatio, 1);
        this.reportsTrendCanvas.height = Math.max(displayHeight * devicePixelRatio, 1);

        const ctx = this.reportsTrendCanvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.save();
        ctx.scale(devicePixelRatio, devicePixelRatio);

        const width = displayWidth;
        const height = displayHeight;
        ctx.clearRect(0, 0, width, height);

        const margin = { top: 20, right: 24, bottom: 32, left: 40 };
        const chartWidth = width - margin.left - margin.right;
        const chartHeight = height - margin.top - margin.bottom;

        if (chartWidth <= 0 || chartHeight <= 0) {
            ctx.restore();
            return;
        }

        const passRates = series.map((point) => {
            const passed = Number(point.passed_tests || 0);
            const failed = Number(point.failed_tests || 0);
            const total = passed + failed;
            if (total === 0) {
                return null;
            }
            return (passed / total) * 100;
        });

        const failureCounts = series.map((point) => Number(point.failed_executions || 0));
        const maxFailure = failureCounts.reduce((max, value) => Math.max(max, value), 0) || 1;

        const stepCount = series.length > 1 ? series.length - 1 : 1;
        const stepX = chartWidth / stepCount;
        const baseY = margin.top + chartHeight;
        const barMaxHeight = chartHeight * 0.35;
        const barWidth = Math.max(6, chartWidth / (series.length * 2));

        // Grid lines
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.08)';
        ctx.lineWidth = 1;
        [0, 25, 50, 75, 100].forEach((pct) => {
            const y = margin.top + chartHeight - (pct / 100) * chartHeight;
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();

            ctx.fillStyle = 'rgba(255, 255, 255, 0.35)';
            ctx.font = '10px Inter, sans-serif';
            ctx.fillText(`${pct}%`, 4, y + 3);
        });

        // Failure bars
        ctx.fillStyle = 'rgba(255, 0, 64, 0.35)';
        series.forEach((point, index) => {
            const fail = failureCounts[index];
            if (!Number.isFinite(fail)) {
                return;
            }
            const x = margin.left + stepX * index;
            const barHeight = Math.min(barMaxHeight, (fail / maxFailure) * barMaxHeight);
            ctx.fillRect(x - barWidth / 2, baseY - barHeight, barWidth, barHeight);
        });

        // Pass rate line
        ctx.strokeStyle = '#00ff41';
        ctx.lineWidth = 2;
        ctx.beginPath();
        passRates.forEach((value, index) => {
            const x = margin.left + stepX * index;
            const y = value === null ? baseY : margin.top + chartHeight - (value / 100) * chartHeight;
            if (index === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });
        ctx.stroke();

        // Pass rate markers
        ctx.fillStyle = '#00ff41';
        passRates.forEach((value, index) => {
            if (value === null) {
                return;
            }
            const x = margin.left + stepX * index;
            const y = margin.top + chartHeight - (value / 100) * chartHeight;
            ctx.beginPath();
            ctx.arc(x, y, 3, 0, Math.PI * 2, false);
            ctx.fill();
        });

        // X-axis labels
        ctx.fillStyle = 'rgba(255, 255, 255, 0.6)';
        ctx.font = '10px Inter, sans-serif';
        series.forEach((point, index) => {
            const bucket = point.bucket ? new Date(point.bucket) : null;
            if (!bucket || Number.isNaN(bucket.getTime())) {
                return;
            }
            const label = bucket.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
            const x = margin.left + stepX * index;
            ctx.fillText(label, x - 18, height - 8);
        });

        ctx.restore();
    }

    renderReportsScenarioTable(overview) {
        const container = document.getElementById('reports-scenarios');
        if (!container) {
            return;
        }

        const scenarios = overview && Array.isArray(overview.scenarios) ? [...overview.scenarios] : [];
        if (scenarios.length === 0) {
            container.innerHTML = '<p class="reports-empty">No scenario analytics yet. Generate and execute suites to populate this view.</p>';
            return;
        }

        scenarios.sort((a, b) => {
            const scoreA = Number(a.health_score || 0);
            const scoreB = Number(b.health_score || 0);
            return scoreA - scoreB;
        });

        const rows = scenarios.map((scenario) => {
            const name = this.escapeHtml(scenario.scenario_name || 'Unknown');
            const healthScore = Math.round(Number(scenario.health_score || 0));
            const passRate = Number(scenario.pass_rate || 0);
            const coverage = Number(scenario.coverage || 0);
            const coverageTarget = Number(scenario.target_coverage || 95);
            const activeFailures = Number(scenario.active_failures || 0);
            const runningExecutions = Number(scenario.running_executions || 0);
            const executionCount = Number(scenario.execution_count || 0);
            const avgDuration = Number(scenario.average_test_duration || 0);
            const lastStatus = scenario.last_execution_status || 'unknown';
            const lastEndedAt = scenario.last_execution_ended_at || null;
            const lastExecutionId = scenario.last_execution_id ? this.escapeHtml(scenario.last_execution_id) : '';

            const descriptor = this.getStatusDescriptor(lastStatus);
            const statusClass = this.getStatusClass(lastStatus);
            const statusLabel = descriptor ? descriptor.label : this.formatLabel(lastStatus || 'unknown');
            const statusIcon = descriptor?.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';

            const vault = scenario.vault || {};
            let vaultHtml = '<span class="text-muted">—</span>';
            if (vault.has_vault) {
                const vaultDescriptor = this.getStatusDescriptor(vault.latest_status || 'completed');
                const vaultStatusClass = this.getStatusClass(vault.latest_status || 'completed');
                const vaultLabel = vaultDescriptor ? vaultDescriptor.label : this.formatLabel(vault.latest_status || 'completed');
                const vaultIcon = vaultDescriptor?.icon ? `<i data-lucide="${vaultDescriptor.icon}"></i>` : '';
                const successRate = this.formatPercent(Number(vault.success_rate || 0), 1);
                vaultHtml = `
                    <div class="vault-summary">
                        <span class="status ${vaultStatusClass}">${vaultIcon}${this.escapeHtml(vaultLabel)}</span>
                        <span class="vault-meta">${successRate}</span>
                    </div>
                `;
            }

            const healthBarWidth = Math.min(Math.max(healthScore, 0), 100);
            const passClass = activeFailures > 0 ? 'text-error' : 'text-success';
            const passRateLabel = this.formatPercent(passRate, 1);
            const coverageLabel = this.formatPercent(coverage, 1);
            const durationLabel = this.formatDurationSeconds(avgDuration);
            const lastRunLabel = lastEndedAt ? this.formatTimestamp(lastEndedAt) : '—';

            return `
                <tr>
                    <td class="reports-cell-scenario">
                        <div class="scenario-name">${name}</div>
                        <div class="scenario-meta">${executionCount} execution${executionCount === 1 ? '' : 's'} · ${runningExecutions} running</div>
                    </td>
                    <td class="reports-cell-health">
                        <div class="progress">
                            <div class="progress-bar" style="width: ${healthBarWidth}%;"></div>
                        </div>
                        <span class="health-score">${healthScore}</span>
                    </td>
                    <td class="reports-cell-pass ${passClass}">${passRateLabel}</td>
                    <td class="reports-cell-fail ${activeFailures > 0 ? 'text-error' : 'text-muted'}">${activeFailures}</td>
                    <td class="reports-cell-coverage">
                        <div class="coverage-meter">
                            <div class="progress"><div class="progress-bar" style="width: ${Math.min(coverage, 100)}%;"></div></div>
                            <span>${coverageLabel}</span>
                        </div>
                        <small class="coverage-target">Target ${coverageTarget}%</small>
                    </td>
                    <td class="reports-cell-vault">${vaultHtml}</td>
                    <td class="reports-cell-last">
                        <div class="last-status status ${statusClass}">${statusIcon}${this.escapeHtml(statusLabel)}</div>
                        <div class="last-meta">${lastRunLabel}${lastExecutionId ? ` · <span class="code">${lastExecutionId.slice(0, 8)}</span>` : ''}</div>
                        <div class="last-meta">Avg test ${durationLabel}</div>
                    </td>
                </tr>
            `;
        }).join('');

        container.innerHTML = `
            <div class="table-wrapper">
                <table class="table reports-table">
                    <thead>
                        <tr>
                            <th>Scenario</th>
                            <th>Quality Index</th>
                            <th>Pass Rate</th>
                            <th>Active Failures</th>
                            <th>Coverage</th>
                            <th>Vault</th>
                            <th>Last Execution</th>
                        </tr>
                    </thead>
                    <tbody>${rows}</tbody>
                </table>
            </div>
        `;
    }

    renderReportsInsights(insightsResponse) {
        const container = document.getElementById('reports-insights');
        if (!container) {
            return;
        }

        const insights = insightsResponse && Array.isArray(insightsResponse.insights)
            ? insightsResponse.insights
            : [];

        if (insights.length === 0) {
            container.innerHTML = '<p class="reports-empty">No insights generated for this window.</p>';
            return;
        }

        const severityClassMap = {
            high: 'insight-severity-high',
            medium: 'insight-severity-medium',
            info: 'insight-severity-info',
            low: 'insight-severity-low'
        };

        const severityIconMap = {
            high: 'alert-triangle',
            medium: 'alert-circle',
            info: 'info',
            low: 'sparkles'
        };

        const items = insights.map((insight) => {
            const severityKey = (insight.severity || 'info').toLowerCase();
            const severityClass = severityClassMap[severityKey] || severityClassMap.info;
            const severityIcon = severityIconMap[severityKey] || severityIconMap.info;
            const severityLabel = this.formatLabel(insight.severity || 'info');
            const actions = Array.isArray(insight.actions) ? insight.actions : [];
            const scenarioBadge = insight.scenario_name
                ? `<span class="insight-scenario"><i data-lucide="box"></i>${this.escapeHtml(insight.scenario_name)}</span>`
                : '';

            const actionsHtml = actions.length
                ? `<ul class="insight-actions">${actions.map((action) => `<li>${this.escapeHtml(action)}</li>`).join('')}</ul>`
                : '';

            return `
                <article class="insight-card">
                    <header class="insight-header">
                        <div>
                            <h3 class="insight-title">${this.escapeHtml(insight.title || 'Insight')}</h3>
                            <div class="insight-meta">${scenarioBadge}</div>
                        </div>
                        <span class="insight-severity ${severityClass}"><i data-lucide="${severityIcon}"></i>${this.escapeHtml(severityLabel)}</span>
                    </header>
                    <p class="insight-detail">${this.escapeHtml(insight.detail || '')}</p>
                    ${actionsHtml}
                </article>
            `;
        }).join('');

        container.innerHTML = items;
    }

    renderReportsError(message) {
        const errorHtml = `<div class="reports-error">${this.escapeHtml(message)}</div>`;
        const trend = document.getElementById('reports-trend-body');
        const scenarios = document.getElementById('reports-scenarios');
        const insights = document.getElementById('reports-insights');

        if (trend) {
            trend.innerHTML = errorHtml;
        }
        if (scenarios) {
            scenarios.innerHTML = errorHtml;
        }
        if (insights) {
            insights.innerHTML = errorHtml;
        }

        const metricSubtitleIds = ['report-quality-subtitle', 'report-regressions-subtitle', 'report-coverage-subtitle', 'report-vault-subtitle'];
        metricSubtitleIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = 'Error loading metrics';
            }
        });
    }

    handleWindowResize() {
        if (this.activePage !== 'reports') {
            return;
        }

        const trends = this.currentData.reports && this.currentData.reports.trends;
        if (!trends || !Array.isArray(trends.series) || trends.series.length === 0) {
            return;
        }

        this.renderReportsTrends(trends);
    }

    renderCoverageTable(coverages) {
        const rows = coverages.map((coverage) => {
            const overall = Number(coverage.overall_coverage || 0).toFixed(1);
            const generatedAt = coverage.generated_at ? this.formatTimestamp(coverage.generated_at) : '—';
            const languages = Array.isArray(coverage.languages)
                ? coverage.languages.map((lang) => {
                    const statements = Number(lang.metrics?.statements ?? lang.metrics?.lines ?? 0).toFixed(1);
                    return `${this.escapeHtml(this.formatLabel(lang.language))} (${statements}%)`;
                  }).join(', ')
                : '—';

            const warnings = Array.isArray(coverage.warnings) && coverage.warnings.length > 0
                ? coverage.warnings.map((w) => `<div class="coverage-warning">⚠ ${this.escapeHtml(w)}</div>`).join('')
                : '';

            return `
                <tr>
                    <td class="cell-scenario"><strong>${this.escapeHtml(coverage.scenario_name || 'unknown')}</strong>${warnings}</td>
                    <td class="cell-coverage">${overall}%</td>
                    <td>${languages || '—'}</td>
                    <td>${generatedAt}</td>
                    <td class="cell-actions">
                        <button class="btn icon-btn" type="button" data-action="coverage-view" data-scenario="${this.escapeHtml(coverage.scenario_name)}">
                            <i data-lucide="eye"></i>
                        </button>
                        <button class="btn icon-btn secondary" type="button" data-action="coverage-generate" data-scenario="${this.escapeHtml(coverage.scenario_name)}">
                            <i data-lucide="zap"></i>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');

        return `
            <table class="table coverage-table">
                <thead>
                    <tr>
                        <th>Scenario</th>
                        <th>Overall Coverage</th>
                        <th>Languages</th>
                        <th>Last Generated</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        `;
    }

    bindCoverageTableActions() {
        if (!this.coverageTableContainer) {
            return;
        }

        this.coverageTableContainer.querySelectorAll('button[data-action]').forEach((button) => {
            button.addEventListener('click', async (event) => {
                const action = button.dataset.action;
                const scenarioName = button.dataset.scenario;
                if (!scenarioName) {
                    return;
                }

                switch (action) {
                    case 'coverage-view':
                        await this.viewCoverageAnalysis(scenarioName, button);
                        break;
                    case 'coverage-generate':
                        await this.generateCoverageAnalysis(scenarioName, button);
                        break;
                    default:
                        break;
                }
            });
        });

        this.refreshIcons();
    }

    async viewCoverageAnalysis(scenarioName, triggerButton) {
        if (!this.coverageDetailOverlay || !this.coverageDetailContent) {
            return;
        }

        this.openCoverageDetailDialog(triggerButton, scenarioName);
        this.coverageDetailContent.innerHTML = '<div class="loading"><div class="spinner"></div>Loading coverage analysis...</div>';

        try {
            const analysis = await this.fetchWithErrorHandling(`/test-analysis/coverage/${encodeURIComponent(scenarioName)}`);
            if (!analysis) {
                this.coverageDetailContent.innerHTML = '<p style="color: var(--accent-error);">Failed to load coverage analysis.</p>';
                return;
            }

            this.coverageDetailContent.innerHTML = this.renderCoverageDetail(scenarioName, analysis);
            this.refreshIcons();
        } catch (error) {
            console.error('Failed to load coverage analysis:', error);
            this.coverageDetailContent.innerHTML = '<p style="color: var(--accent-error);">Failed to load coverage analysis.</p>';
        }
    }

    renderCoverageDetail(scenarioName, analysis) {
        const overall = Number(analysis.overall_coverage ?? 0).toFixed(1);
        const coverageRows = Object.entries(analysis.coverage_by_file || {})
            .sort((a, b) => Number(b[1]) - Number(a[1]))
            .slice(0, 15)
            .map(([file, pct]) => `<tr><td>${this.escapeHtml(file)}</td><td>${Number(pct).toFixed(1)}%</td></tr>`)
            .join('');

        const gaps = analysis.coverage_gaps || {};
        const suggestions = Array.isArray(analysis.improvement_suggestions) && analysis.improvement_suggestions.length
            ? analysis.improvement_suggestions.map((item) => `<li>${this.escapeHtml(item)}</li>`).join('')
            : '<li>No immediate improvements suggested.</li>';
        const priorities = Array.isArray(analysis.priority_areas) && analysis.priority_areas.length
            ? analysis.priority_areas.map((item) => `<li>${this.escapeHtml(item)}</li>`).join('')
            : '<li>Monitor coverage trends over time.</li>';

        const gapSection = [
            { label: 'Untested Functions', items: gaps.untested_functions },
            { label: 'Untested Branches', items: gaps.untested_branches },
            { label: 'Untested Edge Cases', items: gaps.untested_edge_cases }
        ].map(({ label, items }) => {
            if (!Array.isArray(items) || items.length === 0) {
                return '';
            }
            return `
                <div class="coverage-gap-section">
                    <strong>${this.escapeHtml(label)}</strong>
                    <ul>${items.map((item) => `<li>${this.escapeHtml(item)}</li>`).join('')}</ul>
                </div>
            `;
        }).join('');

        const coverageTable = coverageRows
            ? `<table class="table compact"><thead><tr><th>File</th><th>Coverage</th></tr></thead><tbody>${coverageRows}</tbody></table>`
            : '<p style="color: var(--text-muted);">No file-level coverage data available.</p>';

        return `
            <div class="coverage-detail">
                <div class="coverage-detail-header">
                    <h3 style="margin-bottom: 0.25rem;">${this.escapeHtml(scenarioName)}</h3>
                    <p style="color: var(--text-muted);">Overall coverage: <strong>${overall}%</strong></p>
                </div>
                <div class="coverage-detail-grid">
                    <div class="coverage-detail-column">
                        <h4>Coverage by File</h4>
                        ${coverageTable}
                    </div>
                    <div class="coverage-detail-column">
                        <h4>Priority Areas</h4>
                        <ul>${priorities}</ul>
                        <h4>Improvement Suggestions</h4>
                        <ul>${suggestions}</ul>
                        ${gapSection}
                    </div>
                </div>
            </div>
        `;
    }

    async generateCoverageAnalysis(scenarioName, triggerButton) {
        if (!scenarioName) {
            this.showError('Scenario name missing');
            return;
        }

        const previousDisabled = triggerButton?.disabled || false;
        if (triggerButton) {
            triggerButton.disabled = true;
        }

        try {
            const payload = { scenario_name: scenarioName };
            const response = await fetch(`${this.apiBaseUrl}/test-analysis/coverage`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            this.showSuccess(`Coverage analysis generated for ${scenarioName}`);
            await this.loadCoverageSummaries();
            await this.viewCoverageAnalysis(scenarioName, null);
        } catch (error) {
            console.error('Coverage analysis generation failed:', error);
            this.showError(`Coverage analysis failed: ${error.message}`);
        } finally {
            if (triggerButton) {
                triggerButton.disabled = previousDisabled;
            }
        }
    }

    async handleVaultSubmit() {
        const form = document.getElementById('vault-form');
        if (!form) {
            return;
        }

        const scenarioInput = document.getElementById('vault-scenario');
        const vaultNameInput = document.getElementById('vault-name');
        const totalTimeoutInput = document.getElementById('vault-total-timeout');
        const resultCard = document.getElementById('vault-result');
        const submitButton = form.querySelector('button[type="submit"]');

        const scenarioName = (scenarioInput?.value || '').trim();
        if (!scenarioName) {
            this.showError('Please provide a scenario name for the vault.');
            return;
        }

        if (this.vaultSelectedPhases.size === 0) {
            this.showError('Select at least one phase that must pass.');
            return;
        }

        const phases = Array.from(this.vaultSelectedPhases);
        const vaultNameValue = (vaultNameInput?.value || '').trim();
        const vaultName = vaultNameValue || `${scenarioName}-vault-${Date.now()}`;

        const phaseConfigs = {};
        phases.forEach((phase) => {
            const definition = VAULT_PHASE_DEFINITIONS[phase] || { defaultTimeout: 600 };
            const timeoutInput = document.querySelector(`[data-vault-timeout="${phase}"]`);
            const descriptionInput = document.querySelector(`[data-vault-desc="${phase}"]`);
            const draft = this.vaultPhaseDrafts.get(phase) || {};

            let timeout = parseInt(timeoutInput?.value, 10);
            if (!Number.isFinite(timeout) || timeout <= 0) {
                timeout = Number(draft.timeout);
            }
            if (!Number.isFinite(timeout) || timeout <= 0) {
                timeout = definition.defaultTimeout;
            }

            const description = (descriptionInput?.value || draft.description || '').trim();

            phaseConfigs[phase] = {
                name: phase,
                description,
                timeout,
                tests: [],
                validation: {
                    phase_requirements: {}
                }
            };
        });

        const successCriteria = {
            all_phases_completed: document.getElementById('vault-criteria-all')?.checked ?? true,
            no_critical_failures: document.getElementById('vault-criteria-no-critical')?.checked ?? true,
            coverage_threshold: document.getElementById('vault-criteria-coverage')?.checked ? 95 : 0,
            performance_baseline_met: document.getElementById('vault-criteria-performance')?.checked ?? false
        };

        let totalTimeout = parseInt(totalTimeoutInput?.value, 10);
        if (!Number.isFinite(totalTimeout) || totalTimeout <= 0) {
            totalTimeout = phases.reduce((sum, phase) => sum + (phaseConfigs[phase]?.timeout || 0), 0) || 3600;
        }

        if (resultCard) {
            resultCard.classList.remove('visible');
            resultCard.innerHTML = '';
        }

        if (submitButton) {
            submitButton.disabled = true;
            submitButton.dataset.originalLabel = submitButton.innerHTML;
            submitButton.innerHTML = '<div class="spinner"></div> Creating...';
        }

        try {
            const payload = {
                scenario_name: scenarioName,
                vault_name: vaultName,
                phases,
                phase_configurations: phaseConfigs,
                success_criteria: successCriteria,
                total_timeout: totalTimeout
            };

            const response = await fetch(`${this.apiBaseUrl}/test-vault/create`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();

            if (resultCard) {
                const phaseLabels = phases.map((phase) => this.formatPhaseLabel(phase));
                const executionHint = result?.vault_id
                    ? `POST /api/v1/test-vault/${result.vault_id}/execute`
                    : 'POST /api/v1/test-vault/{vault_id}/execute';

                resultCard.innerHTML = `
                    <div style="font-weight:600; margin-bottom: 0.5rem;">Vault created successfully.</div>
                    <div><strong>Name:</strong> ${this.escapeHtml(vaultName)}</div>
                    <div><strong>Scenario:</strong> ${this.escapeHtml(scenarioName)}</div>
                    <div><strong>Phases:</strong> ${phaseLabels.map((label) => this.escapeHtml(label)).join(', ')}</div>
                    <div><strong>Total timeout:</strong> ${this.formatDurationSeconds(totalTimeout)}</div>
                    <div style="margin-top:0.75rem; font-size:0.8rem; color: var(--text-secondary);">
                        Execute via <code>${this.escapeHtml(executionHint)}</code>
                    </div>
                `;
                resultCard.classList.add('visible');
            }

            this.showSuccess(`Vault ready with ${phases.length} phase${phases.length === 1 ? '' : 's'}.`);

            form.reset();
            this.vaultPhaseDrafts.clear();
            this.resetVaultPhaseSelection();
            this.vaultScenarioOptionsLoaded = false;
            await this.loadVaultScenarioOptions(true);
            await this.loadVaultList();
        } catch (error) {
            console.error('Test vault creation failed:', error);
            this.showError(`Test vault creation failed: ${error.message}`);
        } finally {
            if (submitButton) {
                submitButton.disabled = false;
                submitButton.innerHTML = submitButton.dataset.originalLabel || 'Create Vault';
                delete submitButton.dataset.originalLabel;
            }
        }
    }

    // Action handlers
    async executeSuite(suiteId, options = {}) {
        const { navigate = true, showNotifications = true } = options;
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
            if (showNotifications) {
                this.showSuccess(`Test execution started! Execution ID: ${result.execution_id}`);
            }

            if (navigate) {
                this.navigateTo('executions');
            }

            return true;

        } catch (error) {
            console.error('Test execution failed:', error);
            if (showNotifications) {
                this.showError(`Test execution failed: ${error.message}`);
            }
            return false;
        }
    }

    async runSelectedSuites() {
        const suiteIds = Array.from(this.selectedSuiteIds);
        if (suiteIds.length === 0) {
            this.showInfo('Select at least one test suite to run.');
            return;
        }

        const button = this.runSelectedSuitesButton;
        const originalContent = button ? button.innerHTML : null;

        if (button) {
            button.disabled = true;
            button.innerHTML = '<div class="spinner"></div> Running...';
        }

        let successCount = 0;
        let failureCount = 0;

        for (const suiteId of suiteIds) {
            const started = await this.executeSuite(suiteId, {
                navigate: false,
                showNotifications: false
            });

            if (started) {
                successCount += 1;
            } else {
                failureCount += 1;
            }
        }

        this.selectedSuiteIds.clear();
        this.updateSuiteSelectionUI();

        if (button) {
            button.disabled = false;
            if (originalContent !== null) {
                button.innerHTML = originalContent;
            }
        }

        if (successCount > 0) {
            this.showSuccess(`Started ${successCount} test execution${successCount === 1 ? '' : 's'}.`);
            await Promise.all([
                this.loadExecutions(),
                this.loadRecentExecutions()
            ]);
            this.navigateTo('executions');
        }

        if (failureCount > 0) {
            this.showError(`Failed to start ${failureCount} test suite${failureCount === 1 ? '' : 's'}.`);
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

    isOverlayActive(overlay) {
        return overlay ? overlay.classList.contains('active') : false;
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

    viewExecution(executionId, triggerElement = null) {
        this.openExecutionDetail(executionId, triggerElement);
    }

    async deleteExecution(executionId, triggerButton = null, options = {}) {
        if (!executionId) {
            return false;
        }

        const { deferReload = false, suppressNotifications = false } = options;
        const button = triggerButton || null;
        const previousDisabledState = button ? button.disabled : false;
        const originalContent = button ? button.innerHTML : null;

        if (button) {
            button.disabled = true;
            button.innerHTML = '<div class="spinner"></div>';
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}/test-execution/${executionId}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const errorText = await response.text().catch(() => '');
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            const normalizedId = this.normalizeId(executionId);
            this.selectedExecutionIds.delete(normalizedId);

            if (this.normalizeId(this.activeExecutionDetailId) === normalizedId) {
                this.closeExecutionDetail();
            }

            if (!deferReload) {
                await Promise.all([
                    this.loadExecutions(),
                    this.loadRecentExecutions()
                ]);
            } else {
                this.updateExecutionSelectionUI();
            }

            if (!suppressNotifications) {
                this.showSuccess('Test execution deleted');
            }

            return true;
        } catch (error) {
            console.error('Failed to delete test execution:', error);
            if (!suppressNotifications) {
                this.showError(`Failed to delete execution: ${error.message}`);
            }
            return false;
        } finally {
            if (button) {
                button.disabled = previousDisabledState;
                if (originalContent !== null) {
                    button.innerHTML = originalContent;
                }
            }
        }
    }

    async deleteSelectedExecutions() {
        const executionIds = Array.from(this.selectedExecutionIds);
        if (executionIds.length === 0) {
            this.showInfo('Select at least one test execution to delete.');
            return;
        }

        const button = this.deleteSelectedExecutionsButton;
        const originalContent = button ? button.innerHTML : null;

        if (button) {
            button.disabled = true;
            button.innerHTML = '<div class="spinner"></div> Deleting...';
        }

        let deletedCount = 0;
        let failedCount = 0;

        for (const executionId of executionIds) {
            const removed = await this.deleteExecution(executionId, null, {
                deferReload: true,
                suppressNotifications: true
            });

            if (removed) {
                deletedCount += 1;
            } else {
                failedCount += 1;
            }
        }

        this.updateExecutionSelectionUI();

        if (deletedCount > 0) {
            this.showSuccess(`Deleted ${deletedCount} execution${deletedCount === 1 ? '' : 's'}.`);
            await Promise.all([
                this.loadExecutions(),
                this.loadRecentExecutions()
            ]);
        }

        if (failedCount > 0) {
            this.showError(`Failed to delete ${failedCount} execution${failedCount === 1 ? '' : 's'}.`);
        }

        if (button) {
            button.disabled = false;
            if (originalContent !== null) {
                button.innerHTML = originalContent;
            }
        }
    }

    async openExecutionDetail(executionId, triggerElement = null) {
        if (!executionId) {
            return;
        }

        if (!this.executionDetailOverlay) {
            this.showInfo(`Execution ${executionId}`);
            return;
        }

        this.activeExecutionDetailId = executionId;
        this.lastExecutionDetailTrigger = triggerElement;

        document.body.classList.add('execution-detail-open');
        this.executionDetailOverlay.classList.add('active');
        this.executionDetailOverlay.setAttribute('aria-hidden', 'false');

        if (this.executionDetailContent) {
            this.executionDetailContent.innerHTML = '<div class="loading"><div class="spinner"></div>Loading execution details...</div>';
        }

        if (this.executionDetailId) {
            this.executionDetailId.textContent = `Execution ID: ${executionId}`;
        }

        if (this.executionDetailSubtitle) {
            this.executionDetailSubtitle.textContent = '';
        }

        if (this.executionDetailStatus) {
            this.executionDetailStatus.innerHTML = '';
        }

        this.focusElement(this.executionDetailCloseButton);

        const execution = await this.fetchWithErrorHandling(`/test-execution/${executionId}/results`);

        if (!execution) {
            if (this.executionDetailContent) {
                this.executionDetailContent.innerHTML = '<div class="execution-detail-empty">Failed to load execution details.</div>';
            }
            return;
        }

        const suiteName = execution.suite_name || 'Test Execution';
        if (this.executionDetailTitle) {
            this.executionDetailTitle.textContent = `${suiteName} Results`;
        }

        if (this.executionDetailId) {
            const displayedId = execution.execution_id || executionId;
            this.executionDetailId.textContent = `Execution ID: ${displayedId}`;
        }

        if (this.executionDetailSubtitle) {
            this.executionDetailSubtitle.textContent = `Suite: ${suiteName}`;
        }

        if (this.executionDetailStatus) {
            const statusText = execution.status || 'unknown';
            const statusClass = this.getStatusClass(statusText);
            const summary = execution.summary || {};
            const coverageValue = Number(summary.coverage);
            const durationValue = Number(summary.duration);
            const coverageLabel = Number.isFinite(coverageValue) ? `${coverageValue.toFixed(1)}% coverage` : 'Coverage unavailable';
            const durationLabel = Number.isFinite(durationValue) ? `${durationValue.toFixed(1)}s runtime` : 'Duration unavailable';

            this.executionDetailStatus.innerHTML = `
                <span class="status ${statusClass}">${this.escapeHtml(statusText)}</span>
                <span class="execution-detail-chip"><i data-lucide="target"></i>${coverageLabel}</span>
                <span class="execution-detail-chip"><i data-lucide="clock"></i>${durationLabel}</span>
            `;
        }

        if (this.executionDetailContent) {
            this.executionDetailContent.innerHTML = this.renderExecutionDetail(execution);
        }

        this.refreshIcons();
    }

    closeExecutionDetail() {
        if (!this.executionDetailOverlay) {
            return;
        }

        this.executionDetailOverlay.classList.remove('active');
        this.executionDetailOverlay.setAttribute('aria-hidden', 'true');
        document.body.classList.remove('execution-detail-open');
        this.activeExecutionDetailId = null;

        if (this.lastExecutionDetailTrigger) {
            this.focusElement(this.lastExecutionDetailTrigger);
            this.lastExecutionDetailTrigger = null;
        }
    }

    isExecutionDetailOpen() {
        return Boolean(this.executionDetailOverlay && this.executionDetailOverlay.classList.contains('active'));
    }

    renderExecutionDetail(execution) {
        const summary = execution.summary || {};
        const performance = execution.performance_metrics || {};
        const failedTests = Array.isArray(execution.failed_tests) ? execution.failed_tests : [];
        const recommendations = Array.isArray(execution.recommendations) ? execution.recommendations : [];

        const toNumber = (value) => {
            const num = Number(value);
            return Number.isFinite(num) ? num : null;
        };

        const totalTests = toNumber(summary.total_tests) ?? 0;
        const passed = toNumber(summary.passed) ?? 0;
        const failed = toNumber(summary.failed) ?? 0;
        const skipped = toNumber(summary.skipped) ?? 0;
        const durationValue = toNumber(summary.duration);
        const coverageValue = toNumber(summary.coverage);
        const durationLabel = durationValue !== null ? `${durationValue.toFixed(1)}s` : '—';
        const coverageLabel = coverageValue !== null ? `${coverageValue.toFixed(1)}%` : '—';

        const summarySection = `
            <div class="suite-detail-section">
                <h3>Summary</h3>
                <div class="suite-detail-grid">
                    <div class="suite-detail-card">
                        <span class="label">Total Tests</span>
                        <span class="value">${totalTests}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Passed</span>
                        <span class="value text-success">${passed}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Failed</span>
                        <span class="value ${failed > 0 ? 'text-error' : ''}">${failed}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Skipped</span>
                        <span class="value">${skipped}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Duration</span>
                        <span class="value">${durationLabel}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Coverage</span>
                        <span class="value">${coverageLabel}</span>
                    </div>
                </div>
            </div>
        `;

        const performanceCards = [];
        const executionTime = toNumber(performance.execution_time);
        if (executionTime !== null) {
            performanceCards.push(`
                <div class="suite-detail-card">
                    <span class="label">Execution Time</span>
                    <span class="value">${executionTime.toFixed(1)}s</span>
                </div>
            `);
        }

        const errorCount = toNumber(performance.error_count);
        if (errorCount !== null) {
            performanceCards.push(`
                <div class="suite-detail-card">
                    <span class="label">Errors</span>
                    <span class="value ${errorCount > 0 ? 'text-error' : ''}">${errorCount}</span>
                </div>
            `);
        }

        const throughput = toNumber(performance.resource_usage?.tests_per_second);
        if (throughput !== null) {
            performanceCards.push(`
                <div class="suite-detail-card">
                    <span class="label">Tests / Second</span>
                    <span class="value">${throughput.toFixed(2)}</span>
                </div>
            `);
        }

        const additionalUsage = Object.entries(performance.resource_usage || {})
            .filter(([key]) => key !== 'tests_per_second');

        const performanceSection = (performanceCards.length > 0 || additionalUsage.length > 0)
            ? `
                <div class="suite-detail-section">
                    <h3>Performance</h3>
                    ${performanceCards.length > 0 ? `<div class="suite-detail-grid">${performanceCards.join('')}</div>` : ''}
                    ${additionalUsage.length > 0 ? `
                        <div class="suite-detail-subtitle">Resource Usage</div>
                        <div class="execution-detail-badges">
                            ${additionalUsage.map(([key, value]) => `<span class="execution-detail-badge">${this.escapeHtml(this.formatLabel(key))}: ${this.escapeHtml(this.stringifyValue(value))}</span>`).join('')}
                        </div>
                    ` : ''}
                </div>
            `
            : '';

        const failedSection = `
            <div class="suite-detail-section">
                <h3>Failed Tests</h3>
                ${failedTests.length > 0
                    ? `<div class="execution-detail-failed-tests">${failedTests.map((test, index) => this.renderFailedTest(test, index)).join('')}</div>`
                    : '<div class="suite-detail-empty">No failed tests 🎉</div>'}
            </div>
        `;

        const recommendationsSection = recommendations.length > 0
            ? `
                <div class="suite-detail-section">
                    <h3>Recommendations</h3>
                    <ul class="execution-detail-recommendations">
                        ${recommendations.map(item => `<li>${this.escapeHtml(item)}</li>`).join('')}
                    </ul>
                </div>
            `
            : '';

        return [summarySection, performanceSection, failedSection, recommendationsSection]
            .filter(Boolean)
            .join('');
    }

    renderFailedTest(test, index) {
        const name = this.escapeHtml(test.test_case_name || `Test ${index + 1}`);
        const description = test.test_case_description ? `<div class="suite-detail-subtitle">${this.escapeHtml(test.test_case_description)}</div>` : '';
        const durationValue = Number(test.duration);
        const durationLabel = Number.isFinite(durationValue) ? `${durationValue.toFixed(2)}s` : '—';
        const errorMessage = test.error_message ? `<div class="suite-detail-subtitle text-error">${this.escapeHtml(test.error_message)}</div>` : '';
        const assertionsMarkup = this.renderAssertions(test.assertions);
        const artifactsMarkup = this.renderArtifacts(test.artifacts);
        const stackTrace = test.stack_trace
            ? `<details class="execution-detail-stack"><summary>Stack Trace</summary><pre class="suite-detail-code">${this.escapeHtml(test.stack_trace)}</pre></details>`
            : '';

        return `
            <details class="execution-detail-failed-test">
                <summary>
                    <div><strong>${name}</strong></div>
                    <span class="suite-detail-subtitle">Duration: ${durationLabel}</span>
                </summary>
                ${description}
                ${errorMessage}
                ${assertionsMarkup}
                ${artifactsMarkup}
                ${stackTrace}
            </details>
        `;
    }

    renderAssertions(assertions) {
        if (!Array.isArray(assertions) || assertions.length === 0) {
            return '';
        }

        const blocks = assertions.map(assertion => {
            const status = assertion.passed ? 'Passed' : 'Failed';
            const statusClass = assertion.passed ? 'text-success' : 'text-error';
            const name = this.escapeHtml(assertion.name || 'Assertion');
            const message = assertion.message ? `<div class="suite-detail-subtitle">${this.escapeHtml(assertion.message)}</div>` : '';
            const expected = this.stringifyValue(assertion.expected);
            const actual = this.stringifyValue(assertion.actual);
            const expectedLine = expected ? `<div class="suite-detail-subtitle">Expected: ${this.escapeHtml(expected)}</div>` : '';
            const actualLine = actual ? `<div class="suite-detail-subtitle">Actual: ${this.escapeHtml(actual)}</div>` : '';

            return `
                <div class="execution-detail-assertion">
                    <div><strong>${name}</strong> <span class="${statusClass}">${status}</span></div>
                    ${expectedLine}
                    ${actualLine}
                    ${message}
                </div>
            `;
        }).join('');

        return `
            <div class="suite-detail-subtitle">Assertions</div>
            <div class="execution-detail-assertions">
                ${blocks}
            </div>
        `;
    }

    renderArtifacts(artifacts) {
        if (!artifacts || typeof artifacts !== 'object' || Object.keys(artifacts).length === 0) {
            return '';
        }

        const items = Object.entries(artifacts).map(([key, value]) => {
            return `<span class="execution-detail-badge">${this.escapeHtml(this.formatLabel(key))}: ${this.escapeHtml(this.stringifyValue(value))}</span>`;
        }).join('');

        return `
            <div class="suite-detail-subtitle">Artifacts</div>
            <div class="execution-detail-badges">
                ${items}
            </div>
        `;
    }

    async clearAllExecutions(triggerButton) {
        const confirmed = window.confirm('Clear all test executions? This cannot be undone.');
        if (!confirmed) {
            return;
        }

        const button = triggerButton || document.getElementById('executions-clear-btn');
        const originalContent = button ? button.innerHTML : null;

        if (button) {
            button.disabled = true;
            button.innerHTML = '<div class="spinner"></div> Clearing...';
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}/test-executions`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const errorText = await response.text().catch(() => '');
                throw new Error(errorText || `HTTP ${response.status}`);
            }

            const result = await response.json().catch(() => ({}));
            const deleted = typeof result.deleted === 'number' ? result.deleted : 'All';
            this.showSuccess(`Cleared ${deleted} test executions`);
            this.selectedExecutionIds.clear();
            this.updateExecutionSelectionUI();
            this.closeExecutionDetail();

            await Promise.all([
                this.loadExecutions(),
                this.loadRecentExecutions()
            ]);
        } catch (error) {
            console.error('Failed to clear test executions:', error);
            this.showError(`Failed to clear executions: ${error.message}`);
        } finally {
            if (button) {
                button.disabled = false;
                button.innerHTML = originalContent ?? '<i data-lucide="trash-2"></i> Clear All';
                this.refreshIcons();
            }
        }
    }

    bindSuitesTableActions(container) {
        const table = container.querySelector('.suites-table');
        if (!table || table.dataset.bound === 'true') {
            return;
        }

        this.suiteSelectAllCheckbox = table.querySelector('input[data-suite-select-all]') || null;

        table.addEventListener('click', event => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            const { action, suiteId, scenario } = button.dataset;

            if (action === 'generate' && scenario) {
                this.openGenerateDialog(button, scenario);
                return;
            }

            if (action === 'execute' && suiteId) {
                this.executeSuite(suiteId);
            }

            if (action === 'view' && suiteId) {
                this.viewSuite(suiteId, button);
            }
        });

        table.addEventListener('change', event => {
            const checkbox = event.target;
            if (!(checkbox instanceof HTMLInputElement)) {
                return;
            }

            if (checkbox.matches('[data-suite-select-all]')) {
                const shouldSelectAll = checkbox.checked;
                const rowCheckboxes = table.querySelectorAll('input[data-suite-id]');
                rowCheckboxes.forEach(rowCheckbox => {
                    const suiteId = this.normalizeId(rowCheckbox.dataset.suiteId);
                    if (!suiteId) {
                        return;
                    }
                    rowCheckbox.checked = shouldSelectAll;
                    if (shouldSelectAll) {
                        this.selectedSuiteIds.add(suiteId);
                    } else {
                        this.selectedSuiteIds.delete(suiteId);
                    }
                });
                this.updateSuiteSelectionUI();
                return;
            }

            if (checkbox.matches('input[data-suite-id]')) {
                const suiteId = this.normalizeId(checkbox.dataset.suiteId);
                if (!suiteId) {
                    return;
                }

                if (checkbox.checked) {
                    this.selectedSuiteIds.add(suiteId);
                } else {
                    this.selectedSuiteIds.delete(suiteId);
                }
                this.updateSuiteSelectionUI();
            }
        });

        table.dataset.bound = 'true';
    }

    bindExecutionsTableActions(container) {
        const table = container.querySelector('.executions-table');
        if (!table || table.dataset.bound === 'true') {
            return;
        }

        const supportsSelection = table.dataset.selectable === 'true';
        this.executionSelectAllCheckbox = supportsSelection
            ? table.querySelector('input[data-execution-select-all]') || null
            : null;

        table.addEventListener('click', event => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            if (button.dataset.action === 'view-execution' && button.dataset.executionId) {
                this.viewExecution(button.dataset.executionId, button);
                return;
            }

            if (button.dataset.action === 'delete-execution' && button.dataset.executionId) {
                this.deleteExecution(button.dataset.executionId, button);
            }
        });

        if (supportsSelection) {
            table.addEventListener('change', event => {
                const checkbox = event.target;
                if (!(checkbox instanceof HTMLInputElement)) {
                    return;
                }

                if (checkbox.matches('[data-execution-select-all]')) {
                    const shouldSelectAll = checkbox.checked;
                    const rowCheckboxes = table.querySelectorAll('input[data-execution-id]');
                    rowCheckboxes.forEach(rowCheckbox => {
                        const executionId = this.normalizeId(rowCheckbox.dataset.executionId);
                        if (!executionId) {
                            return;
                        }
                        rowCheckbox.checked = shouldSelectAll;
                        if (shouldSelectAll) {
                            this.selectedExecutionIds.add(executionId);
                        } else {
                            this.selectedExecutionIds.delete(executionId);
                        }
                    });
                    this.updateExecutionSelectionUI();
                    return;
                }

                if (checkbox.matches('input[data-execution-id]')) {
                    const executionId = this.normalizeId(checkbox.dataset.executionId);
                    if (!executionId) {
                        return;
                    }

                    if (checkbox.checked) {
                        this.selectedExecutionIds.add(executionId);
                    } else {
                        this.selectedExecutionIds.delete(executionId);
                    }
                    this.updateExecutionSelectionUI();
                }
            });
        }

        table.dataset.bound = 'true';
    }

    refreshIcons() {
        if (typeof lucide !== 'undefined' && typeof lucide.createIcons === 'function') {
            lucide.createIcons();
        }
    }

    toggleElementVisibility(element, shouldShow) {
        if (!element) {
            return;
        }
        element.classList.toggle('is-hidden', !shouldShow);
    }

    updateSuiteSelectionUI() {
        const suiteIdsSource = Array.isArray(this.renderedSuiteIds) && this.renderedSuiteIds.length
            ? this.renderedSuiteIds
            : (this.currentData.suites || []).map(suite => this.normalizeId(suite.id)).filter(Boolean);
        const suiteIds = suiteIdsSource.filter(Boolean);
        const selectedCount = suiteIds.filter(id => this.selectedSuiteIds.has(id)).length;
        const hasSelection = selectedCount > 0;
        const allSelected = suiteIds.length > 0 && selectedCount === suiteIds.length;

        this.toggleElementVisibility(this.runSelectedSuitesButton, hasSelection);

        if (this.runSelectedSuitesLabel) {
            this.runSelectedSuitesLabel.textContent = allSelected ? 'Run All' : 'Run Selected';
        }

        if (this.suiteSelectAllCheckbox) {
            this.suiteSelectAllCheckbox.checked = allSelected;
            this.suiteSelectAllCheckbox.indeterminate = !allSelected && selectedCount > 0;
        }
    }

    updateExecutionSelectionUI() {
        const hasSelection = this.selectedExecutionIds.size > 0;
        this.toggleElementVisibility(this.deleteSelectedExecutionsButton, hasSelection);

        if (this.executionSelectAllCheckbox) {
            const executionIds = (this.currentData.executions || [])
                .map(execution => this.normalizeId(execution.id))
                .filter(Boolean);
            const selectedCount = executionIds.filter(id => this.selectedExecutionIds.has(id)).length;
            const allSelected = executionIds.length > 0 && selectedCount === executionIds.length;
            this.executionSelectAllCheckbox.checked = allSelected;
            this.executionSelectAllCheckbox.indeterminate = !allSelected && selectedCount > 0;
        }
    }

    pruneSuiteSelection(suites) {
        const validIds = new Set();
        (suites || []).forEach((suite) => {
            if (!suite) {
                return;
            }

            if (typeof suite === 'string') {
                const id = this.normalizeId(suite);
                if (id) {
                    validIds.add(id);
                }
                return;
            }

            if (typeof suite === 'object') {
                const candidate = this.normalizeId(suite.id ?? suite.latestSuiteId ?? suite);
                if (candidate) {
                    validIds.add(candidate);
                }
            }
        });
        Array.from(this.selectedSuiteIds).forEach(id => {
            if (!validIds.has(id)) {
                this.selectedSuiteIds.delete(id);
            }
        });
    }

    pruneExecutionSelection(executions) {
        const validIds = new Set((executions || []).map(execution => this.normalizeId(execution.id)).filter(Boolean));
        Array.from(this.selectedExecutionIds).forEach(id => {
            if (!validIds.has(id)) {
                this.selectedExecutionIds.delete(id);
            }
        });
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
    lockDialogScroll() {
        document.body.classList.add('dialog-open');
    }

    unlockDialogScrollIfIdle() {
        const dialogsOpen = this.isOverlayActive(this.coverageDetailOverlay)
            || this.isOverlayActive(this.generateDialogOverlay)
            || this.isOverlayActive(this.vaultDialogOverlay);

        if (!dialogsOpen) {
            document.body.classList.remove('dialog-open');
        }
    }

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

    normalizeId(value) {
        if (value === null || value === undefined) {
            return '';
        }
        return String(value);
    }

    stringifyValue(value) {
        if (value === null || value === undefined) {
            return '';
        }

        if (typeof value === 'string') {
            return value;
        }

        if (typeof value === 'number' || typeof value === 'boolean') {
            return String(value);
        }

        try {
            return JSON.stringify(value);
        } catch (error) {
            return String(value);
        }
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
