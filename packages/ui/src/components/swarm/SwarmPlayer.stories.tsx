import { Box } from "@mui/material";
import { ChatConfigObject, ExecutionStates, PendingToolCallStatus, SwarmSubTask } from "@vrooli/shared";
import { useState } from "react";
// Simple action replacement
const action = (name: string) => (...args: any[]) => console.log(`Action: ${name}`, args);
import { SwarmPlayer } from "./SwarmPlayer.js";
import { SwarmDetailPanel } from "./SwarmDetailPanel.js";
import { useMenu } from "../../hooks/useMenu.js";
import { ELEMENT_IDS } from "../../utils/consts.js";

// Helper to create mock swarm config
function createMockSwarmConfig(overrides?: Partial<ChatConfigObject>): ChatConfigObject {
    const baseConfig: ChatConfigObject = {
        __version: "1.0",
        goal: "Create a comprehensive market analysis report for Q1 2024",
        preferredModel: "claude-3-opus-20240229",
        subtasks: [
            {
                id: "T1",
                description: "Collect market data from multiple sources",
                status: "done",
                assignee_bot_id: "data_collector_bot_789",
                priority: "high",
                created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
            },
            {
                id: "T2",
                description: "Analyze competitor positioning and pricing strategies",
                status: "in_progress",
                depends_on: ["T1"],
                assignee_bot_id: "competitive_analyst_bot_101",
                priority: "high",
                created_at: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
            },
            {
                id: "T3",
                description: "Identify growth trends and market opportunities",
                status: "todo",
                depends_on: ["T1"],
                assignee_bot_id: "trend_analyst_bot_202",
                priority: "medium",
                created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(),
            },
            {
                id: "T4",
                description: "Generate executive summary with actionable insights",
                status: "todo",
                depends_on: ["T2", "T3"],
                assignee_bot_id: "report_writer_bot_303",
                priority: "medium",
                created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
            },
        ],
        swarmLeader: "market_analysis_leader_bot_abc123",
        teamId: "dynamic_market_team_def456",
        subtaskLeaders: {
            "T1": "data_collector_bot_789",
            "T2": "competitive_analyst_bot_101",
            "T3": "trend_analyst_bot_202",
            "T4": "report_writer_bot_303",
        },
        blackboard: [
            {
                id: "insight_1",
                value: "Tech sector showing 15% growth trend compared to Q4 2023",
                created_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
            },
            {
                id: "insight_2",
                value: "Key competitors: CloudFlare (+25%), Nvidia (+18%), ServiceNow (+14%)",
                created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
            },
        ],
        resources: [
            {
                id: "market_data_q1_2024",
                kind: "File",
                mime: "application/json",
                creator_bot_id: "data_collector_bot_789",
                created_at: new Date(Date.now() - 28 * 60 * 1000).toISOString(),
                meta: { size: 2500000, records: 8500, sources: 3 },
            },
            {
                id: "competitor_analysis_draft",
                kind: "Note",
                creator_bot_id: "competitive_analyst_bot_101",
                created_at: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
            },
        ],
        records: [
            {
                id: "exec_001",
                routine_id: "financial_data_collection_v4",
                routine_name: "Tech Sector Data Collection",
                params: { sector: "technology", timeframe: "2024_Q1" },
                output_resource_ids: ["market_data_q1_2024"],
                caller_bot_id: "data_collector_bot_789",
                created_at: new Date(Date.now() - 28 * 60 * 1000).toISOString(),
            },
        ],
        stats: {
            totalToolCalls: 15,
            totalCredits: "4250",
            startedAt: Date.now() - 35 * 60 * 1000,
            lastProcessingCycleEndedAt: Date.now() - 2 * 60 * 1000,
        },
        limits: {
            maxCredits: "10000",
            maxDurationMs: 7200000, // 2 hours
            maxToolCallsPerBotResponse: 15,
        },
        scheduling: {
            requiresApprovalTools: ["premium_api", "expensive_analysis"],
            approvalTimeoutMs: 300000, // 5 minutes
            autoRejectOnTimeout: true,
        },
        pendingToolCalls: [],
        ...overrides,
    };
    
    return baseConfig;
}

/**
 * Storybook configuration for SwarmPlayer
 */
export default {
    title: "Components/Swarm/SwarmPlayer",
    component: SwarmPlayer,
};

/**
 * Default story: Active swarm in running state
 */
