import {
    Box,
    Card,
    CardContent,
    Grid,
    LinearProgress,
    Stack,
    Typography,
} from "@mui/material";
import { IconCommon } from "../../icons/Icons.js";

export interface SystemMetrics {
    timestamp: number;
    uptime: number;
    system: {
        cpu: {
            usage: number;
            cores: number;
            loadAvg: number[];
        };
        memory: {
            heapUsed: number;
            heapTotal: number;
            heapUsedPercent: number;
            rss: number;
            external: number;
            arrayBuffers: number;
        };
        disk?: {
            total: number;
            used: number;
            usagePercent: number;
        };
    };
    application: {
        nodeVersion: string;
        pid: number;
        platform: string;
        environment: string;
        websockets: {
            connections: number;
            rooms: number;
        };
        queues: {
            [queueName: string]: {
                waiting: number;
                active: number;
                completed: number;
                failed: number;
                delayed: number;
                total: number;
            };
        };
        llmServices: {
            [serviceId: string]: {
                state: string;
                cooldownUntil?: number;
            };
        };
    };
    database: {
        connected: boolean;
        poolSize?: number;
    };
    redis: {
        connected: boolean;
        memory?: {
            used: number;
            peak: number;
        };
        stats?: {
            connections: number;
            commands: number;
        };
    };
    api: {
        requestsTotal?: number;
        responseTimes?: {
            min: number;
            max: number;
            avg: number;
        };
    };
}

function formatBytes(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

function formatUptime(seconds: number): string {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${minutes}m`;
}

interface SystemMetricsCardsProps {
    metricsData: SystemMetrics | null;
    loading: boolean;
    error: string | null;
}

export function SystemMetricsCards({ metricsData, loading, error }: SystemMetricsCardsProps) {
    if (loading || error || !metricsData) {
        return null;
    }

    return (
        <Grid container spacing={3}>
            <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ height: "100%" }}>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            <IconCommon name="Stats" size={20} /> CPU
                        </Typography>
                        <Typography variant="h4">
                            {metricsData.system.cpu.usage}%
                        </Typography>
                        <LinearProgress
                            variant="determinate"
                            value={metricsData.system.cpu.usage}
                            sx={{ mt: 1 }}
                            color={metricsData.system.cpu.usage > 80 ? "error" : "primary"}
                        />
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {metricsData.system.cpu.cores} cores
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ height: "100%" }}>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            <IconCommon name="Storage" size={20} /> Memory
                        </Typography>
                        <Typography variant="h4">
                            {metricsData.system.memory.heapUsedPercent}%
                        </Typography>
                        <LinearProgress
                            variant="determinate"
                            value={metricsData.system.memory.heapUsedPercent}
                            sx={{ mt: 1 }}
                            color={metricsData.system.memory.heapUsedPercent > 80 ? "error" : "primary"}
                        />
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {formatBytes(metricsData.system.memory.heapUsed)} / {formatBytes(metricsData.system.memory.heapTotal)}
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ height: "100%" }}>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            <IconCommon name="Schedule" size={20} /> Uptime
                        </Typography>
                        <Typography variant="h4">
                            {formatUptime(metricsData.uptime)}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            Since: {new Date(Date.now() - metricsData.uptime * 1000).toLocaleString()}
                        </Typography>
                    </CardContent>
                </Card>
            </Grid>

            <Grid item xs={12} sm={6} md={3}>
                <Card sx={{ height: "100%" }}>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            <IconCommon name="Link" size={20} /> Connections
                        </Typography>
                        <Stack direction="row" spacing={2}>
                            <Box>
                                <Typography variant="h5">
                                    {metricsData.application.websockets.connections}
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    WebSockets
                                </Typography>
                            </Box>
                            <Box>
                                <Typography variant="h5">
                                    {metricsData.application.websockets.rooms}
                                </Typography>
                                <Typography variant="body2" color="text.secondary">
                                    Rooms
                                </Typography>
                            </Box>
                        </Stack>
                    </CardContent>
                </Card>
            </Grid>

            {/* Disk Usage - only show if available */}
            {metricsData.system.disk && (
                <Grid item xs={12}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>
                                <IconCommon name="Storage" size={20} /> Disk Usage
                            </Typography>
                            <Stack spacing={1}>
                                <Typography variant="h4">
                                    {metricsData.system.disk.usagePercent}%
                                </Typography>
                                <LinearProgress
                                    variant="determinate"
                                    value={metricsData.system.disk.usagePercent}
                                    color={metricsData.system.disk.usagePercent > 85 ? "error" : "primary"}
                                />
                                <Typography variant="body2" color="text.secondary">
                                    {formatBytes(metricsData.system.disk.used)} / {formatBytes(metricsData.system.disk.total)}
                                </Typography>
                            </Stack>
                        </CardContent>
                    </Card>
                </Grid>
            )}
        </Grid>
    );
}
