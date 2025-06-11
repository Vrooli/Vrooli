import { 
    TrendingUp, 
    People, 
    Code, 
    Timeline 
} from "@mui/icons-material";
import { 
    Box,
    Card,
    CardContent,
    Grid,
    Typography,
    Chip,
    LinearProgress,
    Paper,
    Stack,
    CircularProgress,
    IconButton,
    Button,
    Alert
} from "@mui/material";
import { DAYS_1_MS, endpointsStatsSite, StatPeriodType, type StatsSite, type StatsSiteSearchInput, type StatsSiteSearchResult } from "@vrooli/shared";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { statsDisplay } from "../../utils/display/statsDisplay.js";
import { Refresh as RefreshIcon } from "@mui/icons-material";

/**
 * Panel displaying site-wide statistics and analytics for administrators
 * This reuses and extends the existing StatsSiteView functionality
 */
function SiteStatisticsPanel(): React.ReactElement {
    const { t } = useTranslation();
    const [error, setError] = useState<string | null>(null);
    const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

    // Fetch stats data for the last 24 hours (daily view)
    const [getStats, { data: statsData, loading, errors }] = useLazyFetch<StatsSiteSearchInput, StatsSiteSearchResult>({
        ...endpointsStatsSite.findMany,
        inputs: {
            periodType: StatPeriodType.Hourly,
            periodTimeFrame: {
                after: new Date(Date.now() - DAYS_1_MS).toISOString(),
                before: new Date().toISOString(),
            },
        },
    });

    const [stats, setStats] = useState<StatsSite[]>([]);
    
    // Fetch stats on mount and when refresh is triggered
    useEffect(() => {
        getStats();
    }, [lastRefresh]); // eslint-disable-line react-hooks/exhaustive-deps

    useEffect(() => {
        if (errors && errors.length > 0) {
            setError(errors[0].message || t("FailedToLoadStats"));
        } else if (statsData) {
            // Sort stats by periodStart to ensure proper ordering
            const sortedStats = statsData.edges
                .map(edge => edge.node)
                .sort((a, b) => new Date(a.periodStart).getTime() - new Date(b.periodStart).getTime());
            setStats(sortedStats);
            setError(null);
        }
    }, [statsData, errors, t]);

    const handleRefresh = useCallback(() => {
        setLastRefresh(new Date());
    }, []);

    // Use the statsDisplay utility to aggregate stats properly
    const { aggregate } = useMemo(() => {
        if (stats.length === 0) {
            return { 
                aggregate: {
                    activeUsers: 0,
                    teamsCreated: 0,
                    verifiedEmailsCreated: 0,
                    verifiedWalletsCreated: 0,
                    runsStarted: 0,
                    runsCompleted: 0,
                },
                visual: {},
            };
        }
        return statsDisplay(stats);
    }, [stats]);

    // Parse resource data from the latest stats entry
    const resourceStats = useMemo(() => {
        if (stats.length === 0) {
            return {
                routinesCreated: 0,
                apisCreated: 0,
                projectsCreated: 0,
            };
        }

        const latestStats = stats[stats.length - 1];
        try {
            const resourcesCreated = JSON.parse(latestStats.resourcesCreatedByType || '{}');
            const resourcesCompleted = JSON.parse(latestStats.resourcesCompletedByType || '{}');
            
            return {
                routinesCreated: resourcesCreated.Routine || 0,
                apisCreated: resourcesCreated.Api || 0,
                projectsCreated: resourcesCreated.Project || 0,
                routinesCompleted: resourcesCompleted.Routine || 0,
            };
        } catch (e) {
            console.error("Failed to parse resource stats:", e);
            return {
                routinesCreated: 0,
                apisCreated: 0,
                projectsCreated: 0,
                routinesCompleted: 0,
            };
        }
    }, [stats]);

    // Calculate derived metrics
    const metrics = useMemo(() => {
        const runCompletionRate = aggregate.runsStarted > 0 
            ? Math.round((aggregate.runsCompleted / aggregate.runsStarted) * 100)
            : 0;
        
        const totalVerifiedAccounts = aggregate.verifiedEmailsCreated + aggregate.verifiedWalletsCreated;
        
        return {
            activeUsers: aggregate.activeUsers,
            totalVerifiedAccounts,
            runsCompleted: aggregate.runsCompleted,
            runsStarted: aggregate.runsStarted,
            runCompletionRate,
            teamsCreated: aggregate.teamsCreated,
            routinesCreated: resourceStats.routinesCreated,
            routinesCompleted: resourceStats.routinesCompleted,
        };
    }, [aggregate, resourceStats]);

    const StatCard: React.FC<{
        title: string;
        value: string | number;
        subtitle?: string;
        icon: React.ReactNode;
        trend?: "up" | "down" | "stable";
        color?: "primary" | "secondary" | "success" | "warning" | "error";
    }> = ({ title, value, subtitle, icon, trend, color = "primary" }) => (
        <Card>
            <CardContent>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Box 
                        sx={{ 
                            p: 1, 
                            borderRadius: 1, 
                            bgcolor: `${color}.main`,
                            color: `${color}.contrastText`
                        }}
                    >
                        {icon}
                    </Box>
                    <Box flex={1}>
                        <Typography variant="h4" component="div">
                            {value}
                        </Typography>
                        <Typography variant="subtitle1" color="text.secondary">
                            {title}
                        </Typography>
                        {subtitle && (
                            <Typography variant="body2" color="text.secondary">
                                {subtitle}
                            </Typography>
                        )}
                        {trend && (
                            <Chip 
                                size="small" 
                                label={trend} 
                                color={trend === "up" ? "success" : trend === "down" ? "error" : "default"}
                                sx={{ mt: 1 }}
                            />
                        )}
                    </Box>
                </Stack>
            </CardContent>
        </Card>
    );

    if (loading && stats.length === 0) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
                <CircularProgress />
            </Box>
        );
    }

    if (error && stats.length === 0) {
        return (
            <Box>
                <Alert severity="error" sx={{ mb: 2 }}>
                    {error}
                </Alert>
                <Button 
                    variant="outlined" 
                    onClick={handleRefresh}
                    startIcon={<RefreshIcon />}
                >
                    {t("Retry")}
                </Button>
            </Box>
        );
    }

    return (
        <Box>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <Box>
                    <Typography variant="h5" gutterBottom>
                        {t("SiteStatistics")}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        {t("Last24Hours")}
                    </Typography>
                </Box>
                <IconButton 
                    onClick={handleRefresh} 
                    disabled={loading}
                    title={t("Refresh")}
                >
                    <RefreshIcon />
                </IconButton>
            </Box>

            {error && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                    {error}
                </Alert>
            )}

            {/* Key Metrics */}
            <Grid container spacing={3} sx={{ mb: 4 }}>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("ActiveUsers")}
                        value={metrics.activeUsers.toLocaleString()}
                        subtitle={t("PeakConcurrentUsers")}
                        icon={<People />}
                        trend={metrics.activeUsers > 0 ? "up" : "stable"}
                        color="primary"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("RoutinesCreated")}
                        value={metrics.routinesCreated.toLocaleString()}
                        subtitle={t("InLast24Hours")}
                        icon={<Code />}
                        trend={metrics.routinesCreated > 0 ? "up" : "stable"}
                        color="success"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("RunsCompleted")}
                        value={metrics.runsCompleted.toLocaleString()}
                        subtitle={t("RunsStartedCount", { count: metrics.runsStarted })}
                        icon={<Timeline />}
                        trend={metrics.runsCompleted > 0 ? "up" : "stable"}
                        color="secondary"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("NewTeams")}
                        value={metrics.teamsCreated.toLocaleString()}
                        subtitle={t("CreatedToday")}
                        icon={<TrendingUp />}
                        trend={metrics.teamsCreated > 0 ? "up" : "stable"}
                        color="info"
                    />
                </Grid>
            </Grid>

            {/* Activity Summary */}
            <Grid container spacing={3} sx={{ mb: 4 }}>
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            {t("RunCompletionRate")}
                        </Typography>
                        <Box sx={{ mb: 2 }}>
                            <LinearProgress 
                                variant="determinate" 
                                value={metrics.runCompletionRate} 
                                color={metrics.runCompletionRate >= 80 ? "success" : metrics.runCompletionRate >= 50 ? "warning" : "error"}
                                sx={{ height: 8, borderRadius: 4 }}
                            />
                        </Box>
                        <Typography variant="body2" color="text.secondary">
                            {t("RunsCompletedOfStarted", { 
                                completed: metrics.runsCompleted, 
                                started: metrics.runsStarted,
                                percentage: metrics.runCompletionRate,
                            })}
                        </Typography>
                    </Paper>
                </Grid>
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            {t("NewAccounts")}
                        </Typography>
                        <Typography variant="h4" color="primary" sx={{ mb: 1 }}>
                            {metrics.totalVerifiedAccounts}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            {t("VerifiedAccountsCreated")}
                        </Typography>
                    </Paper>
                </Grid>
            </Grid>

            {/* Placeholder for future charts and detailed analytics */}
            <Paper sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                    {t("DetailedAnalytics")}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                    {t("DetailedAnalyticsComingSoon")}
                </Typography>
                {/* Integration with StatsSiteView components can be added here */}
                {/* Charts and detailed analytics will be displayed here */}
            </Paper>
        </Box>
    );
}

export { SiteStatisticsPanel };