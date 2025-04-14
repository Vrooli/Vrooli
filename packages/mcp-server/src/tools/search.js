/**
 * Search tool for the Vrooli MCP server
 * Allows searching for routines and other content in the Vrooli project
 */
import path from 'path';
import { fileURLToPath } from 'url';

// Set up __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get project root path
const projectRoot = path.resolve(__dirname, '../../../../');

/**
 * Tool to search for routines in the project
 */
export const searchRoutines = {
    id: 'search_routines',
    name: 'Search Routines',
    description: 'Search for routines and other content in the Vrooli project',
    parameters: {
        type: 'object',
        properties: {
            query: {
                type: 'string',
                description: 'Search query'
            },
            type: {
                type: 'string',
                enum: ['Routine', 'User', 'Team', 'All'],
                description: 'Type of content to search for',
                default: 'Routine'
            },
            limit: {
                type: 'number',
                description: 'Maximum number of results to return',
                default: 10
            }
        },
        required: ['query']
    },
    handler: async ({ query, type = 'Routine', limit = 10 }) => {
        try {
            // This is a mock implementation
            // In a real implementation, this would query a database or API
            console.log(`Searching for "${query}" in ${type} content (limit: ${limit})`);

            // Mock results based on search type
            const mockResults = {
                Routine: [
                    { id: 'r1', name: 'Create New Project', description: 'A routine for creating new projects' },
                    { id: 'r2', name: 'Deploy Application', description: 'Deploy an application to production' },
                    { id: 'r3', name: 'Onboard New Team Member', description: 'Steps to onboard new team members' }
                ],
                User: [
                    { id: 'u1', name: 'DevBot', isBot: true, description: 'A bot for development tasks' },
                    { id: 'u2', name: 'ResearchBot', isBot: true, description: 'A bot specialized in research' }
                ],
                Team: [
                    { id: 't1', name: 'Development Team', description: 'Team responsible for software development' },
                    { id: 't2', name: 'Research Team', description: 'Team focused on research and innovation' }
                ]
            };

            // Filter results based on the query (case-insensitive partial match)
            const lowerQuery = query.toLowerCase();
            let results;

            if (type === 'All') {
                // Combine all result types
                results = [
                    ...mockResults.Routine,
                    ...mockResults.User,
                    ...mockResults.Team
                ].filter(item =>
                    item.name.toLowerCase().includes(lowerQuery) ||
                    item.description.toLowerCase().includes(lowerQuery)
                );
            } else {
                // Filter for specific type
                results = mockResults[type].filter(item =>
                    item.name.toLowerCase().includes(lowerQuery) ||
                    item.description.toLowerCase().includes(lowerQuery)
                );
            }

            // Apply limit
            results = results.slice(0, limit);

            return {
                query,
                type,
                count: results.length,
                results
            };
        } catch (error) {
            console.error('Error in searchRoutines tool:', error);
            throw new Error(`Failed to search routines: ${error.message}`);
        }
    }
}; 