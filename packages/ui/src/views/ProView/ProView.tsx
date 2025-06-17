import { styled, useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import ButtonBase from "@mui/material/ButtonBase";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import { API_CREDITS_MULTIPLIER, API_CREDITS_PREMIUM, LINKS, PaymentType, YEARS_1_MONTHS, type SubscriptionPricesResponse } from "@vrooli/shared";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Button } from "../../components/buttons/Button.js";
import { Footer } from "../../components/navigation/Footer.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SnackSeverity } from "../../components/snacks/BasicSnack/BasicSnack.js";
import { SessionContext } from "../../contexts/session.js";
import { useStripe } from "../../hooks/useStripe.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { type ViewProps } from "../../types.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { RandomBlobs } from "../main/components/RandomBlobs.js";

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

const HIGHLIGHT_TIMEOUT_MS = 1000;

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
        setTimeout(function removeHighlightTimeout() {
            element.style.backgroundColor = originalBackground;
            element.style.borderRadius = originalBorderRadius;
        }, HIGHLIGHT_TIMEOUT_MS);
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
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(4),
    // eslint-disable-next-line no-magic-numbers
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
    // eslint-disable-next-line no-magic-numbers
    gap: theme.spacing(4),
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    paddingTop: theme.spacing(4), // Add padding to accommodate tooltip
    overflow: "auto",
    maxWidth: "100%",
    position: "relative",
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
    const { t } = useTranslation();
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
                {titleHelp && (
                    <IconButton
                        aria-label={t("Help")}
                        color="info"
                        onClick={titleHelp}
                    >
                        <IconCommon
                            decorative
                            name="Help"
                        />
                    </IconButton>
                )}
            </Typography>
            {priceDisplay}
            {/* Add "Cancel anytime" for Pro subscription */}
            {priceType === PricingTierType.Pro && (
                <Typography
                    variant="body2"
                    sx={{
                        color: palette.success.main,
                        fontWeight: 500,
                        mt: 1,
                        mb: 2,
                    }}
                >
                    ✓ Cancel anytime, no questions asked
                </Typography>
            )}
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
            {buttonText && onClick && recommended && (
                <Button
                    variant={!buttonColor ? "space" : "outline"}
                    color={buttonColor ?? "secondary"}
                    onClick={onClick}
                    style={{
                        marginTop: "16px",
                        ...(buttonColor ? { borderColor: palette.error.main, color: palette.error.main } : {}),
                    }}
                >
                    {buttonText}
                </Button>
            )}
            {buttonText && onClick && !recommended && (
                <Button
                    variant={buttonColor ? "danger" : "outline"}
                    onClick={onClick}
                    className="tw-mt-4"
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
    const { palette } = useTheme();
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
            <Box position="relative">
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
                {/* Phone verification callout for credits */}
                {isLoggedIn && 
                 currentUser.phoneNumberVerified !== true && 
                 currentUser.hasReceivedPhoneVerificationReward !== true && (
                    <Box
                        sx={{
                            position: "absolute",
                            top: -30,
                            right: 20,
                            backgroundColor: palette.secondary.main,
                            color: "white",
                            borderRadius: 2,
                            px: 2,
                            py: 1,
                            fontWeight: "bold",
                            fontSize: "0.875rem",
                            boxShadow: 3,
                            display: { xs: "none", sm: "block" },
                            zIndex: 10,
                            "&::after": {
                                content: '""',
                                position: "absolute",
                                bottom: -8,
                                right: "40%",
                                width: 0,
                                height: 0,
                                borderLeft: "8px solid transparent",
                                borderRight: "8px solid transparent",
                                borderTop: `8px solid ${palette.secondary.main}`,
                            },
                        }}
                    >
                        Get $5 FREE!
                    </Box>
                )}
            </Box>
        </PricingTiersOuter>
    );
}

const SupportOptionBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    padding: theme.spacing(1),
    // eslint-disable-next-line no-magic-numbers
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
        <SupportOptionBox
            aria-label={text}
            onClick={onClick}
        >
            <Box width={20} height={20}>
                <IconCommon
                    decorative
                    fill={palette.secondary.main}
                    name="ArrowUpRight"
                    size={20}
                />
            </Box>
            <Typography variant="body1">{text}</Typography>
        </SupportOptionBox>
    );
}

