import type { Meta, StoryObj } from "@storybook/react";
import { Box, FormControl, FormControlLabel, FormLabel, Radio, RadioGroup, TextField, Typography } from "@mui/material";
import { type ChatConfigObject, ExecutionStates, PendingToolCallStatus, type SwarmSubTask, generatePK } from "@vrooli/shared";
import React, { useState } from "react";
import { SwarmDetailPanel } from "./SwarmDetailPanel.js";
import { Switch } from "../inputs/Switch/Switch.js";
import { http, HttpResponse } from "msw";
import { API_URL, signedInUserId } from "../../__test/storybookConsts.js";
import { getMockApiUrl } from "../../__test/helpers/storybookMocking.js";

const meta: Meta<typeof SwarmDetailPanel> = {
    title: "Components/Chat/SwarmDetailPanel",
    component: SwarmDetailPanel,
    parameters: {
        layout: "fullscreen",
        msw: {
            handlers: [
                // Mock single team endpoint with MOISE+ structure - try multiple patterns
                http.get(getMockApiUrl("/team/:publicId"), ({ params, request }) => {
                    console.log("MSW - Team endpoint hit with URL:", request.url);
                    console.log("MSW - Single team endpoint hit!");
                    const teamId = params.publicId as string;
                    
                    console.log("Team MSW handler - request:", { 
                        teamId,
                        mockTeamId: mockIds.team,
                    });
                    
                    if (teamId === mockIds.team) {
                        console.log("MSW - Returning team with MOISE+ structure");
                        const teamData = {
                            __typename: "Team" as const,
                            id: mockIds.team,
                            name: "Analytics Swarm Team",
                            handle: "analytics_team",
                            config: JSON.stringify({
                                __version: "1.0",
                                structure: {
                                    type: "MOISE+",
                                    version: "1.0",
                                    content: `structure AnalyticsSwarm {
  group researchTeam {
    role coordinator cardinality 1..1
    role dataCollector cardinality 1..3
    role analyst cardinality 2..5
    role reporter cardinality 1..2
    
    link coordinator > dataCollector
    link coordinator > analyst
    link coordinator > reporter
    link analyst > dataCollector communication
  }
  
  group qualityControl {
    role reviewer cardinality 1..2
    role validator cardinality 1..1
    
    link reviewer > validator authority
  }
  
  link researchTeam.coordinator > qualityControl.reviewer delegation
}`,
                                },
                            }),
                            translations: [
                                {
                                    __typename: "TeamTranslation" as const,
                                    id: generatePK().toString(),
                                    language: "en",
                                    name: "Analytics Swarm Team",
                                    bio: "A specialized team for market analysis and reporting",
                                },
                            ],
                        };
                        
                        console.log("MSW - Team response:", { data: teamData });
                        return HttpResponse.json({ data: teamData });
                    }
                    
                    console.log("MSW - No matching team found, returning 404");
                    return new HttpResponse(null, { status: 404 });
                }),
                // Try plural form too
                http.get(getMockApiUrl("/teams/:publicId"), ({ params, request }) => {
                    console.log("MSW - Teams (plural) endpoint hit with URL:", request.url);
                    const teamId = params.publicId as string;
                    
                    if (teamId === mockIds.team) {
                        console.log("MSW - Returning team data from plural endpoint");
                        const teamData = {
                            __typename: "Team" as const,
                            id: mockIds.team,
                            name: "Analytics Swarm Team",
                            handle: "analytics_team",
                            config: JSON.stringify({
                                __version: "1.0",
                                structure: {
                                    type: "MOISE+",
                                    version: "1.0",
                                    content: `structure AnalyticsSwarm {
  group researchTeam {
    role coordinator cardinality 1..1
    role dataCollector cardinality 1..3
    role analyst cardinality 2..5
    role reporter cardinality 1..2
    
    link coordinator > dataCollector
    link coordinator > analyst
    link coordinator > reporter
    link analyst > dataCollector communication
  }
  
  group qualityControl {
    role reviewer cardinality 1..2
    role validator cardinality 1..1
    
    link reviewer > validator authority
  }
  
  link researchTeam.coordinator > qualityControl.reviewer delegation
}`,
                                },
                            }),
                            translations: [
                                {
                                    __typename: "TeamTranslation" as const,
                                    id: generatePK().toString(),
                                    language: "en",
                                    name: "Analytics Swarm Team",
                                    bio: "A specialized team for market analysis and reporting",
                                },
                            ],
                        };
                        
                        return HttpResponse.json({ data: teamData });
                    }
                    
                    return new HttpResponse(null, { status: 404 });
                }),
                // Mock users endpoint for bots
                http.get(getMockApiUrl("/users"), ({ request }) => {
                    console.log("MSW - Users endpoint hit!");
                    const url = new URL(request.url);
                    
                    // Parse the 'where' parameter which contains the query filters
                    const whereParam = url.searchParams.get("where");
                    let botIds: string[] = [];
                    let isBot = false;
                    
                    if (whereParam) {
                        try {
                            const where = JSON.parse(whereParam);
                            if (where.id?.in && Array.isArray(where.id.in)) {
                                botIds = where.id.in;
                            }
                            isBot = where.isBot === true;
                        } catch (e) {
                            console.error("Failed to parse where param:", e);
                        }
                    }
                    
                    // Also check direct params as fallback
                    if (botIds.length === 0) {
                        const idParam = url.searchParams.get("id");
                        if (idParam) {
                            try {
                                const idObj = JSON.parse(idParam);
                                if (idObj.in && Array.isArray(idObj.in)) {
                                    botIds = idObj.in;
                                }
                            } catch (e) {
                                // If it's not JSON, treat it as a single ID
                                botIds = [idParam];
                            }
                        }
                    }
                    
                    if (!isBot) {
                        const isBotParam = url.searchParams.get("isBot");
                        isBot = isBotParam === "true";
                    }
                    
                    console.log("Users MSW handler - request:", { 
                        url: request.url, 
                        searchParams: Object.fromEntries(url.searchParams),
                        whereParam,
                        botIds,
                        isBot,
                        mockBotIds: Object.keys(mockIds),
                    });
                    
                    if (isBot && botIds.length > 0) {
                        console.log("MSW - Returning bot users");
                        // botIds already parsed above
                        const mockBots = {
                            [mockIds.swarmLeader]: { name: "Strategic Coordinator Bot", bio: "Manages overall swarm coordination" },
                            [mockIds.dataCollectorBot]: { name: "Data Harvester Bot", bio: "Collects and aggregates market data" },
                            [mockIds.competitiveAnalystBot]: { name: "Competitive Intelligence Bot", bio: "Analyzes competitor strategies" },
                            [mockIds.trendAnalystBot]: { name: "Trend Analyzer Bot", bio: "Identifies market trends and patterns" },
                            [mockIds.reportWriterBot]: { name: "Report Generator Bot", bio: "Creates comprehensive reports" },
                            [mockIds.reportGeneratorBot]: { name: "Report Generator", bio: "Generates various report formats" },
                            [mockIds.notificationBot]: { name: "Notification Bot", bio: "Handles team notifications" },
                        };
                        
                        const edges = botIds.map((id: string) => {
                            const botData = {
                                cursor: id,
                                node: {
                                    __typename: "User" as const,
                                    id,
                                    name: mockBots[id]?.name || `Bot ${id}`,
                                    handle: (mockBots[id]?.name || id).toLowerCase().replace(/\s+/g, "").replace(/_/g, ""),
                                    isBot: true,
                                    isBotDepictingPerson: false,
                                    translations: [
                                        {
                                            __typename: "UserTranslation" as const,
                                            id: generatePK().toString(),
                                            language: "en",
                                            bio: mockBots[id]?.bio || "An AI assistant bot",
                                        },
                                    ],
                                },
                            };
                            console.log("MSW - Creating bot edge for ID:", id, botData);
                            return botData;
                        });
                        
                        const response = {
                            edges,
                            pageInfo: {
                                __typename: "PageInfo" as const,
                                endCursor: null,
                                hasNextPage: false,
                            },
                        };
                        
                        console.log("MSW - Returning users response:", response);
                        return HttpResponse.json(response);
                    }
                    
                    return HttpResponse.json({ edges: [], pageInfo: { endCursor: null, hasNextPage: false } });
                }),
                // Mock notes endpoint for resources
                http.get(getMockApiUrl("/notes/versions"), () => {
                    return HttpResponse.json({ data: { edges: [], pageInfo: { endCursor: null, hasNextPage: false } } });
                }),
                // Catch-all handler to debug any other requests
                http.get(`${API_URL}/*`, ({ request }) => {
                    console.log("MSW - Catch-all handler hit with URL:", request.url);
                    // Check if this is a team request
                    if (request.url.includes("/team/") || request.url.includes("/teams/")) {
                        console.log("MSW - This is a team request!", request.url);
                        const urlParts = request.url.split("/");
                        const teamId = urlParts[urlParts.length - 1];
                        console.log("MSW - Extracted team ID:", teamId);
                        
                        if (teamId === mockIds.team) {
                            console.log("MSW - Returning team data from catch-all handler");
                            const teamData = {
                                __typename: "Team" as const,
                                id: mockIds.team,
                                name: "Analytics Swarm Team",
                                handle: "analytics_team",
                                config: JSON.stringify({
                                    __version: "1.0",
                                    structure: {
                                        type: "MOISE+",
                                        version: "1.0",
                                        content: `structure AnalyticsSwarm {
  group researchTeam {
    role coordinator cardinality 1..1
    role dataCollector cardinality 1..3
    role analyst cardinality 2..5
    role reporter cardinality 1..2
    
    link coordinator > dataCollector
    link coordinator > analyst
    link coordinator > reporter
    link analyst > dataCollector communication
  }
  
  group qualityControl {
    role reviewer cardinality 1..2
    role validator cardinality 1..1
    
    link reviewer > validator authority
  }
  
  link researchTeam.coordinator > qualityControl.reviewer delegation
}`,
                                    },
                                }),
                                translations: [
                                    {
                                        __typename: "TeamTranslation" as const,
                                        id: generatePK().toString(),
                                        language: "en",
                                        name: "Analytics Swarm Team",
                                        bio: "A specialized team for market analysis and reporting",
                                    },
                                ],
                            };
                            
                            return HttpResponse.json({ data: teamData });
                        }
                    }
                    return new HttpResponse(null, { status: 404 });
                }),
            ],
        },
    },
};

