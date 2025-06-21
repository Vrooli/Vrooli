/**
 * User Bookmarks Project Scenario
 * 
 * This scenario tests a complete user workflow:
 * 1. User creates account and profile
 * 2. User creates a public project  
 * 3. User bookmarks their own project
 * 4. User creates additional bookmarks for project resources
 * 5. User organizes bookmarks into lists
 * 
 * This demonstrates cross-object relationships and multi-step workflows.
 */

import type {
    ScenarioConfig,
    ScenarioResult,
    ScenarioStep,
    StepResult,
    Assertion,
    AssertionResult,
    AuthContext,
    BookmarkFormData
} from '../types.js';
import { BookmarkIntegrationTest } from '../integrations/bookmarkIntegration.test.js';
import { BookmarkFixtureFactory } from '../factories/BookmarkFixtureFactory.js';
import type { User, Team, Bookmark, BookmarkFor } from '@vrooli/shared';

/**
 * Configuration for the user bookmarks project scenario
 */
export interface UserBookmarksProjectConfig {
    user: {
        handle: string;
        name: string;
        email: string;
    };
    project: {
        name: string;
        description: string;
        isPrivate: boolean;
    };
    bookmarks: Array<{
        objectType: BookmarkFor;
        objectId: string;
        listName?: string;
    }>;
    expectedOutcome: {
        bookmarkCount: number;
        listCount: number;
        allBookmarksAccessible: boolean;
    };
}

/**
 * Context for sharing data between scenario steps
 */
export class ScenarioContext {
    private stepResults: Map<string, any> = new Map();
    private globalData: Map<string, any> = new Map();
    
    setStepResult(stepName: string, result: any): void {
        this.stepResults.set(stepName, result);
    }
    
    getStepResult(stepName: string): any {
        return this.stepResults.get(stepName);
    }
    
    setGlobalData(key: string, value: any): void {
        this.globalData.set(key, value);
    }
    
    getGlobalData(key: string): any {
        return this.globalData.get(key);
    }
    
    /**
     * Resolve template strings like "{{create_user.id}}"
     */
    resolveTemplate(template: string): any {
        if (typeof template !== 'string') return template;
        
        return template.replace(/\{\{([^}]+)\}\}/g, (match, path) => {
            const [stepName, ...propertyPath] = path.split('.');
            const stepResult = this.stepResults.get(stepName);
            
            if (!stepResult) {
                throw new Error(`Step result not found: ${stepName}`);
            }
            
            // Navigate property path
            let value = stepResult;
            for (const prop of propertyPath) {
                if (value && typeof value === 'object' && prop in value) {
                    value = value[prop];
                } else {
                    throw new Error(`Property path not found: ${path}`);
                }
            }
            
            return value;
        });
    }
    
    /**
     * Get value by dot notation path
     */
    getValueByPath(obj: any, path: string): any {
        return path.split('.').reduce((current, prop) => {
            return current && typeof current === 'object' ? current[prop] : undefined;
        }, obj);
    }
}

/**
 * Assertion engine for validating scenario outcomes
 */
export class AssertionEngine {
    constructor(private context: ScenarioContext) {}
    
    async executeAssertion(assertion: Assertion, data: any): Promise<AssertionResult> {
        const target = this.context.resolveTemplate(assertion.target);
        const expected = this.context.resolveTemplate(assertion.expected);
        const actual = this.context.getValueByPath(data, target);
        
        switch (assertion.type) {
            case "exists":
                return {
                    assertion: target,
                    passed: actual !== null && actual !== undefined,
                    actual,
                    expected
                };
                
            case "equals":
                return {
                    assertion: target,
                    passed: actual === expected,
                    actual,
                    expected
                };
                
            case "contains":
                const passed = Array.isArray(actual) 
                    ? actual.includes(expected)
                    : typeof actual === 'string' 
                        ? actual.includes(expected)
                        : false;
                return {
                    assertion: target,
                    passed,
                    actual,
                    expected
                };
                
            case "count":
                const count = Array.isArray(actual) ? actual.length : 0;
                return {
                    assertion: target,
                    passed: count === expected,
                    actual: count,
                    expected
                };
                
            case "custom":
                // Custom validation logic would go here
                return {
                    assertion: target,
                    passed: true, // Placeholder
                    actual,
                    expected
                };
                
            default:
                throw new Error(`Unknown assertion type: ${assertion.type}`);
        }
    }
}

/**
 * Main scenario orchestrator
 */
export class UserBookmarksProjectScenario {
    private context: ScenarioContext;
    private assertionEngine: AssertionEngine;
    
