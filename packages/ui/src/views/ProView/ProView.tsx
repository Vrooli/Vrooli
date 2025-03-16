import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, LINKS, PaymentType, SubscriptionPricesResponse, YEARS_1_MONTHS } from "@local/shared";
import { Box, Button, ButtonBase, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, Link, Typography, styled, useTheme } from "@mui/material";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Footer } from "../../components/navigation/Footer/Footer.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { SnackSeverity } from "../../components/snacks/BasicSnack/BasicSnack.js";
import { SessionContext } from "../../contexts.js";
import { useStripe } from "../../hooks/useStripe.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { ArrowUpRightIcon, HeartFilledIcon, HelpIcon } from "../../icons/common.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { RandomBlobs } from "../main/LandingView.js";
import { ProViewProps } from "../types.js";

// Match keys in SubscriptionPricesResponse
export enum BillingCycle {
    Monthly = "monthly",
    Yearly = "yearly",
}

export enum PricingTierType {
    Basic = "Basic",
    Pro = "Pro",
    Credits = "Credits",
}

type PricingInfo = {
    key: PricingTierType;
    title: string;
    features: string[];
    buttonText: string;
    recommended?: boolean;
}

const pricingInfo: Record<PricingTierType, PricingInfo> = {
    [PricingTierType.Basic]: {
        key: PricingTierType.Basic,
        title: "Basic",
        features: [
            "Core features",
            "High usage limits",
            "Community support",
            "Data import/export",
            "Mobile app",
            "Ad-supported experience",
        ],
        buttonText: "Start for free",
    },
    [PricingTierType.Pro]: {
        key: PricingTierType.Pro,
        title: "Pro Subscription",
        features: [
            "AI-powered routines and chats",
            "Nearly unlimited usage",
            "Data import/export",
            "Mobile app",
            "Ad-free experience",
            "API access",
            "Early access to new features",
        ],
        buttonText: "Subscribe now",
        recommended: true,
    },
    [PricingTierType.Credits]: {
        key: PricingTierType.Credits,
        title: "Pro Credits",
        features: [
            "AI-powered routines and chats",
            "High usage limits",
            "Community support",
            "Data import/export",
            "Mobile app",
            "Ad-supported experience",
            "API access",
        ],
        buttonText: "Buy credits",
    },
};

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
    } else {
        console.error(`Element with id ${elementId} not found`);
    }
}

type BillingCycleToggleProps = {
    value: BillingCycle;
    onChange: (value: BillingCycle) => unknown;
}

export function BillingCycleToggle({ value, onChange }: BillingCycleToggleProps) {
    const theme = useTheme();

    function switchToAnnual() {
        onChange(BillingCycle.Yearly);
    }
    function switchToMonthly() {
        onChange(BillingCycle.Monthly);
    }

    return (
        <Box
            role="radiogroup"
            aria-label="Billing cycle"
            sx={{
                position: "relative",
                display: "flex",
                borderRadius: "999px",
                width: 300,
                maxWidth: "100%",
                margin: "16px auto",
                backgroundColor: theme.palette.background.paper,
                boxShadow: theme.shadows[2],
                overflow: "overlay",
                "&:hover": {
                    filter: "brightness(1.05)",
                },
            }}
        >
            {/* Highlighter */}
            <Box
                sx={{
                    position: "absolute",
                    top: 0,
                    left: value === BillingCycle.Yearly ? 0 : "50%",
                    width: "50%",
                    height: "100%",
                    backgroundColor: theme.palette.secondary.main,
                    transition: "left 0.3s ease", // Smooth sliding effect
                    zIndex: 0, // Behind the text
                }}
            />

            {/* Annual Discount Option */}
            <ButtonBase
                role="radio"
                aria-checked={value === BillingCycle.Yearly}
                onClick={switchToAnnual}
                sx={{
                    flex: 1,
                    padding: "10px 16px",
                    color: value === BillingCycle.Yearly ? theme.palette.primary.contrastText : theme.palette.background.textSecondary,
                    fontWeight: value === BillingCycle.Yearly ? "bold" : "normal",
                    zIndex: 1, // Above the highlighter
                    textAlign: "center",
                }}
            >
                Annual discount
            </ButtonBase>

            {/* Monthly Plans Option */}
            <ButtonBase
                role="radio"
                aria-checked={value === "monthly"}
                onClick={switchToMonthly}
                sx={{
                    flex: 1,
                    padding: "10px 16px",
                    color: value === BillingCycle.Monthly ? theme.palette.primary.contrastText : theme.palette.background.textSecondary,
                    fontWeight: value === BillingCycle.Monthly ? "bold" : "normal",
                    zIndex: 1,
                    textAlign: "center",
                }}
            >
                Monthly plans
            </ButtonBase>
        </Box>
    );
}

