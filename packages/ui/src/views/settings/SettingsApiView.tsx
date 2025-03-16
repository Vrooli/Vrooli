import { API_CREDITS_FREE, ApiKey, ApiKeyCreateInput, ApiKeyCreated, ApiKeyExternal, ApiKeyExternalCreateInput, ApiKeyExternalUpdateInput, ApiKeyUpdateInput, DeleteOneInput, DeleteType, FormStructureType, Success, User, endpointsActions, endpointsApiKey, endpointsApiKeyExternal, noop } from "@local/shared";
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Divider, IconButton, MenuItem, Select, Stack, TextField, Typography, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { LazyRequestWithResult } from "../../api/types.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { FormTip } from "../../components/inputs/form/FormTip/FormTip.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { DeleteIcon, EditIcon, PlusIcon, SuccessIcon } from "../../icons/common.js";
import { ScrollBox } from "../../styles.js";
import { randomString } from "../../utils/codes.js";
import { BUSINESS_DATA } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { SettingsApiViewProps } from "./types.js";

// Define known services and supported integrations
const KNOWN_SERVICES = ["OpenAI", "Google Maps", "Microsoft"];
const SUPPORTED_INTEGRATIONS = [
    { name: "OpenAI", url: "https://openai.com/authorize" },
    { name: "Microsoft", url: "https://microsoft.com/authorize" },
    // Add more as needed (ensure logos are imported)
];

const dialogPaperProps = {
    sx: {
        bgcolor: "background.default",
    },
} as const;

type ApiKeyViewDialogProps = {
    apiKey: string | null;
    onClose: () => void;
    open: boolean;
}

function ApiKeyViewDialog({
    apiKey,
    onClose,
    open,
}: ApiKeyViewDialogProps) {
    const { palette } = useTheme();

    const handleCopyKey = useCallback(() => {
        if (!apiKey) {
            console.error("Can't copy api key.", { component: "ApiKeyViewDialog", function: "handleCopyKey", apiKey });
            return;
        }
        navigator.clipboard.writeText(apiKey)
            .then(() => {
                PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
            })
            .catch(() => {
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            });
    }, [apiKey]);

    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby="api-key-view-dialog"
            PaperProps={dialogPaperProps}
        >
            <DialogTitle id="api-key-view-dialog">
                Your New API Key
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2}>
                    <Typography variant="body1">
                        This is your new API key. Please store it in a secure location.
                    </Typography>
                    <Typography variant="body2" color="error">
                        Important: This key will only be shown once. You won't be able to see it again after closing this dialog.
                    </Typography>
                    <Box
                        position="relative"
                        p={2}
                        border={1}
                        borderColor="divider"
                        borderRadius={1}
                        bgcolor={palette.background.paper}
                    >
                        <Typography variant="body2" sx={{ wordBreak: "break-all", pr: 4 }}>
                            {apiKey}
                        </Typography>
                        <Button
                            sx={{
                                position: "absolute",
                                top: 8,
                                right: 8,
                                minWidth: "auto",
                                p: 1,
                            }}
                            onClick={handleCopyKey}
                        >
                            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                                <path d="M16 1H4C2.9 1 2 1.9 2 3V17H4V3H16V1ZM19 5H8C6.9 5 6 5.9 6 7V21C6 22.1 6.9 23 8 23H19C20.1 23 21 22.1 21 21V7C21 5.9 20.1 5 19 5ZM19 21H8V7H19V21Z" />
                            </svg>
                        </Button>
                    </Box>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Close</Button>
            </DialogActions>
        </Dialog>
    );
}

