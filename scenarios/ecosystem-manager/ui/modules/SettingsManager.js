// Settings Management Module
export class SettingsManager {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.settings = null;
        this.originalTheme = null; // Track original theme for preview cancellation
        this.defaultSettings = {
            theme: 'light',
            slots: 1,
            refresh_interval: 30,
            active: false,
            max_turns: 60,
            allowed_tools: 'Read,Write,Edit,Bash,LS,Glob,Grep',
            skip_permissions: true,
            task_timeout: 30,
            condensed_mode: false
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

        // Update queue processor status UI
        this.updateProcessorToggleUI(settings.active);

        // Apply theme to body element
        this.applyTheme(settings.theme || 'light');
        
        // Store the original theme for potential reversion
        this.originalTheme = settings.theme || 'light';
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
        return formData;
    }

    applyTheme(theme) {
        const body = document.body;
        if (theme === 'dark') {
            body.classList.add('dark-mode');
        } else {
            body.classList.remove('dark-mode');
        }
        
        // Cache the theme in localStorage for immediate application on next load
        localStorage.setItem('ecosystemManager_theme', theme);
        console.log(`Theme applied: ${theme}, body classes:`, body.className);
    }

    // Static method to ensure cached theme is applied (redundant check)
    static applyCachedTheme() {
        const cachedTheme = localStorage.getItem('ecosystemManager_theme');
        
        // Apply proper dark mode class and remove inline styles
        if (cachedTheme === 'dark' || window.applyDarkModeOnLoad) {
            document.body.classList.add('dark-mode');
            // Remove the temporary inline style if it exists
            const tempStyle = document.querySelector('head style');
            if (tempStyle && tempStyle.textContent.includes('background-color: #1a1a1a')) {
                tempStyle.remove();
            }
        } else {
            document.body.classList.remove('dark-mode');
        }
        console.log(`Cached theme verified on load: ${cachedTheme || 'light'}`);
    }

    revertThemePreview() {
        // Revert to the original theme if user cancels
        if (this.originalTheme) {
            this.applyTheme(this.originalTheme);
        }
    }

    resetToDefaults() {
        this.settings = { ...this.defaultSettings };
        this.applySettingsToUI(this.settings);
        return this.settings;
    }
}