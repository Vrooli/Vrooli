/**
 * DialogManager - Dialog and Overlay Management
 * Handles opening/closing dialogs, modal overlays, and focus management
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { focusElement, lockDialogScroll, unlockDialogScrollIfIdle } from '../utils/domHelpers.js';

/**
 * DialogManager class - Manages dialogs and overlays
 */
export class DialogManager {
    constructor(eventBusInstance = eventBus, stateManagerInstance = stateManager) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;

        // Dialog references
        this.dialogs = {
            generate: null,
            vault: null,
            health: null,
            coverageDetail: null,
            suiteDetail: null,
            executionDetail: null
        };

        // Dialog state tracking
        this.lastTriggers = {
            generate: null,
            vault: null,
            health: null
        };

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize dialog manager
     */
    initialize() {
        this.setupDialogReferences();
        this.setupDialogListeners();
        this.setupEscapeKeyListener();

        if (this.debug) {
            console.log('[DialogManager] Dialog system initialized');
        }
    }

    /**
     * Setup dialog DOM references
     * @private
     */
    setupDialogReferences() {
        this.dialogs.generate = document.getElementById('generate-dialog-overlay');
        this.dialogs.vault = document.getElementById('vault-dialog-overlay');
        this.dialogs.health = document.getElementById('health-status-overlay');
        this.dialogs.coverageDetail = document.getElementById('coverage-detail-overlay');
        this.dialogs.suiteDetail = document.getElementById('suite-detail');
        this.dialogs.executionDetail = document.getElementById('execution-detail');
    }

