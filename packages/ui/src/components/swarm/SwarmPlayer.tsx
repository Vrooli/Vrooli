import { Badge, Box, LinearProgress, Skeleton, Typography } from "@mui/material";
import { IconButton } from "../buttons/IconButton.js";
import { styled, useTheme } from "@mui/material/styles";
import { type ChatConfigObject, ExecutionStates, PendingToolCallStatus, noop } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import "./SwarmPlayer.css";

const PlayerContainer = styled(Box)(({ theme }) => ({
    position: "relative",
    backgroundColor: theme.palette.mode === "dark" ? theme.palette.grey[900] : theme.palette.grey[100],
    borderRadius: theme.shape.borderRadius * 3,
    overflow: "hidden",
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    transition: "all 0.3s ease",
    cursor: "pointer",
    width: "100%",
    maxWidth: "700px",
    margin: "auto",
    boxSizing: "border-box",
    "&:hover": {
        backgroundColor: theme.palette.mode === "dark" ? theme.palette.grey[800] : theme.palette.grey[200],
    },
}));

const PlayerContent = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: theme.spacing(1),
    width: "100%",
    padding: theme.spacing(1, 2),
}));

const InfoSection = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(1),
    flex: 1,
    minWidth: 0, // Allow text truncation
    overflow: "hidden", // Prevent content from extending out
}));

const StatusBadge = styled(Box)<{ status: string }>(({ theme, status }) => {
    const getStatusColor = () => {
        switch (status) {
            case ExecutionStates.RUNNING:
                return theme.palette.success.main;
            case ExecutionStates.PAUSED:
                return theme.palette.warning.main;
            case ExecutionStates.FAILED:
                return theme.palette.error.main;
            case ExecutionStates.STOPPED:
            case ExecutionStates.TERMINATED:
                return theme.palette.text.disabled;
            default:
                return theme.palette.text.secondary;
        }
    };

    return {
        display: "flex",
        alignItems: "center",
        gap: theme.spacing(0.5),
        color: getStatusColor(),
        fontSize: "0.875rem",
        fontWeight: 500,
        flexShrink: 0, // Prevent badge from shrinking
        whiteSpace: "nowrap", // Keep text on one line
    };
});

const CreditDisplay = styled(Typography)(({ theme }) => ({
    fontSize: "0.875rem",
    color: theme.palette.text.secondary,
    fontFamily: "monospace",
    flexShrink: 0,
    whiteSpace: "nowrap",
}));

const ControlSection = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(0.5),
    flexShrink: 0,
}));

export interface SwarmPlayerProps {
    swarmConfig: ChatConfigObject | null;
    swarmStatus?: string;
    chatId?: string | null;
    onStart?: () => void;
    onPause?: () => void;
    onResume?: () => void;
    onStop?: () => void;
    isLoading?: boolean;
}

/**
 * A compact "player" component that shows swarm status and controls.
 * Clicking it opens the detailed SwarmDetailPanel.
 */
