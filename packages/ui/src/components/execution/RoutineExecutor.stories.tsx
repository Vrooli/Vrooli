import { Box, FormControl, FormLabel, RadioGroup, FormControlLabel, Radio, Select, MenuItem, Slider, Typography, Switch } from "@mui/material";
import { generatePK, type ResourceVersion, type Run, type RunStatus } from "@vrooli/shared";
import { useState } from "react";
import { signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { RoutineExecutor } from "./RoutineExecutor.js";

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
            <Box sx={{ p: 2, height: "100vh", overflow: "auto" }}>
                <Story />
            </Box>
        ),
    ],
};

// Mock ResourceVersion for testing
function createMockResourceVersion(name: string = "Data Processing Routine"): ResourceVersion {
    return {
        id: generatePK().toString(),
        name,
        description: "A comprehensive routine for processing data files",
        versionLabel: "1.0.2",
        isComplete: true,
        isPrivate: false,
        resourceSubType: "Routine",
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
    const [progressPercent, setProgressPercent] = useState(45);
    const [stepCount, setStepCount] = useState(8);
    const [currentStep, setCurrentStep] = useState(4);
    const [hasError, setHasError] = useState(false);
    
    // Mock data generation
    const runId = generatePK().toString();
    const resourceVersion = createMockResourceVersion(routineName);
    
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
            height: "100%"
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
                    {/* Routine Name */}
                    <FormControl size="small" fullWidth>
                        <FormLabel sx={{ fontSize: "0.875rem", mb: 1 }}>Routine Name</FormLabel>
                        <Select
                            value={routineName}
                            onChange={(e) => setRoutineName(e.target.value)}
                        >
                            <MenuItem value="Data Processing Routine">Data Processing Routine</MenuItem>
                            <MenuItem value="File Converter">File Converter</MenuItem>
                            <MenuItem value="API Integration">API Integration</MenuItem>
                            <MenuItem value="Machine Learning Pipeline">Machine Learning Pipeline</MenuItem>
                            <MenuItem value="Database Sync">Database Sync</MenuItem>
                            <MenuItem value="Report Generator">Report Generator</MenuItem>
                        </Select>
                    </FormControl>
                    
                    {/* Run Status */}
                    <FormControl component="fieldset" size="small">
                        <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Run Status</FormLabel>
                        <RadioGroup
                            value={runStatus}
                            onChange={(e) => setRunStatus(e.target.value as RunStatus)}
                            sx={{ gap: 0.5 }}
                        >
                            <FormControlLabel value="Scheduled" control={<Radio size="small" />} label="Scheduled" sx={{ m: 0 }} />
                            <FormControlLabel value="InProgress" control={<Radio size="small" />} label="In Progress" sx={{ m: 0 }} />
                            <FormControlLabel value="Running" control={<Radio size="small" />} label="Running" sx={{ m: 0 }} />
                            <FormControlLabel value="Completed" control={<Radio size="small" />} label="Completed" sx={{ m: 0 }} />
                            <FormControlLabel value="CompletedWithErrors" control={<Radio size="small" />} label="Completed (Errors)" sx={{ m: 0 }} />
                            <FormControlLabel value="Failed" control={<Radio size="small" />} label="Failed" sx={{ m: 0 }} />
                            <FormControlLabel value="Cancelled" control={<Radio size="small" />} label="Cancelled" sx={{ m: 0 }} />
                        </RadioGroup>
                    </FormControl>
                    
                    {/* Progress */}
                    <FormControl component="fieldset" size="small">
                        <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>
                            Progress: {progressPercent}%
                        </FormLabel>
                        <Slider
                            value={progressPercent}
                            onChange={(_, value) => setProgressPercent(value as number)}
                            min={0}
                            max={100}
                            marks={[
                                { value: 0, label: "0%" },
                                { value: 25, label: "25%" },
                                { value: 50, label: "50%" },
                                { value: 75, label: "75%" },
                                { value: 100, label: "100%" },
                            ]}
                        />
                    </FormControl>
                    
                    {/* Step Progress */}
                    <FormControl component="fieldset" size="small">
                        <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>
                            Current Step: {currentStep} of {stepCount}
                        </FormLabel>
                        <Box sx={{ mb: 2 }}>
                            <Typography variant="caption" color="text.secondary">
                                Total Steps:
                            </Typography>
                            <Slider
                                value={stepCount}
                                onChange={(_, value) => {
                                    const newStepCount = value as number;
                                    setStepCount(newStepCount);
                                    if (currentStep > newStepCount) {
                                        setCurrentStep(newStepCount);
                                    }
                                }}
                                min={1}
                                max={20}
                                marks
                                sx={{ mt: 1 }}
                            />
                        </Box>
                        <Box>
                            <Typography variant="caption" color="text.secondary">
                                Current Step:
                            </Typography>
                            <Slider
                                value={currentStep}
                                onChange={(_, value) => setCurrentStep(value as number)}
                                min={1}
                                max={stepCount}
                                marks
                                sx={{ mt: 1 }}
                            />
                        </Box>
                    </FormControl>
                    
                    {/* Display Options */}
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5 }}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={isCollapsed}
                                    onChange={(e) => setIsCollapsed(e.target.checked)}
                                    size="small"
                                />
                            }
                            label="Collapsed State"
                            sx={{ m: 0 }}
                        />
                        
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={chatMode}
                                    onChange={(e) => setChatMode(e.target.checked)}
                                    size="small"
                                />
                            }
                            label="Chat Mode"
                            sx={{ m: 0 }}
                        />
                        
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={hasError}
                                    onChange={(e) => setHasError(e.target.checked)}
                                    size="small"
                                />
                            }
                            label="Has Error"
                            sx={{ m: 0 }}
                        />
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
                overflow: "hidden"
            }}>
                <Typography variant="h5" sx={{ mb: 3 }}>RoutineExecutor Preview</Typography>
                
                <Box sx={{ 
                    bgcolor: "background.default", 
                    borderRadius: 2, 
                    p: 2,
                    minHeight: 500,
                    overflow: "auto"
                }}>
                    <Box sx={{ 
                        maxWidth: chatMode ? 600 : 800,
                        border: 1,
                        borderColor: "divider",
                        borderRadius: 1,
                        overflow: "hidden"
                    }}>
                        <RoutineExecutor
                            runId={runId}
                            resourceVersion={resourceVersion}
                            isCollapsed={isCollapsed}
                            onToggleCollapse={handleToggleCollapse}
                            onRemove={chatMode ? handleRemove : undefined}
                            onClose={!chatMode ? handleClose : undefined}
                            className={chatMode ? "chat-routine-executor" : "standalone-routine-executor"}
                            chatMode={chatMode}
                        />
                    </Box>
                </Box>
                
                {/* State Information */}
                <Box sx={{ mt: 3 }}>
                    <Typography variant="subtitle2" color="text.secondary">
                        Current Configuration:
                    </Typography>
                    <Typography variant="body2" color="text.secondary" component="pre" sx={{ mt: 1, fontSize: "0.75rem" }}>
                        {JSON.stringify({
                            runId,
                            routineName,
                            runStatus,
                            progressPercent,
                            currentStep,
                            stepCount,
                            isCollapsed,
                            chatMode,
                            hasError,
                        }, null, 2)}
                    </Typography>
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

