// Test Genie Dashboard JavaScript
// ES6 Module Imports
import { DEFAULT_VAULT_PHASES, VAULT_PHASE_DEFINITIONS, DEFAULT_GENERATION_PHASES, STORAGE_KEYS } from './utils/constants.js';
import { dashboardPage } from './pages/DashboardPage.js';
import { executionsPage } from './pages/ExecutionsPage.js';
import { coveragePage } from './pages/CoveragePage.js';
import { settingsPage } from './pages/SettingsPage.js';
import { reportsPage } from './pages/ReportsPage.js';
import { suitesPage } from './pages/SuitesPage.js';
import { vaultPage } from './pages/VaultPage.js';
import { eventBus, EVENT_TYPES } from './core/EventBus.js';
import { stateManager } from './core/StateManager.js';
import { apiClient } from './core/ApiClient.js';
import { wsClient } from './core/WebSocketClient.js';
import { notificationManager } from './managers/NotificationManager.js';
import { dialogManager } from './managers/DialogManager.js';
import { navigationManager } from './managers/NavigationManager.js';
import { selectionManager } from './managers/SelectionManager.js';
import {
    formatTimestamp,
    formatDateTime,
    formatDateRange,
    formatDurationSeconds,
    formatPercent,
    formatPhaseLabel,
    parseDate,
    getStatusDescriptor,
    getStatusClass,
    formatLabel,
    formatRelativeTime,
    formatDetailedTimestamp,
    clampPercentage,
    extractSuiteTypes,
    countBy,
    summarizeCounts,
    calculateDuration,
    normalizeCollection,
    normalizeId,
    stringifyValue,
    escapeHtml
} from './utils/index.js';

