import { LINKS, PaymentType } from "@local/shared";
import { Box, Button, Link, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, keyframes, useTheme } from "@mui/material";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import { Testimonials } from "components/Testimonials/Testimonials";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useStripe } from "hooks/useStripe";
import { CompleteIcon, LogInIcon } from "icons";
import { useTranslation } from "react-i18next";
import { stringifySearchParams, useLocation } from "route";
import { PremiumViewProps } from "../types";
import { useWindowSize } from "hooks/useWindowSize";
import { getDisplay } from "utils/display/listTools";
import { pagePaddingBottom } from "styles";
import { getDeviceInfo } from "utils/display/device";

const purpleRadial = "radial-gradient(circle, rgb(16 6 46) 15%, rgb(11 1 36) 55%, rgb(8 3 20) 85%)";

const blob1Animation = keyframes`
    0% {
        transform: translateY(0) scale(0.55);
        filter: hue-rotate(0deg) blur(140px);
    }
    33% {
        transform: translateY(-150px) scale(0.85) rotate(-100deg);
        filter: hue-rotate(40deg) blur(140px);
    }
    66% {
        transform: translate(45px, 10px) scale(0.65) rotate(-180deg);
        filter: hue-rotate(80deg) blur(140px);
    }
    100% {
        transform: translate(0px, 0px) scale(0.55) rotate(0deg);
        filter: hue-rotate(0deg) blur(140px);
    }
`;

const blob2Animation = keyframes`
    0% {
        transform: translateX(0) scale(0.95);
        filter: hue-rotate(0deg) blur(60px);
    }
    50% {
        transform: translateX(140px) scale(1.1);
        filter: hue-rotate(-60deg) blur(60px);
    }
    100% {
        transform: translateX(0) scale(0.95);
        filter: hue-rotate(0deg) blur(60px);
    }
`;

const blob3Animation = keyframes`
    0% {
        transform: translateY(0) scale(0.7);
        filter: hue-rotate(0deg) blur(100px);
    }
    33% {
        transform: translateY(-110px) scale(1) rotate(-150deg);
        filter: hue-rotate(40deg) blur(100px);
    }
    66% {
        transform: translate(30px, -50px) scale(0.8) rotate(-250deg);
        filter: hue-rotate(80deg) blur(100px);
    }
    100% {
        transform: translate(0px, 0px) scale(0.7) rotate(0deg);
        filter: hue-rotate(0deg) blur(100px);
    }
`;

