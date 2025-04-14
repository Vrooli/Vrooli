/**
 * Project structure resource for the Vrooli MCP server
 * Provides a structural overview of the Vrooli project
 */
import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

// Set up __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get project root path
const projectRoot = path.resolve(__dirname, '../../../../');

// Function to get directory structure recursively
async function getDirectoryStructure(dir, depth = 2, currentDepth = 0) {
    if (currentDepth >= depth) {
        return { type: 'directory', name: path.basename(dir), note: 'Max depth reached' };
    }

    try {
        const items = await fs.readdir(dir);
        const itemStats = await Promise.all(
            items.map(async item => {
                const itemPath = path.join(dir, item);
                const stats = await fs.stat(itemPath);
                return { item, itemPath, isDirectory: stats.isDirectory() };
            })
        );

        const structure = await Promise.all(
            itemStats.map(async ({ item, itemPath, isDirectory }) => {
                // Skip node_modules and .git directories
                if (item === 'node_modules' || item === '.git') {
                    return {
                        type: 'directory',
                        name: item,
                        note: 'Directory contents skipped'
                    };
                }

                if (isDirectory) {
                    return getDirectoryStructure(itemPath, depth, currentDepth + 1);
                } else {
                    return {
                        type: 'file',
                        name: item,
                        size: (await fs.stat(itemPath)).size
                    };
                }
            })
        );

        return {
            type: 'directory',
            name: path.basename(dir),
            contents: structure
        };
    } catch (error) {
        console.error(`Error reading directory ${dir}:`, error);
        return {
            type: 'directory',
            name: path.basename(dir),
            error: error.message
        };
    }
}

/**
 * Resource for the project structure
 */
export const projectStructureResource = {
    id: 'project_structure',
    name: 'Project Structure',
    description: 'The directory structure of the Vrooli project',
    resource_type: 'text',
    parameters: {
        type: 'object',
        properties: {
            depth: {
                type: 'number',
                description: 'How many directory levels deep to retrieve',
                default: 2
            },
            startDir: {
                type: 'string',
                description: 'The directory to start from, relative to project root',
                default: ''
            }
        }
    },
    handler: async ({ depth = 2, startDir = '' }) => {
        try {
            const targetDir = path.join(projectRoot, startDir);
            const structure = await getDirectoryStructure(targetDir, depth);

            // Convert structure to formatted text
            function formatStructure(node, indent = 0) {
                const indentStr = '  '.repeat(indent);
                let output = '';

                if (node.type === 'directory') {
                    output += `${indentStr}📁 ${node.name}/\n`;

                    if (node.note) {
                        output += `${indentStr}  (${node.note})\n`;
                    } else if (node.contents) {
                        node.contents.forEach(item => {
                            output += formatStructure(item, indent + 1);
                        });
                    }
                } else {
                    output += `${indentStr}📄 ${node.name}\n`;
                }

                return output;
            }

            const formattedStructure = formatStructure(structure);

            return {
                content: formattedStructure,
                metadata: {
                    depth,
                    startDir,
                    type: 'text',
                    title: `Vrooli Project Structure (${startDir || 'root'})`
                }
            };
        } catch (error) {
            console.error('Error in projectStructureResource:', error);
            throw new Error(`Failed to retrieve project structure: ${error.message}`);
        }
    }
}; 