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
    Select,
    TextField,
    Typography,
    Alert,
    Stack,
    List,
    ListItem,
    ListItemText,
} from "@mui/material";
import { Tooltip } from "../../../components/Tooltip/Tooltip.js";
import { DialogTitle } from "../../../components/dialogs/DialogTitle/DialogTitle.js";
import React, { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
    ResourceType,
    type ResourceSearchInput,
    type ResourceCreateInput,
    type ResourceUpdateInput,
    type ResourceSearchResult,
    type Resource,
    ApiVersionConfig,
    generatePublicId,
    endpointsResource,
    endpointsActions,
    DeleteType,
    type ApiVersionConfigObject,
    type DeleteOneInput,
    type Success,
} from "@vrooli/shared";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { PubSub } from "../../../utils/pubsub.js";

// Service provider types - mapped from Resource
interface ExternalServiceProvider {
    id: string;
    publicId: string;
    name: string;
    identifier: string;
    type: "oauth2" | "apikey" | "hybrid";
    status: "Active" | "Disabled" | "Maintenance";
    description?: string;
    oauthClientId?: string;
    oauthClientSecret?: string;
    oauthAuthUrl?: string;
    oauthTokenUrl?: string;
    oauthScopes?: string[];
    baseUrl?: string;
    supportedOperations?: string[];
    userCount?: number;
    lastTested?: string;
    testStatus?: "success" | "error" | "warning";
    // Additional fields for API configuration
    authLocation?: string;
    authParameterName?: string;
    resourceId?: string;
    versionId?: string;
}

/**
 * Panel for managing external service providers and integrations
 * Allows admins to add new services without code changes
 */
