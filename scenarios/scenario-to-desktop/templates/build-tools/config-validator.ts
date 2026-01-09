/**
 * Configuration Validator for Desktop App Generation
 * Validates and sanitizes desktop application configuration
 */

interface ValidationError {
    field: string;
    message: string;
    severity: 'error' | 'warning';
}

interface ValidationResult {
    isValid: boolean;
    errors: ValidationError[];
    sanitizedConfig?: any;
}

export class DesktopConfigValidator {
    
    validate(config: any): ValidationResult {
        const errors: ValidationError[] = [];
        const sanitized = { ...config };
        
        // Required fields validation
        this.validateRequiredFields(config, errors);
        
        // App identity validation
        this.validateAppIdentity(config, errors, sanitized);
        
        // Server configuration validation
        this.validateServerConfig(config, errors, sanitized);
        
        // Template and framework validation
        this.validateTemplateConfig(config, errors, sanitized);
        
        // Features validation
        this.validateFeatures(config, errors, sanitized);
        
        // Window configuration validation
        this.validateWindowConfig(config, errors, sanitized);
        
        // Platform validation
        this.validatePlatforms(config, errors, sanitized);
        
        // Output path validation
        this.validateOutputPath(config, errors, sanitized);
        
        const isValid = !errors.some(e => e.severity === 'error');
        
        return {
            isValid,
            errors,
            sanitizedConfig: isValid ? sanitized : undefined
        };
    }
    
    private validateRequiredFields(config: any, errors: ValidationError[]): void {
        const required = [
            'appName',
            'appDisplayName', 
            'appDescription',
            'version',
            'author',
            'appId',
            'serverType',
            'serverPath',
            'apiEndpoint',
            'framework',
            'templateType',
            'outputPath'
        ];
        
        for (const field of required) {
            if (!config[field]) {
                errors.push({
                    field,
                    message: `Required field '${field}' is missing`,
                    severity: 'error'
                });
            }
        }
    }
    
    private validateAppIdentity(config: any, errors: ValidationError[], sanitized: any): void {
        // App name validation
        if (config.appName) {
            if (!/^[a-z0-9-]+$/.test(config.appName)) {
                errors.push({
                    field: 'appName',
                    message: 'App name must contain only lowercase letters, numbers, and hyphens',
                    severity: 'error'
                });
            }
            
            if (config.appName.length > 50) {
                errors.push({
                    field: 'appName',
                    message: 'App name should be 50 characters or less',
                    severity: 'warning'
                });
            }
        }
        
        // App ID validation
        if (config.appId) {
            if (!/^[a-zA-Z0-9.-]+$/.test(config.appId)) {
                errors.push({
                    field: 'appId',
                    message: 'App ID must be in reverse domain format (e.g., com.company.app)',
                    severity: 'error'
                });
            }
        }
        
        // Version validation
        if (config.version) {
            if (!/^\d+\.\d+\.\d+/.test(config.version)) {
                errors.push({
                    field: 'version',
                    message: 'Version must follow semantic versioning (e.g., 1.0.0)',
                    severity: 'warning'
                });
            }
        }
        
        // Set defaults
        sanitized.license = config.license || 'MIT';
        sanitized.appUrl = config.appUrl || '';
    }
    
    private validateServerConfig(config: any, errors: ValidationError[], sanitized: any): void {
        const validServerTypes = ['node', 'static', 'external', 'executable'];
        
        if (config.serverType && !validServerTypes.includes(config.serverType)) {
            errors.push({
                field: 'serverType',
                message: `Invalid server type. Must be one of: ${validServerTypes.join(', ')}`,
                severity: 'error'
            });
        }
        
        // Port validation for server types that need it
        if (['node', 'executable'].includes(config.serverType)) {
            if (!config.serverPort) {
                sanitized.serverPort = 3000; // Default
                errors.push({
                    field: 'serverPort',
                    message: 'Server port not specified, using default 3000',
                    severity: 'warning'
                });
            } else if (config.serverPort < 1024 || config.serverPort > 65535) {
                errors.push({
                    field: 'serverPort',
                    message: 'Server port must be between 1024 and 65535',
                    severity: 'error'
                });
            }
        }
        
        // API endpoint validation
        if (config.apiEndpoint) {
            try {
                new URL(config.apiEndpoint);
            } catch {
                errors.push({
                    field: 'apiEndpoint',
                    message: 'API endpoint must be a valid URL',
                    severity: 'error'
                });
            }
        }
        
        // Scenario dist path default
        sanitized.scenarioDistPath = config.scenarioDistPath || '../ui/dist';
    }
    
