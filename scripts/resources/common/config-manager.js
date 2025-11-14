#!/usr/bin/env node

/**
 * Configuration Manager CLI
 * 
 * This Node.js script provides a command-line interface to the TypeScript
 * ConfigurationManager for use by shell scripts. This ensures atomic,
 * validated configuration updates without shell scripts needing to handle
 * JSON manipulation directly.
 * 
 * Usage:
 *   ./config-manager.js update --category ai --resource ollama --config '{"enabled": true, "baseUrl": "http://localhost:11434"}'
 *   ./config-manager.js remove --category ai --resource ollama
 *   ./config-manager.js validate
 *   ./config-manager.js backup
 */

import { program } from 'commander';
import { readFileSync, writeFileSync, existsSync, mkdirSync, copyFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

// Dynamic import for ES modules compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const projectRoot = resolve(__dirname, '../../..');
const serviceJsonPath = process.env.VROOLI_SERVICE_JSON || resolve(projectRoot, '.vrooli/service.json');
const backupsDir = resolve(projectRoot, '.vrooli/backups');

// Import the ConfigurationManager when available; otherwise fall back to direct JSON edits
async function importConfigManager() {
    try {
        const { ConfigurationManager } = await import('../../packages/server/dist/services/resources/ConfigurationManager.js');
        return ConfigurationManager;
    } catch (error) {
        if (process.env.DEBUG) {
            console.warn('ConfigurationManager import failed, using fallback JSON editor:', error.message);
        }
        return null;
    }
}

// Error handling
function handleError(error, operation) {
    console.error(`Error during ${operation}:`, error.message);
    if (process.env.DEBUG) {
        console.error('Stack trace:', error.stack);
    }
    process.exit(1);
}

// Success response
function handleSuccess(result, operation) {
    console.log(`Success: ${operation} completed`);
    if (result.message) {
        console.log(`Message: ${result.message}`);
    }
    if (result.backupPath) {
        console.log(`Backup created: ${result.backupPath}`);
    }
    if (result.warnings && result.warnings.length > 0) {
        console.warn('Warnings:');
        result.warnings.forEach(warning => console.warn(`  - ${warning}`));
    }
}

function loadServiceConfig() {
    if (!existsSync(serviceJsonPath)) {
        throw new Error(`service.json not found at ${serviceJsonPath}`);
    }
    const contents = readFileSync(serviceJsonPath, 'utf8');
    return JSON.parse(contents);
}

function saveServiceConfig(config) {
    const updated = JSON.stringify(config, null, 2);
    writeFileSync(serviceJsonPath, `${updated}\n`, 'utf8');
}

function ensureDependencies(config) {
    config.dependencies = config.dependencies || {};
    config.dependencies.resources = config.dependencies.resources || {};
    return config;
}

function fallbackUpdateResourceConfig(options) {
    const resourceConfig = JSON.parse(options.config);
    const config = ensureDependencies(loadServiceConfig());
    config.dependencies.resources[options.resource] = resourceConfig;
    saveServiceConfig(config);
    handleSuccess({ message: `Stored ${options.resource} via fallback` }, 'Resource configuration update');
}

function fallbackRemoveResourceConfig(options) {
    const config = ensureDependencies(loadServiceConfig());
    if (config.dependencies.resources[options.resource]) {
        delete config.dependencies.resources[options.resource];
        saveServiceConfig(config);
        handleSuccess({ message: `Removed ${options.resource} via fallback` }, 'Resource configuration removal');
    } else {
        console.log(`No configuration found for ${options.resource}, nothing to remove`);
    }
}

function fallbackValidateConfig() {
    const config = loadServiceConfig();
    if (!config.dependencies || !config.dependencies.resources) {
        throw new Error('Configuration missing dependencies.resources');
    }
    console.log('✅ Configuration is valid JSON and contains dependencies.resources');
}

function fallbackCreateBackup() {
    if (!existsSync(serviceJsonPath)) {
        throw new Error(`service.json not found at ${serviceJsonPath}`);
    }
    if (!existsSync(backupsDir)) {
        mkdirSync(backupsDir, { recursive: true });
    }
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const backupPath = resolve(backupsDir, `service-${timestamp}.json`);
    copyFileSync(serviceJsonPath, backupPath);
    console.log(`✅ Backup created: ${backupPath}`);
}

function fallbackRestoreBackup(options) {
    if (!options.backupPath || !existsSync(options.backupPath)) {
        throw new Error(`Backup file not found: ${options.backupPath}`);
    }
    const contents = readFileSync(options.backupPath, 'utf8');
    JSON.parse(contents); // validate JSON
    writeFileSync(serviceJsonPath, contents, 'utf8');
    handleSuccess({ message: `Restored from ${options.backupPath}` }, 'Configuration restore');
}

function fallbackShowConfig() {
    const config = loadServiceConfig();
    console.log(JSON.stringify(config, null, 2));
}

// Update resource configuration
async function updateResourceConfig(options) {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const resourceConfig = JSON.parse(options.config);
            const result = await configManager.updateResourceConfiguration(
                options.resource,
                options.category,
                resourceConfig
            );
            
            if (result.success) {
                handleSuccess(result, 'Resource configuration update');
            } else {
                throw new Error(`Configuration update failed: ${result.message}`);
            }
        } catch (error) {
            handleError(error, 'resource configuration update');
        }
    } else {
        try {
            fallbackUpdateResourceConfig(options);
        } catch (error) {
            handleError(error, 'resource configuration update (fallback)');
        }
    }
}

