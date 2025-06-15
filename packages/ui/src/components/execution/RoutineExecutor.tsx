import { ResourceVersion, RunTaskInfo, Run, RunStatus } from "@vrooli/shared";
import { useCallback, useEffect, useReducer } from "react";
import { Box } from "@mui/material";
import { ExecutionTimeline } from "./ExecutionTimeline.js";
import { ExecutionHeader } from "./ExecutionHeader.js";
import { StepDetails } from "./StepDetails.js";
import { DecisionPrompt } from "./DecisionPrompt.js";
import { useSocketRun } from "../../hooks/runs.js";
import { noop } from "@vrooli/shared";

interface RoutineExecutorProps {
    runId: string;
    resourceVersion: ResourceVersion;
    runStatus?: string; // Pass the actual run status from ChatMessageRunConfig
    isCollapsed?: boolean;
    defaultCollapsed?: boolean;
    onToggleCollapse?: () => void;
    onClose?: () => void;
    onRemove?: () => void;
    className?: string;
    chatMode?: boolean;
    isFirstInGroup?: boolean;
    isLastInGroup?: boolean;
    showInUnifiedContainer?: boolean;
}

export interface ExecutionStep {
    id: string;
    name: string;
    description?: string;
    status: "pending" | "running" | "completed" | "failed";
    startTime?: string;
    endTime?: string;
    error?: string;
    errorDetails?: string;
    inputs?: Record<string, unknown>;
    outputs?: Record<string, unknown>;
    progress?: number;
    retryCount?: number;
}

interface BranchInfo {
    id: string;
    name: string;
    stepIndex: number;
}

interface DeferredDecisionData {
    stepId: string;
    options: Array<{ id: string; label: string }>;
    prompt: string;
}

interface ExecutionState {
    run: Run | null;
    steps: ExecutionStep[];
    currentStepIndex: number;
    activeBranches: BranchInfo[];
    contextValues: Record<string, unknown>;
    pendingDecision: DeferredDecisionData | null;
    metrics: {
        elapsedTime: number;
        contextSwitches: number;
    };
}

type ExecutionAction = 
    | { type: "UPDATE_RUN"; payload: RunTaskInfo }
    | { type: "DECISION_REQUESTED"; payload: DeferredDecisionData }
    | { type: "DECISION_SUBMITTED"; stepId: string }
    | { type: "UPDATE_METRICS"; elapsedTime: number; contextSwitches: number }
    | { type: "SET_RUN"; run: Run };

function executionReducer(state: ExecutionState, action: ExecutionAction): ExecutionState {
    switch (action.type) {
        case "UPDATE_RUN": {
            const { payload } = action;
            // Update run status
            const updatedRun = state.run ? {
                ...state.run,
                status: payload.runStatus || state.run.status,
            } : null;
            
            // TODO: Process step updates from payload
            // For now, just update the run
            return {
                ...state,
                run: updatedRun,
            };
        }
        case "DECISION_REQUESTED":
            return {
                ...state,
                pendingDecision: action.payload,
            };
        case "DECISION_SUBMITTED":
            return {
                ...state,
                pendingDecision: null,
            };
        case "UPDATE_METRICS":
            return {
                ...state,
                metrics: {
                    elapsedTime: action.elapsedTime,
                    contextSwitches: action.contextSwitches,
                },
            };
        case "SET_RUN":
            return {
                ...state,
                run: action.run,
                // Initialize steps based on run data
                steps: createStepsFromRun(action.run),
            };
        default:
            return state;
    }
}

function createStepsFromRun(run: Run): ExecutionStep[] {
    // TODO: Parse run data to create steps
    // For now, return mock data
    return [
        {
            id: "step-1",
            name: "Initialize",
            status: "completed",
            startTime: new Date().toISOString(),
        },
        {
            id: "step-2",
            name: "Process Data",
            status: "running",
            startTime: new Date().toISOString(),
        },
        {
            id: "step-3",
            name: "Generate Output",
            status: "pending",
        },
    ];
}

const initialState: ExecutionState = {
    run: null,
    steps: [],
    currentStepIndex: 0,
    activeBranches: [],
    contextValues: {},
    pendingDecision: null,
    metrics: {
        elapsedTime: 0,
        contextSwitches: 0,
    },
};