    private validateTemplateConfig(config: any, errors: ValidationError[], sanitized: any): void {
        const validFrameworks = ['electron', 'tauri', 'neutralino'];
        const validTemplateTypes = ['basic', 'advanced', 'kiosk', 'multi_window'];
        
        if (config.framework && !validFrameworks.includes(config.framework)) {
            errors.push({
                field: 'framework',
                message: `Invalid framework. Must be one of: ${validFrameworks.join(', ')}`,
                severity: 'error'
            });
        }
        
        if (config.templateType && !validTemplateTypes.includes(config.templateType)) {
            errors.push({
                field: 'templateType',
                message: `Invalid template type. Must be one of: ${validTemplateTypes.join(', ')}`,
                severity: 'error'
            });
        }
        
        // Warn about framework limitations
        if (config.framework === 'tauri' && config.templateType === 'multi_window') {
            errors.push({
                field: 'framework',
                message: 'Tauri multi-window support is experimental',
                severity: 'warning'
            });
        }
        
        if (config.framework === 'neutralino' && config.templateType === 'advanced') {
            errors.push({
                field: 'framework',
                message: 'Neutralino has limited advanced features support',
                severity: 'warning'
            });
        }
    }
    
    private validateFeatures(config: any, errors: ValidationError[], sanitized: any): void {
        // Ensure features object exists
        sanitized.features = config.features || {};
        
        // Set feature defaults based on template type
        const featureDefaults = this.getFeatureDefaults(config.templateType);
        Object.assign(sanitized.features, featureDefaults, sanitized.features);
        
        // Validate feature combinations
        if (sanitized.features.systemTray && config.templateType === 'kiosk') {
            errors.push({
                field: 'features.systemTray',
                message: 'System tray is not recommended for kiosk mode applications',
                severity: 'warning'
            });
        }
        
        if (sanitized.features.devTools && config.templateType === 'kiosk') {
            errors.push({
                field: 'features.devTools',
                message: 'Dev tools should be disabled for production kiosk applications',
                severity: 'warning'
            });
        }
    }
    
    private getFeatureDefaults(templateType: string): Record<string, boolean> {
        const defaults = {
            basic: {
                splash: true,
                systemTray: false,
                autoUpdater: true,
                devTools: true,
                singleInstance: true
            },
            advanced: {
                splash: true,
                systemTray: true,
                autoUpdater: true,
                devTools: true,
                singleInstance: true
            },
            kiosk: {
                splash: false,
                systemTray: false,
                autoUpdater: true,
                devTools: false,
                singleInstance: true
            },
            multi_window: {
                splash: true,
                systemTray: true,
                autoUpdater: true,
                devTools: true,
                singleInstance: true
            }
        };
        
        return defaults[templateType as keyof typeof defaults] || defaults.basic;
    }
    
    private validateWindowConfig(config: any, errors: ValidationError[], sanitized: any): void {
        // Ensure window object exists
        sanitized.window = config.window || {};
        
        // Set defaults
        const windowDefaults = {
            width: 1200,
            height: 800,
            background: '#f5f5f5'
        };
        Object.assign(sanitized.window, windowDefaults, sanitized.window);
        
        // Validate dimensions
        if (sanitized.window.width < 320) {
            errors.push({
                field: 'window.width',
                message: 'Window width must be at least 320 pixels',
                severity: 'error'
            });
        }
        
        if (sanitized.window.height < 240) {
            errors.push({
                field: 'window.height',
                message: 'Window height must be at least 240 pixels',
                severity: 'error'
            });
        }
        
        // Warn about very large windows
        if (sanitized.window.width > 2560 || sanitized.window.height > 1440) {
            errors.push({
                field: 'window',
                message: 'Very large window dimensions may not fit on all screens',
                severity: 'warning'
            });
        }
        
        // Validate background color
        if (sanitized.window.background && !/^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$/.test(sanitized.window.background)) {
            errors.push({
                field: 'window.background',
                message: 'Background color must be a valid hex color (e.g., #ffffff)',
                severity: 'warning'
            });
            sanitized.window.background = '#f5f5f5'; // Reset to default
        }
    }
    
