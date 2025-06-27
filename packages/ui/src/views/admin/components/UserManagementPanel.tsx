import { IconCommon } from "../../../icons/Icons.js";
import {
    Box,
    Button,
    Card,
    CardContent,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
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
} from "@mui/material";
import { Tooltip } from "../../../components/Tooltip/Tooltip.js";
import { DialogTitle } from "../../../components/dialogs/DialogTitle/DialogTitle.js";
import { AccountStatus } from "@vrooli/shared";
import React, { useState, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";

// Mock data and types - replace with actual API calls
interface AdminUser {
    id: string;
    name?: string;
    email?: string;
    status?: AccountStatus;
    createdAt: string;
    lastActiveAt?: string;
    apiKeyCount: number;
    routineCount: number;
    teamCount: number;
}

interface UserListResponse {
    users: AdminUser[];
    totalCount: number;
}

/**
 * User management panel for administrators
 * Allows viewing, searching, and managing user accounts
 */
export const UserManagementPanel: React.FC = () => {
    const { t } = useTranslation();
    
    // State management
    const [users, setUsers] = useState<AdminUser[]>([]);
    const [loading, setLoading] = useState(false);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(0);
    const [rowsPerPage, setRowsPerPage] = useState(25);
    const [searchTerm, setSearchTerm] = useState("");
    const [statusFilter, setStatusFilter] = useState<AccountStatus | "">("");
    const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
    const [statusDialog, setStatusDialog] = useState(false);
    const [newStatus, setNewStatus] = useState<AccountStatus>(AccountStatus.Unlocked);
    const [statusReason, setStatusReason] = useState("");
    const [confirmDialog, setConfirmDialog] = useState<{
        open: boolean;
        title: string;
        message: string;
        action: () => void;
    }>({
        open: false,
        title: "",
        message: "",
        action: () => {},
    });

    // Mock API calls - replace with actual implementations
    const fetchUsers = useCallback(async () => {
        setLoading(true);
        try {
            // TODO: Replace with actual API call
            const mockResponse: UserListResponse = {
                users: [
                    {
                        id: "1",
                        name: "John Doe",
                        email: "john@example.com",
                        status: AccountStatus.Unlocked,
                        createdAt: new Date(Date.now() - 86400000).toISOString(),
                        lastActiveAt: new Date().toISOString(),
                        apiKeyCount: 2,
                        routineCount: 15,
                        teamCount: 3,
                    },
                    {
                        id: "2",
                        name: "Jane Smith",
                        email: "jane@example.com",
                        status: AccountStatus.SoftLocked,
                        createdAt: new Date(Date.now() - 172800000).toISOString(),
                        lastActiveAt: new Date(Date.now() - 3600000).toISOString(),
                        apiKeyCount: 1,
                        routineCount: 8,
                        teamCount: 1,
                    },
                    {
                        id: "3",
                        name: "Bob Wilson",
                        email: "bob@example.com",
                        status: AccountStatus.HardLocked,
                        createdAt: new Date(Date.now() - 259200000).toISOString(),
                        lastActiveAt: new Date(Date.now() - 86400000).toISOString(),
                        apiKeyCount: 0,
                        routineCount: 3,
                        teamCount: 0,
                    },
                ],
                totalCount: 3,
            };

            setUsers(mockResponse.users);
            setTotalCount(mockResponse.totalCount);
        } catch (error) {
            console.error("Failed to fetch users:", error);
        } finally {
            setLoading(false);
        }
    }, [page, rowsPerPage, searchTerm, statusFilter]);

    useEffect(() => {
        fetchUsers();
    }, [fetchUsers]);

    const handleStatusChange = async (userId: string, status: AccountStatus, reason: string) => {
        try {
            // TODO: Replace with actual API call
            console.log("Updating user status:", { userId, status, reason });
            
            // Update local state
            setUsers(prev => prev.map(user => 
                user.id === userId ? { ...user, status } : user,
            ));
            
            setStatusDialog(false);
            setSelectedUser(null);
            setStatusReason("");
        } catch (error) {
            console.error("Failed to update user status:", error);
        }
    };

    const handlePasswordReset = async (userId: string) => {
        try {
            // TODO: Replace with actual API call
            console.log("Resetting password for user:", userId);
            // Show success message
        } catch (error) {
            console.error("Failed to reset password:", error);
        }
    };

    const getStatusColor = (status: AccountStatus) => {
        switch (status) {
            case AccountStatus.Unlocked: return "success";
            case AccountStatus.SoftLocked: return "warning";
            case AccountStatus.HardLocked: return "error";
            case AccountStatus.Deleted: return "default";
            default: return "default";
        }
    };

    const getStatusIcon = (status: AccountStatus) => {
        switch (status) {
            case AccountStatus.Unlocked: return <IconCommon name="Success" />;
            case AccountStatus.SoftLocked: return <IconCommon name="Warning" />;
            case AccountStatus.HardLocked: return <IconCommon name="Close" />;
            case AccountStatus.Deleted: return <IconCommon name="Delete" />;
            default: return <IconCommon name="User" />;
        }
    };

    const openConfirmDialog = (title: string, message: string, action: () => void) => {
        setConfirmDialog({
            open: true,
            title,
            message,
            action,
        });
    };

    return (
        <Box>
            {/* Header */}
            <Typography variant="h5" gutterBottom>
                {t("UserManagement")}
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
                {t("UserManagementDescription")}
            </Typography>

            {/* Summary Cards */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">{totalCount}</Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("TotalUsers")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {users.filter(u => u.status === AccountStatus.Unlocked).length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("ActiveUsers")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {users.filter(u => u.status === AccountStatus.SoftLocked).length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("SoftLockedUsers")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={6} md={3}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {users.filter(u => u.status === AccountStatus.HardLocked).length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("HardLockedUsers")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* Search and Filters */}
            <Paper sx={{ p: 2, mb: 3 }}>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2} alignItems="center">
                    <TextField
                        fullWidth
                        placeholder={t("SearchUsers")}
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        InputProps={{
                            startAdornment: <IconCommon name="Search" sx={{ mr: 1, color: "text.secondary" }} />,
                        }}
                        sx={{ maxWidth: 300 }}
                    />
                    <FormControl sx={{ minWidth: 200 }}>
                        <InputLabel>{t("Status")}</InputLabel>
                        <Select
                            value={statusFilter}
                            onChange={(e) => setStatusFilter(e.target.value as AccountStatus | "")}
                        >
                            <MenuItem value="">{t("AllStatuses")}</MenuItem>
                            {Object.values(AccountStatus).map(status => (
                                <MenuItem key={status} value={status}>
                                    {t(status)}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <Button variant="outlined" onClick={() => fetchUsers()}>
                        {t("Refresh")}
                    </Button>
                </Stack>
            </Paper>

            {/* Users Table */}
            <TableContainer component={Paper}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>{t("User")}</TableCell>
                            <TableCell>{t("Email")}</TableCell>
                            <TableCell>{t("Status")}</TableCell>
                            <TableCell>{t("Created")}</TableCell>
                            <TableCell>{t("LastActive")}</TableCell>
                            <TableCell>{t("APIKeys")}</TableCell>
                            <TableCell>{t("Routines")}</TableCell>
                            <TableCell>{t("Actions")}</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {users.map((user) => (
                            <TableRow key={user.id}>
                                <TableCell>
                                    <Stack direction="row" spacing={1} alignItems="center">
                                        {getStatusIcon(user.status || AccountStatus.Unlocked)}
                                        <Typography variant="body2">
                                            {user.name || `User ${user.id}`}
                                        </Typography>
                                    </Stack>
                                </TableCell>
                                <TableCell>{user.email}</TableCell>
                                <TableCell>
                                    <Chip
                                        label={user.status || AccountStatus.Unlocked}
                                        color={getStatusColor(user.status || AccountStatus.Unlocked)}
                                        size="small"
                                    />
                                </TableCell>
                                <TableCell>
                                    {new Date(user.createdAt).toLocaleDateString()}
                                </TableCell>
                                <TableCell>
                                    {user.lastActiveAt ? 
                                        new Date(user.lastActiveAt).toLocaleDateString() : 
                                        t("Never")
                                    }
                                </TableCell>
                                <TableCell>{user.apiKeyCount}</TableCell>
                                <TableCell>{user.routineCount}</TableCell>
                                <TableCell>
                                    <Stack direction="row" spacing={1}>
                                        <Tooltip title={t("ChangeStatus")}>
                                            <IconButton
                                                size="small"
                                                onClick={() => {
                                                    setSelectedUser(user);
                                                    setNewStatus(user.status || AccountStatus.Unlocked);
                                                    setStatusDialog(true);
                                                }}
                                            >
                                                <IconCommon name="Edit" />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t("ResetPassword")}>
                                            <IconButton
                                                size="small"
                                                color="warning"
                                                onClick={() => {
                                                    openConfirmDialog(
                                                        t("ResetPassword"),
                                                        t("ResetPasswordConfirm", { user: user.name || user.email }),
                                                        () => handlePasswordReset(user.id),
                                                    );
                                                }}
                                            >
                                                <IconCommon name="Lock" />
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

            {/* Status Change Dialog */}
            <Dialog open={statusDialog} onClose={() => setStatusDialog(false)} maxWidth="sm" fullWidth>
                <DialogTitle id="change-user-status-dialog" title={t("ChangeUserStatus")} />
                <DialogContent>
                    {selectedUser && (
                        <>
                            <Alert severity="warning" sx={{ mb: 2 }}>
                                {t("ChangeStatusWarning")}
                            </Alert>
                            <Typography variant="subtitle1" sx={{ mb: 2 }}>
                                {t("User")}: {selectedUser.name || selectedUser.email}
                            </Typography>
                            <FormControl fullWidth sx={{ mb: 2 }}>
                                <InputLabel>{t("NewStatus")}</InputLabel>
                                <Select
                                    value={newStatus}
                                    onChange={(e) => setNewStatus(e.target.value as AccountStatus)}
                                >
                                    {Object.values(AccountStatus).map(status => (
                                        <MenuItem key={status} value={status}>
                                            <Stack direction="row" spacing={1} alignItems="center">
                                                {getStatusIcon(status)}
                                                <Typography>{t(status)}</Typography>
                                            </Stack>
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                            <TextField
                                fullWidth
                                multiline
                                rows={3}
                                label={t("ReasonForChange")}
                                value={statusReason}
                                onChange={(e) => setStatusReason(e.target.value)}
                                helperText={t("ReasonForChangeHelp")}
                            />
                        </>
                    )}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setStatusDialog(false)}>
                        {t("Cancel")}
                    </Button>
                    <Button
                        onClick={() => {
                            if (selectedUser) {
                                handleStatusChange(selectedUser.id, newStatus, statusReason);
                            }
                        }}
                        variant="contained"
                        color="primary"
                    >
                        {t("UpdateStatus")}
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Confirmation Dialog */}
            <Dialog open={confirmDialog.open} onClose={() => setConfirmDialog(prev => ({ ...prev, open: false }))}>
                <DialogTitle id="confirm-dialog" title={confirmDialog.title} />
                <DialogContent>
                    <Typography>{confirmDialog.message}</Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setConfirmDialog(prev => ({ ...prev, open: false }))}>
                        {t("Cancel")}
                    </Button>
                    <Button
                        onClick={() => {
                            confirmDialog.action();
                            setConfirmDialog(prev => ({ ...prev, open: false }));
                        }}
                        variant="contained"
                        color="warning"
                    >
                        {t("Confirm")}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
};
