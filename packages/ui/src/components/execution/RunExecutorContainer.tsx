import { Box, Collapse, IconButton, Typography, Skeleton } from "@mui/material";
import { ChatMessageRunConfig } from "@vrooli/shared";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { RoutineExecutor } from "./RoutineExecutor.js";
import { useRunStates } from "../../hooks/useRunStates.js";

interface RunExecutorContainerProps {
    runs: ChatMessageRunConfig[];
    chatWidth: number;
    messageId: string;
    isLoading?: boolean;
}

export function RunExecutorContainer({
    runs,
    chatWidth,
    messageId,
    isLoading = false,
}: RunExecutorContainerProps) {
    const { t } = useTranslation();
    const {
        collapsedStates,
        toggleCollapsed,
        removeRun,
        allCollapsed,
        toggleAllCollapsed,
    } = useRunStates(messageId, runs);

    const [showAllRuns, setShowAllRuns] = useState(false);

    // Show only first 3 runs by default, with option to expand
    const visibleRuns = useMemo(() => {
        if (showAllRuns || runs.length <= 3) {
            return runs;
        }
        return runs.slice(0, 3);
    }, [runs, showAllRuns]);

    const hiddenRunsCount = runs.length - visibleRuns.length;

    const handleToggleCollapse = useCallback((runId: string) => {
        toggleCollapsed(runId);
    }, [toggleCollapsed]);

    const handleRemoveRun = useCallback((runId: string) => {
        removeRun(runId);
    }, [removeRun]);


    if (runs.length === 0 && !isLoading) {
        return null;
    }

    return (
        <Box sx={{ mt: 1.5, maxWidth: `${chatWidth}px` }}>
            {isLoading ? (
                <>
                    {/* Header skeleton */}
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "flex-end",
                            mb: 1,
                            px: 1,
                        }}
                    >
                        <Skeleton variant="circular" width={32} height={32} />
                    </Box>

                    {/* Loading executor skeleton */}
                    <Box 
                        sx={{ 
                            bgcolor: "background.paper",
                            borderRadius: 3,
                            boxShadow: 1,
                            overflow: "hidden",
                            p: 2,
                        }}
                    >
                        {/* Header area */}
                        <Box sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 2 }}>
                            <Box sx={{ display: "flex", alignItems: "center", gap: 1, flex: 1 }}>
                                <Skeleton variant="text" width={120} height={24} />
                                <Skeleton variant="rectangular" width={80} height={8} sx={{ borderRadius: 1 }} />
                            </Box>
                            <Box sx={{ display: "flex", gap: 1 }}>
                                <Skeleton variant="circular" width={32} height={32} />
                                <Skeleton variant="circular" width={32} height={32} />
                            </Box>
                        </Box>

                        {/* Content area */}
                        <Box sx={{ display: "flex", gap: 2 }}>
                            {/* Timeline skeleton */}
                            <Box sx={{ width: 220, display: { xs: "none", sm: "block" } }}>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    {[1, 2, 3].map((i) => (
                                        <Box key={i} sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                            <Skeleton variant="circular" width={12} height={12} />
                                            <Skeleton variant="text" width={140} height={16} />
                                        </Box>
                                    ))}
                                </Box>
                            </Box>

                            {/* Details skeleton */}
                            <Box sx={{ flex: 1 }}>
                                <Skeleton variant="text" width="60%" height={20} sx={{ mb: 1 }} />
                                <Skeleton variant="text" width="40%" height={16} sx={{ mb: 2 }} />
                                <Skeleton variant="rectangular" width="100%" height={120} sx={{ borderRadius: 1 }} />
                            </Box>
                        </Box>
                    </Box>
                </>
            ) : (
                <>
                    {/* Header with collapse all button */}
                    {runs.length > 1 && (
                        <Box
                            sx={{
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "flex-end",
                                mb: 1,
                                px: 1,
                            }}
                        >
                            <IconButton 
                                size="small" 
                                onClick={toggleAllCollapsed}
                                title={allCollapsed ? t("ExpandAll") : t("CollapseAll")}
                            >
                                <IconCommon name={allCollapsed ? "ChevronDown" : "ChevronUp"} />
                            </IconButton>
                        </Box>
                    )}

                    {/* Unified container for all runs */}
                    <Box 
                        sx={{ 
                            bgcolor: "background.paper",
                            borderRadius: 3,
                            boxShadow: 1,
                            overflow: "hidden",
                        }}
                    >
                {visibleRuns.map((run, index) => {
                    // Create mock ResourceVersion for RoutineExecutor
                    // TODO: Fetch actual ResourceVersion data
                    const mockResourceVersion = {
                        id: run.resourceVersionId,
                        name: run.resourceVersionName || "Routine",
                        description: "",
                        versionLabel: "1.0",
                        isComplete: true,
                        isPrivate: false,
                        resourceSubType: "Routine",
                        translations: run.resourceVersionName ? [{
                            language: "en",
                            name: run.resourceVersionName,
                            description: "",
                        }] : [],
                    };

                    const isFirst = index === 0;
                    const isLast = index === visibleRuns.length - 1;

                    return (
                        <Box
                            key={run.runId}
                            sx={{
                                // Add divider between runs
                                ...(index > 0 && {
                                    borderTop: 1,
                                    borderColor: "divider",
                                }),
                            }}
                        >
                            <RoutineExecutor
                                runId={run.runId}
                                resourceVersion={mockResourceVersion as any}
                                runStatus={run.runStatus}
                                isCollapsed={collapsedStates[run.runId] || false}
                                onToggleCollapse={() => handleToggleCollapse(run.runId)}
                                onRemove={() => handleRemoveRun(run.runId)}
                                className="chat-routine-executor"
                                chatMode={true}
                                isFirstInGroup={isFirst}
                                isLastInGroup={isLast}
                                showInUnifiedContainer={true}
                            />
                        </Box>
                    );
                })}
            </Box>

                    {/* Show more button if there are hidden runs */}
                    {hiddenRunsCount > 0 && (
                        <Box sx={{ mt: 1, textAlign: "center" }}>
                            <IconButton 
                                onClick={() => setShowAllRuns(true)}
                                size="small"
                                sx={{ fontSize: "0.75rem" }}
                            >
                                <Typography variant="caption" sx={{ mr: 0.5 }}>
                                    {t("ShowMore", { count: hiddenRunsCount })}
                                </Typography>
                                <IconCommon name="ChevronDown" />
                            </IconButton>
                        </Box>
                    )}
                </>
            )}
        </Box>
    );
}