// Features comparison table data
function createData(feature, nonPremium, premium) {
    return { feature, nonPremium, premium };
}
const rows = [
    createData("Routines and processes", "Up to 25 private, 100 public", "Very high limits"),
    createData("*AI-related features", "Credits required", "✔️"),
    createData("*Human and bot collaboration", "Credits required", "✔️"),
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
    display,
    onClose,
}: PremiumViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const { currentUser, prices, startCheckout, redirectToCustomerPortal } = useStripe();

    // TODO convert MaxObjects to list of limit increases 
    return (
        <Box sx={{
            background: purpleRadial,
            backgroundAttachment: "fixed",
            color: "white",
            padding: 2,
            marginBottom: (isMobile && getDeviceInfo().isStandalone) ? pagePaddingBottom : 0,
        }}>
            {/* Blob 1 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                top: -200,
                left: -50,
                width: "100%",
                height: "100%",
                opacity: 0.5,
                filter: "hue-rotate(150deg)",
                transition: "opacity 1s ease-in-out",
                zIndex: 0,
            }}>
                <Box
                    component="img"
                    src={Blob1}
                    alt="Blob 1"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {/* Blob 2 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                bottom: -175,
                right: -300,
                width: "150%",
                height: "150%",
                opacity: 0.5,
                filter: "hue-rotate(300deg)",
                transition: "opacity 1s ease-in-out",
                zIndex: 0,
            }}>
                <Box
                    component="img"
                    src={Blob2}
                    alt="Blob 2"
                    sx={{
                        width: "150%",
                        height: "150%",
                        animation: `${blob2Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {/* Blob 3 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                bottom: -175,
                left: -360,
                width: "100%",
                height: "100%",
                opacity: 0.5,
                filter: "hue-rotate(100deg)",
                transition: "opacity 1s ease-in-out",
                zIndex: 0,
            }}>
                <Box
                    component="img"
                    src={Blob1}
                    alt="Blob 1"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            <TopBar
                display={display}
                hideTitleOnDesktop
                onClose={onClose}
                title={t("Premium")}
            />
            <Box
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "center",
                    alignItems: "center",
                    gap: 2,
                    paddingTop: 2,
                    marginBottom: 2,
                    maxWidth: "min(1200px, 100%)",
                    margin: "auto",
                    position: "relative",
                }}>
                {/* Introduction to premium */}
                <Typography variant="h4" sx={{ textAlign: "center", padding: 2, maxWidth: "min(700px, 100%)" }}>{t("PremiumIntro")}</Typography>
                {/* Main features as table */}
                <Box sx={{ width: "100%", margin: "auto", marginBottom: 2, boxShadow: 3 }}>
                    <Typography variant="body2" mb={1} sx={{ textAlign: "left", color: palette.error.main }}>
                        <span style={{ fontSize: "x-large" }}>*</span> {t("ComingSoon")}
                    </Typography>
                    <TableContainer component={Paper} sx={{ borderRadius: 4 }}>
                        <Table aria-label="features table">
                            <TableHead sx={{ background: palette.primary.light }}>
                                <TableRow>
                                    {/* Bold typography for feature headings */}
                                    <TableCell sx={{ fontWeight: "bold", color: palette.primary.contrastText }}>{t("Feature")}</TableCell>
                                    <TableCell align="center" sx={{ fontWeight: "bold", color: palette.primary.contrastText }}>{t("NotPremium")}</TableCell>
                                    <TableCell align="center" sx={{ fontWeight: "bold", color: palette.primary.contrastText, background: palette.mode === "light" ? "#2b6fb6" : "#7e8db8" }}>{t("Premium")}</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {rows.map((row) => (
                                    <TableRow
                                        key={row.feature}
                                        onMouseOut={(e) => e.currentTarget.style.backgroundColor = ""}
                                    >
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
                                        <TableCell align="center" sx={{ background: palette.mode === "light" ? "#c8ffdd" : "#555f6a" }}>
                                            {row.premium === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : row.premium}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Box>
                <Box sx={{ maxWidth: "800px" }}>
                    <Typography variant="body1" mb={1} sx={{ textAlign: "center" }}>
                        Upgrade to Vrooli Premium for an ad-free experience, AI-powered integrations, advanced analytics tools, and more. Maximize your potential – go Premium now!
                    </Typography>
                    {/* Link to open popup that displays all limit increases */}
                    {/* TODO */}
                    <Box sx={{
                        background: currentUser.id ? palette.background.paper : "transparent", // When not logged in, there's only one button - so we don't need the background 
                        color: palette.background.textPrimary,
                        borderRadius: 4,
                        padding: currentUser.id ? 1 : "8px 0",
                        boxShadow: currentUser.id ? 3 : 0,
                        marginBottom: 4,
                    }}>
                        {currentUser.id && <Stack direction="column" spacing={2} m={2} sx={{ width: "100%", maxWidth: "700px" }}>
                            {!currentUser.hasPremium && <>
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
                            </>}
                            {currentUser.hasPremium && <>
                                <Typography variant="body1" sx={{ textAlign: "center" }}>
                                    {t("AlreadyHavePremium")}
                                </Typography>
                                <Button
                                    fullWidth
                                    onClick={redirectToCustomerPortal}
                                    variant="outlined"
                                >Change Plan</Button>
                            </>}
                            <Button
                                fullWidth
                                onClick={() => { startCheckout(PaymentType.Donation); }}
                                variant="outlined"
                            >{t("DonationButton")}</Button>
                        </Stack>}
                        {/* If not logged in, button to log in first */}
                        {!currentUser.id && <Button
                            fullWidth
                            onClick={() => { setLocation(`${LINKS.Signup}${stringifySearchParams({ redirect: LINKS.Premium })}`); }}
                            startIcon={<LogInIcon />}
                            variant="contained"
                        >{t("SignUpToUpgrade")}</Button>}
                    </Box>
                </Box>
                <Testimonials />
                {/* FAQ Section at the bottom */}
                <Box mt={4} px={4} py={3} borderRadius={4} boxShadow={3} sx={{ background: palette.background.paper, color: palette.background.textPrimary }}>
                    <Typography variant="h5" style={{ textAlign: "center", marginBottom: "1.5rem", color: "primary.main" }}>Frequently Asked Questions</Typography>

                    <Typography variant="h6" color="textSecondary">1. What is the difference between the "Standard" and "Premium" plans?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>The "Standard" plan offers basic functionalities like taking notes, creating reminders, viewing routines, participating in chat groups with other users, and more. The "Premium" plan enhances your experience with advanced features like AI-powered integrations, enhanced analytics, and more.</Typography>

                    <Typography variant="h6" color="textSecondary">2. Are there any long-term commitments or contracts?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>No, there are no long-term commitments or contracts. You can choose to subscribe on a monthly or yearly basis and can cancel anytime.</Typography>

                    <Typography variant="h6" color="textSecondary">3. What happens if I decide to cancel my "Premium" subscription?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>If you decide to cancel, you will be downgraded to the "Standard" plan at the end of your billing cycle and won't be charged thereafter. You'll still have access to all your data and can upgrade again anytime.</Typography>

                    <Typography variant="h6" color="textSecondary">4. How do the "Credits" work for AI-related features?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>With the "Standard" plan, certain AI-related features require credits. Each user gets a small number of credits each month. If you need more, you can purchase additional credits. Premium users have unlimited access.</Typography>

                    <Typography variant="h6" color="textSecondary">5. How secure is my payment information?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Your payment information is secure. We use <Link href="https://stripe.com/" target="_blank" rel="noopener noreferrer">Stripe</Link>, which is a leading payment gateway ensuring the security of your data.</Typography>

                    <Typography variant="h6" color="textSecondary">6. Do you offer refunds?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Yes, we offer a 30-day money-back guarantee. If you're not satisfied, you can contact us to request a refund within this period.</Typography>

                    <Typography variant="h6" color="textSecondary">7. Can I change from a monthly to a yearly subscription, or vice versa?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Absolutely! You can switch between plans from the customer portal or contact our support for assistance.</Typography>

                    <Typography variant="h6" color="textSecondary">8. What does "Early access to updates and improvements" mean?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Premium users get access to new features, improvements, and updates before they are rolled out to all users. It's our way of saying thank you for supporting us! (and also testing new features before wide release, but that doesn't sound as nice)</Typography>

                    <Typography variant="h6" color="textSecondary">9. How does the mobile app differ from the web version?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>The mobile app is the same or almost the same as the web version. It's a Progressive Web App (PWA), which means it's a website that can be installed on your phone and used like a native app. It's a great way to stay productive on the go!</Typography>

                    <Typography variant="h6" color="textSecondary">10. I have more questions. How can I reach out?</Typography>
                    <Typography variant="body1">We're here to help! You can contact our support team at <Link href="mailto:official@vrooli.com" target="_blank" rel="noopener noreferrer">official@vrooli.com</Link> or message us on <Link href="https://x.com/vrooliofficial" target="_blank" rel="noopener noreferrer">X/Twitter</Link>.</Typography>
                </Box>
            </Box>
        </Box>
    );
};