type InternalKeyEditDialogFormValues = Omit<ApiKey, "__typename" | "creditsUsed" | "id" | "disabledAt" | "limitSoft"> & {
    disabled: boolean;
    limitSoft: string;
}
type InternalKeyEditDialogProps = {
    keyData: ApiKey | null;
    onClose: () => unknown;
    open: boolean;
}
function InternalKeyEditDialog({
    keyData,
    onClose,
    open,
}: InternalKeyEditDialogProps) {
    const { t } = useTranslation();
    const [updateInternalKey] = useLazyFetch<ApiKeyUpdateInput, ApiKey>(endpointsApiKey.updateOne);
    const { onProfileUpdate, profile } = useProfileQuery();

    const initialValues = useMemo<InternalKeyEditDialogFormValues>(() => ({
        name: keyData?.name ?? "",
        limitHard: keyData?.limitHard ?? "",
        limitSoft: keyData?.limitSoft ?? "",
        stopAtLimit: keyData?.stopAtLimit ?? false,
        disabled: !!keyData?.disabledAt,
    }), [keyData]);

    const handleSubmit = useCallback(function handleSubmitCallback(values: InternalKeyEditDialogFormValues) {
        if (!keyData) {
            console.error("This error shouldn't happen. Please report it.", { component: "InternalKeyEditDialog", function: "handleSubmit", keyData });
            PubSub.get().publish("snack", { messageKey: "MissingRequiredField", severity: "Error" });
            return;
        }
        const input: ApiKeyUpdateInput = {
            id: keyData.id,
            name: values.name,
            limitHard: values.limitHard,
            limitSoft: values.limitSoft.length > 0 ? values.limitSoft : null,
            stopAtLimit: values.stopAtLimit,
            disabled: values.disabled,
        };
        fetchLazyWrapper<ApiKeyUpdateInput, ApiKey>({
            fetch: updateInternalKey,
            inputs: input,
            onSuccess: (updatedKey) => {
                const updatedKeys = profile?.apiKeys?.map(k => k.id === updatedKey.id ? updatedKey : k) ?? [];
                onProfileUpdate({ ...profile, apiKeys: updatedKeys } as User);
                onClose();
            },
        });
    }, [keyData, onClose, onProfileUpdate, profile, updateInternalKey]);

    return (
        <Dialog open={open} onClose={onClose} PaperProps={dialogPaperProps}>
            <DialogTitle>{t("EditApiKey")}</DialogTitle>
            <DialogContent>
                <Formik enableReinitialize initialValues={initialValues} onSubmit={handleSubmit}>
                    {({ handleSubmit, values, setFieldValue }) => (
                        <form onSubmit={handleSubmit}>
                            <TextField
                                fullWidth
                                label={t("Name")}
                                value={values.name}
                                onChange={(e) => setFieldValue("name", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            <TextField
                                fullWidth
                                label={t("HardLimit")}
                                type="number"
                                value={values.limitHard}
                                onChange={(e) => setFieldValue("limitHard", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            <TextField
                                fullWidth
                                label={t("SoftLimit")}
                                type="number"
                                value={values.limitSoft}
                                onChange={(e) => setFieldValue("limitSoft", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            {/* Add more fields as needed */}
                            <Button type="submit">{t("Save")}</Button>
                        </form>
                    )}
                </Formik>
            </DialogContent>
        </Dialog>
    );
}

type ExternalKeyDialogFormValues = {
    serviceSelect: string;
    customService: string;
    name: string;
    key: string;
}
type ExternalKeyDialogProps = {
    isCreate: boolean;
    keyData: ApiKeyExternal | null;
    onClose: () => unknown;
    open: boolean;
}
function ExternalKeyDialog({
    isCreate,
    keyData,
    onClose,
    open,
}: ExternalKeyDialogProps) {
    const { t } = useTranslation();
    const [createExternalKey] = useLazyFetch<ApiKeyExternalCreateInput, ApiKeyExternal>(endpointsApiKeyExternal.createOne);
    const [updateExternalKey] = useLazyFetch<ApiKeyExternalUpdateInput, ApiKeyExternal>(endpointsApiKeyExternal.updateOne);
    const { onProfileUpdate, profile } = useProfileQuery();

    const initialValues = useMemo<ExternalKeyDialogFormValues>(() => ({
        serviceSelect: keyData && KNOWN_SERVICES.includes(keyData.service) ? keyData.service : "Other",
        customService: !keyData || !KNOWN_SERVICES.includes(keyData.service) ? keyData?.service ?? "" : "",
        name: keyData?.name ?? "",
        key: "",
    }), [keyData]);

    const handleSubmit = useCallback(function handleSubmitCallback(values: ExternalKeyDialogFormValues) {
        const service = values.serviceSelect === "Other" ? values.customService : values.serviceSelect;
        if (!service) {
            PubSub.get().publish("snack", { messageKey: "MissingRequiredField", severity: "Error" });
            return;
        }
        if (!isCreate && !keyData) {
            console.error("This error shouldn't happen. Please report it.", { component: "ExternalKeyDialog", keyData });
            PubSub.get().publish("snack", { messageKey: "MissingRequiredField", severity: "Error" });
            return;
        }
        const input = isCreate ? {
            service,
            name: values.name,
            key: values.key,
        } : {
            id: (keyData as ApiKeyExternal).id,
            service,
            name: values.name,
            key: values.key,
        };
        const fetchFunc = isCreate ? createExternalKey : updateExternalKey;
        fetchLazyWrapper<ApiKeyExternalCreateInput | ApiKeyExternalUpdateInput, ApiKeyExternal>({
            fetch: fetchFunc as LazyRequestWithResult<ApiKeyExternalCreateInput | ApiKeyExternalUpdateInput, ApiKeyExternal>,
            inputs: input,
            onSuccess: (data) => {
                if (isCreate) {
                    const updatedKeys = [...(profile?.apiKeysExternal ?? []), data];
                    onProfileUpdate({ ...profile, apiKeysExternal: updatedKeys } as User);
                } else {
                    const updatedKeys = profile?.apiKeysExternal?.map(k => k.id === data.id ? data : k) ?? [];
                    onProfileUpdate({ ...profile, apiKeysExternal: updatedKeys } as User);
                }
                onClose();
            },
        });
    }, [createExternalKey, isCreate, keyData, onClose, onProfileUpdate, profile, updateExternalKey]);

    return (
        <Dialog open={open} onClose={onClose} PaperProps={dialogPaperProps}>
            <DialogTitle>{isCreate ? t("ApiKeyAdd") : t("ApiKeyUpdate")}</DialogTitle>
            <DialogContent>
                <Formik initialValues={initialValues} onSubmit={handleSubmit}>
                    {({ handleSubmit, values, setFieldValue }) => (
                        <form onSubmit={handleSubmit}>
                            <Select
                                fullWidth
                                value={values.serviceSelect}
                                onChange={(e) => setFieldValue("serviceSelect", e.target.value)}
                                displayEmpty
                                sx={{ mb: 2 }}
                            >
                                <MenuItem value="" disabled>{t("SelectService")}</MenuItem>
                                {KNOWN_SERVICES.map(service => (
                                    <MenuItem key={service} value={service}>{service}</MenuItem>
                                ))}
                                <MenuItem value="Other">{t("Other")}</MenuItem>
                            </Select>
                            {values.serviceSelect === "Other" && (
                                <TextField
                                    fullWidth
                                    label={t("CustomServiceName")}
                                    value={values.customService}
                                    onChange={(e) => setFieldValue("customService", e.target.value)}
                                    sx={{ mb: 2 }}
                                />
                            )}
                            <TextField
                                fullWidth
                                label={t("NameOptional")}
                                value={values.name}
                                onChange={(e) => setFieldValue("name", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            <PasswordTextInput
                                fullWidth
                                name="key"
                                label={t("ApiKey")}
                                value={values.key}
                                onChange={(e) => setFieldValue("key", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            <Button type="submit">{isCreate ? t("Add") : t("Update")}</Button>
                        </form>
                    )}
                </Formik>
            </DialogContent>
        </Dialog>
    );
}

export function SettingsApiView({
    display,
    onClose,
}: SettingsApiViewProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const apiKeys = profile?.apiKeys || [];

    // API endpoints
    const [createInternalKey] = useLazyFetch<ApiKeyCreateInput, ApiKeyCreated>(endpointsApiKey.createOne);
    const [updateInternalKey] = useLazyFetch<ApiKeyUpdateInput, ApiKey>(endpointsApiKey.updateOne);
    const [createExternalKey] = useLazyFetch<ApiKeyExternalCreateInput, ApiKeyExternal>(endpointsApiKeyExternal.createOne);
    const [updateExternalKey] = useLazyFetch<ApiKeyExternalUpdateInput, ApiKeyExternal>(endpointsApiKeyExternal.updateOne);
    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);

    const [newKeyDialogOpen, setNewKeyDialogOpen] = useState(false);
    const [newKey, setNewKey] = useState<string | null>(null);
    const [editingInternalKey, setEditingInternalKey] = useState<ApiKey | null>(null);
    const [isEditingInternalKey, setIsEditingInternalKey] = useState(false);
    const [editingExternalKey, setEditingExternalKey] = useState<ApiKeyExternal | null>(null);
    const [isEditingExternalKey, setIsEditingExternalKey] = useState(false);
    const [isCreatingExternalKey, setIsCreatingExternalKey] = useState(false);
    function onNewKeyDialogClose() {
        setNewKeyDialogOpen(false);
        setNewKey(null);
    }

    const generateNewInternalKey = useCallback(function generateNewInternalKeyCallback() {
        fetchLazyWrapper<ApiKeyCreateInput, ApiKeyCreated>({
            fetch: createInternalKey,
            inputs: {
                disabled: false,
                // Default to one month's worth of credits. Make the user 
                // explicitly update this if they plan on using more than that.
                limitHard: API_CREDITS_FREE.toString(),
                limitSoft: null,
                // Simple naming; adjust as needed
                name: `API key ${randomString(4)}`,
                stopAtLimit: true,
            }, // Simple naming; adjust as needed
            onSuccess: (data) => {
                const { key, ...rest } = data;
                setNewKey(key);
                setNewKeyDialogOpen(true);
                if (profile) {
                    onProfileUpdate({
                        ...profile,
                        apiKeys: [...apiKeys, rest],
                    });
                }
            },
        });
    }, [apiKeys, createInternalKey, onProfileUpdate, profile]);

    // Handler to revoke an API key
    const revokeInternalKey = useCallback((keyId: string) => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: keyId, objectType: DeleteType.ApiKey },
            onSuccess: () => {
                const updatedKeys = apiKeys.filter(key => key.id !== keyId);
                onProfileUpdate({
                    ...profile,
                    apiKeys: updatedKeys,
                });
            },
        });
    }, [apiKeys, deleteOne, onProfileUpdate, profile]);

    const handleEditInternalKey = useCallback((key: ApiKey) => {
        setEditingInternalKey(key);
        setIsEditingInternalKey(true);
    }, []);

    const handleUpdateExternalKey = useCallback((key: ApiKeyExternal) => {
        setEditingExternalKey(key);
        setIsEditingExternalKey(true);
    }, []);

    const handleDeleteExternalKey = useCallback((id: string) => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id, objectType: DeleteType.ApiKeyExternal },
            onSuccess: () => {
                const updatedKeys = profile?.apiKeysExternal?.filter(k => k.id !== id) ?? [];
                onProfileUpdate({ ...profile, apiKeysExternal: updatedKeys } as User);
            },
        });
    }, [deleteOne, onProfileUpdate, profile]);

    function handleInternalKeyEditClose() {
        setIsEditingInternalKey(false);
        setEditingInternalKey(null);
    }

    function handleExternalKeyEditClose() {
        setIsEditingExternalKey(false);
        setEditingExternalKey(null);
    }

    // // Handler for external integrations (e.g. OAuth flow)
    // const handleAuthorizeIntegration = useCallback(function handleAuthorizeIntegration() {
    //     const width = 600;
    //     const height = 600;
    //     const left = window.screenX + (window.outerWidth - width) / 2;
    //     const top = window.screenY + (window.outerHeight - height) / 2;
    //     // TODO: Replace the URL below with the actual authorization URL for the external service.
    //     window.open("https://external.service.com/oauth/authorize", "ExternalIntegration", `width=${width},height=${height},left=${left},top=${top}`);
    // }, []);

    return (
        <>
            <ApiKeyViewDialog apiKey={newKey} onClose={onNewKeyDialogClose} open={newKeyDialogOpen} />
            <InternalKeyEditDialog keyData={editingInternalKey} onClose={handleInternalKeyEditClose} open={isEditingInternalKey && !!editingInternalKey} />
            <ExternalKeyDialog
                isCreate={isCreatingExternalKey}
                keyData={editingExternalKey}
                onClose={handleExternalKeyEditClose}
                open={isCreatingExternalKey || (isEditingExternalKey && !!editingExternalKey)}
            />
            <ScrollBox>
                <SettingsTopBar
                    display={display}
                    onClose={onClose}
                    title={t("Api", { count: 1 })}
                />
                <SettingsContent>
                    <SettingsList />
                    <Stack
                        direction="column"
                        spacing={6}
                        m="auto"
                        pl={2}
                        pr={2}
                        width="100%"
                        maxWidth="min(100%, 700px)"
                    >
                        <Box>
                            <Title title={`${t("ApiKey", { count: 2 })} - ${BUSINESS_DATA.BUSINESS_NAME}`} variant="subheader" addSidePadding={false} />
                            {apiKeys.length > 0 ? (
                                apiKeys.map((key) => {
                                    function handleEdit() {
                                        handleEditInternalKey(key);
                                    }
                                    function handleRevoke() {
                                        revokeInternalKey(key.id);
                                    }

                                    return (
                                        <Box key={key.id} display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                                            <Typography>{key.name}</Typography>
                                            <Box>
                                                <IconButton onClick={handleEdit}>
                                                    <EditIcon fill={palette.secondary.main} />
                                                </IconButton>
                                                <IconButton onClick={handleRevoke}>
                                                    <DeleteIcon fill={palette.error.main} />
                                                </IconButton>
                                            </Box>
                                        </Box>
                                    );
                                })
                            ) : (
                                <Typography variant="body1" color="text.secondary">{t("NoApiKeys")}</Typography>
                            )}
                            <Button onClick={generateNewInternalKey} startIcon={<PlusIcon />} sx={{ mt: 2 }}>{t("CreateNew")}</Button>
                        </Box>
                        <Divider />
                        <Box>
                            <Title title={`${t("ApiKey", { count: 2 })} - ${t("External")}`} variant="subheader" addSidePadding={false} />
                            <FormTip
                                element={{
                                    icon: "Warning",
                                    isMarkdown: false,
                                    label: "WARNING: Giving API keys to other apps can be dangerous. Make sure you trust us.",
                                    type: FormStructureType.Tip,
                                }}
                                isEditing={false}
                                onUpdate={noop}
                                onDelete={noop}
                            />
                            {profile?.apiKeysExternal && profile.apiKeysExternal.length > 0 ? (
                                profile.apiKeysExternal.map((key) => {
                                    function handleUpdate() {
                                        handleUpdateExternalKey(key);
                                    }
                                    function handleDelete() {
                                        handleDeleteExternalKey(key.id);
                                    }
                                    return (
                                        <Box key={key.id} display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                                            <Typography>{key.service}{key.name ? `: ${key.name}` : ""}</Typography>
                                            <Box>
                                                <IconButton onClick={handleUpdate}>
                                                    <EditIcon fill={palette.secondary.main} />
                                                </IconButton>
                                                <IconButton onClick={handleDelete}>
                                                    <DeleteIcon fill={palette.error.main} />
                                                </IconButton>
                                            </Box>
                                        </Box>
                                    );
                                })
                            ) : (
                                <Typography variant="body1" color="text.secondary">{t("NoExternalApiKeys")}</Typography>
                            )}
                            <Button onClick={() => setIsCreatingExternalKey(true)} sx={{ mt: 2 }}>{t("AddExternalKey")}</Button>
                        </Box>
                        <Divider />
                        {SUPPORTED_INTEGRATIONS.length > 0 && (
                            <Box>
                                <Title title={t("ExternalIntegrations")} help={t("ConnectExternalServices")} variant="subheader" addSidePadding={false} />
                                <Box display="flex" flexWrap="wrap" gap={2}>
                                    {SUPPORTED_INTEGRATIONS.map((integration) => {
                                        const isConnected = profile?.apiKeysExternal?.some(k => k.service === integration.name && !k.disabledAt) ?? false;
                                        return (
                                            <Button
                                                key={integration.name}
                                                onClick={() => window.open(integration.url, "_blank")}
                                                variant="contained"
                                                startIcon={<img src={integration.logo} alt={`${integration.name} logo`} style={{ height: "24px" }} />}
                                                endIcon={isConnected ? <SuccessIcon /> : null}
                                            >
                                                {integration.name}
                                            </Button>
                                        );
                                    })}
                                </Box>
                            </Box>
                        )}
                    </Stack>
                </SettingsContent>
            </ScrollBox>
        </>
    );
}
