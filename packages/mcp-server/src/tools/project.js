/**
 * Project info tool for the Vrooli MCP server
 * Provides basic information about the Vrooli project
 */
import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

// Set up __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get project root path
const projectRoot = path.resolve(__dirname, '../../../../');

/**
 * Tool to get project information
 */
export const projectInfo = {
    id: 'project_info',
    name: 'Project Information',
    description: 'Retrieves basic information about the Vrooli project',
    parameters: {
        type: 'object',
        properties: {
            detail: {
                type: 'string',
                enum: ['basic', 'extended', 'full'],
                description: 'Level of detail to retrieve',
                default: 'basic'
            }
        }
    },
    handler: async ({ detail = 'basic' }) => {
        try {
            // Read package.json from project root
            const packageJsonPath = path.join(projectRoot, 'package.json');
            const packageData = JSON.parse(await fs.readFile(packageJsonPath, 'utf8'));

            // Basic information always included
            const info = {
                name: packageData.name,
                version: packageData.version,
                description: 'Vrooli - A polymorphic, collaborative, and self-improving automation platform',
                license: packageData.license,
                author: packageData.author
            };

            // Extended information
            if (detail === 'extended' || detail === 'full') {
                // Get package subdirectories
                const packagesDir = path.join(projectRoot, 'packages');
                const packages = await fs.readdir(packagesDir);

                // Get workspace information
                info.packages = packages;
                info.workspaces = packageData.workspaces || [];
                info.dependencies = {
                    main: Object.keys(packageData.dependencies || {}),
                    dev: Object.keys(packageData.devDependencies || {})
                };
            }

            // Full information
            if (detail === 'full') {
                // Additional project details can be added here
                // This could include more extensive configuration or environment details
                try {
                    const envExample = await fs.readFile(path.join(projectRoot, '.env-example'), 'utf8');
                    info.configOptions = envExample
                        .split('\n')
                        .filter(line => line.trim().startsWith('# ') || line.includes('='))
                        .map(line => line.trim());
                } catch (error) {
                    info.configOptions = 'Unable to read configuration options';
                }
            }

            return info;
        } catch (error) {
            console.error('Error in projectInfo tool:', error);
            throw new Error(`Failed to retrieve project information: ${error.message}`);
        }
    }
}; 