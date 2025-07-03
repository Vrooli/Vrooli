import { Alert, Box, Card, CardContent, Chip, CircularProgress, Grid, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { endpointsAdmin, type AdminSiteStatsOutput } from "@vrooli/shared";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { PubSub } from "../../utils/pubsub.js";

interface CreditStatCardProps {
    title: string;
    value: string | number;
    subtitle?: string;
    icon?: string;
    trend?: number;
    loading?: boolean;
}

function CreditStatCard({ title, value, subtitle, icon, trend, loading }: CreditStatCardProps) {
    const { palette } = useTheme();
    
    return (
        <Card sx={{ height: "100%" }}>
            <CardContent>
                <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                    <Box flex={1}>
                        <Typography color="text.secondary" gutterBottom variant="caption">
                            {title}
                        </Typography>
                        {loading ? (
                            <CircularProgress size={24} />
                        ) : (
                            <>
                                <Typography variant="h4" component="div">
                                    {value}
                                </Typography>
                                {subtitle && (
                                    <Typography variant="body2" color="text.secondary">
                                        {subtitle}
                                    </Typography>
                                )}
                            </>
                        )}
                    </Box>
                    {icon && (
                        <IconCommon 
                            name={icon as any} 
                            fill={palette.primary.main} 
                            size={32} 
                        />
                    )}
                </Stack>
                {trend !== undefined && !loading && (
                    <Box mt={1}>
                        <Typography 
                            variant="caption" 
                            color={trend >= 0 ? "success.main" : "error.main"}
                        >
                            {trend >= 0 ? "+" : ""}{trend}% from last month
                        </Typography>
                    </Box>
                )}
            </CardContent>
        </Card>
    );
}

interface JobStatusBannerProps {
    status: "success" | "partial" | "failed" | "never_run";
    lastRunTime: string | null;
    nextRunTime: string;
}

function JobStatusBanner({ status, lastRunTime, nextRunTime }: JobStatusBannerProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    
    const statusConfig = {
        success: { color: palette.success.main, icon: "CheckCircle", text: "Success" },
        partial: { color: palette.warning.main, icon: "Warning", text: "Partial Success" },
        failed: { color: palette.error.main, icon: "Error", text: "Failed" },
        never_run: { color: palette.grey[500], icon: "Schedule", text: "Never Run" },
    };
    
    const config = statusConfig[status];
    const nextRunDate = new Date(nextRunTime);
    const timeUntilNext = nextRunDate.getTime() - Date.now();
    const daysUntilNext = Math.floor(timeUntilNext / (1000 * 60 * 60 * 24));
    
    return (
        <Paper sx={{ p: 2, mb: 3 }}>
            <Stack direction="row" spacing={2} alignItems="center">
                <IconCommon 
                    name={config.icon as any} 
                    fill={config.color} 
                    size={24} 
                />
                <Box flex={1}>
                    <Typography variant="body1">
                        <strong>Job Status:</strong> {config.text}
                    </Typography>
                    {lastRunTime && (
                        <Typography variant="caption" color="text.secondary">
                            Last run: {new Date(lastRunTime).toLocaleString()}
                        </Typography>
                    )}
                </Box>
                <Box textAlign="right">
                    <Typography variant="body2">
                        Next run in {daysUntilNext} days
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                        {nextRunDate.toLocaleDateString()} at {nextRunDate.toLocaleTimeString()}
                    </Typography>
                </Box>
            </Stack>
        </Paper>
    );
}

// Format credit amounts for display with better BigInt handling
const formatCredits = (amount: string) => {
    try {
        const num = BigInt(amount || "0");
        // Use BigInt division to avoid precision loss for large numbers
        const credits = num / BigInt(1e6);
        return credits.toLocaleString();
    } catch (error) {
        console.error("Error formatting credits:", error);
        return "0";
    }
};

export function CreditStatsPanel() {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [lastRefresh, setLastRefresh] = useState(Date.now());
    
    const [fetchSiteStats, { data, loading, errors }] = useLazyFetch<undefined, AdminSiteStatsOutput>(endpointsAdmin.siteStats);
    
    // Fetch data on mount and when refreshing
    useEffect(() => {
        fetchSiteStats();
    }, [fetchSiteStats, lastRefresh]);
    
    // Auto-refresh every 2 minutes (less aggressive)
    useEffect(() => {
        const interval = setInterval(() => {
            if (!loading) { // Don't refresh if already loading
                setLastRefresh(Date.now());
            }
        }, 120000); // 2 minutes
        
        return () => clearInterval(interval);
    }, [loading]);
    
    
    // Calculate month-over-month trend with error handling
    const calculateTrend = () => {
        try {
            if (!data?.creditStats?.donationsByMonth || data.creditStats.donationsByMonth.length < 2) {
                return undefined;
            }
            const current = BigInt(data.creditStats.donationsByMonth[0].amount);
            const previous = BigInt(data.creditStats.donationsByMonth[1].amount);
            if (previous === BigInt(0)) return undefined;
            return Math.round((Number(current - previous) / Number(previous)) * 100);
        } catch (error) {
            console.error("Error calculating trend:", error);
            return undefined;
        }
    };
    
    // Prepare donation history data with error handling
    const donationHistory = useMemo(() => {
        try {
            if (!data?.creditStats?.donationsByMonth) return [];
            
            return data.creditStats.donationsByMonth.map(item => {
                let monthDisplay: string;
                try {
                    const date = new Date(item.month + "-01");
                    if (isNaN(date.getTime())) {
                        throw new Error("Invalid date");
                    }
                    monthDisplay = date.toLocaleDateString(undefined, { month: "short", year: "numeric" });
                } catch (error) {
                    console.error("Error parsing month:", item.month, error);
                    monthDisplay = item.month; // Fallback to raw month string
                }
                
                return {
                    month: monthDisplay,
                    credits: Number(BigInt(item.amount) / BigInt(1e6)),
                    donors: item.donors,
                    formattedCredits: formatCredits(item.amount),
                };
            });
        } catch (error) {
            console.error("Error preparing donation history:", error);
            return [];
        }
    }, [data]);
    
    if (errors && errors.length > 0) {
        return (
            <Alert severity="error">
                {t("ErrorLoadingStats")}: {errors[0].message}
            </Alert>
        );
    }
    
    if (!data && loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
                <CircularProgress />
            </Box>
        );
    }
    
    const stats = data?.creditStats;
    if (!stats) {
        return (
            <Alert severity="info">
                {t("NoDataAvailable")}
            </Alert>
        );
    }
    
    return (
        <Box>
            {/* Job Status Banner */}
            <JobStatusBanner 
                status={stats.lastRolloverJobStatus}
                lastRunTime={stats.lastRolloverJobTime}
                nextRunTime={stats.nextScheduledRollover}
            />
            
            {/* Overview Cards */}
            <Grid container spacing={3} sx={{ mb: 3 }}>
                <Grid item xs={12} sm={6} md={3}>
                    <CreditStatCard
                        title={t("TotalInCirculation")}
                        value={formatCredits(stats.totalCreditsInCirculation)}
                        subtitle={t("credits")}
                        icon="AccountBalance"
                        loading={loading}
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <CreditStatCard
                        title={t("DonatedThisMonth")}
                        value={formatCredits(stats.totalCreditsDonatedThisMonth)}
                        subtitle={t("credits")}
                        icon="Volunteer"
                        trend={calculateTrend()}
                        loading={loading}
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <CreditStatCard
                        title={t("ActiveDonors")}
                        value={stats.activeDonorsThisMonth}
                        subtitle={t("thisMonth")}
                        icon="Group"
                        loading={loading}
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <CreditStatCard
                        title={t("AvgDonationPercent")}
                        value={`${Math.round(stats.averageDonationPercentage)}%`}
                        subtitle={t("ofFreeCredits")}
                        icon="Percentage"
                        loading={loading}
                    />
                </Grid>
            </Grid>
            
            {/* Donation History Table */}
            <Paper sx={{ p: 3, mb: 3 }}>
                <Typography variant="h6" gutterBottom>
                    {t("DonationHistory")}
                </Typography>
                <TableContainer>
                    <Table>
                        <TableHead>
                            <TableRow>
                                <TableCell>{t("Month")}</TableCell>
                                <TableCell align="right">{t("Credits")}</TableCell>
                                <TableCell align="right">{t("Donors")}</TableCell>
                                <TableCell align="right">{t("AvgPerDonor")}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {donationHistory.map((row) => (
                                <TableRow key={row.month}>
                                    <TableCell>{row.month}</TableCell>
                                    <TableCell align="right">{row.formattedCredits}</TableCell>
                                    <TableCell align="right">{row.donors}</TableCell>
                                    <TableCell align="right">
                                        {row.donors > 0 ? Math.round(row.credits / row.donors).toLocaleString() : "-"}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            </Paper>
            
            {/* Summary Stats */}
            <Paper sx={{ p: 3 }}>
                <Typography variant="h6" gutterBottom>
                    {t("AllTimeStats")}
                </Typography>
                <Stack direction="row" spacing={4} flexWrap="wrap">
                    <Box>
                        <Typography variant="body2" color="text.secondary">
                            {t("TotalDonated")}
                        </Typography>
                        <Typography variant="h5">
                            {formatCredits(stats.totalCreditsDonatedAllTime)} {t("credits")}
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="body2" color="text.secondary">
                            {t("EstimatedValue")}
                        </Typography>
                        <Typography variant="h5">
                            ${(Number(BigInt(stats.totalCreditsDonatedAllTime) / BigInt(1e8)) / 100).toFixed(2)}
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="body2" color="text.secondary">
                            {t("ImpactMessage")}
                        </Typography>
                        <Typography variant="body1">
                            {t("DonationsHelpSwarms")}
                        </Typography>
                    </Box>
                </Stack>
            </Paper>
        </Box>
    );
}