    /**
     * Setup dialog event listeners
     * @private
     */
    setupDialogListeners() {
        // Generate Dialog
        if (this.dialogs.generate) {
            this.dialogs.generate.addEventListener('click', (e) => {
                if (e.target === this.dialogs.generate) {
                    this.closeGenerateDialog();
                }
            });

            const closeBtn = document.getElementById('generate-dialog-close');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => this.closeGenerateDialog());
            }
        }

        // Vault Dialog
        if (this.dialogs.vault) {
            this.dialogs.vault.addEventListener('click', (e) => {
                if (e.target === this.dialogs.vault) {
                    this.closeVaultDialog();
                }
            });

            const closeBtn = document.getElementById('vault-dialog-close');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => this.closeVaultDialog());
            }

            const openBtn = document.getElementById('vault-dialog-open');
            if (openBtn) {
                openBtn.addEventListener('click', (e) => {
                    e.preventDefault();
                    this.openVaultDialog(openBtn);
                });
            }
        }

        // Health Dialog
        if (this.dialogs.health) {
            this.dialogs.health.addEventListener('click', (e) => {
                if (e.target === this.dialogs.health) {
                    this.closeHealthDialog();
                }
            });

            const closeBtn = document.getElementById('health-status-close');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => this.closeHealthDialog());
            }
        }

        // Listen to event bus for dialog control
        this.setupEventBusListeners();
    }

    /**
     * Setup event bus listeners for dialog control
     * @private
     */
    setupEventBusListeners() {
        this.eventBus.on(EVENT_TYPES.UI_DIALOG_OPEN, (event) => {
            const { dialogType, trigger, options } = event.data;
            this.openDialog(dialogType, trigger, options);
        });

        this.eventBus.on(EVENT_TYPES.UI_DIALOG_CLOSE, (event) => {
            const { dialogType } = event.data;
            this.closeDialog(dialogType);
        });

        this.eventBus.on(EVENT_TYPES.UI_SUITE_DETAIL_CLOSE, () => {
            // Handle suite detail close if needed
            this.stateManager.set('ui.activeSuiteDetailId', null);
        });

        this.eventBus.on(EVENT_TYPES.UI_EXECUTION_DETAIL_CLOSE, () => {
            // Handle execution detail close if needed
            this.stateManager.set('ui.activeExecutionDetailId', null);
        });
    }

    /**
     * Setup global escape key listener
     * @private
     */
    setupEscapeKeyListener() {
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                // Close dialogs in priority order
                if (this.isDialogOpen('suiteDetail')) {
                    e.preventDefault();
                    this.eventBus.emit(EVENT_TYPES.UI_SUITE_DETAIL_CLOSE);
                    return;
                }

                if (this.isDialogOpen('executionDetail')) {
                    e.preventDefault();
                    this.eventBus.emit(EVENT_TYPES.UI_EXECUTION_DETAIL_CLOSE);
                    return;
                }

                if (this.isDialogOpen('coverageDetail')) {
                    e.preventDefault();
                    this.closeCoverageDetailDialog();
                    return;
                }

                if (this.isDialogOpen('generate')) {
                    e.preventDefault();
                    this.closeGenerateDialog();
                    return;
                }

                if (this.isDialogOpen('vault')) {
                    e.preventDefault();
                    this.closeVaultDialog();
                    return;
                }

                if (this.isDialogOpen('health')) {
                    e.preventDefault();
                    this.closeHealthDialog();
                    return;
                }
            }
        });
    }

    /**
     * Generic open dialog method
     * @param {string} dialogType - Dialog type to open
     * @param {HTMLElement} trigger - Triggering element
     * @param {Object} options - Additional options
     */
    openDialog(dialogType, trigger = null, options = {}) {
        const methodMap = {
            generate: () => this.openGenerateDialog(trigger, options.scenarioName, options.isBulkMode),
            vault: () => this.openVaultDialog(trigger),
            health: () => this.openHealthDialog()
        };

        if (methodMap[dialogType]) {
            methodMap[dialogType]();
        } else {
            console.warn(`[DialogManager] Unknown dialog type: ${dialogType}`);
        }
    }

    /**
     * Generic close dialog method
     * @param {string} dialogType - Dialog type to close
     */
    closeDialog(dialogType) {
        const methodMap = {
            generate: () => this.closeGenerateDialog(),
            vault: () => this.closeVaultDialog(),
            health: () => this.closeHealthDialog(),
            coverageDetail: () => this.closeCoverageDetailDialog()
        };

        if (methodMap[dialogType]) {
            methodMap[dialogType]();
        } else {
            console.warn(`[DialogManager] Unknown dialog type: ${dialogType}`);
        }
    }

    /**
     * Open Generate Test Suite dialog
     * @param {HTMLElement} triggerElement - Triggering element
     * @param {string} scenarioName - Pre-filled scenario name
     * @param {boolean} isBulkMode - Whether this is bulk generation mode
     */
    openGenerateDialog(triggerElement = null, scenarioName = '', isBulkMode = false) {
        if (!this.dialogs.generate) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Opening generate dialog', { scenarioName, isBulkMode });
        }

        this.lastTriggers.generate = triggerElement;
        this.dialogs.generate.classList.add('active');
        this.dialogs.generate.setAttribute('aria-hidden', 'false');
        lockDialogScroll();

        // Update state
        this.stateManager.set('ui.generateDialogOpen', true);
        this.stateManager.set('ui.generateDialogScenarioName', scenarioName);
        this.stateManager.set('ui.generateDialogBulkMode', isBulkMode);

        // Update dialog title and button based on mode
        const dialogTitle = document.getElementById('generate-dialog-title');
        const generateBtn = document.getElementById('generate-btn');

        if (isBulkMode) {
            if (dialogTitle) {
                dialogTitle.textContent = 'Bulk Request Tests';
            }
            if (generateBtn) {
                generateBtn.innerHTML = '<i data-lucide="sparkles"></i> <span id="generate-btn-text">Create Requests</span>';
            }
        } else {
            if (dialogTitle) {
                dialogTitle.textContent = 'Request Test Suite';
            }
            if (generateBtn) {
                generateBtn.innerHTML = '<i data-lucide="zap"></i> <span id="generate-btn-text">Create Request</span>';
            }
        }

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_OPENED, {
            dialogType: 'generate',
            scenarioName,
            isBulkMode
        });

        // Focus first input
        const firstCheckbox = document.querySelector('#generate-phase-selector input[type="checkbox"]');
        const coverageInput = document.getElementById('coverage-target');
        focusElement(firstCheckbox || coverageInput);
    }

    /**
     * Close Generate Test Suite dialog
     */
    closeGenerateDialog() {
        if (!this.dialogs.generate) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Closing generate dialog');
        }

        this.dialogs.generate.classList.remove('active');
        this.dialogs.generate.setAttribute('aria-hidden', 'true');
        unlockDialogScrollIfIdle(Object.values(this.dialogs));

        // Update state
        this.stateManager.set('ui.generateDialogOpen', false);

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_CLOSED, {
            dialogType: 'generate'
        });

        // Return focus to trigger
        if (this.lastTriggers.generate) {
            focusElement(this.lastTriggers.generate);
            this.lastTriggers.generate = null;
        }
    }

    /**
     * Open Vault Creation dialog
     * @param {HTMLElement} triggerElement - Triggering element
     */
    openVaultDialog(triggerElement = null) {
        if (!this.dialogs.vault) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Opening vault dialog');
        }

        this.lastTriggers.vault = triggerElement;
        this.dialogs.vault.classList.add('active');
        this.dialogs.vault.setAttribute('aria-hidden', 'false');
        lockDialogScroll();

        // Update state
        this.stateManager.set('ui.vaultDialogOpen', true);

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_OPENED, {
            dialogType: 'vault'
        });

        // Focus scenario input
        const scenarioInput = document.getElementById('vault-scenario');
        if (scenarioInput) {
            focusElement(scenarioInput);
        }
    }

    /**
     * Close Vault Creation dialog
     */
    closeVaultDialog() {
        if (!this.dialogs.vault) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Closing vault dialog');
        }

        this.dialogs.vault.classList.remove('active');
        this.dialogs.vault.setAttribute('aria-hidden', 'true');
        unlockDialogScrollIfIdle(Object.values(this.dialogs));

        // Update state
        this.stateManager.set('ui.vaultDialogOpen', false);

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_CLOSED, {
            dialogType: 'vault'
        });

        // Return focus to trigger
        if (this.lastTriggers.vault) {
            focusElement(this.lastTriggers.vault);
            this.lastTriggers.vault = null;
        }
    }

    /**
     * Open Health Status dialog
     */
    async openHealthDialog() {
        if (!this.dialogs.health) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Opening health dialog');
        }

        const systemStatusChip = document.getElementById('system-status-chip');
        this.lastTriggers.health = systemStatusChip || document.activeElement;

        this.dialogs.health.classList.add('active');
        this.dialogs.health.setAttribute('aria-hidden', 'false');

        if (systemStatusChip) {
            systemStatusChip.setAttribute('aria-expanded', 'true');
        }

        lockDialogScroll();

        // Update state
        this.stateManager.set('ui.healthDialogOpen', true);

        // Emit event to request health data refresh
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_OPENED, {
            dialogType: 'health'
        });

        // Request fresh health data
        this.eventBus.emit(EVENT_TYPES.HEALTH_CHECK_REQUESTED);
    }

    /**
     * Close Health Status dialog
     */
    closeHealthDialog() {
        if (!this.dialogs.health) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Closing health dialog');
        }

        this.dialogs.health.classList.remove('active');
        this.dialogs.health.setAttribute('aria-hidden', 'true');

        const systemStatusChip = document.getElementById('system-status-chip');
        if (systemStatusChip) {
            systemStatusChip.setAttribute('aria-expanded', 'false');
        }

        unlockDialogScrollIfIdle(Object.values(this.dialogs));

        // Update state
        this.stateManager.set('ui.healthDialogOpen', false);

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_CLOSED, {
            dialogType: 'health'
        });

        // Return focus to trigger
        if (this.lastTriggers.health && typeof this.lastTriggers.health.focus === 'function') {
            focusElement(this.lastTriggers.health);
        }
        this.lastTriggers.health = null;
    }

    /**
     * Close Coverage Detail dialog
     */
    closeCoverageDetailDialog() {
        if (!this.dialogs.coverageDetail) {
            return;
        }

        if (this.debug) {
            console.log('[DialogManager] Closing coverage detail dialog');
        }

        this.dialogs.coverageDetail.classList.remove('active');
        this.dialogs.coverageDetail.setAttribute('aria-hidden', 'true');
        unlockDialogScrollIfIdle(Object.values(this.dialogs));

        // Update state
        this.stateManager.set('ui.coverageDetailOpen', false);

        // Emit event
        this.eventBus.emit(EVENT_TYPES.UI_DIALOG_CLOSED, {
            dialogType: 'coverageDetail'
        });
    }

    /**
     * Check if a dialog is currently open
     * @param {string} dialogType - Dialog type to check
     * @returns {boolean}
     */
    isDialogOpen(dialogType) {
        const dialog = this.dialogs[dialogType];
        return dialog ? dialog.classList.contains('active') : false;
    }

    /**
     * Check if any dialog is open
     * @returns {boolean}
     */
    isAnyDialogOpen() {
        return Object.values(this.dialogs).some(dialog =>
            dialog && dialog.classList.contains('active')
        );
    }

    /**
     * Close all dialogs
     */
    closeAllDialogs() {
        if (this.debug) {
            console.log('[DialogManager] Closing all dialogs');
        }

        Object.keys(this.dialogs).forEach(dialogType => {
            if (this.isDialogOpen(dialogType)) {
                this.closeDialog(dialogType);
            }
        });
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[DialogManager] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        const openDialogs = Object.keys(this.dialogs)
            .filter(type => this.isDialogOpen(type));

        return {
            openDialogs,
            dialogsRegistered: Object.keys(this.dialogs).filter(type => this.dialogs[type] !== null),
            anyDialogOpen: this.isAnyDialogOpen()
        };
    }
}

// Export singleton instance
export const dialogManager = new DialogManager();

// Export default for convenience
export default dialogManager;