// Remove resource configuration
async function removeResourceConfig(options) {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const result = await configManager.removeResourceConfiguration(
                options.resource,
                options.category
            );
            
            if (result.success) {
                handleSuccess(result, 'Resource configuration removal');
            } else {
                throw new Error(`Configuration removal failed: ${result.message}`);
            }
        } catch (error) {
            handleError(error, 'resource configuration removal');
        }
    } else {
        try {
            fallbackRemoveResourceConfig(options);
        } catch (error) {
            handleError(error, 'resource configuration removal (fallback)');
        }
    }
}

// Validate configuration
async function validateConfig() {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const config = await configManager.loadConfiguration();
            const validationResult = await configManager.validateConfiguration(config);
            
            if (validationResult.valid) {
                console.log('✅ Configuration is valid');
                if (validationResult.warnings && validationResult.warnings.length > 0) {
                    console.warn('Warnings:');
                    validationResult.warnings.forEach(warning => console.warn(`  - ${warning}`));
                }
            } else {
                console.error('❌ Configuration validation failed:');
                if (validationResult.errors) {
                    validationResult.errors.forEach(error => {
                        console.error(`  - ${error.path}: ${error.message}`);
                    });
                }
                process.exit(1);
            }
        } catch (error) {
            handleError(error, 'configuration validation');
        }
    } else {
        try {
            fallbackValidateConfig();
        } catch (error) {
            handleError(error, 'configuration validation (fallback)');
        }
    }
}

// Create backup
async function createBackup() {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const backupPath = await configManager.createBackup();
            console.log(`✅ Backup created: ${backupPath}`);
        } catch (error) {
            handleError(error, 'backup creation');
        }
    } else {
        try {
            fallbackCreateBackup();
        } catch (error) {
            handleError(error, 'backup creation (fallback)');
        }
    }
}

// Restore from backup
async function restoreBackup(options) {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const result = await configManager.restoreFromBackup(options.backupPath);
            
            if (result.success) {
                handleSuccess(result, 'Configuration restore');
            } else {
                throw new Error(`Restore failed: ${result.message}`);
            }
        } catch (error) {
            handleError(error, 'configuration restore');
        }
    } else {
        try {
            fallbackRestoreBackup(options);
        } catch (error) {
            handleError(error, 'configuration restore (fallback)');
        }
    }
}

// Show current configuration
async function showConfig() {
    const ConfigurationManager = await importConfigManager();

    if (ConfigurationManager) {
        const configManager = new ConfigurationManager();
        try {
            const config = await configManager.loadConfiguration();
            console.log(JSON.stringify(config, null, 2));
        } catch (error) {
            handleError(error, 'configuration display');
        }
    } else {
        try {
            fallbackShowConfig();
        } catch (error) {
            handleError(error, 'configuration display (fallback)');
        }
    }
}

// CLI setup
program
    .name('config-manager')
    .description('Vrooli Resource Configuration Manager CLI')
    .version('1.0.0');

program
    .command('update')
    .description('Update resource configuration')
    .requiredOption('--category <category>', 'Resource category (ai, automation, storage, agents)')
    .requiredOption('--resource <resource>', 'Resource ID')
    .requiredOption('--config <config>', 'JSON configuration object')
    .action(updateResourceConfig);

program
    .command('remove')
    .description('Remove resource configuration')
    .requiredOption('--category <category>', 'Resource category (ai, automation, storage, agents)')
    .requiredOption('--resource <resource>', 'Resource ID')
    .action(removeResourceConfig);

program
    .command('validate')
    .description('Validate current configuration')
    .action(validateConfig);

program
    .command('backup')
    .description('Create configuration backup')
    .action(createBackup);

program
    .command('restore')
    .description('Restore configuration from backup')
    .requiredOption('--backup-path <path>', 'Path to backup file')
    .action(restoreBackup);

program
    .command('show')
    .description('Show current configuration')
    .action(showConfig);

// Handle unknown commands
program
    .command('*')
    .action(() => {
        console.error('Unknown command. Use --help for available commands.');
        process.exit(1);
    });

// Parse arguments
program.parse();

// Show help if no command provided
if (process.argv.length <= 2) {
    program.help();
}
