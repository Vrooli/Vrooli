// Settings Management Module
export class SettingsManager {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.settings = null;
        this.defaultSettings = {
            maxConcurrentTasks: 1,
            claudeApiKey: '',
            executionMode: 'sequential',
            taskTimeout: 3600,
            retryFailedTasks: false,
            maxRetries: 3,
            enableNotifications: true,
            autoArchiveCompleted: false,
            archiveAfterDays: 30,
            queueProcessingEnabled: true,
            pollingInterval: 60,
            promptCachingEnabled: false,
            enableDetailedLogging: false
        };
    }

    async loadSettings() {
        try {
            const response = await fetch(`${this.apiBase}/settings`);
            if (!response.ok) {
                throw new Error(`Failed to load settings: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.settings = { ...this.defaultSettings, ...data.settings };
            return this.settings;
        } catch (error) {
            console.error('Failed to load settings:', error);
            this.settings = { ...this.defaultSettings };
            return this.settings;
        }
    }

    async saveSettings(settings) {
        try {
            const response = await fetch(`${this.apiBase}/settings`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ settings })
            });

            if (!response.ok) {
                const errorText = await response.text();
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
        // Map settings keys to form field IDs
        const fieldMapping = {
            'theme': 'settings-theme',
            'condensed_mode': 'settings-condensed-mode',
            'maxConcurrentTasks': 'settings-slots',
            'pollingInterval': 'settings-refresh',
            'queueProcessingEnabled': 'settings-active',
            'max_turns': 'settings-max-turns',
            'allowed_tools': 'settings-tools',
            'skip_permissions': 'settings-skip-permissions',
            'taskTimeout': 'settings-task-timeout'
        };
        
        // Update form fields with settings values
        Object.entries(fieldMapping).forEach(([settingKey, fieldId]) => {
            const element = document.getElementById(fieldId);
            if (element && settings[settingKey] !== undefined) {
                if (element.type === 'checkbox') {
                    element.checked = settings[settingKey];
                } else if (element.type === 'range' || element.type === 'number') {
                    let value = settings[settingKey];
                    // Convert task timeout from seconds to minutes for display
                    if (settingKey === 'taskTimeout') {
                        value = Math.round(value / 60);
                    }
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

        // Update queue processor toggle
        const processorToggle = document.getElementById('queue-processor-toggle');
        if (processorToggle) {
            processorToggle.checked = settings.queueProcessingEnabled;
            this.updateProcessorToggleUI(settings.queueProcessingEnabled);
        }
    }

    updateProcessorToggleUI(isActive) {
        const toggleBtn = document.querySelector('.queue-processor-toggle');
        const statusText = document.getElementById('processor-status-text');
        
        if (toggleBtn) {
            if (isActive) {
                toggleBtn.classList.add('active');
                if (statusText) statusText.textContent = 'Active';
            } else {
                toggleBtn.classList.remove('active');
                if (statusText) statusText.textContent = 'Paused';
            }
        }
    }

    getSettingsFromForm() {
        const formData = {};
        
        // Map form field IDs to settings keys
        const fieldMapping = {
            'settings-theme': 'theme',
            'settings-condensed-mode': 'condensed_mode',
            'settings-slots': 'maxConcurrentTasks',
            'settings-refresh': 'pollingInterval',
            'settings-active': 'queueProcessingEnabled',
            'settings-max-turns': 'max_turns',
            'settings-tools': 'allowed_tools',
            'settings-skip-permissions': 'skip_permissions',
            'settings-task-timeout': 'taskTimeout'
        };
        
        // Collect settings from form using mapped field IDs
        Object.entries(fieldMapping).forEach(([fieldId, settingKey]) => {
            const element = document.getElementById(fieldId);
            if (element) {
                if (element.type === 'checkbox') {
                    formData[settingKey] = element.checked;
                } else if (element.type === 'range' || element.type === 'number') {
                    formData[settingKey] = parseInt(element.value) || 0;
                    // Convert task timeout from minutes to seconds
                    if (settingKey === 'taskTimeout') {
                        formData[settingKey] = formData[settingKey] * 60;
                    }
                } else {
                    formData[settingKey] = element.value;
                }
            }
        });

        return formData;
    }

    resetToDefaults() {
        this.settings = { ...this.defaultSettings };
        this.applySettingsToUI(this.settings);
        return this.settings;
    }
}