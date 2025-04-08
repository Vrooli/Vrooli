import { ApiKey, ApiKeyCreateInput, ApiKeyCreated, ApiKeyExternal, ApiKeyExternalCreateInput, ApiKeyExternalUpdateInput, ApiKeyPermission, ApiKeyUpdateInput, DeleteOneInput, DeleteType, FormStructureType, Success, User, endpointsActions, endpointsApiKey, endpointsApiKeyExternal, noop } from "@local/shared";
import { Box, Button, Checkbox, Chip, Dialog, DialogActions, DialogContent, DialogTitle, Divider, FormControl, FormControlLabel, FormGroup, FormHelperText, FormLabel, IconButton, MenuItem, Radio, RadioGroup, Select, Stack, Typography, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { LazyRequestWithResult } from "../../api/types.js";
import { PageContainer } from "../../components/Page/Page.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { FormTip } from "../../components/inputs/form/FormTip.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { IconCommon, IconFavicon } from "../../icons/Icons.js";
import { ScrollBox } from "../../styles.js";
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

/**
 * Display-friendly names for permissions
 */
export const PERMISSION_NAMES: Record<ApiKeyPermission, string> = {
    [ApiKeyPermission.ReadPublic]: "Read Public Data",
    [ApiKeyPermission.ReadPrivate]: "Read Private Data",
    [ApiKeyPermission.WritePrivate]: "Write Private Data",
    [ApiKeyPermission.ReadAuth]: "Read Auth Info",
    [ApiKeyPermission.WriteAuth]: "Write Auth Info",
};

/**
 * Permission descriptions explaining what each permission allows
 */
export const PERMISSION_DESCRIPTIONS: Record<ApiKeyPermission, string> = {
    [ApiKeyPermission.ReadPublic]: "Access publicly available data (users, projects, etc.)",
    [ApiKeyPermission.ReadPrivate]: "Read your private data (private projects, settings, etc.)",
    [ApiKeyPermission.WritePrivate]: "Update your private data (update projects, preferences, etc.)",
    [ApiKeyPermission.ReadAuth]: "View authentication-related data (session info, login history)",
    [ApiKeyPermission.WriteAuth]: "Update authentication-related data (emails, password)",
};

/**
 * Security level for each permission (for UI indication)
 */
export enum PermissionSecurityLevel {
    Low = "Low",
    Medium = "Medium",
    High = "high",
}

/**
 * Security levels for each permission
 */
export const PERMISSION_SECURITY_LEVELS: Record<ApiKeyPermission, PermissionSecurityLevel> = {
    [ApiKeyPermission.ReadPublic]: PermissionSecurityLevel.Low,
    [ApiKeyPermission.ReadPrivate]: PermissionSecurityLevel.Medium,
    [ApiKeyPermission.WritePrivate]: PermissionSecurityLevel.Medium,
    [ApiKeyPermission.ReadAuth]: PermissionSecurityLevel.High,
    [ApiKeyPermission.WriteAuth]: PermissionSecurityLevel.High,
};

/**
 * Predefined permission sets that users can choose from
 */
export const PERMISSION_PRESETS = {
    READ_ONLY: {
        name: "Read Only",
        description: "Only access to read public data",
        permissions: [
            ApiKeyPermission.ReadPublic
        ]
    },
    STANDARD: {
        name: "Standard Access",
        description: "Read public and your private data",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate
        ]
    },
    DEVELOPER: {
        name: "Developer Access",
        description: "Read and update your data",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate,
            ApiKeyPermission.WritePrivate
        ]
    },
    FULL_ACCESS: {
        name: "Full Access (High Security Risk)",
        description: "Complete access to everything including authentication",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate,
            ApiKeyPermission.WritePrivate,
            ApiKeyPermission.ReadAuth,
            ApiKeyPermission.WriteAuth
        ]
    }
};

/**
 * Check if a set of permissions includes all permissions from a preset
 */
export function matchesPreset(permissions: ApiKeyPermission[], preset: typeof PERMISSION_PRESETS[keyof typeof PERMISSION_PRESETS]): boolean {
    // Check that all permissions in the preset are included in the permissions array
    return preset.permissions.every(p => permissions.includes(p)) &&
        // Check that there are no additional permissions
        permissions.length === preset.permissions.length;
}

