import { Box, Typography, Paper, Divider, LinearProgress, Chip, Alert } from "@mui/material";
import { ExecutionStep } from "./RoutineExecutor.js";
import { IconCommon } from "../../icons/Icons.js";

interface StepDetailsProps {
    step?: ExecutionStep;
    contextValues: Record<string, unknown>;
    totalSteps: number;
    currentStepIndex: number;
}

function formatValue(value: unknown): string {
    if (value === null || value === undefined) return "—";
    if (typeof value === "object") return JSON.stringify(value, null, 2);
    return String(value);
}

function calculateDuration(startTime?: string, endTime?: string): string {
    if (!startTime) return "—";
    const start = new Date(startTime).getTime();
    const end = endTime ? new Date(endTime).getTime() : Date.now();
    const duration = Math.floor((end - start) / 1000);
    
    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
}

export function StepDetails({
    step,
    contextValues,
    totalSteps,
    currentStepIndex,
}: StepDetailsProps) {
    if (!step) {
        return (
            <Box sx={{ p: 3, textAlign: "center" }}>
                <Typography color="text.secondary">
                    No step selected
                </Typography>
            </Box>
        );
    }

    const statusConfig = {
        pending: { color: "default", icon: "Add" },
        running: { color: "primary", icon: "Pause" },
        completed: { color: "success", icon: "Complete" },
        failed: { color: "error", icon: "Error" },
    } as const;

    return (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            {/* Header with step info */}
            <Box>
                <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 1 }}>
                    <Typography variant="h5">
                        {step.name}
                    </Typography>
                    <Chip
                        label={step.status.charAt(0).toUpperCase() + step.status.slice(1)}
                        color={statusConfig[step.status].color as any}
                        size="small"
                        icon={<IconCommon name={statusConfig[step.status].icon} size={16} />}
                    />
                    <Typography variant="body2" color="text.secondary">
                        Step {currentStepIndex + 1} of {totalSteps}
                    </Typography>
                </Box>
                {step.description && (
                    <Typography variant="body1" color="text.secondary">
                        {step.description}
                    </Typography>
                )}
            </Box>

            {/* Execution Metrics */}
            <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                    Execution Details
                </Typography>
                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))", gap: 2, mt: 1 }}>
                    <Box>
                        <Typography variant="caption" color="text.secondary">
                            Duration
                        </Typography>
                        <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {calculateDuration(step.startTime, step.endTime)}
                        </Typography>
                    </Box>
                    {step.startTime && (
                        <Box>
                            <Typography variant="caption" color="text.secondary">
                                Started
                            </Typography>
                            <Typography variant="body2">
                                {new Date(step.startTime).toLocaleTimeString()}
                            </Typography>
                        </Box>
                    )}
                    {step.endTime && (
                        <Box>
                            <Typography variant="caption" color="text.secondary">
                                Completed
                            </Typography>
                            <Typography variant="body2">
                                {new Date(step.endTime).toLocaleTimeString()}
                            </Typography>
                        </Box>
                    )}
                    {step.retryCount !== undefined && step.retryCount > 0 && (
                        <Box>
                            <Typography variant="caption" color="text.secondary">
                                Retries
                            </Typography>
                            <Typography variant="body2" color="warning.main" sx={{ fontWeight: 500 }}>
                                {step.retryCount}
                            </Typography>
                        </Box>
                    )}
                </Box>
                
                {/* Progress indicator for running steps */}
                {step.status === "running" && (
                    <Box sx={{ mt: 2 }}>
                        <Box sx={{ display: "flex", justifyContent: "space-between", mb: 1 }}>
                            <Typography variant="caption" color="text.secondary">
                                Processing...
                            </Typography>
                            {step.progress !== undefined && (
                                <Typography variant="caption" color="primary">
                                    {Math.round(step.progress)}%
                                </Typography>
                            )}
                        </Box>
                        <LinearProgress 
                            variant={step.progress !== undefined ? "determinate" : "indeterminate"}
                            value={step.progress}
                        />
                    </Box>
                )}
            </Paper>

            {/* Error Alert */}
            {step.error && (
                <Alert severity="error" icon={<IconCommon name="Error" size={20} />}>
                    <Typography variant="subtitle2" gutterBottom>
                        Error Details
                    </Typography>
                    <Typography variant="body2">
                        {step.error}
                    </Typography>
                    {step.errorDetails && (
                        <Typography variant="caption" component="pre" sx={{ mt: 1, whiteSpace: "pre-wrap" }}>
                            {step.errorDetails}
                        </Typography>
                    )}
                </Alert>
            )}

            {/* Inputs */}
            {step.inputs && Object.keys(step.inputs).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Inputs
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(step.inputs).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}

            {/* Outputs */}
            {step.outputs && Object.keys(step.outputs).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Outputs
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(step.outputs).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}

            {/* Context Values */}
            {Object.keys(contextValues).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Context
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(contextValues).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}
        </Box>
    );
}