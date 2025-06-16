import { Box, Collapse, FormControl, FormLabel, IconButton, MenuItem, Select, Typography } from "@mui/material";
import { IconCommon } from "../../icons/Icons.js";
import { generatePK, ResourceSubType, type ResourceVersion, type RunStatus } from "@vrooli/shared";
import { Component, ReactNode, useState } from "react";
import { signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../Page/Page.js";
import { FormGroup } from "../inputs/FormGroup.js";
import { Radio } from "../inputs/Radio.js";
import { Switch } from "../inputs/Switch/Switch.js";
import { RoutineExecutor } from "./RoutineExecutor.js";

// Simple error boundary for debugging
class ErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean; error?: Error }> {
    constructor(props: { children: ReactNode }) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError(error: Error) {
        return { hasError: true, error };
    }

    render() {
        if (this.state.hasError) {
            return (
                <Box sx={{ p: 2, bgcolor: "error.light", color: "error.contrastText" }}>
                    <strong>Component Error:</strong> {this.state.error?.message}
                </Box>
            );
        }
        return this.props.children;
    }
}

// Simple action replacement
const action = (name: string) => (...args: any[]) => console.log(`Action: ${name}`, args);

/**
 * Storybook configuration for RoutineExecutor
 */
export default {
    title: "Components/Chat/RoutineExecutor",
    component: RoutineExecutor,
    decorators: [
        (Story) => (
            <PageContainer size="fullSize">
                <Box sx={{
                    p: 2,
                    height: "100%",
                    overflow: "auto",
                    paddingBottom: "120px" // Generous padding for BottomNav
                }}>
                    <Story />
                </Box>
            </PageContainer>
        ),
    ],
};

// Mock ResourceVersion for testing
function createMockResourceVersion(
    name: string = "Data Processing Routine",
    subType: ResourceSubType = ResourceSubType.RoutineMultiStep,
    config?: any
): ResourceVersion {
    // Generate appropriate descriptions and configs based on routine type
    const routineTypeConfigs = {
        [ResourceSubType.RoutineMultiStep]: {
            description: "A comprehensive multi-step workflow for complex processes",
            defaultName: "Multi-Step Workflow",
            config: config || {
                graph: {
                    nodes: [
                        { id: "step-1", data: { routine: { name: "Initialize Process" } } },
                        { id: "step-2", data: { routine: { name: "Process Data" } } },
                        { id: "step-3", data: { routine: { name: "Generate Report" } } },
                        { id: "step-4", data: { routine: { name: "Send Notification" } } }
                    ]
                }
            }
        },
        [ResourceSubType.RoutineInternalAction]: {
            description: "Internal system action for managing Vrooli resources",
            defaultName: "Internal Action",
            config: config || { callDataAction: { toolName: "ResourceManage" } }
        },
        [ResourceSubType.RoutineApi]: {
            description: "REST API call routine for external service integration",
            defaultName: "API Call",
            config: config || { callDataApi: { endpoint: "https://api.example.com/data", method: "POST" } }
        },
        [ResourceSubType.RoutineCode]: {
            description: "Sandboxed code execution routine for data transformation",
            defaultName: "Code Execution",
            config: config || { callDataCode: { inputTemplate: {}, outputMappings: [] } }
        },
        [ResourceSubType.RoutineData]: {
            description: "Data processing and transformation routine",
            defaultName: "Data Processing",
            config: config || { formInput: { elements: [] }, formOutput: { elements: [] } }
        },
        [ResourceSubType.RoutineGenerate]: {
            description: "AI-powered content generation routine",
            defaultName: "AI Generation",
            config: config || { callDataGenerate: { prompt: "Generate content based on input", maxTokens: 1000 } }
        },
        [ResourceSubType.RoutineInformational]: {
            description: "Information retrieval and display routine",
            defaultName: "Information Retrieval",
            config: config || { formInput: { elements: [] }, formOutput: { elements: [] } }
        },
        [ResourceSubType.RoutineSmartContract]: {
            description: "Blockchain smart contract interaction routine",
            defaultName: "Smart Contract",
            config: config || { callDataSmartContract: { contractAddress: "0x...", chain: "ethereum", methodName: "transfer" } }
        },
        [ResourceSubType.RoutineWeb]: {
            description: "Web search and content retrieval routine",
            defaultName: "Web Search",
            config: config || { callDataWeb: { queryTemplate: "Search for {{input.query}}", outputMapping: {} } }
        }
    };

    const typeConfig = routineTypeConfigs[subType] || routineTypeConfigs[ResourceSubType.RoutineMultiStep];
    const displayName = name === "Data Processing Routine" ? typeConfig.defaultName : name;
    
    return {
        id: generatePK().toString(),
        name: displayName,
        description: typeConfig.description,
        versionLabel: "1.0.2",
        isComplete: true,
        isPrivate: false,
        resourceSubType: subType,
        config: typeConfig.config,
        translations: [{
            id: generatePK().toString(),
            language: "en",
            name: displayName,
            description: typeConfig.description,
        }],
        __typename: "ResourceVersion",
    } as ResourceVersion;
}

/**
 * Showcase Story: Interactive playground for testing RoutineExecutor states
 */
