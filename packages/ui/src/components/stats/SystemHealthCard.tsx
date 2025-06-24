import {
    Alert,
    Box,
    Card,
    CardContent,
    Chip,
    CircularProgress,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
    useTheme,
} from "@mui/material";
import { IconCommon } from "../../icons/Icons.js";

export interface HealthStatus {
    healthy: boolean;
    status: "Operational" | "Degraded" | "Down";
    lastChecked: number;
    details?: any;
}

export interface SystemHealth {
    status: "Operational" | "Degraded" | "Down";
    version: string;
    services: {
        api: HealthStatus;
        bus: HealthStatus;
        cronJobs: HealthStatus;
        database: HealthStatus;
        i18n: HealthStatus;
        llm: { [key: string]: HealthStatus };
        mcp: HealthStatus;
        memory: HealthStatus;
        queues: { [key: string]: HealthStatus };
        redis: HealthStatus;
        ssl: HealthStatus;
        stripe: HealthStatus;
        system: HealthStatus;
        websocket: HealthStatus;
        imageStorage: HealthStatus;
        embeddingService: HealthStatus;
    };
    timestamp: number;
}

function StatusChip({ status }: { status: string }) {
    const getStatusColor = (status: string) => {
        switch (status) {
            case "Operational": return "success";
            case "Degraded": return "warning";
            case "Down": return "error";
            default: return "default";
        }
    };

    return (
        <Chip
            label={status}
            color={getStatusColor(status) as any}
            size="small"
            variant="outlined"
        />
    );
}

interface SystemHealthCardProps {
    healthData: SystemHealth | null;
    loading: boolean;
    error: string | null;
    showDetails?: boolean;
}

export function SystemHealthCard({ healthData, loading, error, showDetails = true }: SystemHealthCardProps) {
    const { palette } = useTheme();

    const renderServiceStatus = (name: string, status: HealthStatus) => (
        <TableRow key={name}>
            <TableCell>{name}</TableCell>
            <TableCell>
                <StatusChip status={status.status} />
            </TableCell>
            <TableCell>
                {status.healthy ? (
                    <IconCommon name="CheckCircle" fill={palette.success.main} size={20} />
                ) : (
                    <IconCommon name="Error" fill={palette.error.main} size={20} />
                )}
            </TableCell>
            <TableCell>{new Date(status.lastChecked).toLocaleTimeString()}</TableCell>
        </TableRow>
    );

    return (
        <Card>
            <CardContent>
                <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: showDetails ? 2 : 0 }}>
                    <Typography variant="h6">System Health</Typography>
                    {loading ? (
                        <CircularProgress size={20} />
                    ) : healthData ? (
                        <StatusChip status={healthData.status} />
                    ) : null}
                </Stack>
                
                {error && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                        {error}
                    </Alert>
                )}
                
                {healthData && (
                    <>
                        <Stack direction="row" spacing={2} sx={{ mb: showDetails ? 2 : 0 }}>
                            <Typography variant="body2" color="text.secondary">
                                Version: {healthData.version}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Last Check: {new Date(healthData.timestamp).toLocaleString()}
                            </Typography>
                        </Stack>
                        
                        {showDetails && (
                            <TableContainer>
                                <Table size="small">
                                    <TableHead>
                                        <TableRow>
                                            <TableCell>Service</TableCell>
                                            <TableCell>Status</TableCell>
                                            <TableCell>Health</TableCell>
                                            <TableCell>Last Check</TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {renderServiceStatus("API", healthData.services.api)}
                                        {renderServiceStatus("Database", healthData.services.database)}
                                        {renderServiceStatus("Redis", healthData.services.redis)}
                                        {renderServiceStatus("WebSocket", healthData.services.websocket)}
                                        {renderServiceStatus("Bus", healthData.services.bus)}
                                        {renderServiceStatus("Memory", healthData.services.memory)}
                                        {renderServiceStatus("System", healthData.services.system)}
                                    </TableBody>
                                </Table>
                            </TableContainer>
                        )}
                    </>
                )}
            </CardContent>
        </Card>
    );
}