// Generate stable IDs for mock data (outside of meta to be available in handlers)
// Using a seed-based approach to ensure consistent IDs across re-renders
const createMockId = (seed: string) => {
    // Create a simple hash from the seed string
    let hash = 0;
    for (let i = 0; i < seed.length; i++) {
        const char = seed.charCodeAt(i);
        hash = ((hash << 5) - hash) + char;
        hash = hash & hash; // Convert to 32-bit integer
    }
    // Use the hash to create a consistent but valid-looking Snowflake ID
    return Math.abs(hash).toString().padEnd(18, "0").slice(0, 18);
};

const mockIds = {
    team: createMockId("analytics_team_001"),
    swarmLeader: createMockId("coordinator_bot_123"),
    dataCollectorBot: createMockId("data_collector_bot_789"),
    competitiveAnalystBot: createMockId("competitive_analyst_bot_101"),
    trendAnalystBot: createMockId("trend_analyst_bot_202"),
    reportWriterBot: createMockId("report_writer_bot_303"),
    reportGeneratorBot: createMockId("ReportGenerator"),
    notificationBot: createMockId("NotificationBot"),
};

console.log("SwarmDetailPanel.stories - mockIds:", mockIds);

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
                assignee_bot_id: mockIds.dataCollectorBot,
                priority: "high",
                created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
            },
            {
                id: "T2",
                description: "Analyze competitor positioning and pricing strategies",
                status: "in_progress",
                depends_on: ["T1"],
                assignee_bot_id: mockIds.competitiveAnalystBot,
                priority: "high",
                created_at: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
            },
            {
                id: "T3",
                description: "Identify growth trends and market opportunities",
                status: "todo",
                depends_on: ["T1"],
                assignee_bot_id: mockIds.trendAnalystBot,
                priority: "medium",
                created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(),
            },
            {
                id: "T4",
                description: "Generate executive summary with actionable insights",
                status: "todo",
                depends_on: ["T2", "T3"],
                assignee_bot_id: mockIds.reportWriterBot,
                priority: "medium",
                created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
            },
        ],
        swarmLeader: mockIds.swarmLeader,
        subtaskLeaders: {
            "T1": mockIds.dataCollectorBot,
            "T2": mockIds.competitiveAnalystBot,
            "T3": mockIds.trendAnalystBot,
            "T4": mockIds.reportWriterBot,
        },
        teamId: mockIds.team,
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

