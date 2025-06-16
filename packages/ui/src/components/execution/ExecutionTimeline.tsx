import { Box, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { ExecutionStep } from "./RoutineExecutor.js";

interface ExecutionTimelineProps {
    steps: ExecutionStep[];
    currentStepIndex: number;
    onStepSelect: (index: number) => void;
    compact?: boolean;
}

const statusIcons = {
    pending: "Add", // Using Add as placeholder for unchecked state
    running: "Ellipsis", // Ellipsis icon for running state
    completed: "Complete",
    failed: "Error",
} as const;

const statusColors = {
    pending: "text.secondary",
    running: "primary.main",
    completed: "success.main",
    failed: "error.main",
};

// Animated ellipsis component
function AnimatedEllipsis() {
    const [dots, setDots] = useState("");

    useEffect(() => {
        const interval = setInterval(() => {
            setDots(prev => {
                if (prev.length >= 3) return "";
                return prev + ".";
            });
        }, 500); // Change dots every 500ms

        return () => clearInterval(interval);
    }, []);

    return <span>{dots}</span>;
}

// Animated ellipsis icon component
function AnimatedEllipsisIcon({ size, fill }: { size: number; fill: string }) {
    const [dotCount, setDotCount] = useState(1);
    
    useEffect(() => {
        const interval = setInterval(() => {
            setDotCount(prev => prev >= 3 ? 1 : prev + 1);
        }, 500); // Change dots every 500ms
        
        return () => clearInterval(interval);
    }, []);
    
    const dotSize = size / 10;
    const spacing = size / 4;
    
    return (
        <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} fill="#9e9e9e">
            {/* Dot 1 */}
            <circle
                cx={size / 2 - spacing}
                cy={size / 2}
                r={dotSize}
                opacity={dotCount >= 1 ? 0.8 : 0.2}
            />
            {/* Dot 2 */}
            <circle
                cx={size / 2}
                cy={size / 2}
                r={dotSize}
                opacity={dotCount >= 2 ? 0.8 : 0.2}
            />
            {/* Dot 3 */}
            <circle
                cx={size / 2 + spacing}
                cy={size / 2}
                r={dotSize}
                opacity={dotCount >= 3 ? 0.8 : 0.2}
            />
        </svg>
    );
}

// Helper function to calculate elapsed time
function calculateElapsedTime(startTime?: string, endTime?: string): string {
    if (!startTime) return "";
    const start = new Date(startTime).getTime();
    const end = endTime ? new Date(endTime).getTime() : Date.now();
    const duration = Math.floor((end - start) / 1000);

    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
}

export function ExecutionTimeline({
    steps,
    currentStepIndex,
    onStepSelect,
    compact = false,
}: ExecutionTimelineProps) {
    if (steps.length === 0) return null;

    const stepHeight = compact ? 56 : 64;
    const circleSize = compact ? 28 : 32;
    const lineHeight = (steps.length - 1) * stepHeight;

    // Calculate the progress line height based on completed steps
    const completedSteps = steps.filter((step, index) =>
        step.status === "completed" && index < steps.length - 1
    ).length;
    const progressLineHeight = completedSteps * stepHeight;

    return (
        <Box sx={{ p: compact ? 1.5 : 2, position: "relative", minWidth: compact ? 200 : 260 }}>
            {/* Background connecting line */}
            {steps.length > 1 && (
                <Box
                    sx={{
                        position: "absolute",
                        left: compact ? 32 : 36,
                        top: compact ? 38 : 48,
                        width: 3,
                        height: lineHeight,
                        bgcolor: "divider",
                        borderRadius: 1.5,
                        zIndex: 0,
                    }}
                />
            )}

            {/* Progress line showing completed segments */}
            {steps.length > 1 && progressLineHeight > 0 && (
                <Box
                    sx={{
                        position: "absolute",
                        left: compact ? 32 : 36,
                        top: compact ? 38 : 48,
                        width: 3,
                        height: progressLineHeight,
                        bgcolor: "success.main",
                        borderRadius: 1.5,
                        zIndex: 0,
                        transition: "height 0.3s ease-in-out",
                    }}
                />
            )}

            {steps.map((step, index) => {
                const isActive = index === currentStepIndex;
                const isPast = index < currentStepIndex;
                const isFuture = index > currentStepIndex;

                return (
                    <Box
                        key={step.id}
                        sx={{
                            position: "relative",
                            height: stepHeight,
                            display: "flex",
                            alignItems: "center",
                        }}
                    >
                        <Box
                            onClick={() => onStepSelect(index)}
                            sx={{
                                display: "flex",
                                alignItems: "center",
                                gap: 1.5,
                                py: 1,
                                px: 1,
                                borderRadius: 2,
                                cursor: "pointer",
                                width: "100%",
                                bgcolor: isActive ? "action.selected" : "transparent",
                                "&:hover": {
                                    bgcolor: "action.hover",
                                },
                            }}
                        >
                            {/* Circular step indicator */}
                            <Box
                                sx={{
                                    position: "relative",
                                    zIndex: 1,
                                    width: circleSize,
                                    height: circleSize,
                                    borderRadius: "50%",
                                    display: "flex",
                                    alignItems: "center",
                                    justifyContent: "center",
                                    bgcolor: step.status === "completed"
                                        ? "success.main"
                                        : step.status === "running"
                                            ? "background.paper"
                                            : step.status === "failed"
                                                ? "error.main"
                                                : "background.paper",
                                    border: step.status === "pending" ? 2 : step.status === "running" ? 3 : 0,
                                    borderColor: step.status === "pending" ? "divider" : step.status === "running" ? "info.main" : "transparent",
                                }}
                            >
                                {step.status === "running" ? (
                                    <AnimatedEllipsisIcon 
                                        size={compact ? 16 : 18} 
                                        fill="currentColor"
                                    />
                                ) : statusIcons[step.status] && (
                                    <IconCommon
                                        name={statusIcons[step.status]}
                                        size={compact ? 16 : 18}
                                        fill={step.status === "completed" ? "white" : "text.secondary"}
                                    />
                                )}
                            </Box>

                            <Box sx={{ flex: 1, minWidth: 0 }}>
                                <Typography
                                    variant={compact ? "body2" : "body1"}
                                    sx={{
                                        fontWeight: isActive ? 600 : 500,
                                        color: isActive ? "text.primary" : "text.secondary",
                                        lineHeight: 1.2,
                                    }}
                                >
                                    {step.name}
                                </Typography>
                                {step.status === "completed" && isPast && (
                                    <Typography variant="caption" color="success.main" sx={{ fontWeight: 500 }}>
                                        Completed{calculateElapsedTime(step.startTime, step.endTime) && ` • ${calculateElapsedTime(step.startTime, step.endTime)}`}
                                    </Typography>
                                )}
                                {step.error && (
                                    <Typography variant="caption" color="error" sx={{ fontWeight: 500 }}>
                                        {step.error}
                                    </Typography>
                                )}
                            </Box>
                        </Box>
                    </Box>
                );
            })}
        </Box>
    );
}