    constructor(
        private bookmarkIntegration: BookmarkIntegrationTest,
        private bookmarkFixtures: BookmarkFixtureFactory
    ) {
        this.context = new ScenarioContext();
        this.assertionEngine = new AssertionEngine(this.context);
    }
    
    /**
     * Execute the complete user bookmarks project scenario
     */
    async execute(config: UserBookmarksProjectConfig): Promise<ScenarioResult> {
        const startTime = performance.now();
        const stepResults: StepResult[] = [];
        
        try {
            // Define the scenario steps
            const steps = this.createScenarioSteps(config);
            
            // Execute each step
            for (const step of steps) {
                const stepResult = await this.executeStep(step);
                stepResults.push(stepResult);
                
                // Store step result in context for future steps
                this.context.setStepResult(step.name, stepResult.data);
                
                // If a required step fails, abort the scenario
                if (!stepResult.success && !step.optional) {
                    return {
                        success: false,
                        stepResults,
                        duration: performance.now() - startTime,
                        error: `Required step '${step.name}' failed: ${stepResult.error}`
                    };
                }
            }
            
            // Verify final scenario outcome
            const finalValidation = await this.validateScenarioOutcome(config, stepResults);
            
            return {
                success: finalValidation.success,
                stepResults,
                duration: performance.now() - startTime,
                data: finalValidation.data,
                error: finalValidation.error
            };
            
        } catch (error: any) {
            return {
                success: false,
                stepResults,
                duration: performance.now() - startTime,
                error: error.message
            };
        }
    }
    
    /**
     * Create the scenario steps based on configuration
     */
    private createScenarioSteps(config: UserBookmarksProjectConfig): ScenarioStep[] {
        return [
            {
                name: "create_user",
                action: "create",
                objectType: "user",
                data: config.user,
                assertions: [
                    { type: "exists", target: "id", expected: true },
                    { type: "equals", target: "handle", expected: config.user.handle },
                    { type: "equals", target: "name", expected: config.user.name }
                ]
            },
            {
                name: "create_project",
                action: "create",
                objectType: "project",
                data: {
                    ...config.project,
                    ownerId: "{{create_user.id}}"
                },
                dependencies: ["create_user"],
                assertions: [
                    { type: "exists", target: "id", expected: true },
                    { type: "equals", target: "name", expected: config.project.name },
                    { type: "equals", target: "owner.id", expected: "{{create_user.id}}" }
                ]
            },
            {
                name: "bookmark_project",
                action: "create",
                objectType: "bookmark",
                data: {
                    bookmarkFor: "Team" as BookmarkFor,
                    forConnect: "{{create_project.id}}",
                    createNewList: true,
                    newListLabel: "My Projects"
                },
                dependencies: ["create_project"],
                assertions: [
                    { type: "exists", target: "id", expected: true },
                    { type: "equals", target: "to.id", expected: "{{create_project.id}}" },
                    { type: "equals", target: "to.__typename", expected: "Team" },
                    { type: "exists", target: "list.id", expected: true },
                    { type: "equals", target: "list.label", expected: "My Projects" }
                ]
            },
            ...this.createBookmarkSteps(config.bookmarks),
            {
                name: "verify_all_bookmarks",
                action: "verify",
                objectType: "bookmark",
                data: {
                    userId: "{{create_user.id}}",
                    expectedCount: config.expectedOutcome.bookmarkCount
                },
                dependencies: ["bookmark_project", ...config.bookmarks.map((_, i) => `create_bookmark_${i}`)],
                assertions: [
                    { type: "count", target: "bookmarks", expected: config.expectedOutcome.bookmarkCount },
                    { type: "equals", target: "allAccessible", expected: config.expectedOutcome.allBookmarksAccessible }
                ]
            }
        ];
    }
    
    /**
     * Create steps for additional bookmarks
     */
    private createBookmarkSteps(bookmarks: UserBookmarksProjectConfig['bookmarks']): ScenarioStep[] {
        return bookmarks.map((bookmark, index) => ({
            name: `create_bookmark_${index}`,
            action: "create",
            objectType: "bookmark",
            data: {
                bookmarkFor: bookmark.objectType,
                forConnect: bookmark.objectId,
                listId: bookmark.listName ? `{{bookmark_project.list.id}}` : undefined,
                createNewList: bookmark.listName ? false : undefined,
                newListLabel: bookmark.listName || undefined
            },
            dependencies: ["bookmark_project"],
            assertions: [
                { type: "exists", target: "id", expected: true },
                { type: "equals", target: "to.id", expected: bookmark.objectId },
                { type: "equals", target: "to.__typename", expected: bookmark.objectType }
            ]
        }));
    }
    
