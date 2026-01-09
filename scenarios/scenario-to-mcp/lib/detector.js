/**
 * MCP Detection Library
 * Scans Vrooli scenarios to detect existing MCP implementations
 */

const fs = require('fs').promises;
const path = require('path');

class MCPDetector {
    constructor(scenariosPath) {
        // Security: Use absolute path from environment or well-known location
        // Never accept arbitrary user input for path traversal protection
        if (scenariosPath && path.isAbsolute(scenariosPath)) {
            this.scenariosPath = scenariosPath;
        } else {
            // Default to Vrooli scenarios directory using HOME or VROOLI_ROOT
            const rootDir = process.env.VROOLI_ROOT || path.join(process.env.HOME || '/tmp', 'Vrooli');
            this.scenariosPath = path.join(rootDir, 'scenarios');
        }
        this.mcpIndicators = {
            directories: ['mcp', '.mcp'],
            files: ['mcp-server.js', 'mcp-server.ts', 'server.mcp.js', 'manifest.json'],
            packageIndicators: ['@modelcontextprotocol/sdk', 'mcp-sdk', 'model-context-protocol']
        };
    }

    /**
     * Scan all scenarios and detect MCP support
     * @returns {Promise<Array>} Array of scenario MCP status objects
     */
    async scanAllScenarios() {
        try {
            const scenarios = await this.listScenarios();
            const results = await Promise.all(
                scenarios.map(scenario => this.checkScenarioMCPStatus(scenario))
            );
            return results.filter(r => r !== null);
        } catch (error) {
            console.error('Error scanning scenarios:', error);
            return [];
        }
    }

    /**
     * List all available scenarios
     * @returns {Promise<Array>} Array of scenario names
     */
    async listScenarios() {
        try {
            const entries = await fs.readdir(this.scenariosPath, { withFileTypes: true });
            return entries
                .filter(entry => entry.isDirectory() && !entry.name.startsWith('.'))
                .map(entry => entry.name);
        } catch (error) {
            console.error('Error listing scenarios:', error);
            return [];
        }
    }

    /**
     * Check MCP status for a specific scenario
     * @param {string} scenarioName - Name of the scenario
     * @returns {Promise<Object>} MCP status object
     */
    async checkScenarioMCPStatus(scenarioName) {
        const scenarioPath = path.join(this.scenariosPath, scenarioName);
        
        try {
            const status = {
                name: scenarioName,
                path: scenarioPath,
                hasMCP: false,
                mcpPath: null,
                manifest: null,
                port: null,
                tools: [],
                confidence: 'none',
                details: {}
            };

            // Check for MCP directory
            const mcpDirPath = path.join(scenarioPath, 'mcp');
            const hasMCPDir = await this.pathExists(mcpDirPath);
            
            if (hasMCPDir) {
                status.hasMCP = true;
                status.mcpPath = mcpDirPath;
                status.confidence = 'high';
                status.details.hasDirectory = true;

                // Check for manifest.json
                const manifestPath = path.join(mcpDirPath, 'manifest.json');
                if (await this.pathExists(manifestPath)) {
                    const manifest = await this.readJSON(manifestPath);
                    if (manifest) {
                        status.manifest = manifest;
                        status.tools = this.extractToolsFromManifest(manifest);
                        status.details.hasManifest = true;
                    }
                }

                // Check for server file
                const serverFiles = ['server.js', 'index.js', 'mcp-server.js'];
                for (const file of serverFiles) {
                    const serverPath = path.join(mcpDirPath, file);
                    if (await this.pathExists(serverPath)) {
                        status.details.serverFile = file;
                        
                        // Try to extract port from server file
                        const port = await this.extractPortFromFile(serverPath);
                        if (port) {
                            status.port = port;
                        }
                        break;
                    }
                }
            }

            // Check service.json for MCP configuration
            const serviceJsonPath = path.join(scenarioPath, '.vrooli', 'service.json');
            if (await this.pathExists(serviceJsonPath)) {
                const serviceConfig = await this.readJSON(serviceJsonPath);
                if (serviceConfig && serviceConfig.mcp) {
                    status.hasMCP = true;
                    status.confidence = status.confidence === 'high' ? 'high' : 'medium';
                    status.details.hasServiceConfig = true;
                    
                    if (serviceConfig.mcp.port) {
                        status.port = serviceConfig.mcp.port;
                    }
                    if (serviceConfig.mcp.enabled !== undefined) {
                        status.details.mcpEnabled = serviceConfig.mcp.enabled;
                    }
                }
            }

            // Check package.json for MCP dependencies
            const packageJsonPath = path.join(scenarioPath, 'package.json');
            if (await this.pathExists(packageJsonPath)) {
                const packageJson = await this.readJSON(packageJsonPath);
                if (packageJson && packageJson.dependencies) {
                    const hasMCPDep = this.mcpIndicators.packageIndicators.some(
                        indicator => packageJson.dependencies[indicator]
                    );
                    if (hasMCPDep) {
                        status.hasMCP = true;
                        status.confidence = status.confidence === 'none' ? 'low' : status.confidence;
                        status.details.hasMCPDependency = true;
                    }
                }
            }

            return status;
        } catch (error) {
            console.error(`Error checking MCP status for ${scenarioName}:`, error);
            return {
                name: scenarioName,
                path: scenarioPath,
                hasMCP: false,
                error: error.message
            };
        }
    }

