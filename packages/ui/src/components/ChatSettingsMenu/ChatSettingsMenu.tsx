// Create a new ChatSettingsMenu component to consolidate chat/model settings

import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import { Dialog, DialogContent } from "../dialogs/Dialog/Dialog.js";
import FormGroup from "@mui/material/FormGroup";
import { Checkbox } from "../inputs/Checkbox.js";
import { IconButton } from "../buttons/IconButton.js";
import InputAdornment from "@mui/material/InputAdornment";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import MenuItem from "@mui/material/MenuItem";
import { Switch } from "../inputs/Switch/index.js";
import Tab from "@mui/material/Tab";
import Tabs from "@mui/material/Tabs";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useMediaQuery } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import type { ChatInviteShape, ListObject, ResourceVersion } from "@vrooli/shared";
import { type CanConnect, type Chat, type ChatParticipantShape, type LlmModel, ResourceType, type Resource, type ResourceSearchInput, endpointsResource, endpointsAuth, type OAuthInitiateInput } from "@vrooli/shared";
import { Form, Formik } from "formik";
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { getAvailableModels, getExistingAIConfig, getPreferredAvailableModel, type AvailableModel } from "../../api/ai.js";
import { SessionContext } from "../../contexts/session.js";
import { IconCommon } from "../../icons/Icons.js";
import { useModelPreferencesStore } from "../../stores/modelPreferencesStore.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { FindObjectDialog } from "../dialogs/FindObjectDialog/FindObjectDialog.js";
import { TranslatedAdvancedInput } from "../inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../inputs/AdvancedInput/styles.js";
import { LanguageInput } from "../inputs/LanguageInput/LanguageInput.js";

/**
 * Model configuration interface.
 */
export interface ModelConfig {
    model: AvailableModel;
    toolSettings?: Record<string, boolean>;
    requireConfirmation?: boolean;
}

/**
 * Share settings for chat link access.
 */
export interface ShareSettings {
    enabled: boolean;
    link?: string;
}

// Type for a connected integration (e.g., a Project)
export type Integration = Pick<ResourceVersion, "id" | "__typename" | "translations">; // Using ProjectVersion

// Type for external app connections
export interface ExternalApp {
    id: string;
    name: string;
    url?: string;
    description?: string;
    isConnected: boolean;
    authType?: "oauth" | "apikey" | "none";
}

// Type for the overall integration settings state
export interface IntegrationSettings {
    projects: CanConnect<Integration>[]; // List of connected projects
    externalApps: ExternalApp[]; // List of connected external apps
}

/**
 * Props for ChatSettingsMenu.
 */
export interface ChatSettingsMenuProps {
    open: boolean;
    onClose: () => void;
    /** Minimal chat object, needs __typename */
    chat: Pick<Chat, "id" | "publicId" | "translations" | "__typename">;
    /** Current chat participants */
    participants: ChatParticipantShape[];
    /** Pending chat invites */
    invites?: ChatInviteShape[];
    /** Integration settings */
    integrationSettings: IntegrationSettings;
    modelConfigs: ModelConfig[];
    shareSettings: ShareSettings;
    onModelConfigChange: (configs: ModelConfig[]) => void;
    onAddParticipant: (id: string) => void;
    /** Remove an existing participant */
    onRemoveParticipant: (id: string) => void;
    /** Cancel a pending invite */
    onCancelInvite?: (inviteId: string) => void;
    onUpdateDetails: (data: { name: string; description?: string }) => void;
    onToggleShare: (enabled: boolean) => void;
    onIntegrationSettingsChange: (settings: IntegrationSettings) => void;
    /** Callback to initiate chat deletion */
    onDeleteChat?: () => void;
}

/**
 * ChatSettingsMenu component renders a dialog with
 * vertical tabs for Model, Participants, Details, and Share settings.
 */
