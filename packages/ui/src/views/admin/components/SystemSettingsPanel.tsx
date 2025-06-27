import { IconCommon } from "../../../icons/Icons.js";
import {
    Box,
    Button,
    Card,
    CardContent,
    CardHeader,
    Divider,
    FormControl,
    Grid,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    TextField,
    Typography,
    Alert,
    Stack,
    Accordion,
    AccordionSummary,
    AccordionDetails,
    Chip,
    LinearProgress,
} from "@mui/material";
import { Switch } from "../../../components/inputs/Switch/Switch.js";
import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";

// System settings types
interface SystemSettings {
    site: {
        name: string;
        description: string;
        domain: string;
        maintenanceMode: boolean;
        registrationEnabled: boolean;
        maxUsersPerTeam: number;
        maxApiKeysPerUser: number;
    };
    security: {
        sessionTimeout: number;
        maxLoginAttempts: number;
        passwordMinLength: number;
        requireEmailVerification: boolean;
        enableTwoFactor: boolean;
        corsOrigins: string;
    };
    performance: {
        cacheEnabled: boolean;
        cacheTimeout: number;
        rateLimitEnabled: boolean;
        rateLimitPerMinute: number;
        maxUploadSize: number;
        enableCompression: boolean;
    };
    notifications: {
        emailEnabled: boolean;
        smtpHost: string;
        smtpPort: number;
        smtpUser: string;
        smtpSecure: boolean;
        fromEmail: string;
        fromName: string;
    };
    storage: {
        provider: "local" | "s3" | "gcs";
        maxStoragePerUser: number;
        cleanupEnabled: boolean;
        cleanupIntervalDays: number;
        backupEnabled: boolean;
        backupRetentionDays: number;
    };
    integrations: {
        stripeEnabled: boolean;
        stripePublishableKey: string;
        openaiEnabled: boolean;
        githubEnabled: boolean;
        discordEnabled: boolean;
        slackEnabled: boolean;
    };
}

/**
 * System configuration and settings management panel for administrators
 * Allows configuration of site-wide settings, security, performance, and integrations
 */