// Phone verification offer banner
function PhoneVerificationBanner() {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);

    // Don't show if user is not logged in, already has a verified phone, or already received the reward
    // TODO: Backend needs to be updated to populate phoneNumberVerified and hasReceivedPhoneVerificationReward fields in SessionUser
    if (!session?.isLoggedIn || 
        currentUser.phoneNumberVerified === true || 
        currentUser.hasReceivedPhoneVerificationReward === true) {
        return null;
    }

    function handleVerifyPhone() {
        if (!session?.isLoggedIn) {
            // If not logged in, redirect to signup with a redirect back to settings
            const redirectUrl = `${window.location.origin}${LINKS.SettingsAuthentication}`;
            openLink(setLocation, LINKS.Signup, { searchParams: { redirect: redirectUrl } });
        } else {
            openLink(setLocation, LINKS.SettingsAuthentication);
        }
    }

    // Show different message based on whether they can still get the reward
    const isFirstTimeOffer = currentUser.hasReceivedPhoneVerificationReward !== true;

    return (
        <Box
            sx={{
                background: `linear-gradient(135deg, ${palette.secondary.light}22, ${palette.primary.light}22)`,
                border: `2px solid ${palette.secondary.main}`,
                borderRadius: 2,
                p: 2,
                mx: 2,
                mb: 4,
                display: "flex",
                flexDirection: { xs: "column", sm: "row" },
                alignItems: "center",
                gap: 2,
                boxShadow: `0 4px 12px ${palette.secondary.main}33`,
            }}
        >
            <Box display="flex" alignItems="center" gap={1} flex={1}>
                <IconCommon
                    name="Phone"
                    fill={palette.secondary.main}
                    size={32}
                />
                <Box>
                    <Typography variant="h6" sx={{ fontWeight: "bold", color: palette.text.primary }}>
                        {isFirstTimeOffer ? "First-time offer: Get $5 in FREE Credits!" : "Verify Your Phone Number"}
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                        {isFirstTimeOffer 
                            ? "Verify your phone number to claim your one-time bonus" 
                            : "Add phone verification for enhanced account security"
                        }
                    </Typography>
                </Box>
            </Box>
            <Button
                variant={isFirstTimeOffer ? "space" : "primary"}
                onClick={handleVerifyPhone}
                startIcon={<IconCommon name="Phone" />}
            >
                Verify Now
            </Button>
        </Box>
    );
}