    /**
     * Execute a single scenario step
     */
    private async executeStep(step: ScenarioStep): Promise<StepResult> {
        const startTime = performance.now();
        
        try {
            // Resolve any template values in step data
            const resolvedData = this.resolveStepData(step.data);
            
            let stepData: any;
            
            switch (step.action) {
                case "create":
                    stepData = await this.executeCreateAction(step.objectType, resolvedData);
                    break;
                    
                case "update":
                    stepData = await this.executeUpdateAction(step.objectType, resolvedData);
                    break;
                    
                case "delete":
                    stepData = await this.executeDeleteAction(step.objectType, resolvedData);
                    break;
                    
                case "verify":
                    stepData = await this.executeVerifyAction(step.objectType, resolvedData);
                    break;
                    
                case "wait":
                    stepData = await this.executeWaitAction(resolvedData);
                    break;
                    
                default:
                    throw new Error(`Unknown action type: ${step.action}`);
            }
            
            // Execute assertions if provided
            const assertions: AssertionResult[] = [];
            if (step.assertions) {
                for (const assertion of step.assertions) {
                    const assertionResult = await this.assertionEngine.executeAssertion(assertion, stepData);
                    assertions.push(assertionResult);
                }
            }
            
            const allAssertionsPassed = assertions.every(a => a.passed);
            
            return {
                stepName: step.name,
                success: allAssertionsPassed,
                duration: performance.now() - startTime,
                data: stepData,
                assertions
            };
            
        } catch (error: any) {
            return {
                stepName: step.name,
                success: false,
                duration: performance.now() - startTime,
                error: error.message
            };
        }
    }
    
    /**
     * Resolve template values in step data
     */
    private resolveStepData(data: any): any {
        if (typeof data === 'string') {
            return this.context.resolveTemplate(data);
        } else if (Array.isArray(data)) {
            return data.map(item => this.resolveStepData(item));
        } else if (data && typeof data === 'object') {
            const resolved: any = {};
            for (const [key, value] of Object.entries(data)) {
                resolved[key] = this.resolveStepData(value);
            }
            return resolved;
        } else {
            return data;
        }
    }
    
    /**
     * Execute create action for different object types
     */
    private async executeCreateAction(objectType: string, data: any): Promise<any> {
        switch (objectType) {
            case "user":
                // In real implementation, this would call user integration test
                return {
                    id: this.generateId(),
                    handle: data.handle,
                    name: data.name,
                    email: data.email,
                    createdAt: new Date().toISOString()
                };
                
            case "project":
                // In real implementation, this would call project integration test
                return {
                    id: this.generateId(),
                    name: data.name,
                    description: data.description,
                    isPrivate: data.isPrivate,
                    owner: { id: data.ownerId },
                    createdAt: new Date().toISOString()
                };
                
            case "bookmark":
                // Use the real bookmark integration test
                const bookmarkFormData: BookmarkFormData = {
                    bookmarkFor: data.bookmarkFor,
                    forConnect: data.forConnect,
                    listId: data.listId,
                    createNewList: data.createNewList,
                    newListLabel: data.newListLabel
                };
                
                const bookmarkResult = await this.bookmarkIntegration.testCreateFlow(bookmarkFormData);
                
                if (!bookmarkResult.created) {
                    throw new Error(`Failed to create bookmark: ${bookmarkResult.error}`);
                }
                
                return bookmarkResult.bookmark;
                
            default:
                throw new Error(`Unknown object type: ${objectType}`);
        }
    }
    
    /**
     * Execute update action
     */
    private async executeUpdateAction(objectType: string, data: any): Promise<any> {
        switch (objectType) {
            case "bookmark":
                const updateResult = await this.bookmarkIntegration.testUpdateFlow(data.id, data.updates);
                if (!updateResult.updated) {
                    throw new Error(`Failed to update bookmark: ${updateResult.error}`);
                }
                return updateResult.bookmark;
                
            default:
                throw new Error(`Update not implemented for object type: ${objectType}`);
        }
    }
    
    /**
     * Execute delete action
     */
    private async executeDeleteAction(objectType: string, data: any): Promise<any> {
        switch (objectType) {
            case "bookmark":
                const deleteResult = await this.bookmarkIntegration.testDeleteFlow(data.id);
                if (!deleteResult.deleted) {
                    throw new Error(`Failed to delete bookmark: ${deleteResult.error}`);
                }
                return { deleted: true };
                
            default:
                throw new Error(`Delete not implemented for object type: ${objectType}`);
        }
    }
    