export function ChatSettingsMenu({
    open,
    onClose,
    chat,
    participants,
    invites = [],
    integrationSettings,
    modelConfigs,
    shareSettings,
    onModelConfigChange,
    onAddParticipant,
    onRemoveParticipant,
    onCancelInvite,
    onUpdateDetails,
    onToggleShare,
    onIntegrationSettingsChange,
    onDeleteChat,
}: ChatSettingsMenuProps) {
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down("md"));
    const session = useContext(SessionContext);
    const languages = getUserLanguages(session);
    const { t } = useTranslation();
    const [currentTab, setCurrentTab] = useState(0);

    const handleDeleteClick = useCallback(() => {
        if (!onDeleteChat) return;
        PubSub.get().publish("alertDialog", {
            messageKey: "DeleteConfirm",
            buttons: [{
                labelKey: "Delete",
                onClick: onDeleteChat,
            }, {
                labelKey: "Cancel",
            }],
        });
    }, [onDeleteChat, t]);

    const handleTabChange = useCallback((_: React.SyntheticEvent, idx: number) => {
        setCurrentTab(idx);
    }, []);

    return (
        <Dialog
            isOpen={open}
            onClose={onClose}
            size="full"
            title="Chat Settings"
        >
            <DialogContent sx={{ padding: 0 }}>
                <Box display="flex" height="100%">
                <Box
                    sx={{
                        display: "flex",
                        flexDirection: "column",
                        borderRight: 1,
                        borderColor: "divider",
                        width: isMobile ? theme.spacing(8) : 200,
                        height: "100%",
                    }}
                >
                    <Tabs
                        orientation="vertical"
                        value={currentTab}
                        onChange={handleTabChange}
                        sx={{
                            width: "100%",
                            "& .MuiTabs-flexContainerVertical": {
                                justifyContent: "flex-start",
                            },
                            "& .MuiTab-root": {
                                minHeight: "48px",
                                paddingTop: theme.spacing(1),
                                paddingBottom: theme.spacing(1),
                                justifyContent: "flex-start",
                                "&.Mui-selected": {
                                    "& svg": {
                                        fill: theme.palette.secondary.main,
                                    },
                                },
                            },
                        }}
                    >
                        <Tab icon={<IconCommon name="Bot" fill={currentTab === 0 ? "secondary.main" : undefined} />} iconPosition="start" label={isMobile ? undefined : "Model"} />
                        <Tab icon={<IconCommon name="Link" fill={currentTab === 1 ? "secondary.main" : undefined} />} iconPosition="start" label={isMobile ? undefined : "Integrations"} />
                        <Tab icon={<IconCommon name="Team" fill={currentTab === 2 ? "secondary.main" : undefined} />} iconPosition="start" label={isMobile ? undefined : "Participants"} />
                        <Tab icon={<IconCommon name="Info" fill={currentTab === 3 ? "secondary.main" : undefined} />} iconPosition="start" label={isMobile ? undefined : "Details"} />
                        <Tab icon={<IconCommon name="Export" fill={currentTab === 4 ? "secondary.main" : undefined} />} iconPosition="start" label={isMobile ? undefined : "Share"} />
                        {onDeleteChat && (
                            <ListItemButton
                                onClick={handleDeleteClick}
                                sx={{
                                    color: "error.main",
                                    marginTop: "auto",
                                    paddingTop: theme.spacing(1),
                                    paddingBottom: theme.spacing(1),
                                    paddingLeft: theme.spacing(2),
                                    justifyContent: "flex-start",
                                    gap: isMobile ? 0 : theme.spacing(1.5),
                                }}
                            >
                                <IconCommon name="Delete" fill="error.main" />
                                {!isMobile && t("Delete")}
                            </ListItemButton>
                        )}
                    </Tabs>
                </Box>

                <Box flex={1} overflow="auto" p={2} bgcolor="background.default">
                    {currentTab === 0 && (
                        <ModelConfigTab
                            modelConfigs={modelConfigs}
                            onChange={onModelConfigChange}
                        />
                    )}
                    {currentTab === 1 && (
                        <IntegrationsTab
                            settings={integrationSettings}
                            onChange={onIntegrationSettingsChange}
                        />
                    )}
                    {currentTab === 2 && (
                        <ParticipantsTab
                            participants={participants}
                            invites={invites}
                            onAdd={onAddParticipant}
                            onRemove={onRemoveParticipant}
                            onCancelInvite={onCancelInvite}
                        />
                    )}
                    {currentTab === 3 && (
                        <DetailsTab
                            chat={chat}
                            onUpdate={onUpdateDetails}
                        />
                    )}
                    {currentTab === 4 && (
                        <ShareLinkTab
                            enabled={shareSettings.enabled}
                            onToggle={onToggleShare}
                            chatPublicId={chat.publicId}
                        />
                    )}
                </Box>
                </Box>
            </DialogContent>
        </Dialog>
    );
}

// ---- Stub panels below; replace with real UI in each tab ----

/**
 * Props for the ModelConfigTab panel.
 */
export interface ModelConfigTabProps {
    modelConfigs: ModelConfig[];
    onChange: (configs: ModelConfig[]) => void;
}

// Define styles used in ModelConfigTab
const searchBoxStyle = { p: 2 } as const;
const noResultsStyle = { p: 2, textAlign: "center" } as const;
const searchInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                name="Search"
                size={20}
            />
        </InputAdornment>
    ),
} as const;