// Payment security badges component
function SecurityBadges() {
    const { palette } = useTheme();

    return (
        <Box
            display="flex"
            alignItems="center"
            justifyContent="center"
            gap={2}
            sx={{
                opacity: 0.8,
                mt: 2,
            }}
        >
            <Box display="flex" alignItems="center" gap={0.5}>
                <IconCommon
                    name="Lock"
                    size={20}
                    fill={palette.mode === "dark" ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)"}
                />
                <Typography
                    variant="caption"
                    sx={{ color: palette.mode === "dark" ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)" }}
                >
                    SSL Secured
                </Typography>
            </Box>
            <Typography
                variant="caption"
                sx={{ color: palette.mode === "dark" ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)" }}
            >
                •
            </Typography>
            <Box display="flex" alignItems="center" gap={0.5}>
                <Typography
                    variant="caption"
                    sx={{
                        color: palette.mode === "dark" ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
                        fontWeight: 600
                    }}
                >
                    Powered by Stripe
                </Typography>
            </Box>
        </Box>
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
    const { palette } = useTheme();
    const [selectedAmount, setSelectedAmount] = useState<number | null>(20);
    const [customAmount, setCustomAmount] = useState("");
    const [showCustom, setShowCustom] = useState(false);
    
    // Credit packages with psychological anchoring
    const creditPackages = [
        { 
            value: 10, 
            label: "$10", 
            credits: "~1,000",
            tasks: "~50 tasks",
            emoji: "⚡"
        },
        { 
            value: 20, 
            label: "$20", 
            credits: "~2,000",
            tasks: "~100 tasks",
            emoji: "🔥",
            popular: true,
            savings: null
        },
        { 
            value: 50, 
            label: "$50", 
            credits: "~5,000",
            tasks: "~250 tasks",
            emoji: "🚀"
        },
        { 
            value: 100, 
            label: "$100", 
            credits: "~10,000",
            tasks: "~500 tasks",
            emoji: "💎"
        }
    ];

    function handleAmountSelect(amount: number) {
        setSelectedAmount(amount);
        setShowCustom(false);
        setCustomAmount("");
    }

    function handleCustomClick() {
        setShowCustom(true);
        setSelectedAmount(null);
    }

    function onCustomAmountChange(event: React.ChangeEvent<HTMLInputElement>) {
        const value = event.target.value;
        setCustomAmount(value);
        const numValue = Number(value);
        if (numValue > 0) {
            setSelectedAmount(numValue);
        } else {
            setSelectedAmount(null);
        }
    }

    function handleCheckout() {
        if (selectedAmount && selectedAmount > 0) {
            startCheckout("credits", selectedAmount * 100);
            onClose();
        }
    }

    // Calculate approximate credits and tasks
    function getCreditsInfo(amount: number | null) {
        if (!amount) return null;
        const credits = amount * 100; // Approximate
        const tasks = Math.floor(credits / 20); // Approximate tasks
        return { credits, tasks };
    }

    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby="credit-dialog"
            PaperProps={{
                sx: {
                    bgcolor: "background.default",
                    borderRadius: 3,
                    maxWidth: 520,
                    width: "100%",
                }
            }}
        >
            <DialogTitle id="credit-dialog" sx={{ textAlign: "center", pb: 1 }}>
                <Typography variant="h5" sx={{ fontWeight: "bold", mb: 1 }}>
                    Power Up Your AI Assistant 🚀
                </Typography>
                <Typography variant="body2" color="textSecondary">
                    Get credits to run AI-powered tasks and automations
                </Typography>
            </DialogTitle>
            
            <DialogContent sx={{ px: 3 }}>
                {/* Value proposition banner */}
                <Box sx={{ 
                    mb: 3, 
                    p: 2, 
                    background: `linear-gradient(135deg, ${palette.primary.main}22, ${palette.secondary.main}22)`,
                    borderRadius: 2,
                    textAlign: "center"
                }}>
                    <Typography variant="body2" sx={{ fontWeight: "bold", color: palette.primary.main }}>
                        💡 Pro Tip: Buy credits as you need them!
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                        Credits never expire • Use them anytime
                    </Typography>
                </Box>

                {/* Credit packages grid */}
                <Box display="grid" gridTemplateColumns="repeat(2, 1fr)" gap={2} mb={3}>
                    {creditPackages.map((pkg) => (
                        <Button
                            key={pkg.value}
                            onClick={() => handleAmountSelect(pkg.value)}
                            variant={selectedAmount === pkg.value ? "primary" : "outline"}
                            className="tw-h-auto"
                            borderRadius="minimal"
                            sx={{
                                position: "relative",
                                py: 3,
                                px: 2,
                                minHeight: "120px",
                                // Override the button's internal span to display content vertically
                                "& > span:first-of-type": {
                                    flexDirection: "column !important",
                                    alignItems: "center !important",
                                    gap: "4px !important",
                                    height: "100%",
                                    width: "100%"
                                },
                                ...(pkg.popular && {
                                    border: `2px solid ${palette.secondary.main}`,
                                    "&::before": {
                                        content: '"Most Popular"',
                                        position: "absolute",
                                        top: -10,
                                        left: "50%",
                                        transform: "translateX(-50%)",
                                        bgcolor: palette.secondary.main,
                                        color: "white",
                                        fontSize: "0.65rem",
                                        px: 1,
                                        py: 0.25,
                                        borderRadius: 1,
                                        fontWeight: "bold",
                                        zIndex: 1
                                    }
                                })
                            }}
                        >
                            <Typography variant="h6" sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                                {pkg.emoji} {pkg.label}
                            </Typography>
                            <Typography variant="caption" color="textSecondary">
                                {pkg.credits} credits
                            </Typography>
                            <Typography variant="caption" sx={{ 
                                color: palette.primary.main,
                                fontWeight: "bold" 
                            }}>
                                {pkg.tasks}
                            </Typography>
                        </Button>
                    ))}
                </Box>

                {/* Custom amount option */}
                <Box sx={{ mb: 3 }}>
                    <Button
                        fullWidth
                        onClick={handleCustomClick}
                        variant={showCustom ? "primary" : "ghost"}
                        size="md"
                    >
                        Custom Amount
                    </Button>
                </Box>

                {showCustom && (
                    <Box sx={{ mb: 2 }}>
                        <input
                            type="number"
                            value={customAmount}
                            onChange={onCustomAmountChange}
                            min={1}
                            style={{ 
                                width: "100%", 
                                padding: "12px",
                                fontSize: "16px",
                                borderRadius: "8px",
                                border: `2px solid ${palette.primary.main}`,
                                outline: "none"
                            }}
                            placeholder="Enter amount ($)"
                            autoFocus
                        />
                        {selectedAmount && showCustom && (
                            <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: "block" }}>
                                You'll get approximately {getCreditsInfo(selectedAmount)?.credits.toLocaleString()} credits
                            </Typography>
                        )}
                    </Box>
                )}

                {/* What you can do with credits */}
                {selectedAmount && (
                    <Box sx={{
                        p: 2,
                        bgcolor: palette.info.main + "11",
                        borderRadius: 2,
                        mb: 2
                    }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: "bold", mb: 1 }}>
                            With ${selectedAmount} you can:
                        </Typography>
                        <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
                            {selectedAmount >= 50 && (
                                <Typography component="li" variant="body2">
                                    Run complex AI workflows for a month
                                </Typography>
                            )}
                            {selectedAmount >= 20 && (
                                <Typography component="li" variant="body2">
                                    Generate 100+ AI responses
                                </Typography>
                            )}
                            <Typography component="li" variant="body2">
                                Automate {getCreditsInfo(selectedAmount)?.tasks}+ daily tasks
                            </Typography>
                            <Typography component="li" variant="body2">
                                Access premium AI models
                            </Typography>
                        </Box>
                    </Box>
                )}

                {/* Purchase button */}
                <Button
                    fullWidth
                    onClick={handleCheckout}
                    variant="space"
                    size="lg"
                    disabled={!selectedAmount || selectedAmount < 1}
                    startIcon={<IconCommon name="ShoppingCart" />}
                    sx={{ mb: 2 }}
                >
                    Buy {selectedAmount ? `$${selectedAmount} in Credits` : "Credits"}
                </Button>

                {/* Trust signals */}
                <Box sx={{ textAlign: "center", opacity: 0.7 }}>
                    <Typography variant="caption" color="textSecondary">
                        🔒 Secure checkout • Credits never expire • Instant activation
                    </Typography>
                </Box>
            </DialogContent>
            
            <DialogActions sx={{ justifyContent: "center", pb: 2 }}>
                <Button onClick={onClose} variant="ghost" size="sm">
                    Maybe later
                </Button>
            </DialogActions>
        </Dialog>
    );
}

