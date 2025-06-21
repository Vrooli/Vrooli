# Scenario Orchestrators

This directory contains multi-step workflow orchestrators that test complex user scenarios involving multiple objects and operations. These scenarios go beyond simple CRUD operations to test realistic user workflows.

## Purpose

Scenario orchestrators provide:
- **Multi-step workflow testing** for complex user journeys
- **Cross-object relationship testing** between different entities
- **State management validation** across multiple operations
- **User story verification** through automated scenarios
- **Business logic validation** in realistic contexts

## Architecture

Each scenario orchestrator follows this pattern:

```typescript
export class UserScenarioOrchestrator {
    constructor(
        private integrationTests: Record<string, IntegrationTest>,
        private fixtureFactories: Record<string, FixtureFactory>,
        private testContext: TestContext
    ) {}
    
    async execute(config: ScenarioConfig): Promise<ScenarioResult> {
        const stepResults: StepResult[] = [];
        
        for (const step of config.steps) {
            const result = await this.executeStep(step);
            stepResults.push(result);
            
            if (!result.success && step.required) {
                return { success: false, stepResults, error: result.error };
            }
        }
        
        return { success: true, stepResults };
    }
}
```

## Scenario Types

### 1. User Journey Scenarios
Test complete user workflows from start to finish:
- User registration → Profile setup → First bookmark creation
- Project creation → Team member invitation → Resource bookmarking
- Content discovery → Multiple bookmarks → List organization

### 2. Cross-Object Scenarios  
Test relationships between different object types:
- User creates team → Team creates project → User bookmarks project
- User creates routine → Comments on routine → Bookmarks comment
- Resource creation → Version management → User bookmarks resource

### 3. Permission Scenarios
Test access control across different contexts:
- Public project bookmarking by different users
- Private resource access and bookmark attempts
- Team member permissions for project bookmarking

### 4. Error Recovery Scenarios
Test system behavior during failures:
- Network interruption during multi-step operations
- Partial completion with rollback requirements
- Data corruption detection and recovery

## Example Scenario Implementation

### User Bookmarks Project Scenario
```typescript
export class UserBookmarksProjectScenario {
    async execute(config: {
        user: UserFormData;
        project: ProjectFormData;
        bookmarks: string[];
    }): Promise<ScenarioResult> {
        const steps: ScenarioStep[] = [
            {
                name: "create_user",
                action: "create",
                objectType: "user",
                data: config.user,
                assertions: [
                    { type: "exists", target: "user.id", expected: true }
                ]
            },
            {
                name: "create_project",
                action: "create", 
                objectType: "project",
                data: config.project,
                dependencies: ["create_user"],
                assertions: [
                    { type: "exists", target: "project.id", expected: true },
                    { type: "equals", target: "project.owner.id", expected: "{{create_user.id}}" }
                ]
            },
            {
                name: "bookmark_project",
                action: "create",
                objectType: "bookmark",
                data: {
                    bookmarkFor: "Team",
                    forConnect: "{{create_project.id}}",
                    createNewList: true,
                    newListLabel: "My Projects"
                },
                dependencies: ["create_project"],
                assertions: [
                    { type: "exists", target: "bookmark.id", expected: true },
                    { type: "equals", target: "bookmark.to.id", expected: "{{create_project.id}}" }
                ]
            }
        ];
        
        return await this.orchestrator.execute({ 
            name: "user_bookmarks_project",
            description: "User creates account, project, and bookmarks it",
            steps 
        });
    }
}
```

## Scenario Configuration

### Scenario Config
```typescript
interface ScenarioConfig {
    name: string;
    description: string;
    steps: ScenarioStep[];
    cleanup?: boolean;
    timeout?: number;
    retries?: number;
}
```

### Scenario Step
```typescript
interface ScenarioStep {
    name: string;
    action: "create" | "update" | "delete" | "verify" | "wait";
    objectType: string;
    data?: any;
    assertions?: Assertion[];
    dependencies?: string[];
    timeout?: number;
    retries?: number;
    optional?: boolean;
}
```

### Assertion Types
```typescript
interface Assertion {
    type: "exists" | "equals" | "contains" | "count" | "custom";
    target: string; // Path to value (e.g., "user.profile.name")
    expected: any;
    message?: string;
    customValidator?: (actual: any, expected: any) => boolean;
}
```

## Data Flow and Dependencies

### Step Dependencies
Steps can depend on results from previous steps:

```typescript
// Step 1: Create user
{
    name: "create_user",
    action: "create",
    objectType: "user",
    data: userFormData
}

// Step 2: Create project (depends on user)
{
    name: "create_project", 
    action: "create",
    objectType: "project",
    data: {
        ...projectFormData,
        ownerId: "{{create_user.id}}" // Reference previous step result
    },
    dependencies: ["create_user"]
}
```

### Context Sharing
Data flows between steps through context:

```typescript
export class ScenarioContext {
    private stepResults: Map<string, any> = new Map();
    
    setStepResult(stepName: string, result: any): void {
        this.stepResults.set(stepName, result);
    }
    
    resolveTemplate(template: string): any {
        // Resolve templates like "{{create_user.id}}"
        return template.replace(/\{\{([^}]+)\}\}/g, (match, path) => {
            return this.getValueByPath(path);
        });
    }
}
```

## Assertion System

