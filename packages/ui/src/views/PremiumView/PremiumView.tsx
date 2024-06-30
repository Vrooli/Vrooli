import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, LINKS, PaymentType } from "@local/shared";
import { Box, Button, Link, List, ListItem, ListItemText, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { RandomBlobs } from "components/RandomBlobs/RandomBlobs";
import { Testimonials } from "components/Testimonials/Testimonials";
import { IntegerInputBase } from "components/inputs/IntegerInput/IntegerInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useStripe } from "hooks/useStripe";
import { useWindowSize } from "hooks/useWindowSize";
import { CompleteIcon, OpenInNewIcon, ShoppingCartIcon } from "icons";
import { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { getDeviceInfo } from "utils/display/device";
import { PremiumViewProps } from "../types";

const purpleRadial = "radial-gradient(circle, rgb(16 6 46) 15%, rgb(11 1 36) 55%, rgb(8 3 20) 85%)";

// Features comparison table data
const rows = [
    ["Create public and private bots", "✔️", "✔️"],
    ["Chat with one or more bots", "Credits required", "✔️"],
    ["Build routines to complete complex tasks", "Up to 25 private, 100 public", "Very high limits"],
    ["Store reminders, notes, events, and more", "Limits vary", "✔️"],
    ["Share and remix with the community", "✔️", "✔️"],
    ["Auto-fill forms using AI", "Credits required", "✔️"],
    ["*Analytics dashboard", "Essential", "Advanced"],
    ["*Data import/export", "✔️", "✔️"],
    ["*Mobile app", "✔️", "✔️"],
    ["*Calendar integration", "❌", "✔️"],
    ["*Premium API access", "❌", "✔️"],
    ["Ad-free experience", "❌", "✔️"],
    ["Early access to new features", "❌", "✔️"],
].map(([feature, free, pro]) => ({ feature, free, pro }));

export function PremiumView({
    display,
    onClose,
}: PremiumViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [isCustomCreditAmountOpen, setIsCustomCreditAmountOpen] = useState(false);
    const [customCreditAmount, setCustomCreditAmount] = useState(5);

    const [isCustomDonationAmountOpen, setIsCustomDonationAmountOpen] = useState(false);
    const [customDonationAmount, setCustomDonationAmount] = useState(20);

    const {
        checkFailedCredits,
        checkFailedSubscription,
        currentUser,
        prices,
        redirectToCustomerPortal,
        startCheckout,
    } = useStripe();

    function scrollToElement(elementId: string) {
        const element = document.getElementById(elementId);
        if (element) {
            // Scroll so element is in middle of screen
            element.scrollIntoView({ behavior: "smooth", block: "center" });

            // Highlight effect with CSS
            const originalBackground = element.style.backgroundColor;
            const originalBorderRadius = element.style.borderRadius;
            element.style.transition = "background-color, 0.5s ease-in-out, border-radius, 0.5s ease-in-out";
            element.style.backgroundColor = "#ffff9944";
            element.style.borderRadius = "8px";

            // Remove highlight after some time
            setTimeout(() => {
                element.style.backgroundColor = originalBackground;
                element.style.borderRadius = originalBorderRadius;
            }, 1000);
        }
    }

    return (
        <Box sx={{
            background: purpleRadial,
            backgroundAttachment: "fixed",
            color: "white",
            padding: 2,
            marginBottom: (isMobile && getDeviceInfo().isStandalone) ? pagePaddingBottom : 0,
        }}>
            <RandomBlobs numberOfBlobs={isMobile ? 3 : 5} />
            <TopBar
                display={display}
                hideTitleOnDesktop
                onClose={onClose}
                title={t("Pricing")}
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
                {/* Introduction*/}
                <Typography variant="h4" sx={{ textAlign: "center", padding: 2, maxWidth: "min(700px, 100%)" }}>{t("ProIntro1")}</Typography>
                <Typography variant="h5" sx={{ textAlign: "center", paddingTop: 2, maxWidth: "min(700px, 100%)" }}>{t("ProIntro2")}</Typography>
                <List>
                    <ListItem>
                        <Button
                            variant="text"
                            endIcon={<OpenInNewIcon />}
                            onClick={() => scrollToElement("get-subscription")}
                            sx={{ textTransform: "none" }}
                        >
                            <ListItemText primary={"— " + t("ProSupport1", { monthlyCredits: `$${(Number(API_CREDITS_PREMIUM / BigInt(1_000_000)) / 100).toFixed(2)}` })} sx={{ color: "white" }} />
                        </Button>
                    </ListItem>
                    <ListItem>
                        <Button
                            variant="text"
                            endIcon={<OpenInNewIcon />}
                            onClick={() => scrollToElement("buy-credits")}
                            sx={{ textTransform: "none" }}
                        >
                            <ListItemText primary={"— " + t("ProSupport2")} sx={{ color: "white" }} />
                        </Button>
                    </ListItem>
                    <ListItem>
                        <Button
                            variant="text"
                            endIcon={<OpenInNewIcon />}
                            onClick={() => scrollToElement("donate")}
                            sx={{ textTransform: "none" }}
                        >
                            <ListItemText primary={"— " + t("ProSupport3")} sx={{ color: "white" }} />
                        </Button>
                    </ListItem>
                    <ListItem>
                        <Button
                            variant="text"
                            endIcon={<OpenInNewIcon />}
                            onClick={() => openLink(setLocation, LINKS.Create)}
                            sx={{ textTransform: "none" }}
                        >
                            <ListItemText primary={"— " + t("ProSupport4")} sx={{ color: "white" }} />
                        </Button>
                    </ListItem>
                </List>
                {/* Main features as table */}
                <Box id="pro-features" sx={{ width: "100%", margin: "auto", boxShadow: 3 }}>
                    <Typography variant="h5" sx={{ textAlign: "center" }}>{t("Feature", { count: 2 })}</Typography>
                    <Typography variant="body2" mb={1} sx={{ textAlign: "left", color: palette.error.main }}>
                        <span style={{ fontSize: "x-large", verticalAlign: "middle" }}>*</span> {t("ComingSoon")}
                    </Typography>
                    <TableContainer component={Paper} sx={{ borderRadius: 4 }}>
                        <Table aria-label="features table">
                            <TableHead sx={{ background: palette.primary.light }}>
                                <TableRow>
                                    {/* Bold typography for feature headings */}
                                    <TableCell sx={{ fontWeight: "bold", color: palette.primary.contrastText }}>{t("Feature", { count: 1 })}</TableCell>
                                    <TableCell align="center" sx={{ fontWeight: "bold", color: palette.primary.contrastText }}>{t("Free")}</TableCell>
                                    <TableCell align="center" sx={{ fontWeight: "bold", color: palette.primary.contrastText, background: palette.mode === "light" ? "#2b6fb6" : "#7e8db8" }}>{t("Pro")}</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {rows.map(({ feature, free, pro }) => (
                                    <TableRow
                                        key={feature}
                                        onMouseOut={(e) => e.currentTarget.style.backgroundColor = ""}
                                    >
                                        <TableCell component="th" scope="row">
                                            {feature.startsWith("*") ? (
                                                <>
                                                    <span style={{ color: palette.error.main, fontSize: "x-large" }}>*</span>
                                                    {feature.slice(1)}
                                                </>
                                            ) : (
                                                feature
                                            )}
                                        </TableCell>
                                        <TableCell align="center">
                                            {free === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : free}
                                        </TableCell>
                                        <TableCell align="center" sx={{ background: palette.mode === "light" ? "#c8ffdd" : "#555f6a" }}>
                                            {pro === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : pro}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Box>
                <Box id="get-subscription" mb={4} sx={{ maxWidth: "800px", width: "100%" }}>
                    <Typography variant="h6" pt={3} pb={2} sx={{ textAlign: "center" }}>
                        Upgrade to Vrooli Pro for AI-powered integrations, advanced analytics tools, and more. Maximize your potential — become a Vrooli Pro user today!
                    </Typography>
                    <Box sx={{
                        background: palette.background.paper + (palette.mode === "light" ? "44" : "c4"),
                        color: "white",
                        borderRadius: 2,
                        padding: 1,
                    }}>
                        <Stack direction="column" spacing={2} m={2} mb={1} sx={{ maxWidth: "100%" }}>
                            {!currentUser.hasPremium && <>
                                <Button
                                    fullWidth
                                    onClick={() => { startCheckout(PaymentType.PremiumYearly); }}
                                    variant="contained"
                                >
                                    <Box display="flex" justifyContent="center" alignItems="center" width="100%">
                                        ${(prices?.yearly ?? 0) / 100}/{t("Year")}
                                        <Box component="span" fontStyle="italic" color="orange" pl={1}>
                                            {t("BestDeal")}
                                        </Box>
                                    </Box>
                                </Button>
                                <Button
                                    fullWidth
                                    onClick={() => { startCheckout(PaymentType.PremiumMonthly); }}
                                    variant="outlined"
                                >${(prices?.monthly ?? 0) / 100}/{t("Month")}</Button>
                                <Button
                                    fullWidth
                                    onClick={() => { checkFailedSubscription(); }}
                                    variant="text"
                                    sx={{
                                        textDecoration: "underline",
                                        textTransform: "none",
                                        color: "lightgray",
                                    }}
                                >Didn't receive Pro? Check status</Button>
                            </>}
                            {currentUser.hasPremium && <>
                                <Typography variant="body1" sx={{ textAlign: "center" }}>
                                    {t("AlreadyHavePro")}
                                </Typography>
                                <Button
                                    fullWidth
                                    onClick={redirectToCustomerPortal}
                                    variant="outlined"
                                >Change Plan</Button>
                            </>}
                        </Stack>
                    </Box>
                </Box>
                <Box id="buy-credits" mb={4} sx={{ maxWidth: "800px", width: "100%" }}>
                    <Typography variant="h6" pt={3} pb={2} sx={{ textAlign: "center" }}>
                        Buy credits to perform AI-related tasks, such as running routines, messaging bots, and auto-filling forms.
                    </Typography>
                    <Box sx={{
                        background: (theme) => `${theme.palette.background.paper}${theme.palette.mode === "light" ? "44" : "c4"}`,
                        color: "white",
                        borderRadius: 2,
                        padding: 1,
                        display: "flex",
                        flexDirection: "column",
                        gap: 1,
                    }}>
                        {session?.isLoggedIn && (
                            <Typography variant="body1" sx={{ textAlign: "center", paddingTop: 1, paddingBottom: 1 }}>
                                {`Current credits: $${(Number(BigInt(currentUser.credits ?? "0") / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}`}
                            </Typography>
                        )}
                        {!isCustomCreditAmountOpen && <Box sx={{
                            display: "flex",
                            flexDirection: { xs: "column", sm: "row" },
                            gap: 1,
                        }}>
                            <Button
                                fullWidth
                                onClick={() => { startCheckout(PaymentType.Credits, 500); }}
                                variant="outlined"
                            >$5</Button>
                            <Button
                                fullWidth
                                onClick={() => { startCheckout(PaymentType.Credits, 2000); }}
                                variant="outlined"
                            >$20</Button>
                            <Button
                                fullWidth
                                onClick={() => { setIsCustomCreditAmountOpen(true); }}
                                variant="outlined"
                            >Custom</Button>
                        </Box>}
                        {isCustomCreditAmountOpen &&
                            <Box sx={{
                                maxWidth: "500px",
                                width: "100%",
                                margin: "auto",
                                marginTop: 2,
                                display: "flex",
                                flexDirection: "column",
                                gap: 1,
                            }}>
                                <IntegerInputBase
                                    fullWidth
                                    label="Custom Amount ($)"
                                    max={1000}
                                    min={5}
                                    name="customCreditAmount"
                                    value={customCreditAmount}
                                    onChange={(newValue) => setCustomCreditAmount(newValue)}
                                />
                                <Button
                                    fullWidth
                                    onClick={() => { startCheckout(PaymentType.Credits, customCreditAmount * 100); }}
                                    variant="contained"
                                    disabled={!customCreditAmount}
                                    endIcon={<ShoppingCartIcon />}
                                >Buy Credits</Button>
                            </Box>
                        }
                        <Box sx={{
                            display: "flex",
                            flexDirection: { xs: "column", sm: "row" },
                            gap: 1,
                            justifyContent: "center",

                        }}>
                            <Button
                                fullWidth
                                onClick={() => { scrollToElement("credits-explainer"); }}
                                variant="text"
                                sx={{
                                    textDecoration: "underline",
                                    textTransform: "none",
                                    color: "lightgray",
                                }}
                            >How do credits work?</Button>
                            <Button
                                fullWidth
                                onClick={() => { checkFailedCredits(); }}
                                variant="text"
                                sx={{
                                    textDecoration: "underline",
                                    textTransform: "none",
                                    color: "lightgray",
                                }}
                            >Didn't receive credits? Check status</Button>
                        </Box>
                    </Box>
                </Box>
                <Box id="donate" mb={4} sx={{ maxWidth: "800px", width: "100%" }}>
                    <Typography variant="h6" pt={3} pb={2} sx={{ textAlign: "center" }}>
                        Support Vrooli directly by making a donation. Your contribution helps us maintain and improve our services💙
                    </Typography>
                    <Box sx={{
                        background: (theme) => `${theme.palette.background.paper}${theme.palette.mode === "light" ? "44" : "c4"}`,
                        color: "white",
                        borderRadius: 2,
                        padding: 1,
                        display: "flex",
                        flexDirection: "column",
                        gap: 1,
                    }}>
                        {!isCustomDonationAmountOpen && <Box sx={{
                            display: "flex",
                            flexDirection: { xs: "column", sm: "row" },
                            gap: 1,
                        }}>
                            <Button
                                fullWidth
                                onClick={() => { startCheckout(PaymentType.Donation, 500); }}
                                variant="outlined"
                            >$5</Button>
                            <Button
                                fullWidth
                                onClick={() => { startCheckout(PaymentType.Donation, 2000); }}
                                variant="outlined"
                            >$20</Button>
                            <Button
                                fullWidth
                                onClick={() => { setIsCustomDonationAmountOpen(true); }}
                                variant="outlined"
                            >Custom</Button>
                        </Box>}
                        {isCustomDonationAmountOpen &&
                            <Box sx={{
                                maxWidth: "500px",
                                width: "100%",
                                margin: "auto",
                                marginTop: 2,
                                display: "flex",
                                flexDirection: "column",
                                gap: 1,
                            }}>
                                <IntegerInputBase
                                    fullWidth
                                    label="Custom Amount ($)"
                                    max={1000}
                                    min={1}
                                    name="customDonationAmount"
                                    value={customDonationAmount}
                                    onChange={(newValue) => setCustomDonationAmount(newValue)}
                                />
                                <Button
                                    fullWidth
                                    onClick={() => { startCheckout(PaymentType.Donation, customDonationAmount * 100); }}
                                    variant="contained"
                                    disabled={!customDonationAmount}
                                    endIcon={<ShoppingCartIcon />}
                                >Donate</Button>
                            </Box>
                        }
                    </Box>
                </Box>
                <Typography variant="h5">Made-Up Testimonials</Typography>
                <Testimonials />
                {/* FAQ Section at the bottom */}
                <Box mt={4} px={4} py={3} borderRadius={4} boxShadow={3} sx={{
                    background: palette.background.paper + (palette.mode === "light" ? "44" : "c4"),
                    color: "white",
                }}>
                    <Typography variant="h5" style={{ textAlign: "center", marginBottom: "1.5rem", color: "primary.main" }}>Frequently Asked Questions</Typography>

                    <Typography variant="h6" color="lightgray" id="credits-explainer">1. How is credit usage calculated?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>We base the cost of performing AI-related tasks by the model used, the size of the text passed into each request, and the size of the text returned. We try our best to charge almost exactly what it costs us to run the AI models, and we're always looking for ways to make it cheaper for you.</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Over time, as models become cheaper and more efficient routines are created, the cost of performing tasks should decrase.</Typography>

                    <Typography variant="h6" color="lightgray">2. Can I buy more credits?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Yes! You can buy more credits at any time. If you have a <i>Pro</i> subscription, you can also wait until the first of the month to receive your monthly credits.</Typography>

                    <Typography variant="h6" color="lightgray">3. What happens to leftover credits at the beginning of the month?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>If you still have credits left over from the previous month, they will be added to your new monthly credits — up to a maximum of 6 months' worth of credits. If you have more than 6 months' worth of credits, your credit balance won't change.</Typography>

                    <Typography variant="h6" color="lightgray">4. What happens if my credit balance goes negative?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>If your credit balance goes negative, you won't be able to perform AI-related tasks until your balance is positive again. This is our mistake, so you won't be charged any extra fees.</Typography>

                    <Typography variant="h6" color="lightgray">5. What are the differences between the <i>Free</i> and <i>Pro</i> plans?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>The <i>Free</i>  plan offers basic functionalities like taking notes, creating reminders, viewing routines, participating in chat groups with other users, and more. The <i>Pro</i> plan enhances your experience with advanced features like AI-powered integrations, enhanced analytics, and more.</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Note that you can still use AI-related features in the <i>Free</i> plan, but you'll need to buy credits.</Typography>

                    <Typography variant="h6" color="lightgray">6. Are there any long-term commitments or contracts?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>No, there are no long-term commitments or contracts. You can choose to subscribe on a monthly or yearly basis and can cancel anytime.</Typography>

                    <Typography variant="h6" color="lightgray">7. What happens if I decide to cancel my subscription?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>If you decide to cancel, you will be downgraded to the <i>Free</i>  plan at the end of your billing cycle and won't be charged thereafter. You'll still have access to all your data and can upgrade again anytime.</Typography>

                    <Typography variant="h6" color="lightgray">8. How secure is my payment information?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Payments are handled securely using <Link href="https://stripe.com/" target="_blank" rel="noopener noreferrer">Stripe</Link>, which is a leading third-party payment processor. We don't store any payment information on our servers.</Typography>

                    <Typography variant="h6" color="lightgray">9. Do you offer refunds?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>We do not offer refunds for credits or subscriptions at this time. If you cancel your subscription, you will still have access to the pro features until the end of your billing cycle.</Typography>

                    <Typography variant="h6" color="lightgray">10. Can I change from a monthly to a yearly subscription, or vice versa?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Absolutely! You can switch between plans from the customer portal or contact our support for assistance.</Typography>

                    <Typography variant="h6" color="lightgray">11. What does "Early access to updates and improvements" mean?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>Pro users get access to new features, improvements, and updates before they are rolled out to all users. It's our way of saying thank you for supporting us! (and also testing new features before wide release, but that doesn't sound as nice)</Typography>

                    <Typography variant="h6" color="lightgray">12. How does the mobile app differ from the web version?</Typography>
                    <Typography variant="body1" style={{ marginBottom: "1rem" }}>The mobile app is the same or almost the same as the web version. It's a Progressive Web App (PWA), which means it's a website that can be installed on your phone and used like a native app. It's a great way to stay productive on the go!</Typography>

                    <Typography variant="h6" color="lightgray">13. I have more questions. How can I reach out?</Typography>
                    <Typography variant="body1">We're here to help! You can contact our support team at <Link href="mailto:official@vrooli.com" target="_blank" rel="noopener noreferrer">official@vrooli.com</Link> or message us on <Link href="https://x.com/vrooliofficial" target="_blank" rel="noopener noreferrer">X</Link>.</Typography>
                </Box>
            </Box >
        </Box >
    );
}
