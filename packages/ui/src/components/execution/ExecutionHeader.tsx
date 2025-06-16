import { Box, IconButton, LinearProgress, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { OrbitalSpinner, CircularProgress } from "../indicators/CircularProgress.js";

interface ExecutionHeaderProps {
    title: string;
    description?: string;
    progress: number;
    elapsedTime: number;
    isCollapsed: boolean;
    onToggleCollapse: () => void;
    onClose: () => void;
    chatMode?: boolean;
    currentStepTitle?: string;
    runStatus?: string; // "Completed" | "InProgress" | "Failed" | "Scheduled" | "Paused" | "Cancelled"
    onPause?: () => void;
    onResume?: () => void;
    onRetry?: () => void;
}

function formatElapsedTime(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
        return `${hours}:${minutes.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
    }
    return `${minutes}:${secs.toString().padStart(2, "0")}`;
}

export function ExecutionHeader({
    title,
    description,
    progress,
    elapsedTime,
    isCollapsed,
    onToggleCollapse,
    onClose,
    chatMode = false,
    currentStepTitle,
    runStatus,
    onPause,
    onResume,
    onRetry,
}: ExecutionHeaderProps) {
    const theme = useTheme();
    const [displayTime, setDisplayTime] = useState(elapsedTime);

    useEffect(() => {
        setDisplayTime(elapsedTime);
    }, [elapsedTime]);

    const handleToggleCollapse = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        onToggleCollapse();
    }, [onToggleCollapse]);

    const handleClose = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        onClose();
    }, [onClose]);

    // Remove separate collapsed state rendering - always use the same structure

    return (
        <Box>
            <Box
                onClick={handleToggleCollapse}
                sx={{
                    display: "flex",
                    alignItems: "center",
                    p: chatMode ? 1 : 1.5,
                    borderBottom: 1,
                    borderColor: "divider",
                    cursor: "pointer",
                    "&:hover": {
                        bgcolor: "action.hover",
                    },
                }}
            >
                <Box sx={{ display: "flex", alignItems: "center" }}>
                    <IconButton 
                        size="small" 
                        onClick={(e) => {
                            e.stopPropagation(); // Prevent double-click when clicking the icon directly
                            handleToggleCollapse(e);
                        }}
                    >
                        <IconCommon name={isCollapsed ? "ExpandMore" : "ExpandLess"} size={chatMode ? 20 : 24} />
                    </IconButton>
                    {/* Show spinner when run is in progress */}
                    {(runStatus === "InProgress" || runStatus === "Running") && (
                        <Box sx={{ ml: 1, display: "flex", alignItems: "center" }}>
                            {theme.palette.mode === "dark" ? (
                                <OrbitalSpinner size={chatMode ? 28 : 32} />
                            ) : (
                                <CircularProgress 
                                    size={chatMode ? 28 : 32} 
                                    thickness={3}
                                    variant="primary"
                                />
                            )}
                        </Box>
                    )}
                </Box>
                <Box sx={{ flex: 1, mx: 1 }}>
                    <Typography variant={chatMode ? "subtitle2" : "h6"} sx={{ fontWeight: 500 }}>
                        {title}
                    </Typography>
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1.5, mt: 0.25 }}>
                        <Typography variant={chatMode ? "caption" : "body2"} color="text.secondary">
                            {progress}% complete
                        </Typography>
                        <Typography variant={chatMode ? "caption" : "body2"} color="text.secondary">
                            •
                        </Typography>
                        <Typography variant={chatMode ? "caption" : "body2"} color="text.secondary">
                            {formatElapsedTime(displayTime)} elapsed
                        </Typography>
                    </Box>
                </Box>
                {/* Show current step title when collapsed */}
                {isCollapsed && currentStepTitle && (
                    <Box sx={{ 
                        flex: 1, 
                        mx: 1,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "flex-end",
                    }}>
                        <Typography 
                            variant={chatMode ? "caption" : "body2"} 
                            color="text.secondary"
                            sx={{ 
                                fontStyle: "italic",
                                textAlign: "right",
                                overflow: "hidden",
                                textOverflow: "ellipsis",
                                display: "-webkit-box",
                                WebkitLineClamp: 2,
                                WebkitBoxOrient: "vertical",
                                maxWidth: "200px",
                            }}
                        >
                            {currentStepTitle}
                        </Typography>
                    </Box>
                )}
                {/* Action buttons based on run status */}
                {runStatus && (
                    <>
                        {/* Pause button for active runs */}
                        {(runStatus === "InProgress" || runStatus === "Running") && onPause && (
                            <IconButton 
                                size="small" 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onPause();
                                }}
                                title="Pause"
                                color="secondary"
                            >
                                <IconCommon name="Pause" size={chatMode ? 20 : 24} />
                            </IconButton>
                        )}
                        {/* Play button for paused/scheduled runs */}
                        {(runStatus === "Scheduled" || runStatus === "Paused") && onResume && (
                            <IconButton 
                                size="small" 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onResume();
                                }}
                                title={runStatus === "Scheduled" ? "Start" : "Resume"}
                                color="secondary"
                            >
                                <IconCommon name="Play" size={chatMode ? 20 : 24} />
                            </IconButton>
                        )}
                        {/* Retry button for failed runs */}
                        {runStatus === "Failed" && onRetry && (
                            <IconButton 
                                size="small" 
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onRetry();
                                }}
                                title="Retry"
                                color="secondary"
                            >
                                <IconCommon name="Refresh" size={chatMode ? 20 : 24} />
                            </IconButton>
                        )}
                        {/* Note: Completed and Cancelled runs show no action buttons */}
                    </>
                )}
                {!chatMode && (
                    <IconButton 
                        size="small" 
                        onClick={(e) => {
                            e.stopPropagation(); // Prevent triggering collapse when clicking close
                            handleClose(e);
                        }}
                    >
                        <IconCommon name="Close" size={24} />
                    </IconButton>
                )}
            </Box>
            {!isCollapsed && (
                <LinearProgress
                    variant="determinate"
                    value={progress}
                    sx={{
                        height: 4,
                        bgcolor: "action.hover",
                        "& .MuiLinearProgress-bar": {
                            bgcolor: "primary.main",
                        },
                    }}
                />
            )}
        </Box>
    );
}