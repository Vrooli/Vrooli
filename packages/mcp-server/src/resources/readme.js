/**
 * README resource for the Vrooli MCP server
 * Provides access to the project's README file
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
 * Resource for the project README
 */
export const readmeResource = {
    id: 'project_readme',
    name: 'Project README',
    description: 'The README file for the Vrooli project',
    resource_type: 'text',
    handler: async () => {
        try {
            const readmePath = path.join(projectRoot, 'README.md');
            const readme = await fs.readFile(readmePath, 'utf8');

            return {
                content: readme,
                metadata: {
                    path: readmePath,
                    type: 'markdown',
                    title: 'Vrooli README'
                }
            };
        } catch (error) {
            console.error('Error in readmeResource:', error);
            throw new Error(`Failed to retrieve README: ${error.message}`);
        }
    }
}; 