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
    Stack
} from "@mui/material";
import React from "react";
import { useTranslation } from "react-i18next";

/**
 * Panel displaying site-wide statistics and analytics for administrators
 * This reuses and extends the existing StatsSiteView functionality
 */
export const SiteStatisticsPanel: React.FC = () => {
    const { t } = useTranslation();

    // TODO: Replace with actual data fetching hooks
    const mockStats = {
        totalUsers: 1234,
        activeUsers: 567,
        totalRoutines: 890,
        activeRoutines: 445,
        totalApiCalls: 12345,
        apiCallsToday: 234,
        creditUsage: 75, // percentage
        storageUsage: 45, // percentage
    };

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

    return (
        <Box>
            <Typography variant="h5" gutterBottom>
                {t("SiteStatistics")}
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
                {t("SiteStatisticsDescription")}
            </Typography>

            {/* Key Metrics */}
            <Grid container spacing={3} sx={{ mb: 4 }}>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("TotalUsers")}
                        value={mockStats.totalUsers.toLocaleString()}
                        subtitle={`${mockStats.activeUsers} active`}
                        icon={<People />}
                        trend="up"
                        color="primary"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("TotalRoutines")}
                        value={mockStats.totalRoutines.toLocaleString()}
                        subtitle={`${mockStats.activeRoutines} active`}
                        icon={<Code />}
                        trend="up"
                        color="success"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("APICallsToday")}
                        value={mockStats.apiCallsToday.toLocaleString()}
                        subtitle={`${mockStats.totalApiCalls.toLocaleString()} total`}
                        icon={<Timeline />}
                        trend="stable"
                        color="secondary"
                    />
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <StatCard
                        title={t("SystemHealth")}
                        value="Excellent"
                        subtitle="All systems operational"
                        icon={<TrendingUp />}
                        color="success"
                    />
                </Grid>
            </Grid>

            {/* Resource Usage */}
            <Grid container spacing={3} sx={{ mb: 4 }}>
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            {t("CreditUsage")}
                        </Typography>
                        <Box sx={{ mb: 2 }}>
                            <LinearProgress 
                                variant="determinate" 
                                value={mockStats.creditUsage} 
                                color={mockStats.creditUsage > 80 ? "warning" : "primary"}
                                sx={{ height: 8, borderRadius: 4 }}
                            />
                        </Box>
                        <Typography variant="body2" color="text.secondary">
                            {mockStats.creditUsage}% of monthly allocation used
                        </Typography>
                    </Paper>
                </Grid>
                <Grid item xs={12} md={6}>
                    <Paper sx={{ p: 3 }}>
                        <Typography variant="h6" gutterBottom>
                            {t("StorageUsage")}
                        </Typography>
                        <Box sx={{ mb: 2 }}>
                            <LinearProgress 
                                variant="determinate" 
                                value={mockStats.storageUsage} 
                                color="primary"
                                sx={{ height: 8, borderRadius: 4 }}
                            />
                        </Box>
                        <Typography variant="body2" color="text.secondary">
                            {mockStats.storageUsage}% of storage capacity used
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
                {/* TODO: Integrate with existing StatsSiteView components */}
                {/* TODO: Add time-based charts (daily, weekly, monthly) */}
                {/* TODO: Add user growth charts */}
                {/* TODO: Add API usage trends */}
            </Paper>
        </Box>
    );
};