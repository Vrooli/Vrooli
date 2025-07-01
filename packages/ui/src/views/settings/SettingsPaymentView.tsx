// AI_CHECK: TYPE_SAFETY=fixed-mui-imports-to-named-imports-and-security-icon | LAST: 2025-06-30
import { Box, Button, Stack, Typography, useTheme, Divider, Paper, Slider, Switch, FormControlLabel } from "@mui/material";
import { Tooltip } from "../../components/Tooltip/Tooltip.js";
import { IconButton } from "../../components/buttons/IconButton.js";
import { API_CREDITS_PREMIUM, LINKS, PaymentType, CreditConfig, endpointsUser, type CreditConfigObject, type ProfileUpdateInput, type User } from "@vrooli/shared";
import { useContext, useMemo, useState, useEffect, useCallback, useRef } from "react";
import { useTranslation } from "react-i18next";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { SessionContext } from "../../contexts/session.js";
import { useStripe } from "../../hooks/useStripe.js";
import { IconCommon } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { type SettingsPaymentViewProps } from "./types.js";
import { fetchWrapper } from "../../api/fetchWrapper.js";
import { PubSub } from "../../utils/pubsub.js";

export function SettingsPaymentView({
    display,
    onClose,
}: SettingsPaymentViewProps) {
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    
    // Credit settings state
    const [creditConfig, setCreditConfig] = useState<CreditConfig>(new CreditConfig());
    const [isLoadingSettings, setIsLoadingSettings] = useState(false);
    const [isSavingSettings, setIsSavingSettings] = useState(false);
    const [saveError, setSaveError] = useState<string | null>(null);
    const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    
    // Initialize credit settings from user data
    useEffect(() => {
        if (currentUser.creditSettings) {
            setCreditConfig(new CreditConfig(currentUser.creditSettings));
        }
    }, [currentUser.creditSettings]);

    const {
        prices,
        startCheckout,
        redirectToCustomerPortal,
    } = useStripe();
    
    // Save credit settings to API
    const saveCreditsSettings = useCallback(async (newConfig: CreditConfig) => {
        if (isSavingSettings) return;
        
        setIsSavingSettings(true);
        
        await fetchWrapper<ProfileUpdateInput, User>({
            endpoint: endpointsUser.profileUpdate.endpoint,
            method: endpointsUser.profileUpdate.method,
            inputs: {
                id: currentUser.id,
                creditSettings: newConfig.toObject(),
            },
            onSuccess: (data) => {
                // Update local state
                setCreditConfig(newConfig);
                PubSub.get().publish("snack", {
                    messageKey: "SettingsSaved",
                    severity: "Success",
                });
            },
            onError: () => {
                PubSub.get().publish("snack", {
                    messageKey: "ErrorUnknown",
                    severity: "Error",
                });
            },
            onCompleted: () => {
                setIsSavingSettings(false);
            },
        });
    }, [currentUser.id, isSavingSettings]);
    
    const handleDonationToggle = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        try {
            const newConfig = new CreditConfig(creditConfig.toObject());
            newConfig.updateDonationSettings({ enabled: event.target.checked });
            setCreditConfig(newConfig); // Update UI immediately
        } catch (error) {
            console.error("Failed to update donation settings:", error);
            PubSub.get().publish("snack", {
                messageKey: "ErrorUnknown",
                severity: "Error",
            });
        }
    }, [creditConfig]);
    
    const handleDonationPercentageChange = useCallback((_: Event, value: number | number[]) => {
        try {
            const percentage = Array.isArray(value) ? value[0] : value;
            
            // Validate percentage range
            if (percentage < 0 || percentage > 100) {
                PubSub.get().publish("snack", {
                    message: t("DonationPercentageMustBeBetween0And100"),
                    severity: "Error",
                });
                return;
            }
            
            const newConfig = new CreditConfig(creditConfig.toObject());
            newConfig.updateDonationSettings({ percentage });
            setCreditConfig(newConfig); // Update UI immediately
        } catch (error) {
            console.error("Failed to update donation percentage:", error);
            PubSub.get().publish("snack", {
                messageKey: "ErrorUnknown",
                severity: "Error",
            });
        }
    }, [creditConfig, t]);
    
    // Debounced save effect for all credit settings changes
    useEffect(() => {
        // Clear previous timeout
        if (debounceTimeoutRef.current) {
            clearTimeout(debounceTimeoutRef.current);
        }
        
        // Set new timeout
        debounceTimeoutRef.current = setTimeout(() => {
            saveCreditsSettings(creditConfig);
        }, 1000);
        
        // Cleanup function
        return () => {
            if (debounceTimeoutRef.current) {
                clearTimeout(debounceTimeoutRef.current);
            }
        };
    }, [creditConfig, saveCreditsSettings]);
    
    const getCreditsPerMonth = () => {
        return Number(API_CREDITS_PREMIUM);
    };
    
    const calculateDonationAmount = () => {
        // For now, we estimate based on monthly allocation
        // TODO: Connect to actual free credits balance API
        const estimatedFreeCredits = Math.min(Number(currentUser.credits), getCreditsPerMonth());
        return Math.floor((estimatedFreeCredits * creditConfig.donation.percentage) / 100);
    };

    return (
        <ScrollBox>
            <Navbar title={t("Payment", { count: 1 })} />
            <SettingsContent>
                <SettingsList />
                <Box m="auto">
                    <Typography variant="h6" textAlign="center">{t(currentUser.hasPremium ? "AlreadyHavePro" : "DoNotHavePro")}</Typography>
                    <Stack direction="column" spacing={2}>
                        <Button
                            color="secondary"
                            fullWidth
                            startIcon={<IconCommon name="OpenInNew" />}
                            onClick={() => openLink(setLocation, LINKS.Pro)}
                            variant={!currentUser.hasPremium ? "outlined" : "contained"}
                            sx={{ marginTop: 2, marginBottom: 2 }}
                        >{t("ViewBenefits")}</Button>
                        {!currentUser.hasPremium && <Button
                            fullWidth
                            onClick={() => { startCheckout(PaymentType.PremiumYearly); }}
                            variant="contained"
                        >
                            <Box display="flex" justifyContent="center" alignItems="center" width="100%">
                                ${(prices?.yearly ?? 0) / 100}/{t("Year")}
                                <Box component="span" fontStyle="italic" color="green" pl={1}>
                                    {t("BestDeal")}
                                </Box>
                            </Box>
                        </Button>}
                        {!currentUser.hasPremium && <Button
                            fullWidth
                            onClick={() => { startCheckout(PaymentType.PremiumMonthly); }}
                            variant="outlined"
                        >${(prices?.monthly ?? 0) / 100}/{t("Month")}</Button>}
                        {currentUser.hasPremium && <Button
                            fullWidth
                            onClick={redirectToCustomerPortal}
                            variant="outlined"
                        >Change Plan</Button>}
                        <Button
                            fullWidth
                            onClick={() => { startCheckout(PaymentType.Donation); }}
                            variant="outlined"
                        >{t("DonationButton")}</Button>
                        {currentUser.hasPremium && <Button
                            fullWidth
                            onClick={redirectToCustomerPortal}
                            startIcon={<IconCommon name="Cancel" />}
                            variant="outlined"
                            sx={{ color: palette.error.main, borderColor: palette.error.main, marginTop: "48px!important" }}
                        >Cancel Premium</Button>}
                    </Stack>
                    
                    {/* Credits Donation Settings */}
                    {currentUser.hasPremium && (
                        <Paper 
                            elevation={0}
                            sx={{ 
                                mt: 4, 
                                p: 3, 
                                border: `1px solid ${palette.divider}`,
                                borderRadius: 2,
                                backgroundColor: palette.background.paper,
                            }}
                        >
                            <Stack spacing={3}>
                                <Box>
                                    <Typography variant="h6" gutterBottom>
                                        {t("CreditsRolloverDonation", "Credits Rollover Donation")}
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary" paragraph>
                                        {t("DonateUnusedCreditsDescription", "Donate a percentage of your unused monthly credits to support swarms that create and improve the most desired routines. This helps improve the platform for everyone while ensuring only monthly free credits (not purchased credits) are donated.")}
                                    </Typography>
                                </Box>
                                
                                <Divider />
                                
                                <FormControlLabel
                                    control={
                                        <Switch
                                            checked={creditConfig.donation.enabled}
                                            onChange={handleDonationToggle}
                                            disabled={isLoadingSettings || isSavingSettings}
                                            color="primary"
                                        />
                                    }
                                    label={
                                        <Box>
                                            <Typography variant="body1">
                                                {t("EnableCreditsDonation", "Enable Credits Donation")}
                                            </Typography>
                                            <Typography variant="caption" color="text.secondary">
                                                {t("DonateToImprovePlatform", "Help support swarms that improve the platform for everyone")}
                                            </Typography>
                                        </Box>
                                    }
                                />
                                
                                {creditConfig.donation.enabled && (
                                    <>
                                        <Box>
                                            <Box display="flex" alignItems="center" justifyContent="space-between" mb={1}>
                                                <Typography variant="body2">
                                                    {t("DonationPercentage", "Donation Percentage")}
                                                </Typography>
                                                <Typography variant="h6" color="primary">
                                                    {creditConfig.donation.percentage}%
                                                </Typography>
                                            </Box>
                                            <Slider
                                                value={creditConfig.donation.percentage}
                                                onChange={handleDonationPercentageChange}
                                                disabled={isLoadingSettings || isSavingSettings}
                                                min={0}
                                                max={100}
                                                step={5}
                                                marks={[
                                                    { value: 0, label: "0%" },
                                                    { value: 25, label: "25%" },
                                                    { value: 50, label: "50%" },
                                                    { value: 75, label: "75%" },
                                                    { value: 100, label: "100%" },
                                                ]}
                                                valueLabelDisplay="auto"
                                                sx={{ mt: 2, mb: 1 }}
                                            />
                                        </Box>
                                        
                                        <Paper 
                                            elevation={0} 
                                            sx={{ 
                                                p: 2, 
                                                backgroundColor: palette.mode === "dark" ? "rgba(255,255,255,0.05)" : "rgba(0,0,0,0.03)",
                                                borderRadius: 1,
                                            }}
                                        >
                                            <Stack spacing={1}>
                                                <Box display="flex" justifyContent="space-between">
                                                    <Typography variant="body2" color="text.secondary">
                                                        {t("MonthlyFreeCredits", "Monthly Free Credits")}:
                                                    </Typography>
                                                    <Typography variant="body2">
                                                        {getCreditsPerMonth().toLocaleString()}
                                                    </Typography>
                                                </Box>
                                                <Box display="flex" justifyContent="space-between">
                                                    <Typography variant="body2" color="text.secondary">
                                                        {t("EstimatedMonthlyDonation", "Estimated Monthly Donation")}:
                                                    </Typography>
                                                    <Typography variant="body2" fontWeight="bold" color="primary">
                                                        {calculateDonationAmount().toLocaleString()} {t("credits", "credits")}
                                                    </Typography>
                                                </Box>
                                            </Stack>
                                        </Paper>
                                        
                                        <Box 
                                            display="flex" 
                                            alignItems="flex-start" 
                                            gap={1}
                                            p={2}
                                            bgcolor={palette.mode === "dark" ? "rgba(46, 125, 50, 0.1)" : "rgba(76, 175, 80, 0.1)"}
                                            borderRadius={1}
                                        >
                                            <IconCommon name="Info" sx={{ fontSize: 20, color: palette.success.main, mt: 0.25 }} />
                                            <Typography variant="caption" color="text.secondary">
                                                {t("DonationImpactMessage", "Your donated credits will be used to incentivize swarms that create and improve the most desired routines, making the platform better for everyone. This is a great way to give back to the community while helping drive innovation.")}
                                            </Typography>
                                        </Box>
                                        
                                        <Box 
                                            display="flex" 
                                            alignItems="flex-start" 
                                            gap={1}
                                            p={2}
                                            bgcolor={palette.mode === "dark" ? "rgba(255, 193, 7, 0.1)" : "rgba(255, 152, 0, 0.1)"}
                                            borderRadius={1}
                                        >
                                            <IconCommon name="Lock" size={20} fill={palette.warning.main} style={{ marginTop: 2 }} />
                                            <Typography variant="caption" color="text.secondary">
                                                {t("DonationSafetyMessage", "Only your free monthly credits are donated—never your purchased credits. Donations are processed on the 2nd of each month.")}
                                            </Typography>
                                        </Box>
                                    </>
                                )}
                                
                                {isSavingSettings && (
                                    <Typography variant="caption" color="primary" sx={{ fontStyle: "italic" }}>
                                        {t("SavingSettings", "Saving settings")}...
                                    </Typography>
                                )}
                            </Stack>
                        </Paper>
                    )}
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}