function DonationDialog({
    open,
    onClose,
    startCheckout,
}) {
    const { palette } = useTheme();
    const [selectedAmount, setSelectedAmount] = useState<number | null>(15);
    const [customAmount, setCustomAmount] = useState("");
    const [showCustom, setShowCustom] = useState(false);
    
    // Suggested amounts with psychological anchoring
    const donationOptions = [
        { 
            value: 5, 
            label: "$5", 
            impact: "Keeps us caffeinated",
            emoji: "☕"
        },
        { 
            value: 15, 
            label: "$15", 
            impact: "Fuels a day of coding",
            emoji: "🔥",
            popular: true
        },
        { 
            value: 25, 
            label: "$25", 
            impact: "Supports server costs",
            emoji: "🎯"
        },
        { 
            value: 50, 
            label: "$50", 
            impact: "Funds new features",
            emoji: "🚀"
        }
    ];

    function handleAmountSelect(amount: number) {
        setSelectedAmount(amount);
        setShowCustom(false);
        setCustomAmount("");
    }

    function handleCustomClick() {
        setShowCustom(true);
        setSelectedAmount(null);
    }

    function onCustomAmountChange(event: React.ChangeEvent<HTMLInputElement>) {
        const value = event.target.value;
        setCustomAmount(value);
        const numValue = Number(value);
        if (numValue > 0) {
            setSelectedAmount(numValue);
        } else {
            setSelectedAmount(null);
        }
    }

    function handleDonate() {
        if (selectedAmount && selectedAmount > 0) {
            startCheckout("donation", selectedAmount * 100);
            onClose();
        }
    }

    // Calculate impact message based on amount
    function getImpactMessage(amount: number | null) {
        if (!amount) return "";
        if (amount >= 100) return "You're a hero! This funds a week of development 🦸";
        if (amount >= 50) return "Amazing! This helps us add new features 🎉";
        if (amount >= 25) return "Wonderful! This keeps our servers running smoothly";
        if (amount >= 15) return "Thank you! This fuels our development team";
        if (amount >= 5) return "Every bit helps! This keeps us caffeinated";
        return "Your support means the world to us";
    }

    return (
        <Dialog
            open={open}
            onClose={onClose}
            aria-labelledby="donation-dialog"
            PaperProps={{
                sx: {
                    bgcolor: "background.default",
                    borderRadius: 3,
                    maxWidth: 480,
                    width: "100%",
                }
            }}
        >
            <DialogTitle id="donation-dialog" sx={{ textAlign: "center", pb: 1 }}>
                <Typography variant="h5" sx={{ fontWeight: "bold", mb: 1 }}>
                    Support Vrooli's Mission 💙
                </Typography>
                <Typography variant="body2" color="textSecondary">
                    Help us democratize AI and keep building amazing features
                </Typography>
            </DialogTitle>
            
            <DialogContent sx={{ px: 3 }}>
                {/* Community support progress */}
                {/* TODO: Replace hardcoded values (78%, 234 supporters) with actual data from API */}
                <Box sx={{ 
                    mb: 3, 
                    p: 2, 
                    background: `linear-gradient(135deg, ${palette.secondary.main}15, ${palette.primary.main}15)`,
                    borderRadius: 2 
                }}>
                    <Box display="flex" justifyContent="space-between" mb={1}>
                        <Typography variant="body2" color="textSecondary">
                            Monthly Support Goal
                        </Typography>
                        <Typography variant="body2" sx={{ fontWeight: "bold", color: palette.primary.main }}>
                            78% reached
                        </Typography>
                    </Box>
                    <Box sx={{ 
                        height: 8, 
                        bgcolor: palette.action.disabledBackground,
                        borderRadius: 1,
                        overflow: "hidden"
                    }}>
                        <Box sx={{
                            width: "78%",
                            height: "100%",
                            background: `linear-gradient(90deg, ${palette.secondary.main}, ${palette.primary.main})`,
                            transition: "width 0.5s ease"
                        }} />
                    </Box>
                    <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, display: "block" }}>
                        234 amazing supporters this month 🙏
                    </Typography>
                </Box>

                {/* Quick donation amounts */}
                <Typography variant="subtitle2" sx={{ mb: 2, fontWeight: "bold" }}>
                    Choose an amount:
                </Typography>
                
                <Box display="grid" gridTemplateColumns="repeat(2, 1fr)" gap={2} mb={3}>
                    {donationOptions.map((option) => (
                        <Button
                            key={option.value}
                            onClick={() => handleAmountSelect(option.value)}
                            variant={selectedAmount === option.value ? "primary" : "outline"}
                            className="tw-h-auto"
                            borderRadius="minimal"
                            sx={{
                                position: "relative",
                                py: 3,
                                px: 2,
                                minHeight: "100px",
                                // Override the button's internal span to display content vertically
                                "& > span:first-of-type": {
                                    flexDirection: "column !important",
                                    alignItems: "center !important",
                                    gap: "4px !important",
                                    height: "100%",
                                    width: "100%"
                                },
                                ...(option.popular && {
                                    border: `2px solid ${palette.secondary.main}`,
                                    "&::before": {
                                        content: '"Most Common"',
                                        position: "absolute",
                                        top: -10,
                                        left: "50%",
                                        transform: "translateX(-50%)",
                                        bgcolor: palette.secondary.main,
                                        color: "white",
                                        fontSize: "0.65rem",
                                        px: 1,
                                        py: 0.25,
                                        borderRadius: 1,
                                        fontWeight: "bold",
                                        zIndex: 1
                                    }
                                })
                            }}
                        >
                            <Typography variant="h6" sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                                {option.emoji} {option.label}
                            </Typography>
                            <Typography variant="caption" color="textSecondary">
                                {option.impact}
                            </Typography>
                        </Button>
                    ))}
                </Box>

                {/* Custom amount option */}
                <Box sx={{ mb: 3 }}>
                    <Button
                        fullWidth
                        onClick={handleCustomClick}
                        variant={showCustom ? "primary" : "ghost"}
                        size="md"
                    >
                        Other Amount
                    </Button>
                </Box>

                {showCustom && (
                    <Box sx={{ mb: 2 }}>
                        <input
                            type="number"
                            value={customAmount}
                            onChange={onCustomAmountChange}
                            min={1}
                            style={{ 
                                width: "100%", 
                                padding: "12px",
                                fontSize: "16px",
                                borderRadius: "8px",
                                border: `2px solid ${palette.primary.main}`,
                                outline: "none"
                            }}
                            placeholder="Enter amount ($)"
                            autoFocus
                        />
                    </Box>
                )}

                {/* Impact message */}
                {selectedAmount && (
                    <Box sx={{
                        p: 2,
                        bgcolor: palette.success.main + "15",
                        borderRadius: 2,
                        mb: 2,
                        textAlign: "center"
                    }}>
                        <Typography variant="body2" sx={{ fontWeight: "bold", color: palette.success.dark }}>
                            {getImpactMessage(selectedAmount)}
                        </Typography>
                    </Box>
                )}

                {/* Personal message from founder */}
                <Box sx={{
                    p: 2,
                    bgcolor: palette.background.paper,
                    borderRadius: 2,
                    mb: 2,
                    borderLeft: `4px solid ${palette.primary.main}`
                }}>
                    <Typography variant="caption" sx={{ fontStyle: "italic" }}>
                        "Your support directly helps us build a platform where AI is accessible to everyone. 
                        Thank you for believing in our mission!" 
                    </Typography>
                    <Typography variant="caption" sx={{ display: "block", mt: 1, fontWeight: "bold" }}>
                        - Matt, Founder
                    </Typography>
                </Box>

                {/* Donate button */}
                <Button
                    fullWidth
                    onClick={handleDonate}
                    variant="space"
                    size="lg"
                    disabled={!selectedAmount || selectedAmount < 1}
                    startIcon={<IconCommon name="HeartFilled" />}
                    sx={{ mb: 2 }}
                >
                    Donate {selectedAmount ? `$${selectedAmount}` : ""}
                </Button>

                {/* Trust signals */}
                <Box sx={{ textAlign: "center", opacity: 0.7 }}>
                    <Typography variant="caption" color="textSecondary">
                        🔒 Secure via Stripe • 100% goes to development
                    </Typography>
                </Box>
            </DialogContent>
            
            <DialogActions sx={{ justifyContent: "center", pb: 2 }}>
                <Button onClick={onClose} variant="ghost" size="sm">
                    Maybe later
                </Button>
            </DialogActions>
        </Dialog>
    );
}

