/**
 * VaultPage - Test Vault Management View
 * Handles vault creation, configuration, execution tracking, and phase management
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import {
    normalizeCollection,
    normalizeId,
    formatTimestamp,
    formatDateTime,
    formatDurationSeconds,
    formatPhaseLabel,
    getStatusDescriptor,
    getStatusClass,
    escapeHtml,
    parseDate
} from '../utils/index.js';
import { DEFAULT_VAULT_PHASES, VAULT_PHASE_DEFINITIONS } from '../utils/constants.js';
import { enableDragScroll, refreshIcons } from '../utils/domHelpers.js';

/**
 * VaultPage class - Manages test vault functionality
 */
export class VaultPage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        apiClientInstance = apiClient,
        notificationManagerInstance = notificationManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.apiClient = apiClientInstance;
        this.notificationManager = notificationManagerInstance;

        // DOM references
        this.vaultForm = null;
        this.vaultPhaseConfigContainer = null;
        this.vaultResultCard = null;
        this.vaultRefreshButton = null;
        this.vaultListContainer = null;
        this.vaultTimelineContainer = null;
        this.vaultTimelineContent = null;
        this.vaultHistoryContainer = null;
        this.vaultHistoryContent = null;
        this.vaultPhaseCheckboxes = [];

        // State
        this.vaultSelectedPhases = new Set();
        this.vaultPhaseDrafts = new Map();
        this.vaultIndexById = new Map();
        this.activeVaultId = null;
        this.vaultScenarioOptionsLoaded = false;
        this.vaultInitialized = false;

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize vault page
     */
    initialize() {
        this.setupEventListeners();

        if (this.debug) {
            console.log('[VaultPage] Vault page initialized');
        }
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'vault') {
                this.load();
            }
        });
    }

    /**
     * Load vault page
     */
    async load() {
        if (this.debug) {
            console.log('[VaultPage] Loading vault page');
        }

        try {
            // Set loading state
            this.stateManager.setLoading('vault', true);

            // Setup vault page (one-time initialization)
            this.setupVaultPage();

            // Load data in parallel
            await Promise.all([
                this.loadVaultScenarioOptions(),
                this.loadVaultList()
            ]);

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'vault' });

        } catch (error) {
            console.error('[VaultPage] Failed to load vault page:', error);
            this.notificationManager.showError('Failed to load vault page');
        } finally {
            this.stateManager.setLoading('vault', false);
        }
    }

    /**
     * Setup vault page DOM references and event handlers
     * @private
     */
    setupVaultPage() {
        if (this.vaultInitialized) {
            return;
        }

        // Setup DOM references
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

        // Initialize phase selection
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

        // Setup form reset handler
        if (this.vaultForm) {
            this.vaultForm.addEventListener('reset', () => {
                this.vaultPhaseDrafts.clear();
                window.setTimeout(() => this.resetVaultPhaseSelection(), 0);
            });

            this.vaultForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                await this.handleVaultSubmit();
            });
        }

        // Setup refresh button
        if (this.vaultRefreshButton) {
            this.vaultRefreshButton.addEventListener('click', async (event) => {
                event.preventDefault();
                await this.loadVaultList();
            });
        }

        // Reset to default phase selection
        this.resetVaultPhaseSelection();
        this.hideVaultDetails('Select a vault to inspect execution details.');

        this.vaultInitialized = true;
    }

    /**
     * Reset vault phase selection to defaults
     * @private
     */
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

    /**
     * Update vault phase configuration UI
     * @private
     */
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
                label: formatPhaseLabel(phase),
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
                    <span class="phase-name">${escapeHtml(definition.label)}</span>
                    <div class="phase-timeout">
                        <label>Timeout:</label>
                        <input type="number" class="timeout-input" data-vault-timeout="${phase}" value="${timeoutValue}" min="30">
                        <span>seconds</span>
                    </div>
                </div>
                <textarea class="form-textarea" placeholder="${escapeHtml(definition.description)}" data-vault-desc="${phase}"></textarea>
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

    /**
     * Load vault scenario options for autocomplete
     * @param {boolean} forceRefresh - Force refresh scenario options
     */
    async loadVaultScenarioOptions(forceRefresh = false) {
        const datalist = document.getElementById('vault-scenario-options');
        if (!datalist) {
            return;
        }

        if (this.vaultScenarioOptionsLoaded && !forceRefresh) {
            return;
        }

        try {
            const scenariosResponse = await this.apiClient.getScenarios();
            const scenarios = normalizeCollection(scenariosResponse, 'scenarios');
            const names = Array.from(new Set((scenarios || [])
                .map((scenario) => scenario?.name || scenario?.scenario_name)
                .filter(Boolean)))
                .sort((a, b) => a.localeCompare(b));

            if (names.length) {
                datalist.innerHTML = names.map((name) => `<option value="${escapeHtml(name)}"></option>`).join('');
                this.vaultScenarioOptionsLoaded = true;
            }
        } catch (error) {
            console.error('[VaultPage] Failed to load scenario options:', error);
        }
    }

    /**
     * Load vault list
     */
    async loadVaultList() {
        if (!this.vaultListContainer) {
            return;
        }

        if (this.debug) {
            console.log('[VaultPage] Loading vault list');
        }

        this.vaultListContainer.innerHTML = '<div class="loading-message">Loading vaults...</div>';

        try {
            const response = await this.apiClient.getTestVaults();
            const vaults = normalizeCollection(response, 'vaults');
            this.stateManager.setData('vaults', Array.isArray(vaults) ? vaults : []);

            const vaultsData = this.stateManager.get('data.vaults') || [];

            if (!vaultsData.length) {
                this.vaultListContainer.innerHTML = '<div class="empty-message">No vaults created yet</div>';
                this.vaultIndexById.clear();
                this.hideVaultDetails('Create a vault to populate execution data.');
                return;
            }

            this.renderVaultList(vaultsData);
        } catch (error) {
            console.error('[VaultPage] Failed to load vaults:', error);
            this.vaultListContainer.innerHTML = '<div class="error-message">Failed to load vaults</div>';
            this.hideVaultDetails('Unable to load vault details.');
        }
    }

    /**
     * Render vault list
     * @param {Array} vaults - Array of vaults
     * @private
     */
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
            const id = normalizeId(vault.id);
            this.vaultIndexById.set(id, vault);
            const descriptor = getStatusDescriptor(vault.status);
            const phaseCount = Array.isArray(vault.phases) ? vault.phases.length : 0;

            fragments.push(`
                <div class="vault-item" data-vault-id="${escapeHtml(id)}">
                    <div class="vault-item-header">
                        <span class="vault-item-name">${escapeHtml(vault.vault_name || id)}</span>
                        <span class="status-pill" data-status="${descriptor.key}"><i data-lucide="${descriptor.icon}"></i>${descriptor.label}</span>
                    </div>
                    <div class="vault-item-meta">
                        <span><i data-lucide="package"></i> ${escapeHtml(vault.scenario_name || 'Unknown')}</span>
                        <span><i data-lucide="layers"></i> ${phaseCount} phase${phaseCount === 1 ? '' : 's'}</span>
                        <span><i data-lucide="clock"></i> ${formatTimestamp(vault.created_at || vault.updated_at || Date.now())}</span>
                    </div>
                </div>
            `);
        });

        this.vaultListContainer.innerHTML = fragments.join('');
        const vaultPanelBody = this.vaultListContainer.parentElement;
        if (vaultPanelBody) {
            vaultPanelBody.scrollLeft = 0;
            enableDragScroll(vaultPanelBody);
        }
        refreshIcons();

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
            targetId = normalizeId(vaults[0].id);
        }

        if (targetId) {
            this.setActiveVaultItem(targetId);
            this.showVaultExecution(targetId);
        } else {
            this.hideVaultDetails('Select a vault to inspect execution details.');
        }
    }

    /**
     * Set active vault item in UI
     * @param {string} vaultId - Vault ID
     * @private
     */
    setActiveVaultItem(vaultId) {
        this.activeVaultId = vaultId;
        if (!this.vaultListContainer) {
            return;
        }
        this.vaultListContainer.querySelectorAll('.vault-item').forEach((item) => {
            item.classList.toggle('is-active', item.dataset.vaultId === vaultId);
        });
    }

    /**
     * Hide vault execution details
     * @param {string} message - Message to display
     * @private
     */
    hideVaultDetails(message = '') {
        if (this.vaultTimelineContainer && this.vaultTimelineContent) {
            if (message) {
                this.vaultTimelineContainer.style.display = 'block';
                this.vaultTimelineContent.innerHTML = `<div class="history-empty">${escapeHtml(message)}</div>`;
            } else {
                this.vaultTimelineContainer.style.display = 'none';
                this.vaultTimelineContent.innerHTML = '';
            }
        }

        if (this.vaultHistoryContainer && this.vaultHistoryContent) {
            if (message) {
                this.vaultHistoryContainer.style.display = 'block';
                this.vaultHistoryContent.innerHTML = `<div class="history-empty">${escapeHtml(message)}</div>`;
            } else {
                this.vaultHistoryContainer.style.display = 'none';
                this.vaultHistoryContent.innerHTML = '';
            }
        }
    }

    /**
     * Show vault execution details
     * @param {string} vaultId - Vault ID
     */
    async showVaultExecution(vaultId) {
        if (!vaultId || !this.vaultTimelineContent || !this.vaultHistoryContent) {
            return;
        }

        if (this.debug) {
            console.log('[VaultPage] Showing vault execution:', vaultId);
        }

        this.vaultTimelineContainer.style.display = 'block';
        this.vaultHistoryContainer.style.display = 'block';
        this.vaultTimelineContent.innerHTML = '<div class="loading-message">Loading execution details...</div>';
        this.vaultHistoryContent.innerHTML = '<div class="loading-message">Loading vault history...</div>';

        try {
            const [vaultDetails, executionsResponse] = await Promise.all([
                this.apiClient.getTestVault(vaultId),
                this.apiClient.getVaultExecutions(vaultId)
            ]);

            const vault = vaultDetails || this.vaultIndexById.get(vaultId) || null;
            const configuredPhases = Array.isArray(vault?.phases) ? vault.phases : [];

            const executionsRaw = normalizeCollection(executionsResponse, 'executions');
            let executions = Array.isArray(executionsRaw) ? executionsRaw : [];

            if (executions.length) {
                const normalizedId = vaultId.toLowerCase();
                const filtered = executions.filter((execution) => normalizeId(execution.vault_id).toLowerCase() === normalizedId);
                if (filtered.length) {
                    executions = filtered;
                }
            }

            if (!executions.length) {
                this.vaultTimelineContent.innerHTML = '<div class="history-empty">No vault executions yet. Trigger the vault to see progress.</div>';
                this.vaultHistoryContent.innerHTML = '<div class="history-empty">No execution history available.</div>';
                refreshIcons();
                return;
            }

            executions.sort((a, b) => {
                const aDate = parseDate(a.start_time) || new Date(0);
                const bDate = parseDate(b.start_time) || new Date(0);
                return bDate.getTime() - aDate.getTime();
            });

            const latestExecution = executions[0];
            let executionDetails = null;
            const latestExecutionId = normalizeId(latestExecution.id);
            if (latestExecutionId) {
                executionDetails = await this.apiClient.getVaultExecutionResults(latestExecutionId);
            }

            this.renderVaultTimeline(configuredPhases, executionDetails, latestExecution);
            this.renderVaultHistory(executions);
        } catch (error) {
            console.error('[VaultPage] Failed to load vault execution data:', error);
            this.vaultTimelineContent.innerHTML = '<div class="error-message">Failed to load execution details</div>';
            this.vaultHistoryContent.innerHTML = '<div class="error-message">Failed to load execution history</div>';
        } finally {
            refreshIcons();
        }
    }

    /**
     * Render vault timeline (phase execution progress)
     * @param {Array} phases - Configured phases
     * @param {Object} executionDetails - Execution details
     * @param {Object} latestExecution - Latest execution summary
     * @private
     */
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
            let descriptor = getStatusDescriptor(phaseResults[lowerPhase]?.status);

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
                detailParts.push(`Duration: ${formatDurationSeconds(duration)}`);
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
            const label = formatPhaseLabel(phaseName);

            return `
                <div class="timeline-phase">
                    <div class="timeline-indicator ${descriptor.key}">${iconHtml}</div>
                    <div class="timeline-content">
                        <div class="timeline-phase-name">${escapeHtml(label)}</div>
                        <div class="timeline-phase-status">${detailParts.map((part) => escapeHtml(part)).join(' • ')}</div>
                    </div>
                </div>
            `;
        }).join('');

        this.vaultTimelineContent.innerHTML = timelineHtml;
    }

    /**
     * Render vault execution history table
     * @param {Array} executions - Array of executions
     * @private
     */
    renderVaultHistory(executions) {
        if (!this.vaultHistoryContent) {
            return;
        }

        if (!Array.isArray(executions) || executions.length === 0) {
            this.vaultHistoryContent.innerHTML = '<div class="history-empty">No execution history available.</div>';
            return;
        }

        const rows = executions.map((execution, index) => {
            const descriptor = getStatusDescriptor(execution.status);
            const statusHtml = `<span class="status-pill" data-status="${descriptor.key}"><i data-lucide="${descriptor.icon}"></i>${descriptor.label}</span>`;

            let durationSeconds = Number(execution.duration);
            if (!Number.isFinite(durationSeconds)) {
                const start = parseDate(execution.start_time);
                const end = parseDate(execution.end_time);
                if (start && end) {
                    durationSeconds = (end.getTime() - start.getTime()) / 1000;
                }
            }

            const completedCount = Array.isArray(execution.completed_phases) ? execution.completed_phases.length : 0;
            const failedCount = Array.isArray(execution.failed_phases) ? execution.failed_phases.length : 0;

            return `
                <tr ${index === 0 ? 'data-latest="true"' : ''}>
                    <td>${escapeHtml(formatDateTime(execution.start_time))}</td>
                    <td>${statusHtml}</td>
                    <td>${completedCount}</td>
                    <td>${failedCount}</td>
                    <td>${escapeHtml(formatDurationSeconds(durationSeconds))}</td>
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

        refreshIcons();
    }

    /**
     * Ensure phase order from various sources
     * @param {Array} configuredPhases - Configured phases
     * @param {Object} executionDetails - Execution details
     * @param {Object} latestExecution - Latest execution
     * @returns {Array} Ordered phase names
     * @private
     */
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

    /**
     * Handle vault form submission
     */
    async handleVaultSubmit() {
        if (!this.vaultForm) {
            return;
        }

        if (this.debug) {
            console.log('[VaultPage] Submitting vault creation form');
        }

        const scenarioInput = document.getElementById('vault-scenario');
        const vaultNameInput = document.getElementById('vault-name');
        const totalTimeoutInput = document.getElementById('vault-total-timeout');
        const submitButton = this.vaultForm.querySelector('button[type="submit"]');

        const scenarioName = (scenarioInput?.value || '').trim();
        if (!scenarioName) {
            this.notificationManager.showError('Please provide a scenario name for the vault.');
            return;
        }

        if (this.vaultSelectedPhases.size === 0) {
            this.notificationManager.showError('Select at least one phase that must pass.');
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

        if (this.vaultResultCard) {
            this.vaultResultCard.classList.remove('visible');
            this.vaultResultCard.innerHTML = '';
        }

        if (submitButton) {
            submitButton.disabled = true;
            submitButton.dataset.originalLabel = submitButton.innerHTML;
            submitButton.innerHTML = '<div class="spinner"></div> Creating...';
        }

        try {
            const result = await this.apiClient.createTestVault({
                scenario_name: scenarioName,
                vault_name: vaultName,
                phases,
                phase_configurations: phaseConfigs,
                success_criteria: successCriteria,
                total_timeout: totalTimeout
            });

            if (this.vaultResultCard) {
                const phaseLabels = phases.map((phase) => formatPhaseLabel(phase));
                const executionHint = result?.vault_id
                    ? `POST /api/v1/test-vault/${result.vault_id}/execute`
                    : 'POST /api/v1/test-vault/{vault_id}/execute';

                this.vaultResultCard.innerHTML = `
                    <div style="font-weight:600; margin-bottom: 0.5rem;">Vault created successfully.</div>
                    <div><strong>Name:</strong> ${escapeHtml(vaultName)}</div>
                    <div><strong>Scenario:</strong> ${escapeHtml(scenarioName)}</div>
                    <div><strong>Phases:</strong> ${phaseLabels.map((label) => escapeHtml(label)).join(', ')}</div>
                    <div><strong>Total timeout:</strong> ${formatDurationSeconds(totalTimeout)}</div>
                    <div style="margin-top:0.75rem; font-size:0.8rem; color: var(--text-secondary);">
                        Execute via <code>${escapeHtml(executionHint)}</code>
                    </div>
                `;
                this.vaultResultCard.classList.add('visible');
            }

            this.notificationManager.showSuccess(`Vault ready with ${phases.length} phase${phases.length === 1 ? '' : 's'}.`);

            this.vaultForm.reset();
            this.vaultPhaseDrafts.clear();
            this.resetVaultPhaseSelection();
            this.vaultScenarioOptionsLoaded = false;
            await this.loadVaultScenarioOptions(true);
            await this.loadVaultList();
        } catch (error) {
            console.error('[VaultPage] Test vault creation failed:', error);
            this.notificationManager.showError(`Test vault creation failed: ${error.message}`);
        } finally {
            if (submitButton) {
                submitButton.disabled = false;
                submitButton.innerHTML = submitButton.dataset.originalLabel || 'Create Vault';
                delete submitButton.dataset.originalLabel;
            }
        }
    }

    /**
     * Refresh vault page data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Vaults refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[VaultPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            vaultInitialized: this.vaultInitialized,
            selectedPhasesCount: this.vaultSelectedPhases.size,
            phaseDraftsCount: this.vaultPhaseDrafts.size,
            vaultsCount: (this.stateManager.get('data.vaults') || []).length,
            activeVaultId: this.activeVaultId
        };
    }
}

// Export singleton instance
export const vaultPage = new VaultPage();

// Export default for convenience
export default vaultPage;