export function SwarmPlayer({
    swarmConfig,
    swarmStatus = ExecutionStates.UNINITIALIZED,
    chatId,
    onStart = noop,
    onPause = noop,
    onResume = noop,
    onStop = noop,
    isLoading = false,
}: SwarmPlayerProps) {
    const { t } = useTranslation();
    // TODO: Add these translation keys:
    // Running, Paused, Idle, Stopped, Failed, Starting, Inactive
    // Credits, PauseSwarm, ResumeSwarm, StopSwarm
    const theme = useTheme();
    const goalRef = useRef<HTMLDivElement>(null);
    const [isOverflowing, setIsOverflowing] = useState(false);

    // Calculate progress based on subtasks
    const progress = useMemo(() => {
        if (!swarmConfig?.subtasks || swarmConfig.subtasks.length === 0) {
            return 0;
        }
        const completed = swarmConfig.subtasks.filter(task => task.status === "done").length;
        return Math.round((completed / swarmConfig.subtasks.length) * 100);
    }, [swarmConfig?.subtasks]);

    // Calculate pending approvals count
    const pendingApprovalsCount = useMemo(() => {
        if (!swarmConfig?.pendingToolCalls) return 0;
        return swarmConfig.pendingToolCalls.filter(
            tc => tc.status === PendingToolCallStatus.PENDING_APPROVAL,
        ).length;
    }, [swarmConfig?.pendingToolCalls]);

    // Format credits for display as USD
    const formattedCredits = useMemo(() => {
        if (!swarmConfig?.stats?.totalCredits) return "$0.00";
        try {
            const credits = BigInt(swarmConfig.stats.totalCredits);
            // Convert credits to USD: 1 USD = 100,000,000 credits
            const usd = Number(credits / BigInt(1e8)) / 100;
            return `$${usd.toFixed(2)}`;
        } catch (error) {
            console.error("Error formatting credits:", error);
            return "$0.00";
        }
    }, [swarmConfig?.stats?.totalCredits]);

    // Get status display text
    const statusText = useMemo(() => {
        switch (swarmStatus) {
            case ExecutionStates.RUNNING:
                return t("Running");
            case ExecutionStates.PAUSED:
                return t("Paused");
            case ExecutionStates.IDLE:
                return t("Idle");
            case ExecutionStates.STOPPED:
                return t("Stopped");
            case ExecutionStates.FAILED:
                return t("Failed");
            case ExecutionStates.STARTING:
                return t("Starting");
            default:
                return t("Inactive");
        }
    }, [swarmStatus, t]);

    // Get appropriate status icon
    const statusIcon = useMemo(() => {
        switch (swarmStatus) {
            case ExecutionStates.RUNNING:
                return "Play";
            case ExecutionStates.PAUSED:
                return "Pause";
            case ExecutionStates.FAILED:
                return "Error";
            case ExecutionStates.STOPPED:
            case ExecutionStates.TERMINATED:
                return "Cancel";
            default:
                return "Info";
        }
    }, [swarmStatus]);

    const handlePlayerClick = useCallback(() => {
        // Open the detail panel with swarm data
        PubSub.get().publish("menu", { 
            id: ELEMENT_IDS.RightDrawer, 
            isOpen: true,
            data: { 
                view: "swarmDetail",
                chatId,
                swarmConfig,
                swarmStatus,
                onApproveToolCall: (pendingId: string) => {
                    // TODO: Implement tool approval via API
                    console.log("Approve tool:", pendingId);
                },
                onRejectToolCall: (pendingId: string, reason?: string) => {
                    // TODO: Implement tool rejection via API
                    console.log("Reject tool:", pendingId, reason);
                },
                onStart,
                onPause,
                onResume,
                onStop,
            },
        });
    }, [chatId, swarmConfig, swarmStatus, onStart, onPause, onResume, onStop]);

    // Check if goal text overflows
    useEffect(() => {
        const checkOverflow = () => {
            if (goalRef.current) {
                const isOverflow = goalRef.current.scrollWidth > goalRef.current.clientWidth;
                setIsOverflowing(isOverflow);
            }
        };

        checkOverflow();
        window.addEventListener("resize", checkOverflow);
        return () => window.removeEventListener("resize", checkOverflow);
    }, [swarmConfig?.goal]);

    const handleControlClick = useCallback((e: React.MouseEvent, action: () => void) => {
        e.stopPropagation();
        action();
    }, []);

    // Don't show player if no swarm config
    if (!swarmConfig && !isLoading) {
        return null;
    }

    return (
        <Badge 
            badgeContent={pendingApprovalsCount} 
            color="warning"
            overlap="rectangular"
            anchorOrigin={{
                vertical: "top",
                horizontal: "right",
            }}
            sx={{
                display: "flex",
                width: "100%",
                maxWidth: "700px",
                margin: "auto",
                alignItems: "center",
                justifyContent: "center",
                "& .MuiBadge-badge": {
                    right: 8,
                    top: 8,
                    fontSize: "0.875rem",
                    fontWeight: "bold",
                    display: pendingApprovalsCount > 0 ? "flex" : "none",
                },
            }}
        >
            <PlayerContainer onClick={handlePlayerClick}>
                {isLoading ? (
                    <>
                        <PlayerContent>
                            <InfoSection>
                                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                    <Skeleton variant="circular" width={20} height={20} />
                                    <Skeleton variant="text" width={80} height={20} />
                                </Box>
                                <Skeleton variant="text" width="40%" height={20} sx={{ flex: 1, mx: 2 }} />
                                <Skeleton variant="text" width={60} height={20} />
                            </InfoSection>
                        </PlayerContent>
                        <Skeleton 
                            variant="rectangular" 
                            width="100%" 
                            height={4} 
                            sx={{ 
                                position: "absolute",
                                bottom: 0,
                                left: 0,
                                right: 0,
                                borderRadius: 0,
                            }} 
                        />
                    </>
                ) : (
                    <>
                        <PlayerContent>
                            <InfoSection>
                                <StatusBadge status={swarmStatus}>
                                    <IconCommon name={statusIcon} size={20} />
                                    <span>{statusText}</span>
                                </StatusBadge>
                                
                                {swarmConfig?.goal && (
                                    <Box
                                        ref={goalRef}
                                        className="marquee-container"
                                        sx={{
                                            flex: 1,
                                            color: "text.secondary",
                                        }}
                                    >
                                        <Typography
                                            variant="caption"
                                            component="span"
                                            className={`marquee-content ${isOverflowing ? "scrolling" : ""}`}
                                            data-text={swarmConfig.goal}
                                        >
                                            {swarmConfig.goal}
                                        </Typography>
                                    </Box>
                                )}
                                
                                <CreditDisplay>
                                    {formattedCredits}
                                </CreditDisplay>
                            </InfoSection>

                            <ControlSection>
                                {/* Pause/Resume button */}
                                {(swarmStatus === ExecutionStates.RUNNING || swarmStatus === ExecutionStates.IDLE) && (
                                    <IconButton
                                        variant="transparent"
                                        size="sm"
                                        onClick={(e) => handleControlClick(e, onPause)}
                                        title={t("PauseSwarm")}
                                    >
                                        <IconCommon name="Pause" size={20} />
                                    </IconButton>
                                )}
                                {swarmStatus === ExecutionStates.PAUSED && (
                                    <IconButton
                                        variant="transparent"
                                        size="sm"
                                        onClick={(e) => handleControlClick(e, onResume)}
                                        title={t("ResumeSwarm")}
                                    >
                                        <IconCommon name="Play" size={20} />
                                    </IconButton>
                                )}

                                {/* Stop button */}
                                {[ExecutionStates.RUNNING, ExecutionStates.IDLE, ExecutionStates.PAUSED].includes(swarmStatus) && (
                                    <IconButton
                                        variant="transparent"
                                        size="sm"
                                        onClick={(e) => handleControlClick(e, onStop)}
                                        title={t("StopSwarm")}
                                    >
                                        <IconCommon name="Stop" size={20} />
                                    </IconButton>
                                )}

                            </ControlSection>
                        </PlayerContent>

                        {/* Progress bar */}
                        {swarmConfig?.subtasks && swarmConfig.subtasks.length > 0 && (
                            <LinearProgress
                                variant="determinate"
                                value={progress}
                                sx={{
                                    position: "absolute",
                                    bottom: 0,
                                    left: 0,
                                    right: 0,
                                    height: 4,
                                    borderRadius: 0,
                                    bgcolor: "action.hover",
                                    "& .MuiLinearProgress-bar": {
                                        bgcolor: swarmStatus === ExecutionStates.FAILED 
                                            ? "error.main" 
                                            : progress === 100
                                            ? "success.main"
                                            : "secondary.main",
                                    },
                                }}
                            />
                        )}
                    </>
                )}
            </PlayerContainer>
        </Badge>
    );
}