export function Showcase() {
    // State controls
    const [runStatus, setRunStatus] = useState<RunStatus>("InProgress");
    const [isCollapsed, setIsCollapsed] = useState(false);
    const [chatMode, setChatMode] = useState(true);
    const [routineName, setRoutineName] = useState("Data Processing Routine");
    const [routineType, setRoutineType] = useState<ResourceSubType>(ResourceSubType.RoutineMultiStep);
    const [progressPercent, setProgressPercent] = useState(45);
    const [stepCount, setStepCount] = useState(8);
    const [currentStep, setCurrentStep] = useState(4);
    const [controlsOpen, setControlsOpen] = useState(true);

    // Mock data generation
    const runId = generatePK().toString();
    const resourceVersion = createMockResourceVersion(routineName, routineType);

    const handleToggleCollapse = () => {
        setIsCollapsed(!isCollapsed);
        action("handleToggleCollapse")();
    };

    const handleRemove = () => {
        action("handleRemove")();
    };

    const handleClose = () => {
        action("handleClose")();
    };

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            gap: 3,
            maxWidth: 1400,
            mx: "auto",
            minHeight: "100%"
        }}>
            {/* Controls Section */}
            <Box sx={{
                p: 3,
                bgcolor: "background.paper",
                borderRadius: 2,
                boxShadow: 1,
                height: "fit-content"
            }}>
                <Box sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 2 }}>
                    <Typography variant="h5">Controls</Typography>
                    <IconButton
                        onClick={() => setControlsOpen(!controlsOpen)}
                        sx={{
                            transform: controlsOpen ? "rotate(180deg)" : "rotate(0deg)",
                            transition: "transform 0.2s"
                        }}
                    >
                        <IconCommon name="ExpandMore" />
                    </IconButton>
                </Box>

                <Collapse in={controlsOpen}>
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    {/* Routine Type */}
                    <FormControl size="small" fullWidth>
                        <FormLabel sx={{ fontSize: "0.875rem", mb: 1 }}>Routine Type</FormLabel>
                        <Select
                            value={routineType}
                            onChange={(e) => setRoutineType(e.target.value as ResourceSubType)}
                        >
                            <MenuItem value={ResourceSubType.RoutineMultiStep}>Multi-Step Workflow</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineInternalAction}>Internal Action</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineApi}>API Call</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineCode}>Code Execution</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineData}>Data Processing</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineGenerate}>AI Generation</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineInformational}>Information Retrieval</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineSmartContract}>Smart Contract</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineWeb}>Web Search</MenuItem>
                        </Select>
                    </FormControl>

                    {/* Run Status */}
                    <Box>
                        <Typography variant="body2" sx={{ fontSize: "0.875rem", mb: 1, fontWeight: 500 }}>
                            Run Status
                        </Typography>
                        <FormGroup>
                            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
                                {["InProgress", "Completed", "Failed"].map((status) => (
                                    <Box key={status} sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                        <Radio
                                            checked={runStatus === status}
                                            onChange={() => setRunStatus(status as RunStatus)}
                                            size="sm"
                                            variant="primary"
                                        />
                                        <Typography variant="body2" sx={{ fontSize: "0.875rem" }}>
                                            {status === "InProgress" ? "In Progress" : status}
                                        </Typography>
                                    </Box>
                                ))}
                            </Box>
                        </FormGroup>
                    </Box>

                    {/* Display Options */}
                    <Box>
                        <Typography variant="body2" sx={{ fontSize: "0.875rem", mb: 1, fontWeight: 500 }}>
                            Display Options
                        </Typography>
                        <Box sx={{ display: "flex", gap: 3, flexWrap: "wrap" }}>
                            <Switch
                                checked={isCollapsed}
                                onChange={setIsCollapsed}
                                size="sm"
                                variant="default"
                                label="Collapsed State"
                                labelPosition="right"
                            />

                            <Switch
                                checked={chatMode}
                                onChange={setChatMode}
                                size="sm"
                                variant="default"
                                label="Chat Mode"
                                labelPosition="right"
                            />
                        </Box>
                    </Box>
                    </Box>
                </Collapse>
            </Box>

            {/* Preview Section */}
            <Box sx={{
                flex: 1,
                display: "flex",
                flexDirection: "column",
                mb: 8
            }}>
                <Box sx={{
                    maxWidth: chatMode ? 600 : 800,
                    width: "100%",
                    mx: "auto",
                    display: "flex",
                    flexDirection: "column"
                }}>
                        <ErrorBoundary>
                            <RoutineExecutor
                                runId={runId}
                                resourceVersion={resourceVersion}
                                runStatus={runStatus}
                                isCollapsed={isCollapsed}
                                onToggleCollapse={handleToggleCollapse}
                                onRemove={chatMode ? handleRemove : undefined}
                                onClose={!chatMode ? handleClose : undefined}
                                className={chatMode ? "chat-routine-executor" : "standalone-routine-executor"}
                                chatMode={chatMode}
                            />
                    </ErrorBoundary>
                </Box>
            </Box>
        </Box>
    );
}

Showcase.parameters = {
    session: signedInPremiumWithCreditsSession,
};