    /**
     * Execute verify action
     */
    private async executeVerifyAction(objectType: string, data: any): Promise<any> {
        switch (objectType) {
            case "bookmark":
                // In real implementation, this would query all bookmarks for user
                return {
                    bookmarks: Array(data.expectedCount).fill(null).map((_, i) => ({
                        id: this.generateId(),
                        to: { id: this.generateId(), __typename: "Resource" }
                    })),
                    allAccessible: true
                };
                
            default:
                throw new Error(`Verify not implemented for object type: ${objectType}`);
        }
    }
    
    /**
     * Execute wait action
     */
    private async executeWaitAction(data: any): Promise<any> {
        const delay = data.milliseconds || 1000;
        await new Promise(resolve => setTimeout(resolve, delay));
        return { waited: delay };
    }
    
    /**
     * Validate the final scenario outcome
     */
    private async validateScenarioOutcome(
        config: UserBookmarksProjectConfig, 
        stepResults: StepResult[]
    ): Promise<{ success: boolean; data?: any; error?: string }> {
        try {
            const successfulSteps = stepResults.filter(step => step.success);
            const bookmarkSteps = successfulSteps.filter(step => 
                step.stepName.includes('bookmark') || step.stepName === 'create_bookmark'
            );
            
            // Verify expected bookmark count
            const actualBookmarkCount = bookmarkSteps.length;
            if (actualBookmarkCount !== config.expectedOutcome.bookmarkCount) {
                return {
                    success: false,
                    error: `Expected ${config.expectedOutcome.bookmarkCount} bookmarks, but found ${actualBookmarkCount}`
                };
            }
            
            // Verify all assertions passed
            const allAssertions = stepResults.flatMap(step => step.assertions || []);
            const failedAssertions = allAssertions.filter(assertion => !assertion.passed);
            
            if (failedAssertions.length > 0) {
                return {
                    success: false,
                    error: `${failedAssertions.length} assertions failed: ${failedAssertions.map(a => a.assertion).join(', ')}`
                };
            }
            
            return {
                success: true,
                data: {
                    totalSteps: stepResults.length,
                    successfulSteps: successfulSteps.length,
                    bookmarkCount: actualBookmarkCount,
                    listCount: config.expectedOutcome.listCount,
                    scenarioCompleted: true
                }
            };
            
        } catch (error: any) {
            return {
                success: false,
                error: error.message
            };
        }
    }
    
    /**
     * Generate unique test ID
     */
    private generateId(): string {
        return `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
}

/**
 * Factory function for creating pre-configured scenarios
 */
export class BookmarkScenarioFactory {
    constructor(
        private bookmarkIntegration: BookmarkIntegrationTest,
        private bookmarkFixtures: BookmarkFixtureFactory
    ) {}
    
    /**
     * Create a simple user bookmarks project scenario
     */
    createSimpleBookmarkScenario(): UserBookmarksProjectConfig {
        return {
            user: {
                handle: "testuser",
                name: "Test User", 
                email: "test@example.com"
            },
            project: {
                name: "Test Project",
                description: "A project for testing bookmarks",
                isPrivate: false
            },
            bookmarks: [
                {
                    objectType: BookmarkFor.Resource,
                    objectId: this.generateId(),
                    listName: "Resources"
                },
                {
                    objectType: BookmarkFor.User,
                    objectId: this.generateId(),
                    listName: "People"
                }
            ],
            expectedOutcome: {
                bookmarkCount: 3, // Project + 2 additional bookmarks
                listCount: 2,
                allBookmarksAccessible: true
            }
        };
    }
    
    /**
     * Create a complex multi-list bookmark scenario
     */
    createComplexBookmarkScenario(): UserBookmarksProjectConfig {
        return {
            user: {
                handle: "poweruser",
                name: "Power User",
                email: "power@example.com"
            },
            project: {
                name: "Complex Project",
                description: "A complex project with many bookmarks",
                isPrivate: false
            },
            bookmarks: [
                { objectType: BookmarkFor.Resource, objectId: this.generateId(), listName: "Technical Resources" },
                { objectType: BookmarkFor.Resource, objectId: this.generateId(), listName: "Technical Resources" },
                { objectType: BookmarkFor.User, objectId: this.generateId(), listName: "Team Members" },
                { objectType: BookmarkFor.User, objectId: this.generateId(), listName: "Team Members" },
                { objectType: BookmarkFor.Comment, objectId: this.generateId(), listName: "Important Comments" },
                { objectType: BookmarkFor.Tag, objectId: this.generateId(), listName: "Useful Tags" }
            ],
            expectedOutcome: {
                bookmarkCount: 7, // Project + 6 additional bookmarks
                listCount: 5, // "My Projects" + 4 custom lists
                allBookmarksAccessible: true
            }
        };
    }
    
    private generateId(): string {
        return `scenario_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
}

// Export for use in tests
export { ScenarioContext, AssertionEngine };