function ModelConfigTab({ modelConfigs, onChange }: ModelConfigTabProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down("md"));
    const [searchQuery, setSearchQuery] = useState("");
    const searchInputRef = useRef<HTMLInputElement>(null);

    // TODO: Handle multiple model configs if chat has multiple bots
    const currentConfig = modelConfigs?.[0];

    const availableModels = useMemo(() => {
        const config = getExistingAIConfig()?.service?.config;
        return config ? getAvailableModels(config) : [];
    }, []);

    // State for selected model - initialize from props or default
    const [selectedModel, setSelectedModel] = useState<AvailableModel | null>(
        currentConfig?.model || (availableModels.length > 0 ? availableModels[0] : null),
    );

    // Helper to safely initialize tool settings state
    const getInitialToolSettings = (config?: ModelConfig) => ({
        siteActions: config?.toolSettings?.siteActions ?? true,
        webSearch: config?.toolSettings?.webSearch ?? true,
        fileRetrieval: config?.toolSettings?.fileRetrieval ?? true,
    });

    const [toolSettings, setToolSettings] = useState(getInitialToolSettings(currentConfig));

    // State for requireConfirmation - initialize from props or default
    const [requireConfirmation, setRequireConfirmation] = useState<boolean>(
        currentConfig?.requireConfirmation || false,
    );

    // Update local state if props change (e.g., parent selects a different chat)
    useEffect(() => {
        try {
            // Use the validated preferred model if no current config, otherwise use current config
            let modelToUse = currentConfig?.model || null;
            
            if (!modelToUse && availableModels.length > 0) {
                const preferredModel = getPreferredAvailableModel(availableModels);
                modelToUse = preferredModel || availableModels[0];
            }
            
            setSelectedModel(modelToUse);
            setToolSettings(getInitialToolSettings(currentConfig));
            setRequireConfirmation(currentConfig?.requireConfirmation || false);
        } catch (error) {
            // Fallback to first available model
            if (availableModels.length > 0) {
                setSelectedModel(availableModels[0]);
            } else {
                setSelectedModel(null);
            }
            setToolSettings(getInitialToolSettings(currentConfig));
            setRequireConfirmation(currentConfig?.requireConfirmation || false);
        }
    }, [currentConfig, availableModels]);

    // Propagate changes up when local state is updated
    const triggerOnChange = useCallback((updates: Partial<ModelConfig>) => {
        // TODO: Handle multiple configs update
        const updatedConfig = { ...currentConfig, ...updates } as ModelConfig;
        onChange([updatedConfig]); // Assuming single config for now
    }, [currentConfig, onChange]);

    const handleModelSelect = useCallback((model: AvailableModel) => {
        setSelectedModel(model);
        triggerOnChange({ model });
        // Save the model preference to the store
        useModelPreferencesStore.getState().setPreferredModel(model.value);
    }, [triggerOnChange]);

    const handleToggleTool = useCallback((toolName: keyof typeof toolSettings) => {
        setToolSettings(prev => {
            const newSettings = { ...prev, [toolName]: !prev[toolName] };
            triggerOnChange({ toolSettings: newSettings });
            return newSettings;
        });
    }, [triggerOnChange]);

    const handleRequireConfirmationChange = useCallback((checked: boolean) => {
        setRequireConfirmation(checked);
        triggerOnChange({ requireConfirmation: checked });
    }, [triggerOnChange]);

    const handleSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(event.target.value);
    }, []);

    // Focus the search input (consider if needed in Dialog context)
    // useEffect(() => {
    //     if (searchInputRef.current) {
    //         setTimeout(() => { searchInputRef.current?.focus(); }, 100);
    //     }
    // }, []);

    const filteredModels = useMemo(() => {
        if (!searchQuery) return availableModels;
        const query = searchQuery.toLowerCase();
        return availableModels.filter(model =>
            t(model.name, { ns: "service" }).toLowerCase().includes(query) ||
            (model.description && t(model.description, { ns: "service", defaultValue: "" }).toLowerCase().includes(query)),
        );
    }, [availableModels, searchQuery, t]);

    // Group models by provider
    const modelsByProvider = useMemo(() => {
        const configData = getExistingAIConfig();
        const servicesInfo = configData?.service?.config;
        
        if (!servicesInfo) return {};
        
        const groups: Record<string, AvailableModel[]> = {};
        
        // Iterate through each service and its models
        for (const [serviceName, serviceInfo] of Object.entries(servicesInfo.services)) {
            if (!serviceInfo.enabled) continue;
            
            const serviceModels = filteredModels.filter(model => {
                // Check if this model belongs to this service by looking at the service's model values
                const serviceModelValues = Object.keys(serviceInfo.models);
                return serviceModelValues.includes(model.value);
            });
            
            if (serviceModels.length > 0) {
                groups[serviceInfo.name] = serviceModels;
            }
        }
        
        return groups;
    }, [filteredModels]);

    // Show message if no models are available at all
    if (availableModels.length === 0) {
        return (
            <Box>
                <Typography variant="h6" gutterBottom>
                    Model Configuration
                </Typography>
                <Box sx={{ p: 3, textAlign: "center" }}>
                    <Typography variant="body1" color="textSecondary" gutterBottom>
                        No AI models are currently available.
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                        Please check your connection and try again later.
                    </Typography>
                </Box>
            </Box>
        );
    }

    return (
        <Box sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
                Model Configuration
            </Typography>
            <Box sx={searchBoxStyle}>
                <TextField
                    fullWidth
                    placeholder="Search models"
                    value={searchQuery}
                    onChange={handleSearchChange}
                    variant="outlined"
                    size="small"
                    InputProps={searchInputProps}
                    inputRef={searchInputRef}
                />
            </Box>
            <Box sx={{ 
                display: "flex", 
                flexDirection: "column", 
                mt: 1, 
                flexGrow: 1,
                gap: 2 
            }}>
                <Box sx={{ 
                    overflowY: "auto", 
                    borderBottom: 1,
                    borderColor: "divider", 
                    pb: 2,
                    mb: 2,
                    maxHeight: "300px"
                }}>
                    {Object.keys(modelsByProvider).length === 0 ? (
                        <Box sx={noResultsStyle}>
                            <Typography variant="body2" color="textSecondary">
                                No models found
                            </Typography>
                        </Box>
                    ) : (
                        Object.entries(modelsByProvider).map(([providerName, models]) => (
                            <Box key={providerName}>
                                <Typography 
                                    variant="overline" 
                                    sx={{ 
                                        px: 2, 
                                        py: 1, 
                                        display: "block",
                                        fontWeight: "bold",
                                        color: "text.secondary",
                                        backgroundColor: "action.hover",
                                        borderBottom: 1,
                                        borderColor: "divider"
                                    }}
                                >
                                    {providerName}
                                </Typography>
                                {models.map((model) => (
                                    <MenuItem
                                        key={model.value}
                                        onClick={() => handleModelSelect(model)}
                                        selected={selectedModel?.value === model.value}
                                        sx={{ pl: 3 }}
                                    >
                                        <ListItemIcon>
                                            {selectedModel?.value === model.value && (
                                                <IconCommon
                                                    fill="background.textPrimary"
                                                    name="Complete"
                                                    size={20}
                                                />
                                            )}
                                        </ListItemIcon>
                                        <Box>
                                            <Typography variant="body1">
                                                {t(model.name, { ns: "service" })}
                                            </Typography>
                                            <Typography variant="caption" color="textSecondary">
                                                {t(model.description || "", { ns: "service", defaultValue: "" })}
                                            </Typography>
                                        </Box>
                                    </MenuItem>
                                ))}
                            </Box>
                        ))
                    )}
                </Box>
                <Box>
                    <Typography variant="subtitle1" gutterBottom>
                        Tool Configuration
                    </Typography>
                    <FormGroup>
                        {/* Require User Confirmation - moved to top */}
                        <Box 
                            mb={3} 
                            p={1.5}
                            borderRadius={1}
                            onClick={() => handleRequireConfirmationChange(!requireConfirmation)}
                            sx={{
                                cursor: "pointer",
                                transition: "background-color 0.2s ease",
                                "&:hover": {
                                    backgroundColor: "action.hover",
                                },
                            }}
                        >
                            <Box display="flex" alignItems="center" gap={1} mb={1}>
                                <IconCommon
                                    name="Warning"
                                    size={20}
                                    fill="secondary.main"
                                />
                                <Switch
                                    checked={requireConfirmation}
                                    onChange={handleRequireConfirmationChange}
                                    size="sm"
                                    label={
                                        requireConfirmation
                                            ? t("RequireUserConfirmation", { defaultValue: "Require User Confirmation" })
                                            : t("ManualToolUse", { defaultValue: "Manual Tool Use" })
                                    }
                                    labelPosition="right"
                                />
                            </Box>
                            <Typography variant="caption" color="textSecondary" display="block" ml={3}>
                                {t(
                                    "ToolUseConfirmationHelp",
                                    { defaultValue: "Toggle between manual and automatic tool usage, prompting the user when required." },
                                )}
                            </Typography>
                        </Box>
                        <Box 
                            mb={3} 
                            p={1.5}
                            borderRadius={1}
                            onClick={() => handleToggleTool("siteActions")}
                            sx={{
                                cursor: "pointer",
                                transition: "background-color 0.2s ease",
                                "&:hover": {
                                    backgroundColor: "action.hover",
                                },
                            }}
                        >
                            <Box display="flex" alignItems="center" gap={1} mb={1}>
                                <IconCommon
                                    name="Action"
                                    size={20}
                                    fill="secondary.main"
                                />
                                <Box display="flex" alignItems="center">
                                    <Checkbox
                                        checked={toolSettings.siteActions}
                                        onChange={() => handleToggleTool("siteActions")}
                                        size="sm"
                                    />
                                    <Typography variant="body2" ml={1}>
                                        {t("SiteActions", { defaultValue: "Site Actions" })}
                                    </Typography>
                                </Box>
                            </Box>
                            <Typography variant="caption" color="textSecondary" display="block" ml={3}>
                                {t(
                                    "SiteActionsHelp",
                                    { defaultValue: "Allow the model to perform site actions such as creating notes, running routines, etc." },
                                )}
                            </Typography>
                        </Box>
                        <Box 
                            mb={3} 
                            p={1.5}
                            borderRadius={1}
                            onClick={() => handleToggleTool("webSearch")}
                            sx={{
                                cursor: "pointer",
                                transition: "background-color 0.2s ease",
                                "&:hover": {
                                    backgroundColor: "action.hover",
                                },
                            }}
                        >
                            <Box display="flex" alignItems="center" gap={1} mb={1}>
                                <IconCommon
                                    name="Language"
                                    size={20}
                                    fill="secondary.main"
                                />
                                <Box display="flex" alignItems="center">
                                    <Checkbox
                                        checked={toolSettings.webSearch}
                                        onChange={() => handleToggleTool("webSearch")}
                                        size="sm"
                                    />
                                    <Typography variant="body2" ml={1}>
                                        {t("WebSearch", { defaultValue: "Web Search" })}
                                    </Typography>
                                </Box>
                            </Box>
                            <Typography variant="caption" color="textSecondary" display="block" ml={3}>
                                {t(
                                    "WebSearchHelp",
                                    { defaultValue: "Allow the model to perform web searches to retrieve real-time information." },
                                )}
                            </Typography>
                        </Box>
                        <Box 
                            mb={3} 
                            p={1.5}
                            borderRadius={1}
                            onClick={() => handleToggleTool("fileRetrieval")}
                            sx={{
                                cursor: "pointer",
                                transition: "background-color 0.2s ease",
                                "&:hover": {
                                    backgroundColor: "action.hover",
                                },
                            }}
                        >
                            <Box display="flex" alignItems="center" gap={1} mb={1}>
                                <IconCommon
                                    name="Link"
                                    size={20}
                                    fill="secondary.main"
                                />
                                <Box display="flex" alignItems="center">
                                    <Checkbox
                                        checked={toolSettings.fileRetrieval}
                                        onChange={() => handleToggleTool("fileRetrieval")}
                                        size="sm"
                                    />
                                    <Typography variant="body2" ml={1}>
                                        {t("FileRetrieval", { defaultValue: "File Retrieval" })}
                                    </Typography>
                                </Box>
                            </Box>
                            <Typography variant="caption" color="textSecondary" display="block" ml={3}>
                                {t(
                                    "FileRetrievalHelp",
                                    { defaultValue: "Allow the model to fetch and read files from the system." },
                                )}
                            </Typography>
                        </Box>
                    </FormGroup>
                </Box>
            </Box>
        </Box>
    );
}

