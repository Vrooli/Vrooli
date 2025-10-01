// Settings Management Module
export class SettingsManager {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.settings = null;
        this.originalTheme = null; // Track original theme for preview cancellation
        this.recyclerModelCache = {};
        this.recyclerModelRequests = {};
        this.defaultSettings = {
            theme: 'dark',
            slots: 1,
            refresh_interval: 30,
            active: false,
            max_turns: 60,
            allowed_tools: 'Read,Write,Edit,Bash,LS,Glob,Grep',
            skip_permissions: true,
            task_timeout: 30,
            condensed_mode: false,
            recycler: {
                enabled_for: 'off',
                interval_seconds: 60,
                model_provider: 'ollama',
                model_name: 'llama3.1:8b',
                completion_threshold: 3,
                failure_threshold: 5
            }
        };
    }

    async loadSettings() {
        try {
            const response = await fetch(`${this.apiBase}/settings`);
            if (!response.ok) {
                throw new Error(`Failed to load settings: ${response.statusText}`);
            }
            
            const data = await response.json();
            const defaults = JSON.parse(JSON.stringify(this.defaultSettings));
            const incoming = data.settings || {};
            this.settings = {
                ...defaults,
                ...incoming,
                recycler: {
                    ...defaults.recycler,
                    ...(incoming.recycler || {})
                }
            };
            return this.settings;
        } catch (error) {
            console.error('Failed to load settings:', error);
            this.settings = JSON.parse(JSON.stringify(this.defaultSettings));
            return this.settings;
        }
    }

    async saveSettings(settings) {
        try {
            // Debug logging
            console.log('Saving settings:', settings);
            
            const response = await fetch(`${this.apiBase}/settings`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(settings)
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Settings save failed:', errorText);
                throw new Error(`Failed to save settings: ${errorText}`);
            }

            const result = await response.json();
            this.settings = settings;
            return result;
        } catch (error) {
            console.error('Failed to save settings:', error);
            throw error;
        }
    }

    applySettingsToUI(settings) {
        const cachedThemePreference = localStorage.getItem('ecosystemManager_theme');
        const resolvedTheme = cachedThemePreference || settings.theme || 'dark';
        settings.theme = resolvedTheme;

        // Map settings keys to form field IDs (using backend field names)
        const fieldMapping = {
            'theme': 'settings-theme',
            'condensed_mode': 'settings-condensed-mode',
            'slots': 'settings-slots',
            'refresh_interval': 'settings-refresh',
            'active': 'settings-active',
            'max_turns': 'settings-max-turns',
            'allowed_tools': 'settings-tools',
            'skip_permissions': 'settings-skip-permissions',
            'task_timeout': 'settings-task-timeout'
        };
        
        // Update form fields with settings values
        Object.entries(fieldMapping).forEach(([settingKey, fieldId]) => {
            const element = document.getElementById(fieldId);
            if (element && settings[settingKey] !== undefined) {
                if (element.type === 'checkbox') {
                    element.checked = settings[settingKey];
                } else if (element.type === 'range' || element.type === 'number') {
                    let value = settings[settingKey];
                    // Task timeout is already in minutes from backend, no conversion needed
                    // (previously we incorrectly assumed it was in seconds)
                    element.value = value;
                    // Update slider value display if present
                    const valueDisplay = document.getElementById(fieldId.replace('settings-', '') + '-value');
                    if (valueDisplay) {
                        valueDisplay.textContent = value;
                    }
                } else {
                    element.value = settings[settingKey] || '';
                }
            }
        });

        // Apply recycler-specific settings
        this.applyRecyclerSettings(settings.recycler || this.defaultSettings.recycler);

        // Update queue processor status UI
        this.updateProcessorToggleUI(settings.active);

        // Apply theme to body element
        this.applyTheme(resolvedTheme, { markUserPreference: false });
        
        // Store the original theme for potential reversion
        this.originalTheme = resolvedTheme;
    }

    updateProcessorToggleUI(isActive) {
        // Update status text and icon
        const statusText = document.getElementById('processor-status');
        if (statusText) {
            statusText.textContent = isActive ? 'Active' : 'Paused';
        }
        
        const statusIcon = document.getElementById('processor-status-icon');
        if (statusIcon) {
            statusIcon.className = isActive ? 'fas fa-play' : 'fas fa-pause';
        }

        const toggleButton = document.getElementById('processor-toggle-btn');
        if (toggleButton) {
            const stateLabel = isActive ? 'active' : 'paused';
            toggleButton.setAttribute('aria-label', `Queue processor ${stateLabel}. Tap to ${isActive ? 'pause' : 'activate'}.`);
            toggleButton.title = `Processor ${stateLabel}`;
        }

        // Show/hide additional indicators based on processor state
        const queueTimer = document.querySelector('.queue-timer');
        if (queueTimer) {
            queueTimer.style.display = isActive ? 'flex' : 'none';
        }
        
        const queueSlotsDiv = document.querySelector('.queue-slots');
        if (queueSlotsDiv) {
            queueSlotsDiv.style.display = isActive ? 'flex' : 'none';
        }
    }

    applyRecyclerSettings(recyclerSettings = {}) {
        const enabledSelect = document.getElementById('settings-recycler-enabled');
        if (enabledSelect) {
            enabledSelect.value = recyclerSettings.enabled_for || 'off';
        }

        const intervalInput = document.getElementById('settings-recycler-interval');
        if (intervalInput) {
            intervalInput.value = recyclerSettings.interval_seconds ?? this.defaultSettings.recycler.interval_seconds;
        }

        const providerSelect = document.getElementById('settings-recycler-model-provider');
        const providerValue = recyclerSettings.model_provider || 'ollama';
        if (providerSelect) {
            providerSelect.value = providerValue;
        }
        this.populateRecyclerModelSelector(providerValue, recyclerSettings.model_name || '').catch(err => {
            console.error('Failed to populate recycler models:', err);
        });

        const completionInput = document.getElementById('settings-recycler-completion-threshold');
        if (completionInput) {
            completionInput.value = recyclerSettings.completion_threshold ?? this.defaultSettings.recycler.completion_threshold;
        }

        const failureInput = document.getElementById('settings-recycler-failure-threshold');
        if (failureInput) {
            failureInput.value = recyclerSettings.failure_threshold ?? this.defaultSettings.recycler.failure_threshold;
        }
    }

    getSettingsFromForm() {
        const formData = {};
        
        // Map form field IDs to settings keys (using backend field names)
        const fieldMapping = {
            'settings-theme': 'theme',
            'settings-condensed-mode': 'condensed_mode',
            'settings-slots': 'slots',
            'settings-refresh': 'refresh_interval',
            'settings-active': 'active',
            'settings-max-turns': 'max_turns',
            'settings-tools': 'allowed_tools',
            'settings-skip-permissions': 'skip_permissions',
            'settings-task-timeout': 'task_timeout'
        };
        
        // Collect settings from form using mapped field IDs
        Object.entries(fieldMapping).forEach(([fieldId, settingKey]) => {
            const element = document.getElementById(fieldId);
            if (element) {
                if (element.type === 'checkbox') {
                    formData[settingKey] = element.checked;
                } else if (element.type === 'range' || element.type === 'number') {
                    let value = parseInt(element.value) || 0;
                    // Ensure slots is at least 1
                    if (settingKey === 'slots' && value < 1) {
                        value = 1;
                    }
                    // Convert task timeout from minutes to seconds BEFORE setting
                    if (settingKey === 'task_timeout') {
                        // Backend expects minutes, but let's make sure we have a valid value
                        if (value < 5) value = 5;
                        if (value > 240) value = 240;
                    }
                    formData[settingKey] = value;
                } else {
                    formData[settingKey] = element.value;
                }
                console.log(`Collected ${settingKey}: ${formData[settingKey]} (from element ${fieldId})`);
            } else {
                console.warn(`Element not found: ${fieldId}`);
            }
        });

        console.log('Form data collected:', formData);

        const recyclerDefaults = this.defaultSettings.recycler;
        const modelSelect = document.getElementById('settings-recycler-model-name');
        let selectedModel = modelSelect?.value || '';
        if (selectedModel === '__custom__') {
            selectedModel = document.getElementById('settings-recycler-model-name-custom')?.value?.trim() || '';
        }
        const recycler = {
            enabled_for: document.getElementById('settings-recycler-enabled')?.value || recyclerDefaults.enabled_for,
            interval_seconds: parseInt(document.getElementById('settings-recycler-interval')?.value, 10) || recyclerDefaults.interval_seconds,
            model_provider: document.getElementById('settings-recycler-model-provider')?.value || recyclerDefaults.model_provider,
            model_name: selectedModel || recyclerDefaults.model_name,
            completion_threshold: parseInt(document.getElementById('settings-recycler-completion-threshold')?.value, 10) || recyclerDefaults.completion_threshold,
            failure_threshold: parseInt(document.getElementById('settings-recycler-failure-threshold')?.value, 10) || recyclerDefaults.failure_threshold
        };

        // Clamp numeric values to expected ranges
        recycler.interval_seconds = Math.min(Math.max(recycler.interval_seconds, 30), 1800);
        recycler.completion_threshold = Math.min(Math.max(recycler.completion_threshold, 1), 10);
        recycler.failure_threshold = Math.min(Math.max(recycler.failure_threshold, 1), 10);

        formData.recycler = recycler;

        return formData;
    }

    applyTheme(theme, options = {}) {
        const { markUserPreference = false } = options;
        const body = document.body;
        if (theme === 'dark') {
            body.classList.add('dark-mode');
        } else {
            body.classList.remove('dark-mode');
        }
        
        // Cache the theme in localStorage for immediate application on next load
        localStorage.setItem('ecosystemManager_theme', theme);
        if (markUserPreference) {
            localStorage.setItem('ecosystemManager_theme_user_choice', 'true');
        }
        console.log(`Theme applied: ${theme}, body classes:`, body.className);
    }

    // Static method to ensure cached theme is applied (redundant check)
    static applyCachedTheme() {
        const cachedTheme = localStorage.getItem('ecosystemManager_theme');
        const userPreferenceSet = localStorage.getItem('ecosystemManager_theme_user_choice') === 'true';
        let themeToApply = cachedTheme || 'dark';

        if (!userPreferenceSet) {
            themeToApply = 'dark';
            localStorage.setItem('ecosystemManager_theme', themeToApply);
        }

        // Apply proper dark mode class and remove inline styles
        if (themeToApply === 'dark' || window.applyDarkModeOnLoad) {
            document.body.classList.add('dark-mode');
            // Remove the temporary inline style if it exists
            const tempStyle = document.querySelector('head style');
            if (tempStyle && tempStyle.textContent.includes('background-color: #1a1a1a')) {
                tempStyle.remove();
            }
        } else {
            document.body.classList.remove('dark-mode');
        }
        console.log(`Cached theme verified on load: ${themeToApply}`);
    }

    revertThemePreview() {
        // Revert to the original theme if user cancels
        if (this.originalTheme) {
            this.applyTheme(this.originalTheme, { markUserPreference: false });
        }
    }

    async fetchRecyclerModels(provider) {
        const normalized = (provider || 'ollama').toLowerCase();
        if (this.recyclerModelCache[normalized]) {
            return this.recyclerModelCache[normalized];
        }

        if (this.recyclerModelRequests[normalized]) {
            return this.recyclerModelRequests[normalized];
        }

        const request = (async () => {
            try {
                const response = await fetch(`${this.apiBase}/settings/recycler/models?provider=${encodeURIComponent(normalized)}`);
                if (!response.ok) {
                    const text = await response.text();
                    throw new Error(text || response.statusText);
                }

                const data = await response.json();
                const models = Array.isArray(data.models) ? data.models : [];
                if (data.error) {
                    console.warn(`Recycler model fetch warning for ${normalized}:`, data.error);
                    this.showToast(`Model list warning for ${normalized}: ${data.error}`, 'warning');
                }
                this.recyclerModelCache[normalized] = models;
                return models;
            } catch (error) {
                console.error(`Failed to fetch ${normalized} models:`, error);
                this.showToast(`Failed to load ${normalized} models: ${error.message}`, 'error');
                return [];
            } finally {
                delete this.recyclerModelRequests[normalized];
            }
        })();

        this.recyclerModelRequests[normalized] = request;
        return request;
    }

    async populateRecyclerModelSelector(provider, selectedModel = '') {
        const select = document.getElementById('settings-recycler-model-name');
        const customInput = document.getElementById('settings-recycler-model-name-custom');
        if (!select || !customInput) {
            return;
        }

        this.toggleCustomModelInput(false);

        const targetProvider = (provider || 'ollama').toLowerCase();
        const pendingValue = (selectedModel || '').trim();

        select.disabled = true;
        select.innerHTML = '';
        const loadingOption = document.createElement('option');
        loadingOption.value = '';
        loadingOption.textContent = 'Loading models…';
        select.appendChild(loadingOption);

        const models = await this.fetchRecyclerModels(targetProvider);

        select.innerHTML = '';
        const placeholderOption = document.createElement('option');
        placeholderOption.value = '';
        placeholderOption.textContent = models.length ? 'Select a model…' : 'No models found';
        select.appendChild(placeholderOption);

        models.forEach(model => {
            if (!model || !model.id) return;
            const option = document.createElement('option');
            option.value = model.id;
            option.textContent = model.label || model.id;
            select.appendChild(option);
        });

        const customOption = document.createElement('option');
        customOption.value = '__custom__';
        customOption.textContent = 'Custom model…';
        select.appendChild(customOption);

        select.disabled = false;

        let resolvedValue = pendingValue;
        if (pendingValue) {
            const match = models.find(model => (model.id || '').toLowerCase() === pendingValue.toLowerCase());
            if (match) {
                select.value = match.id;
                resolvedValue = match.id;
                this.toggleCustomModelInput(false);
            } else {
                select.value = '__custom__';
                resolvedValue = pendingValue;
                this.toggleCustomModelInput(true, pendingValue);
            }
        } else {
            select.value = '';
            resolvedValue = '';
            this.toggleCustomModelInput(false);
        }

        this.handleRecyclerModelSelection(select.value, resolvedValue);
    }

    handleRecyclerModelSelection(value, pendingValue = '') {
        if (value === '__custom__') {
            this.toggleCustomModelInput(true, pendingValue);
        } else {
            this.toggleCustomModelInput(false);
        }
    }

    async handleRecyclerProviderChange(provider) {
        const previousProvider = this.settings?.recycler?.model_provider || this.defaultSettings.recycler.model_provider;
        const preservedValue = provider === previousProvider ? this.getCurrentRecyclerModelValue() : '';
        await this.populateRecyclerModelSelector(provider, preservedValue);
    }

    toggleCustomModelInput(show, presetValue) {
        const customInput = document.getElementById('settings-recycler-model-name-custom');
        if (!customInput) {
            return;
        }

        if (typeof presetValue === 'string') {
            customInput.value = presetValue;
        }

        if (show) {
            customInput.classList.remove('hidden');
            try {
                customInput.focus({ preventScroll: true });
            } catch (err) {
                // Older browsers may not support the options parameter; fall back silently.
                customInput.focus();
            }
        } else {
            customInput.classList.add('hidden');
        }
    }

    getCurrentRecyclerModelValue() {
        const select = document.getElementById('settings-recycler-model-name');
        if (!select) {
            return '';
        }
        if (select.value === '__custom__') {
            return document.getElementById('settings-recycler-model-name-custom')?.value?.trim() || '';
        }
        return select.value || '';
    }

    resetToDefaults() {
        this.settings = { ...this.defaultSettings };
        this.applySettingsToUI(this.settings);
        return this.settings;
    }
}