export const ExternalServicesPanel: React.FC = () => {
    const { t } = useTranslation();
    
    // API hooks
    const [findResources] = useLazyFetch<ResourceSearchInput, ResourceSearchResult>(endpointsResource.findMany);
    const [createResource] = useLazyFetch<ResourceCreateInput, Resource>(endpointsResource.createOne);
    const [updateResource] = useLazyFetch<ResourceUpdateInput, Resource>(endpointsResource.updateOne);
    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    
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
        type: "apikey" as "oauth2" | "apikey" | "hybrid",
        description: "",
        oauthClientId: "",
        oauthClientSecret: "",
        oauthScopes: "",
        oauthAuthUrl: "",
        oauthTokenUrl: "",
        baseUrl: "",
        authMethod: "header",
        authParameterName: "Authorization",
        supportedOperations: "",
    });

    // Helper function to convert Resource to ExternalServiceProvider
    const resourceToProvider = useCallback((resource: Resource): ExternalServiceProvider | null => {
        const version = resource.versions?.[0];
        if (!version) return null;
        
        const translation = version.translations?.[0];
        if (!translation) return null;
        
        let config: ApiVersionConfig | null = null;
        try {
            config = ApiVersionConfig.parse(version, console);
        } catch (e) {
            console.error("Failed to parse API config", e);
        }
        
        const authSettings = config?.authentication?.settings || {};
        
        return {
            id: resource.id,
            publicId: resource.publicId,
            name: translation.name,
            identifier: resource.publicId, // Using publicId as identifier
            type: (config?.authentication?.type || "apikey") as any,
            status: "Active", // TODO: Derive from resource status
            description: translation.description || undefined,
            oauthClientId: authSettings.clientId,
            oauthClientSecret: authSettings.clientSecret,
            oauthAuthUrl: authSettings.authUrl,
            oauthTokenUrl: authSettings.tokenUrl,
            oauthScopes: authSettings.scopes,
            baseUrl: config?.callLink || version.callLink || undefined,
            supportedOperations: authSettings.operations || [],
            userCount: 0, // TODO: Get from analytics
            resourceId: resource.id,
            versionId: version.id,
            authLocation: config?.authentication?.location,
            authParameterName: config?.authentication?.parameterName,
        };
    }, []);

    // Load API resources
    const loadProviders = useCallback(async () => {
        setLoading(true);
        try {
            const searchInput: ResourceSearchInput = {
                resourceType: ResourceType.Api,
                isDeleted: false,
                take: 100,
            };
            
            fetchLazyWrapper({
                fetch: findResources,
                inputs: searchInput,
                onSuccess: (data) => {
                    const providers = data.edges
                        .map(edge => resourceToProvider(edge.node))
                        .filter(Boolean) as ExternalServiceProvider[];
                    setProviders(providers);
                },
                onError: () => {
                    PubSub.get().publish("snack", { 
                        messageKey: "FailedToLoadServices", 
                        severity: "Error", 
                    });
                },
            });
        } finally {
            setLoading(false);
        }
    }, [findResources, resourceToProvider]);

    useEffect(() => {
        loadProviders();
    }, [loadProviders]);

    const handleSubmit = async () => {
        try {
            // Build API configuration
            const apiConfig: ApiVersionConfigObject = {
                __version: "1.0",
                resources: [],
                authentication: {
                    type: formData.type,
                    location: formData.authMethod,
                    parameterName: formData.authParameterName,
                    settings: {
                        ...(formData.type === "oauth2" || formData.type === "hybrid" ? {
                            clientId: formData.oauthClientId,
                            clientSecret: formData.oauthClientSecret,
                            authUrl: formData.oauthAuthUrl,
                            tokenUrl: formData.oauthTokenUrl,
                            scopes: formData.oauthScopes.split(",").map(s => s.trim()).filter(Boolean),
                        } : {}),
                        operations: formData.supportedOperations.split(",").map(s => s.trim()).filter(Boolean),
                    },
                },
                rateLimiting: {
                    requestsPerMinute: 1000,
                    burstLimit: 100,
                },
                callLink: formData.baseUrl,
            };

            if (editProvider) {
                // Update existing resource
                const updateInput: ResourceUpdateInput = {
                    id: editProvider.resourceId!,
                    versions: [{
                        id: editProvider.versionId!,
                        callLink: formData.baseUrl,
                        config: JSON.stringify(apiConfig),
                        translations: [{
                            language: "en",
                            name: formData.name,
                            description: formData.description,
                        }],
                    }],
                };

                fetchLazyWrapper({
                    fetch: updateResource,
                    inputs: updateInput,
                    onSuccess: () => {
                        loadProviders();
                        setAddDialog(false);
                        setEditProvider(null);
                        resetForm();
                        PubSub.get().publish("snack", { 
                            messageKey: "ServiceUpdated", 
                            severity: "Success", 
                        });
                    },
                    onError: () => {
                        PubSub.get().publish("snack", { 
                            messageKey: "FailedToUpdateService", 
                            severity: "Error", 
                        });
                    },
                });
            } else {
                // Create new resource
                const createInput: ResourceCreateInput = {
                    resourceType: ResourceType.Api,
                    isPublic: true,
                    versions: [{
                        versionLabel: "1.0",
                        callLink: formData.baseUrl,
                        config: JSON.stringify(apiConfig),
                        translations: [{
                            language: "en",
                            name: formData.name,
                            description: formData.description,
                        }],
                    }],
                };

                fetchLazyWrapper({
                    fetch: createResource,
                    inputs: createInput,
                    onSuccess: () => {
                        loadProviders();
                        setAddDialog(false);
                        resetForm();
                        PubSub.get().publish("snack", { 
                            messageKey: "ServiceCreated", 
                            severity: "Success", 
                        });
                    },
                    onError: () => {
                        PubSub.get().publish("snack", { 
                            messageKey: "FailedToCreateService", 
                            severity: "Error", 
                        });
                    },
                });
            }
        } catch (error) {
            console.error("Failed to save provider:", error);
            PubSub.get().publish("snack", { 
                messageKey: "ErrorUnknown", 
                severity: "Error", 
            });
        }
    };

    const resetForm = () => {
        setFormData({
            name: "",
            identifier: "",
            type: "apikey",
            description: "",
            oauthClientId: "",
            oauthClientSecret: "",
            oauthScopes: "",
            oauthAuthUrl: "",
            oauthTokenUrl: "",
            baseUrl: "",
            authMethod: "header",
            authParameterName: "Authorization",
            supportedOperations: "",
        });
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
                        testStatus: mockResults.success ? "success" : "error",
                    }
                    : p,
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
            const provider = providers.find(p => p.id === providerId);
            if (!provider?.resourceId) {
                console.error("No resource ID for provider");
                return;
            }
            
            fetchLazyWrapper({
                fetch: deleteOne,
                inputs: { 
                    id: provider.resourceId,
                    objectType: DeleteType.Resource,
                },
                onSuccess: () => {
                    loadProviders();
                    PubSub.get().publish("snack", { 
                        messageKey: "ServiceDeleted", 
                        severity: "Success", 
                    });
                },
                onError: () => {
                    PubSub.get().publish("snack", { 
                        messageKey: "FailedToDeleteService", 
                        severity: "Error", 
                    });
                },
            });
        } catch (error) {
            console.error("Failed to delete provider:", error);
            PubSub.get().publish("snack", { 
                messageKey: "ErrorUnknown", 
                severity: "Error", 
            });
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
                                                        oauthClientSecret: provider.oauthClientSecret || "",
                                                        oauthScopes: provider.oauthScopes?.join(", ") || "",
                                                        oauthAuthUrl: provider.oauthAuthUrl || "",
                                                        oauthTokenUrl: provider.oauthTokenUrl || "",
                                                        baseUrl: provider.baseUrl || "",
                                                        authMethod: provider.authLocation || "header",
                                                        authParameterName: provider.authParameterName || "Authorization",
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
                <DialogTitle id="add-edit-service-dialog" title={editProvider ? t("EditService") : t("AddService")} />
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
                                    <MenuItem value="oauth2">OAuth 2.0</MenuItem>
                                    <MenuItem value="apikey">API Key</MenuItem>
                                    <MenuItem value="hybrid">Hybrid</MenuItem>
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
                        
                        {(formData.type === "oauth2" || formData.type === "hybrid") && (
                            <>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        label={t("OAuthClientID")}
                                        value={formData.oauthClientId}
                                        onChange={(e) => setFormData({...formData, oauthClientId: e.target.value})}
                                        required
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        type="password"
                                        label={t("OAuthClientSecret")}
                                        value={formData.oauthClientSecret}
                                        onChange={(e) => setFormData({...formData, oauthClientSecret: e.target.value})}
                                        required
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        label={t("OAuthAuthorizationURL")}
                                        value={formData.oauthAuthUrl}
                                        onChange={(e) => setFormData({...formData, oauthAuthUrl: e.target.value})}
                                        placeholder="https://example.com/oauth/authorize"
                                        required
                                    />
                                </Grid>
                                <Grid item xs={12} md={6}>
                                    <TextField
                                        fullWidth
                                        label={t("OAuthTokenURL")}
                                        value={formData.oauthTokenUrl}
                                        onChange={(e) => setFormData({...formData, oauthTokenUrl: e.target.value})}
                                        placeholder="https://example.com/oauth/token"
                                        required
                                    />
                                </Grid>
                                <Grid item xs={12}>
                                    <TextField
                                        fullWidth
                                        label={t("OAuthScopes")}
                                        value={formData.oauthScopes}
                                        onChange={(e) => setFormData({...formData, oauthScopes: e.target.value})}
                                        helperText={t("CommaSeparatedScopes")}
                                        placeholder="read, write, profile"
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
                <DialogTitle 
                    id="test-results-dialog"
                    title={`${t("TestResults")} - ${testDialog.provider?.name}`} 
                />
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