/**
 * Props for the ParticipantsTab panel.
 */
export interface ParticipantsTabProps {
    /** Current chat participants */
    participants: ChatParticipantShape[];
    /** Pending chat invites */
    invites?: ChatInviteShape[];
    /** Add a participant by ID or from search */
    onAdd: (id: string) => void;
    /** Remove an existing participant */
    onRemove: (id: string) => void;
    /** Cancel a pending invite */
    onCancelInvite?: (inviteId: string) => void;
}

/**
 * Panel for managing chat invites and participants with filter and add functionality.
 */
function ParticipantsTab({ participants, invites = [], onAdd, onRemove, onCancelInvite }: ParticipantsTabProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const languages = getUserLanguages(session);

    // Search term state
    const [searchTerm, setSearchTerm] = useState<string>("");
    // Find dialog visibility
    const [isDialogOpen, setIsDialogOpen] = useState<boolean>(false);

    // Filter invites by user ID
    const filteredInvites = useMemo(
        () => invites!.filter(inv => inv.user.id.toLowerCase().includes(searchTerm.toLowerCase())),
        [invites, searchTerm],
    );

    // Filter participants by name or ID
    const filteredParticipants = useMemo(
        () => participants.filter(p => {
            const name = p.user.name ?? "";
            return (
                name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                p.user.id.toLowerCase().includes(searchTerm.toLowerCase())
            );
        }),
        [participants, searchTerm],
    );

    return (
        <Box sx={{ p: 3 }}>
            {/* Header with title and add button */}
            <Box sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 2 }}>
                <Typography variant="h6">
                    {t("Participants", { defaultValue: "Participants" })} ({participants.length})
                </Typography>
                <IconButton
                    size="sm"
                    variant="transparent"
                    aria-label={t("AddParticipant", { defaultValue: "Add Participant" })}
                    onClick={() => setIsDialogOpen(true)}
                >
                    <IconCommon name="Add" />
                </IconButton>
            </Box>

            {/* Search field */}
            <TextField
                fullWidth
                size="small"
                variant="outlined"
                placeholder={t("SearchParticipantsPlaceholder", { defaultValue: "Search participants..." })}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                sx={{ mb: 2 }}
            />

            {/* Invites section */}
            {filteredInvites.length > 0 && (
                <Box sx={{ mb: 2 }}>
                    <Typography variant="subtitle1" gutterBottom>
                        {t("Invites", { defaultValue: "Invites" })}
                    </Typography>
                    {filteredInvites.map(inv => (
                        <Box key={inv.id} sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 1 }}>
                            <Typography>{inv.user.id}</Typography>
                            <IconButton
                                size="sm"
                                variant="transparent"
                                aria-label={t("CancelInvite", { defaultValue: "Cancel Invite" })}
                                onClick={() => onCancelInvite?.(inv.id)}
                            >
                                <IconCommon name="Delete" />
                            </IconButton>
                        </Box>
                    ))}
                </Box>
            )}

            {/* Participants list */}
            <Box>
                <Typography variant="subtitle1" gutterBottom>
                    {t("Members", { defaultValue: "Members" })}
                </Typography>
                {filteredParticipants.map(p => {
                    const { title, adornments } = getDisplay(p.user as unknown as ListObject, languages);
                    return (
                        <Box key={p.id} sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", mb: 1 }}>
                            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                <Typography>{title}</Typography>
                                {adornments.map(({ Adornment, key }) => (
                                    <Box key={key}>{Adornment}</Box>
                                ))}
                            </Box>
                            <IconButton
                                size="sm"
                                variant="transparent"
                                aria-label={t("RemoveParticipant", { defaultValue: "Remove Participant" })}
                                onClick={() => onRemove(p.id)}
                            >
                                <IconCommon name="Delete" />
                            </IconButton>
                        </Box>
                    );
                })}
            </Box>

            {/* Dialog to find and add a participant */}
            {isDialogOpen && (
                <FindObjectDialog
                    find={"List" as const}
                    isOpen={isDialogOpen}
                    handleCancel={() => setIsDialogOpen(false)}
                    handleComplete={(data) => {
                        // data is a ListObject when find='List'
                        const obj = data as ListObject;
                        if (obj.id) {
                            onAdd(obj.id);
                        }
                        setIsDialogOpen(false);
                    }}
                    limitTo={["User"] as const}
                />
            )}
        </Box>
    );
}

