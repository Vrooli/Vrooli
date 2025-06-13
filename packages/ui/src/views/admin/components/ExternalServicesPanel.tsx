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
    DialogTitle,
    FormControl,
    Grid,
    IconButton,
    InputLabel,
    MenuItem,
    Select,
    TextField,
    Typography,
    Alert,
    Stack,
    List,
    ListItem,
    ListItemText,
    Tooltip,
} from "@mui/material";
import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";

// Service provider types
interface ExternalServiceProvider {
    id: string;
    name: string;
    identifier: string;
    type: "OAuth" | "ApiKey" | "Hybrid";
    status: "Active" | "Disabled" | "Maintenance";
    description?: string;
    oauthClientId?: string;
    baseUrl?: string;
    supportedOperations?: string[];
    userCount?: number;
    lastTested?: string;
    testStatus?: "success" | "error" | "warning";
}

/**
 * Panel for managing external service providers and integrations
 * Allows admins to add new services without code changes
 */
export const ExternalServicesPanel: React.FC = () => {
    const { t } = useTranslation();
    
    // State management
    const [providers, setProviders] = useState<ExternalServiceProvider[]>([]);
    const [loading, setLoading] = useState(false);
    const [addDialog, setAddDialog] = useState(false);
    const [editProvider, setEditProvider] = useState<ExternalServiceProvider | null>(null);
    const [testDialog, setTestDialog] = useState<{
        open: boolean;
        provider: ExternalServiceProvider | null;
        testing: boolean;
        results: any;
    }>({
        open: false,
        provider: null,
        testing: false,
        results: null,
    });

    const [formData, setFormData] = useState({
        name: "",
        identifier: "",
        type: "ApiKey" as const,
        description: "",
        oauthClientId: "",
        oauthClientSecret: "",
        oauthScopes: "",
        baseUrl: "",
        authMethod: "header",
        supportedOperations: "",
    });

    // Mock data - replace with actual API calls
    useEffect(() => {
        const mockProviders: ExternalServiceProvider[] = [
            {
                id: "1",
                name: "OpenAI",
                identifier: "openai",
                type: "ApiKey",
                status: "Active",
                description: "OpenAI GPT models for AI-powered routines",
                baseUrl: "https://api.openai.com/v1",
                supportedOperations: ["chat", "completion", "embedding"],
                userCount: 45,
                lastTested: new Date().toISOString(),
                testStatus: "success",
            },
            {
                id: "2",
                name: "GitHub",
                identifier: "github",
                type: "Hybrid",
                status: "Active",
                description: "GitHub integration for repository management",
                oauthClientId: "github_client_id",
                baseUrl: "https://api.github.com",
                supportedOperations: ["repos", "issues", "pulls"],
                userCount: 23,
                lastTested: new Date(Date.now() - 3600000).toISOString(),
                testStatus: "success",
            },
            {
                id: "3",
                name: "Google Drive",
                identifier: "google_drive",
                type: "OAuth",
                status: "Active",
                description: "Google Drive for file storage and sharing",
                oauthClientId: "google_client_id",
                supportedOperations: ["files", "folders", "sharing"],
                userCount: 12,
                lastTested: new Date(Date.now() - 86400000).toISOString(),
                testStatus: "warning",
            },
        ];
        setProviders(mockProviders);
    }, []);

    const handleSubmit = async () => {
        try {
            // TODO: Replace with actual API call
            const newProvider: ExternalServiceProvider = {
                id: Date.now().toString(),
                ...formData,
                status: "Active",
                supportedOperations: formData.supportedOperations.split(",").map(s => s.trim()),
                userCount: 0,
            };

            if (editProvider) {
                setProviders(prev => prev.map(p => p.id === editProvider.id ? { ...newProvider, id: editProvider.id } : p));
            } else {
                setProviders(prev => [...prev, newProvider]);
            }

            setAddDialog(false);
            setEditProvider(null);
            setFormData({
                name: "",
                identifier: "",
                type: "ApiKey",
                description: "",
                oauthClientId: "",
                oauthClientSecret: "",
                oauthScopes: "",
                baseUrl: "",
                authMethod: "header",
                supportedOperations: "",
            });
        } catch (error) {
            console.error("Failed to save provider:", error);
        }
    };

    const handleTest = async (provider: ExternalServiceProvider) => {
        setTestDialog({
            open: true,
            provider,
            testing: true,
            results: null,
        });

        try {
            // TODO: Replace with actual API call
            await new Promise(resolve => setTimeout(resolve, 2000)); // Mock delay
            
            const mockResults = {
                success: Math.random() > 0.3,
                connectivity: Math.random() > 0.2,
                authentication: Math.random() > 0.1,
                operations: provider.supportedOperations?.map(op => ({
                    name: op,
                    status: Math.random() > 0.2 ? "success" : "error",
                })) || [],
            };

            setTestDialog(prev => ({
                ...prev,
                testing: false,
                results: mockResults,
            }));

            // Update provider test status
            setProviders(prev => prev.map(p => 
                p.id === provider.id 
                    ? { 
                        ...p, 
                        lastTested: new Date().toISOString(),
                        testStatus: mockResults.success ? "success" : "error"
                    }
                    : p
            ));
        } catch (error) {
            setTestDialog(prev => ({
                ...prev,
                testing: false,
                results: { success: false, error: error.message },
            }));
        }
    };

    const handleDelete = async (providerId: string) => {
        try {
            // TODO: Replace with actual API call
            setProviders(prev => prev.filter(p => p.id !== providerId));
        } catch (error) {
            console.error("Failed to delete provider:", error);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case "Active": return "success";
            case "Disabled": return "error";
            case "Maintenance": return "warning";
            default: return "default";
        }
    };

    const getTestStatusIcon = (status?: string) => {
        switch (status) {
            case "success": return <IconCommon name="Success" fill="success.main" />;
            case "error": return <IconCommon name="Error" fill="error.main" />;
            case "warning": return <IconCommon name="Warning" fill="warning.main" />;
            default: return null;
        }
    };

    return (
        <Box>
            {/* Header */}
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
                <Box>
                    <Typography variant="h5" gutterBottom>
                        {t("ExternalServices")}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        {t("ExternalServicesDescription")}
                    </Typography>
                </Box>
                <Button
                    variant="contained"
                    startIcon={<IconCommon name="Add" />}
                    onClick={() => setAddDialog(true)}
                >
                    {t("AddService")}
                </Button>
            </Box>

            {/* Summary */}
            <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12} sm={4}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">{providers.length}</Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("TotalServices")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={4}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {providers.filter(p => p.status === "Active").length}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("ActiveServices")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} sm={4}>
                    <Card>
                        <CardContent>
                            <Typography variant="h6">
                                {providers.reduce((sum, p) => sum + (p.userCount || 0), 0)}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t("TotalIntegrations")}
                            </Typography>
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>

            {/* Service Cards */}
            <Grid container spacing={3}>
                {providers.map((provider) => (
                    <Grid item xs={12} md={6} lg={4} key={provider.id}>
                        <Card>
                            <CardContent>
                                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", mb: 2 }}>
                                    <Box>
                                        <Typography variant="h6">{provider.name}</Typography>
                                        <Typography variant="body2" color="text.secondary">
                                            {provider.identifier}
                                        </Typography>
                                        <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                                            <Chip 
                                                label={provider.type} 
                                                size="small" 
                                                variant="outlined"
                                            />
                                            <Chip 
                                                label={provider.status} 
                                                size="small" 
                                                color={getStatusColor(provider.status)}
                                            />
                                        </Stack>
                                    </Box>
                                    <Box>
                                        <Tooltip title={t("TestConnection")}>
                                            <IconButton 
                                                size="small"
                                                onClick={() => handleTest(provider)}
                                            >
                                                <IconCommon name="External" />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t("EditService")}>
                                            <IconButton 
                                                size="small"
                                                onClick={() => {
                                                    setEditProvider(provider);
                                                    setFormData({
                                                        name: provider.name,
                                                        identifier: provider.identifier,
                                                        type: provider.type,
                                                        description: provider.description || "",
                                                        oauthClientId: provider.oauthClientId || "",
                                                        oauthClientSecret: "",
                                                        oauthScopes: "",
                                                        baseUrl: provider.baseUrl || "",
                                                        authMethod: "header",
                                                        supportedOperations: provider.supportedOperations?.join(", ") || "",
                                                    });
                                                    setAddDialog(true);
                                                }}
                                            >
                                                <IconCommon name="Edit" />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t("DeleteService")}>
                                            <IconButton 
                                                size="small" 
                                                color="error"
                                                onClick={() => handleDelete(provider.id)}
                                            >
                                                <IconCommon name="Delete" />
                                            </IconButton>
                                        </Tooltip>
                                    </Box>
                                </Box>

                                <Typography variant="body2" sx={{ mb: 2 }}>
                                    {provider.description}
                                </Typography>

                                {provider.supportedOperations && (
                                    <Box sx={{ mb: 2 }}>
                                        <Typography variant="caption" color="text.secondary">
                                            {t("SupportedOperations")}:
                                        </Typography>
                                        <Box sx={{ mt: 0.5 }}>
                                            {provider.supportedOperations.slice(0, 3).map((op) => (
                                                <Chip 
                                                    key={op} 
                                                    label={op} 
                                                    size="small" 
                                                    variant="outlined" 
                                                    sx={{ mr: 0.5, mb: 0.5 }}
                                                />
                                            ))}
                                            {provider.supportedOperations.length > 3 && (
                                                <Chip 
                                                    label={`+${provider.supportedOperations.length - 3} more`} 
                                                    size="small" 
                                                    variant="outlined" 
                                                />
                                            )}
                                        </Box>
                                    </Box>
                                )}

                                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                                    <Typography variant="body2" color="text.secondary">
                                        {provider.userCount} {t("integrations")}
                                    </Typography>
                                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                        {getTestStatusIcon(provider.testStatus)}
                                        {provider.lastTested && (
                                            <Typography variant="caption" color="text.secondary">
                                                {t("LastTested")}: {new Date(provider.lastTested).toLocaleDateString()}
                                            </Typography>
                                        )}
                                    </Box>
                                </Box>
                            </CardContent>
                        </Card>
                    </Grid>
                ))}
            </Grid>

            {/* Add/Edit Service Dialog */}
            <Dialog open={addDialog} onClose={() => setAddDialog(false)} maxWidth="md" fullWidth>
                <DialogTitle>
                    {editProvider ? t("EditService") : t("AddService")}
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2} sx={{ mt: 1 }}>
                        <Grid item xs={12} md={6}>
                            <TextField
                                fullWidth
                                label={t("ServiceName")}
                                value={formData.name}
                                onChange={(e) => setFormData({...formData, name: e.target.value})}
                                required
                            />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <TextField
                                fullWidth
                                label={t("Identifier")}
                                value={formData.identifier}
                                onChange={(e) => setFormData({...formData, identifier: e.target.value})}
                                helperText={t("IdentifierHelp")}
                                required
                            />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <FormControl fullWidth required>
                                <InputLabel>{t("Type")}</InputLabel>
                                <Select
                                    value={formData.type}
                                    onChange={(e) => setFormData({...formData, type: e.target.value as any})}
                                >
                                    <MenuItem value="OAuth">OAuth</MenuItem>
                                    <MenuItem value="ApiKey">API Key</MenuItem>
                                    <MenuItem value="Hybrid">Hybrid</MenuItem>
                                </Select>
                            </FormControl>
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <TextField
                                fullWidth
                                label={t("BaseURL")}
                                value={formData.baseUrl}
                                onChange={(e) => setFormData({...formData, baseUrl: e.target.value})}
                                placeholder="https://api.example.com"
                            />
                        </Grid>
                        
                        {(formData.type === "OAuth" || formData.type === "Hybrid") && (
                            <>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        label={t("OAuthClientID")}
                                        value={formData.oauthClientId}
                                        onChange={(e) => setFormData({...formData, oauthClientId: e.target.value})}
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        type="password"
                                        label={t("OAuthClientSecret")}
                                        value={formData.oauthClientSecret}
                                        onChange={(e) => setFormData({...formData, oauthClientSecret: e.target.value})}
                                    />
                                </Grid>
                            </>
                        )}
                        
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                multiline
                                rows={2}
                                label={t("Description")}
                                value={formData.description}
                                onChange={(e) => setFormData({...formData, description: e.target.value})}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                label={t("SupportedOperations")}
                                value={formData.supportedOperations}
                                onChange={(e) => setFormData({...formData, supportedOperations: e.target.value})}
                                helperText={t("SupportedOperationsHelp")}
                                placeholder="create, read, update, delete"
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setAddDialog(false)}>
                        {t("Cancel")}
                    </Button>
                    <Button variant="contained" onClick={handleSubmit}>
                        {editProvider ? t("Update") : t("Create")}
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Test Results Dialog */}
            <Dialog 
                open={testDialog.open} 
                onClose={() => setTestDialog(prev => ({ ...prev, open: false }))}
                maxWidth="sm" 
                fullWidth
            >
                <DialogTitle>
                    {t("TestResults")} - {testDialog.provider?.name}
                </DialogTitle>
                <DialogContent>
                    {testDialog.testing ? (
                        <Box sx={{ textAlign: "center", py: 3 }}>
                            <Typography>{t("TestingConnection")}</Typography>
                        </Box>
                    ) : testDialog.results ? (
                        <Box>
                            <Alert 
                                severity={testDialog.results.success ? "success" : "error"}
                                sx={{ mb: 2 }}
                            >
                                {testDialog.results.success ? t("TestSuccessful") : t("TestFailed")}
                            </Alert>
                            
                            <List>
                                <ListItem>
                                    <ListItemText 
                                        primary={t("Connectivity")}
                                        secondary={testDialog.results.connectivity ? t("Success") : t("Failed")}
                                    />
                                    {testDialog.results.connectivity ? 
                                        <IconCommon name="Success" fill="success.main" /> : 
                                        <IconCommon name="Error" fill="error.main" />
                                    }
                                </ListItem>
                                <ListItem>
                                    <ListItemText 
                                        primary={t("Authentication")}
                                        secondary={testDialog.results.authentication ? t("Success") : t("Failed")}
                                    />
                                    {testDialog.results.authentication ? 
                                        <IconCommon name="Success" fill="success.main" /> : 
                                        <IconCommon name="Error" fill="error.main" />
                                    }
                                </ListItem>
                            </List>

                            {testDialog.results.operations && (
                                <Box>
                                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                        {t("Operations")}:
                                    </Typography>
                                    {testDialog.results.operations.map((op: any) => (
                                        <Chip
                                            key={op.name}
                                            label={op.name}
                                            color={op.status === "success" ? "success" : "error"}
                                            size="small"
                                            sx={{ mr: 1, mb: 1 }}
                                        />
                                    ))}
                                </Box>
                            )}
                        </Box>
                    ) : null}
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setTestDialog(prev => ({ ...prev, open: false }))}>
                        {t("Close")}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
};