    private validatePlatforms(config: any, errors: ValidationError[], sanitized: any): void {
        const validPlatforms = ['win', 'mac', 'linux'];
        
        if (!config.platforms || !Array.isArray(config.platforms)) {
            sanitized.platforms = ['win', 'mac', 'linux']; // Default to all
            errors.push({
                field: 'platforms',
                message: 'No platforms specified, defaulting to all platforms',
                severity: 'warning'
            });
        } else {
            const invalidPlatforms = config.platforms.filter((p: string) => !validPlatforms.includes(p));
            if (invalidPlatforms.length > 0) {
                errors.push({
                    field: 'platforms',
                    message: `Invalid platforms: ${invalidPlatforms.join(', ')}. Valid: ${validPlatforms.join(', ')}`,
                    severity: 'error'
                });
            }
            
            sanitized.platforms = config.platforms.filter((p: string) => validPlatforms.includes(p));
        }
    }
    
    private validateOutputPath(config: any, errors: ValidationError[], sanitized: any): void {
        if (!config.outputPath) {
            return; // Already caught by required fields validation
        }
        
        // Check for dangerous paths
        const dangerousPaths = ['/', '/usr', '/System', 'C:\\', 'C:\\Windows'];
        if (dangerousPaths.some(dangerous => config.outputPath.startsWith(dangerous))) {
            errors.push({
                field: 'outputPath',
                message: 'Output path appears to be a system directory',
                severity: 'error'
            });
        }
        
        // Warn about existing important directories
        const importantPaths = ['node_modules', '.git', 'dist', 'build'];
        const pathSegments = config.outputPath.split(/[/\\]/);
        if (importantPaths.some(important => pathSegments.includes(important))) {
            errors.push({
                field: 'outputPath',
                message: 'Output path contains important directory names that might be overwritten',
                severity: 'warning'
            });
        }
    }
    
    validateStyleConfig(styling: any): ValidationError[] {
        const errors: ValidationError[] = [];
        
        if (!styling) return errors;
        
        const colorFields = ['splashBackgroundStart', 'splashBackgroundEnd', 'splashTextColor', 'splashAccentColor'];
        
        for (const field of colorFields) {
            if (styling[field] && !/^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$/.test(styling[field])) {
                errors.push({
                    field: `styling.${field}`,
                    message: `${field} must be a valid hex color (e.g., #ffffff)`,
                    severity: 'warning'
                });
            }
        }
        
        return errors;
    }
    
    generateSampleConfig(): any {
        return {
            appName: "my-scenario-desktop",
            appDisplayName: "My Scenario Desktop",
            appDescription: "Desktop application for My Scenario",
            version: "1.0.0",
            author: "Your Name",
            license: "MIT",
            appId: "com.yourcompany.myscenario",
            appUrl: "https://yourcompany.com/myscenario",
            
            serverType: "node",
            serverPort: 3000,
            serverPath: "backend/dist/server.js",
            apiEndpoint: "http://localhost:3001/api",
            scenarioDistPath: "../ui/dist",
            
            framework: "electron",
            templateType: "basic",
            
            features: {
                splash: true,
                systemTray: false,
                autoUpdater: true,
                devTools: true,
                singleInstance: true
            },
            
            window: {
                width: 1200,
                height: 800,
                background: "#f5f5f5"
            },
            
            platforms: ["win", "mac", "linux"],
            
            outputPath: "./desktop-app",
            
            styling: {
                splashBackgroundStart: "#4a90e2",
                splashBackgroundEnd: "#357abd",
                splashTextColor: "#ffffff",
                splashAccentColor: "#64b5f6"
            }
        };
    }
}

export { ValidationError, ValidationResult };