/**
 * Multi-user collaboration event fixtures for testing real-time collaborative features
 */

import { type RunTaskInfo, type DeferredDecisionData } from "../../../run/types.js";
import { type RunSocketEventPayloads } from "../../../consts/socketEvents.js";
import { runFixtures } from "../api/runFixtures.js";
import { userFixtures } from "../api/userFixtures.js";

export const collaborationEventFixtures = {
    runTask: {
        // Task created
        taskCreated: {
            event: "runTask",
            data: {
                id: "task_123",
                runId: "run_456",
                status: "pending",
                name: "Data Processing",
                description: "Process input data for analysis",
                order: 1,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies RunTaskInfo,
        },

        // Task in progress
        taskInProgress: {
            event: "runTask",
            data: {
                id: "task_123",
                runId: "run_456",
                status: "in_progress",
                name: "Data Processing",
                description: "Process input data for analysis",
                order: 1,
                startedAt: new Date().toISOString(),
                assignedTo: userFixtures.minimal.find,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies RunTaskInfo,
        },

        // Task completed
        taskCompleted: {
            event: "runTask",
            data: {
                id: "task_123",
                runId: "run_456",
                status: "completed",
                name: "Data Processing",
                description: "Process input data for analysis",
                order: 1,
                startedAt: new Date(Date.now() - 60000).toISOString(),
                completedAt: new Date().toISOString(),
                assignedTo: userFixtures.minimal.find,
                result: {
                    processed: 1000,
                    errors: 0,
                },
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies RunTaskInfo,
        },

        // Task failed
        taskFailed: {
            event: "runTask",
            data: {
                id: "task_123",
                runId: "run_456",
                status: "failed",
                name: "Data Processing",
                description: "Process input data for analysis",
                order: 1,
                startedAt: new Date(Date.now() - 30000).toISOString(),
                failedAt: new Date().toISOString(),
                assignedTo: userFixtures.minimal.find,
                error: "Invalid data format",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } satisfies RunTaskInfo,
        },

        // Parallel tasks
        parallelTasks: [
            {
                id: "task_parallel_1",
                runId: "run_456",
                status: "in_progress",
                name: "Fetch Data",
                order: 1,
                assignedTo: userFixtures.minimal.find,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            },
            {
                id: "task_parallel_2",
                runId: "run_456",
                status: "in_progress",
                name: "Validate Schema",
                order: 1,
                assignedTo: userFixtures.complete.find,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            },
        ] satisfies RunTaskInfo[],
    },

    decisionRequest: {
        // Simple decision
        simpleDecision: {
            event: "runTaskDecisionRequest",
            data: {
                id: "decision_123",
                runId: "run_456",
                taskId: "task_789",
                type: "boolean",
                title: "Proceed with analysis?",
                description: "The data contains anomalies. Do you want to continue?",
                options: [
                    { id: "yes", label: "Yes, continue", value: true },
                    { id: "no", label: "No, stop here", value: false },
                ],
                requestedBy: userFixtures.minimal.find,
                requestedAt: new Date().toISOString(),
                timeoutAt: new Date(Date.now() + 300000).toISOString(), // 5 minutes
            } satisfies DeferredDecisionData,
        },

        // Multiple choice decision
        multipleChoice: {
            event: "runTaskDecisionRequest",
            data: {
                id: "decision_multi",
                runId: "run_456",
                taskId: "task_789",
                type: "choice",
                title: "Select processing method",
                description: "Choose how to handle the duplicate records",
                options: [
                    { id: "merge", label: "Merge duplicates", value: "merge" },
                    { id: "keep_first", label: "Keep first occurrence", value: "keep_first" },
                    { id: "keep_last", label: "Keep last occurrence", value: "keep_last" },
                    { id: "keep_all", label: "Keep all records", value: "keep_all" },
                ],
                requestedBy: userFixtures.minimal.find,
                requestedAt: new Date().toISOString(),
                timeoutAt: new Date(Date.now() + 600000).toISOString(), // 10 minutes
            } satisfies DeferredDecisionData,
        },

        // Input decision
        inputDecision: {
            event: "runTaskDecisionRequest",
            data: {
                id: "decision_input",
                runId: "run_456",
                taskId: "task_789",
                type: "input",
                title: "Provide API key",
                description: "Enter your API key for the external service",
                inputType: "password",
                validation: {
                    required: true,
                    pattern: "^[A-Za-z0-9]{32}$",
                },
                requestedBy: userFixtures.minimal.find,
                requestedAt: new Date().toISOString(),
                timeoutAt: new Date(Date.now() + 900000).toISOString(), // 15 minutes
            } satisfies DeferredDecisionData,
        },

        // Approval decision
        approvalDecision: {
            event: "runTaskDecisionRequest",
            data: {
                id: "decision_approval",
                runId: "run_456",
                taskId: "task_789",
                type: "approval",
                title: "Approve deployment",
                description: "Review and approve the deployment to production",
                metadata: {
                    environment: "production",
                    changes: ["Update API endpoints", "Add new features", "Fix bugs"],
                    risk: "medium",
                },
                options: [
                    { id: "approve", label: "Approve", value: "approved" },
                    { id: "reject", label: "Reject", value: "rejected" },
                    { id: "defer", label: "Defer", value: "deferred" },
                ],
                requestedBy: userFixtures.minimal.find,
                requestedAt: new Date().toISOString(),
                requiresComment: true,
            } satisfies DeferredDecisionData,
        },
    },

    // Event sequences for testing collaboration flows
    sequences: {
        // Sequential task execution
        sequentialTasks: [
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCreated.data, name: "Step 1: Initialize" } },
            { delay: 2000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, name: "Step 1: Initialize" } },
            { delay: 5000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCompleted.data, name: "Step 1: Initialize" } },
            { delay: 1000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCreated.data, id: "task_124", name: "Step 2: Process", order: 2 } },
            { delay: 2000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, id: "task_124", name: "Step 2: Process", order: 2 } },
        ],

        // Parallel task execution
        parallelExecution: [
            { event: "runTask", data: collaborationEventFixtures.runTask.parallelTasks[0] },
            { event: "runTask", data: collaborationEventFixtures.runTask.parallelTasks[1] },
            { delay: 3000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.parallelTasks[0], status: "completed" } },
            { delay: 2000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.parallelTasks[1], status: "completed" } },
            { delay: 500 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCreated.data, id: "task_merge", name: "Merge Results", order: 2 } },
        ],

        // Decision flow
        decisionFlow: [
            { event: "runTask", data: collaborationEventFixtures.runTask.taskInProgress.data },
            { delay: 3000 },
            { event: "runTaskDecisionRequest", data: collaborationEventFixtures.decisionRequest.simpleDecision.data },
            { delay: 10000 }, // User thinking time
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, metadata: { decision: "yes" } } },
            { delay: 2000 },
            { event: "runTask", data: collaborationEventFixtures.runTask.taskCompleted.data },
        ],

        // Multi-user collaboration
        multiUserFlow: [
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCreated.data, assignedTo: userFixtures.minimal.find } },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCreated.data, id: "task_124", assignedTo: userFixtures.complete.find } },
            { delay: 1000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, assignedTo: userFixtures.minimal.find } },
            { delay: 500 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, id: "task_124", assignedTo: userFixtures.complete.find } },
            { delay: 4000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCompleted.data, assignedTo: userFixtures.minimal.find } },
            { delay: 2000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCompleted.data, id: "task_124", assignedTo: userFixtures.complete.find } },
        ],

        // Error recovery with decision
        errorRecoveryFlow: [
            { event: "runTask", data: collaborationEventFixtures.runTask.taskInProgress.data },
            { delay: 3000 },
            { event: "runTask", data: collaborationEventFixtures.runTask.taskFailed.data },
            { delay: 1000 },
            { event: "runTaskDecisionRequest", data: {
                ...collaborationEventFixtures.decisionRequest.simpleDecision.data,
                title: "Retry failed task?",
                description: "The task failed with error: Invalid data format. Do you want to retry?",
            }},
            { delay: 5000 },
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskInProgress.data, metadata: { retry: true } } },
            { delay: 3000 },
            { event: "runTask", data: collaborationEventFixtures.runTask.taskCompleted.data },
        ],

        // Approval workflow
        approvalWorkflow: [
            { event: "runTask", data: { ...collaborationEventFixtures.runTask.taskCompleted.data, name: "Build Complete" } },
            { delay: 1000 },
            { event: "runTaskDecisionRequest", data: collaborationEventFixtures.decisionRequest.approvalDecision.data },
            { delay: 20000 }, // Manager review time
            { event: "runTask", data: {
                ...collaborationEventFixtures.runTask.taskCreated.data,
                id: "task_deploy",
                name: "Deploy to Production",
                metadata: { approved: true, approvedBy: "manager_123" },
            }},
            { delay: 2000 },
            { event: "runTask", data: {
                ...collaborationEventFixtures.runTask.taskInProgress.data,
                id: "task_deploy",
                name: "Deploy to Production",
            }},
        ],
    },

    // Factory functions for dynamic event creation
    factories: {
        createRunTask: (runId: string, task: Partial<RunTaskInfo>) => ({
            event: "runTask",
            data: {
                id: `task_${Date.now()}`,
                runId,
                status: "pending",
                name: "Task",
                order: 1,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                ...task,
            } satisfies RunTaskInfo,
        }),

        createDecisionRequest: (runId: string, taskId: string, decision: Partial<DeferredDecisionData>) => ({
            event: "runTaskDecisionRequest",
            data: {
                id: `decision_${Date.now()}`,
                runId,
                taskId,
                type: "boolean",
                title: "Decision Required",
                description: "Please make a decision",
                options: [
                    { id: "yes", label: "Yes", value: true },
                    { id: "no", label: "No", value: false },
                ],
                requestedBy: userFixtures.minimal.find,
                requestedAt: new Date().toISOString(),
                ...decision,
            } satisfies DeferredDecisionData,
        }),

        createTaskUpdate: (taskId: string, updates: Partial<RunTaskInfo>) => ({
            event: "runTask",
            data: {
                id: taskId,
                updated_at: new Date().toISOString(),
                ...updates,
            } as RunTaskInfo,
        }),
    },
};