/**
 * Props for the DetailsTab panel.
 */
export interface DetailsTabProps {
    /** The chat object holding translations */
    chat: Pick<Chat, "translations">;
    /** Called with updated name and description */
    onUpdate: (data: { name: string; description?: string }) => void;
}

function DetailsTab({ chat, onUpdate }: DetailsTabProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const languages = getUserLanguages(session);
    const [language, setLanguage] = useState<string>(languages[0]);

    // Use Formik to manage translation edits
    const initialValues = useMemo(() => ({ translations: chat.translations }), [chat.translations]);

    return (
        <Formik
            initialValues={initialValues}
            onSubmit={(values) => {
                const translation = values.translations.find(tr => tr.language === language);
                if (translation) {
                    onUpdate({ name: translation.name || "", description: translation.description ?? "" });
                }
            }}
        >
            {({ handleSubmit }) => (
                <Form onBlur={handleSubmit}>
                    <Box sx={{ p: 3 }}>
                        {/* Panel title */}
                        <Typography variant="h6" gutterBottom>
                            {t("ChatDetails", { defaultValue: "Chat Details" })}
                        </Typography>
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mb: 2 }}>
                        <TranslatedAdvancedInput
                            language={language}
                            name="name"
                            features={nameInputFeatures}
                            isRequired
                            title={t("Name", { defaultValue: "Name" })}
                            placeholder={t("ChatNamePlaceholder", { defaultValue: "Enter chat name..." })}
                        />
                        <TranslatedAdvancedInput
                            language={language}
                            name="description"
                            features={detailsInputFeatures}
                            title={t("Description", { defaultValue: "Description" })}
                            placeholder={t("ChatDescriptionPlaceholder", { defaultValue: "Enter chat description..." })}
                        />
                        <Box sx={{ display: "flex", gap: 2, mb: 2 }}>
                            <LanguageInput
                                currentLanguage={language}
                                languages={languages}
                                handleAdd={() => { /* no-op */ }}
                                handleDelete={() => { /* no-op */ }}
                                handleCurrent={setLanguage}
                            />
                        </Box>
                    </Box>
                    </Box>
                </Form>
            )}
        </Formik>
    );
}

/**
 * Props for the ShareLinkTab panel.
 */