type FrostedBoxProps = {
    currentTheme: "dark" | "light";
}
const FrostedBox = styled(Box)<FrostedBoxProps>(({ theme, currentTheme }) => ({
    background: currentTheme === "dark" ? "rgba(0, 0, 0, 0.7)" : "rgba(255, 255, 255, 0.7)",
    backdropFilter: "blur(10px)",
    border: 0,
    borderRadius: theme.spacing(4),
    padding: theme.spacing(4),
    textAlign: "center",
    display: "flex",
    flexDirection: "column",
    justifyContent: "space-between",
    height: "100%",
}));
const PricingTiersOuter = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    width: "max-content",
    flexWrap: "nowrap",
    gap: theme.spacing(4),
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    overflow: "auto",
    maxWidth: "100%",
    [theme.breakpoints.down("sm")]: {
        flexDirection: "column",
    },
}));

type PricingTierProps = {
    aboveButtonText?: string;
    buttonColor?: "error";
    buttonText?: string;
    currentTheme: "dark" | "light";
    features: string[];
    onClick: () => unknown;
    priceDisplay: React.ReactNode;
    priceType: PricingTierType;
    recommended?: boolean;
    title: string;
    titleHelp?: () => unknown;
    zIndex?: number;
}

export function PricingTier({
    aboveButtonText,
    buttonColor,
    buttonText,
    currentTheme,
    features,
    onClick,
    priceDisplay,
    priceType,
    recommended = false,
    title,
    titleHelp,
    zIndex,
}: PricingTierProps) {
    const { palette, shadows } = useTheme();

    return (
        <FrostedBox
            id={`${priceType}-tier`}
            p={4}
            border={1}
            borderRadius={2}
            currentTheme={currentTheme}
            textAlign="center"
            boxShadow={recommended ? shadows[3] : shadows[0]}
            display="flex"
            flexDirection="column"
            justifyContent="space-between"
            height="100%"
            width="350px"
            zIndex={zIndex}
        >
            {recommended && (
                <Box
                    bgcolor="secondary.main"
                    color="white"
                    p={1}
                    borderRadius="16px 16px 0 0"
                    marginX={-4}
                    marginTop={-4}
                    textAlign="center"
                >
                    <Typography variant="subtitle1">Recommended</Typography>
                </Box>
            )}
            <Typography variant="h6" component="h4" color={currentTheme === "dark" ? "white" : "black"} sx={{ mt: recommended ? 2 : 4, mb: 2, display: "flex", alignItems: "center", justifyContent: "center" }}>
                {title}
                {titleHelp && <IconButton color="info" onClick={titleHelp}><HelpIcon /></IconButton>}
            </Typography>
            {priceDisplay}
            <Box flexGrow={1}>
                <ul
                    style={{
                        color: currentTheme === "dark" ? "rgba(255, 255, 255, 0.7)" : "rgba(0, 0, 0, 0.7)",
                        textAlign: "left",
                        paddingLeft: "0",
                        listStyleType: "none",
                    }}
                >
                    {features.map((feature, index) => (
                        <li key={index} style={{ marginBottom: "8px", display: "flex", alignItems: "center" }}>
                            <span style={{ marginRight: "8px", color: palette.success.main }}>✓</span>
                            {feature}
                        </li>
                    ))}
                </ul>
            </Box>
            {aboveButtonText && (
                <Typography variant="body1" color="textSecondary" sx={{ mt: 2 }}>
                    {aboveButtonText}
                </Typography>
            )}
            {buttonText && onClick && (
                <Button
                    variant={recommended && !buttonColor ? "contained" : "outlined"}
                    color={buttonColor ?? (recommended ? "secondary" : "primary")}
                    onClick={onClick}
                    style={{
                        marginTop: "16px",
                        ...(buttonColor ? { borderColor: palette.error.main, color: palette.error.main } : {}),
                    }}
                >
                    {buttonText}
                </Button>
            )}
        </FrostedBox>
    );
}