// Custom styling for status check buttons using our Button component
const checkStatusButtonStyles = "tw-underline tw-normal-case tw-text-gray-600";

export function ProView(_props: ViewProps) {
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
                    <Navbar title={t("ProGet")} />
                    <Box display="flex" flexDirection="column" gap={8} margin="auto">
                        <WaysToSupportUs />
                        {/* Phone verification banner */}
                        <PhoneVerificationBanner />
                        <Box display="flex" flexDirection="column" pt={4} alignItems="center" minHeight="100vh" justifyContent="center">
                            <Typography variant="h4" m="auto" p={2}>Select the perfect plan for your needs.</Typography>
                            <Typography variant="body1" textAlign="center" p={2}>
                                Maximize your potential — become a Vrooli Pro user today!
                            </Typography>
                            <Box mt={2} mb={2}>
                                <BillingCycleToggle value={billingCycle} onChange={setBillingCycle} />
                            </Box>
                            {/* Security badges */}
                            <SecurityBadges />
                            <PricingTiers
                                billingCycle={billingCycle}
                                currentTheme={palette.mode}
                                onTierClick={onTierClick}
                                prices={prices}
                            />
                            <Box display="flex" justifyContent="center" alignItems="center" gap={1} width="min(800px, 100%)" m="auto">
                                {!currentUser.hasPremium && <Button
                                    onClick={checkFailedSubscription}
                                    variant="ghost"
                                    className={checkStatusButtonStyles}
                                >Didn&apos;t receive Pro?</Button>}
                                {!currentUser.hasPremium && session?.isLoggedIn && <p>•</p>}
                                {session?.isLoggedIn && <Button
                                    onClick={checkFailedCredits}
                                    variant="ghost"
                                    className={checkStatusButtonStyles}
                                >Didn&apos;t receive credits?</Button>}
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
                                    variant="space"
                                    borderRadius="pill"
                                    onClick={onDonateClick}
                                    startIcon={<IconCommon
                                        decorative
                                        name="HeartFilled"
                                    />}
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