export interface ShareLinkTabProps {
    /** Whether link sharing is enabled */
    enabled: boolean;
    /** Callback when toggled */
    onToggle: (enabled: boolean) => void;
    /** Chat ID for generating link */
    chatPublicId: string;
}

function ShareLinkTab({ enabled, onToggle, chatPublicId }: ShareLinkTabProps) {
    const { t } = useTranslation();
    // Compute the invite link from chatId
    const inviteLink = useMemo(() => `${window.location.origin}/chat/${chatPublicId}`, [chatPublicId]);

    return (
        <Box sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
                {t("ShareLink", { defaultValue: "Share Link" })}
            </Typography>
            <Switch
                checked={enabled}
                onChange={(checked) => onToggle(checked)}
                size="md"
                label={t("EnableShareLink", { defaultValue: "Enable share link" })}
                labelPosition="right"
            />
            {enabled && (
                <Box mt={2}>
                    <TextField
                        fullWidth
                        variant="outlined"
                        size="small"
                        value={inviteLink}
                        InputProps={{
                            readOnly: true,
                            endAdornment: (
                                <InputAdornment position="end">
                                    <IconButton onClick={() => navigator.clipboard.writeText(inviteLink)} size="sm" variant="transparent">
                                        <IconCommon name="Copy" />
                                    </IconButton>
                                </InputAdornment>
                            ),
                        }}
                    />
                </Box>
            )}
        </Box>
    );
}

// ---- Integrations Tab ----

/** Props for the IntegrationsTab panel. */
export interface IntegrationsTabProps {
    settings: IntegrationSettings;
    onChange: (settings: IntegrationSettings) => void;
}