/**
 * Collapsed State: Shows the executor in collapsed state
 */
export function Collapsed() {
    const runId = generatePK().toString();
    const resourceVersion = createMockResourceVersion("File Converter");
    
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
                    isCollapsed={true}
                    onToggleCollapse={action("onToggleCollapse")}
                    onClose={action("onClose")}
                />
            </Box>
        </Box>
    );
}

Collapsed.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * Multiple States: Shows different execution states side by side
 */
export function MultipleStates() {
    const states: Array<{ name: string; status: RunStatus; isCollapsed?: boolean }> = [
        { name: "Scheduled Task", status: "Scheduled" },
        { name: "Running Process", status: "InProgress" },
        { name: "Completed Successfully", status: "Completed" },
        { name: "Failed Process", status: "Failed" },
        { name: "Completed (Collapsed)", status: "Completed", isCollapsed: true },
    ];
    
    return (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2, maxWidth: 1000, mx: "auto" }}>
            <Typography variant="h6" sx={{ mb: 2 }}>Different Execution States</Typography>
            {states.map((state, index) => (
                <Box key={index} sx={{ 
                    border: 1,
                    borderColor: "divider",
                    borderRadius: 1,
                    overflow: "hidden"
                }}>
                    <RoutineExecutor
                        runId={generatePK().toString()}
                        resourceVersion={createMockResourceVersion(state.name)}
                        isCollapsed={state.isCollapsed}
                        onToggleCollapse={action(`onToggleCollapse-${index}`)}
                        onClose={action(`onClose-${index}`)}
                        chatMode={true}
                    />
                </Box>
            ))}
        </Box>
    );
}

MultipleStates.parameters = {
    session: signedInPremiumWithCreditsSession,
};