/**
 * Find which preset (if any) matches the given permissions
 */
export function findMatchingPreset(permissions: ApiKeyPermission[]): string | null {
    for (const [key, preset] of Object.entries(PERMISSION_PRESETS)) {
        if (matchesPreset(permissions, preset)) {
            return key;
        }
    }
    return null;
}

// Extract permissions from a stringified JSON field
function extractPermissions(apiKey: ApiKey | null): ApiKeyPermission[] {
    if (!apiKey) return [];

    try {
        // 'permissions' is a stringified JSON array of permission strings
        const permissionsStr = (apiKey as any).permissions;
        if (!permissionsStr) return [];

        const parsedPermissions = JSON.parse(permissionsStr);
        return Array.isArray(parsedPermissions)
            ? parsedPermissions.filter(p => Object.values(ApiKeyPermission).includes(p as ApiKeyPermission))
            : [];
    } catch (e) {
        console.error("Error parsing permissions", e);
        return [];
    }
}

// Get a user-friendly description of permissions
function getPermissionsDescription(apiKey: ApiKey): string {
    try {
        const permissions = extractPermissions(apiKey);
        const presetKey = findMatchingPreset(permissions);

        if (presetKey) {
            return PERMISSION_PRESETS[presetKey as keyof typeof PERMISSION_PRESETS].name;
        }

        if (permissions.length === 0) {
            return "No permissions";
        }

        if (permissions.length === 1) {
            return "Limited access";
        }

        return `${permissions.length} permissions`;
    } catch (e) {
        return "Unknown permissions";
    }
}

const dialogPaperProps = {
    sx: {
        bgcolor: "background.default",
    },
} as const;
interface ApiKeyPermissionsSelectorProps {
    selectedPermissions: ApiKeyPermission[];
    onChange: (permissions: ApiKeyPermission[]) => void;
    compact?: boolean;
}