### Built-in Assertions
```typescript
export class AssertionEngine {
    async executeAssertion(assertion: Assertion, data: any): Promise<AssertionResult> {
        const actual = this.getValueByPath(data, assertion.target);
        
        switch (assertion.type) {
            case "exists":
                return {
                    assertion: assertion.target,
                    passed: actual !== null && actual !== undefined,
                    actual,
                    expected: assertion.expected
                };
                
            case "equals":
                return {
                    assertion: assertion.target,
                    passed: actual === assertion.expected,
                    actual,
                    expected: assertion.expected
                };
                
            case "contains":
                return {
                    assertion: assertion.target,
                    passed: Array.isArray(actual) ? actual.includes(assertion.expected) : 
                            typeof actual === 'string' ? actual.includes(assertion.expected) : false,
                    actual,
                    expected: assertion.expected
                };
                
            case "count":
                const count = Array.isArray(actual) ? actual.length : 0;
                return {
                    assertion: assertion.target,
                    passed: count === assertion.expected,
                    actual: count,
                    expected: assertion.expected
                };
                
            case "custom":
                return {
                    assertion: assertion.target,
                    passed: assertion.customValidator ? assertion.customValidator(actual, assertion.expected) : false,
                    actual,
                    expected: assertion.expected
                };
                
            default:
                throw new Error(`Unknown assertion type: ${assertion.type}`);
        }
    }
}
```

## Error Handling and Recovery

### Failure Strategies
```typescript
interface StepFailureStrategy {
    retry?: {
        attempts: number;
        delay: number;
        backoff?: "linear" | "exponential";
    };
    fallback?: {
        action: "skip" | "alternative" | "abort";
        alternativeStep?: ScenarioStep;
    };
    cleanup?: {
        required: boolean;
        steps: string[];
    };
}
```

### Rollback Support
```typescript
export class ScenarioRollback {
    async rollbackScenario(scenarioResult: ScenarioResult): Promise<void> {
        // Rollback steps in reverse order
        const completedSteps = scenarioResult.stepResults
            .filter(step => step.success)
            .reverse();
            
        for (const step of completedSteps) {
            await this.rollbackStep(step);
        }
    }
}
```

## Performance and Monitoring

### Scenario Metrics
```typescript
interface ScenarioMetrics {
    totalDuration: number;
    stepDurations: Record<string, number>;
    assertionCounts: {
        total: number;
        passed: number;
        failed: number;
    };
    resourceUsage: {
        memoryPeak: number;
        cpuTime: number;
        networkRequests: number;
    };
}
```

### Parallel Execution
```typescript
export class ParallelScenarioExecutor {
    async executeParallel(scenarios: ScenarioConfig[]): Promise<ScenarioResult[]> {
        // Execute independent scenarios in parallel
        const independentScenarios = this.groupByDependencies(scenarios);
        
        const results: ScenarioResult[] = [];
        
        for (const group of independentScenarios) {
            const groupResults = await Promise.all(
                group.map(scenario => this.orchestrator.execute(scenario))
            );
            results.push(...groupResults);
        }
        
        return results;
    }
}
```

## Usage Examples

### Simple Bookmark Scenario
```typescript
describe('Bookmark Scenarios', () => {
    it('should execute user bookmarks project scenario', async () => {
        const scenario = new UserBookmarksProjectScenario(
            integrationTests,
            fixtureFactories,
            testContext
        );
        
        const result = await scenario.execute({
            user: userFixtures.createFormData('complete'),
            project: projectFixtures.createFormData('public'),
            bookmarks: ['resource1', 'resource2']
        });
        
        expect(result.success).toBe(true);
        expect(result.stepResults).toHaveLength(3);
        expect(result.stepResults.every(step => step.success)).toBe(true);
    });
});
```

### Complex Multi-User Scenario
```typescript
describe('Multi-User Scenarios', () => {
    it('should handle team collaboration workflow', async () => {
        const scenario = new TeamCollaborationScenario();
        
        const result = await scenario.execute({
            teamLead: userFixtures.createFormData('teamLead'),
            members: [
                userFixtures.createFormData('developer'),
                userFixtures.createFormData('designer')
            ],
            project: projectFixtures.createFormData('collaborative'),
            workflows: ['design', 'development', 'review']
        });
        
        expect(result.success).toBe(true);
        
        // Verify all team members can bookmark shared resources
        const bookmarkSteps = result.stepResults.filter(step => 
            step.stepName.includes('bookmark')
        );
        expect(bookmarkSteps).toHaveLength(6); // 2 members × 3 resources
        expect(bookmarkSteps.every(step => step.success)).toBe(true);
    });
});
```

## Best Practices

### DO's ✅
- Design scenarios based on real user stories
- Include both happy path and error scenarios
- Use meaningful step names and descriptions
- Implement proper cleanup and rollback
- Test cross-object relationships thoroughly
- Include performance assertions
- Use dependency management for step ordering

### DON'Ts ❌
- Create overly complex scenarios that are hard to debug
- Skip error handling and recovery testing
- Hardcode data that should be parameterized
- Create scenarios without clear success criteria
- Ignore performance implications of long scenarios
- Mix unit test concerns with scenario testing
- Create scenarios that depend on external services

## Troubleshooting

### Common Issues
1. **Step Dependencies**: Ensure proper dependency ordering
2. **Context Resolution**: Verify template resolution works correctly
3. **Assertion Failures**: Check assertion logic and expected values
4. **Timeout Issues**: Adjust timeouts for slow operations
5. **Resource Cleanup**: Ensure proper cleanup prevents test pollution

### Debug Strategies
- Enable detailed step logging
- Use step-by-step execution mode
- Verify context data at each step
- Test individual steps in isolation
- Monitor resource usage during execution

This scenario-based testing approach provides confidence that complex user workflows function correctly across the entire application.