export function PricingTiers({
    billingCycle,
    currentTheme,
    onTierClick,
    prices,
}: {
    billingCycle: BillingCycle,
    currentTheme: "dark" | "light",
    onTierClick: (tier: PricingTierType) => unknown,
    prices: SubscriptionPricesResponse | undefined,
}) {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const creditsAsBigInt = useMemo(() => {
        try {
            return currentUser.credits ? BigInt(currentUser.credits) : BigInt(0);
        } catch (error) {
            console.error(error);
            return BigInt(0);
        }
    }, [currentUser.credits]);

    const monthlyPricePerMonth = prices?.monthly ? (prices.monthly / 100).toFixed(2) : "Error";
    const yearlyPricePerMonth = prices?.yearly ? ((prices.yearly / 100) / YEARS_1_MONTHS).toFixed(2) : "Error";

    function onBasicClick() {
        onTierClick(PricingTierType.Basic);
    }
    function onProClick() {
        onTierClick(PricingTierType.Pro);
    }
    function onCreditsClick() {
        onTierClick(PricingTierType.Credits);
    }
    function onCreditsHelpClick() {
        const explainerId = "credits-explainer";
        if (window.location.href.includes(LINKS.Pro)) {
            scrollToElement(explainerId);
        } else {
            setLocation(`${LINKS.Pro}#${explainerId}`);
        }
    }

    const isLoggedIn = session?.isLoggedIn === true;
    const hasPremium = currentUser.hasPremium === true;
    const primaryColor = currentTheme === "dark" ? "white" : "black";
    const secondaryColor = currentTheme === "dark" ? "rgba(255, 255, 255, 0.7)" : "rgba(0, 0, 0, 0.7)";

    return (
        <PricingTiersOuter>
            <Box>
                <PricingTier
                    aboveButtonText={isLoggedIn && !hasPremium ? "Current plan" : undefined}
                    buttonText={!isLoggedIn ? pricingInfo[PricingTierType.Basic].buttonText : undefined}
                    currentTheme={currentTheme}
                    features={pricingInfo[PricingTierType.Basic].features}
                    onClick={onBasicClick}
                    priceType={PricingTierType.Basic}
                    priceDisplay={<Typography variant="h4" component="p" color={primaryColor} mb={4}>Free</Typography>}
                    recommended={pricingInfo[PricingTierType.Basic].recommended}
                    title={pricingInfo[PricingTierType.Basic].title}
                />
            </Box>
            <Box>
                <PricingTier
                    aboveButtonText={isLoggedIn && hasPremium ? "You're subscribed!" : undefined}
                    buttonColor={isLoggedIn && hasPremium ? "error" : undefined}
                    buttonText={isLoggedIn && hasPremium ? "Change plan" : pricingInfo[PricingTierType.Pro].buttonText}
                    currentTheme={currentTheme}
                    features={pricingInfo[PricingTierType.Pro].features}
                    onClick={onProClick}
                    priceDisplay={billingCycle === "monthly" ? (
                        <Box>
                            <Typography variant="h3" component="p" color={primaryColor}>${monthlyPricePerMonth}/mo</Typography>
                            <Typography variant="body2" color={secondaryColor}>Billed monthly</Typography>
                        </Box>
                    ) : (
                        <Box>
                            <Typography variant="h6" component="p" color={secondaryColor} style={{ textDecoration: "line-through" }}>
                                ${monthlyPricePerMonth}/mo
                            </Typography>
                            <Typography variant="h3" component="p" color={primaryColor}>${yearlyPricePerMonth}/mo</Typography>
                            <Typography variant="body2" color={secondaryColor}> Billed annually</Typography>
                        </Box >
                    )}
                    priceType={PricingTierType.Pro}
                    recommended={pricingInfo[PricingTierType.Pro].recommended}
                    title={pricingInfo[PricingTierType.Pro].title}
                />
            </Box>
            <Box>
                <PricingTier
                    aboveButtonText={creditsAsBigInt > 0 ? `Current credits: $${(Number(creditsAsBigInt / BigInt(API_CREDITS_MULTIPLIER)) / 100).toFixed(2)}` : undefined}
                    buttonText={pricingInfo[PricingTierType.Credits].buttonText}
                    currentTheme={currentTheme}
                    features={pricingInfo[PricingTierType.Credits].features}
                    onClick={onCreditsClick}
                    priceDisplay={<Typography variant="h4" component="p" color={primaryColor} mb={4}>Pay as you go</Typography>}
                    priceType={PricingTierType.Credits}
                    recommended={pricingInfo[PricingTierType.Credits].recommended}
                    title={pricingInfo[PricingTierType.Credits].title}
                    titleHelp={onCreditsHelpClick}
                />
            </Box>
        </PricingTiersOuter>
    );
}

const SupportOptionBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    padding: theme.spacing(1),
    borderRadius: theme.spacing(1.5),
    cursor: "pointer",
    // Background glow on hover
    "&:hover": {
        boxShadow: `0 0 10px 5px ${theme.palette.secondary.light}44`,
    },
    transition: "box-shadow 0.3s ease-in-out",
}));

function SupportOption({
    onClick,
    text,
}: {
    onClick?: () => unknown,
    text: string,
}) {
    const { palette } = useTheme();

    return (
        <SupportOptionBox onClick={onClick}>
            <Box width={20} height={20}>
                <ArrowUpRightIcon width={20} height={20} fill={palette.secondary.main} />
            </Box>
            <Typography variant="body1">{text}</Typography>
        </SupportOptionBox>
    );
}

function WaysToSupportUs() {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    function toSubscriptionBox() {
        scrollToElement(`${PricingTierType.Pro}-tier`);
    }
    function toBuyCreditsBox() {
        scrollToElement(`${PricingTierType.Credits}-tier`);
    }
    function toDonateBox() {
        scrollToElement(ELEMENT_IDS.ProViewDonateBox);
    }
    function toCreate() {
        openLink(setLocation, LINKS.Create);
    }

    return (
        <Box display="flex" flexDirection="column" gap={4} p={2} maxWidth="min(800px, 100%)" m="auto">
            <Typography variant="h4" m="auto">
                {t("ProIntro1")}
            </Typography>

            <Box display="flex" flexDirection="column" gap={2} p={2} justifyContent="center">
                <Typography variant="h6" sx={{ fontWeight: "bold" }}>{t("ProIntro2")}</Typography>
                <SupportOption
                    onClick={toSubscriptionBox}
                    text={t("ProSupport1", { monthlyCredits: `$${(Number(API_CREDITS_PREMIUM / API_CREDITS_MULTIPLIER) / 100).toFixed(2)}` })}
                />
                <SupportOption
                    onClick={toBuyCreditsBox}
                    text={t("ProSupport2")}
                />
                <SupportOption
                    onClick={toDonateBox}
                    text={t("ProSupport3")}
                />
                <SupportOption
                    onClick={toCreate}
                    text={t("ProSupport4")}
                />
            </Box>
        </Box>
    );
}

const dialogPaperProps = {
    sx: {
        bgcolor: "background.default",
    },
} as const;