export function ApiKeyPermissionsSelector({
    selectedPermissions,
    onChange,
    compact = false
}: ApiKeyPermissionsSelectorProps) {
    const { palette } = useTheme();
    const [selectedPreset, setSelectedPreset] = useState<string | 'custom'>("READ_ONLY");

    // Update the preset when permissions change externally
    useEffect(() => {
        const matchingPreset = findMatchingPreset(selectedPermissions);
        setSelectedPreset(matchingPreset || 'custom');
    }, [selectedPermissions]);

    // When a preset is selected
    const handlePresetChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const presetKey = event.target.value;
        setSelectedPreset(presetKey);

        if (presetKey !== 'custom') {
            const preset = PERMISSION_PRESETS[presetKey as keyof typeof PERMISSION_PRESETS];
            onChange(preset.permissions);
        }
    }, [onChange]);

    // When individual permissions are toggled
    const handlePermissionToggle = useCallback((permission: ApiKeyPermission, checked: boolean) => {
        let newPermissions: ApiKeyPermission[];

        if (checked) {
            newPermissions = [...selectedPermissions, permission];
        } else {
            newPermissions = selectedPermissions.filter(p => p !== permission);
        }

        onChange(newPermissions);

        // Update the preset if it matches, otherwise set to custom
        const matchingPreset = findMatchingPreset(newPermissions);
        setSelectedPreset(matchingPreset || 'custom');
    }, [selectedPermissions, onChange]);

    // Get security level color
    const getSecurityLevelColor = (level: PermissionSecurityLevel) => {
        switch (level) {
            case PermissionSecurityLevel.Low:
                return palette.success.main;
            case PermissionSecurityLevel.Medium:
                return palette.warning.main;
            case PermissionSecurityLevel.High:
                return palette.error.main;
            default:
                return palette.text.primary;
        }
    };

    if (compact) {
        return (
            <FormControl fullWidth sx={{ mt: 2 }}>
                <FormLabel>API Key Permissions</FormLabel>
                <Select
                    value={selectedPreset}
                    onChange={(e) => handlePresetChange({ target: { value: e.target.value } } as any)}
                    fullWidth
                    sx={{ mt: 1 }}
                >
                    {Object.entries(PERMISSION_PRESETS).map(([key, preset]) => (
                        <MenuItem key={key} value={key}>
                            {preset.name}
                        </MenuItem>
                    ))}
                    <MenuItem value="custom">Custom Permissions</MenuItem>
                </Select>
                {selectedPreset === 'custom' && (
                    <Box mt={2}>
                        <FormGroup>
                            {Object.values(ApiKeyPermission).map((permission) => (
                                <FormControlLabel
                                    key={permission}
                                    control={
                                        <Checkbox
                                            checked={selectedPermissions.includes(permission)}
                                            onChange={(e) => handlePermissionToggle(permission, e.target.checked)}
                                        />
                                    }
                                    label={PERMISSION_NAMES[permission]}
                                />
                            ))}
                        </FormGroup>
                    </Box>
                )}
            </FormControl>
        );
    }

    return (
        <Box>
            <FormControl component="fieldset" fullWidth>
                <FormLabel component="legend">API Key Permission Level</FormLabel>
                <RadioGroup
                    value={selectedPreset}
                    onChange={handlePresetChange}
                >
                    {Object.entries(PERMISSION_PRESETS).map(([key, preset]) => (
                        <FormControlLabel
                            key={key}
                            value={key}
                            control={<Radio />}
                            label={
                                <Box>
                                    <Typography variant="body1">{preset.name}</Typography>
                                    <Typography variant="body2" color="text.secondary">{preset.description}</Typography>
                                </Box>
                            }
                        />
                    ))}
                    <FormControlLabel
                        value="custom"
                        control={<Radio />}
                        label={
                            <Box>
                                <Typography variant="body1">Custom Permissions</Typography>
                                <Typography variant="body2" color="text.secondary">Select individual permissions</Typography>
                            </Box>
                        }
                    />
                </RadioGroup>
            </FormControl>

            {selectedPreset === 'custom' && (
                <Box mt={3} ml={4}>
                    <FormControl component="fieldset" fullWidth>
                        <FormLabel component="legend">Individual Permissions</FormLabel>
                        <FormGroup>
                            {Object.values(ApiKeyPermission).map((permission) => (
                                <Box key={permission} mb={2}>
                                    <FormControlLabel
                                        control={
                                            <Checkbox
                                                checked={selectedPermissions.includes(permission)}
                                                onChange={(e) => handlePermissionToggle(permission, e.target.checked)}
                                            />
                                        }
                                        label={
                                            <Box display="flex" alignItems="center">
                                                <Typography variant="body1">{PERMISSION_NAMES[permission]}</Typography>
                                                <Chip
                                                    label={PERMISSION_SECURITY_LEVELS[permission].toUpperCase()}
                                                    size="small"
                                                    sx={{
                                                        ml: 1,
                                                        backgroundColor: getSecurityLevelColor(PERMISSION_SECURITY_LEVELS[permission]),
                                                        color: '#fff'
                                                    }}
                                                />
                                            </Box>
                                        }
                                    />
                                    <FormHelperText sx={{ ml: 4 }}>{PERMISSION_DESCRIPTIONS[permission]}</FormHelperText>
                                </Box>
                            ))}
                        </FormGroup>
                    </FormControl>
                </Box>
            )}
        </Box>
    );
};

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
                        Important: This key will only be shown once. You won&apos;t be able to see it again after closing this dialog.
                    </Typography>
                    <Box
                        position="relative"
                        p={2}
                        border={1}
                        borderColor="divider"
                        borderRadius={1}
                        bgcolor={palette.background.paper}
                    >
                        <Typography variant="body2" pr={4} sx={{ wordBreak: "break-all" }}>
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
                            <IconCommon name="Copy" />
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

type ApiKeyDialogProps = {
    open: boolean;
    onClose: () => void;
    keyData?: ApiKey | null; // If provided, we're in edit mode, otherwise create mode
    onCreateKey?: (input: ApiKeyCreateInput) => Promise<ApiKeyCreated | null>;
    onUpdateKey?: (input: ApiKeyUpdateInput) => void;
};

interface ApiKeyFormValues {
    name: string;
    limitHard: string;
    limitSoft: string;
    stopAtLimit: boolean;
    disabled: boolean;
    permissions: ApiKeyPermission[];
}

