// AI_CHECK: TYPE_SAFETY=fixed-18-type-assertions-in-api-settings | LAST: 2025-06-30
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Chip from "@mui/material/Chip";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import Divider from "@mui/material/Divider";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormGroup from "@mui/material/FormGroup";
import FormHelperText from "@mui/material/FormHelperText";
import FormLabel from "@mui/material/FormLabel";
import { IconButton } from "../../components/buttons/IconButton.js";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Radio from "@mui/material/Radio";
import RadioGroup from "@mui/material/RadioGroup";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { ApiKeyPermission, DeleteType, FormStructureType, ResourceType, type ResourceSearchInput, type ResourceSearchResult, type Resource, ApiVersionConfig, endpointsActions, endpointsApiKey, endpointsApiKeyExternal, endpointsResource, endpointsAuth, noop, type ApiKey, type ApiKeyCreateInput, type ApiKeyCreated, type ApiKeyExternal, type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput, type ApiKeyUpdateInput, type DeleteOneInput, type Success, type User, type OAuthInitiateInput, type OAuthInitiateResult } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { type LazyRequestWithResult } from "../../api/types.js";
import { PageContainer } from "../../components/Page/Page.js";
import { DialogTitle } from "../../components/dialogs/DialogTitle/DialogTitle.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { FormTip } from "../../components/inputs/form/FormTip.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { IconCommon, IconFavicon } from "../../icons/Icons.js";
import { ScrollBox } from "../../styles.js";
import { BUSINESS_DATA } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { type SettingsApiViewProps } from "./types.js";

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
            ApiKeyPermission.ReadPublic,
        ],
    },
    STANDARD: {
        name: "Standard Access",
        description: "Read public and your private data",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate,
        ],
    },
    DEVELOPER: {
        name: "Developer Access",
        description: "Read and update your data",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate,
            ApiKeyPermission.WritePrivate,
        ],
    },
    FULL_ACCESS: {
        name: "Full Access (High Security Risk)",
        description: "Complete access to everything including authentication",
        permissions: [
            ApiKeyPermission.ReadPublic,
            ApiKeyPermission.ReadPrivate,
            ApiKeyPermission.WritePrivate,
            ApiKeyPermission.ReadAuth,
            ApiKeyPermission.WriteAuth,
        ],
    },
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
        const permissionsStr = (apiKey as { permissions?: string }).permissions;
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
    compact = false,
}: ApiKeyPermissionsSelectorProps) {
    const { palette } = useTheme();
    const [selectedPreset, setSelectedPreset] = useState<string | "custom">("READ_ONLY");

    // Update the preset when permissions change externally
    useEffect(() => {
        const matchingPreset = findMatchingPreset(selectedPermissions);
        setSelectedPreset(matchingPreset || "custom");
    }, [selectedPermissions]);

    // When a preset is selected
    const handlePresetChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const presetKey = event.target.value;
        setSelectedPreset(presetKey);

        if (presetKey !== "custom") {
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
        setSelectedPreset(matchingPreset || "custom");
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
                    onChange={(e) => handlePresetChange({ target: { value: e.target.value } } as React.ChangeEvent<HTMLInputElement>)}
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
                {selectedPreset === "custom" && (
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
        <Box sx={{ mt: 2 }}>
            <FormControl component="fieldset" fullWidth>
                <FormLabel component="legend" sx={{ fontWeight: 600, color: "text.primary", mb: 2 }}>
                    API Key Permission Level
                </FormLabel>
                <RadioGroup
                    value={selectedPreset}
                    onChange={handlePresetChange}
                    sx={{ gap: 1.5 }}
                >
                    {Object.entries(PERMISSION_PRESETS).map(([key, preset]) => (
                        <Box
                            key={key}
                            onClick={() => handlePresetChange({ target: { value: key } } as React.ChangeEvent<HTMLInputElement>)}
                            sx={{
                                border: 1,
                                borderColor: selectedPreset === key ? "primary.main" : "divider",
                                borderRadius: 2,
                                p: 2,
                                bgcolor: selectedPreset === key ? "rgba(15, 170, 170, 0.08)" : "background.paper",
                                transition: "all 0.2s ease",
                                cursor: "pointer",
                                "&:hover": {
                                    borderColor: "primary.main",
                                    bgcolor: "rgba(15, 170, 170, 0.08)",
                                },
                            }}
                        >
                            <FormControlLabel
                                value={key}
                                control={<Radio sx={{ mt: -0.5 }} />}
                                label={
                                    <Box>
                                        <Typography variant="body1" fontWeight={600}>
                                            {preset.name}
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                                            {preset.description}
                                        </Typography>
                                    </Box>
                                }
                                sx={{ m: 0, alignItems: "flex-start", width: "100%" }}
                            />
                        </Box>
                    ))}
                    <Box
                        onClick={(e) => {
                            // Don't trigger if clicking on a child checkbox
                            if ((e.target as HTMLElement).type !== "checkbox") {
                                handlePresetChange({ target: { value: "custom" } } as React.ChangeEvent<HTMLInputElement>);
                            }
                        }}
                        sx={{
                            border: 1,
                            borderColor: selectedPreset === "custom" ? "primary.main" : "divider",
                            borderRadius: 2,
                            p: 2,
                            bgcolor: selectedPreset === "custom" ? "rgba(15, 170, 170, 0.08)" : "background.paper",
                            transition: "all 0.2s ease",
                            cursor: "pointer",
                            "&:hover": {
                                borderColor: "primary.main",
                                bgcolor: "rgba(15, 170, 170, 0.08)",
                            },
                        }}
                    >
                        <FormControlLabel
                            value="custom"
                            control={<Radio sx={{ mt: -0.5 }} />}
                            label={
                                <Box>
                                    <Typography variant="body1" fontWeight={600}>
                                        Custom Permissions
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                                        Select individual permissions
                                    </Typography>
                                </Box>
                            }
                            sx={{ m: 0, alignItems: "flex-start", width: "100%" }}
                        />
                        
                        {/* Individual permissions inside the custom box when selected */}
                        {selectedPreset === "custom" && (
                            <Box sx={{ mt: 3, pl: 4, pr: 1 }}>
                                <Divider sx={{ mb: 2 }} />
                                <FormGroup sx={{ gap: 2 }}>
                                    {Object.values(ApiKeyPermission).map((permission) => (
                                        <Box 
                                            key={permission}
                                            onClick={() => handlePermissionToggle(permission, !selectedPermissions.includes(permission))}
                                            sx={{
                                                p: 2,
                                                border: 1,
                                                borderColor: "divider",
                                                borderRadius: 1,
                                                bgcolor: "background.default",
                                                cursor: "pointer",
                                                display: "flex",
                                                alignItems: "flex-start",
                                                gap: 2,
                                                "&:hover": {
                                                    borderColor: "primary.main",
                                                    bgcolor: "background.paper",
                                                },
                                            }}
                                        >
                                            <Checkbox
                                                checked={selectedPermissions.includes(permission)}
                                                onChange={(e) => {
                                                    e.stopPropagation();
                                                    handlePermissionToggle(permission, e.target.checked);
                                                }}
                                                sx={{
                                                    p: 0,
                                                    mt: 0.25,
                                                    color: "primary.main",
                                                    "&.Mui-checked": {
                                                        color: "primary.main",
                                                    },
                                                }}
                                            />
                                            <Box flex={1}>
                                                <Box display="flex" alignItems="center" gap={1}>
                                                    <Typography variant="body1" fontWeight={500}>
                                                        {PERMISSION_NAMES[permission]}
                                                    </Typography>
                                                    <Chip
                                                        label={PERMISSION_SECURITY_LEVELS[permission].toUpperCase()}
                                                        size="small"
                                                        sx={{
                                                            backgroundColor: getSecurityLevelColor(PERMISSION_SECURITY_LEVELS[permission]),
                                                            color: "#fff",
                                                            fontSize: "0.7rem",
                                                            height: "20px",
                                                        }}
                                                    />
                                                </Box>
                                                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                                                    {PERMISSION_DESCRIPTIONS[permission]}
                                                </Typography>
                                            </Box>
                                        </Box>
                                    ))}
                                </FormGroup>
                            </Box>
                        )}
                    </Box>
                </RadioGroup>
            </FormControl>
        </Box>
    );
}

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
            PaperProps={{
                sx: {
                    bgcolor: "background.default",
                    borderRadius: 3,
                    boxShadow: 24,
                    maxWidth: 600,
                },
            }}
            maxWidth={false}
        >
            <DialogTitle 
                id="api-key-view-dialog"
                sxs={{ 
                    root: {
                        bgcolor: "success.main", 
                        color: "success.contrastText",
                        borderRadius: "12px 12px 0 0",
                    },
                }}
            >
                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                    <IconCommon name="Success" size={24} />
                    <Typography variant="h5" component="h2" fontWeight={600}>
                        Your New API Key
                    </Typography>
                </Box>
            </DialogTitle>
            <DialogContent sx={{ p: 4 }}>
                <Box display="flex" flexDirection="column" gap={3}>
                    <Box>
                        <Typography variant="body1" gutterBottom sx={{ fontWeight: 500 }}>
                            Your API key has been created successfully!
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            Please store it in a secure location.
                        </Typography>
                    </Box>
                    
                    <Box 
                        sx={{ 
                            p: 2, 
                            border: 2, 
                            borderColor: "warning.main", 
                            borderRadius: 2, 
                            bgcolor: "warning.light",
                            display: "flex",
                            alignItems: "center",
                            gap: 2,
                        }}
                    >
                        <IconCommon name="Warning" fill="warning.main" size={24} />
                        <Typography variant="body2" color="warning.dark" fontWeight={500}>
                            Important: This key will only be shown once. You won&apos;t be able to see it again after closing this dialog.
                        </Typography>
                    </Box>

                    <Box
                        sx={{
                            position: "relative",
                            p: 3,
                            border: 1,
                            borderColor: "divider",
                            borderRadius: 2,
                            bgcolor: palette.background.paper,
                            fontFamily: "monospace",
                        }}
                    >
                        <Typography 
                            variant="body2" 
                            sx={{ 
                                wordBreak: "break-all",
                                fontSize: "0.9rem",
                                lineHeight: 1.5,
                                pr: 6,
                            }}
                        >
                            {apiKey}
                        </Typography>
                        <Button
                            variant="contained"
                            size="small"
                            startIcon={<IconCommon name="Copy" />}
                            sx={{
                                position: "absolute",
                                top: 12,
                                right: 12,
                                borderRadius: 1.5,
                            }}
                            onClick={handleCopyKey}
                        >
                            Copy
                        </Button>
                    </Box>
                </Box>
            </DialogContent>
            <DialogActions sx={{ 
                p: 3, 
                borderTop: 1, 
                borderColor: "divider",
                bgcolor: "background.paper",
            }}>
                <Button 
                    onClick={onClose}
                    variant="contained"
                    sx={{ borderRadius: 2, minWidth: 100 }}
                >
                    Close
                </Button>
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
                            onProfileUpdate({ ...(profile as User), apiKeys: updatedKeys });
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
            PaperProps={{
                sx: {
                    bgcolor: "background.default",
                    borderRadius: 3,
                    boxShadow: 24,
                },
            }}
            maxWidth="lg"
            fullWidth
        >
            <DialogTitle 
                id="api-key-edit-dialog"
                sxs={{ 
                    root: {
                        bgcolor: "primary.main", 
                        color: "primary.contrastText",
                        borderRadius: "12px 12px 0 0",
                    },
                }}
            >
                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                    <IconCommon name={isEditMode ? "Edit" : "Key"} size={24} />
                    <Typography variant="h5" component="h2" fontWeight={600}>
                        {isEditMode ? t("ApiKeyUpdate") : t("ApiKeyAdd")}
                    </Typography>
                </Box>
            </DialogTitle>
            <DialogContent sx={{ p: 0 }}>
                <Formik
                    enableReinitialize
                    initialValues={initialValues}
                    onSubmit={handleSubmit}
                >
                    {({ values, handleSubmit, setFieldValue }) => (
                        <form onSubmit={handleSubmit}>
                            <Box sx={{ p: 3 }}>
                                {/* Basic Information Section */}
                                <Box sx={{ mb: 4 }}>
                                    <Typography variant="h6" gutterBottom sx={{ fontWeight: 600, color: "text.primary", mb: 3 }}>
                                        Basic Information
                                    </Typography>
                                    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                                        <TextInput
                                            fullWidth
                                            isRequired
                                            label={t("Name")}
                                            placeholder={isEditMode ? undefined : "My API Key"}
                                            value={values.name}
                                            onChange={(e) => setFieldValue("name", e.target.value)}
                                            sx={{
                                                "& .MuiOutlinedInput-root": {
                                                    borderRadius: 2,
                                                },
                                            }}
                                        />
                                    </Box>
                                </Box>

                                <Divider sx={{ my: 4 }} />

                                {/* Rate Limits Section */}
                                <Box sx={{ mb: 4 }}>
                                    <Typography variant="h6" gutterBottom sx={{ fontWeight: 600, color: "text.primary", mb: 1 }}>
                                        Rate Limits
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                                        Control how many requests can be made with this API key
                                    </Typography>
                                    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                                        <TextInput
                                            fullWidth
                                            isRequired
                                            label={t("HardLimit")}
                                            type="number"
                                            value={values.limitHard}
                                            onChange={(e) => setFieldValue("limitHard", e.target.value)}
                                            sx={{
                                                "& .MuiOutlinedInput-root": {
                                                    borderRadius: 2,
                                                },
                                            }}
                                        />
                                        <TextInput
                                            fullWidth
                                            label={t("SoftLimit")}
                                            type="number"
                                            value={values.limitSoft}
                                            onChange={(e) => setFieldValue("limitSoft", e.target.value)}
                                            sx={{
                                                "& .MuiOutlinedInput-root": {
                                                    borderRadius: 2,
                                                },
                                            }}
                                        />
                                    </Box>
                                </Box>

                                <Divider sx={{ my: 4 }} />

                                {/* Permissions Section */}
                                <Box sx={{ mb: 4 }}>
                                    <Typography variant="h6" gutterBottom sx={{ fontWeight: 600, color: "text.primary", mb: 1 }}>
                                        Permissions
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                                        Choose what this API key can access. Higher permissions allow more actions but increase security risk.
                                    </Typography>
                                    <ApiKeyPermissionsSelector
                                        selectedPermissions={values.permissions}
                                        onChange={(permissions) => setFieldValue("permissions", permissions)}
                                    />
                                </Box>
                            </Box>
                            
                            <DialogActions sx={{ 
                                p: 3, 
                                borderTop: 1, 
                                borderColor: "divider",
                                bgcolor: "background.paper",
                                gap: 2,
                            }}>
                                <Button 
                                    onClick={onClose} 
                                    disabled={isSubmitting}
                                    variant="outlined"
                                    sx={{ borderRadius: 2 }}
                                >
                                    {t("Cancel")}
                                </Button>
                                <Button
                                    type="submit"
                                    variant="contained"
                                    disabled={isSubmitting || !values.name}
                                    startIcon={isSubmitting ? undefined : <IconCommon name={isEditMode ? "Save" : "Plus"} />}
                                    sx={{ borderRadius: 2, minWidth: 120 }}
                                >
                                    {isSubmitting ? "Processing..." : (isEditMode ? t("Save") : t("Add"))}
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
    resourceId?: string;
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
    const [isSubmitting, setIsSubmitting] = useState(false);

    const initialValues = useMemo<ExternalKeyDialogFormValues>(() => ({
        serviceSelect: keyData && KNOWN_SERVICES.includes(keyData.service) ? keyData.service : "Other",
        customService: !keyData || !KNOWN_SERVICES.includes(keyData.service) ? keyData?.service ?? "" : "",
        name: keyData?.name ?? "",
        key: "",
        resourceId: keyData?.resourceId ?? undefined,
    }), [keyData]);

    const handleSubmit = useCallback(async function handleSubmitCallback(values: ExternalKeyDialogFormValues) {
        setIsSubmitting(true);
        try {
            const service = values.serviceSelect === "Other" ? values.customService : values.serviceSelect;
            if (!service) {
                PubSub.get().publish("snack", { messageKey: "MissingRequiredField", severity: "Error" });
                setIsSubmitting(false);
                return;
            }
            if (!isCreate && !keyData) {
                console.error("This error shouldn't happen. Please report it.", { component: "ExternalKeyDialog", keyData });
                PubSub.get().publish("snack", { messageKey: "MissingRequiredField", severity: "Error" });
                setIsSubmitting(false);
                return;
            }
            const input = isCreate ? {
                service,
                name: values.name,
                key: values.key,
                resourceId: values.resourceId,
            } : {
                id: keyData.id,
                service,
                name: values.name,
                key: values.key,
                resourceId: values.resourceId,
            };
            const fetchFunc = isCreate ? createExternalKey : updateExternalKey;
            fetchLazyWrapper<typeof input, ApiKeyExternal>({
                fetch: fetchFunc as LazyRequestWithResult<typeof input, ApiKeyExternal>,
                inputs: input,
                onSuccess: (data) => {
                    if (isCreate) {
                        const updatedKeys = [...(profile?.apiKeysExternal ?? []), data];
                        onProfileUpdate({ ...(profile as User), apiKeysExternal: updatedKeys });
                    } else {
                        const updatedKeys = profile?.apiKeysExternal?.map(k => k.id === data.id ? data : k) ?? [];
                        onProfileUpdate({ ...(profile as User), apiKeysExternal: updatedKeys });
                    }
                    setIsSubmitting(false);
                    onClose();
                },
                onError: () => {
                    setIsSubmitting(false);
                },
            });
        } catch (error) {
            console.error("Error handling external API key", error);
            setIsSubmitting(false);
        }
    }, [createExternalKey, isCreate, keyData, onClose, onProfileUpdate, profile, updateExternalKey]);

    return (
        <Dialog 
            open={open} 
            onClose={onClose} 
            PaperProps={{
                sx: {
                    bgcolor: "background.default",
                    borderRadius: 3,
                    boxShadow: 24,
                },
            }}
            maxWidth="sm"
            fullWidth
        >
            <DialogTitle 
                id="external-api-key-dialog"
                sxs={{ 
                    root: {
                        bgcolor: "primary.main", 
                        color: "primary.contrastText",
                        borderRadius: "12px 12px 0 0",
                    },
                }}
            >
                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                    <IconCommon name={isCreate ? "Key" : "Edit"} size={24} />
                    <Typography variant="h5" component="h2" fontWeight={600}>
                        {isCreate ? "Add External API Key" : "Update External API Key"}
                    </Typography>
                </Box>
            </DialogTitle>
            <DialogContent sx={{ p: 0 }}>
                <Formik initialValues={initialValues} onSubmit={handleSubmit}>
                    {({ handleSubmit, values, setFieldValue }) => (
                        <form onSubmit={handleSubmit}>
                            <Box sx={{ p: 3 }}>
                                {/* Service Selection Section */}
                                <Box sx={{ mb: 4 }}>
                                    <Typography variant="h6" gutterBottom sx={{ fontWeight: 600, color: "text.primary", mb: 3 }}>
                                        Service Information
                                    </Typography>
                                    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                                        <FormControl fullWidth>
                                            <InputLabel id="service-select-label">Service</InputLabel>
                                            <Select
                                                labelId="service-select-label"
                                                value={values.serviceSelect}
                                                onChange={(e) => setFieldValue("serviceSelect", e.target.value)}
                                                label="Service"
                                                sx={{
                                                    borderRadius: 2,
                                                }}
                                            >
                                                {KNOWN_SERVICES.map(service => (
                                                    <MenuItem key={service} value={service}>
                                                        <Box display="flex" alignItems="center" gap={1}>
                                                            <IconFavicon href={`https://${service.toLowerCase()}.com`} size={20} />
                                                            {service}
                                                        </Box>
                                                    </MenuItem>
                                                ))}
                                                <MenuItem value="Other">
                                                    <Box display="flex" alignItems="center" gap={1}>
                                                        <IconCommon name="Settings" size={20} />
                                                        {t("Other")}
                                                    </Box>
                                                </MenuItem>
                                            </Select>
                                        </FormControl>
                                        
                                        {values.serviceSelect === "Other" && (
                                            <TextInput
                                                fullWidth
                                                isRequired={true}
                                                label="Custom Service Name"
                                                placeholder="e.g., MyCustomAPI"
                                                value={values.customService}
                                                onChange={(e) => setFieldValue("customService", e.target.value)}
                                                sx={{
                                                    "& .MuiOutlinedInput-root": {
                                                        borderRadius: 2,
                                                    },
                                                }}
                                            />
                                        )}
                                        
                                        <TextInput
                                            fullWidth
                                            isRequired={false}
                                            label="Description (Optional)"
                                            placeholder="e.g., Production API Key"
                                            value={values.name}
                                            onChange={(e) => setFieldValue("name", e.target.value)}
                                            sx={{
                                                "& .MuiOutlinedInput-root": {
                                                    borderRadius: 2,
                                                },
                                            }}
                                        />
                                    </Box>
                                </Box>

                                <Divider sx={{ my: 4 }} />

                                {/* API Key Section */}
                                <Box sx={{ mb: 4 }}>
                                    <Typography variant="h6" gutterBottom sx={{ fontWeight: 600, color: "text.primary", mb: 1 }}>
                                        API Key
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                                        Enter the API key provided by {values.serviceSelect === "Other" ? "your service" : values.serviceSelect}
                                    </Typography>
                                    <PasswordTextInput
                                        fullWidth
                                        isRequired={true}
                                        name="key"
                                        label={t("ApiKey")}
                                        placeholder="sk-..."
                                        value={values.key}
                                        onChange={(e) => setFieldValue("key", e.target.value)}
                                        sx={{
                                            "& .MuiOutlinedInput-root": {
                                                borderRadius: 2,
                                                fontFamily: "monospace",
                                            },
                                        }}
                                    />
                                </Box>

                                {/* Security Notice */}
                                <FormTip
                                    element={{
                                        id: "external-api-key-security-tip",
                                        icon: "Lock",
                                        isMarkdown: false,
                                        label: "Your API key will be encrypted and stored securely. We never share your keys with third parties.",
                                        type: FormStructureType.Tip,
                                    }}
                                    isEditing={false}
                                    onUpdate={noop}
                                    onDelete={noop}
                                />
                            </Box>
                            
                            <DialogActions sx={{ 
                                p: 3, 
                                borderTop: 1, 
                                borderColor: "divider",
                                bgcolor: "background.paper",
                                gap: 2,
                            }}>
                                <Button 
                                    onClick={onClose} 
                                    disabled={isSubmitting}
                                    variant="outlined"
                                    sx={{ borderRadius: 2 }}
                                >
                                    {t("Cancel")}
                                </Button>
                                <Button
                                    type="submit"
                                    variant="contained"
                                    disabled={isSubmitting || !values.key || (values.serviceSelect === "Other" && !values.customService)}
                                    startIcon={isSubmitting ? undefined : <IconCommon name={isCreate ? "Plus" : "Save"} />}
                                    sx={{ borderRadius: 2, minWidth: 120 }}
                                >
                                    {isSubmitting ? "Processing..." : (isCreate ? t("Add") : t("Update"))}
                                </Button>
                            </DialogActions>
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
    const [findResources] = useLazyFetch<ResourceSearchInput, ResourceSearchResult>(endpointsResource.findMany);
    const [initiateOAuth] = useLazyFetch<OAuthInitiateInput, OAuthInitiateResult>(endpointsAuth.oauthInitiate);

    const [newKeyDialogOpen, setNewKeyDialogOpen] = useState(false);
    const [apiKeyDialogOpen, setApiKeyDialogOpen] = useState(false);
    const [newKey, setNewKey] = useState<string | null>(null);
    const [editingInternalKey, setEditingInternalKey] = useState<ApiKey | null>(null);
    const [editingExternalKey, setEditingExternalKey] = useState<ApiKeyExternal | null>(null);
    const [isEditingExternalKey, setIsEditingExternalKey] = useState(false);
    const [isCreatingExternalKey, setIsCreatingExternalKey] = useState(false);
    
    // State for dynamic API resources
    const [apiResources, setApiResources] = useState<Resource[]>([]);
    const [isLoadingResources, setIsLoadingResources] = useState(false);

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
                },
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
                onProfileUpdate({ ...(profile as User), apiKeys: updatedKeys });
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
                onProfileUpdate({ ...(profile as User), apiKeysExternal: updatedKeys });
            },
        });
    }, [deleteOne, onProfileUpdate, profile]);

    const handleExternalKeyEditClose = useCallback(() => {
        setIsEditingExternalKey(false);
        setIsCreatingExternalKey(false);
        setEditingExternalKey(null);
    }, []);

    // Load API resources
    const loadApiResources = useCallback(async () => {
        setIsLoadingResources(true);
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
                    setApiResources(data.edges.map(edge => edge.node));
                },
                onError: () => {
                    PubSub.get().publish("snack", { 
                        messageKey: "FailedToLoadServices", 
                        severity: "Error", 
                    });
                },
            });
        } finally {
            setIsLoadingResources(false);
        }
    }, [findResources]);

    useEffect(() => {
        loadApiResources();
    }, [loadApiResources]);

    // Handle OAuth initiation
    const handleOAuthConnect = useCallback(async (resourceId: string) => {
        fetchLazyWrapper({
            fetch: initiateOAuth,
            inputs: {
                resourceId,
                redirectUri: `${window.location.origin}/settings/api/oauth/callback`,
            },
            onSuccess: (data) => {
                // Redirect to the OAuth authorization URL
                window.location.href = data.authUrl;
            },
            onError: () => {
                PubSub.get().publish("snack", { 
                    messageKey: "FailedToInitiateOAuth", 
                    severity: "Error", 
                });
            },
        });
    }, [initiateOAuth]);

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
                open={isCreatingExternalKey || isEditingExternalKey}
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
                            <Button 
                onClick={() => {
                    setEditingExternalKey(null);
                    setIsEditingExternalKey(false);
                    setIsCreatingExternalKey(true);
                }} 
                sx={{ mt: 2 }}
            >
                Add External Key
            </Button>
                        </Box>
                        <Divider />
                        {apiResources.length > 0 && (
                            <Box>
                                <Title title={"External Services"} help={"Connect to external services"} variant="subheader" addSidePadding={false} />
                                {isLoadingResources ? (
                                    <Typography variant="body2" color="text.secondary">Loading services...</Typography>
                                ) : (
                                    <Box display="flex" flexWrap="wrap" gap={2}>
                                        {apiResources.map((resource) => {
                                            const version = resource.versions?.[0];
                                            const translation = version?.translations?.[0];
                                            if (!version || !translation) return null;
                                            
                                            // Parse the config to check auth type
                                            let config: ApiVersionConfig | null = null;
                                            try {
                                                config = ApiVersionConfig.parse(version, console);
                                            } catch (e) {
                                                console.error("Failed to parse API config", e);
                                            }
                                            
                                            const isOAuth = config?.authentication?.type === "oauth2";
                                            const isApiKey = config?.authentication?.type === "apikey";
                                            const baseUrl = config?.callLink || version.callLink;
                                            
                                            // Check if user has connected this service
                                            const isConnected = profile?.apiKeysExternal?.some(k => k.resourceId === resource.id && !k.disabledAt) ?? false;

                                            function handleClick() {
                                                if (isOAuth) {
                                                    handleOAuthConnect(resource.id);
                                                } else if (isApiKey) {
                                                    // For API key services, open the external key dialog
                                                    setEditingExternalKey({
                                                        id: "",
                                                        createdAt: new Date(),
                                                        updatedAt: new Date(),
                                                        key: "",
                                                        disabledAt: null,
                                                        name: "",
                                                        service: translation.name,
                                                        resourceId: resource.id,
                                                    });
                                                    setIsCreatingExternalKey(true);
                                                } else if (baseUrl) {
                                                    // For services without auth, just open the URL
                                                    window.open(baseUrl, "_blank");
                                                }
                                            }

                                            return (
                                                <Button
                                                    key={resource.id}
                                                    onClick={handleClick}
                                                    variant="contained"
                                                    startIcon={baseUrl ? <IconFavicon href={baseUrl} /> : <IconCommon name="Api" />}
                                                    endIcon={isConnected ? <IconCommon name="Success" /> : null}
                                                    sx={{ 
                                                        position: "relative",
                                                        "&:hover .auth-type-chip": {
                                                            opacity: 1,
                                                        },
                                                    }}
                                                >
                                                    {translation.name}
                                                    {(isOAuth || isApiKey) && (
                                                        <Chip 
                                                            className="auth-type-chip"
                                                            label={isOAuth ? "OAuth" : "API Key"} 
                                                            size="small" 
                                                            sx={{ 
                                                                position: "absolute",
                                                                top: -8,
                                                                right: -8,
                                                                fontSize: "0.65rem",
                                                                height: "18px",
                                                                opacity: 0,
                                                                transition: "opacity 0.2s",
                                                            }}
                                                        />
                                                    )}
                                                </Button>
                                            );
                                        })}
                                    </Box>
                                )}
                            </Box>
                        )}
                    </Stack>
                </SettingsContent>
            </ScrollBox>
        </PageContainer>
    );
}