    /**
     * Extract tools from MCP manifest
     * @param {Object} manifest - MCP manifest object
     * @returns {Array} Array of tool definitions
     */
    extractToolsFromManifest(manifest) {
        const tools = [];
        
        if (manifest.tools) {
            if (Array.isArray(manifest.tools)) {
                return manifest.tools;
            } else if (typeof manifest.tools === 'object') {
                return Object.entries(manifest.tools).map(([name, def]) => ({
                    name,
                    ...def
                }));
            }
        }
        
        if (manifest.capabilities && manifest.capabilities.tools) {
            return manifest.capabilities.tools;
        }
        
        return tools;
    }

    /**
     * Extract port number from server file
     * @param {string} filePath - Path to server file
     * @returns {Promise<number|null>} Port number or null
     */
    async extractPortFromFile(filePath) {
        try {
            const content = await fs.readFile(filePath, 'utf-8');
            
            // Look for port patterns
            const patterns = [
                /port\s*[=:]\s*(\d{4,5})/i,
                /MCP_PORT['"]\s*[=:]\s*(\d{4,5})/,
                /listen\s*\(\s*(\d{4,5})/,
                /process\.env\.MCP_PORT\s*\|\|\s*(\d{4,5})/
            ];
            
            for (const pattern of patterns) {
                const match = content.match(pattern);
                if (match && match[1]) {
                    return parseInt(match[1]);
                }
            }
            
            return null;
        } catch (error) {
            return null;
        }
    }

    /**
     * Get detailed MCP information for a scenario
     * @param {string} scenarioName - Name of the scenario
     * @returns {Promise<Object>} Detailed MCP information
     */
    async getScenarioMCPDetails(scenarioName) {
        const status = await this.checkScenarioMCPStatus(scenarioName);
        
        if (!status.hasMCP) {
            return status;
        }
        
        // Enrich with additional details
        if (status.mcpPath) {
            try {
                const files = await fs.readdir(status.mcpPath);
                status.details.mcpFiles = files;
                
                // Look for handlers directory
                const handlersPath = path.join(status.mcpPath, 'handlers');
                if (await this.pathExists(handlersPath)) {
                    const handlers = await fs.readdir(handlersPath);
                    status.details.handlers = handlers;
                }
            } catch (error) {
                status.details.filesError = error.message;
            }
        }
        
        return status;
    }

    /**
     * Check if a path exists
     * @param {string} pathToCheck - Path to check
     * @returns {Promise<boolean>} True if exists
     */
    async pathExists(pathToCheck) {
        try {
            await fs.access(pathToCheck);
            return true;
        } catch {
            return false;
        }
    }

    /**
     * Read and parse JSON file
     * @param {string} filePath - Path to JSON file
     * @returns {Promise<Object|null>} Parsed JSON or null
     */
    async readJSON(filePath) {
        try {
            const content = await fs.readFile(filePath, 'utf-8');
            return JSON.parse(content);
        } catch (error) {
            return null;
        }
    }

    /**
     * Generate MCP readiness report
     * @returns {Promise<Object>} Readiness report
     */
    async generateReadinessReport() {
        const scenarios = await this.scanAllScenarios();
        
        const report = {
            timestamp: new Date().toISOString(),
            summary: {
                total: scenarios.length,
                withMCP: scenarios.filter(s => s.hasMCP).length,
                withoutMCP: scenarios.filter(s => !s.hasMCP).length,
                withErrors: scenarios.filter(s => s.error).length
            },
            byConfidence: {
                high: scenarios.filter(s => s.confidence === 'high').length,
                medium: scenarios.filter(s => s.confidence === 'medium').length,
                low: scenarios.filter(s => s.confidence === 'low').length,
                none: scenarios.filter(s => s.confidence === 'none').length
            },
            scenarios: scenarios.map(s => ({
                name: s.name,
                hasMCP: s.hasMCP,
                confidence: s.confidence,
                tools: s.tools.length,
                port: s.port,
                error: s.error
            }))
        };
        
        return report;
    }

    /**
     * Find scenarios that are good candidates for MCP addition
     * @returns {Promise<Array>} Array of candidate scenarios
     */
    async findMCPCandidates() {
        const scenarios = await this.scanAllScenarios();
        const candidates = [];
        
        for (const scenario of scenarios.filter(s => !s.hasMCP)) {
            const scenarioPath = path.join(this.scenariosPath, scenario.name);
            const candidate = {
                name: scenario.name,
                score: 0,
                reasons: []
            };
            
            // Check for API directory
            if (await this.pathExists(path.join(scenarioPath, 'api'))) {
                candidate.score += 3;
                candidate.reasons.push('Has API implementation');
            }
            
            // Check for CLI directory
            if (await this.pathExists(path.join(scenarioPath, 'cli'))) {
                candidate.score += 2;
                candidate.reasons.push('Has CLI interface');
            }
            
            // Check for complex functionality in service.json
            const serviceJsonPath = path.join(scenarioPath, '.vrooli', 'service.json');
            if (await this.pathExists(serviceJsonPath)) {
                const serviceConfig = await this.readJSON(serviceJsonPath);
                if (serviceConfig && serviceConfig.resources) {
                    const resourceCount = Object.keys(serviceConfig.resources).length;
                    if (resourceCount > 2) {
                        candidate.score += 2;
                        candidate.reasons.push(`Uses ${resourceCount} resources`);
                    }
                }
            }
            
            // Check PRD for value proposition
            const prdPath = path.join(scenarioPath, 'PRD.md');
            if (await this.pathExists(prdPath)) {
                candidate.score += 1;
                candidate.reasons.push('Has PRD documentation');
            }
            
            if (candidate.score > 0) {
                candidates.push(candidate);
            }
        }
        
        // Sort by score
        return candidates.sort((a, b) => b.score - a.score);
    }
}

// Export for use in other modules
module.exports = MCPDetector;

// CLI interface if run directly
if (require.main === module) {
    const detector = new MCPDetector();
    
    const command = process.argv[2];
    
    async function main() {
        switch (command) {
            case 'scan':
                const results = await detector.scanAllScenarios();
                console.log(JSON.stringify(results, null, 2));
                break;
                
            case 'report':
                const report = await detector.generateReadinessReport();
                console.log(JSON.stringify(report, null, 2));
                break;
                
            case 'candidates':
                const candidates = await detector.findMCPCandidates();
                console.log(JSON.stringify(candidates, null, 2));
                break;
                
            case 'check':
                const scenarioName = process.argv[3];
                if (!scenarioName) {
                    console.error('Usage: node detector.js check <scenario-name>');
                    process.exit(1);
                }
                const details = await detector.getScenarioMCPDetails(scenarioName);
                console.log(JSON.stringify(details, null, 2));
                break;
                
            default:
                console.log('MCP Detector - Usage:');
                console.log('  node detector.js scan       - Scan all scenarios');
                console.log('  node detector.js report     - Generate readiness report');
                console.log('  node detector.js candidates - Find MCP candidates');
                console.log('  node detector.js check <scenario> - Check specific scenario');
        }
    }
    
    main().catch(console.error);
}