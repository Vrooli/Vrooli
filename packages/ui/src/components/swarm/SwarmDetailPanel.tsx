import { Box, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Paper, Typography, Chip, Tooltip, LinearProgress, Alert } from "@mui/material";
import { styled, useTheme } from "@mui/material/styles";
import { ChatConfigObject, ExecutionStates, SwarmSubTask, SwarmResource, ChatToolCallRecord, PendingToolCallStatus, PendingToolCallEntry, getTranslation } from "@vrooli/shared";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { useContext } from "react";
import { SessionContext } from "../../contexts/session.js";
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
}));

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function CustomTabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;
    return (
        <TabPanel
            role="tabpanel"
            hidden={value !== index}
            id={`swarm-tabpanel-${index}`}
            aria-labelledby={`swarm-tab-${index}`}
            {...other}
        >
            {value === index && children}
        </TabPanel>
    );
}

export interface SwarmDetailPanelProps {
    swarmConfig: ChatConfigObject | null;
    swarmStatus?: string;
    onApproveToolCall?: (pendingId: string) => void;
    onRejectToolCall?: (pendingId: string, reason?: string) => void;
}

/**
 * Detailed panel showing comprehensive swarm information.
 * Displayed in the right drawer when SwarmPlayer is clicked.
 */
export function SwarmDetailPanel({
    swarmConfig,
    swarmStatus = ExecutionStates.UNINITIALIZED,
    onApproveToolCall,
    onRejectToolCall,
}: SwarmDetailPanelProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const session = useContext(SessionContext);
    const languages = useMemo(() => getUserLanguages(session), [session]);
    
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
            isOpen: false 
        });
    }, []);

    // Calculate statistics
    const stats = useMemo(() => {
        if (!swarmConfig) return null;

        const totalTasks = swarmConfig.subtasks?.length || 0;
        const completedTasks = swarmConfig.subtasks?.filter(t => t.status === "done").length || 0;
        const failedTasks = swarmConfig.subtasks?.filter(t => t.status === "failed").length || 0;
        const pendingApprovals = swarmConfig.pendingToolCalls?.filter(
            tc => tc.status === PendingToolCallStatus.PENDING_APPROVAL
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

    // Get task icon based on status
    const getTaskIcon = useCallback((status: string) => {
        switch (status) {
            case "done":
                return "CheckCircle";
            case "in_progress":
                return "Loop";
            case "failed":
                return "Cancel";
            case "blocked":
                return "Block";
            case "canceled":
                return "DoNotDisturb";
            default:
                return "RadioButtonUnchecked";
        }
    }, []);

    // Get resource icon based on kind
    const getResourceIcon = useCallback((kind: string) => {
        switch (kind) {
            case "File":
                return "Description";
            case "URL":
                return "Link";
            case "Image":
                return "Image";
            case "Vector":
                return "DataArray";
            case "Note":
                return "StickyNote";
            default:
                return "Folder";
        }
    }, []);

    if (!swarmConfig) {
        return (
            <PanelContainer>
                <PanelHeader>
                    <Typography variant="h6">{t("SwarmDetails")}</Typography>
                    <IconButton onClick={handleClose}>
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

    return (
        <PanelContainer>
            <PanelHeader>
                <Box>
                    <Typography variant="h6">{t("SwarmDetails")}</Typography>
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                        {swarmConfig.goal || t("NoGoalSet")}
                    </Typography>
                </Box>
                <IconButton onClick={handleClose}>
                    <IconCommon name="Close" />
                </IconButton>
            </PanelHeader>

            {/* Statistics Overview */}
            <Box sx={{ p: 2, borderBottom: 1, borderColor: "divider" }}>
                <Box display="grid" gridTemplateColumns="repeat(3, 1fr)" gap={1}>
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
                            {stats?.credits.toLocaleString()}
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
                
                {stats?.pendingApprovals && stats.pendingApprovals > 0 && (
                    <Alert severity="warning" sx={{ mt: 2 }}>
                        {t("PendingApprovals", { count: stats.pendingApprovals })}
                    </Alert>
                )}
            </Box>

            <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                <PageTabs
                    ariaLabel="swarm-detail-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabOptions}
                    fullWidth={false}
                />
            </Box>

            {/* Tasks Tab */}
            <CustomTabPanel value={currTab.index} index={0}>
                <List>
                    {swarmConfig.subtasks?.map((task) => (
                        <TaskItem key={task.id} status={task.status}>
                            <ListItemIcon>
                                <IconCommon name={getTaskIcon(task.status)} />
                            </ListItemIcon>
                            <ListItemText
                                primary={
                                    <Box display="flex" alignItems="center" gap={1}>
                                        <Typography variant="body1">
                                            {task.description}
                                        </Typography>
                                        {task.priority && (
                                            <Chip 
                                                label={task.priority}
                                                size="small"
                                                color={
                                                    task.priority === "high" ? "error" :
                                                    task.priority === "medium" ? "warning" :
                                                    "default"
                                                }
                                                variant="outlined"
                                            />
                                        )}
                                    </Box>
                                }
                                secondary={
                                    <Box>
                                        {task.assignee_bot_id && (
                                            <Typography variant="caption" color="text.secondary">
                                                {t("AssignedTo")}: {task.assignee_bot_id}
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
            <CustomTabPanel value={currTab.index} index={1}>
                <List>
                    {swarmConfig.swarmLeader && (
                        <ListItem>
                            <ListItemIcon>
                                <IconCommon name="Star" color="warning" />
                            </ListItemIcon>
                            <ListItemText
                                primary={t("SwarmLeader")}
                                secondary={swarmConfig.swarmLeader}
                            />
                        </ListItem>
                    )}
                    {swarmConfig.teamId && (
                        <ListItem>
                            <ListItemIcon>
                                <IconCommon name="Group" />
                            </ListItemIcon>
                            <ListItemText
                                primary={t("Team")}
                                secondary={swarmConfig.teamId}
                            />
                        </ListItem>
                    )}
                    <Divider sx={{ my: 1 }} />
                    <Typography variant="subtitle2" sx={{ px: 2, py: 1 }}>
                        {t("SubtaskLeaders")}
                    </Typography>
                    {swarmConfig.subtaskLeaders && Object.entries(swarmConfig.subtaskLeaders).map(([taskId, botId]) => (
                        <ListItem key={taskId}>
                            <ListItemIcon>
                                <IconCommon name="SmartToy" />
                            </ListItemIcon>
                            <ListItemText
                                primary={taskId}
                                secondary={botId}
                            />
                        </ListItem>
                    ))}
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
                                    {resource.id}
                                </Typography>
                                <Typography variant="caption" color="text.secondary">
                                    {resource.kind} • {resource.mime || t("Unknown")}
                                </Typography>
                                {resource.meta && (
                                    <Typography variant="caption" display="block" sx={{ mt: 0.5 }}>
                                        {Object.entries(resource.meta).map(([key, value]) => 
                                            `${key}: ${value}`
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
                                            {t("RequestedBy")}: {toolCall.callerBotId}
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
                                    <IconButton
                                        color="error"
                                        onClick={() => onRejectToolCall?.(toolCall.pendingId)}
                                        title={t("Reject")}
                                    >
                                        <IconCommon name="Close" />
                                    </IconButton>
                                    <IconButton
                                        color="success"
                                        onClick={() => onApproveToolCall?.(toolCall.pendingId)}
                                        title={t("Approve")}
                                    >
                                        <IconCommon name="Check" />
                                    </IconButton>
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
                                            {t("CalledBy")}: {record.caller_bot_id}
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