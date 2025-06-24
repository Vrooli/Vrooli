import { Alert, Box, Button, Chip, Divider, List, ListItem, ListItemButton, ListItemIcon, ListItemText, Paper, Skeleton, Typography } from "@mui/material";
import { IconButton } from "../buttons/IconButton.js";
import { styled, useTheme } from "@mui/material/styles";
import { type ChatConfigObject, ExecutionStates, PendingToolCallStatus, getObjectUrl } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useSocketSwarm } from "../../hooks/useSocketSwarm.js";
import { useSwarmData } from "../../hooks/useSwarmData.js";
import { IconCommon } from "../../icons/Icons.js";
import { Link } from "../../route/router.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { PageTabs } from "../PageTabs/PageTabs.js";

const PanelContainer = styled(Box)(({ theme }) => ({
    height: "100%",
    display: "flex",
    flexDirection: "column",
    backgroundColor: theme.palette.background.paper,
}));

const PanelHeader = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
    borderBottom: `1px solid ${theme.palette.divider}`,
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
}));

const TabPanel = styled(Box)(({ theme }) => ({
    padding: theme.spacing(2),
    overflow: "auto",
    flex: 1,
}));

const TaskItem = styled(ListItem)<{ status: string }>(({ theme, status }) => {
    const getStatusColor = () => {
        switch (status) {
            case "done":
                return theme.palette.success.light;
            case "in_progress":
                return theme.palette.info.light;
            case "failed":
                return theme.palette.error.light;
            case "blocked":
                return theme.palette.warning.light;
            case "canceled":
                return theme.palette.text.disabled;
            default:
                return "transparent";
        }
    };

    return {
        borderLeft: `4px solid ${getStatusColor()}`,
        marginBottom: theme.spacing(1),
        "&:hover": {
            backgroundColor: theme.palette.action.hover,
        },
    };
});

const ResourceCard = styled(Paper)(({ theme }) => ({
    padding: theme.spacing(1.5),
    marginBottom: theme.spacing(1),
    cursor: "pointer",
    transition: "all 0.2s ease",
    "&:hover": {
        backgroundColor: theme.palette.action.hover,
        transform: "translateY(-2px)",
        boxShadow: theme.shadows[4],
    },
}));

const ApprovalCard = styled(Paper)<{ urgent?: boolean }>(({ theme, urgent }) => ({
    padding: theme.spacing(2),
    marginBottom: theme.spacing(1),
    border: urgent ? `2px solid ${theme.palette.warning.main}` : undefined,
    backgroundColor: urgent ? theme.palette.warning.light + "10" : undefined,
}));

const StatBox = styled(Box)(({ theme }) => ({
    padding: theme.spacing(1),
    textAlign: "center",
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.action.hover,
    minWidth: "120px",
    flexShrink: 0,
}));

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function CustomTabPanel(props: TabPanelProps & { sx?: any }) {
    const { children, value, index, sx, ...other } = props;
    return (
        <TabPanel
            role="tabpanel"
            hidden={value !== index}
            id={`swarm-tabpanel-${index}`}
            aria-labelledby={`swarm-tab-${index}`}
            sx={sx}
            {...other}
        >
            {value === index && children}
        </TabPanel>
    );
}

export interface SwarmDetailPanelProps {
    swarmConfig: ChatConfigObject | null;
    swarmStatus?: string;
    chatId?: string | null;
    onApproveToolCall?: (pendingId: string) => void;
    onRejectToolCall?: (pendingId: string, reason?: string) => void;
    onConfigUpdate?: (config: Partial<ChatConfigObject>) => void;
    onStart?: () => void;
    onPause?: () => void;
    onResume?: () => void;
    onStop?: () => void;
    isLoading?: boolean;
}

/**
 * Detailed panel showing comprehensive swarm information.
 * Displayed in the right drawer when SwarmPlayer is clicked.
 */