// Legacy compatibility - these constants are now imported from utils/constants.js
const DEFAULT_SETTINGS_STORAGE_KEY = STORAGE_KEYS.DEFAULT_SETTINGS;

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
        this.generateDialogScenarioName = '';
        this.lastCoverageDialogTrigger = null;
        this.lastVaultDialogTrigger = null;
        this.coverageTargetInput = null;
        this.coverageTargetDisplay = null;
        this.generateForm = null;
        this.generateResultCard = null;
        this.generateResultIcon = null;
        this.generateResultTitle = null;
        this.generateResultMessage = null;
        this.generateResultMetadata = null;
        this.generateResultLinks = null;
        this.generateResultIssueButton = null;
        this.generateResultAgainButton = null;
        this.generateResultCloseButton = null;
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
        this.availableSuiteIds = [];
        this.renderedExecutionIds = [];
        this.currentSuiteRows = [];
        this.currentExecutionRows = [];
        this.reportsWindowDays = 30;
        this.reportsTrendCanvas = null;
        this.reportsTrendCtx = null;
        this.reportsIsLoading = false;
        this.suiteFilters = { search: '', status: 'all' };
        this.executionFilters = { search: '', status: 'all' };
        this.healthStatusData = null;
        this.healthStatusUpdatedAt = null;
        this.systemStatusChip = null;
        this.healthDialogOverlay = null;
        this.healthDialogCloseButton = null;
        this.healthDialogContent = null;
        this.healthStatusSubtitle = null;
        this.lastHealthDialogTrigger = null;
        this.defaultSettings = {
            coverageTarget: 80,
            phases: new Set(DEFAULT_GENERATION_PHASES)
        };
        this.pendingSettings = {
            coverageTarget: 80,
            phases: new Set(DEFAULT_GENERATION_PHASES)
        };
        this.settingsCoverageSlider = null;
        this.settingsCoverageDisplay = null;
        this.settingsPhaseCheckboxes = [];
        this.settingsSaveButton = null;
        this.suitesSearchInput = null;
        this.suitesStatusFilter = null;
        this.executionsSearchInput = null;
        this.executionsStatusFilter = null;
        this.handleWindowResize = this.handleWindowResize.bind(this);

        this.init();
    }

    async init() {
        this.setupEventListeners();
        navigationManager.initialize();
        // Settings are now handled by SettingsPage module
        // Scrollable containers are handled by individual page modules
        await this.loadInitialData();
    }

    setupEventListeners() {
        // Navigation
        document.addEventListener('click', (e) => {
            const navItem = e.target.closest('.nav-item');
            if (navItem) {
                const page = navItem.getAttribute('data-page');
                if (page) {
                    navigationManager.navigateTo(page);
                    if (window.innerWidth <= this.mobileBreakpoint) {
                        this.closeMobileSidebar();
                    }
                }
                return;
            }
        });

        this.sidebarToggleButton = document.getElementById('sidebar-toggle');
        this.sidebarOverlay = document.getElementById('sidebar-overlay');
        this.sidebarCloseButton = document.getElementById('sidebar-close');
        this.sidebarElement = document.getElementById('sidebar');
        this.systemStatusChip = document.getElementById('system-status-chip');

        if (this.sidebarToggleButton) {
            this.sidebarToggleButton.addEventListener('click', (e) => {
                e.preventDefault();
                this.toggleSidebar();
            });
        }

        if (this.systemStatusChip) {
            this.systemStatusChip.addEventListener('click', (event) => {
                event.preventDefault();
                this.handleSystemStatusChipActivate();
            });
            this.systemStatusChip.addEventListener('keydown', (event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    this.handleSystemStatusChipActivate();
                }
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

        // Form submissions
        document.getElementById('generate-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleGenerateSubmit();
        });

        document.getElementById('vault-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleVaultSubmit();
        });

        this.coverageTargetInput = document.getElementById('coverage-target');
        this.coverageTargetDisplay = document.getElementById('coverage-target-display');

        if (this.coverageTargetInput) {
            this.updateCoverageTargetDisplay();
            this.coverageTargetInput.addEventListener('input', () => this.updateCoverageTargetDisplay());
            this.coverageTargetInput.addEventListener('change', () => this.updateCoverageTargetDisplay());
        }

        // Filter event listeners are now handled by SuitesPage and ExecutionsPage modules
        // this.suitesSearchInput = document.getElementById('suites-search-input');
        // this.suitesStatusFilter = document.getElementById('suites-status-filter');
        // this.executionsSearchInput = document.getElementById('executions-search-input');
        // this.executionsStatusFilter = document.getElementById('executions-status-filter');

        this.runSelectedSuitesButton = document.getElementById('suites-run-selected-btn');
        this.runSelectedSuitesLabel = document.getElementById('suites-run-selected-label');
        if (this.runSelectedSuitesButton) {
            this.runSelectedSuitesButton.addEventListener('click', async (event) => {
                event.preventDefault();
                await this.runSelectedSuites();
            });
        }

        const suitesBulkGenerateButton = document.getElementById('suites-bulk-generate-btn');
        if (suitesBulkGenerateButton) {
            suitesBulkGenerateButton.addEventListener('click', (event) => {
                event.preventDefault();
                dialogManager.openGenerateDialog(suitesBulkGenerateButton, '', true);
            });
        }

        const suitesRefreshButton = document.getElementById('suites-refresh-btn');
        if (suitesRefreshButton) {
            suitesRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                const previousDisabled = suitesRefreshButton.disabled;
                suitesRefreshButton.disabled = true;
                try {
                    await suitesPage.load();
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
                    await executionsPage.load();
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
                await reportsPage.load();
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
                await reportsPage.load();
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
                    await coveragePage.load();
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
                if (dialogManager.isSuiteDetailOpen()) {
                    e.preventDefault();
                    dialogManager.closeSuiteDetail();
                    return;
                }

                if (dialogManager.isExecutionDetailOpen()) {
                    e.preventDefault();
                    dialogManager.closeExecutionDetail();
                    return;
                }

                if (dialogManager.isDialogOpen(this.coverageDetailOverlay)) {
                    e.preventDefault();
                    dialogManager.closeCoverageDetailDialog();
                    return;
                }

                if (dialogManager.isDialogOpen(this.generateDialogOverlay)) {
                    e.preventDefault();
                    dialogManager.closeGenerateDialog();
                    return;
                }

                if (dialogManager.isDialogOpen(this.vaultDialogOverlay)) {
                    e.preventDefault();
                    dialogManager.closeVaultDialog();
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
                        navigationManager.navigateTo('dashboard');
                        break;
                    case '2':
                        e.preventDefault();
                        navigationManager.navigateTo('suites');
                        break;
                    case '3':
                        e.preventDefault();
                        navigationManager.navigateTo('executions');
                        break;
                    case '4':
                        e.preventDefault();
                        navigationManager.navigateTo('coverage');
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

    // ========================================================================
    // LEGACY SETTINGS PAGE METHODS - Replaced by SettingsPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

    handleSystemStatusChipActivate() {
        dialogManager.openHealthDialog();
    }

    async openHealthDialog() {
        if (!this.healthDialogOverlay) {
            return;
        }

        this.lastHealthDialogTrigger = this.systemStatusChip || document.activeElement;
        this.healthDialogOverlay.classList.add('active');
        this.healthDialogOverlay.setAttribute('aria-hidden', 'false');
        if (this.systemStatusChip) {
            this.systemStatusChip.setAttribute('aria-expanded', 'true');
        }
        this.lockDialogScroll();
        this.renderHealthDialogContent({ loading: true });

        await this.ensureHealthDataFresh();
        this.renderHealthDialogContent();
    }

    renderHealthDialogContent({ loading = false } = {}) {
        if (!this.healthDialogContent) {
            return;
        }

        if (loading) {
            this.healthDialogContent.innerHTML = `
                <div class="loading" style="padding: var(--spacing-md);">
                    <div class="spinner"></div>
                    Loading health details…
                </div>
            `;
            return;
        }

        const data = this.healthStatusData;
        if (!data) {
            this.healthDialogContent.innerHTML = '<p style="padding: var(--spacing-lg); color: var(--text-muted);">No health data available.</p>';
            if (this.healthStatusSubtitle) {
                this.healthStatusSubtitle.textContent = 'Health data unavailable.';
            }
            return;
        }

        const pre = document.createElement('pre');
        pre.textContent = JSON.stringify(data, null, 2);

        this.healthDialogContent.innerHTML = '';
        this.healthDialogContent.appendChild(pre);

        if (this.healthStatusSubtitle) {
            const healthy = this.isHealthPayloadHealthy(data);
            const statusLabel = healthy ? 'Healthy' : 'Degraded';
            const updatedLabel = this.healthStatusUpdatedAt
                ? formatDetailedTimestamp(this.healthStatusUpdatedAt)
                : 'unknown';
            this.healthStatusSubtitle.textContent = `${statusLabel} • Updated ${updatedLabel}`;
        }
    }

    async ensureHealthDataFresh() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/health`);
            const data = await response.json();
            this.updateSystemStatus(data);
        } catch (error) {
            this.updateSystemStatus({
                healthy: false,
                status: 'error',
                message: 'Connection failed',
                error: error instanceof Error ? error.message : String(error)
            });
        }
    }

    updateCoverageTargetDisplay() {
        if (!this.coverageTargetInput || !this.coverageTargetDisplay) {
            return;
        }

        const value = parseInt(this.coverageTargetInput.value, 10);
        const normalized = Number.isFinite(value) ? value : 80;
        this.coverageTargetDisplay.textContent = `${normalized}%`;
    }

    updateGenerateScenarioDisplay() {
        const scenarioDisplay = document.getElementById('scenario-name-display');
        const scenarioChecklist = document.getElementById('scenario-checklist');
        const scenariosLabelSuffix = document.getElementById('scenarios-label-suffix');

        if (!scenarioDisplay || !scenarioChecklist) {
            return;
        }

        // Bulk mode: show checklist, hide single display
        if (this.generateDialogIsBulkMode) {
            scenarioDisplay.hidden = true;
            scenarioChecklist.hidden = false;
            if (scenariosLabelSuffix) {
                scenariosLabelSuffix.textContent = 's';
            }
            this.populateScenarioChecklist();
        }
        // Single mode: show single display, hide checklist
        else {
            scenarioDisplay.hidden = false;
            scenarioChecklist.hidden = true;
            if (scenariosLabelSuffix) {
                scenariosLabelSuffix.textContent = '';
            }

            if (this.generateDialogScenarioName) {
                scenarioDisplay.textContent = this.generateDialogScenarioName;
                scenarioDisplay.classList.remove('placeholder');
            } else {
                scenarioDisplay.textContent = 'Select a scenario from the table to generate a test suite.';
                scenarioDisplay.classList.add('placeholder');
            }
        }
    }

    populateScenarioChecklist() {
        const checklistItems = document.getElementById('scenario-checklist-items');
        const selectAllCheckbox = document.getElementById('scenario-select-all');

        if (!checklistItems) {
            return;
        }

        // Get scenarios without tests (isMissing = true)
        const scenariosWithoutTests = (this.currentSuiteRows || []).filter(scenario => scenario.isMissing);

        if (scenariosWithoutTests.length === 0) {
            checklistItems.innerHTML = '<p style="padding: var(--spacing-sm); color: var(--text-muted); text-align: center;">All scenarios have tests!</p>';
            if (selectAllCheckbox) {
                selectAllCheckbox.disabled = true;
            }
            return;
        }

        // Enable select all
        if (selectAllCheckbox) {
            selectAllCheckbox.disabled = false;
            selectAllCheckbox.checked = false;
        }

        // Render checklist items
        checklistItems.innerHTML = scenariosWithoutTests.map((scenario, index) => `
            <div class="scenario-checklist-item">
                <input type="checkbox" id="scenario-check-${index}" data-scenario-name="${escapeHtml(scenario.scenarioName)}">
                <label for="scenario-check-${index}">${escapeHtml(scenario.scenarioName)}</label>
            </div>
        `).join('');

        // Bind Select All functionality
        if (selectAllCheckbox) {
            selectAllCheckbox.addEventListener('change', (e) => {
                const checkboxes = checklistItems.querySelectorAll('input[type="checkbox"]');
                checkboxes.forEach(cb => {
                    cb.checked = e.target.checked;
                });
            });
        }

        // Update Select All when individual checkboxes change
        checklistItems.addEventListener('change', () => {
            if (selectAllCheckbox) {
                const checkboxes = checklistItems.querySelectorAll('input[type="checkbox"]');
                const allChecked = Array.from(checkboxes).every(cb => cb.checked);
                selectAllCheckbox.checked = allChecked;
            }
        });

        this.refreshIcons();
    }

    showGenerateForm(resetSelections = false, shouldFocus = false) {
        const form = this.generateForm || document.getElementById('generate-form');
        if (form) {
            form.classList.remove('generate-form--hidden');
            if (resetSelections && typeof form.reset === 'function') {
                form.reset();
            }
        }

        if (resetSelections) {
            // Settings handled by SettingsPage;
        }

        this.updateCoverageTargetDisplay();

        if (this.generateResultCard) {
            this.generateResultCard.hidden = true;
        }

        if (this.generateResultMetadata) {
            this.generateResultMetadata.innerHTML = '';
            this.generateResultMetadata.setAttribute('hidden', 'true');
        }

        if (this.generateResultMessage) {
            this.generateResultMessage.textContent = '';
        }

        if (this.generateResultTitle) {
            this.generateResultTitle.textContent = '';
        }

        if (this.generateResultLinks) {
            this.generateResultLinks.hidden = true;
        }

        if (this.generateResultIssueButton) {
            this.generateResultIssueButton.dataset.issueUrl = '';
            this.generateResultIssueButton.setAttribute('hidden', 'true');
        }

        if (this.generateResultIcon) {
            this.generateResultIcon.innerHTML = '<i data-lucide="check-circle"></i>';
            this.generateResultIcon.classList.remove('info', 'success');
            this.generateResultIcon.classList.add('success');
        }

        this.refreshIcons();

        if (shouldFocus) {
            const focusTarget = form?.querySelector('#generate-phase-selector input[type="checkbox"]')
                || this.coverageTargetInput
                || form?.querySelector('button, input, select, textarea');
            if (focusTarget) {
                this.focusElement(focusTarget);
            }
        }
    }

    showGenerateResultCard({ icon = 'check-circle', tone = 'success', title = '', message = '', metadata = [], issueUrl = '' } = {}) {
        if (this.generateForm) {
            this.generateForm.classList.add('generate-form--hidden');
        }

        if (this.generateResultCard) {
            this.generateResultCard.hidden = false;
        }

        if (this.generateResultIcon) {
            this.generateResultIcon.innerHTML = `<i data-lucide="${escapeHtml(icon)}"></i>`;
            this.generateResultIcon.classList.remove('success', 'info');
            const toneClass = tone === 'info' ? 'info' : 'success';
            this.generateResultIcon.classList.add(toneClass);
        }

        if (this.generateResultTitle) {
            this.generateResultTitle.textContent = title || '';
        }

        if (this.generateResultMessage) {
            this.generateResultMessage.textContent = message || '';
        }

        const preparedMetadata = Array.isArray(metadata)
            ? metadata.filter((item) => item && item.label && item.value !== undefined && item.value !== null && String(item.value).trim() !== '')
            : [];

        if (this.generateResultMetadata) {
            if (preparedMetadata.length) {
                this.generateResultMetadata.innerHTML = preparedMetadata.map(({ label, value }) => {
                    const labelHtml = escapeHtml(label);
                    const valueHtml = escapeHtml(String(value));
                    return `<dt>${labelHtml}</dt><dd>${valueHtml}</dd>`;
                }).join('');
                this.generateResultMetadata.removeAttribute('hidden');
            } else {
                this.generateResultMetadata.innerHTML = '';
                this.generateResultMetadata.setAttribute('hidden', 'true');
            }
        }

        if (this.generateResultLinks) {
            this.generateResultLinks.hidden = !issueUrl;
        }

        if (this.generateResultIssueButton) {
            if (issueUrl) {
                const issueUrlString = String(issueUrl);
                this.generateResultIssueButton.dataset.issueUrl = issueUrlString;
                this.generateResultIssueButton.removeAttribute('hidden');
            } else {
                this.generateResultIssueButton.dataset.issueUrl = '';
                this.generateResultIssueButton.setAttribute('hidden', 'true');
            }
        }

        this.refreshIcons();

        if (issueUrl && this.generateResultIssueButton) {
            this.focusElement(this.generateResultIssueButton);
        } else if (this.generateResultAgainButton) {
            this.focusElement(this.generateResultAgainButton);
        } else if (this.generateResultCloseButton) {
            this.focusElement(this.generateResultCloseButton);
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

    async loadInitialData() {
        try {
            await this.checkApiHealth();
            // Use new DashboardPage module
            await dashboardPage.load();
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showError('Failed to connect to Test Genie API');
        }
    }

    async checkApiHealth() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/health`);
            const data = await response.json();
            this.updateSystemStatus(data);
        } catch (error) {
            console.error('Health check failed:', error);
            this.updateSystemStatus({
                healthy: false,
                status: 'error',
                message: 'Connection failed',
                error: error instanceof Error ? error.message : String(error)
            });
        }
    }

    isHealthPayloadHealthy(payload) {
        if (!payload) {
            return false;
        }
        if (typeof payload.healthy === 'boolean') {
            return payload.healthy;
        }
        const status = (payload.status || '').toString().toLowerCase();
        return status === 'healthy' || status === 'ok' || status === 'up';
    }

    updateSystemStatus(payload) {
        const resolvedPayload = typeof payload === 'boolean' ? { healthy: payload } : payload;
        const statusDot = document.querySelector('.status-dot');
        const statusText = document.querySelector('.system-status span');
        const healthy = this.isHealthPayloadHealthy(resolvedPayload);
        const statusLabel = (resolvedPayload && typeof resolvedPayload.message === 'string' && resolvedPayload.message.trim())
            ? resolvedPayload.message.trim()
            : (healthy ? 'System Healthy' : 'System Offline');

        if (statusDot) {
            let color = healthy ? '#39ff14' : '#ff0040';
            const normalizedStatus = (resolvedPayload?.status || '').toString().toLowerCase();
            if (!healthy && ['degraded', 'warning', 'attention', 'partial'].includes(normalizedStatus)) {
                color = '#ff6b35';
            }
            statusDot.style.background = color;
            statusDot.style.boxShadow = `0 0 10px ${color}`;
        }

        if (statusText) {
            statusText.textContent = statusLabel;
        }

        if (this.systemStatusChip) {
            this.systemStatusChip.setAttribute('aria-label', `System status: ${statusLabel}`);
        }

        this.healthStatusData = resolvedPayload || null;
        this.healthStatusUpdatedAt = new Date();

        if (dialogManager.isDialogOpen(this.healthDialogOverlay)) {
            this.renderHealthDialogContent();
        }
    }

    // ========================================================================
    // LEGACY DASHBOARD METHODS - Replaced by DashboardPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

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
                        <th data-sortable="suite">Suite Name</th>
                        <th data-sortable="status">Status</th>
                        <th data-sortable="duration">Duration</th>
                        <th data-sortable="passed">Passed</th>
                        <th data-sortable="failed">Failed</th>
                        <th data-sortable="time">Time</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${executions.map(exec => {
                        const executionId = normalizeId(exec.id);
                        const isChecked = selectable && executionId && this.selectedExecutionIds.has(executionId);
                        const failedClass = exec.failed > 0 ? 'text-error' : 'text-muted';
                        const durationLabel = Number.isFinite(exec.duration) ? `${exec.duration}s` : '—';
                        const statusClass = exec.statusClass || getStatusClass(exec.status);
                        const selectionCell = selectable
                            ? `<td class="cell-select"><input type="checkbox" data-execution-id="${executionId}" ${isChecked ? 'checked' : ''} aria-label="Select execution ${escapeHtml(exec.suiteName || executionId || '')}"></td>`
                            : '';
                        return `
                        <tr>
                            ${selectionCell}
                            <td class="cell-scenario"><strong>${exec.suiteName}</strong></td>
                            <td class="cell-status"><span class="status ${statusClass}">${exec.status}</span></td>
                            <td>${durationLabel}</td>
                            <td class="text-success">${exec.passed}</td>
                            <td class="${failedClass}">${exec.failed}</td>
                            <td class="cell-created">${formatTimestamp(exec.timestamp)}</td>
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

    // ========================================================================
    // LEGACY VAULT PAGE METHODS - Replaced by VaultPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

    async loadPageData(page) {
        switch (page) {
            case 'suites':
                // Use new SuitesPage module
                await suitesPage.load();
                break;
            case 'executions':
                // Use new ExecutionsPage module
                await executionsPage.load();
                break;
            case 'dashboard':
                // Use new DashboardPage module
                await dashboardPage.load();
                break;
            case 'coverage':
                // Use new CoveragePage module
                await coveragePage.load();
                break;
            case 'vault':
                // Use new VaultPage module
                await vaultPage.load();
                break;
            case 'reports':
                // Use new ReportsPage module
                await reportsPage.load();
                break;
            case 'settings':
                // Use new SettingsPage module
                await settingsPage.load();
                break;
        }
    }

    // ========================================================================
    // LEGACY SUITES PAGE METHODS - Replaced by SuitesPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

    // ========================================================================
    // LEGACY EXECUTIONS PAGE METHODS - Replaced by ExecutionsPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

    async handleGenerateSubmit() {
        const form = this.generateForm || document.getElementById('generate-form');
        const btn = document.getElementById('generate-btn');

        if (!form || !btn) {
            console.error('Generate dialog elements are missing from the DOM.');
            return;
        }

        // Get scenarios to generate tests for
        let scenarioNames = [];
        if (this.generateDialogIsBulkMode) {
            // Bulk mode: collect checked scenarios from checklist
            const checklistItems = document.getElementById('scenario-checklist-items');
            if (checklistItems) {
                const checkedInputs = checklistItems.querySelectorAll('input[type="checkbox"]:checked');
                scenarioNames = Array.from(checkedInputs).map(input => input.dataset.scenarioName).filter(Boolean);
            }
        } else {
            // Single mode: use the single scenario name
            const scenarioName = (this.generateDialogScenarioName || '').trim();
            if (scenarioName) {
                scenarioNames = [scenarioName];
            }
        }

        let phaseInputs = Array.from(form.querySelectorAll('#generate-phase-selector input[type="checkbox"]'));
        if (!phaseInputs.length) {
            phaseInputs = Array.from(document.querySelectorAll('#generate-phase-selector input[type="checkbox"]'));
        }
        const selectedPhases = phaseInputs
            .filter((input) => input.checked)
            .map((input) => String(input.value || '').trim())
            .filter(Boolean);

        const coverageRaw = this.coverageTargetInput
            ? parseInt(this.coverageTargetInput.value, 10)
            : parseInt(document.getElementById('coverage-target')?.value ?? '80', 10);
        const coverageTarget = Number.isFinite(coverageRaw) ? coverageRaw : 80;
        const normalizedCoverage = Math.min(100, Math.max(50, coverageTarget));

        if (scenarioNames.length === 0 || selectedPhases.length === 0) {
            this.showError('Please fill in all required fields');
            return;
        }

        const includePerformance = selectedPhases.includes('performance');
        const includeSecurity = selectedPhases.includes('security');

        this.showGenerateForm(false, false);
        btn.disabled = true;
        const buttonText = this.generateDialogIsBulkMode
            ? `<div class="spinner"></div> Creating ${scenarioNames.length} ${scenarioNames.length === 1 ? 'Request' : 'Requests'}...`
            : '<div class="spinner"></div> Creating Request...';
        btn.innerHTML = buttonText;

        try {
            let response;

            if (this.generateDialogIsBulkMode && scenarioNames.length > 1) {
                // Bulk mode: call batch endpoint
                const requestData = {
                    scenario_names: scenarioNames,
                    test_types: selectedPhases,
                    coverage_target: normalizedCoverage,
                    options: {
                        include_performance_tests: includePerformance,
                        include_security_tests: includeSecurity,
                        custom_test_patterns: [],
                        execution_timeout: 300
                    }
                };

                response = await fetch(`${this.apiBaseUrl}/test-suite/generate-batch`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(requestData)
                });
            } else {
                // Single mode: call regular endpoint
                const requestData = {
                    scenario_name: scenarioNames[0],
                    test_types: selectedPhases,
                    coverage_target: normalizedCoverage,
                    options: {
                        include_performance_tests: includePerformance,
                        include_security_tests: includeSecurity,
                        custom_test_patterns: [],
                        execution_timeout: 300
                    }
                };

                response = await fetch(`${this.apiBaseUrl}/test-suite/generate`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(requestData)
                });
            }

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();

            // Handle bulk mode response
            if (this.generateDialogIsBulkMode && result.results) {
                const created = result.created || 0;
                const skipped = result.skipped || 0;
                const issues = result.issues || [];

                const metadata = [
                    { label: 'Created', value: `${created} ${created === 1 ? 'request' : 'requests'}` },
                    { label: 'Skipped', value: `${skipped} (already tested)` }
                ];

                const message = created > 0
                    ? `Created ${created} test ${created === 1 ? 'request' : 'requests'}${skipped > 0 ? `, skipped ${skipped} already tested` : ''}.`
                    : 'All selected scenarios already have tests.';

                this.showGenerateResultCard({
                    icon: created > 0 ? 'send' : 'info',
                    tone: created > 0 ? 'info' : 'warning',
                    title: `Bulk Test Requests ${created > 0 ? 'Created' : 'Skipped'}`,
                    message,
                    metadata,
                    issueUrl: issues.length > 0 ? issues[0].issue_url : ''
                });

                this.showSuccess(message);
            }
            // Handle single mode response
            else {
                const requestId = result.request_id || result.suite_id || 'unknown';
                const status = (result.status || 'submitted').toUpperCase();
                const baseMetadata = [
                    { label: 'Request ID', value: requestId },
                    { label: 'Status', value: status }
                ];

                if ((result.status || '').toLowerCase() === 'generated_locally') {
                    const metadata = [...baseMetadata];

                    if (Number.isFinite(Number(result.generated_tests))) {
                        metadata.push({ label: 'Generated Tests', value: String(result.generated_tests) });
                    }

                    if (Number.isFinite(Number(result.estimated_coverage))) {
                        metadata.push({ label: 'Estimated Coverage', value: `${result.estimated_coverage}%` });
                    }

                    if (Number.isFinite(Number(result.generation_time))) {
                        metadata.push({ label: 'Generation Time', value: `${result.generation_time}s` });
                    }

                    if (result.test_files && typeof result.test_files === 'object') {
                        Object.entries(result.test_files).forEach(([type, files]) => {
                            const count = Array.isArray(files) ? files.length : 0;
                            metadata.push({
                                label: `${formatLabel(type)} Files`,
                                value: `${count} ${count === 1 ? 'file' : 'files'}`
                            });
                        });
                    }

                    const generatedTests = Number.isFinite(Number(result.generated_tests)) ? Number(result.generated_tests) : 0;
                    const successMessage = generatedTests > 0
                        ? `Generated ${generatedTests} fallback ${generatedTests === 1 ? 'test' : 'tests'} locally.`
                        : 'Local test suite created without detected test cases.';

                    this.showGenerateResultCard({
                        icon: 'check-circle',
                        tone: 'success',
                        title: 'Local Test Suite Generated',
                        message: successMessage,
                        metadata,
                        issueUrl: ''
                    });

                    this.showSuccess(successMessage);
                } else {
                    const metadata = [...baseMetadata];
                    if (result.issue_id) {
                        metadata.push({ label: 'Issue ID', value: String(result.issue_id) });
                    }
                    if (!result.issue_url) {
                        metadata.push({ label: 'Follow-up', value: 'Track progress in app-issue-tracker.' });
                    }

                    this.showGenerateResultCard({
                        icon: 'send',
                        tone: 'info',
                        title: 'Test Request Created',
                        message: result.message || 'Delegated to app-issue-tracker.',
                        metadata,
                        issueUrl: result.issue_url || ''
                    });

                    this.showSuccess('Test request created in app-issue-tracker.');
                }
            }

            await Promise.all([
                this.loadTestSuites(),
                // Vault scenario options handled by VaultPage
            ]);

        } catch (error) {
            console.error('Test generation failed:', error);
            this.showError(`Test generation failed: ${error.message}`);
        } finally {
            btn.disabled = false;
            const finalButtonText = this.generateDialogIsBulkMode
                ? '<i data-lucide="sparkles"></i> Create Requests'
                : '<i data-lucide="zap"></i> Create Request';
            btn.innerHTML = finalButtonText;
            this.refreshIcons();
        }
    }

    // ========================================================================
    // LEGACY COVERAGE PAGE METHODS - Replaced by CoveragePage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

    // ========================================================================
    // LEGACY REPORTS PAGE METHODS - Replaced by ReportsPage module
    // Kept for reference during migration. Will be removed in Phase 7 cleanup.
    // ========================================================================

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
            const generatedAt = coverage.generated_at ? formatTimestamp(coverage.generated_at) : '—';
            const languages = Array.isArray(coverage.languages)
                ? coverage.languages.map((lang) => {
                    const statements = Number(lang.metrics?.statements ?? lang.metrics?.lines ?? 0).toFixed(1);
                    return `${escapeHtml(formatLabel(lang.language))} (${statements}%)`;
                  }).join(', ')
                : '—';

            const warnings = Array.isArray(coverage.warnings) && coverage.warnings.length > 0
                ? coverage.warnings.map((w) => `<div class="coverage-warning">⚠ ${escapeHtml(w)}</div>`).join('')
                : '';

            return `
                <tr>
                    <td class="cell-scenario"><strong>${escapeHtml(coverage.scenario_name || 'unknown')}</strong>${warnings}</td>
                    <td class="cell-coverage">${overall}%</td>
                    <td>${languages || '—'}</td>
                    <td>${generatedAt}</td>
                    <td class="cell-actions">
                        <button class="btn icon-btn" type="button" data-action="coverage-view" data-scenario="${escapeHtml(coverage.scenario_name)}">
                            <i data-lucide="eye"></i>
                        </button>
                        <button class="btn icon-btn secondary" type="button" data-action="coverage-generate" data-scenario="${escapeHtml(coverage.scenario_name)}">
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

        dialogManager.openCoverageDetailDialog(triggerButton, scenarioName);
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
            .map(([file, pct]) => `<tr><td>${escapeHtml(file)}</td><td>${Number(pct).toFixed(1)}%</td></tr>`)
            .join('');

        const gaps = analysis.coverage_gaps || {};
        const suggestions = Array.isArray(analysis.improvement_suggestions) && analysis.improvement_suggestions.length
            ? analysis.improvement_suggestions.map((item) => `<li>${escapeHtml(item)}</li>`).join('')
            : '<li>No immediate improvements suggested.</li>';
        const priorities = Array.isArray(analysis.priority_areas) && analysis.priority_areas.length
            ? analysis.priority_areas.map((item) => `<li>${escapeHtml(item)}</li>`).join('')
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
                    <strong>${escapeHtml(label)}</strong>
                    <ul>${items.map((item) => `<li>${escapeHtml(item)}</li>`).join('')}</ul>
                </div>
            `;
        }).join('');

        const coverageTable = coverageRows
            ? `<table class="table compact"><thead><tr><th>File</th><th>Coverage</th></tr></thead><tbody>${coverageRows}</tbody></table>`
            : '<p style="color: var(--text-muted);">No file-level coverage data available.</p>';

        return `
            <div class="coverage-detail">
                <div class="coverage-detail-header">
                    <h3 style="margin-bottom: 0.25rem;">${escapeHtml(scenarioName)}</h3>
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
            await coveragePage.load();
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
                const phaseLabels = phases.map((phase) => formatPhaseLabel(phase));
                const executionHint = result?.vault_id
                    ? `POST /api/v1/test-vault/${result.vault_id}/execute`
                    : 'POST /api/v1/test-vault/{vault_id}/execute';

                resultCard.innerHTML = `
                    <div style="font-weight:600; margin-bottom: 0.5rem;">Vault created successfully.</div>
                    <div><strong>Name:</strong> ${escapeHtml(vaultName)}</div>
                    <div><strong>Scenario:</strong> ${escapeHtml(scenarioName)}</div>
                    <div><strong>Phases:</strong> ${phaseLabels.map((label) => escapeHtml(label)).join(', ')}</div>
                    <div><strong>Total timeout:</strong> ${formatDurationSeconds(totalTimeout)}</div>
                    <div style="margin-top:0.75rem; font-size:0.8rem; color: var(--text-secondary);">
                        Execute via <code>${escapeHtml(executionHint)}</code>
                    </div>
                `;
                resultCard.classList.add('visible');
            }

            this.showSuccess(`Vault ready with ${phases.length} phase${phases.length === 1 ? '' : 's'}.`);

            form.reset();
            this.vaultPhaseDrafts.clear();
            this.resetVaultPhaseSelection();
            this.vaultScenarioOptionsLoaded = false;
            // Vault scenario options handled by VaultPage;
            await vaultPage.load();
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
                navigationManager.navigateTo('executions');
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
        selectionManager.updateSuiteSelectionUI();

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
            navigationManager.navigateTo('executions');
        }

        if (failureCount > 0) {
            this.showError(`Failed to start ${failureCount} test suite${failureCount === 1 ? '' : 's'}.`);
        }
    }

    viewSuite(suiteId, triggerElement = null) {
        dialogManager.openSuiteDetail(suiteId, triggerElement);
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
            const createdText = createdAt ? `Generated ${formatDetailedTimestamp(createdAt)}` : 'Generated date unavailable';
            const executedText = lastExecutedAt ? `Last executed ${formatDetailedTimestamp(lastExecutedAt)}` : 'Not executed yet';
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
        const statusLabel = formatLabel(suite.status || 'unknown');
        const statusClass = getStatusClass(suite.status);
        const lastExecutedAt = suite.last_executed || suite.lastExecuted;
        const lastExecutedSummary = lastExecutedAt ? `Ran ${formatRelativeTime(lastExecutedAt)}` : 'No executions yet';

        const suiteTypes = extractSuiteTypes(suite, testCases);
        const suiteTypeSummary = suiteTypes.length ? suiteTypes.map(type => formatLabel(type)).join(', ') : 'No types recorded';

        const priorityCounts = countBy(testCases.map(testCase => (testCase.priority || 'unspecified')));
        const prioritySummary = summarizeCounts(priorityCounts, 'priority');

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
                        <span class="value"><span class="status ${statusClass}">${escapeHtml(statusLabel)}</span></span>
                        <span class="value-small">${escapeHtml(lastExecutedSummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Test Cases</span>
                        <span class="value">${totalTests}</span>
                        <span class="value-small">${escapeHtml(prioritySummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Suite Types</span>
                        <span class="value-small">${escapeHtml(suiteTypeSummary)}</span>
                    </div>
                    <div class="suite-detail-card">
                        <span class="label">Metadata</span>
                        <span class="value-small">${escapeHtml(metadataPieces.length ? metadataPieces.join(' • ') : 'No metadata captured')}</span>
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
            const percent = clampPercentage(Number(entry.value));
            return `
                <div class="coverage-item">
                    <div class="coverage-item-header">
                        <span>${escapeHtml(entry.label)}</span>
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
            const name = escapeHtml(testCase.name || `Test ${index + 1}`);
            const description = testCase.description ? escapeHtml(testCase.description) : '';
            const testType = escapeHtml(formatLabel(testCase.test_type || 'unspecified'));
            const priority = escapeHtml(formatLabel(testCase.priority || 'unspecified'));
            const timeout = Number(testCase.execution_timeout);
            const timeoutText = Number.isFinite(timeout) ? `${timeout}s` : '—';
            const expected = escapeHtml(testCase.expected_result || '—');

            const tags = Array.isArray(testCase.tags) && testCase.tags.length > 0
                ? testCase.tags.filter(Boolean).map(tag => `<span class="suite-detail-tag">${escapeHtml(tag)}</span>`).join('')
                : '<span class="suite-detail-subtitle">None</span>';

            const dependencies = Array.isArray(testCase.dependencies) && testCase.dependencies.length > 0
                ? testCase.dependencies.filter(Boolean).map(dep => `<span class="suite-detail-tag">${escapeHtml(dep)}</span>`).join('')
                : '<span class="suite-detail-subtitle">None</span>';

            const codeBlock = testCase.test_code
                ? `<details><summary>View Generated Test</summary><pre class="suite-detail-code">${escapeHtml(testCase.test_code)}</pre></details>`
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
        dialogManager.openExecutionDetail(executionId, triggerElement);
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

            const normalizedId = normalizeId(executionId);
            this.selectedExecutionIds.delete(normalizedId);

            if (normalizeId(this.activeExecutionDetailId) === normalizedId) {
                dialogManager.closeExecutionDetail();
            }

            if (!deferReload) {
                await Promise.all([
                    this.loadExecutions(),
                    this.loadRecentExecutions()
                ]);
            } else {
                selectionManager.updateExecutionSelectionUI();
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

        selectionManager.updateExecutionSelectionUI();

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
            const statusClass = getStatusClass(statusText);
            const summary = execution.summary || {};
            const coverageValue = Number(summary.coverage);
            const durationValue = Number(summary.duration);
            const coverageLabel = Number.isFinite(coverageValue) ? `${coverageValue.toFixed(1)}% coverage` : 'Coverage unavailable';
            const durationLabel = Number.isFinite(durationValue) ? `${durationValue.toFixed(1)}s runtime` : 'Duration unavailable';

            this.executionDetailStatus.innerHTML = `
                <span class="status ${statusClass}">${escapeHtml(statusText)}</span>
                <span class="execution-detail-chip"><i data-lucide="target"></i>${coverageLabel}</span>
                <span class="execution-detail-chip"><i data-lucide="clock"></i>${durationLabel}</span>
            `;
        }

        if (this.executionDetailContent) {
            this.executionDetailContent.innerHTML = this.renderExecutionDetail(execution);
        }

        this.refreshIcons();
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
                            ${additionalUsage.map(([key, value]) => `<span class="execution-detail-badge">${escapeHtml(formatLabel(key))}: ${escapeHtml(stringifyValue(value))}</span>`).join('')}
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
                        ${recommendations.map(item => `<li>${escapeHtml(item)}</li>`).join('')}
                    </ul>
                </div>
            `
            : '';

        return [summarySection, performanceSection, failedSection, recommendationsSection]
            .filter(Boolean)
            .join('');
    }

    renderFailedTest(test, index) {
        const name = escapeHtml(test.test_case_name || `Test ${index + 1}`);
        const description = test.test_case_description ? `<div class="suite-detail-subtitle">${escapeHtml(test.test_case_description)}</div>` : '';
        const durationValue = Number(test.duration);
        const durationLabel = Number.isFinite(durationValue) ? `${durationValue.toFixed(2)}s` : '—';
        const errorMessage = test.error_message ? `<div class="suite-detail-subtitle text-error">${escapeHtml(test.error_message)}</div>` : '';
        const assertionsMarkup = this.renderAssertions(test.assertions);
        const artifactsMarkup = this.renderArtifacts(test.artifacts);
        const stackTrace = test.stack_trace
            ? `<details class="execution-detail-stack"><summary>Stack Trace</summary><pre class="suite-detail-code">${escapeHtml(test.stack_trace)}</pre></details>`
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
            const name = escapeHtml(assertion.name || 'Assertion');
            const message = assertion.message ? `<div class="suite-detail-subtitle">${escapeHtml(assertion.message)}</div>` : '';
            const expected = stringifyValue(assertion.expected);
            const actual = stringifyValue(assertion.actual);
            const expectedLine = expected ? `<div class="suite-detail-subtitle">Expected: ${escapeHtml(expected)}</div>` : '';
            const actualLine = actual ? `<div class="suite-detail-subtitle">Actual: ${escapeHtml(actual)}</div>` : '';

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
            return `<span class="execution-detail-badge">${escapeHtml(formatLabel(key))}: ${escapeHtml(stringifyValue(value))}</span>`;
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
            selectionManager.updateExecutionSelectionUI();
            dialogManager.closeExecutionDetail();

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
        const alreadyBound = container.dataset.bound === 'true';
        if (alreadyBound) {
            return;
        }

        this.suiteSelectAllCheckbox = table ? table.querySelector('input[data-suite-select-all]') || null : null;

        // Add sorting functionality
        if (table) {
            const thead = table.querySelector('thead');
            if (thead) {
                thead.addEventListener('click', event => {
                    const th = event.target.closest('th[data-sortable]');
                    if (!th) return;

                    const sortKey = th.dataset.sortable;
                    const currentDirection = th.dataset.sortDirection || 'none';
                    const newDirection = currentDirection === 'asc' ? 'desc' : 'asc';

                    // Reset all other headers
                    thead.querySelectorAll('th[data-sortable]').forEach(header => {
                        header.dataset.sortDirection = 'none';
                        header.style.position = 'relative';
                    });

                    th.dataset.sortDirection = newDirection;
                    this.sortSuitesTable(table, sortKey, newDirection);
                });
            }
        }

        container.addEventListener('click', event => {
            // Check if clicking on a button first
            const button = event.target.closest('[data-action]');
            if (button) {
                const { action, suiteId, scenario } = button.dataset;

                if (action === 'generate' && scenario) {
                    dialogManager.openGenerateDialog(button, scenario);
                    return;
                }

                if (action === 'execute' && suiteId) {
                    this.executeSuite(suiteId);
                    return;
                }

                if (action === 'view' && suiteId) {
                    this.viewSuite(suiteId, button);
                    return;
                }
            }

            // If not clicking a button, check for row or card click
            const row = event.target.closest('tr[data-suite-id]');
            const card = event.target.closest('.suite-card[data-suite-id]');
            const clickTarget = row || card;

            if (clickTarget) {
                // Ignore clicks on checkboxes and buttons
                if (event.target.closest('input[type="checkbox"]') || event.target.closest('button')) {
                    return;
                }

                const suiteId = clickTarget.dataset.suiteId;
                if (suiteId) {
                    this.viewSuite(suiteId, clickTarget);
                }
            }
        });

        container.addEventListener('change', event => {
            const checkbox = event.target;
            if (!(checkbox instanceof HTMLInputElement)) {
                return;
            }

            if (checkbox.matches('[data-suite-select-all]')) {
                const shouldSelectAll = checkbox.checked;
                const allCheckboxes = container.querySelectorAll('input[data-suite-id]');
                allCheckboxes.forEach(rowCheckbox => {
                    const suiteId = normalizeId(rowCheckbox.dataset.suiteId);
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
                selectionManager.updateSuiteSelectionUI();
                return;
            }

            if (checkbox.matches('input[data-suite-id]')) {
                const suiteId = normalizeId(checkbox.dataset.suiteId);
                if (!suiteId) {
                    return;
                }

                if (checkbox.checked) {
                    this.selectedSuiteIds.add(suiteId);
                } else {
                    this.selectedSuiteIds.delete(suiteId);
                }
                selectionManager.updateSuiteSelectionUI();
            }
        });

        container.dataset.bound = 'true';
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

        // Add sorting functionality
        const thead = table.querySelector('thead');
        if (thead) {
            thead.addEventListener('click', event => {
                const th = event.target.closest('th[data-sortable]');
                if (!th) return;

                const sortKey = th.dataset.sortable;
                const currentDirection = th.dataset.sortDirection || 'none';
                const newDirection = currentDirection === 'asc' ? 'desc' : 'asc';

                // Reset all other headers
                thead.querySelectorAll('th[data-sortable]').forEach(header => {
                    header.dataset.sortDirection = 'none';
                    header.style.position = 'relative';
                });

                th.dataset.sortDirection = newDirection;
                this.sortExecutionsTable(table, sortKey, newDirection);
            });
        }

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
                        const executionId = normalizeId(rowCheckbox.dataset.executionId);
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
                    selectionManager.updateExecutionSelectionUI();
                    return;
                }

                if (checkbox.matches('input[data-execution-id]')) {
                    const executionId = normalizeId(checkbox.dataset.executionId);
                    if (!executionId) {
                        return;
                    }

                    if (checkbox.checked) {
                        this.selectedExecutionIds.add(executionId);
                    } else {
                        this.selectedExecutionIds.delete(executionId);
                    }
                    selectionManager.updateExecutionSelectionUI();
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

    // Utility methods
    async refreshCurrentPage() {
        await this.loadPageData(this.activePage);
        this.showSuccess('Page refreshed!');
    }

    refreshDashboard() {
        dashboardPage.load();
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

    // WebSocket connection for real-time updates
    // Enhanced periodic updates with better error handling
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
