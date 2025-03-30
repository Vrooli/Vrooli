import { LINKS, PaymentType } from "@local/shared";
import { Box, Button, Stack, Typography, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { SessionContext } from "../../contexts/session.js";
import { useStripe } from "../../hooks/useStripe.js";
import { IconCommon } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { SettingsPaymentViewProps } from "./types.js";

export function SettingsPaymentView({
    display,
    onClose,
}: SettingsPaymentViewProps) {
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const {
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
                    {/* TODO add opt-in for donating unused monthly credits, with slider 
                    for what percentage you want to donate. For now, it should be clear that it's 
                    donated back to us. Later on when we add fundraising, we can let the user pick where it goes to
                    (not sure about the legal implications here)
                    
                    TODO NOTE: Make sure that this doesn't accidentally donate additional credits that someone bought. I think this can be solved by adding the max rollover number to be the monthly credits increment. So the calculation would be if (credits > DONATION_THRESHOLD) ? (MONTHLY_CREDITS_INCREMENT * DONATION_PERCENTAGE) : 0, where the donation threshold is defaulted to 1 month (but maybe we can have another slider for this*/}
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}
