import type { Meta, StoryObj } from "@storybook/react";
import { Box, FormControl, FormControlLabel, FormLabel, Radio, RadioGroup, TextField, Typography } from "@mui/material";
import { type ChatConfigObject, ExecutionStates, PendingToolCallStatus, type SwarmSubTask } from "@vrooli/shared";
import { useState } from "react";
import { SwarmPlayer } from "./SwarmPlayer.js";
import { Switch } from "../inputs/Switch/Switch.js";

const meta: Meta<typeof SwarmPlayer> = {
    title: "Components/Chat/SwarmPlayer",
    component: SwarmPlayer,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

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
        swarmLeader: "coordinator_bot_123",
        subtaskLeaders: {
            "T1": "data_collector_bot_789",
            "T2": "competitive_analyst_bot_101",
            "T3": "trend_analyst_bot_202",
            "T4": "report_writer_bot_303",
        },
        teamId: "analytics_team_001",
        stats: {
            totalToolCalls: 23,
            totalCredits: "4750",
            startedAt: Date.now() - 45 * 60 * 1000,
            lastProcessingCycleEndedAt: Date.now() - 2 * 60 * 1000,
        },
        ...overrides,
    };
    return baseConfig;
}

export const SwarmPlayerShowcase: Story = {
    render: () => {
        const [status, setStatus] = useState<string>(ExecutionStates.RUNNING);
        const [isLoading, setIsLoading] = useState(false);
        const [hasSwarm, setHasSwarm] = useState(true);
        const [goalLength, setGoalLength] = useState<"short" | "medium" | "long">("medium");
        const [creditAmount, setCreditAmount] = useState<"low" | "medium" | "high">("medium");
        const [taskProgress, setTaskProgress] = useState<"none" | "partial" | "complete">("partial");
        const [hasPendingApprovals, setHasPendingApprovals] = useState(false);

        // Build swarm config based on controls
        const getGoalText = () => {
            switch (goalLength) {
                case "short":
                    return "Analyze Q1 2024 data";
                case "long":
                    return "Create a comprehensive market analysis report for Q1 2024 including competitor analysis, growth trends, market opportunities, customer sentiment analysis, technological disruptions, regulatory changes, and strategic recommendations for the next fiscal year with detailed financial projections and risk assessments";
                default:
                    return "Create a comprehensive market analysis report for Q1 2024";
            }
        };

        const getCredits = () => {
            switch (creditAmount) {
                case "low":
                    return "7500000"; // $0.08
                case "high":
                    return "123456700000000"; // $1,234.57
                default:
                    return "475000000"; // $4.75
            }
        };

        const getSubtasks = (): SwarmSubTask[] => {
            const baseTasks = createMockSwarmConfig().subtasks || [];
            switch (taskProgress) {
                case "none":
                    return baseTasks.map(task => ({ ...task, status: "todo" }));
                case "complete":
                    return baseTasks.map(task => ({ ...task, status: "done" }));
                default:
                    return baseTasks;
            }
        };

        const swarmConfig = hasSwarm ? createMockSwarmConfig({
            goal: getGoalText(),
            subtasks: getSubtasks(),
            stats: {
                ...createMockSwarmConfig().stats,
                totalCredits: getCredits(),
            },
            pendingToolCalls: hasPendingApprovals ? [
                {
                    pendingId: "pending_001",
                    toolCallId: "call_001",
                    toolName: "premium_analysis",
                    toolArguments: JSON.stringify({ depth: "comprehensive" }),
                    callerBotId: "analyst_bot",
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 60000,
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    approvalTimeoutAt: Date.now() + 240000,
                    executionAttempts: 0,
                },
            ] : [],
        }) : null;

        const statuses = [
            ExecutionStates.UNINITIALIZED,
            ExecutionStates.STARTING,
            ExecutionStates.RUNNING,
            ExecutionStates.IDLE,
            ExecutionStates.PAUSED,
            ExecutionStates.STOPPED,
            ExecutionStates.FAILED,
            ExecutionStates.TERMINATED,
        ];

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default", 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1200, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>SwarmPlayer Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Status Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Swarm Status</FormLabel>
                                <RadioGroup
                                    value={status}
                                    onChange={(e) => setStatus(e.target.value)}
                                    sx={{ gap: 0.5 }}
                                >
                                    {statuses.map(s => (
                                        <FormControlLabel 
                                            key={s} 
                                            value={s} 
                                            control={<Radio size="small" />} 
                                            label={s.charAt(0) + s.slice(1).toLowerCase().replace("_", " ")} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
                            </FormControl>

                            {/* Goal Length Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Goal Text Length</FormLabel>
                                <RadioGroup
                                    value={goalLength}
                                    onChange={(e) => setGoalLength(e.target.value as typeof goalLength)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="short" control={<Radio size="small" />} label="Short" sx={{ m: 0 }} />
                                    <FormControlLabel value="medium" control={<Radio size="small" />} label="Medium" sx={{ m: 0 }} />
                                    <FormControlLabel value="long" control={<Radio size="small" />} label="Long (Scrolling)" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Credit Amount Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Credits Used</FormLabel>
                                <RadioGroup
                                    value={creditAmount}
                                    onChange={(e) => setCreditAmount(e.target.value as typeof creditAmount)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="low" control={<Radio size="small" />} label="Low ($0.08)" sx={{ m: 0 }} />
                                    <FormControlLabel value="medium" control={<Radio size="small" />} label="Medium ($4.75)" sx={{ m: 0 }} />
                                    <FormControlLabel value="high" control={<Radio size="small" />} label="High ($1,234.57)" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Task Progress Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Task Progress</FormLabel>
                                <RadioGroup
                                    value={taskProgress}
                                    onChange={(e) => setTaskProgress(e.target.value as typeof taskProgress)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="0% (All Todo)" sx={{ m: 0 }} />
                                    <FormControlLabel value="partial" control={<Radio size="small" />} label="25% (Partial)" sx={{ m: 0 }} />
                                    <FormControlLabel value="complete" control={<Radio size="small" />} label="100% (Complete)" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Toggle Controls */}
                            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                <Switch
                                    checked={hasSwarm}
                                    onChange={setHasSwarm}
                                    size="sm"
                                    label="Has Swarm"
                                    labelPosition="right"
                                />
                                <Switch
                                    checked={isLoading}
                                    onChange={setIsLoading}
                                    size="sm"
                                    label="Loading State"
                                    labelPosition="right"
                                />
                                <Switch
                                    checked={hasPendingApprovals}
                                    onChange={setHasPendingApprovals}
                                    size="sm"
                                    label="Pending Approvals"
                                    labelPosition="right"
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Player Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>SwarmPlayer Preview</Typography>
                        
                        <Box sx={{ maxWidth: 800, mx: "auto" }}>
                            <SwarmPlayer
                                swarmConfig={swarmConfig}
                                swarmStatus={status}
                                isLoading={isLoading}
                                onStart={() => {
                                    console.log("Start clicked");
                                    setStatus(ExecutionStates.STARTING);
                                    setTimeout(() => setStatus(ExecutionStates.RUNNING), 1000);
                                }}
                                onPause={() => {
                                    console.log("Pause clicked");
                                    setStatus(ExecutionStates.PAUSED);
                                }}
                                onResume={() => {
                                    console.log("Resume clicked");
                                    setStatus(ExecutionStates.RUNNING);
                                }}
                                onStop={() => {
                                    console.log("Stop clicked");
                                    setStatus(ExecutionStates.STOPPED);
                                }}
                            />
                        </Box>

                        {/* Status Information */}
                        <Box sx={{ mt: 4 }}>
                            <Typography variant="h6" sx={{ mb: 2 }}>Current State</Typography>
                            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 2 }}>
                                <Typography variant="body2">
                                    <strong>Status:</strong> {status}
                                </Typography>
                                <Typography variant="body2">
                                    <strong>Progress:</strong> {
                                        taskProgress === "none" ? "0%" : 
                                        taskProgress === "complete" ? "100%" : 
                                        "25%"
                                    }
                                </Typography>
                                <Typography variant="body2">
                                    <strong>Credits:</strong> {(() => {
                                        if (!swarmConfig?.stats.totalCredits) return "$0.00";
                                        try {
                                            const credits = BigInt(swarmConfig.stats.totalCredits);
                                            const usd = Number(credits / BigInt(1e8)) / 100;
                                            return `$${usd.toFixed(2)}`;
                                        } catch {
                                            return "$0.00";
                                        }
                                    })()}
                                </Typography>
                                <Typography variant="body2">
                                    <strong>Goal Length:</strong> {getGoalText().length} chars
                                </Typography>
                            </Box>
                        </Box>

                        {/* Action Buttons Info */}
                        <Box sx={{ mt: 2 }}>
                            <Typography variant="body2" color="text.secondary">
                                Click anywhere on the player to open the detail panel. 
                                Action buttons (pause/resume/stop) are shown based on the current status.
                                {goalLength === "long" && " Long text will scroll with a marquee effect."}
                            </Typography>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
