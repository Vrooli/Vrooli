import {
    Flag,
    Warning,
    CheckCircle,
    Block,
    Visibility,
    VisibilityOff,
    Delete,
    More,
} from "@mui/icons-material";
import {
    Box,
    Button,
    Card,
    CardContent,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    Grid,
    IconButton,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TablePagination,
    TableRow,
    TextField,
    Typography,
    Alert,
    Stack,
    Tooltip,
    Menu,
    ListItemIcon,
    ListItemText,
} from "@mui/material";
import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";

// Report types and interfaces
interface ContentReport {
    id: string;
    reportedContent: {
        id: string;
        type: "Routine" | "Comment" | "User" | "Team" | "Project";
        title?: string;
        author?: string;
    };
    reporter: {
        id: string;
        name?: string;
        email?: string;
    };
    reason: string;
    category: "Spam" | "Inappropriate" | "Copyright" | "Violence" | "Harassment" | "Other";
    status: "Pending" | "Reviewing" | "Resolved" | "Dismissed";
    severity: "Low" | "Medium" | "High" | "Critical";
    createdAt: string;
    reviewedAt?: string;
    reviewedBy?: string;
    resolution?: string;
}

interface ReportListResponse {
    reports: ContentReport[];
    totalCount: number;
}

/**
 * Content moderation and reports management panel for administrators
 * Handles user reports, content moderation, and platform safety
 */
