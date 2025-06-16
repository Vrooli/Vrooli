import { Box, FormControl, FormLabel, Select, MenuItem, Typography } from "@mui/material";
import { generatePK, type ResourceVersion, type Run, type RunStatus, ResourceSubType } from "@vrooli/shared";
import { useState, Component, ReactNode } from "react";
import { signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { RoutineExecutor } from "./RoutineExecutor.js";
import { PageContainer } from "../Page/Page.js";
import { Switch } from "../inputs/Switch/Switch.js";
import { Slider } from "../inputs/Slider.js";
import { Radio } from "../inputs/Radio.js";
import { FormGroup } from "../inputs/FormGroup.js";

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
    return {
        id: generatePK().toString(),
        name,
        description: "A comprehensive routine for processing data files",
        versionLabel: "1.0.2",
        isComplete: true,
        isPrivate: false,
        resourceSubType: subType,
        config: config,
        translations: [{
            id: generatePK().toString(),
            language: "en",
            name,
            description: "A comprehensive routine for processing data files",
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
            gap: 3, 
            flexDirection: { xs: "column", lg: "row" },
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
                height: "fit-content",
                minWidth: { lg: 350 },
                position: "sticky",
                top: 0
            }}>
                <Typography variant="h5" sx={{ mb: 3 }}>RoutineExecutor Controls</Typography>
                
                <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                    {/* Routine Type */}
                    <FormControl size="small" fullWidth>
                        <FormLabel sx={{ fontSize: "0.875rem", mb: 1 }}>Routine Type</FormLabel>
                        <Select
                            value={routineType}
                            onChange={(e) => setRoutineType(e.target.value as ResourceSubType)}
                        >
                            <MenuItem value={ResourceSubType.RoutineMultiStep}>Multi-Step Workflow</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineApi}>API Call</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineGenerate}>AI Generation</MenuItem>
                            <MenuItem value={ResourceSubType.RoutineCode}>Code Execution</MenuItem>
                        </Select>
                    </FormControl>
                    
                    {/* Run Status */}
                    <Box>
                        <Typography variant="body2" sx={{ fontSize: "0.875rem", mb: 1, fontWeight: 500 }}>
                            Run Status
                        </Typography>
                        <FormGroup>
                            {["InProgress", "Completed", "Failed"].map((status) => (
                                <Box key={status} sx={{ display: "flex", alignItems: "center", gap: 1, mb: 0.5 }}>
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
                        </FormGroup>
                    </Box>
                    
                    {/* Display Options */}
                    <Box>
                        <Typography variant="body2" sx={{ fontSize: "0.875rem", mb: 1, fontWeight: 500 }}>
                            Display Options
                        </Typography>
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5 }}>
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
            </Box>
            
            {/* Preview Section */}
            <Box sx={{ 
                p: 3, 
                bgcolor: "background.paper", 
                borderRadius: 2, 
                boxShadow: 1,
                flex: 1,
                display: "flex",
                flexDirection: "column",
                mb: 8
            }}>
                <Typography variant="h5" sx={{ mb: 3, color: "primary.main" }}>
                    RoutineExecutor Preview
                </Typography>
                
                <Box sx={{ 
                    bgcolor: "background.default", 
                    borderRadius: 2, 
                    p: 2,
                    flex: 1,
                    display: "flex",
                    flexDirection: "column"
                }}>
                    <Box sx={{ 
                        maxWidth: chatMode ? 600 : 800,
                        width: "100%",
                        mx: "auto",
                        border: 1,
                        borderColor: "divider",
                        borderRadius: 1,
                        bgcolor: "background.paper",
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
        </Box>
    );
}

Showcase.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * Basic Example: Simple RoutineExecutor in default state
 */
export function Basic() {
    const runId = generatePK().toString();
    const resourceVersion = createMockResourceVersion();
    
    return (
        <Box sx={{ maxWidth: 800, mx: "auto" }}>
            <Box sx={{ 
                border: 1,
                borderColor: "divider",
                borderRadius: 1,
                overflow: "hidden"
            }}>
                <RoutineExecutor
                    runId={runId}
                    resourceVersion={resourceVersion}
                    onToggleCollapse={action("onToggleCollapse")}
                    onClose={action("onClose")}
                />
            </Box>
        </Box>
    );
}

Basic.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * Chat Mode: RoutineExecutor optimized for chat display
 */
export function ChatMode() {
    const runId = generatePK().toString();
    const resourceVersion = createMockResourceVersion("API Integration");
    
    return (
        <Box sx={{ maxWidth: 600, mx: "auto" }}>
            <Box sx={{ 
                border: 1,
                borderColor: "divider",
                borderRadius: 1,
                overflow: "hidden"
            }}>
                <RoutineExecutor
                    runId={runId}
                    resourceVersion={resourceVersion}
                    onToggleCollapse={action("onToggleCollapse")}
                    onRemove={action("onRemove")}
                    className="chat-routine-executor"
                    chatMode={true}
                />
            </Box>
        </Box>
    );
}

ChatMode.parameters = {
    session: signedInPremiumWithCreditsSession,
};