export function CreditDialog({
    open,
    onClose,
    startCheckout,
}) {
    const [customCreditAmount, setCustomCreditAmount] = useState(25);
    function onCustomCreditAmountChange(event: React.ChangeEvent<HTMLInputElement>) {
        const value = Number(event.target.value);
        if (value < 1) {
            setCustomCreditAmount(1);
        } else {
            setCustomCreditAmount(value);
        }
    }

    function handleCheckout10Dollars() {
        const dollars = 10;
        startCheckout("credits", dollars * 100);
        onClose();
    }
    function handleCheckout20Dollars() {
        const dollars = 20;
        startCheckout("credits", dollars * 100);
        onClose();
    }
    function handleCheckout50Dollars() {
        const dollars = 50;
        startCheckout("credits", dollars * 100);
        onClose();
    }
    function handleCheckoutCustomAmount() {
        const dollars = customCreditAmount;
        startCheckout("credits", dollars * 100);
        onClose();
    }

    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby="select-credit-amount-dialog"
            PaperProps={dialogPaperProps}
        >
            <DialogTitle id="select-credit-amount-dialog">
                Select Credit Amount
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2}>
                    <Typography variant="body1">
                        Select the amount of credits to buy:
                    </Typography>
                    <Box display="flex" gap={1}>
                        <Button
                            fullWidth
                            onClick={handleCheckout10Dollars}
                            variant="outlined"
                        >
                            $10
                        </Button>
                        <Button
                            fullWidth
                            onClick={handleCheckout20Dollars}
                            variant="outlined"
                        >
                            $20
                        </Button>
                        <Button
                            fullWidth
                            onClick={handleCheckout50Dollars}
                            variant="outlined"
                        >
                            $50
                        </Button>
                    </Box>
                    <Typography variant="body1">Or enter a custom amount:</Typography>
                    <input
                        type="number"
                        value={customCreditAmount}
                        onChange={onCustomCreditAmountChange}
                        min={1}
                        style={{ width: "100%", padding: "8px" }}
                        placeholder="Custom Amount ($)"
                    />
                    <Button
                        fullWidth
                        onClick={handleCheckoutCustomAmount}
                        variant="contained"
                        disabled={!customCreditAmount || customCreditAmount < 1}
                    >
                        Buy Custom Amount
                    </Button>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
            </DialogActions>
        </Dialog>
    );
}

function DonationDialog({
    open,
    onClose,
    startCheckout,
}) {
    const [customDonationAmount, setCustomDonationAmount] = useState(25);
    function onCustomDonationAmountChange(event: React.ChangeEvent<HTMLInputElement>) {
        const value = Number(event.target.value);
        if (value < 1) {
            setCustomDonationAmount(1);
        } else {
            setCustomDonationAmount(value);
        }
    }

    function handleCheckoutCustomAmount() {
        const dollars = customDonationAmount;
        startCheckout("donation", dollars * 100);
        onClose();
    }

    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby="custom-donation-dialog"
        >
            <DialogTitle id="custom-donation-dialog">
                Enter Custom Donation Amount
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2}>
                    <input
                        type="number"
                        value={customDonationAmount}
                        onChange={onCustomDonationAmountChange}
                        min={1}
                        style={{ width: "100%", padding: "8px" }}
                        placeholder="Custom Amount ($)"
                    />
                    <Button
                        fullWidth
                        onClick={handleCheckoutCustomAmount}
                        variant="contained"
                        disabled={!customDonationAmount || customDonationAmount < 1}
                    >
                        Donate Custom Amount
                    </Button>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
            </DialogActions>
        </Dialog>
    );
}

const CheckStatusButton = styled(Button)(({ theme }) => ({
    textDecoration: "underline",
    textTransform: "none",
    color: theme.palette.background.textSecondary,
}));

