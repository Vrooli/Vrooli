/**
 * Example usage of the ExecutionArchitecture
 * 
 * This demonstrates how external systems can interact with the
 * unified execution architecture.
 */

import { getExecutionArchitecture, type ExecutionArchitecture } from './executionArchitecture.js';
import {
    type TierExecutionRequest,
    type SwarmCoordinationInput,
    type RoutineExecutionInput,
    type StepExecutionInput,
    generatePk,
} from '@vrooli/shared';

/**
 * Example 1: Using the architecture for swarm coordination (Tier 1)
 */
async function exampleSwarmCoordination() {
    // Get the architecture instance
    const architecture = await getExecutionArchitecture({
        useRedis: false, // Use in-memory for this example
        telemetryEnabled: true,
    });
    
    // Get Tier 1 interface
    const tier1 = architecture.getTier1();
    
    // Create a swarm coordination request
    const request: TierExecutionRequest<SwarmCoordinationInput> = {
        context: {
            executionId: generatePk(),
            swarmId: generatePk(),
            userId: 'user123',
            timestamp: new Date(),
            correlationId: generatePk(),
        },
        input: {
            goal: 'Research and summarize the latest AI trends',
            availableAgents: [
                {
                    id: 'agent1',
                    name: 'Research Agent',
                    capabilities: ['web_search', 'content_analysis'],
                    currentLoad: 0.3,
                    maxConcurrentTasks: 5,
                },
                {
                    id: 'agent2',
                    name: 'Writing Agent',
                    capabilities: ['text_generation', 'summarization'],
                    currentLoad: 0.5,
                    maxConcurrentTasks: 3,
                },
            ],
        },
        allocation: {
            maxCredits: '1000',
            maxDurationMs: 300000, // 5 minutes
            maxMemoryMB: 512,
            maxConcurrentSteps: 10,
        },
    };
    
    // Execute the swarm coordination
    const result = await tier1.execute(request);
    
    console.log('Swarm coordination result:', result);
}

/**
 * Example 2: Direct routine execution (Tier 2)
 */
async function exampleRoutineExecution() {
    const architecture = await getExecutionArchitecture();
    const tier2 = architecture.getTier2();
    
    const request: TierExecutionRequest<RoutineExecutionInput> = {
        context: {
            executionId: generatePk(),
            swarmId: 'default',
            userId: 'user123',
            timestamp: new Date(),
            correlationId: generatePk(),
        },
        input: {
            routineId: 'routine123',
            parameters: {
                inputText: 'Analyze this data',
                outputFormat: 'json',
            },
            workflow: {
                steps: [
                    {
                        id: 'step1',
                        name: 'Parse Input',
                        toolName: 'text_parser',
                        parameters: { text: '{{inputText}}' },
                        strategy: 'deterministic',
                    },
                    {
                        id: 'step2',
                        name: 'Analyze Content',
                        toolName: 'content_analyzer',
                        parameters: { data: '{{step1.output}}' },
                        strategy: 'reasoning',
                    },
                ],
                dependencies: [
                    { stepId: 'step2', dependsOn: ['step1'] },
                ],
            },
        },
        allocation: {
            maxCredits: '100',
            maxDurationMs: 60000,
            maxMemoryMB: 256,
            maxConcurrentSteps: 2,
        },
    };
    
    const result = await tier2.execute(request);
    console.log('Routine execution result:', result);
}

/**
 * Example 3: Direct step execution (Tier 3)
 */
async function exampleStepExecution() {
    const architecture = await getExecutionArchitecture();
    const tier3 = architecture.getTier3();
    
    const request: TierExecutionRequest<StepExecutionInput> = {
        context: {
            executionId: generatePk(),
            swarmId: 'default',
            userId: 'user123',
            timestamp: new Date(),
            correlationId: generatePk(),
            stepId: 'step123',
            stepType: 'analysis',
        },
        input: {
            stepId: 'step123',
            stepType: 'content_analysis',
            toolName: 'text_analyzer',
            parameters: {
                text: 'Analyze this sample text',
                mode: 'detailed',
            },
            strategy: 'reasoning',
        },
        allocation: {
            maxCredits: '10',
            maxDurationMs: 30000,
            maxMemoryMB: 128,
            maxConcurrentSteps: 1,
        },
    };
    
    const result = await tier3.execute(request);
    console.log('Step execution result:', result);
}

/**
 * Example 4: Using the architecture with monitoring
 */
async function exampleWithMonitoring() {
    const architecture = await getExecutionArchitecture();
    
    // Get architecture status
    const status = architecture.getStatus();
    console.log('Architecture status:', status);
    
    // Get combined capabilities
    const capabilities = await architecture.getCapabilities();
    console.log('Architecture capabilities:', capabilities);
    
    // Execute a request and monitor it
    const tier1 = architecture.getTier1();
    const executionId = generatePk();
    
    const request: TierExecutionRequest<SwarmCoordinationInput> = {
        context: {
            executionId,
            swarmId: generatePk(),
            userId: 'user123',
            timestamp: new Date(),
            correlationId: generatePk(),
        },
        input: {
            goal: 'Monitor this execution',
            availableAgents: [],
        },
        allocation: {
            maxCredits: '100',
            maxDurationMs: 60000,
            maxMemoryMB: 256,
            maxConcurrentSteps: 5,
        },
    };
    
    // Start execution
    const executionPromise = tier1.execute(request);
    
    // Monitor status
    const monitorInterval = setInterval(async () => {
        const status = await tier1.getExecutionStatus(executionId);
        console.log('Execution status:', status);
    }, 1000);
    
    // Wait for completion
    const result = await executionPromise;
    clearInterval(monitorInterval);
    
    console.log('Final result:', result);
}

/**
 * Example 5: Graceful shutdown
 */
async function exampleShutdown() {
    const architecture = await getExecutionArchitecture();
    
    // Do some work...
    
    // Gracefully stop the architecture
    await architecture.stop();
    console.log('Architecture stopped gracefully');
}

// Export examples for testing
export {
    exampleSwarmCoordination,
    exampleRoutineExecution,
    exampleStepExecution,
    exampleWithMonitoring,
    exampleShutdown,
};