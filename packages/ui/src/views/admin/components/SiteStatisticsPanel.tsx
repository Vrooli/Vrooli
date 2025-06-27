import { IconCommon } from "../../../icons/Icons.js";
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
    Alert,
    useTheme,
} from "@mui/material";
import { DAYS_1_MS, WEEKS_1_MS, MONTHS_1_MS, YEARS_1_MS, endpointsStatsSite, type StatPeriodType, type StatsSite, type StatsSiteSearchInput, type StatsSiteSearchResult } from "@vrooli/shared";
import React, { useCallback, useEffect, useMemo, useState, type ChangeEvent } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { statsDisplay } from "../../../utils/display/statsDisplay.js";
import { PageTabs } from "../../../components/PageTabs/PageTabs.js";
import { LineGraph } from "../../../components/graphs/LineGraph/LineGraph.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { DateRangeMenu } from "../../../components/lists/DateRangeMenu/DateRangeMenu.js";
import { displayDate } from "../../../utils/display/stringTools.js";

/**
 * Period options for stats display
 */
enum StatsPeriodOption {
    Daily = "Daily",
    Weekly = "Weekly", 
    Monthly = "Monthly",
    Yearly = "Yearly",
}

/** Maps period options to time frame intervals (in milliseconds) */
const periodIntervals: { [key in StatsPeriodOption]: number } = {
    Daily: DAYS_1_MS,
    Weekly: WEEKS_1_MS,
    Monthly: MONTHS_1_MS,
    Yearly: YEARS_1_MS,
};

/** Maps period options to PeriodType for API calls */
const periodTypes: { [key in StatsPeriodOption]: StatPeriodType } = {
    Daily: "Hourly",
    Weekly: "Daily", 
    Monthly: "Weekly",
    Yearly: "Monthly",
};

// Stats should not be earlier than February 2023.
const MIN_DATE = new Date(2023, 1, 1);

/**
 * Panel displaying site-wide statistics and analytics for administrators
 * This reuses and extends the existing StatsSiteView functionality
 */