export function RoutineExecutor({
    runId,
    resourceVersion,
    runStatus,
    isCollapsed = false,
    defaultCollapsed = false,
    onToggleCollapse = noop,
    onClose = noop,
    onRemove = noop,
    className = "",
    chatMode = false,
    isFirstInGroup = true,
    isLastInGroup = true,
    showInUnifiedContainer = false,
}: RoutineExecutorProps) {
    const [state, dispatch] = useReducer(executionReducer, {
        ...initialState,
        // Initialize with mock steps for display
        steps: [
            {
                id: "step-1",
                name: "Initialize Environment",
                status: "completed",
                startTime: new Date(Date.now() - 5000).toISOString(),
                endTime: new Date(Date.now() - 4000).toISOString(),
            },
            {
                id: "step-2", 
                name: "Process Input Data",
                status: "completed",
                startTime: new Date(Date.now() - 4000).toISOString(),
                endTime: new Date(Date.now() - 2000).toISOString(),
            },
            {
                id: "step-3",
                name: "Execute Main Logic",
                status: "running",
                startTime: new Date(Date.now() - 2000).toISOString(),
            },
            {
                id: "step-4",
                name: "Generate Output",
                status: "pending",
            },
            {
                id: "step-5",
                name: "Cleanup Resources",
                status: "pending",
            },
        ],
        currentStepIndex: 2, // Currently on step 3 (0-indexed)
        metrics: {
            elapsedTime: 5, // Start with some elapsed time
            contextSwitches: 0,
        },
    });


    // Handle run updates from socket
    const handleRunUpdate = useCallback((updatedRun: Run | null) => {
        if (updatedRun) {
            dispatch({ type: "SET_RUN", run: updatedRun });
        }
    }, []);

    // Handle decision requests from socket
    const handleDecisionRequest = useCallback((decisionData: DeferredDecisionData) => {
        dispatch({ type: "DECISION_REQUESTED", payload: decisionData });
    }, []);

    // Connect to socket for real-time updates
    useSocketRun({
        runId,
        applyRunUpdate: handleRunUpdate,
        onDecisionRequest: handleDecisionRequest,
    });

    // Handle decision submission
    const handleDecisionSubmit = useCallback((stepId: string, decision: unknown) => {
        // TODO: Emit decision to server via socket
        dispatch({ type: "DECISION_SUBMITTED", stepId });
    }, []);

    // Handle step selection
    const handleStepSelect = useCallback((stepIndex: number) => {
        // TODO: Update current step index
    }, []);

    // Calculate progress based on completed steps
    const calculateProgress = useCallback(() => {
        if (state.steps.length === 0) return 0;
        const completedSteps = state.steps.filter(step => step.status === "completed").length;
        const runningSteps = state.steps.filter(step => step.status === "running").length;
        // Give partial credit for running steps
        return Math.round(((completedSteps + runningSteps * 0.5) / state.steps.length) * 100);
    }, [state.steps]);

    const progress = calculateProgress();

    // Update metrics periodically
    useEffect(() => {
        const interval = setInterval(() => {
            // TODO: Calculate real elapsed time
            dispatch({
                type: "UPDATE_METRICS",
                elapsedTime: state.metrics.elapsedTime + 1,
                contextSwitches: state.metrics.contextSwitches,
            });
        }, 1000);

        return () => clearInterval(interval);
    }, [state.metrics]);

    // Always return the full structure, but hide content when collapsed
    const shouldShowRoundedCorners = !showInUnifiedContainer || (isFirstInGroup && isLastInGroup);

    return (
        <Box 
            className={`routine-executor ${className}`}
            sx={{
                display: "flex",
                flexDirection: "column",
                height: chatMode ? "auto" : "100%",
                minHeight: chatMode && !isCollapsed ? 200 : "auto",
                maxHeight: chatMode && !isCollapsed ? 400 : "none",
                minWidth: chatMode ? 600 : 700,
                bgcolor: showInUnifiedContainer ? "transparent" : "background.paper",
                borderRadius: showInUnifiedContainer 
                    ? 0 
                    : shouldShowRoundedCorners ? 3 : 0,
                borderTopLeftRadius: showInUnifiedContainer && isFirstInGroup ? 0 : undefined,
                borderTopRightRadius: showInUnifiedContainer && isFirstInGroup ? 0 : undefined,
                borderBottomLeftRadius: showInUnifiedContainer && !isLastInGroup ? 0 : undefined,
                borderBottomRightRadius: showInUnifiedContainer && !isLastInGroup ? 0 : undefined,
                boxShadow: showInUnifiedContainer ? "none" : chatMode ? 1 : 2,
                overflow: "hidden",
            }}
        >
            <ExecutionHeader
                title={resourceVersion.name || "Routine"}
                description={chatMode ? undefined : resourceVersion.description}
                progress={progress}
                elapsedTime={state.metrics.elapsedTime}
                isCollapsed={isCollapsed}
                onToggleCollapse={onToggleCollapse}
                onClose={chatMode ? onRemove : onClose}
                chatMode={chatMode}
                currentStepTitle={state.steps[state.currentStepIndex]?.name}
                runStatus={runStatus || state.run?.status}
                onPause={() => console.log("Pause run:", runId)}
                onResume={() => console.log("Resume run:", runId)}
                onRetry={() => console.log("Retry run:", runId)}
            />
            
            {!isCollapsed && (
                <Box sx={{ display: "flex", flex: 1, overflow: "hidden" }}>
                    <Box sx={{ 
                        width: chatMode ? 220 : 280, 
                        borderRight: 1, 
                        borderColor: "divider",
                        display: chatMode ? { xs: "none", sm: "block" } : "block"
                    }}>
                        <ExecutionTimeline
                            steps={state.steps}
                            currentStepIndex={state.currentStepIndex}
                            onStepSelect={handleStepSelect}
                            compact={chatMode}
                        />
                    </Box>
                    
                    <Box sx={{ flex: 1, overflow: "auto", p: 2 }}>
                        {state.pendingDecision ? (
                            <DecisionPrompt
                                decision={state.pendingDecision}
                                onSubmit={(decision) => handleDecisionSubmit(state.pendingDecision!.stepId, decision)}
                                onCancel={() => dispatch({ type: "DECISION_SUBMITTED", stepId: state.pendingDecision!.stepId })}
                            />
                        ) : (
                            <StepDetails
                                step={state.steps[state.currentStepIndex]}
                                contextValues={state.contextValues}
                                totalSteps={state.steps.length}
                                currentStepIndex={state.currentStepIndex}
                            />
                        )}
                    </Box>
                </Box>
            )}
        </Box>
    );
}