export function ActiveSwarm() {
    const [swarmStatus, setSwarmStatus] = useState(ExecutionStates.RUNNING);
    const swarmConfig = createMockSwarmConfig();

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={swarmStatus}
                onPause={() => {
                    action("onPause")();
                    setSwarmStatus(ExecutionStates.PAUSED);
                }}
                onResume={() => {
                    action("onResume")();
                    setSwarmStatus(ExecutionStates.RUNNING);
                }}
                onStop={() => {
                    action("onStop")();
                    setSwarmStatus(ExecutionStates.STOPPED);
                }}
            />
        </Box>
    );
}

/**
 * Paused swarm
 */
export function PausedSwarm() {
    const [swarmStatus, setSwarmStatus] = useState(ExecutionStates.PAUSED);
    const swarmConfig = createMockSwarmConfig({
        subtasks: [
            ...createMockSwarmConfig().subtasks!.slice(0, 2),
            {
                id: "T3",
                description: "Task failed due to API error",
                status: "failed",
                assignee_bot_id: "trend_analyst_bot_202",
                priority: "high",
                created_at: new Date().toISOString(),
            },
            ...createMockSwarmConfig().subtasks!.slice(3),
        ],
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={swarmStatus}
                onPause={() => {
                    action("onPause")();
                    setSwarmStatus(ExecutionStates.PAUSED);
                }}
                onResume={() => {
                    action("onResume")();
                    setSwarmStatus(ExecutionStates.RUNNING);
                }}
                onStop={() => {
                    action("onStop")();
                    setSwarmStatus(ExecutionStates.STOPPED);
                }}
            />
        </Box>
    );
}

/**
 * Failed swarm
 */
export function FailedSwarm() {
    const swarmConfig = createMockSwarmConfig({
        subtasks: createMockSwarmConfig().subtasks!.map((task, index) => ({
            ...task,
            status: index < 2 ? "done" : index === 2 ? "failed" : "blocked",
        })) as SwarmSubTask[],
        stats: {
            ...createMockSwarmConfig().stats,
            totalCredits: "9850", // Near limit
        },
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={ExecutionStates.FAILED}
            />
        </Box>
    );
}

/**
 * Loading state
 */
export function LoadingSwarm() {
    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={null}
                swarmStatus={ExecutionStates.STARTING}
                isLoading={true}
            />
        </Box>
    );
}

/**
 * No swarm active
 */
export function NoSwarm() {
    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={null}
                swarmStatus={ExecutionStates.UNINITIALIZED}
            />
        </Box>
    );
}

/**
 * Swarm with high credit usage
 */
