/**
 * Unit tests for the Vrooli MCP server
 */
import { expect } from 'chai';
import sinon from 'sinon';

// Import tools for testing
import { projectInfo } from '../src/tools/project.js';
import { searchRoutines } from '../src/tools/search.js';

// Import resources for testing
import { readmeResource } from '../src/resources/readme.js';
import { projectStructureResource } from '../src/resources/structure.js';

// Basic MCP server implementation (simplified version of what's in index.js)
class McpServer {
    constructor(config) {
        this.id = config.id;
        this.name = config.name;
        this.description = config.description;
        this.version = config.version;
        this.capabilities = config.capabilities || {};
        this.tools = config.tools || [];
        this.resources = config.resources || [];

        // Map tools and resources for easy lookup
        this.toolsMap = new Map();
        this.resourcesMap = new Map();

        this.tools.forEach(tool => this.toolsMap.set(tool.id, tool));
        this.resources.forEach(resource => this.resourcesMap.set(resource.id, resource));
    }

    listTools() {
        return this.tools.map(tool => ({
            id: tool.id,
            name: tool.name,
            description: tool.description
        }));
    }

    listResources() {
        return this.resources.map(resource => ({
            id: resource.id,
            name: resource.name,
            description: resource.description
        }));
    }

    async callTool(id, params = {}) {
        const tool = this.toolsMap.get(id);
        if (!tool) {
            throw new Error(`Tool not found: ${id}`);
        }
        return await tool.handler(params);
    }

    async getResource(id, params = {}) {
        const resource = this.resourcesMap.get(id);
        if (!resource) {
            throw new Error(`Resource not found: ${id}`);
        }
        return await resource.handler(params);
    }
}

describe('Vrooli MCP Server', () => {
    let sandbox;
    let mcpServer;

    beforeEach(() => {
        // Create a sinon sandbox for stubs and spies
        sandbox = sinon.createSandbox();

        // Create a test MCP server
        mcpServer = new McpServer({
            id: 'test-mcp-server',
            name: 'Test MCP Server',
            description: 'Test server for unit testing',
            version: '0.1.0',
            capabilities: {
                tools: true,
                resources: true,
                prompts: false,
            },
            tools: [
                projectInfo,
                searchRoutines
            ],
            resources: [
                readmeResource,
                projectStructureResource
            ]
        });
    });

    afterEach(() => {
        // Restore stubs and spies
        sandbox.restore();
    });

    describe('Tools', () => {
        it('should have projectInfo tool registered', () => {
            const tools = mcpServer.listTools();
            const projectInfoTool = tools.find(tool => tool.id === 'project_info');
            expect(projectInfoTool).to.exist;
            expect(projectInfoTool.id).to.equal('project_info');
        });

        it('should have searchRoutines tool registered', () => {
            const tools = mcpServer.listTools();
            const searchTool = tools.find(tool => tool.id === 'search_routines');
            expect(searchTool).to.exist;
            expect(searchTool.id).to.equal('search_routines');
        });

        it('should handle projectInfo tool call', async () => {
            // Stub handler to avoid file system operations
            const infoStub = sandbox.stub(projectInfo, 'handler').resolves({
                name: 'test',
                version: '1.0.0'
            });

            const result = await mcpServer.callTool('project_info', { detail: 'basic' });

            expect(infoStub.calledOnce).to.be.true;
            expect(result).to.be.an('object');
            expect(result).to.have.property('name', 'test');
        });

        it('should handle searchRoutines tool call', async () => {
            // Stub handler to avoid file system operations
            const searchStub = sandbox.stub(searchRoutines, 'handler').resolves({
                query: 'test',
                count: 1,
                results: [{ id: 'r1', name: 'Test Routine' }]
            });

            const result = await mcpServer.callTool('search_routines', { query: 'test' });

            expect(searchStub.calledOnce).to.be.true;
            expect(result).to.be.an('object');
            expect(result).to.have.property('count', 1);
        });
    });

    describe('Resources', () => {
        it('should have readmeResource registered', () => {
            const resources = mcpServer.listResources();
            const readmeRes = resources.find(res => res.id === 'project_readme');
            expect(readmeRes).to.exist;
            expect(readmeRes.id).to.equal('project_readme');
        });

        it('should have projectStructureResource registered', () => {
            const resources = mcpServer.listResources();
            const structureRes = resources.find(res => res.id === 'project_structure');
            expect(structureRes).to.exist;
            expect(structureRes.id).to.equal('project_structure');
        });

        it('should handle readmeResource request', async () => {
            // Stub handler to avoid file system operations
            const readmeStub = sandbox.stub(readmeResource, 'handler').resolves({
                content: '# Test README',
                metadata: { type: 'markdown' }
            });

            const result = await mcpServer.getResource('project_readme');

            expect(readmeStub.calledOnce).to.be.true;
            expect(result).to.be.an('object');
            expect(result.content).to.equal('# Test README');
        });

        it('should handle projectStructureResource request', async () => {
            // Stub handler to avoid file system operations
            const structureStub = sandbox.stub(projectStructureResource, 'handler').resolves({
                content: '📁 root/\n  📄 file.txt\n',
                metadata: { depth: 2 }
            });

            const result = await mcpServer.getResource('project_structure', { depth: 2 });

            expect(structureStub.calledOnce).to.be.true;
            expect(result).to.be.an('object');
            expect(result.content).to.include('📁 root/');
        });
    });
}); 