function IntegrationsTab({ settings, onChange }: IntegrationsTabProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const languages = getUserLanguages(session);
    const [isProjectDialogOpen, setIsProjectDialogOpen] = useState(false);
    const [isExternalAppDialogOpen, setIsExternalAppDialogOpen] = useState(false);
    
    // Get external apps from settings with fallback
    const externalApps = settings.externalApps || [];

    const handleConnectProject = useCallback((projectId: string) => {
        const newSettings: IntegrationSettings = {
            ...settings,
            projects: [...settings.projects, { __typename: "ProjectVersion", id: projectId }],
        };
        onChange(newSettings);
    }, [settings, onChange]);

    const handleDisconnectProject = useCallback((projectId: string) => {
        const newSettings: IntegrationSettings = {
            ...settings,
            projects: settings.projects.filter(p => p.id !== projectId),
        };
        onChange(newSettings);
    }, [settings, onChange]);

    const handleConnectExternalApp = useCallback((app: ExternalApp) => {
        const newSettings: IntegrationSettings = {
            ...settings,
            externalApps: [...externalApps, app],
        };
        onChange(newSettings);
    }, [settings, externalApps, onChange]);

    const handleDisconnectExternalApp = useCallback((appId: string) => {
        const newSettings: IntegrationSettings = {
            ...settings,
            externalApps: externalApps.filter(app => app.id !== appId),
        };
        onChange(newSettings);
    }, [settings, externalApps, onChange]);

    return (
        <Box sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom sx={{ mb: 3 }}>
                {t("Integrations", { defaultValue: "Integrations" })}
            </Typography>

            {/* Project Connections Section */}
            <Box sx={{ mb: 4 }}>
                <Box sx={{ display: "flex", alignItems: "center", justifyContent: "flex-start", gap: 1, mb: 2 }}>
                    <Typography variant="subtitle1" sx={{ fontWeight: "bold" }}>
                        {t("ConnectedProjects", { defaultValue: "Connected Projects" })}
                    </Typography>
                    <IconButton
                        size="sm"
                        variant="transparent"
                        aria-label={t("ConnectProject", { defaultValue: "Connect Project" })}
                        onClick={() => setIsProjectDialogOpen(true)}
                    >
                        <IconCommon name="Add" />
                    </IconButton>
                </Box>
                <Typography variant="caption" color="textSecondary" sx={{ display: "block", mb: 3 }}>
                    {t("ConnectProjectsHelp", { defaultValue: "Connect projects to allow the chat model to retrieve and use files from them." })}
                </Typography>
                {settings.projects.length === 0 ? (
                    <Box sx={{ 
                        p: 3, 
                        border: 1, 
                        borderColor: "divider", 
                        borderRadius: 1, 
                        backgroundColor: "action.hover",
                        textAlign: "center"
                    }}>
                        <Typography variant="body2" color="textSecondary" sx={{ fontStyle: "italic" }}>
                            {t("NoProjectsConnected", { defaultValue: "No projects connected." })}
                        </Typography>
                    </Box>
                ) : (
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                        {settings.projects.map(proj => {
                            // Need to fetch full project details or pass them in for display
                            // Using placeholder display for now
                            const displayTitle = proj.id; // Replace with actual display logic later
                            return (
                                <Box 
                                    key={proj.id} 
                                    sx={{ 
                                        display: "flex", 
                                        alignItems: "center", 
                                        justifyContent: "space-between", 
                                        p: 2, 
                                        border: 1, 
                                        borderColor: "divider", 
                                        borderRadius: 1,
                                        backgroundColor: "background.paper"
                                    }}
                                >
                                    <Typography>{displayTitle}</Typography>
                                    <IconButton
                                        size="sm"
                                        variant="transparent"
                                        aria-label={t("DisconnectProject", { defaultValue: "Disconnect Project" })}
                                        onClick={() => handleDisconnectProject(proj.id!)}
                                    >
                                        <IconCommon name="Delete" />
                                    </IconButton>
                                </Box>
                            );
                        })}
                    </Box>
                )}
            </Box>

            {/* External Apps Section */}
            <Box sx={{ mb: 4 }}>
                <Box sx={{ display: "flex", alignItems: "center", justifyContent: "flex-start", gap: 1, mb: 2 }}>
                    <Typography variant="subtitle1" sx={{ fontWeight: "bold" }}>
                        {t("ExternalApps", { defaultValue: "External Apps" })}
                    </Typography>
                    <IconButton
                        size="sm"
                        variant="transparent"
                        aria-label={t("ConnectApp", { defaultValue: "Connect App" })}
                        onClick={() => setIsExternalAppDialogOpen(true)}
                    >
                        <IconCommon name="Add" />
                    </IconButton>
                </Box>
                <Typography variant="caption" color="textSecondary" sx={{ display: "block", mb: 3 }}>
                    {t("ExternalAppsHelp", { defaultValue: "Connect external applications (e.g., calendar, email) to enhance chat capabilities." })}
                </Typography>
                
                {/* External Apps List */}
                {externalApps.length === 0 ? (
                    <Box sx={{ 
                        p: 3, 
                        border: 1, 
                        borderColor: "divider", 
                        borderRadius: 1, 
                        backgroundColor: "action.hover",
                        textAlign: "center"
                    }}>
                        <Typography variant="body2" color="textSecondary" sx={{ fontStyle: "italic" }}>
                            {t("NoExternalAppsConnected", { defaultValue: "No external apps connected." })}
                        </Typography>
                    </Box>
                ) : (
                    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                        {externalApps.map((app) => (
                            <Box 
                                key={app.id} 
                                sx={{ 
                                    display: "flex", 
                                    alignItems: "center", 
                                    justifyContent: "space-between", 
                                    p: 2, 
                                    border: 1, 
                                    borderColor: "divider", 
                                    borderRadius: 1,
                                    backgroundColor: "background.paper"
                                }}
                            >
                                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                                    <IconCommon name="Link" size={20} />
                                    <Box>
                                        <Typography variant="body1">{app.name}</Typography>
                                        <Typography variant="caption" color="textSecondary">
                                            {app.url || app.description || "External service"}
                                        </Typography>
                                    </Box>
                                </Box>
                                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                    <Chip 
                                        label={app.isConnected ? "Connected" : "Disconnected"} 
                                        size="small" 
                                        color={app.isConnected ? "success" : "default"}
                                        variant={app.isConnected ? "filled" : "outlined"}
                                    />
                                    <IconButton
                                        size="sm"
                                        variant="transparent"
                                        aria-label={t("DisconnectApp", { defaultValue: "Disconnect App" })}
                                        onClick={() => handleDisconnectExternalApp(app.id)}
                                    >
                                        <IconCommon name="Delete" />
                                    </IconButton>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                )}
            </Box>

            {/* Dialog for finding projects */}
            {isProjectDialogOpen && (
                <FindObjectDialog
                    find={"List" as const}
                    isOpen={isProjectDialogOpen}
                    handleCancel={() => setIsProjectDialogOpen(false)}
                    handleComplete={(data) => {
                        const obj = data as ListObject;
                        if (obj.id) {
                            handleConnectProject(obj.id);
                        }
                        setIsProjectDialogOpen(false);
                    }}
                    limitTo={["Project"] as const}
                />
            )}

            {/* Simple External App Connection Dialog */}
            {isExternalAppDialogOpen && (
                <ExternalAppConnectionDialog
                    isOpen={isExternalAppDialogOpen}
                    onClose={() => setIsExternalAppDialogOpen(false)}
                    onConnect={handleConnectExternalApp}
                />
            )}
        </Box>
    );
}

// External App Connection Dialog Component
interface ExternalAppConnectionDialogProps {
    isOpen: boolean;
    onClose: () => void;
    onConnect: (app: ExternalApp) => void;
}

function ExternalAppConnectionDialog({ isOpen, onClose, onConnect }: ExternalAppConnectionDialogProps) {
    const { t } = useTranslation();
    const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
    const [selectedService, setSelectedService] = useState<string>("");
    const [customName, setCustomName] = useState("");
    const [customUrl, setCustomUrl] = useState("");
    
    // State for dynamic API resources
    const [apiResources, setApiResources] = useState<Resource[]>([]);
    const [isLoadingResources, setIsLoadingResources] = useState(false);
    
    // Fetch endpoints
    const [findResources] = useLazyFetch(endpointsResource.findMany);
    const [initiateOAuth] = useLazyFetch(endpointsAuth.oauthInitiate);
    
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
                    // Handle error silently for now
                },
            });
        } finally {
            setIsLoadingResources(false);
        }
    }, [findResources]);

    useEffect(() => {
        if (isOpen) {
            loadApiResources();
        }
    }, [isOpen, loadApiResources]);

    const handleConnect = useCallback(async () => {
        if (selectedService === "custom") {
            // Create custom app
            if (customName.trim()) {
                const newApp: ExternalApp = {
                    id: `custom-${Date.now()}`,
                    name: customName.trim(),
                    url: customUrl.trim() || undefined,
                    description: "Custom external service",
                    isConnected: true,
                    authType: "none",
                };
                onConnect(newApp);
                onClose();
            }
        } else if (selectedResource) {
            // Handle real API resource connection
            const version = selectedResource.versions?.[0];
            const translation = version?.translations?.[0];
            const config = version?.apiVersionConfig;
            
            if (!version || !translation || !config) return;
            
            if (config.authType === "oauth2" && selectedResource.id) {
                // Initiate OAuth flow
                try {
                    const result = await fetchLazyWrapper({
                        fetch: initiateOAuth,
                        inputs: {
                            resourceId: selectedResource.id,
                            redirectUri: `${window.location.origin}/settings/api/oauth/callback`,
                        } as OAuthInitiateInput,
                        onSuccess: (data) => {
                            // Redirect to OAuth authorization URL
                            window.location.href = data.authUrl;
                        },
                        onError: () => {
                            // Handle OAuth initiation error
                        },
                    });
                } catch (error) {
                    // Handle error
                }
            } else {
                // For non-OAuth services, create the connection directly
                const newApp: ExternalApp = {
                    id: selectedResource.id || `resource-${Date.now()}`,
                    name: translation.name || "External Service",
                    description: translation.description || "External service integration",
                    isConnected: true,
                    authType: config.authType === "oauth2" ? "oauth" : config.authType === "apikey" ? "apikey" : "none",
                };
                onConnect(newApp);
                onClose();
            }
        }
    }, [selectedService, selectedResource, customName, customUrl, onConnect, onClose, initiateOAuth]);

    const isValid = selectedService === "custom" ? customName.trim().length > 0 : selectedResource !== null;

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            title="Connect External App"
            size="md"
        >
            <DialogContent sx={{ p: 3 }}>
                <Typography variant="body2" color="textSecondary" sx={{ mb: 3 }}>
                    {t("ConnectExternalAppHelp", { defaultValue: "Connect external services to enhance chat functionality. Popular services are pre-configured for easy connection." })}
                </Typography>

                {/* Available Services */}
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: "bold" }}>
                    {t("AvailableServices", { defaultValue: "Available Services" })}
                </Typography>
                {isLoadingResources ? (
                    <Box sx={{ p: 3, textAlign: "center" }}>
                        <Typography variant="body2" color="text.secondary">Loading services...</Typography>
                    </Box>
                ) : apiResources.length === 0 ? (
                    <Box sx={{ p: 3, textAlign: "center", border: 1, borderColor: "divider", borderRadius: 1, mb: 3 }}>
                        <Typography variant="body2" color="text.secondary">No external services available.</Typography>
                    </Box>
                ) : (
                    <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: 1, mb: 3 }}>
                        {apiResources.map((resource) => {
                            const version = resource.versions?.[0];
                            const translation = version?.translations?.[0];
                            const config = version?.apiVersionConfig;
                            
                            if (!version || !translation) return null;
                            
                            const isSelected = selectedResource?.id === resource.id;
                            
                            return (
                                <Box
                                    key={resource.id}
                                    onClick={() => {
                                        setSelectedResource(resource);
                                        setSelectedService("");
                                    }}
                                    sx={{
                                        p: 2,
                                        border: 1,
                                        borderColor: isSelected ? "primary.main" : "divider",
                                        borderRadius: 1,
                                        cursor: "pointer",
                                        backgroundColor: isSelected ? "primary.light" : "background.paper",
                                        "&:hover": {
                                            backgroundColor: isSelected ? "primary.light" : "action.hover",
                                        },
                                    }}
                                >
                                    <Typography variant="body2" sx={{ fontWeight: "bold" }}>
                                        {translation.name}
                                    </Typography>
                                    <Typography variant="caption" color="textSecondary">
                                        {translation.description || "External service integration"}
                                    </Typography>
                                    {config && (
                                        <Typography variant="caption" color="primary" sx={{ display: "block", mt: 0.5 }}>
                                            Auth: {config.authType}
                                        </Typography>
                                    )}
                                </Box>
                            );
                        })}
                    </Box>
                )}

                {/* Custom Service */}
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: "bold" }}>
                    {t("CustomService", { defaultValue: "Custom Service" })}
                </Typography>
                <Box
                    onClick={() => {
                        setSelectedService("custom");
                        setSelectedResource(null);
                    }}
                    sx={{
                        p: 2,
                        border: 1,
                        borderColor: selectedService === "custom" ? "primary.main" : "divider",
                        borderRadius: 1,
                        cursor: "pointer",
                        backgroundColor: selectedService === "custom" ? "primary.light" : "background.paper",
                        "&:hover": {
                            backgroundColor: selectedService === "custom" ? "primary.light" : "action.hover",
                        },
                        mb: 2,
                    }}
                >
                    <Typography variant="body2" sx={{ fontWeight: "bold" }}>
                        {t("CustomApplication", { defaultValue: "Custom Application" })}
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                        {t("ConnectCustomService", { defaultValue: "Connect a custom external service or API" })}
                    </Typography>
                </Box>

                {/* Custom Service Form */}
                {selectedService === "custom" && (
                    <Box sx={{ mt: 2, p: 2, border: 1, borderColor: "divider", borderRadius: 1, backgroundColor: "action.hover" }}>
                        <TextField
                            fullWidth
                            label={t("ServiceName", { defaultValue: "Service Name" })}
                            value={customName}
                            onChange={(e) => setCustomName(e.target.value)}
                            placeholder={t("EnterServiceName", { defaultValue: "Enter service name..." })}
                            sx={{ mb: 2 }}
                        />
                        <TextField
                            fullWidth
                            label={t("ServiceUrl", { defaultValue: "Service URL (Optional)" })}
                            value={customUrl}
                            onChange={(e) => setCustomUrl(e.target.value)}
                            placeholder={t("EnterServiceUrl", { defaultValue: "Enter service URL..." })}
                        />
                    </Box>
                )}

                {/* Action Buttons */}
                <Box sx={{ display: "flex", justifyContent: "flex-end", gap: 2, mt: 3 }}>
                    <Button variant="outlined" onClick={onClose}>
                        {t("Cancel", { defaultValue: "Cancel" })}
                    </Button>
                    <Button 
                        variant="contained" 
                        onClick={handleConnect}
                        disabled={!isValid}
                    >
                        {t("Connect", { defaultValue: "Connect" })}
                    </Button>
                </Box>
            </DialogContent>
        </Dialog>
    );
}