// MOISE+ organization structure is now visible in the Agents tab when viewing swarms
// with teamId "analytics_team_001". The MSW handlers above provide the mock data.

export const SwarmDetailPanelShowcase: Story = {
    render: () => {
        const [status, setStatus] = useState<string>(ExecutionStates.RUNNING);
        const [isLoading, setIsLoading] = useState(false);
        const [hasSwarm, setHasSwarm] = useState(true);
        const [goalLength, setGoalLength] = useState<"short" | "medium" | "long">("medium");
        const [creditAmount, setCreditAmount] = useState<"low" | "medium" | "high">("medium");
        const [taskProgress, setTaskProgress] = useState<"none" | "partial" | "complete">("partial");
        const [hasPendingApprovals, setHasPendingApprovals] = useState(true);
        const [hasResources, setHasResources] = useState(true);
        const [hasRecords, setHasRecords] = useState(true);

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
            resources: hasResources ? [
                {
                    id: "resource-1",
                    kind: "File",
                    mime: "text/csv",
                    creator_bot_id: mockIds.dataCollectorBot,
                    created_at: new Date(Date.now() - 3000000).toISOString(),
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
                    creator_bot_id: mockIds.competitiveAnalystBot,
                    created_at: new Date(Date.now() - 2500000).toISOString(),
                    meta: {
                        endpoint: "https://api.example.com/trends",
                        last_updated: "2024-01-15",
                    },
                },
                {
                    id: "resource-3",
                    kind: "Image",
                    mime: "image/png",
                    creator_bot_id: mockIds.reportWriterBot,
                    created_at: new Date(Date.now() - 600000).toISOString(),
                    meta: {
                        width: "1920",
                        height: "1080",
                        type: "chart",
                    },
                },
            ] : [],
            pendingToolCalls: hasPendingApprovals ? [
                // Required approval - will wait indefinitely
                {
                    pendingId: "pending_001",
                    toolCallId: "call_001",
                    toolName: "DeleteFile",
                    toolArguments: JSON.stringify({
                        path: "/data/customer_database.csv",
                        reason: "Cleanup old data files",
                    }),
                    callerBotId: mockIds.dataCollectorBot,
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 120000, // 2 minutes ago
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    approvalTimeoutAt: null, // No timeout - requires manual approval
                    executionAttempts: 0,
                },
                // Auto-accept after timeout
                {
                    pendingId: "pending_002",
                    toolCallId: "call_002",
                    toolName: "WriteFile",
                    toolArguments: JSON.stringify({
                        path: "/reports/analysis_summary.txt",
                        content: "Q1 2024 Market Analysis Summary\n\nKey Findings:\n- Market growth: 15%\n- Top competitors: A, B, C\n- Opportunities: Mobile, AI integration",
                    }),
                    callerBotId: mockIds.reportWriterBot,
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 30000, // 30 seconds ago
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    approvalTimeoutAt: Date.now() + 60000, // Auto-approve in 1 minute
                    executionAttempts: 0,
                },
                // Urgent - auto-accept soon
                {
                    pendingId: "pending_003",
                    toolCallId: "call_003",
                    toolName: "SendEmail",
                    toolArguments: JSON.stringify({
                        to: ["team@example.com"],
                        subject: "Analysis Report Ready",
                        body: "The Q1 2024 market analysis report has been completed and is ready for review.\n\nView the full report at: https://example.com/reports/q1-2024",
                    }),
                    callerBotId: mockIds.notificationBot,
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 45000,
                    status: PendingToolCallStatus.PENDING_APPROVAL,
                    approvalTimeoutAt: Date.now() + 15000, // Urgent! Only 15 seconds left
                    executionAttempts: 0,
                },
                // Already approved
                {
                    pendingId: "pending_004",
                    toolCallId: "call_004",
                    toolName: "HttpRequest",
                    toolArguments: JSON.stringify({
                        url: "https://api.marketdata.com/v1/trends",
                        method: "GET",
                        headers: {
                            "Authorization": "Bearer [REDACTED]",
                        },
                    }),
                    callerBotId: mockIds.dataCollectorBot,
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 300000, // 5 minutes ago
                    status: PendingToolCallStatus.APPROVED,
                    approvalTimeoutAt: Date.now() - 240000, // Approved 4 minutes ago
                    approvedAt: Date.now() - 240000,
                    approvedBy: signedInUserId,
                    executionAttempts: 1,
                },
                // Already executed after approval
                {
                    pendingId: "pending_005",
                    toolCallId: "call_005",
                    toolName: "CreateChart",
                    toolArguments: JSON.stringify({
                        type: "line",
                        title: "Market Growth Trends Q1 2024",
                        data: {
                            labels: ["Jan", "Feb", "Mar"],
                            datasets: [{
                                label: "Revenue",
                                data: [1200000, 1350000, 1500000],
                            }],
                        },
                    }),
                    callerBotId: mockIds.trendAnalystBot,
                    conversationId: "conv_123",
                    requestedAt: Date.now() - 600000, // 10 minutes ago
                    status: PendingToolCallStatus.EXECUTED,
                    approvalTimeoutAt: Date.now() - 540000,
                    approvedAt: Date.now() - 540000,
                    approvedBy: signedInUserId,
                    executedAt: Date.now() - 535000,
                    executionAttempts: 1,
                    result: JSON.stringify({
                        success: true,
                        chartUrl: "/charts/market-growth-q1-2024.png",
                    }),
                },
            ] : [],
            records: hasRecords ? [
                {
                    id: "record-1",
                    routine_name: "DataCollection",
                    caller_bot_id: mockIds.dataCollectorBot,
                    created_at: new Date(Date.now() - 3600000).toISOString(),
                    output_resource_ids: ["resource-1"],
                },
                {
                    id: "record-2",
                    routine_name: "DataAnalysis",
                    caller_bot_id: mockIds.competitiveAnalystBot,
                    created_at: new Date(Date.now() - 1800000).toISOString(),
                    output_resource_ids: [],
                },
                {
                    id: "record-3",
                    routine_name: "ChartGeneration",
                    caller_bot_id: mockIds.reportWriterBot,
                    created_at: new Date(Date.now() - 600000).toISOString(),
                    output_resource_ids: ["resource-3"],
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
                        <Typography variant="h5" sx={{ mb: 3 }}>SwarmDetailPanel Controls</Typography>
                        
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                            Note: MOISE+ organization structure is visible in the Agents tab. The Analytics Swarm Team 
                            includes a comprehensive MOISE+ model with research and quality control groups.
                        </Typography>
                        
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
                                <Switch
                                    checked={hasResources}
                                    onChange={setHasResources}
                                    size="sm"
                                    label="Has Resources"
                                    labelPosition="right"
                                />
                                <Switch
                                    checked={hasRecords}
                                    onChange={setHasRecords}
                                    size="sm"
                                    label="Has Records"
                                    labelPosition="right"
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* SwarmDetailPanel Display */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>SwarmDetailPanel Preview</Typography>
                        
                        <Box sx={{ maxWidth: 800, mx: "auto" }}>
                            <SwarmDetailPanel
                                swarmConfig={swarmConfig}
                                swarmStatus={status}
                                isLoading={isLoading}
                                chatId="chat_123"
                                onApproveToolCall={(pendingId) => {
                                    console.log("Approve tool:", pendingId);
                                }}
                                onRejectToolCall={(pendingId, reason) => {
                                    console.log("Reject tool:", pendingId, reason);
                                }}
                                onStart={() => {
                                    console.log("Start swarm");
                                    setStatus(ExecutionStates.STARTING);
                                    setTimeout(() => setStatus(ExecutionStates.RUNNING), 1000);
                                }}
                                onPause={() => {
                                    console.log("Pause swarm");
                                    setStatus(ExecutionStates.PAUSED);
                                }}
                                onResume={() => {
                                    console.log("Resume swarm");
                                    setStatus(ExecutionStates.RUNNING);
                                }}
                                onStop={() => {
                                    console.log("Stop swarm");
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

                        {/* Panel Info */}
                        <Box sx={{ mt: 2 }}>
                            <Typography variant="body2" color="text.secondary">
                                This panel shows detailed information about the swarm execution including tasks, resources, and pending approvals.
                                Use the controls above to simulate different states and configurations.
                            </Typography>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
