import type { Meta, StoryObj } from "@storybook/react";
import { ChatConfigObject, ExecutionStates, PendingToolCallStatus } from "@vrooli/shared";
import { SwarmDetailPanel } from "./SwarmDetailPanel.js";

const meta: Meta<typeof SwarmDetailPanel> = {
    title: "Components/Swarm/SwarmDetailPanel",
    component: SwarmDetailPanel,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

const baseSwarmConfig: ChatConfigObject = {
    goal: "Analyze market trends and generate a comprehensive report",
    subtasks: [
        {
            id: "task-1",
            description: "Gather market data from various sources",
            status: "done",
            priority: "high",
            assignee_bot_id: "bot-researcher-1",
            depends_on: [],
        },
        {
            id: "task-2",
            description: "Clean and normalize collected data",
            status: "done",
            priority: "medium",
            assignee_bot_id: "bot-processor-1",
            depends_on: ["task-1"],
        },
        {
            id: "task-3",
            description: "Perform statistical analysis",
            status: "in_progress",
            priority: "high",
            assignee_bot_id: "bot-analyst-1",
            depends_on: ["task-2"],
        },
        {
            id: "task-4",
            description: "Generate visualizations",
            status: "blocked",
            priority: "medium",
            assignee_bot_id: "bot-visualizer-1",
            depends_on: ["task-3"],
        },
        {
            id: "task-5",
            description: "Compile final report",
            status: "todo",
            priority: "high",
            assignee_bot_id: null,
            depends_on: ["task-3", "task-4"],
        },
    ],
    swarmLeader: "bot-coordinator-1",
    subtaskLeaders: {
        "task-1": "bot-researcher-1",
        "task-2": "bot-processor-1",
        "task-3": "bot-analyst-1",
        "task-4": "bot-visualizer-1",
    },
    resources: [
        {
            id: "resource-1",
            kind: "File",
            mime: "text/csv",
            meta: {
                size: "15.2MB",
                rows: "50000",
                source: "market-data-2024",
            },
        },
        {
            id: "resource-2",
            kind: "URL",
            mime: "application/json",
            meta: {
                endpoint: "https://api.example.com/trends",
                last_updated: "2024-01-15",
            },
        },
        {
            id: "resource-3",
            kind: "Image",
            mime: "image/png",
            meta: {
                width: "1920",
                height: "1080",
                type: "chart",
            },
        },
    ],
    pendingToolCalls: [
        {
            pendingId: "pending-1",
            toolName: "WriteFile",
            toolArguments: JSON.stringify({
                path: "/reports/market-analysis.pdf",
                content: "Base64 encoded PDF content...",
            }),
            callerBotId: "bot-reporter-1",
            status: PendingToolCallStatus.PENDING_APPROVAL,
            approvalTimeoutAt: Date.now() + 300000, // 5 minutes
        },
        {
            pendingId: "pending-2",
            toolName: "SendEmail",
            toolArguments: JSON.stringify({
                to: ["stakeholders@example.com"],
                subject: "Market Analysis Report Ready",
                body: "Please find the attached market analysis report...",
            }),
            callerBotId: "bot-notifier-1",
            status: PendingToolCallStatus.PENDING_APPROVAL,
            approvalTimeoutAt: Date.now() + 45000, // 45 seconds - urgent!
        },
    ],
    records: [
        {
            id: "record-1",
            routine_name: "DataCollection",
            caller_bot_id: "bot-researcher-1",
            created_at: new Date(Date.now() - 3600000).toISOString(),
            output_resource_ids: ["resource-1"],
        },
        {
            id: "record-2",
            routine_name: "DataNormalization",
            caller_bot_id: "bot-processor-1",
            created_at: new Date(Date.now() - 1800000).toISOString(),
            output_resource_ids: [],
        },
        {
            id: "record-3",
            routine_name: "ChartGeneration",
            caller_bot_id: "bot-visualizer-1",
            created_at: new Date(Date.now() - 600000).toISOString(),
            output_resource_ids: ["resource-3"],
        },
    ],
    stats: {
        totalCredits: "125000",
        totalToolCalls: 47,
        startedAt: Date.now() - 7200000, // 2 hours ago
    },
    teamId: "team-analytics",
};

export const Default: Story = {
    args: {
        swarmConfig: baseSwarmConfig,
        swarmStatus: ExecutionStates.RUNNING,
        onApproveToolCall: (pendingId: string) => {
            console.log("Approved tool call:", pendingId);
        },
        onRejectToolCall: (pendingId: string, reason?: string) => {
            console.log("Rejected tool call:", pendingId, reason);
        },
    },
};

export const NoSwarmActive: Story = {
    args: {
        swarmConfig: null,
        swarmStatus: ExecutionStates.UNINITIALIZED,
    },
};

export const Paused: Story = {
    args: {
        ...Default.args,
        swarmStatus: ExecutionStates.PAUSED,
    },
};

export const Failed: Story = {
    args: {
        ...Default.args,
        swarmStatus: ExecutionStates.FAILED,
        swarmConfig: {
            ...baseSwarmConfig,
            subtasks: baseSwarmConfig.subtasks?.map((task, index) => ({
                ...task,
                status: index === 2 ? "failed" : index < 2 ? "done" : "canceled",
            })),
        },
    },
};

export const NoResources: Story = {
    args: {
        ...Default.args,
        swarmConfig: {
            ...baseSwarmConfig,
            resources: [],
        },
    },
};

export const NoPendingApprovals: Story = {
    args: {
        ...Default.args,
        swarmConfig: {
            ...baseSwarmConfig,
            pendingToolCalls: [],
        },
    },
};

export const CompleteSwarm: Story = {
    args: {
        ...Default.args,
        swarmStatus: ExecutionStates.COMPLETED,
        swarmConfig: {
            ...baseSwarmConfig,
            subtasks: baseSwarmConfig.subtasks?.map(task => ({
                ...task,
                status: "done",
            })),
            pendingToolCalls: [],
            stats: {
                ...baseSwarmConfig.stats,
                totalCredits: "285000",
                totalToolCalls: 142,
            },
        },
    },
};