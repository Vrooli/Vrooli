// Create a new ChatSettingsMenu component to consolidate chat/model settings

import type { ChatInviteShape, ListObject, ResourceVersion } from '@local/shared';
import { CanConnect, Chat, ChatParticipantShape, LlmModel, getAvailableModels } from '@local/shared';
import { Box, Checkbox, Dialog, FormControlLabel, FormGroup, IconButton, InputAdornment, ListItemButton, ListItemIcon, MenuItem, Switch, Tab, Tabs, TextField, Typography, useMediaQuery, useTheme } from '@mui/material';
import { Form, Formik } from 'formik';
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getExistingAIConfig } from '../../api/ai.js';
import { SessionContext } from '../../contexts/session.js';
import { IconCommon } from '../../icons/Icons.js';
import { getDisplay } from '../../utils/display/listTools.js';
import { getUserLanguages } from '../../utils/display/translationTools.js';
import { PubSub } from '../../utils/pubsub.js';
import { FindObjectDialog } from '../dialogs/FindObjectDialog/FindObjectDialog.js';
import { TranslatedAdvancedInput } from '../inputs/AdvancedInput/AdvancedInput.js';
import { detailsInputFeatures, nameInputFeatures } from '../inputs/AdvancedInput/styles.js';
import { LanguageInput } from '../inputs/LanguageInput/LanguageInput.js';

/**
 * Model configuration interface.
 * TODO: extend with actual fields for tool settings, model choice, confirmation, etc.
 */