export const SystemSettingsPanel: React.FC = () => {
    const { t } = useTranslation();
    
    // State management
    const [settings, setSettings] = useState<SystemSettings | null>(null);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [changed, setChanged] = useState(false);
    const [expandedSection, setExpandedSection] = useState<string>("site");

    // Mock data - replace with actual API calls
    useEffect(() => {
        const mockSettings: SystemSettings = {
            site: {
                name: "Vrooli",
                description: "AI-powered platform for creating and sharing intelligent routines",
                domain: "vrooli.com",
                maintenanceMode: false,
                registrationEnabled: true,
                maxUsersPerTeam: 50,
                maxApiKeysPerUser: 10,
            },
            security: {
                sessionTimeout: 24,
                maxLoginAttempts: 5,
                passwordMinLength: 8,
                requireEmailVerification: true,
                enableTwoFactor: false,
                corsOrigins: "https://vrooli.com,https://app.vrooli.com",
            },
            performance: {
                cacheEnabled: true,
                cacheTimeout: 3600,
                rateLimitEnabled: true,
                rateLimitPerMinute: 60,
                maxUploadSize: 10,
                enableCompression: true,
            },
            notifications: {
                emailEnabled: true,
                smtpHost: "smtp.vrooli.com",
                smtpPort: 587,
                smtpUser: "noreply@vrooli.com",
                smtpSecure: true,
                fromEmail: "noreply@vrooli.com",
                fromName: "Vrooli",
            },
            storage: {
                provider: "s3",
                maxStoragePerUser: 1024,
                cleanupEnabled: true,
                cleanupIntervalDays: 30,
                backupEnabled: true,
                backupRetentionDays: 90,
            },
            integrations: {
                stripeEnabled: true,
                stripePublishableKey: "pk_test_***",
                openaiEnabled: true,
                githubEnabled: true,
                discordEnabled: false,
                slackEnabled: false,
            },
        };
        setSettings(mockSettings);
    }, []);

    const handleSave = async () => {
        if (!settings) return;
        
        setSaving(true);
        try {
            // TODO: Replace with actual API call
            console.log("Saving settings:", settings);
            await new Promise(resolve => setTimeout(resolve, 1000)); // Mock delay
            setChanged(false);
        } catch (error) {
            console.error("Failed to save settings:", error);
        } finally {
            setSaving(false);
        }
    };

    const handleReset = () => {
        // TODO: Fetch original settings from API
        setChanged(false);
    };

    const updateSetting = (section: keyof SystemSettings, key: string, value: any) => {
        if (!settings) return;
        
        setSettings(prev => ({
            ...prev!,
            [section]: {
                ...prev![section],
                [key]: value,
            },
        }));
        setChanged(true);
    };

    const handleAccordionChange = (panel: string) => (event: React.SyntheticEvent, isExpanded: boolean) => {
        setExpandedSection(isExpanded ? panel : "");
    };

    if (!settings) {
        return <LinearProgress />;
    }

    return (
        <Box>
            {/* Header */}
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 3 }}>
                <Box>
                    <Typography variant="h5" gutterBottom>
                        {t("Settings")}
                    </Typography>
                    <Typography variant="body1" color="text.secondary">
                        {t("Settings")}
                    </Typography>
                </Box>
                <Stack direction="row" spacing={2}>
                    {changed && (
                        <Chip 
                            label={t("Change")} 
                            color="warning" 
                            variant="outlined"
                        />
                    )}
                    <Button
                        variant="outlined"
                        startIcon={<IconCommon name="Refresh" />}
                        onClick={handleReset}
                        disabled={!changed}
                    >
                        {t("Reset")}
                    </Button>
                    <Button
                        variant="contained"
                        startIcon={<IconCommon name="Save" />}
                        onClick={handleSave}
                        disabled={!changed || saving}
                    >
                        {saving ? t("Loading") : t("Save")}
                    </Button>
                </Stack>
            </Box>

            {changed && (
                <Alert severity="warning" sx={{ mb: 3 }}>
                    {t("UnsavedChangesBeforeCancel")}
                </Alert>
            )}

            {/* Settings Sections */}
            <Stack spacing={2}>
                {/* Site Settings */}
                <Accordion 
                    expanded={expandedSection === "site"} 
                    onChange={handleAccordionChange("site")}
                >
                    <AccordionSummary expandIcon={<IconCommon name="ExpandMore" />}>
                        <Stack direction="row" spacing={2} alignItems="center">
                            <IconCommon name="Language" fill="primary.main" />
                            <Typography variant="h6">{t("Settings")}</Typography>
                        </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container spacing={3}>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    label={t("Name")}
                                    value={settings.site.name}
                                    onChange={(e) => updateSetting("site", "name", e.target.value)}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    label={t("Domain")}
                                    value={settings.site.domain}
                                    onChange={(e) => updateSetting("site", "domain", e.target.value)}
                                />
                            </Grid>
                            <Grid item xs={12}>
                                <TextField
                                    fullWidth
                                    multiline
                                    rows={2}
                                    label={t("Description")}
                                    value={settings.site.description}
                                    onChange={(e) => updateSetting("site", "description", e.target.value)}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("Max") + " " + t("User_other")}
                                    value={settings.site.maxUsersPerTeam}
                                    onChange={(e) => updateSetting("site", "maxUsersPerTeam", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("Max") + " " + t("ApiKey_other")}
                                    value={settings.site.maxApiKeysPerUser}
                                    onChange={(e) => updateSetting("site", "maxApiKeysPerUser", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.site.maintenanceMode}
                                    onChange={(checked) => updateSetting("site", "maintenanceMode", checked)}
                                    label={t("Settings")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.site.registrationEnabled}
                                    onChange={(checked) => updateSetting("site", "registrationEnabled", checked)}
                                    label={t("SignUp")}
                                />
                            </Grid>
                        </Grid>
                    </AccordionDetails>
                </Accordion>

                {/* Security Settings */}
                <Accordion 
                    expanded={expandedSection === "security"} 
                    onChange={handleAccordionChange("security")}
                >
                    <AccordionSummary expandIcon={<IconCommon name="ExpandMore" />}>
                        <Stack direction="row" spacing={2} alignItems="center">
                            <IconCommon name="Lock" fill="primary.main" />
                            <Typography variant="h6">{t("Security")}</Typography>
                        </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container spacing={3}>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("Settings")}
                                    value={settings.security.sessionTimeout}
                                    onChange={(e) => updateSetting("security", "sessionTimeout", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("MaxLoginAttempts")}
                                    value={settings.security.maxLoginAttempts}
                                    onChange={(e) => updateSetting("security", "maxLoginAttempts", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("PasswordMinLength")}
                                    value={settings.security.passwordMinLength}
                                    onChange={(e) => updateSetting("security", "passwordMinLength", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    label={t("CORSOrigins")}
                                    value={settings.security.corsOrigins}
                                    onChange={(e) => updateSetting("security", "corsOrigins", e.target.value)}
                                    helperText={t("CORSOriginsHelp")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.security.requireEmailVerification}
                                    onChange={(checked) => updateSetting("security", "requireEmailVerification", checked)}
                                    label={t("Email") + " " + t("Verified")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.security.enableTwoFactor}
                                    onChange={(checked) => updateSetting("security", "enableTwoFactor", checked)}
                                    label={t("Authentication")}
                                />
                            </Grid>
                        </Grid>
                    </AccordionDetails>
                </Accordion>

                {/* Performance Settings */}
                <Accordion 
                    expanded={expandedSection === "performance"} 
                    onChange={handleAccordionChange("performance")}
                >
                    <AccordionSummary expandIcon={<IconCommon name="ExpandMore" />}>
                        <Stack direction="row" spacing={2} alignItems="center">
                            <IconCommon name="Stats" fill="primary.main" />
                            <Typography variant="h6">{t("PerformanceSettings")}</Typography>
                        </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container spacing={3}>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("CacheTimeoutSeconds")}
                                    value={settings.performance.cacheTimeout}
                                    onChange={(e) => updateSetting("performance", "cacheTimeout", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("RateLimitPerMinute")}
                                    value={settings.performance.rateLimitPerMinute}
                                    onChange={(e) => updateSetting("performance", "rateLimitPerMinute", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("MaxUploadSizeMB")}
                                    value={settings.performance.maxUploadSize}
                                    onChange={(e) => updateSetting("performance", "maxUploadSize", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.performance.cacheEnabled}
                                    onChange={(checked) => updateSetting("performance", "cacheEnabled", checked)}
                                    label={t("Caching")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.performance.rateLimitEnabled}
                                    onChange={(checked) => updateSetting("performance", "rateLimitEnabled", checked)}
                                    label={t("Settings")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.performance.enableCompression}
                                    onChange={(checked) => updateSetting("performance", "enableCompression", checked)}
                                    label={t("Compress")}
                                />
                            </Grid>
                        </Grid>
                    </AccordionDetails>
                </Accordion>

                {/* Storage Settings */}
                <Accordion 
                    expanded={expandedSection === "storage"} 
                    onChange={handleAccordionChange("storage")}
                >
                    <AccordionSummary expandIcon={<IconCommon name="ExpandMore" />}>
                        <Stack direction="row" spacing={2} alignItems="center">
                            <IconCommon name="Save" fill="primary.main" />
                            <Typography variant="h6">{t("StorageSettings")}</Typography>
                        </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container spacing={3}>
                            <Grid item xs={12} md={6}>
                                <FormControl fullWidth>
                                    <InputLabel>{t("StorageProvider")}</InputLabel>
                                    <Select
                                        value={settings.storage.provider}
                                        onChange={(e) => updateSetting("storage", "provider", e.target.value)}
                                    >
                                        <MenuItem value="local">{t("LocalStorage")}</MenuItem>
                                        <MenuItem value="s3">{t("AmazonS3")}</MenuItem>
                                        <MenuItem value="gcs">{t("GoogleCloudStorage")}</MenuItem>
                                    </Select>
                                </FormControl>
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("MaxStoragePerUserMB")}
                                    value={settings.storage.maxStoragePerUser}
                                    onChange={(e) => updateSetting("storage", "maxStoragePerUser", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("CleanupIntervalDays")}
                                    value={settings.storage.cleanupIntervalDays}
                                    onChange={(e) => updateSetting("storage", "cleanupIntervalDays", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    type="number"
                                    label={t("BackupRetentionDays")}
                                    value={settings.storage.backupRetentionDays}
                                    onChange={(e) => updateSetting("storage", "backupRetentionDays", parseInt(e.target.value))}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.storage.cleanupEnabled}
                                    onChange={(checked) => updateSetting("storage", "cleanupEnabled", checked)}
                                    label={t("Clear")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.storage.backupEnabled}
                                    onChange={(checked) => updateSetting("storage", "backupEnabled", checked)}
                                    label={t("Export")}
                                />
                            </Grid>
                        </Grid>
                    </AccordionDetails>
                </Accordion>

                {/* Integration Settings */}
                <Accordion 
                    expanded={expandedSection === "integrations"} 
                    onChange={handleAccordionChange("integrations")}
                >
                    <AccordionSummary expandIcon={<IconCommon name="ExpandMore" />}>
                        <Stack direction="row" spacing={2} alignItems="center">
                            <IconCommon name="External" fill="primary.main" />
                            <Typography variant="h6">{t("IntegrationSettings")}</Typography>
                        </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Grid container spacing={3}>
                            <Grid item xs={12}>
                                <Alert severity="info">
                                    {t("IntegrationSettingsInfo")}
                                </Alert>
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <TextField
                                    fullWidth
                                    label={t("StripePublishableKey")}
                                    value={settings.integrations.stripePublishableKey}
                                    onChange={(e) => updateSetting("integrations", "stripePublishableKey", e.target.value)}
                                    type="password"
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.integrations.stripeEnabled}
                                    onChange={(checked) => updateSetting("integrations", "stripeEnabled", checked)}
                                    label={t("StripeEnabled")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.integrations.openaiEnabled}
                                    onChange={(checked) => updateSetting("integrations", "openaiEnabled", checked)}
                                    label={t("OpenAIEnabled")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.integrations.githubEnabled}
                                    onChange={(checked) => updateSetting("integrations", "githubEnabled", checked)}
                                    label={t("GitHubEnabled")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.integrations.discordEnabled}
                                    onChange={(checked) => updateSetting("integrations", "discordEnabled", checked)}
                                    label={t("DiscordEnabled")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                <Switch
                                    variant="default"
                                    size="md"
                                    checked={settings.integrations.slackEnabled}
                                    onChange={(checked) => updateSetting("integrations", "slackEnabled", checked)}
                                    label={t("SlackEnabled")}
                                />
                            </Grid>
                        </Grid>
                    </AccordionDetails>
                </Accordion>
            </Stack>
        </Box>
    );
};
