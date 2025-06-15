import { Box, IconButton, LinearProgress, Skeleton, Typography } from "@mui/material";
import { styled, useTheme } from "@mui/material/styles";
import { ChatConfigObject, ExecutionStates, noop } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import "./SwarmPlayer.css";

const PlayerContainer = styled(Box)(({ theme }) => ({
    position: "relative",
    backgroundColor: theme.palette.mode === "dark" ? theme.palette.grey[900] : theme.palette.grey[100],
    borderRadius: theme.shape.borderRadius,
    padding: theme.spacing(1, 2),
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

    // Format credits for display
    const formattedCredits = useMemo(() => {
        if (!swarmConfig?.stats?.totalCredits) return "0";
        const credits = BigInt(swarmConfig.stats.totalCredits);
        if (credits < 1000n) return credits.toString();
        if (credits < 1000000n) return `${(Number(credits) / 1000).toFixed(1)}K`;
        return `${(Number(credits) / 1000000).toFixed(1)}M`;
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
                return "PlayCircle";
            case ExecutionStates.PAUSED:
                return "PauseCircle";
            case ExecutionStates.FAILED:
                return "ErrorOutline";
            case ExecutionStates.STOPPED:
            case ExecutionStates.TERMINATED:
                return "StopCircle";
            default:
                return "CircleOutlined";
        }
    }, [swarmStatus]);

    const handlePlayerClick = useCallback(() => {
        // Open the detail panel with swarm data
        PubSub.get().publish("menu", { 
            id: ELEMENT_IDS.RightDrawer, 
            isOpen: true,
            data: { 
                view: "swarmDetail",
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
            }
        });
    }, [swarmConfig, swarmStatus]);

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
        <PlayerContainer onClick={handlePlayerClick}>
                {isLoading ? (
                    <Box>
                        <Skeleton variant="text" width="100%" height={32} />
                        <Skeleton variant="rectangular" width="100%" height={4} sx={{ mt: 1 }} />
                    </Box>
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
                                            className={`marquee-content ${isOverflowing ? 'scrolling' : ''}`}
                                            data-text={swarmConfig.goal}
                                        >
                                            {swarmConfig.goal}
                                        </Typography>
                                    </Box>
                                )}
                                
                                <CreditDisplay>
                                    {formattedCredits} {t("Credits")}
                                </CreditDisplay>
                            </InfoSection>

                            <ControlSection>
                                {/* Pause/Resume button */}
                                {(swarmStatus === ExecutionStates.RUNNING || swarmStatus === ExecutionStates.IDLE) && (
                                    <IconButton
                                        size="small"
                                        onClick={(e) => handleControlClick(e, onPause)}
                                        title={t("PauseSwarm")}
                                    >
                                        <IconCommon name="Pause" size={20} />
                                    </IconButton>
                                )}
                                {swarmStatus === ExecutionStates.PAUSED && (
                                    <IconButton
                                        size="small"
                                        onClick={(e) => handleControlClick(e, onResume)}
                                        title={t("ResumeSwarm")}
                                    >
                                        <IconCommon name="Play" size={20} />
                                    </IconButton>
                                )}

                                {/* Stop button */}
                                {[ExecutionStates.RUNNING, ExecutionStates.IDLE, ExecutionStates.PAUSED].includes(swarmStatus) && (
                                    <IconButton
                                        size="small"
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
                                    mt: 1,
                                    height: 4,
                                    borderRadius: 2,
                                    bgcolor: "action.hover",
                                    "& .MuiLinearProgress-bar": {
                                        bgcolor: swarmStatus === ExecutionStates.FAILED 
                                            ? "error.main" 
                                            : "primary.main",
                                    },
                                }}
                            />
                        )}
                    </>
                )}
            </PlayerContainer>
    );
}