import { LINKS, PaymentType } from "@local/shared";
import { Box, Button, Stack, Typography, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar/SettingsTopBar.js";
import { useStripe } from "../../hooks/useStripe.js";
import { CancelIcon, OpenInNewIcon } from "../../icons/common.js";
import { ScrollBox } from "../../styles.js";
import { SettingsPaymentViewProps } from "./types.js";

export function SettingsPaymentView({
    display,
    onClose,
}: SettingsPaymentViewProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const {
        currentUser,
        prices,
        startCheckout,
        redirectToCustomerPortal,
    } = useStripe();

    return (
        <ScrollBox>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Payment", { count: 1 })}
            />
            <SettingsContent>
                <SettingsList />
                <Box m="auto">
                    <Typography variant="h6" textAlign="center">{t(currentUser.hasPremium ? "AlreadyHavePro" : "DoNotHavePro")}</Typography>
                    <Stack direction="column" spacing={2}>
                        <Button
                            color="secondary"
                            fullWidth
                            startIcon={<OpenInNewIcon />}
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
                            startIcon={<CancelIcon />}
                            variant="outlined"
                            sx={{ color: palette.error.main, borderColor: palette.error.main, marginTop: "48px!important" }}
                        >Cancel Premium</Button>}
                    </Stack>
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}
