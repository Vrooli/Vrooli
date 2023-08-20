import { LINKS, PaymentType } from "@local/shared";
import { Box, Button, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useStripe } from "hooks/useStripe";
import { CompleteIcon } from "icons";
import { useTranslation } from "react-i18next";
import { stringifySearchParams, useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { PremiumViewProps } from "../types";

// Features comparison table data
function createData(feature, nonPremium, premium) {
    return { feature, nonPremium, premium };
}
const rows = [
    createData("Routines and processes", "Up to 25 private, 100 public", "Very high limits"),
    createData("*AI-related features", "GPT API key required", "✔️"),
    createData("*Human and bot collaboration", "GPT API key required", "✔️"),
    createData("*Customize and replicate public organizations", "✔️", "✔️"),
    createData("*Copy and adapt public routines", "✔️", "✔️"),
    createData("*Analytics dashboard", "Essential", "Advanced"),
    createData("Customizable user experience", "✔️", "✔️"),
    createData("Community sharing", "✔️", "✔️"),
    createData("*Data import/export", "✔️", "✔️"),
    createData("Industry-standard templates", "✔️", "✔️"),
    createData("Mobile app", "✔️", "✔️"),
    createData("*Tutorial resources", "✔️", "✔️"),
    createData("Community support", "✔️", "✔️"),
    createData("Updates and improvements", "✔️", "Early access"),
    createData("Task management", "✔️", "✔️"),
    createData("*Calendar integration", "✔️", "✔️"),
    createData("Customized notifications", "✔️", "✔️"),
    createData("Provide feedback", "✔️", "✔️"),
    createData("Ad-free experience", "❌", "✔️"),
    createData("Enhanced focus modes", "❌", "✔️"),
    createData("*Premium API access", "❌", "✔️"),
];

export const PremiumView = ({
    isOpen,
    onClose,
}: PremiumViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const {
        currentUser,
        prices,
        startCheckout,
        redirectToCustomerPortal,
    } = useStripe();

    // TODO convert MaxObjects to list of limit increases 
    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Premium")}
            />
            <Stack direction="column" spacing={4} mt={2} mb={2} justifyContent="center" alignItems="center">
                {/* Introduction to premium */}
                <Typography variant="h6" sx={{ textAlign: "center", margin: 2 }}>{t("PremiumIntro1")}</Typography>
                <Typography variant="h6" sx={{ textAlign: "center", margin: 2 }}>{t("PremiumIntro2")}</Typography>
                {/* Main features as table */}
                <Box>
                    <Typography variant="body2" mb={1} sx={{ textAlign: "left", color: palette.error.main }}><span style={{ fontSize: "x-large" }}>*</span> {t("ComingSoon")}</Typography>
                    <TableContainer component={Paper} sx={{ maxWidth: 800 }}>
                        <Table aria-label="features table">
                            <TableHead sx={{ background: palette.primary.light }}>
                                <TableRow>
                                    <TableCell sx={{ color: palette.primary.contrastText }}>{t("Feature")}</TableCell>
                                    <TableCell align="center" sx={{ color: palette.primary.contrastText }}>{t("NotPremium")}</TableCell>
                                    <TableCell align="center" sx={{ color: palette.primary.contrastText }}>{t("Premium")}</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {rows.map((row) => (
                                    <TableRow key={row.feature}>
                                        <TableCell component="th" scope="row">
                                            {row.feature.startsWith("*") ? (
                                                <>
                                                    <span style={{ color: palette.error.main, fontSize: "x-large" }}>*</span>
                                                    {row.feature.slice(1)}
                                                </>
                                            ) : (
                                                row.feature
                                            )}
                                        </TableCell>
                                        <TableCell align="center">
                                            {row.nonPremium === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : row.nonPremium}
                                        </TableCell>
                                        <TableCell align="center">
                                            {row.premium === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : row.premium}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Box>
                <Typography variant="body1" sx={{ textAlign: "center" }}>
                    Upgrade to Vrooli Premium for an ad-free experience, AI-powered integrations, advanced analytics tools, and more. Maximize your potential – go Premium now!
                </Typography>
                {/* Link to open popup that displays all limit increases */}
                {/* TODO */}
                {/* Button row for different subscriptions, with donation option at bottom. Should have dialog like on matthalloran.info */}
                {currentUser.id && <Stack direction="column" spacing={2} m={2} sx={{ width: "100%", maxWidth: "700px" }}>
                    <Button
                        disabled={currentUser.hasPremium}
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
                    </Button>
                    <Button
                        disabled={currentUser.hasPremium}
                        fullWidth
                        onClick={() => { startCheckout(PaymentType.PremiumMonthly); }}
                        variant="outlined"
                    >${(prices?.monthly ?? 0) / 100}/{t("Month")}</Button>
                    {currentUser.hasPremium && (
                        // TODO need way to change from monthly to yearly and vice versa
                        <Typography variant="body1" sx={{ textAlign: "center" }}>
                            {t("AlreadyHavePremium")}
                        </Typography>
                    )}
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
                </Stack>}
                {/* If not logged in, button to log in first */}
                {!currentUser.id && <Button
                    fullWidth
                    onClick={() => { setLocation(`${LINKS.Start}${stringifySearchParams({ redirect: LINKS.Premium })}`); }}
                    variant="contained"
                >{t("LogInToUpgrade")}</Button>}
            </Stack>
        </>
    );
};