function SiteStatisticsPanel(): React.ReactElement {
    const { t } = useTranslation();
    const [error, setError] = useState<string | null>(null);
    const [lastRefresh, setLastRefresh] = useState<Date>(new Date());
    
    // Period selection state
    const [currentPeriod, setCurrentPeriod] = useState<StatsPeriodOption>(StatsPeriodOption.Daily);
    
    // Date range state - defaults to the period's time frame
    const [period, setPeriod] = useState<{ after: Date, before: Date }>({
        after: new Date(Date.now() - periodIntervals[StatsPeriodOption.Daily]),
        before: new Date(),
    });
    
    // Date range menu state
    const [dateRangeAnchorEl, setDateRangeAnchorEl] = useState<Element | null>(null);
    
    const handleDateRangeOpen = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setDateRangeAnchorEl(event.currentTarget);
    }, []);
    
    const handleDateRangeClose = useCallback(() => {
        setDateRangeAnchorEl(null);
    }, []);
    
    const handleDateRangeSubmit = useCallback((newAfter?: Date | undefined, newBefore?: Date | undefined) => {
        setPeriod({
            after: newAfter || period.after,
            before: newBefore || period.before,
        });
        handleDateRangeClose();
    }, [period.after, period.before]);
    
    // Create period tabs
    const periodTabs = useMemo(() => [
        { key: "daily", label: t("Daily"), index: 0, iconInfo: { name: "Today" as const, type: "Common" as const } },
        { key: "weekly", label: t("Weekly"), index: 1, iconInfo: { name: "Week" as const, type: "Common" as const } },
        { key: "monthly", label: t("Monthly"), index: 2, iconInfo: { name: "Month" as const, type: "Common" as const } },
        { key: "yearly", label: t("Yearly"), index: 3, iconInfo: { name: "Day" as const, type: "Common" as const } },
    ], [t]);
    
    const [currTab, setCurrTab] = useState(periodTabs[0]);
    
    const handleTabChange = useCallback((event: React.SyntheticEvent, tab: typeof periodTabs[0]) => {
        setCurrTab(tab);
        const newPeriod = Object.values(StatsPeriodOption)[tab.index];
        setCurrentPeriod(newPeriod);
        
        // Reset date range based on tab selection
        const periodInterval = periodIntervals[newPeriod];
        const newAfter = new Date(Math.max(Date.now() - periodInterval, MIN_DATE.getTime()));
        const newBefore = new Date();
        setPeriod({ after: newAfter, before: newBefore });
    }, []);

    // Fetch stats data based on selected period and date range
    const [getStats, { data: statsData, loading, errors }] = useLazyFetch<StatsSiteSearchInput, StatsSiteSearchResult>({
        ...endpointsStatsSite.findMany,
        inputs: {
            periodType: periodTypes[currentPeriod],
            periodTimeFrame: {
                after: period.after.toISOString(),
                before: period.before.toISOString(),
            },
        },
    });

    const [stats, setStats] = useState<StatsSite[]>([]);
    
    // Fetch stats on mount, when refresh is triggered, or when period/date range changes
    useEffect(() => {
        getStats();
    }, [lastRefresh, currentPeriod, period]); // eslint-disable-line react-hooks/exhaustive-deps

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

    // Use the statsDisplay utility to aggregate stats and prepare visual data
    const { aggregate, visual } = useMemo(() => {
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
            const resourcesCreated = JSON.parse(latestStats.resourcesCreatedByType || "{}");
            const resourcesCompleted = JSON.parse(latestStats.resourcesCompletedByType || "{}");
            
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
        chartData?: number[];
        fieldKey?: string;
    }> = ({ title, value, subtitle, icon, trend, color = "primary", chartData = [], fieldKey }) => {
        const { dimensions, ref } = useDimensions<HTMLDivElement>();
        const theme = useTheme();
        
        // Get the actual color value from theme
        const getThemeColor = (colorName: typeof color) => {
            switch (colorName) {
                case "primary": return theme.palette.primary.main;
                case "secondary": return theme.palette.secondary.main;
                case "success": return theme.palette.success.main;
                case "warning": return theme.palette.warning.main;
                case "error": return theme.palette.error.main;
                default: return theme.palette.primary.main;
            }
        };
        
        const themeColor = getThemeColor(color);
        
        return (
        <Card 
            ref={ref}
            sx={{ 
                height: "100%",
                border: "1px solid",
                borderColor: "divider",
            }}
        >
            <CardContent sx={{ p: 3, "&:last-child": { pb: 3 } }}>
                <Stack spacing={2}>
                    {/* Header with icon and trend */}
                    <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                        <Box 
                            sx={{ 
                                p: 1.5, 
                                borderRadius: 2, 
                                bgcolor: `${color}.main`,
                                color: `${color}.contrastText`,
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                                minWidth: 48,
                                minHeight: 48,
                            }}
                        >
                            {icon}
                        </Box>
                        {trend && (
                            <Chip 
                                size="small" 
                                label={trend === "up" ? "↗" : trend === "down" ? "↘" : "→"}
                                color={trend === "up" ? "success" : trend === "down" ? "error" : "default"}
                                sx={{ 
                                    fontSize: "0.75rem",
                                    height: 24,
                                    "& .MuiChip-label": {
                                        px: 1,
                                    },
                                }}
                            />
                        )}
                    </Stack>
                    
                    {/* Main value */}
                    <Box>
                        <Typography 
                            variant="h3" 
                            component="div" 
                            sx={{ 
                                fontWeight: 700,
                                lineHeight: 1.2,
                                color: "text.primary",
                                mb: 0.5,
                            }}
                        >
                            {value}
                        </Typography>
                        <Typography 
                            variant="h6" 
                            sx={{ 
                                fontWeight: 500,
                                color: "text.primary",
                                mb: subtitle ? 0.5 : 0,
                            }}
                        >
                            {title}
                        </Typography>
                        {subtitle && (
                            <Typography 
                                variant="body2" 
                                sx={{ 
                                    color: "text.secondary",
                                    lineHeight: 1.4,
                                }}
                            >
                                {subtitle}
                            </Typography>
                        )}
                    </Box>

                    {/* Mini chart */}
                    {chartData.length > 1 && dimensions.width > 0 && (
                        <Box sx={{ height: 60, mt: 2 }}>
                            <LineGraph
                                data={chartData}
                                dims={{
                                    width: dimensions.width - 48, // Account for padding
                                    height: 60,
                                }}
                                lineColor={themeColor}
                                dotColor={themeColor}
                                hideAxes={true}
                                hideTooltips={false}
                                lineWidth={3}
                            />
                        </Box>
                    )}
                </Stack>
            </CardContent>
        </Card>
        );
    };

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
                    startIcon={<IconCommon name="Refresh" />}
                >
                    {t("Retry")}
                </Button>
            </Box>
        );
    }

    return (
        <Box>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
                <Box>
                    <Typography variant="h5" gutterBottom>
                        {t("SiteStats")}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        {currentPeriod === StatsPeriodOption.Daily ? t("TimeDay") : 
                         currentPeriod === StatsPeriodOption.Weekly ? t("TimeWeek") :
                         currentPeriod === StatsPeriodOption.Monthly ? t("TimeMonth") :
                         t("TimeYear")}
                    </Typography>
                </Box>
                <IconButton 
                    onClick={handleRefresh} 
                    disabled={loading}
                    title={t("Refresh")}
                >
                    <IconCommon name="Refresh" />
                </IconButton>
            </Box>

            {/* Period Selection Tabs */}
            <Paper sx={{ mb: 3 }}>
                <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
                    <PageTabs
                        ariaLabel="stats-period-tabs"
                        currTab={currTab}
                        onChange={handleTabChange}
                        tabs={periodTabs}
                        fullWidth={true}
                    />
                </Box>
            </Paper>

            {/* Date Range Picker */}
            <DateRangeMenu
                anchorEl={dateRangeAnchorEl}
                minDate={MIN_DATE}
                maxDate={new Date()}
                onClose={handleDateRangeClose}
                onSubmit={handleDateRangeSubmit}
                range={period}
                strictIntervalRange={periodIntervals[currentPeriod]}
            />
            
            {/* Date Range Display */}
            <Typography
                component="div"
                variant="body1"
                textAlign="center"
                onClick={handleDateRangeOpen}
                sx={{ 
                    cursor: "pointer", 
                    mb: 3, 
                    mt: 1,
                    p: 1,
                    borderRadius: 1,
                    "&:hover": {
                        backgroundColor: "action.hover",
                    },
                    color: "primary.main",
                    fontWeight: 500,
                }}
            >
                📅 {displayDate(period.after.getTime(), false)} - {displayDate(period.before.getTime(), false)}
            </Typography>

            {error && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                    {error}
                </Alert>
            )}

            {/* Key Metrics */}
            <Grid container spacing={3} sx={{ mb: 4, px: 2 }}>
                <Grid item xs={12} sm={6} lg={3}>
                    <StatCard
                        title={t("ActiveUsers")}
                        value={metrics.activeUsers.toLocaleString()}
                        subtitle={`Over ${currentPeriod.toLowerCase()} period`}
                        icon={<IconCommon name="Team" size={24} />}
                        trend={metrics.activeUsers > 0 ? "up" : "stable"}
                        color="primary"
                        chartData={visual.activeUsers || []}
                        fieldKey="activeUsers"
                    />
                </Grid>
                <Grid item xs={12} sm={6} lg={3}>
                    <StatCard
                        title={t("RoutinesCreated")}
                        value={metrics.routinesCreated.toLocaleString()}
                        subtitle={`New routines in ${currentPeriod.toLowerCase()}`}
                        icon={<IconCommon name="Build" size={24} />}
                        trend={metrics.routinesCreated > 0 ? "up" : "stable"}
                        color="success"
                        chartData={visual.routinesCreated || []}
                        fieldKey="routinesCreated"
                    />
                </Grid>
                <Grid item xs={12} sm={6} lg={3}>
                    <StatCard
                        title={t("RunsCompleted")}
                        value={metrics.runsCompleted.toLocaleString()}
                        subtitle={`${metrics.runCompletionRate}% completion rate`}
                        icon={<IconCommon name="Complete" size={24} />}
                        trend={metrics.runsCompleted > 0 ? "up" : "stable"}
                        color="success"
                        chartData={visual.runsCompleted || []}
                        fieldKey="runsCompleted"
                    />
                </Grid>
                <Grid item xs={12} sm={6} lg={3}>
                    <StatCard
                        title={t("NewTeams")}
                        value={metrics.teamsCreated.toLocaleString()}
                        subtitle={`Teams created in ${currentPeriod.toLowerCase()}`}
                        icon={<IconCommon name="Team" size={24} />}
                        trend={metrics.teamsCreated > 0 ? "up" : "stable"}
                        color="secondary"
                        chartData={visual.teamsCreated || []}
                        fieldKey="teamsCreated"
                    />
                </Grid>
            </Grid>

            {/* Activity Summary */}
            <Grid container spacing={3} sx={{ mb: 4, px: 2 }}>
                <Grid item xs={12} md={6}>
                    <Card 
                        sx={{ 
                            height: "100%",
                            border: "1px solid",
                            borderColor: "divider",
                        }}
                    >
                        <CardContent sx={{ p: 3 }}>
                            <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 2 }}>
                                <Box 
                                    sx={{ 
                                        p: 1, 
                                        borderRadius: 2, 
                                        bgcolor: metrics.runCompletionRate >= 80 ? "success.main" : metrics.runCompletionRate >= 50 ? "warning.main" : "error.main",
                                        color: "white",
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }}
                                >
                                    <IconCommon name="Stats" size={20} />
                                </Box>
                                <Typography variant="h6" sx={{ fontWeight: 600 }}>
                                    {t("RunCompletionRate")}
                                </Typography>
                            </Stack>
                            
                            <Typography 
                                variant="h4" 
                                sx={{ 
                                    fontWeight: 700,
                                    mb: 2,
                                    color: metrics.runCompletionRate >= 80 ? "success.main" : metrics.runCompletionRate >= 50 ? "warning.main" : "error.main",
                                }}
                            >
                                {metrics.runCompletionRate}%
                            </Typography>
                            
                            <Box sx={{ mb: 2 }}>
                                <LinearProgress 
                                    variant="determinate" 
                                    value={metrics.runCompletionRate} 
                                    color={metrics.runCompletionRate >= 80 ? "success" : metrics.runCompletionRate >= 50 ? "warning" : "error"}
                                    sx={{ 
                                        height: 10, 
                                        borderRadius: 5,
                                        bgcolor: "grey.200",
                                    }}
                                />
                            </Box>
                            
                            <Typography variant="body2" color="text.secondary">
                                {metrics.runsCompleted.toLocaleString()} completed of {metrics.runsStarted.toLocaleString()} started
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} md={6}>
                    <Card 
                        sx={{ 
                            height: "100%",
                            border: "1px solid",
                            borderColor: "divider",
                        }}
                    >
                        <CardContent sx={{ p: 3 }}>
                            <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 2 }}>
                                <Box 
                                    sx={{ 
                                        p: 1, 
                                        borderRadius: 2, 
                                        bgcolor: "primary.main",
                                        color: "primary.contrastText",
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }}
                                >
                                    <IconCommon name="Add" size={20} />
                                </Box>
                                <Typography variant="h6" sx={{ fontWeight: 600 }}>
                                    {t("NewAccounts")}
                                </Typography>
                            </Stack>
                            
                            <Typography 
                                variant="h4" 
                                sx={{ 
                                    fontWeight: 700,
                                    color: "primary.main",
                                    mb: 1,
                                }}
                            >
                                {metrics.totalVerifiedAccounts.toLocaleString()}
                            </Typography>
                            
                            <Typography variant="body2" color="text.secondary">
                                Verified accounts created today
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>
        </Box>
    );
}

export { SiteStatisticsPanel };
