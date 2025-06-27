import {
    Accordion,
    AccordionDetails,
    AccordionSummary,
    Alert,
    Box,
    Button,
    Chip,
    Grid,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
} from "@mui/material";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../../icons/Icons.js";
import { SystemHealthCard } from "../../../components/stats/SystemHealthCard.js";
import { SystemMetricsCards } from "../../../components/stats/SystemMetricsCards.js";
import { useSystemHealth } from "../../../hooks/useSystemHealth.js";

function StatusChip({ status }: { status: string }) {
    const getStatusColor = (status: string) => {
        switch (status) {
            case "Operational": return "success";
            case "Degraded": return "warning";
            case "Down": return "error";
            case "Active": return "success";
            case "Cooldown": return "warning";
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

export function HealthAndMetricsPanel() {
    const { t } = useTranslation();
    const { 
        healthData, 
        metricsData, 
        healthLoading, 
        metricsLoading, 
        healthError, 
        metricsError, 
        refresh, 
    } = useSystemHealth();

    return (
        <Box sx={{ p: 3 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
                <Typography variant="h5">System Health & Metrics</Typography>
                <Button
                    variant="outlined"
                    onClick={refresh}
                    disabled={healthLoading || metricsLoading}
                    startIcon={<IconCommon name="Refresh" size={16} />}
                >
                    Refresh
                </Button>
            </Stack>

            <Grid container spacing={3}>
                {/* System Metrics Cards */}
                {metricsData && (
                    <Grid item xs={12}>
                        <SystemMetricsCards 
                            metricsData={metricsData}
                            loading={metricsLoading}
                            error={metricsError}
                        />
                    </Grid>
                )}

                {/* System Health Card */}
                <Grid item xs={12}>
                    <SystemHealthCard
                        healthData={healthData}
                        loading={healthLoading}
                        error={healthError}
                        showDetails={true}
                    />
                </Grid>

                {/* Additional detailed service info */}
                {healthData && (
                    <>
                        {/* Queues */}
                        {healthData.services.queues && Object.keys(healthData.services.queues).length > 0 && (
                            <Grid item xs={12}>
                                <Accordion>
                                    <AccordionSummary expandIcon={<IconCommon name="ArrowDropDown" size={16} />}>
                                        <Typography variant="h6">Queue Health Details</Typography>
                                    </AccordionSummary>
                                    <AccordionDetails>
                                        <TableContainer>
                                            <Table size="small">
                                                <TableHead>
                                                    <TableRow>
                                                        <TableCell>Queue</TableCell>
                                                        <TableCell>Status</TableCell>
                                                        <TableCell>Health</TableCell>
                                                        <TableCell>Details</TableCell>
                                                    </TableRow>
                                                </TableHead>
                                                <TableBody>
                                                    {Object.entries(healthData.services.queues).map(([name, queue]) => (
                                                        <TableRow key={name}>
                                                            <TableCell>{name}</TableCell>
                                                            <TableCell>
                                                                <StatusChip status={queue.status} />
                                                            </TableCell>
                                                            <TableCell>
                                                                {queue.healthy ? (
                                                                    <IconCommon name="CheckCircle" fill="green" size={20} />
                                                                ) : (
                                                                    <IconCommon name="Error" fill="red" size={20} />
                                                                )}
                                                            </TableCell>
                                                            <TableCell>
                                                                {queue.details && typeof queue.details === "object" && (
                                                                    <Typography variant="body2" color="text.secondary">
                                                                        {JSON.stringify(queue.details.metrics || queue.details, null, 2)}
                                                                    </Typography>
                                                                )}
                                                            </TableCell>
                                                        </TableRow>
                                                    ))}
                                                </TableBody>
                                            </Table>
                                        </TableContainer>
                                    </AccordionDetails>
                                </Accordion>
                            </Grid>
                        )}

                        {/* LLM Services */}
                        {healthData.services.llm && Object.keys(healthData.services.llm).length > 0 && (
                            <Grid item xs={12}>
                                <Accordion>
                                    <AccordionSummary expandIcon={<IconCommon name="ArrowDropDown" size={16} />}>
                                        <Typography variant="h6">LLM Services Health</Typography>
                                    </AccordionSummary>
                                    <AccordionDetails>
                                        <TableContainer>
                                            <Table size="small">
                                                <TableHead>
                                                    <TableRow>
                                                        <TableCell>Service</TableCell>
                                                        <TableCell>Status</TableCell>
                                                        <TableCell>Health</TableCell>
                                                        <TableCell>Details</TableCell>
                                                    </TableRow>
                                                </TableHead>
                                                <TableBody>
                                                    {Object.entries(healthData.services.llm).map(([id, service]) => (
                                                        <TableRow key={id}>
                                                            <TableCell>{id}</TableCell>
                                                            <TableCell>
                                                                <StatusChip status={service.status} />
                                                            </TableCell>
                                                            <TableCell>
                                                                {service.healthy ? (
                                                                    <IconCommon name="CheckCircle" fill="green" size={20} />
                                                                ) : (
                                                                    <IconCommon name="Error" fill="red" size={20} />
                                                                )}
                                                            </TableCell>
                                                            <TableCell>
                                                                {service.details && (
                                                                    <Typography variant="body2" color="text.secondary">
                                                                        {service.details.state || "N/A"}
                                                                        {service.details.cooldownUntil && ` - Cooldown until: ${new Date(service.details.cooldownUntil).toLocaleString()}`}
                                                                    </Typography>
                                                                )}
                                                            </TableCell>
                                                        </TableRow>
                                                    ))}
                                                </TableBody>
                                            </Table>
                                        </TableContainer>
                                    </AccordionDetails>
                                </Accordion>
                            </Grid>
                        )}
                    </>
                )}

                {/* Queue Metrics */}
                {metricsData && Object.keys(metricsData.application.queues).length > 0 && (
                    <Grid item xs={12}>
                        <Accordion>
                            <AccordionSummary expandIcon={<IconCommon name="ArrowDropDown" size={16} />}>
                                <Typography variant="h6">Queue Metrics</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <TableContainer>
                                    <Table size="small">
                                        <TableHead>
                                            <TableRow>
                                                <TableCell>Queue</TableCell>
                                                <TableCell>Waiting</TableCell>
                                                <TableCell>Active</TableCell>
                                                <TableCell>Completed</TableCell>
                                                <TableCell>Failed</TableCell>
                                                <TableCell>Total</TableCell>
                                            </TableRow>
                                        </TableHead>
                                        <TableBody>
                                            {Object.entries(metricsData.application.queues).map(([name, queue]) => (
                                                <TableRow key={name}>
                                                    <TableCell>{name}</TableCell>
                                                    <TableCell>{queue.waiting}</TableCell>
                                                    <TableCell>{queue.active}</TableCell>
                                                    <TableCell>{queue.completed}</TableCell>
                                                    <TableCell>{queue.failed}</TableCell>
                                                    <TableCell>{queue.total}</TableCell>
                                                </TableRow>
                                            ))}
                                        </TableBody>
                                    </Table>
                                </TableContainer>
                            </AccordionDetails>
                        </Accordion>
                    </Grid>
                )}

                {/* LLM Service Metrics */}
                {metricsData && Object.keys(metricsData.application.llmServices).length > 0 && (
                    <Grid item xs={12}>
                        <Accordion>
                            <AccordionSummary expandIcon={<IconCommon name="ArrowDropDown" size={16} />}>
                                <Typography variant="h6">LLM Service Metrics</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <TableContainer>
                                    <Table size="small">
                                        <TableHead>
                                            <TableRow>
                                                <TableCell>Service</TableCell>
                                                <TableCell>State</TableCell>
                                                <TableCell>Cooldown Until</TableCell>
                                            </TableRow>
                                        </TableHead>
                                        <TableBody>
                                            {Object.entries(metricsData.application.llmServices).map(([id, service]) => (
                                                <TableRow key={id}>
                                                    <TableCell>{id}</TableCell>
                                                    <TableCell>
                                                        <StatusChip status={service.state} />
                                                    </TableCell>
                                                    <TableCell>
                                                        {service.cooldownUntil ? new Date(service.cooldownUntil).toLocaleString() : "N/A"}
                                                    </TableCell>
                                                </TableRow>
                                            ))}
                                        </TableBody>
                                    </Table>
                                </TableContainer>
                            </AccordionDetails>
                        </Accordion>
                    </Grid>
                )}

                {metricsError && (
                    <Grid item xs={12}>
                        <Alert severity="error">
                            {metricsError}
                        </Alert>
                    </Grid>
                )}
            </Grid>
        </Box>
    );
}