export interface ModelConfig {
    model: LlmModel;
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
export type Integration = Pick<ResourceVersion, 'id' | '__typename' | 'translations'>; // Using ProjectVersion

// Type for the overall integration settings state
export interface IntegrationSettings {
    projects: CanConnect<Integration>[]; // List of connected projects
    // Add other integration types here, e.g., externalApps: [], dataSources: []
}

/**
 * Props for ChatSettingsMenu.
 */
export interface ChatSettingsMenuProps {
    open: boolean;
    onClose: () => void;
    /** Minimal chat object, needs __typename */
    chat: Pick<Chat, 'id' | 'publicId' | 'translations' | '__typename'>;
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
    const isMobile = useMediaQuery(theme.breakpoints.down('md'));
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
            open={open}
            onClose={onClose}
            maxWidth="md"
            fullWidth
            PaperProps={{
                sx: {
                    borderRadius: theme.spacing(3),
                    height: `calc(100% - ${theme.spacing(8)})`,
                    maxHeight: '70vh',
                }
            }}
        >
            <Box display="flex" height="100%">
                <Box
                    sx={{
                        display: 'flex',
                        flexDirection: 'column',
                        borderRight: 1,
                        borderColor: 'divider',
                        width: isMobile ? theme.spacing(8) : 200,
                        height: '100%',
                    }}
                >
                    <Tabs
                        orientation="vertical"
                        value={currentTab}
                        onChange={handleTabChange}
                        sx={{
                            width: '100%',
                            "& .MuiTabs-flexContainerVertical": {
                                justifyContent: 'flex-start',
                            },
                            "& .MuiTab-root": {
                                minHeight: '48px',
                                paddingTop: theme.spacing(1),
                                paddingBottom: theme.spacing(1),
                                justifyContent: 'flex-start',
                            },
                        }}
                    >
                        <Tab icon={<IconCommon name="Bot" />} iconPosition="start" label={isMobile ? undefined : "Model"} />
                        <Tab icon={<IconCommon name="Link" />} iconPosition="start" label={isMobile ? undefined : "Integrations"} />
                        <Tab icon={<IconCommon name="Team" />} iconPosition="start" label={isMobile ? undefined : "Participants"} />
                        <Tab icon={<IconCommon name="Info" />} iconPosition="start" label={isMobile ? undefined : "Details"} />
                        <Tab icon={<IconCommon name="Export" />} iconPosition="start" label={isMobile ? undefined : "Share"} />
                        {onDeleteChat && (
                            <ListItemButton
                                onClick={handleDeleteClick}
                                sx={{
                                    color: 'error.main',
                                    marginTop: 'auto',
                                    paddingTop: theme.spacing(1),
                                    paddingBottom: theme.spacing(1),
                                    paddingLeft: theme.spacing(2),
                                    justifyContent: 'flex-start',
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
    const isMobile = useMediaQuery(theme.breakpoints.down('md'));
    const [searchQuery, setSearchQuery] = useState("");
    const searchInputRef = useRef<HTMLInputElement>(null);

    // TODO: Handle multiple model configs if chat has multiple bots
    const currentConfig = modelConfigs?.[0];

    const availableModels = useMemo(() => {
        const config = getExistingAIConfig()?.service?.config;
        return config ? getAvailableModels(config) : [];
    }, []);

    // State for selected model - initialize from props or default
    const [selectedModel, setSelectedModel] = useState<LlmModel | null>(
        currentConfig?.model || (availableModels.length > 0 ? availableModels[0] : null)
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
        currentConfig?.requireConfirmation || false
    );

    // Update local state if props change (e.g., parent selects a different chat)
    useEffect(() => {
        setSelectedModel(currentConfig?.model || (availableModels.length > 0 ? availableModels[0] : null));
        setToolSettings(getInitialToolSettings(currentConfig));
        setRequireConfirmation(currentConfig?.requireConfirmation || false);
    }, [currentConfig, availableModels]);

    // Propagate changes up when local state is updated
    const triggerOnChange = useCallback((updates: Partial<ModelConfig>) => {
        // TODO: Handle multiple configs update
        const updatedConfig = { ...currentConfig, ...updates } as ModelConfig;
        onChange([updatedConfig]); // Assuming single config for now
    }, [currentConfig, onChange]);

    const handleModelSelect = useCallback((model: LlmModel) => {
        setSelectedModel(model);
        triggerOnChange({ model });
    }, [triggerOnChange]);

    const handleToggleTool = useCallback((toolName: keyof typeof toolSettings) => {
        setToolSettings(prev => {
            const newSettings = { ...prev, [toolName]: !prev[toolName] };
            triggerOnChange({ toolSettings: newSettings });
            return newSettings;
        });
    }, [triggerOnChange]);

    const handleRequireConfirmationChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const isChecked = event.target.checked;
        setRequireConfirmation(isChecked);
        triggerOnChange({ requireConfirmation: isChecked });
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

    return (
        <Box>
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
            <Box sx={{ display: "flex", mt: 1, flexGrow: 1 }}>
                <Box sx={{ flex: 1, overflowY: "auto", borderRight: 1, borderColor: 'divider', pr: 1 }}>
                    {filteredModels.length === 0 ? (
                        <Box sx={noResultsStyle}>
                            <Typography variant="body2" color="textSecondary">
                                No models found
                            </Typography>
                        </Box>
                    ) : (
                        filteredModels.map((model) => (
                            <MenuItem
                                key={model.value}
                                onClick={() => handleModelSelect(model)}
                                selected={selectedModel?.value === model.value}
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
                        ))
                    )}
                </Box>
                <Box sx={{ width: isMobile ? 200 : 300, pl: 2 }}>
                    <Typography variant="subtitle1" gutterBottom>
                        Tool Configuration
                    </Typography>
                    <FormGroup>
                        <Box mb={2}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={toolSettings.siteActions}
                                        onChange={() => handleToggleTool("siteActions")}
                                        size="small"
                                    />
                                }
                                label={t("SiteActions", { defaultValue: "Site Actions" })}
                            />
                            <Typography variant="caption" color="textSecondary">
                                {t(
                                    "SiteActionsHelp",
                                    { defaultValue: "Allow the model to perform site actions such as creating notes, running routines, etc." }
                                )}
                            </Typography>
                        </Box>
                        <Box mb={2}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={toolSettings.webSearch}
                                        onChange={() => handleToggleTool("webSearch")}
                                        size="small"
                                    />
                                }
                                label={t("WebSearch", { defaultValue: "Web Search" })}
                            />
                            <Typography variant="caption" color="textSecondary">
                                {t(
                                    "WebSearchHelp",
                                    { defaultValue: "Allow the model to perform web searches to retrieve real-time information." }
                                )}
                            </Typography>
                        </Box>
                        <Box mb={2}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={toolSettings.fileRetrieval}
                                        onChange={() => handleToggleTool("fileRetrieval")}
                                        size="small"
                                    />
                                }
                                label={t("FileRetrieval", { defaultValue: "File Retrieval" })}
                            />
                            <Typography variant="caption" color="textSecondary">
                                {t(
                                    "FileRetrievalHelp",
                                    { defaultValue: "Allow the model to fetch and read files from the system." }
                                )}
                            </Typography>
                        </Box>
                    </FormGroup>
                    <Box mb={2}>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={requireConfirmation}
                                    onChange={handleRequireConfirmationChange}
                                    size="small"
                                />
                            }
                            label={
                                requireConfirmation
                                    ? t("RequireUserConfirmation", { defaultValue: "Require User Confirmation" })
                                    : t("ManualToolUse", { defaultValue: "Manual Tool Use" })
                            }
                            sx={{ mt: 1 }}
                        />
                        <Typography variant="caption" color="textSecondary">
                            {t(
                                "ToolUseConfirmationHelp",
                                { defaultValue: "Toggle between manual and automatic tool usage, prompting the user when required." }
                            )}
                        </Typography>
                    </Box>
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
    const [searchTerm, setSearchTerm] = useState<string>('');
    // Find dialog visibility
    const [isDialogOpen, setIsDialogOpen] = useState<boolean>(false);

    // Filter invites by user ID
    const filteredInvites = useMemo(
        () => invites!.filter(inv => inv.user.id.toLowerCase().includes(searchTerm.toLowerCase())),
        [invites, searchTerm]
    );

    // Filter participants by name or ID
    const filteredParticipants = useMemo(
        () => participants.filter(p => {
            const name = p.user.name ?? '';
            return (
                name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                p.user.id.toLowerCase().includes(searchTerm.toLowerCase())
            );
        }),
        [participants, searchTerm]
    );

    return (
        <Box>
            {/* Header with title and add button */}
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <Typography variant="h6">
                    {t('Participants', { defaultValue: 'Participants' })} ({participants.length})
                </Typography>
                <IconButton
                    size="small"
                    aria-label={t('AddParticipant', { defaultValue: 'Add Participant' })}
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
                placeholder={t('SearchParticipantsPlaceholder', { defaultValue: 'Search participants...' })}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                sx={{ mb: 2 }}
            />

            {/* Invites section */}
            {filteredInvites.length > 0 && (
                <Box sx={{ mb: 2 }}>
                    <Typography variant="subtitle1" gutterBottom>
                        {t('Invites', { defaultValue: 'Invites' })}
                    </Typography>
                    {filteredInvites.map(inv => (
                        <Box key={inv.id} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                            <Typography>{inv.user.id}</Typography>
                            <IconButton
                                size="small"
                                aria-label={t('CancelInvite', { defaultValue: 'Cancel Invite' })}
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
                    {t('Members', { defaultValue: 'Members' })}
                </Typography>
                {filteredParticipants.map(p => {
                    const { title, adornments } = getDisplay(p.user as unknown as ListObject, languages);
                    return (
                        <Box key={p.id} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Typography>{title}</Typography>
                                {adornments.map(({ Adornment, key }) => (
                                    <Box key={key}>{Adornment}</Box>
                                ))}
                            </Box>
                            <IconButton
                                size="small"
                                aria-label={t('RemoveParticipant', { defaultValue: 'Remove Participant' })}
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
                    find={'List' as const}
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
                    limitTo={['User'] as const}
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
    chat: Pick<Chat, 'translations'>;
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
                    onUpdate({ name: translation.name || '', description: translation.description ?? '' });
                }
            }}
        >
            {({ handleSubmit }) => (
                <Form onBlur={handleSubmit}>
                    {/* Panel title */}
                    <Typography variant="h6" gutterBottom>
                        {t('ChatDetails', { defaultValue: 'Chat Details' })}
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mb: 2 }}>
                        <TranslatedAdvancedInput
                            language={language}
                            name="name"
                            features={nameInputFeatures}
                            isRequired
                            title={t('Name', { defaultValue: 'Name' })}
                            placeholder={t('ChatNamePlaceholder', { defaultValue: 'Enter chat name...' })}
                        />
                        <TranslatedAdvancedInput
                            language={language}
                            name="description"
                            features={detailsInputFeatures}
                            title={t('Description', { defaultValue: 'Description' })}
                            placeholder={t('ChatDescriptionPlaceholder', { defaultValue: 'Enter chat description...' })}
                        />
                        <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
                            <LanguageInput
                                currentLanguage={language}
                                languages={languages}
                                handleAdd={() => { /* no-op */ }}
                                handleDelete={() => { /* no-op */ }}
                                handleCurrent={setLanguage}
                            />
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
        <Box>
            <Typography variant="h6" gutterBottom>
                {t('ShareLink', { defaultValue: 'Share Link' })}
            </Typography>
            <FormControlLabel
                control={
                    <Switch
                        checked={enabled}
                        onChange={(e) => onToggle(e.target.checked)}
                        size="small"
                    />
                }
                label={t('EnableShareLink', { defaultValue: 'Enable share link' })}
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
                                    <IconButton onClick={() => navigator.clipboard.writeText(inviteLink)} size="small">
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

    const handleConnectProject = useCallback((projectId: string) => {
        const newSettings: IntegrationSettings = {
            ...settings,
            projects: [...settings.projects, { __typename: 'ProjectVersion', id: projectId }],
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

    return (
        <Box>
            <Typography variant="h6" gutterBottom>
                {t('Integrations', { defaultValue: 'Integrations' })}
            </Typography>

            {/* Project Connections Section */}
            <Box sx={{ mb: 3 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', gap: 1, mb: 1 }}>
                    <Typography variant="subtitle1">
                        {t('ConnectedProjects', { defaultValue: 'Connected Projects' })}
                    </Typography>
                    <IconButton
                        size="small"
                        aria-label={t('ConnectProject', { defaultValue: 'Connect Project' })}
                        onClick={() => setIsProjectDialogOpen(true)}
                    >
                        <IconCommon name="Add" />
                    </IconButton>
                </Box>
                <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 2 }}>
                    {t('ConnectProjectsHelp', { defaultValue: 'Connect projects to allow the chat model to retrieve and use files from them.' })}
                </Typography>
                {settings.projects.length === 0 && (
                    <Typography variant="body2" color="textSecondary" sx={{ fontStyle: 'italic' }}>
                        {t('NoProjectsConnected', { defaultValue: 'No projects connected.' })}
                    </Typography>
                )}
                {settings.projects.map(proj => {
                    // Need to fetch full project details or pass them in for display
                    // Using placeholder display for now
                    const displayTitle = proj.id; // Replace with actual display logic later
                    return (
                        <Box key={proj.id} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                            <Typography>{displayTitle}</Typography>
                            <IconButton
                                size="small"
                                aria-label={t('DisconnectProject', { defaultValue: 'Disconnect Project' })}
                                onClick={() => handleDisconnectProject(proj.id!)}
                            >
                                <IconCommon name="Delete" />
                            </IconButton>
                        </Box>
                    );
                })}
            </Box>

            {/* Placeholder for other integrations */}
            <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle1">
                    {t('ExternalApps', { defaultValue: 'External Apps' })}
                </Typography>
                <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 2 }}>
                    {t('ExternalAppsHelp', { defaultValue: 'Connect external applications (e.g., calendar, email) to enhance capabilities.' })}
                </Typography>
                <Typography variant="body2" color="textSecondary" sx={{ fontStyle: 'italic' }}>
                    {t('ComingSoon', { defaultValue: 'Coming soon...' })}
                </Typography>
            </Box>

            {/* Dialog for finding projects */}
            {isProjectDialogOpen && (
                <FindObjectDialog
                    find={'List' as const}
                    isOpen={isProjectDialogOpen}
                    handleCancel={() => setIsProjectDialogOpen(false)}
                    handleComplete={(data) => {
                        const obj = data as ListObject;
                        if (obj.id) {
                            handleConnectProject(obj.id);
                        }
                        setIsProjectDialogOpen(false);
                    }}
                    limitTo={['Project'] as const}
                />
            )}
        </Box>
    );
}