export const ReportsPanel: React.FC = () => {
    const { t } = useTranslation();
    
    // State management
    const [reports, setReports] = useState<ContentReport[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(0);
    const [rowsPerPage, setRowsPerPage] = useState(25);
    const [statusFilter, setStatusFilter] = useState<string>("");
    const [categoryFilter, setCategoryFilter] = useState<string>("");
    const [severityFilter, setSeverityFilter] = useState<string>("");
    const [selectedReport, setSelectedReport] = useState<ContentReport | null>(null);
    const [reviewDialog, setReviewDialog] = useState(false);
    const [resolution, setResolution] = useState("");
    const [actionMenu, setActionMenu] = useState<{
        anchorEl: HTMLElement | null;
        report: ContentReport | null;
    }>({
        anchorEl: null,
        report: null,
    });

    // Mock API calls - replace with actual implementations
    const fetchReports = useCallback(async () => {
        setLoading(true);
        try {
            // TODO: Replace with actual API call
            const mockResponse: ReportListResponse = {
                reports: [
                    {
                        id: "1",
                        reportedContent: {
                            id: "content1",
                            type: "Routine",
                            title: "Suspicious AI Routine",
                            author: "user123",
                        },
                        reporter: {
                            id: "reporter1",
                            name: "John Doe",
                            email: "john@example.com",
                        },
                        reason: "This routine seems to be designed for malicious purposes",
                        category: "Inappropriate",
                        status: "Pending",
                        severity: "High",
                        createdAt: new Date().toISOString(),
                    },
                    {
                        id: "2",
                        reportedContent: {
                            id: "content2",
                            type: "Comment",
                            title: "Offensive comment on Project Alpha",
                            author: "user456",
                        },
                        reporter: {
                            id: "reporter2",
                            name: "Jane Smith",
                            email: "jane@example.com",
                        },
                        reason: "Contains hate speech and harassment",
                        category: "Harassment",
                        status: "Reviewing",
                        severity: "Critical",
                        createdAt: new Date(Date.now() - 3600000).toISOString(),
                        reviewedAt: new Date().toISOString(),
                        reviewedBy: "admin1",
                    },
                    {
                        id: "3",
                        reportedContent: {
                            id: "content3",
                            type: "User",
                            title: "Spam Account",
                            author: "spammer789",
                        },
                        reporter: {
                            id: "reporter3",
                            name: "Bob Wilson",
                            email: "bob@example.com",
                        },
                        reason: "Account is posting spam across multiple projects",
                        category: "Spam",
                        status: "Resolved",
                        severity: "Medium",
                        createdAt: new Date(Date.now() - 86400000).toISOString(),
                        reviewedAt: new Date(Date.now() - 3600000).toISOString(),
                        reviewedBy: "admin2",
                        resolution: "User account suspended for 7 days",
                    },
                ],
                totalCount: 3,
            };

            setReports(mockResponse.reports);
            setTotalCount(mockResponse.totalCount);
        } catch (error) {
            console.error("Failed to fetch reports:", error);
        } finally {
            setLoading(false);
        }
    }, [page, rowsPerPage, statusFilter, categoryFilter, severityFilter]);

    useEffect(() => {
        fetchReports();
    }, [fetchReports]);

    const handleReportReview = async (reportId: string, status: string, resolution: string) => {
        try {
            // TODO: Replace with actual API call
            console.log("Reviewing report:", { reportId, status, resolution });
            
            // Update local state
            setReports(prev => prev.map(report => 
                report.id === reportId 
                    ? { 
                        ...report, 
                        status: status as any,
                        resolution,
                        reviewedAt: new Date().toISOString(),
                        reviewedBy: "current_admin",
                    }
                    : report
            ));
            
            setReviewDialog(false);
            setSelectedReport(null);
            setResolution("");
        } catch (error) {
            console.error("Failed to review report:", error);
        }
    };

    const handleContentAction = async (action: string, reportId: string) => {
        try {
            // TODO: Replace with actual API call
            console.log("Content action:", { action, reportId });
            
            // Update report status based on action
            const newStatus = action === "hide" || action === "delete" ? "Resolved" : "Reviewing";
            setReports(prev => prev.map(report => 
                report.id === reportId 
                    ? { ...report, status: newStatus as any }
                    : report
            ));
        } catch (error) {
            console.error("Failed to perform content action:", error);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case "Pending": return "warning";
            case "Reviewing": return "info";
            case "Resolved": return "success";
            case "Dismissed": return "default";
            default: return "default";
        }
    };

    const getSeverityColor = (severity: string) => {
        switch (severity) {
            case "Critical": return "error";
            case "High": return "warning";
            case "Medium": return "info";
            case "Low": return "default";
            default: return "default";
        }
    };

    const getSeverityIcon = (severity: string) => {
        switch (severity) {
            case "Critical": return <Flag color="error" />;
            case "High": return <Warning color="warning" />;
            case "Medium": return <Warning color="info" />;
            case "Low": return <Flag color="disabled" />;
            default: return <Flag />;
        }
    };

    return (
        <Box>
            {/* Header */}
            <Typography variant="h5" gutterBottom>
                {t("ReportsModeration")}
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
                {t("ReportsModerationDescription")}
            </Typography>

            {/* Summary Cards */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {reports.filter(r => r.status === "Pending").length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("PendingReports")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {reports.filter(r => r.status === "Reviewing").length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("UnderReview")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {reports.filter(r => r.severity === "Critical" || r.severity === "High").length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("HighPriorityReports")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {reports.filter(r => r.status === "Resolved").length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("ResolvedToday")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* Filters */}
            <Paper sx={{ p: 2, mb: 3 }}>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2} alignItems="center">
                    <FormControl sx={{ minWidth: 150 }}>
                        <InputLabel>{t("Status")}</InputLabel>
                        <Select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value)}
                        >
                            <MenuItem value="">{t("AllStatuses")}</MenuItem>
                            <MenuItem value="Pending">{t("Pending")}</MenuItem>
                            <MenuItem value="Reviewing">{t("Reviewing")}</MenuItem>
                            <MenuItem value="Resolved">{t("Resolved")}</MenuItem>
                            <MenuItem value="Dismissed">{t("Dismissed")}</MenuItem>
                        </Select>
                    </FormControl>
                    <FormControl sx={{ minWidth: 150 }}>
                        <InputLabel>{t("Category")}</InputLabel>
                        <Select
                            value={categoryFilter}
                            onChange={(e) => setCategoryFilter(e.target.value)}
                        >
                            <MenuItem value="">{t("AllCategories")}</MenuItem>
                            <MenuItem value="Spam">{t("Spam")}</MenuItem>
                            <MenuItem value="Inappropriate">{t("Inappropriate")}</MenuItem>
                            <MenuItem value="Harassment">{t("Harassment")}</MenuItem>
                            <MenuItem value="Violence">{t("Violence")}</MenuItem>
                            <MenuItem value="Copyright">{t("Copyright")}</MenuItem>
                            <MenuItem value="Other">{t("Other")}</MenuItem>
                        </Select>
                    </FormControl>
                    <FormControl sx={{ minWidth: 150 }}>
                        <InputLabel>{t("Severity")}</InputLabel>
                        <Select
                            value={severityFilter}
                            onChange={(e) => setSeverityFilter(e.target.value)}
                        >
                            <MenuItem value="">{t("AllSeverities")}</MenuItem>
                            <MenuItem value="Critical">{t("Critical")}</MenuItem>
                            <MenuItem value="High">{t("High")}</MenuItem>
                            <MenuItem value="Medium">{t("Medium")}</MenuItem>
                            <MenuItem value="Low">{t("Low")}</MenuItem>
                        </Select>
                    </FormControl>
                    <Button variant="outlined" onClick={() => fetchReports()}>
                        {t("Refresh")}
                    </Button>
                </Stack>
            </Paper>

            {/* Reports Table */}
            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>{t("Severity")}</TableCell>
                            <TableCell>{t("Content")}</TableCell>
                            <TableCell>{t("Category")}</TableCell>
                            <TableCell>{t("Reporter")}</TableCell>
                            <TableCell>{t("Status")}</TableCell>
                            <TableCell>{t("Created")}</TableCell>
                            <TableCell>{t("Actions")}</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {reports.map((report) => (
                            <TableRow key={report.id}>
                                <TableCell>
                                    <Stack direction="row" spacing={1} alignItems="center">
                                        {getSeverityIcon(report.severity)}
                                        <Chip
                                            label={report.severity}
                                            color={getSeverityColor(report.severity)}
                                            size="small"
                                        />
                                    </Stack>
                                </TableCell>
                                <TableCell>
                                    <Box>
                                        <Typography variant="body2" fontWeight="bold">
                                            {report.reportedContent.title}
                                        </Typography>
                                        <Typography variant="caption" color="text.secondary">
                                            {report.reportedContent.type} by {report.reportedContent.author}
                                        </Typography>
                                    </Box>
                                </TableCell>
                                <TableCell>
                                    <Chip 
                                        label={report.category} 
                                        size="small" 
                                        variant="outlined"
                                    />
                                </TableCell>
                                <TableCell>
                                    <Typography variant="body2">
                                        {report.reporter.name || report.reporter.email}
                                    </Typography>
                                </TableCell>
                                <TableCell>
                                    <Chip
                                        label={report.status}
                                        color={getStatusColor(report.status)}
                                        size="small"
                                    />
                                </TableCell>
                                <TableCell>
                                    {new Date(report.createdAt).toLocaleDateString()}
                                </TableCell>
                                <TableCell>
                                    <Stack direction="row" spacing={1}>
                                        <Tooltip title={t("ReviewReport")}>
                                            <IconButton
                                                size="small"
                                                onClick={() => {
                                                    setSelectedReport(report);
                                                    setReviewDialog(true);
                                                }}
                                                disabled={report.status === "Resolved" || report.status === "Dismissed"}
                                            >
                                                <CheckCircle />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t("MoreActions")}>
                                            <IconButton
                                                size="small"
                                                onClick={(event) => {
                                                    setActionMenu({
                                                        anchorEl: event.currentTarget,
                                                        report,
                                                    });
                                                }}
                                            >
                                                <More />
                                            </IconButton>
                                        </Tooltip>
                                    </Stack>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
                <TablePagination
                    component="div"
                    count={totalCount}
                    page={page}
                    onPageChange={(event, newPage) => setPage(newPage)}
                    rowsPerPage={rowsPerPage}
                    onRowsPerPageChange={(event) => {
                        setRowsPerPage(parseInt(event.target.value, 10));
                        setPage(0);
                    }}
                />
            </TableContainer>

            {/* Review Dialog */}
            <Dialog open={reviewDialog} onClose={() => setReviewDialog(false)} maxWidth="md" fullWidth>
                <DialogTitle>{t("ReviewReport")}</DialogTitle>
                <DialogContent>
                    {selectedReport && (
                        <>
                            <Alert severity="info" sx={{ mb: 2 }}>
                                {t("ReviewReportInfo")}
                            </Alert>
                            
                            <Box sx={{ mb: 3 }}>
                                <Typography variant="subtitle1" gutterBottom>
                                    {t("ReportedContent")}:
                                </Typography>
                                <Typography variant="body2">
                                    {selectedReport.reportedContent.title} ({selectedReport.reportedContent.type})
                                </Typography>
                                <Typography variant="caption" color="text.secondary">
                                    by {selectedReport.reportedContent.author}
                                </Typography>
                            </Box>

                            <Box sx={{ mb: 3 }}>
                                <Typography variant="subtitle1" gutterBottom>
                                    {t("ReportReason")}:
                                </Typography>
                                <Typography variant="body2">
                                    {selectedReport.reason}
                                </Typography>
                            </Box>

                            <TextField
                                fullWidth
                                multiline
                                rows={4}
                                label={t("Resolution")}
                                value={resolution}
                                onChange={(e) => setResolution(e.target.value)}
                                helperText={t("ResolutionHelp")}
                            />
                        </>
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setReviewDialog(false)}>
                        {t("Cancel")}
                    </Button>
                    <Button
                        onClick={() => {
                            if (selectedReport) {
                                handleReportReview(selectedReport.id, "Dismissed", resolution);
                            }
                        }}
                        color="inherit"
                    >
                        {t("Dismiss")}
                    </Button>
                    <Button
                        onClick={() => {
                            if (selectedReport) {
                                handleReportReview(selectedReport.id, "Resolved", resolution);
                            }
                        }}
                        variant="contained"
                        color="primary"
                    >
                        {t("Resolve")}
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Action Menu */}
            <Menu
                anchorEl={actionMenu.anchorEl}
                open={Boolean(actionMenu.anchorEl)}
                onClose={() => setActionMenu({ anchorEl: null, report: null })}
            >
                <MenuItem
                    onClick={() => {
                        if (actionMenu.report) {
                            handleContentAction("hide", actionMenu.report.id);
                        }
                        setActionMenu({ anchorEl: null, report: null });
                    }}
                >
                    <ListItemIcon>
                        <VisibilityOff />
                    </ListItemIcon>
                    <ListItemText primary={t("HideContent")} />
                </MenuItem>
                <MenuItem
                    onClick={() => {
                        if (actionMenu.report) {
                            handleContentAction("delete", actionMenu.report.id);
                        }
                        setActionMenu({ anchorEl: null, report: null });
                    }}
                >
                    <ListItemIcon>
                        <Delete />
                    </ListItemIcon>
                    <ListItemText primary={t("DeleteContent")} />
                </MenuItem>
                <MenuItem
                    onClick={() => {
                        if (actionMenu.report) {
                            handleContentAction("block_user", actionMenu.report.id);
                        }
                        setActionMenu({ anchorEl: null, report: null });
                    }}
                >
                    <ListItemIcon>
                        <Block />
                    </ListItemIcon>
                    <ListItemText primary={t("BlockUser")} />
                </MenuItem>
            </Menu>
        </Box>
    );
};