export function ApiKeyDialog({
    open,
    onClose,
    keyData,
    onCreateKey,
    onUpdateKey,
}: ApiKeyDialogProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [isSubmitting, setIsSubmitting] = useState(false);
    const { onProfileUpdate, profile } = useProfileQuery();
    const [updateInternalKey] = useLazyFetch<ApiKeyUpdateInput, ApiKey>(endpointsApiKey.updateOne);

    const isEditMode = !!keyData;

    const initialValues = useMemo<ApiKeyFormValues>(() => {
        if (isEditMode) {
            return {
                name: keyData?.name ?? "",
                limitHard: keyData?.limitHard ?? "",
                limitSoft: keyData?.limitSoft ?? "",
                stopAtLimit: keyData?.stopAtLimit ?? false,
                disabled: !!keyData?.disabledAt,
                permissions: extractPermissions(keyData),
            };
        } else {
            return {
                name: "",
                limitHard: "25000000000", // Default value
                limitSoft: "",
                stopAtLimit: true,
                disabled: false,
                permissions: PERMISSION_PRESETS.READ_ONLY.permissions,
            };
        }
    }, [keyData, isEditMode]);

    const handleSubmit = useCallback(async (values: ApiKeyFormValues) => {
        setIsSubmitting(true);
        try {
            if (isEditMode) {
                // Edit mode
                if (!keyData) {
                    console.error("This error shouldn't happen. Please report it.", { component: "ApiKeyDialog", function: "handleSubmit", keyData });
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
                    permissions: JSON.stringify(values.permissions),
                };

                if (onUpdateKey) {
                    onUpdateKey(input);
                } else {
                    fetchLazyWrapper<ApiKeyUpdateInput, ApiKey>({
                        fetch: updateInternalKey,
                        inputs: input,
                        onSuccess: (updatedKey) => {
                            const updatedKeys = profile?.apiKeys?.map(k => k.id === updatedKey.id ? updatedKey : k) ?? [];
                            onProfileUpdate({ ...profile, apiKeys: updatedKeys } as User);
                            onClose();
                        },
                    });
                }
            } else {
                // Create mode
                const input: ApiKeyCreateInput = {
                    name: values.name,
                    disabled: values.disabled,
                    limitHard: values.limitHard,
                    limitSoft: values.limitSoft.length > 0 ? values.limitSoft : null,
                    stopAtLimit: values.stopAtLimit,
                    permissions: JSON.stringify(values.permissions),
                };

                if (onCreateKey) {
                    await onCreateKey(input);
                    onClose();
                }
            }
        } catch (error) {
            console.error("Error handling API key", error);
        } finally {
            setIsSubmitting(false);
        }
    }, [isEditMode, keyData, onUpdateKey, updateInternalKey, profile, onClose, onCreateKey]);

    const dialogPaperProps = {
        sx: {
            bgcolor: "background.default",
        },
    };

    return (
        <Dialog
            open={open}
            onClose={onClose}
            PaperProps={dialogPaperProps}
            maxWidth="md"
            fullWidth
        >
            <DialogTitle>
                {isEditMode ? t("ApiKeyUpdate") : t("ApiKeyAdd")}
            </DialogTitle>
            <DialogContent>
                <Formik
                    enableReinitialize
                    initialValues={initialValues}
                    onSubmit={handleSubmit}
                >
                    {({ values, handleSubmit, setFieldValue }) => (
                        <form onSubmit={handleSubmit}>
                            <TextInput
                                fullWidth
                                isRequired
                                label={t("Name")}
                                placeholder={isEditMode ? undefined : "My API Key"}
                                value={values.name}
                                onChange={(e) => setFieldValue("name", e.target.value)}
                                sx={{ mb: 2, mt: 1 }}
                            />

                            <TextInput
                                fullWidth
                                isRequired
                                label={t("HardLimit")}
                                type="number"
                                value={values.limitHard}
                                onChange={(e) => setFieldValue("limitHard", e.target.value)}
                                sx={{ mb: 2 }}
                            />

                            <TextInput
                                fullWidth
                                label={t("SoftLimit")}
                                type="number"
                                value={values.limitSoft}
                                onChange={(e) => setFieldValue("limitSoft", e.target.value)}
                                sx={{ mb: 2 }}
                            />

                            <ApiKeyPermissionsSelector
                                selectedPermissions={values.permissions}
                                onChange={(permissions) => setFieldValue("permissions", permissions)}
                            />

                            <DialogActions sx={{ mt: 3, p: 0 }}>
                                <Button onClick={onClose} disabled={isSubmitting}>
                                    {t("Cancel")}
                                </Button>
                                <Button
                                    type="submit"
                                    variant="contained"
                                    disabled={isSubmitting || !values.name}
                                >
                                    {isEditMode ? t("Save") : t("Add")}
                                </Button>
                            </DialogActions>
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
                                <MenuItem value="" disabled>Select service</MenuItem>
                                {KNOWN_SERVICES.map(service => (
                                    <MenuItem key={service} value={service}>{service}</MenuItem>
                                ))}
                                <MenuItem value="Other">{t("Other")}</MenuItem>
                            </Select>
                            {values.serviceSelect === "Other" && (
                                <TextInput
                                    fullWidth
                                    isRequired={true}
                                    label="Custom service name"
                                    value={values.customService}
                                    onChange={(e) => setFieldValue("customService", e.target.value)}
                                    sx={{ mb: 2 }}
                                />
                            )}
                            <TextInput
                                fullWidth
                                isRequired={false}
                                label="Name"
                                value={values.name}
                                onChange={(e) => setFieldValue("name", e.target.value)}
                                sx={{ mb: 2 }}
                            />
                            <PasswordTextInput
                                fullWidth
                                isRequired={true}
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
    const [apiKeyDialogOpen, setApiKeyDialogOpen] = useState(false);
    const [newKey, setNewKey] = useState<string | null>(null);
    const [editingInternalKey, setEditingInternalKey] = useState<ApiKey | null>(null);
    const [editingExternalKey, setEditingExternalKey] = useState<ApiKeyExternal | null>(null);
    const [isEditingExternalKey, setIsEditingExternalKey] = useState(false);
    const [isCreatingExternalKey, setIsCreatingExternalKey] = useState(false);

    function onNewKeyDialogClose() {
        setNewKeyDialogOpen(false);
        setNewKey(null);
    }

    const handleCreateKey = useCallback(async (input: ApiKeyCreateInput): Promise<ApiKeyCreated | null> => {
        return new Promise((resolve, reject) => {
            fetchLazyWrapper<ApiKeyCreateInput, ApiKeyCreated>({
                fetch: createInternalKey,
                inputs: input,
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
                    resolve(data);
                },
                onError: (error) => {
                    reject(error);
                }
            });
        });
    }, [apiKeys, createInternalKey, onProfileUpdate, profile]);

    // Handler to revoke an API key
    const revokeInternalKey = useCallback((keyId: string) => {
        if (!profile) {
            console.error("This error shouldn't happen. Please report it.", { component: "SettingsApiView", function: "revokeInternalKey", profile });
            return;
        }
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
        setApiKeyDialogOpen(true);
    }, []);

    const handleUpdateInternalKey = useCallback((input: ApiKeyUpdateInput) => {
        fetchLazyWrapper<ApiKeyUpdateInput, ApiKey>({
            fetch: updateInternalKey,
            inputs: input,
            onSuccess: (updatedKey) => {
                const updatedKeys = profile?.apiKeys?.map(k => k.id === updatedKey.id ? updatedKey : k) ?? [];
                onProfileUpdate({ ...profile, apiKeys: updatedKeys } as User);
                setApiKeyDialogOpen(false);
                setEditingInternalKey(null);
            },
        });
    }, [updateInternalKey, profile, onProfileUpdate]);

    const handleOpenCreateDialog = useCallback(() => {
        setEditingInternalKey(null);
        setApiKeyDialogOpen(true);
    }, []);

    const handleApiKeyDialogClose = useCallback(() => {
        setApiKeyDialogOpen(false);
        setEditingInternalKey(null);
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

    function handleExternalKeyEditClose() {
        setIsEditingExternalKey(false);
        setEditingExternalKey(null);
    }

    return (
        <PageContainer size="fullSize">
            <ApiKeyViewDialog apiKey={newKey} onClose={onNewKeyDialogClose} open={newKeyDialogOpen} />
            <ApiKeyDialog
                open={apiKeyDialogOpen}
                onClose={handleApiKeyDialogClose}
                keyData={editingInternalKey}
                onCreateKey={handleCreateKey}
                onUpdateKey={handleUpdateInternalKey}
            />
            <ExternalKeyDialog
                isCreate={isCreatingExternalKey}
                keyData={editingExternalKey}
                onClose={handleExternalKeyEditClose}
                open={isCreatingExternalKey || (isEditingExternalKey && !!editingExternalKey)}
            />
            <ScrollBox>
                <Navbar title={t("Api", { count: 1 })} />
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
                            <FormTip
                                element={{
                                    id: "api-key-tip",
                                    icon: "Info",
                                    isMarkdown: false,
                                    label: "API keys can access your account with different permission levels. Only share them with trusted services.",
                                    type: FormStructureType.Tip,
                                }}
                                isEditing={false}
                                onUpdate={noop}
                                onDelete={noop}
                            />
                            {apiKeys.length > 0 ? (
                                apiKeys.map((key) => {
                                    function handleEdit() {
                                        handleEditInternalKey(key);
                                    }
                                    function handleRevoke() {
                                        revokeInternalKey(key.id);
                                    }

                                    return (
                                        <Box key={key.id} display="flex" justifyContent="space-between" alignItems="center" mb={2} p={1} border={1} borderColor="divider" borderRadius={1}>
                                            <Box>
                                                <Typography>{key.name}</Typography>
                                                <Typography variant="body2" color="text.secondary">
                                                    {getPermissionsDescription(key)}
                                                </Typography>
                                            </Box>
                                            <Box>
                                                <IconButton onClick={handleEdit}>
                                                    <IconCommon name="Edit" fill="secondary.main" />
                                                </IconButton>
                                                <IconButton onClick={handleRevoke}>
                                                    <IconCommon name="Delete" fill="error.main" />
                                                </IconButton>
                                            </Box>
                                        </Box>
                                    );
                                })
                            ) : (
                                <Typography variant="body1" color="text.secondary">No API keys found</Typography>
                            )}
                            <Button
                                onClick={handleOpenCreateDialog}
                                startIcon={<IconCommon name="Plus" />}
                                sx={{ mt: 2 }}
                                variant="contained"
                            >
                                Create New API Key
                            </Button>
                        </Box>
                        <Divider />
                        <Box>
                            <Title title={`${t("ApiKey", { count: 2 })} - External`} variant="subheader" addSidePadding={false} />
                            <FormTip
                                element={{
                                    id: "external-api-key-tip",
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
                                                    <IconCommon name="Edit" fill="secondary.main" />
                                                </IconButton>
                                                <IconButton onClick={handleDelete}>
                                                    <IconCommon name="Delete" fill="error.main" />
                                                </IconButton>
                                            </Box>
                                        </Box>
                                    );
                                })
                            ) : (
                                <Typography variant="body1" color="text.secondary">No external API keys found</Typography>
                            )}
                            <Button onClick={() => setIsCreatingExternalKey(true)} sx={{ mt: 2 }}>Add External Key</Button>
                        </Box>
                        <Divider />
                        {SUPPORTED_INTEGRATIONS.length > 0 && (
                            <Box>
                                <Title title={"External Services"} help={"Connect to external services"} variant="subheader" addSidePadding={false} />
                                <Box display="flex" flexWrap="wrap" gap={2}>
                                    {SUPPORTED_INTEGRATIONS.map((integration) => {
                                        const isConnected = profile?.apiKeysExternal?.some(k => k.service === integration.name && !k.disabledAt) ?? false;

                                        function handleClick() {
                                            window.open(integration.url, "_blank");
                                        }

                                        return (
                                            <Button
                                                key={integration.name}
                                                onClick={handleClick}
                                                variant="contained"
                                                startIcon={<IconFavicon href={integration.url} />}
                                                endIcon={isConnected ? <IconCommon name="Success" /> : null}
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
        </PageContainer>
    );
}