export function SwarmDetailPanel({
    swarmConfig: initialSwarmConfig,
    swarmStatus = ExecutionStates.UNINITIALIZED,
    chatId,
    onApproveToolCall,
    onRejectToolCall,
    onConfigUpdate,
    onStart,
    onPause,
    onResume,
    onStop,
    isLoading = false,
}: SwarmDetailPanelProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const session = useContext(SessionContext);
    const languages = useMemo(() => getUserLanguages(session), [session]);
    
    // Helper function to get display info for objects
    const getDisplayInfo = useCallback((object: any) => {
        if (!object) return { title: "", subtitle: "", adornments: [] };
        return getDisplay(object, languages, theme.palette);
    }, [languages, theme.palette]);
    
    // Helper function to get URL for objects
    const getUrl = useCallback((object: any) => {
        if (!object?.id || !object?.__typename) return "";
        try {
            return getObjectUrl(object);
        } catch (e) {
            console.warn("Could not get URL for object:", object, e);
            return "";
        }
    }, []);

    // Local state for swarm config to handle real-time updates
    const [swarmConfig, setSwarmConfig] = useState<ChatConfigObject | null>(initialSwarmConfig);
    const [localSwarmStatus, setLocalSwarmStatus] = useState(swarmStatus);

    // Update local state when props change
    useEffect(() => {
        setSwarmConfig(initialSwarmConfig);
    }, [initialSwarmConfig]);

    useEffect(() => {
        setLocalSwarmStatus(swarmStatus);
    }, [swarmStatus]);

    // Load referenced data (bots, resources, teams)
    const { bots, resources, team, loading, errors, getBotById, getResourceById } = useSwarmData(swarmConfig);
    
    // Debug logging
    useEffect(() => {
        console.log("SwarmDetailPanel - swarmConfig:", swarmConfig);
        console.log("SwarmDetailPanel - teamId from config:", swarmConfig?.teamId);
        console.log("SwarmDetailPanel - team loaded:", team);
        console.log("SwarmDetailPanel - loading states:", { 
            isLoading: loading.team, 
            team: loading.team,
            errors: errors.team, 
        });
    }, [team, swarmConfig, loading, errors]);

    // Handle socket events for real-time updates
    useSocketSwarm({
        chatId,
        onConfigUpdate: useCallback((configUpdate) => {
            setSwarmConfig(prev => prev ? { ...prev, ...configUpdate } : null);
            onConfigUpdate?.(configUpdate);
        }, [onConfigUpdate]),
        onStateUpdate: useCallback((stateUpdate) => {
            setLocalSwarmStatus(stateUpdate.state);
        }, []),
        onResourceUpdate: useCallback((resourceUpdate) => {
            // Update stats with resource consumption
            setSwarmConfig(prev => {
                if (!prev) return null;
                return {
                    ...prev,
                    stats: {
                        ...prev.stats,
                        totalCredits: resourceUpdate.resources.consumed.toString(),
                    },
                };
            });
        }, []),
        onTeamUpdate: useCallback((teamUpdate) => {
            setSwarmConfig(prev => {
                if (!prev) return null;
                return {
                    ...prev,
                    teamId: teamUpdate.teamId || prev.teamId,
                    swarmLeader: teamUpdate.swarmLeader || prev.swarmLeader,
                    subtaskLeaders: teamUpdate.subtaskLeaders || prev.subtaskLeaders,
                };
            });
        }, []),
    });

    // Configure tabs for PageTabs component
    const tabOptions = useMemo(() => [
        { key: "tasks", label: t("Tasks"), index: 0, iconInfo: { name: "Reminder" as const, type: "Common" as const } },
        { key: "agents", label: t("Agents"), index: 1, iconInfo: { name: "Bot" as const, type: "Common" as const } },
        { key: "resources", label: t("Resources"), index: 2, iconInfo: { name: "File" as const, type: "Common" as const } },
        { key: "approvals", label: t("Approvals"), index: 3, iconInfo: { name: "Lock" as const, type: "Common" as const } },
        { key: "history", label: t("History"), index: 4, iconInfo: { name: "History" as const, type: "Common" as const } },
    ], [t]);

    const [currTab, setCurrTab] = useState(tabOptions[0]);

    const handleTabChange = useCallback((event: React.SyntheticEvent, tab: typeof tabOptions[0]) => {
        setCurrTab(tab);
    }, []);

    const handleClose = useCallback(() => {
        PubSub.get().publish("menu", {
            id: ELEMENT_IDS.RightDrawer,
            isOpen: false,
        });
    }, []);

    // Calculate statistics
    const stats = useMemo(() => {
        if (!swarmConfig) return null;

        const totalTasks = swarmConfig.subtasks?.length || 0;
        const completedTasks = swarmConfig.subtasks?.filter(t => t.status === "done").length || 0;
        const failedTasks = swarmConfig.subtasks?.filter(t => t.status === "failed").length || 0;
        const pendingApprovals = swarmConfig.pendingToolCalls?.filter(
            tc => tc.status === PendingToolCallStatus.PENDING_APPROVAL,
        ).length || 0;

        const elapsedTime = swarmConfig.stats.startedAt
            ? Date.now() - swarmConfig.stats.startedAt
            : 0;

        return {
            totalTasks,
            completedTasks,
            failedTasks,
            pendingApprovals,
            elapsedTime,
            toolCalls: swarmConfig.stats.totalToolCalls || 0,
            credits: BigInt(swarmConfig.stats.totalCredits || "0"),
        };
    }, [swarmConfig]);

    // Format elapsed time
    const formatElapsedTime = useCallback((ms: number) => {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);

        if (hours > 0) {
            return t("SwarmElapsedHours", { hours, minutes: minutes % 60 });
        } else if (minutes > 0) {
            return t("SwarmElapsedMinutes", { minutes, seconds: seconds % 60 });
        } else {
            return t("SwarmElapsedSeconds", { seconds });
        }
    }, [t]);

    // Get status chip color and label
    const getStatusDisplay = useCallback((status: string) => {
        switch (status) {
            case ExecutionStates.RUNNING:
                return { label: t("Running"), color: "success" as const };
            case ExecutionStates.PAUSED:
                return { label: t("Paused"), color: "warning" as const };
            case ExecutionStates.IDLE:
                return { label: t("Idle"), color: "info" as const };
            case ExecutionStates.STOPPED:
                return { label: t("Stopped"), color: "default" as const };
            case ExecutionStates.FAILED:
                return { label: t("Failed"), color: "error" as const };
            case ExecutionStates.STARTING:
                return { label: t("Starting"), color: "info" as const };
            case ExecutionStates.TERMINATED:
                return { label: t("Terminated"), color: "error" as const };
            default:
                return { label: t("Inactive"), color: "default" as const };
        }
    }, [t]);

    // Get task icon based on status
    const getTaskIcon = useCallback((status: string) => {
        switch (status) {
            case "done":
                return "Complete";
            case "in_progress":
                return "Refresh";
            case "failed":
                return "Cancel";
            case "blocked":
                return "Lock";
            case "canceled":
                return "Cancel";
            default:
                return "Info";
        }
    }, []);

    // Get resource icon based on kind
    const getResourceIcon = useCallback((kind: string) => {
        switch (kind) {
            case "File":
                return "File";
            case "URL":
                return "Link";
            case "Image":
                return "Image";
            case "Vector":
                return "Grid";
            case "Note":
                return "Note";
            default:
                return "File";
        }
    }, []);

    if (!swarmConfig && !isLoading) {
        return (
            <PanelContainer>
                <PanelHeader>
                    <Typography variant="h6">{t("SwarmDetails")}</Typography>
                    <IconButton variant="transparent" size="md" onClick={handleClose}>
                        <IconCommon name="Close" />
                    </IconButton>
                </PanelHeader>
                <Box p={3} textAlign="center">
                    <Typography color="text.secondary">
                        {t("NoSwarmActive")}
                    </Typography>
                </Box>
            </PanelContainer>
        );
    }

    // Show full loading skeleton only when explicitly loading or when we don't have swarm config
    // Individual sections will handle their own loading states for referenced data
    if (isLoading) {
        return (
            <PanelContainer>
                <PanelHeader>
                    <Box>
                        <Skeleton variant="text" width={140} height={28} />
                        <Skeleton variant="text" width={200} height={20} sx={{ mt: 0.5 }} />
                    </Box>
                    <IconButton variant="transparent" size="md" onClick={handleClose}>
                        <IconCommon name="Close" />
                    </IconButton>
                </PanelHeader>

                {/* Statistics Overview Skeleton */}
                <Box sx={{ p: 2, borderBottom: 1, borderColor: "divider" }}>
                    <Box display="grid" gridTemplateColumns="repeat(3, 1fr)" gap={1}>
                        {[1, 2, 3].map((i) => (
                            <Box
                                key={i}
                                sx={{
                                    p: 1,
                                    textAlign: "center",
                                    borderRadius: 1,
                                    backgroundColor: "action.hover",
                                    minWidth: "120px",
                                }}
                            >
                                <Skeleton variant="text" width={80} height={16} sx={{ mx: "auto" }} />
                                <Skeleton variant="text" width={60} height={24} sx={{ mx: "auto", mt: 0.5 }} />
                            </Box>
                        ))}
                    </Box>
                </Box>

                {/* Tabs Skeleton */}
                <Box sx={{ borderBottom: 1, borderColor: "divider", px: 2 }}>
                    <Box sx={{ display: "flex", gap: 3, py: 1 }}>
                        {[1, 2, 3, 4, 5].map((i) => (
                            <Skeleton key={i} variant="text" width={80} height={40} />
                        ))}
                    </Box>
                </Box>

                {/* Content Area Skeleton */}
                <TabPanel>
                    <List>
                        {[1, 2, 3, 4].map((i) => (
                            <ListItem key={i} sx={{ mb: 1 }}>
                                <ListItemIcon>
                                    <Skeleton variant="circular" width={24} height={24} />
                                </ListItemIcon>
                                <ListItemText
                                    primary={<Skeleton variant="text" width="60%" height={20} />}
                                    secondary={<Skeleton variant="text" width="40%" height={16} />}
                                />
                            </ListItem>
                        ))}
                    </List>
                </TabPanel>
            </PanelContainer>
        );
    }

    return (
        <PanelContainer>
            <PanelHeader>
                <Box sx={{ flex: 1 }}>
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 0.5 }}>
                        <Typography variant="h6">{t("SwarmDetails")}</Typography>
                        <Chip 
                            label={getStatusDisplay(localSwarmStatus).label}
                            color={getStatusDisplay(localSwarmStatus).color}
                            size="small"
                            variant="filled"
                        />
                    </Box>
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                        {swarmConfig.goal || t("NoGoalSet")}
                    </Typography>
                </Box>
                
                {/* Control buttons */}
                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                    {/* Start button - show when stopped/inactive */}
                    {[ExecutionStates.STOPPED, ExecutionStates.UNINITIALIZED, ExecutionStates.TERMINATED].includes(localSwarmStatus) && onStart && (
                        <IconButton
                            variant="transparent"
                            size="sm"
                            onClick={() => {
                                setLocalSwarmStatus(ExecutionStates.STARTING);
                                onStart();
                            }}
                            title={t("StartSwarm")}
                            color="success"
                        >
                            <IconCommon name="Play" />
                        </IconButton>
                    )}
                    
                    {/* Pause button - show when running/idle */}
                    {[ExecutionStates.RUNNING, ExecutionStates.IDLE].includes(localSwarmStatus) && onPause && (
                        <IconButton
                            variant="transparent"
                            size="sm"
                            onClick={() => {
                                setLocalSwarmStatus(ExecutionStates.PAUSED);
                                onPause();
                            }}
                            title={t("PauseSwarm")}
                            color="warning"
                        >
                            <IconCommon name="Pause" />
                        </IconButton>
                    )}
                    
                    {/* Resume button - show when paused */}
                    {localSwarmStatus === ExecutionStates.PAUSED && onResume && (
                        <IconButton
                            variant="transparent"
                            size="sm"
                            onClick={() => {
                                setLocalSwarmStatus(ExecutionStates.RUNNING);
                                onResume();
                            }}
                            title={t("ResumeSwarm")}
                            color="success"
                        >
                            <IconCommon name="Play" />
                        </IconButton>
                    )}
                    
                    {/* Stop button - show when running/paused/idle */}
                    {[ExecutionStates.RUNNING, ExecutionStates.PAUSED, ExecutionStates.IDLE].includes(localSwarmStatus) && onStop && (
                        <IconButton
                            variant="transparent"
                            size="sm"
                            onClick={() => {
                                setLocalSwarmStatus(ExecutionStates.STOPPED);
                                onStop();
                            }}
                            title={t("StopSwarm")}
                            color="error"
                        >
                            <IconCommon name="Stop" />
                        </IconButton>
                    )}
                    
                    <IconButton variant="transparent" size="md" onClick={handleClose}>
                        <IconCommon name="Close" />
                    </IconButton>
                </Box>
            </PanelHeader>

            {/* Statistics Overview */}
            <Box sx={{
                p: 2,
                borderBottom: 1,
                borderColor: "divider",
                overflowX: "auto",
                "&::-webkit-scrollbar": {
                    height: 6,
                },
                "&::-webkit-scrollbar-track": {
                    backgroundColor: "transparent",
                },
                "&::-webkit-scrollbar-thumb": {
                    backgroundColor: "divider",
                    borderRadius: 3,
                },
            }}>
                <Box
                    display="grid"
                    gridTemplateColumns="repeat(3, 1fr)"
                    gap={1}
                    sx={{ minWidth: "fit-content" }}>
                    <StatBox>
                        <Typography variant="caption" color="text.secondary">
                            {t("TasksComplete")}
                        </Typography>
                        <Typography variant="h6">
                            {stats?.completedTasks}/{stats?.totalTasks}
                        </Typography>
                    </StatBox>
                    <StatBox>
                        <Typography variant="caption" color="text.secondary">
                            {t("CreditsUsed")}
                        </Typography>
                        <Typography variant="h6">
                            {(() => {
                                if (!stats?.credits) return "$0.00";
                                try {
                                    const usd = Number(stats.credits / BigInt(1e8)) / 100;
                                    return `$${usd.toFixed(2)}`;
                                } catch (error) {
                                    console.error("Error formatting credits:", error);
                                    return "$0.00";
                                }
                            })()}
                        </Typography>
                    </StatBox>
                    <StatBox>
                        <Typography variant="caption" color="text.secondary">
                            {t("ElapsedTime")}
                        </Typography>
                        <Typography variant="h6">
                            {formatElapsedTime(stats?.elapsedTime || 0)}
                        </Typography>
                    </StatBox>
                </Box>

                {stats && stats.pendingApprovals > 0 && (
                    <Alert 
                        severity="warning" 
                        sx={{ 
                            mt: 2,
                            cursor: "pointer",
                            "&:hover": {
                                bgcolor: "warning.light",
                                "& .MuiAlert-message": {
                                    textDecoration: "underline",
                                },
                            },
                        }}
                        onClick={() => setCurrTab(tabOptions[3])} // Navigate to Approvals tab (index 3)
                    >
                        {t("PendingApprovals", { count: stats.pendingApprovals })}
                    </Alert>
                )}
            </Box>

            <Box sx={{ borderBottom: 1, borderColor: "divider", px: 2 }}>
                <PageTabs
                    ariaLabel="swarm-detail-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabOptions}
                    fullWidth={true}
                />
            </Box>

            {/* Tasks Tab */}
            <CustomTabPanel value={currTab.index} index={0}>
                <List>
                    {swarmConfig.subtasks?.map((task) => (
                        <TaskItem key={task.id} status={task.status}>
                            <ListItemText
                                primary={
                                    <Box display="flex" alignItems="center" gap={1}>
                                        <Typography variant="body1">
                                            {task.description}
                                        </Typography>
                                    </Box>
                                }
                                secondary={
                                    <Box>
                                        {task.assignee_bot_id && (
                                            <Typography variant="caption" color="text.secondary">
                                                {t("AssignedTo")}: {
                                                    loading.bots ? (
                                                        <Skeleton component="span" variant="text" width={80} />
                                                    ) : (
                                                        getBotById(task.assignee_bot_id)?.name || task.assignee_bot_id
                                                    )
                                                }
                                            </Typography>
                                        )}
                                        {task.depends_on && task.depends_on.length > 0 && (
                                            <Typography variant="caption" color="text.secondary" sx={{ ml: 2 }}>
                                                {t("DependsOn")}: {task.depends_on.join(", ")}
                                            </Typography>
                                        )}
                                    </Box>
                                }
                            />
                        </TaskItem>
                    ))}
                    {(!swarmConfig.subtasks || swarmConfig.subtasks.length === 0) && (
                        <Box textAlign="center" py={3}>
                            <Typography color="text.secondary">
                                {t("NoTasksYet")}
                            </Typography>
                        </Box>
                    )}
                </List>
            </CustomTabPanel>

            {/* Agents Tab */}
            <CustomTabPanel value={currTab.index} index={1} sx={{ p: 0 }}>
                {console.log("SwarmDetailPanel - Agents tab render:", { 
                    currTabIndex: currTab.index, 
                    teamId: swarmConfig?.teamId,
                    team,
                    teamConfig: team?.config, 
                })}
                <List>
                    {swarmConfig.swarmLeader && (() => {
                        const bot = getBotById(swarmConfig.swarmLeader);
                        const display = bot ? getDisplayInfo(bot) : null;
                        const url = bot ? getUrl(bot) : "";
                        
                        return (
                            <ListItem disablePadding>
                                <ListItemButton 
                                    component={url ? Link : "div"}
                                    to={url || undefined}
                                    disabled={!url}
                                >
                                    <ListItemIcon>
                                        <IconCommon name="Award" fill="warning.main" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary={t("SwarmLeader")}
                                        secondary={
                                            loading.bots ? (
                                                <Skeleton variant="text" width={120} />
                                            ) : display ? (
                                                <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                                                    {display.title}
                                                    {display.adornments.map(adornment => 
                                                        <Box key={adornment.key}>{adornment.Adornment}</Box>,
                                                    )}
                                                    {display.subtitle && (
                                                        <Typography variant="caption" color="text.disabled" sx={{ ml: 1 }}>
                                                            {display.subtitle}
                                                        </Typography>
                                                    )}
                                                </Box>
                                            ) : (
                                                swarmConfig.swarmLeader
                                            )
                                        }
                                    />
                                </ListItemButton>
                            </ListItem>
                        );
                    })()}
                    {swarmConfig.teamId && (() => {
                        const display = team ? getDisplayInfo(team) : null;
                        const url = team ? getUrl(team) : "";
                        
                        return (
                            <ListItem disablePadding>
                                <ListItemButton 
                                    component={url ? Link : "div"}
                                    to={url || undefined}
                                    disabled={!url}
                                >
                                    <ListItemIcon>
                                        <IconCommon name="Team" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary={t("Team")}
                                        secondary={
                                            loading.team ? (
                                                <Skeleton variant="text" width={100} />
                                            ) : display ? (
                                                <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                                                    {display.title}
                                                    {display.adornments.map(adornment => 
                                                        <Box key={adornment.key}>{adornment.Adornment}</Box>,
                                                    )}
                                                    {display.subtitle && (
                                                        <Typography variant="caption" color="text.disabled" sx={{ ml: 1 }}>
                                                            {display.subtitle}
                                                        </Typography>
                                                    )}
                                                </Box>
                                            ) : (
                                                swarmConfig.teamId
                                            )
                                        }
                                    />
                                </ListItemButton>
                            </ListItem>
                        );
                    })()}
                    <Divider sx={{ my: 1 }} />
                    <Typography variant="subtitle2" sx={{ px: 2, py: 1 }}>
                        {t("SubtaskLeaders")}
                    </Typography>
                    {swarmConfig.subtaskLeaders && Object.entries(swarmConfig.subtaskLeaders).map(([taskId, botId]) => {
                        const bot = getBotById(botId);
                        const display = bot ? getDisplayInfo(bot) : null;
                        const url = bot ? getUrl(bot) : "";
                        
                        return (
                            <ListItem key={taskId} disablePadding>
                                <ListItemButton 
                                    component={url ? Link : "div"}
                                    to={url || undefined}
                                    disabled={!url}
                                >
                                    <ListItemIcon>
                                        <IconCommon name="Bot" />
                                    </ListItemIcon>
                                    <ListItemText
                                        primary={taskId}
                                        secondary={
                                            loading.bots ? (
                                                <Skeleton variant="text" width={100} />
                                            ) : display ? (
                                                <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                                                    {display.title}
                                                    {display.adornments.map(adornment => 
                                                        <Box key={adornment.key}>{adornment.Adornment}</Box>,
                                                    )}
                                                    {display.subtitle && (
                                                        <Typography variant="caption" color="text.disabled" sx={{ ml: 1 }}>
                                                            {display.subtitle}
                                                        </Typography>
                                                    )}
                                                </Box>
                                            ) : (
                                                botId
                                            )
                                        }
                                    />
                                </ListItemButton>
                            </ListItem>
                        );
                    })}
                    
                    {/* MOISE+ Organization Structure */}
                    {console.log("SwarmDetailPanel - MOISE+ check:", { 
                        hasTeam: !!team,
                        hasConfig: !!team?.config,
                        teamConfig: team?.config,
                        configType: typeof team?.config,
                        teamLoading: loading.team,
                    })}
                    {loading.team && swarmConfig.teamId ? (
                        <>
                            <Divider sx={{ my: 1 }} />
                            <Typography variant="subtitle2" sx={{ px: 2, py: 1 }}>
                                <Skeleton variant="text" width={150} />
                            </Typography>
                            <ListItem>
                                <Box sx={{ width: "100%", p: 2 }}>
                                    <Skeleton variant="rectangular" width="100%" height={120} />
                                </Box>
                            </ListItem>
                        </>
                    ) : team?.config && (() => {
                        try {
                            const teamConfig = typeof team.config === "string" ? JSON.parse(team.config) : team.config;
                            console.log("SwarmDetailPanel - parsed teamConfig:", teamConfig);
                            console.log("SwarmDetailPanel - has structure?:", !!teamConfig.structure);
                            console.log("SwarmDetailPanel - has content?:", !!teamConfig.structure?.content);
                            if (teamConfig.structure && teamConfig.structure.content) {
                                console.log("SwarmDetailPanel - rendering MOISE+ structure");
                                return (
                                    <>
                                        <Divider sx={{ my: 1 }} />
                                        <Typography variant="subtitle2" sx={{ px: 2, py: 1 }}>
                                            {t("OrganizationStructure")} ({teamConfig.structure.type || "MOISE+"})
                                        </Typography>
                                        <ListItem>
                                            <Box sx={{ 
                                                width: "100%",
                                                p: 2,
                                                bgcolor: "background.default",
                                                borderRadius: 1,
                                                fontFamily: "monospace",
                                                fontSize: "0.875rem",
                                                overflow: "auto",
                                                maxHeight: 200,
                                                whiteSpace: "pre-wrap",
                                                wordBreak: "break-word",
                                            }}>
                                                <pre style={{ margin: 0 }}>
                                                    {teamConfig.structure.content}
                                                </pre>
                                            </Box>
                                        </ListItem>
                                    </>
                                );
                            }
                        } catch (e) {
                            console.error("SwarmDetailPanel - error parsing team config:", e);
                        }
                        return null;
                    })()}
                </List>
            </CustomTabPanel>

            {/* Resources Tab */}
            <CustomTabPanel value={currTab.index} index={2}>
                {swarmConfig.resources?.map((resource) => (
                    <ResourceCard key={resource.id} elevation={1}>
                        <Box display="flex" alignItems="center" gap={2}>
                            <IconCommon name={getResourceIcon(resource.kind)} size={32} />
                            <Box flex={1}>
                                <Typography variant="body1" fontWeight="medium">
                                    {getResourceById(resource.id)?.name || resource.id}
                                </Typography>
                                <Typography variant="caption" color="text.secondary">
                                    {resource.kind} • {resource.mime || t("Unknown")}
                                </Typography>
                                {resource.meta && (
                                    <Typography variant="caption" display="block" sx={{ mt: 0.5 }}>
                                        {Object.entries(resource.meta).map(([key, value]) =>
                                            `${key}: ${value}`,
                                        ).join(" • ")}
                                    </Typography>
                                )}
                            </Box>
                        </Box>
                    </ResourceCard>
                ))}
                {(!swarmConfig.resources || swarmConfig.resources.length === 0) && (
                    <Box textAlign="center" py={3}>
                        <Typography color="text.secondary">
                            {t("NoResourcesCreated")}
                        </Typography>
                    </Box>
                )}
            </CustomTabPanel>

            {/* Approvals Tab */}
            <CustomTabPanel value={currTab.index} index={3}>
                {swarmConfig.pendingToolCalls
                    ?.filter(tc => tc.status === PendingToolCallStatus.PENDING_APPROVAL)
                    .sort((a, b) => {
                        // Sort by urgency - most urgent first
                        // 1. No timeout (manual approval required) goes last
                        if (!a.approvalTimeoutAt && !b.approvalTimeoutAt) return 0;
                        if (!a.approvalTimeoutAt) return 1;
                        if (!b.approvalTimeoutAt) return -1;
                        
                        // 2. Sort by time remaining (least time = most urgent)
                        const timeRemainingA = a.approvalTimeoutAt - Date.now();
                        const timeRemainingB = b.approvalTimeoutAt - Date.now();
                        
                        // If either is already expired, put it first
                        if (timeRemainingA <= 0 && timeRemainingB <= 0) return 0;
                        if (timeRemainingA <= 0) return -1;
                        if (timeRemainingB <= 0) return 1;
                        
                        // Otherwise sort by time remaining (ascending)
                        return timeRemainingA - timeRemainingB;
                    })
                    .map((toolCall) => {
                        const timeRemaining = toolCall.approvalTimeoutAt
                            ? toolCall.approvalTimeoutAt - Date.now()
                            : null;
                        const isUrgent = timeRemaining && timeRemaining < 60000; // Less than 1 minute

                        return (
                            <ApprovalCard key={toolCall.pendingId} urgent={isUrgent}>
                                <Box display="flex" justifyContent="space-between" alignItems="start" mb={1}>
                                    <Box>
                                        <Typography variant="h6">
                                            {toolCall.toolName}
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            {t("RequestedBy")}: {
                                                loading.bots ? (
                                                    <Skeleton component="span" variant="text" width={80} />
                                                ) : (
                                                    getBotById(toolCall.callerBotId)?.name || toolCall.callerBotId
                                                )
                                            }
                                        </Typography>
                                    </Box>
                                    {timeRemaining && timeRemaining > 0 && (
                                        <Chip
                                            label={formatElapsedTime(timeRemaining)}
                                            size="small"
                                            color={isUrgent ? "warning" : "default"}
                                        />
                                    )}
                                </Box>

                                <Box sx={{
                                    p: 1.5,
                                    bgcolor: "background.default",
                                    borderRadius: 1,
                                    mb: 2,
                                    fontFamily: "monospace",
                                    fontSize: "0.875rem",
                                }}>
                                    <pre style={{ margin: 0, whiteSpace: "pre-wrap" }}>
                                        {JSON.stringify(JSON.parse(toolCall.toolArguments), null, 2)}
                                    </pre>
                                </Box>

                                <Box display="flex" gap={1} justifyContent="flex-end">
                                    <Button
                                        variant="outlined"
                                        color="error"
                                        onClick={() => onRejectToolCall?.(toolCall.pendingId)}
                                        startIcon={<IconCommon name="Close" />}
                                        size="small"
                                    >
                                        {t("Deny")}
                                    </Button>
                                    <Button
                                        variant="contained"
                                        color="success"
                                        onClick={() => onApproveToolCall?.(toolCall.pendingId)}
                                        startIcon={<IconCommon name="Complete" />}
                                        size="small"
                                    >
                                        {t("Approve")}
                                    </Button>
                                </Box>
                            </ApprovalCard>
                        );
                    })}
                {(!swarmConfig.pendingToolCalls ||
                    swarmConfig.pendingToolCalls.filter(tc => tc.status === PendingToolCallStatus.PENDING_APPROVAL).length === 0) && (
                        <Box textAlign="center" py={3}>
                            <Typography color="text.secondary">
                                {t("NoPendingApprovals")}
                            </Typography>
                        </Box>
                    )}
            </CustomTabPanel>

            {/* History Tab */}
            <CustomTabPanel value={currTab.index} index={4}>
                <List>
                    {swarmConfig.records?.map((record) => (
                        <ListItem key={record.id}>
                            <ListItemIcon>
                                <IconCommon name="History" />
                            </ListItemIcon>
                            <ListItemText
                                primary={record.routine_name}
                                secondary={
                                    <Box>
                                        <Typography variant="caption" display="block">
                                            {t("CalledBy")}: {
                                                loading.bots ? (
                                                    <Skeleton component="span" variant="text" width={80} />
                                                ) : (
                                                    getBotById(record.caller_bot_id)?.name || record.caller_bot_id
                                                )
                                            }
                                        </Typography>
                                        <Typography variant="caption" display="block">
                                            {new Date(record.created_at).toLocaleString(languages[0])}
                                        </Typography>
                                        {record.output_resource_ids.length > 0 && (
                                            <Typography variant="caption" display="block">
                                                {t("CreatedResources")}: {record.output_resource_ids.length}
                                            </Typography>
                                        )}
                                    </Box>
                                }
                            />
                        </ListItem>
                    ))}
                    {(!swarmConfig.records || swarmConfig.records.length === 0) && (
                        <Box textAlign="center" py={3}>
                            <Typography color="text.secondary">
                                {t("NoExecutionHistory")}
                            </Typography>
                        </Box>
                    )}
                </List>
            </CustomTabPanel>
        </PanelContainer>
    );
}