export function HighCreditUsage() {
    const swarmConfig = createMockSwarmConfig({
        stats: {
            ...createMockSwarmConfig().stats,
            totalCredits: "1234567", // 1.2M credits
            totalToolCalls: 250,
        },
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={ExecutionStates.RUNNING}
                onPause={action("onPause")}
                onResume={action("onResume")}
                onStop={action("onStop")}
            />
        </Box>
    );
}

/**
 * Swarm with pending approvals
 */
export function WithPendingApprovals() {
    const swarmConfig = createMockSwarmConfig({
        pendingToolCalls: [
            {
                pendingId: "pending_001",
                toolCallId: "call_001",
                toolName: "premium_competitive_intelligence",
                toolArguments: JSON.stringify({
                    companies: ["AAPL", "MSFT", "GOOGL"],
                    analysis_depth: "comprehensive",
                }),
                callerBotId: "competitive_analyst_bot_101",
                conversationId: "conv_123",
                requestedAt: Date.now() - 60000,
                status: PendingToolCallStatus.PENDING_APPROVAL,
                approvalTimeoutAt: Date.now() + 240000, // 4 minutes remaining
                executionAttempts: 0,
            },
            {
                pendingId: "pending_002",
                toolCallId: "call_002",
                toolName: "expensive_data_analysis",
                toolArguments: JSON.stringify({
                    dataset: "market_data_q1_2024",
                    algorithms: ["regression", "clustering"],
                }),
                callerBotId: "trend_analyst_bot_202",
                conversationId: "conv_123",
                requestedAt: Date.now() - 30000,
                status: PendingToolCallStatus.PENDING_APPROVAL,
                approvalTimeoutAt: Date.now() + 30000, // 30 seconds remaining (urgent!)
                executionAttempts: 0,
            },
        ],
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={ExecutionStates.RUNNING}
                onPause={action("onPause")}
                onResume={action("onResume")}
                onStop={action("onStop")}
            />
        </Box>
    );
}

/**
 * Complete swarm (all tasks done)
 */
export function CompleteSwarm() {
    const swarmConfig = createMockSwarmConfig({
        subtasks: createMockSwarmConfig().subtasks!.map(task => ({
            ...task,
            status: "done",
        })) as SwarmSubTask[],
        stats: {
            totalToolCalls: 47,
            totalCredits: "8750",
            startedAt: Date.now() - 95 * 60 * 1000,
            lastProcessingCycleEndedAt: Date.now() - 5 * 60 * 1000,
        },
        resources: [
            ...createMockSwarmConfig().resources!,
            {
                id: "final_report_q1_2024",
                kind: "File",
                mime: "application/pdf",
                creator_bot_id: "report_writer_bot_303",
                created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
                meta: { pages: 25, size: 850000 },
            },
        ],
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={ExecutionStates.STOPPED}
            />
        </Box>
    );
}

/**
 * Story with long goal text to test marquee scrolling
 */
export function LongGoalText() {
    const [swarmStatus, setSwarmStatus] = useState(ExecutionStates.RUNNING);
    const swarmConfig = createMockSwarmConfig({
        goal: "Create a comprehensive market analysis report for Q1 2024 including competitor analysis, growth trends, market opportunities, customer sentiment analysis, technological disruptions, regulatory changes, and strategic recommendations for the next fiscal year",
    });

    return (
        <Box sx={{ p: 2, maxWidth: 800, mx: "auto" }}>
            <SwarmPlayer
                swarmConfig={swarmConfig}
                swarmStatus={swarmStatus}
                onPause={() => {
                    action("onPause")();
                    setSwarmStatus(ExecutionStates.PAUSED);
                }}
                onResume={() => {
                    action("onResume")();
                    setSwarmStatus(ExecutionStates.RUNNING);
                }}
                onStop={() => {
                    action("onStop")();
                    setSwarmStatus(ExecutionStates.STOPPED);
                }}
            />
        </Box>
    );
}

/**
 * Story showing SwarmPlayer with SwarmDetailPanel integration
 */
export function WithDetailPanel() {
    const [swarmStatus, setSwarmStatus] = useState(ExecutionStates.RUNNING);
    const { isOpen: isRightDrawerOpen } = useMenu({
        id: ELEMENT_IDS.RightDrawer,
        isMobile: false,
    });
    
    const swarmConfig = createMockSwarmConfig({
        pendingToolCalls: [
            {
                pendingId: "pending_003",
                toolCallId: "call_003",
                toolName: "web_search",
                toolArguments: JSON.stringify({
                    query: "latest tech market trends 2024",
                    num_results: 10,
                }),
                callerBotId: "trend_analyst_bot_202",
                conversationId: "conv_123",
                requestedAt: Date.now() - 120000,
                status: PendingToolCallStatus.PENDING_APPROVAL,
                approvalTimeoutAt: Date.now() + 180000,
                executionAttempts: 0,
            },
        ],
    });

    return (
        <Box sx={{ display: "flex", height: "100vh" }}>
            <Box sx={{ flex: 1, p: 2 }}>
                <Box sx={{ maxWidth: 800, mx: "auto" }}>
                    <SwarmPlayer
                        swarmConfig={swarmConfig}
                        swarmStatus={swarmStatus}
                        onPause={() => {
                            action("onPause")();
                            setSwarmStatus(ExecutionStates.PAUSED);
                        }}
                        onResume={() => {
                            action("onResume")();
                            setSwarmStatus(ExecutionStates.RUNNING);
                        }}
                        onStop={() => {
                            action("onStop")();
                            setSwarmStatus(ExecutionStates.STOPPED);
                        }}
                    />
                </Box>
            </Box>
            
            {/* Simulate right drawer */}
            <Box
                sx={{
                    width: isRightDrawerOpen ? 400 : 0,
                    height: "100%",
                    borderLeft: 1,
                    borderColor: "divider",
                    overflow: "hidden",
                    transition: "width 0.3s ease",
                }}
            >
                {isRightDrawerOpen && (
                    <SwarmDetailPanel
                        swarmConfig={swarmConfig}
                        swarmStatus={swarmStatus}
                        onApproveToolCall={action("onApproveToolCall")}
                        onRejectToolCall={action("onRejectToolCall")}
                    />
                )}
            </Box>
        </Box>
    );
}