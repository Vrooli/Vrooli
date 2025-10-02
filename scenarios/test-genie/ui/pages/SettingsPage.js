/**
 * SettingsPage - Application Settings View
 * Handles loading, saving, and managing default settings for test generation
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { notificationManager } from '../managers/NotificationManager.js';
import { normalizeCoverageTarget, normalizePhaseList } from '../utils/validators.js';
import { DEFAULT_GENERATION_PHASES, STORAGE_KEYS } from '../utils/constants.js';

/**
 * SettingsPage class - Manages application settings
 */
export class SettingsPage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        notificationManagerInstance = notificationManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.notificationManager = notificationManagerInstance;

        // DOM references
        this.settingsCoverageSlider = null;
        this.settingsCoverageDisplay = null;
        this.settingsPhaseCheckboxes = [];
        this.settingsSaveButton = null;

        // Settings state
        this.defaultSettings = {
            coverageTarget: 80,
            phases: new Set(DEFAULT_GENERATION_PHASES)
        };

        this.pendingSettings = {
            coverageTarget: 80,
            phases: new Set(DEFAULT_GENERATION_PHASES)
        };

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize settings page
     */
    initialize() {
        this.setupEventListeners();

        if (this.debug) {
            console.log('[SettingsPage] Settings page initialized');
        }
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'settings') {
                this.load();
            }
        });
    }

    /**
     * Load settings page
     */
    async load() {
        if (this.debug) {
            console.log('[SettingsPage] Loading settings page');
        }

        try {
            // Set loading state
            this.stateManager.setLoading('settings', true);

            // Load stored settings
            const stored = this.loadStoredDefaultSettings();
            this.defaultSettings.coverageTarget = stored.coverageTarget;
            this.defaultSettings.phases = new Set(stored.phases);
            this.pendingSettings.coverageTarget = stored.coverageTarget;
            this.pendingSettings.phases = new Set(stored.phases);

            // Setup DOM references
            this.setupDOMReferences();

            // Apply settings to UI
            this.applyDefaultSettingsToSettingsControls();

            // Setup event handlers
            this.setupSettingsEventHandlers();

            // Apply default settings to generate dialog
            this.applyDefaultSettingsToGenerateDialog(true);

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'settings' });

        } catch (error) {
            console.error('[SettingsPage] Failed to load settings:', error);
            this.notificationManager.showError('Failed to load settings');
        } finally {
            this.stateManager.setLoading('settings', false);
        }
    }

    /**
     * Setup DOM element references
     * @private
     */
    setupDOMReferences() {
        this.settingsCoverageSlider = document.getElementById('settings-default-coverage');
        this.settingsCoverageDisplay = document.getElementById('settings-default-coverage-display');
        this.settingsPhaseCheckboxes = Array.from(document.querySelectorAll('#settings-default-phases input[type="checkbox"][data-phase]'));
        this.settingsSaveButton = document.getElementById('settings-save-btn');
    }

    /**
     * Setup settings event handlers
     * @private
     */
    setupSettingsEventHandlers() {
        // Coverage slider handler
        if (this.settingsCoverageSlider) {
            const handleSliderInput = () => {
                const value = normalizeCoverageTarget(this.settingsCoverageSlider.value, this.pendingSettings.coverageTarget);
                this.pendingSettings.coverageTarget = value;
                if (this.settingsCoverageSlider.value !== String(value)) {
                    this.settingsCoverageSlider.value = String(value);
                }
                this.updateSettingsCoverageDisplay(value);
            };
            this.settingsCoverageSlider.addEventListener('input', handleSliderInput);
            this.settingsCoverageSlider.addEventListener('change', handleSliderInput);
        }

        // Phase checkboxes handler
        if (this.settingsPhaseCheckboxes.length) {
            this.settingsPhaseCheckboxes.forEach((checkbox) => {
                checkbox.addEventListener('change', () => {
                    const phase = checkbox.dataset.phase;
                    if (!phase) {
                        return;
                    }
                    if (checkbox.checked) {
                        this.pendingSettings.phases.add(phase);
                    } else {
                        this.pendingSettings.phases.delete(phase);
                    }
                });
            });
        }

        // Save button handler
        if (this.settingsSaveButton) {
            this.settingsSaveButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.saveDefaultSettings();
            });
        }
    }

    /**
     * Apply default settings to settings controls
     * @private
     */
    applyDefaultSettingsToSettingsControls() {
        const coverageValue = this.pendingSettings.coverageTarget;
        if (this.settingsCoverageSlider) {
            this.settingsCoverageSlider.value = String(coverageValue);
        }
        this.updateSettingsCoverageDisplay(coverageValue);

        if (this.settingsPhaseCheckboxes.length) {
            this.settingsPhaseCheckboxes.forEach((checkbox) => {
                const phase = checkbox.dataset.phase;
                checkbox.checked = phase ? this.pendingSettings.phases.has(phase) : false;
            });
        }
    }

    /**
     * Update settings coverage display
     * @param {number} value - Coverage target value
     * @private
     */
    updateSettingsCoverageDisplay(value) {
        if (this.settingsCoverageDisplay) {
            this.settingsCoverageDisplay.textContent = `${value}%`;
        }
    }

    /**
     * Load stored default settings from localStorage
     * @returns {Object} Settings object
     * @private
     */
    loadStoredDefaultSettings() {
        try {
            const raw = localStorage.getItem(STORAGE_KEYS.DEFAULT_SETTINGS);
            if (!raw) {
                return {
                    coverageTarget: 80,
                    phases: DEFAULT_GENERATION_PHASES.slice()
                };
            }

            const parsed = JSON.parse(raw);
            const coverageTarget = normalizeCoverageTarget(parsed?.coverageTarget, 80);
            const phases = normalizePhaseList(parsed?.phases);
            return {
                coverageTarget,
                phases
            };
        } catch (error) {
            console.warn('[SettingsPage] Failed to parse stored default settings:', error);
            return {
                coverageTarget: 80,
                phases: DEFAULT_GENERATION_PHASES.slice()
            };
        }
    }

    /**
     * Save default settings
     */
    saveDefaultSettings() {
        if (this.pendingSettings.phases.size === 0) {
            this.notificationManager.showError('Select at least one default phase.');
            return;
        }

        if (this.debug) {
            console.log('[SettingsPage] Saving default settings');
        }

        this.defaultSettings.coverageTarget = this.pendingSettings.coverageTarget;
        this.defaultSettings.phases = new Set(this.pendingSettings.phases);
        this.persistDefaultSettings();
        this.applyDefaultSettingsToSettingsControls();
        this.applyDefaultSettingsToGenerateDialog(true);
        this.notificationManager.showSuccess('Default settings updated');

        // Emit settings updated event
        this.eventBus.emit(EVENT_TYPES.SETTINGS_UPDATED, {
            coverageTarget: this.defaultSettings.coverageTarget,
            phases: Array.from(this.defaultSettings.phases)
        });
    }

    /**
     * Persist default settings to localStorage
     * @private
     */
    persistDefaultSettings() {
        try {
            const payload = {
                coverageTarget: this.defaultSettings.coverageTarget,
                phases: Array.from(this.defaultSettings.phases)
            };
            localStorage.setItem(STORAGE_KEYS.DEFAULT_SETTINGS, JSON.stringify(payload));
        } catch (error) {
            console.warn('[SettingsPage] Failed to persist default settings:', error);
        }
    }

    /**
     * Apply default settings to generate dialog
     * @param {boolean} updateAttribute - Whether to update HTML attributes
     * @private
     */
    applyDefaultSettingsToGenerateDialog(updateAttribute = false) {
        const coverageTargetInput = document.getElementById('coverage-target');
        if (coverageTargetInput) {
            const coverage = this.defaultSettings.coverageTarget;
            coverageTargetInput.value = String(coverage);
            if (updateAttribute) {
                coverageTargetInput.setAttribute('value', String(coverage));
            }
        }

        const checkboxes = document.querySelectorAll('#generate-phase-selector input[type="checkbox"]');
        if (checkboxes.length) {
            checkboxes.forEach((checkbox) => {
                const phase = String(checkbox.value || '').trim().toLowerCase();
                checkbox.checked = this.defaultSettings.phases.has(phase);
            });
        }

        // Update coverage target display in generate dialog
        this.updateCoverageTargetDisplay();
    }

    /**
     * Update coverage target display in generate dialog
     * @private
     */
    updateCoverageTargetDisplay() {
        const coverageTargetInput = document.getElementById('coverage-target');
        const coverageTargetDisplay = document.getElementById('coverage-target-display');

        if (coverageTargetInput && coverageTargetDisplay) {
            const value = normalizeCoverageTarget(coverageTargetInput.value, this.defaultSettings.coverageTarget);
            coverageTargetDisplay.textContent = `${value}%`;
        }
    }

    /**
     * Get current default settings
     * @returns {Object} Settings object
     */
    getDefaultSettings() {
        return {
            coverageTarget: this.defaultSettings.coverageTarget,
            phases: Array.from(this.defaultSettings.phases)
        };
    }

    /**
     * Reset settings to defaults
     */
    resetToDefaults() {
        if (this.debug) {
            console.log('[SettingsPage] Resetting to default settings');
        }

        this.pendingSettings.coverageTarget = 80;
        this.pendingSettings.phases = new Set(DEFAULT_GENERATION_PHASES);
        this.applyDefaultSettingsToSettingsControls();
        this.notificationManager.showInfo('Settings reset to defaults (not saved)');
    }

    /**
     * Refresh settings page
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Settings refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[SettingsPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            defaultSettings: {
                coverageTarget: this.defaultSettings.coverageTarget,
                phases: Array.from(this.defaultSettings.phases)
            },
            pendingSettings: {
                coverageTarget: this.pendingSettings.coverageTarget,
                phases: Array.from(this.pendingSettings.phases)
            },
            hasSettingsCoverageSlider: this.settingsCoverageSlider !== null,
            settingsPhaseCheckboxesCount: this.settingsPhaseCheckboxes.length
        };
    }
}

// Export singleton instance
export const settingsPage = new SettingsPage();

// Export default for convenience
export default settingsPage;