export function ProView({
    display,
    onClose,
}: ProViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [isCreditDialogOpen, setIsCreditDialogOpen] = useState(false);
    const [isDonationDialogOpen, setIsDonationDialogOpen] = useState(false);
    function closeCreditDialog() {
        setIsCreditDialogOpen(false);
    }
    function closeDonationDialog() {
        setIsDonationDialogOpen(false);
    }

    const {
        checkFailedCredits,
        checkFailedSubscription,
        prices,
        redirectToCustomerPortal,
        startCheckout,
    } = useStripe();
    const [billingCycle, setBillingCycle] = useState<BillingCycle>(BillingCycle.Yearly);

    function onTierClick(tier: PricingTierType) {
        switch (tier) {
            case PricingTierType.Basic:
                openLink(setLocation, LINKS.Signup);
                break;
            case PricingTierType.Pro:
                if (currentUser.hasPremium) {
                    redirectToCustomerPortal();
                } else {
                    startCheckout(billingCycle === BillingCycle.Yearly ? PaymentType.PremiumYearly : PaymentType.PremiumMonthly);
                }
                break;
            case PricingTierType.Credits:
                if (session?.isLoggedIn) {
                    setIsCreditDialogOpen(true);
                } else {
                    PubSub.get().publish("snack", { message: "Please sign in first", severity: SnackSeverity.Warning });
                    const currentUrl = window.location.href;
                    openLink(setLocation, LINKS.Signup, { searchParams: { redirect: currentUrl } });
                }
                break;
        }
    }

    function onDonateClick() {
        setIsDonationDialogOpen(true);
    }

    return (
        <>
            <CreditDialog
                open={isCreditDialogOpen}
                onClose={closeCreditDialog}
                startCheckout={startCheckout}
            />
            <DonationDialog
                open={isDonationDialogOpen}
                onClose={closeDonationDialog}
                startCheckout={startCheckout}
            />
            <PageContainer size="fullSize">
                {palette.mode === "dark" && <RandomBlobs numberOfBlobs={isMobile ? 5 : 8} />}
                <ScrollBox>
                    <TopBar
                        display={display}
                        onClose={onClose}
                        title={t("ProGet")}
                        titleBehaviorDesktop="ShowIn"
                    />
                    <Box display="flex" flexDirection="column" gap={8} margin="auto">
                        <WaysToSupportUs />
                        <Box display="flex" flexDirection="column" pt={4} alignItems="center" minHeight="100vh" justifyContent="center">
                            <Typography variant="h4" m="auto" p={2}>Select the perfect plan for your needs.</Typography>
                            <Typography variant="body1" textAlign="center" p={2}>
                                Maximize your potential — become a Vrooli Pro user today!
                            </Typography>
                            <Box mt={2} mb={2}>
                                <BillingCycleToggle value={billingCycle} onChange={setBillingCycle} />
                            </Box>
                            <PricingTiers
                                billingCycle={billingCycle}
                                currentTheme={palette.mode}
                                onTierClick={onTierClick}
                                prices={prices}
                            />
                            <Box display="flex" justifyContent="center" alignItems="center" gap={1} width="min(800px, 100%)" m="auto">
                                {!currentUser.hasPremium && <CheckStatusButton
                                    onClick={checkFailedSubscription}
                                    variant="text"
                                >Didn&apos;t receive Pro?</CheckStatusButton>}
                                {!currentUser.hasPremium && session?.isLoggedIn && <p>•</p>}
                                {session?.isLoggedIn && <CheckStatusButton
                                    onClick={checkFailedCredits}
                                    variant="text"
                                >Didn&apos;t receive credits?</CheckStatusButton>}
                            </Box>
                        </Box>
                        <Box id={ELEMENT_IDS.ProViewDonateBox} p={2} m="auto" mt={4} mb={4} maxWidth="800px" width="100%">
                            <Typography variant="h4" pb={2}>
                                Love Vrooli? Consider donating to support us!
                            </Typography>
                            <Typography variant="body1" pb={4}>
                                Vrooli is a labor of love. I&apos;ve spent countless hours and life savings to develop this product. Consider donating to support our efforts to democratize AI. Every dollar helps!💙
                            </Typography>
                            <Box width="min(300px, 100%)" margin="auto">
                                <Button
                                    fullWidth
                                    variant="contained"
                                    onClick={onDonateClick}
                                    startIcon={<HeartFilledIcon />}
                                >
                                    Donate
                                </Button>
                            </Box>
                        </Box>
                        {/* TODO add video to showcase pro features */}
                        <Box id={ELEMENT_IDS.ProViewFAQBox} mt={4} px={4} py={3} width="min(800px, 100%)" m="auto">
                            <Typography variant="h4" style={{ textAlign: "center", marginBottom: "1.5rem", color: "primary.main" }}>Frequently Asked Questions</Typography>

                            <Typography variant="h6" color="textSecondary" id="credits-explainer">1. How is credit usage calculated?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>We base the cost of performing AI-related tasks by the model used, the size of the text passed into each request, and the size of the text returned. We try our best to charge almost exactly what it costs us to run the AI models, and we're always looking for ways to make it cheaper for you.</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Over time, as models become cheaper and more efficient routines are created, the cost of performing tasks should decrase.</Typography>

                            <Typography variant="h6" color="textSecondary">2. Can I buy more credits?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Yes! You can buy more credits at any time. If you have a <i>Pro</i> subscription, you can also wait until the first of the month to receive your monthly credits.</Typography>

                            <Typography variant="h6" color="textSecondary">3. What happens to leftover credits at the beginning of the month?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>If you still have credits left over from the previous month, they will be added to your new monthly credits — up to a maximum of 6 months' worth of credits. If you have more than 6 months' worth of credits, your credit balance won't change.</Typography>

                            <Typography variant="h6" color="textSecondary">4. What happens if my credit balance goes negative?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>If your credit balance goes negative, you won't be able to perform AI-related tasks until your balance is positive again. This is our mistake, so you won't be charged any extra fees.</Typography>

                            <Typography variant="h6" color="textSecondary">5. What are the differences between the <i>Free</i> and <i>Pro</i> plans?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>The <i>Free</i>  plan offers basic functionalities like taking notes, creating reminders, viewing routines, participating in chat groups with other users, and more. The <i>Pro</i> plan enhances your experience with advanced features like AI-powered integrations, enhanced analytics, and more.</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Note that you can still use AI-related features in the <i>Free</i> plan, but you'll need to buy credits.</Typography>

                            <Typography variant="h6" color="textSecondary">6. Are there any long-term commitments or contracts?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>No, there are no long-term commitments or contracts. You can choose to subscribe on a monthly or yearly basis and can cancel anytime.</Typography>

                            <Typography variant="h6" color="textSecondary">7. What happens if I decide to cancel my subscription?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>If you decide to cancel, you will be downgraded to the <i>Free</i>  plan at the end of your billing cycle and won't be charged thereafter. You'll still have access to all your data and can upgrade again anytime.</Typography>

                            <Typography variant="h6" color="textSecondary">8. How secure is my payment information?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Payments are handled securely using <Link href="https://stripe.com/" target="_blank" rel="noopener noreferrer">Stripe</Link>, which is a leading third-party payment processor. We don't store any payment information on our servers.</Typography>

                            <Typography variant="h6" color="textSecondary">9. Do you offer refunds?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>We do not offer refunds for credits or subscriptions at this time. If you cancel your subscription, you will still have access to the pro features until the end of your billing cycle.</Typography>

                            <Typography variant="h6" color="textSecondary">10. Can I change from a monthly to a yearly subscription, or vice versa?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Absolutely! You can switch between plans from the customer portal or contact our support for assistance.</Typography>

                            <Typography variant="h6" color="textSecondary">11. What does "Early access to updates and improvements" mean?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>Pro users get access to new features, improvements, and updates before they are rolled out to all users. It's our way of saying thank you for supporting us! (and also testing new features before wide release, but that doesn't sound as nice)</Typography>

                            <Typography variant="h6" color="textSecondary">12. How does the mobile app differ from the web version?</Typography>
                            <Typography variant="body1" style={{ marginBottom: "1rem" }}>The mobile app is the same or almost the same as the web version. It's a Progressive Web App (PWA), which means it's a website that can be installed on your phone and used like a native app. It's a great way to stay productive on the go!</Typography>

                            <Typography variant="h6" color="textSecondary">13. I have more questions. How can I reach out?</Typography>
                            <Typography variant="body1">We&apos;re here to help! You can contact our support team at <Link href="mailto:official@vrooli.com" target="_blank" rel="noopener noreferrer">official@vrooli.com</Link> or message us on <Link href="https://x.com/vrooliofficial" target="_blank" rel="noopener noreferrer">X</Link>.</Typography>
                        </Box>
                    </Box>
                    <Footer />
                </ScrollBox>
            </PageContainer>
        </>
    );
}
