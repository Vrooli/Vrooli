import { styled, useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Fade from "@mui/material/Fade";
import Typography from "@mui/material/Typography";
import { convertToDot, valueFromDot } from "@vrooli/shared";
import { useCallback, type ChangeEvent } from "react";
import { useTranslation } from "react-i18next";
import privacyMarkdown from "../../assets/policy/privacy.md";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { Footer } from "../../components/navigation/Footer.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { MarkdownDisplay } from "../../components/text/MarkdownDisplay.js";
import { useMarkdown } from "../../hooks/useMarkdown.js";
import { useTabs, type PageTab } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { type ViewProps } from "../../types.js";
import { BUSINESS_DATA } from "../../utils/consts.js";
import { PolicyTabOption, policyTabParams, type PolicyTabsInfo, type TabParamBase } from "../../utils/search/objectToSearch.js";

function injectBusinessData(markdown: string) {
    const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
    business_fields.forEach(f => {
        const regex = new RegExp(`<${f}>`, "g");
        markdown = markdown?.replace(regex, valueFromDot(BUSINESS_DATA, f) || "") ?? "";
    });
    return markdown;
}

const StyledContainer = styled(Container)(({ theme }) => ({
    paddingTop: theme.spacing(5),
    paddingBottom: theme.spacing(10),
    [theme.breakpoints.down("sm")]: {
        paddingTop: theme.spacing(2),
        paddingBottom: theme.spacing(4),
    },
}));

const ContentBox = styled(Box)(({ theme }) => ({
    padding: theme.spacing(7),
    borderRadius: theme.spacing(3),
    background: theme.palette.background.paper,
    boxShadow: theme.shadows[3],
    border: `1.5px solid ${theme.palette.divider}`,
    position: "relative",
    overflow: "hidden",
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(4),
    transition: "box-shadow 0.2s",
    '&:hover': {
        boxShadow: theme.shadows[6],
    },
    "& .MuiTypography-root": {
        color: theme.palette.text.primary,
    },
    "& h1, & h2, & h3, & h4, & h5, & h6": {
        color: theme.palette.text.primary,
        fontWeight: 700,
        marginTop: theme.spacing(3),
        marginBottom: theme.spacing(1.5),
    },
    "& a": {
        color: theme.palette.primary.main,
        textDecoration: "none",
        fontWeight: 500,
        '&:hover': {
            textDecoration: "underline",
        },
    },
    "& code": {
        backgroundColor: theme.palette.mode === "dark" ? "rgba(255,255,255,0.07)" : "rgba(0,0,0,0.07)",
        padding: "2px 8px",
        borderRadius: "5px",
        fontSize: "0.95em",
    },
    "& blockquote": {
        borderLeft: `4px solid ${theme.palette.primary.main}`,
        paddingLeft: theme.spacing(2),
        marginLeft: 0,
        fontStyle: "italic",
        color: theme.palette.text.secondary,
        background: theme.palette.action.hover,
        borderRadius: theme.spacing(1),
    },
    [theme.breakpoints.down("md")]: {
        padding: theme.spacing(4),
    },
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(2),
        borderRadius: theme.spacing(1),
        boxShadow: theme.shadows[1],
    },
}));

const HeaderSection = styled(Box)(({ theme }) => ({
    textAlign: "center",
    marginBottom: theme.spacing(5),
    paddingTop: theme.spacing(3),
    position: "relative",
}));

const IconWrapper = styled(Box)(({ theme }) => ({
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 72,
    height: 72,
    borderRadius: "50%",
    background: `linear-gradient(135deg, ${theme.palette.primary.light} 0%, ${theme.palette.primary.main} 100%)`,
    boxShadow: theme.shadows[2],
    marginBottom: theme.spacing(2),
    border: `2px solid ${theme.palette.background.paper}`,
    "& svg": {
        fontSize: 38,
        color: theme.palette.primary.contrastText,
    },
}));

const HeaderDivider = styled("hr")(({ theme }) => ({
    border: 0,
    height: "2px",
    background: theme.palette.divider,
    margin: `${theme.spacing(3)} auto ${theme.spacing(2)} auto`,
    width: "60%",
    borderRadius: theme.spacing(1),
}));

const LastUpdatedText = styled(Typography)(({ theme }) => ({
    color: theme.palette.text.secondary,
    fontSize: "0.95rem",
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(1),
}));

const enhancedTabsSx = {
    mb: 3,
    ".MuiTabs-indicator": {
        height: 4,
        borderRadius: 2,
        backgroundColor: "primary.main",
    },
    ".MuiTab-root": {
        fontWeight: 600,
        fontSize: "1.1rem",
        textTransform: "none",
        letterSpacing: 0.2,
        minHeight: 48,
        '&.Mui-selected': {
            color: "primary.main",
        },
        '&:hover': {
            backgroundColor: "action.hover",
        },
    },
};

const FooterWrapper = styled(Box)(({ theme }) => ({
    borderTop: `1.5px solid ${theme.palette.divider}`,
    marginTop: theme.spacing(6),
    background: theme.palette.background.paper,
}));

export function PrivacyPolicyView(_props: ViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const theme = useTheme();
    const content = useMarkdown(privacyMarkdown, injectBusinessData);
    const { currTab, tabs } = useTabs({ id: "privacy-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Privacy, display: "Dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<TabParamBase<PolicyTabsInfo>>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <PageContainer contentType="text" size="fullSize">
            <ScrollBox>
                <Navbar title={t("Privacy")} />
                <StyledContainer maxWidth="md">
                    <Fade in timeout={800}>
                        <Box>
                            <HeaderSection>
                                <IconWrapper>
                                    <IconCommon name="Lock" decorative />
                                </IconWrapper>
                                <Typography
                                    variant="h2"
                                    component="h1"
                                    sx={{
                                        fontWeight: 800,
                                        color: theme.palette.text.primary,
                                        mb: 1,
                                        fontSize: { xs: "2.1rem", sm: "2.5rem" },
                                    }}
                                >
                                    {t("PrivacyPolicy")}
                                </Typography>
                                <Typography
                                    variant="body1"
                                    sx={{
                                        color: theme.palette.text.secondary,
                                        maxWidth: "600px",
                                        mx: "auto",
                                        fontWeight: 400,
                                        fontSize: { xs: "1rem", sm: "1.15rem" },
                                    }}
                                >
                                    {t("PrivacyPolicyDescription", { defaultValue: "We take your privacy seriously. This policy describes how we collect, use, and protect your information." })}
                                </Typography>
                                <LastUpdatedText>
                                    {t("LastUpdated", { defaultValue: "Last updated" })}: October 02, 2021
                                </LastUpdatedText>
                                <HeaderDivider />
                            </HeaderSection>
                            <PageTabs<typeof policyTabParams>
                                ariaLabel="privacy policy and terms tabs"
                                currTab={currTab}
                                fullWidth
                                onChange={handleTabChange}
                                tabs={tabs}
                                sx={enhancedTabsSx}
                            />
                            <ContentBox>
                                <MarkdownDisplay content={content} />
                            </ContentBox>
                        </Box>
                    </Fade>
                </StyledContainer>
                <FooterWrapper>
                    <Footer />
                </FooterWrapper>
            </ScrollBox>
        </PageContainer>
    );
}

export function TermsView(_props: ViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const theme = useTheme();
    const content = useMarkdown(privacyMarkdown, injectBusinessData);
    const { currTab, tabs } = useTabs({ id: "terms-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Terms, display: "Dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<TabParamBase<PolicyTabsInfo>>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <PageContainer contentType="text" size="fullSize">
            <ScrollBox minHeight="100vh">
                <Navbar title={t("Terms")} />
                <StyledContainer maxWidth="md">
                    <Fade in timeout={800}>
                        <Box>
                            <HeaderSection>
                                <IconWrapper>
                                    <IconCommon name="Article" decorative />
                                </IconWrapper>
                                <Typography
                                    variant="h2"
                                    component="h1"
                                    sx={{
                                        fontWeight: 800,
                                        color: theme.palette.text.primary,
                                        mb: 1,
                                        fontSize: { xs: "2.1rem", sm: "2.5rem" },
                                    }}
                                >
                                    {t("TermsOfService")}
                                </Typography>
                                <Typography
                                    variant="body1"
                                    sx={{
                                        color: theme.palette.text.secondary,
                                        maxWidth: "600px",
                                        mx: "auto",
                                        fontWeight: 400,
                                        fontSize: { xs: "1rem", sm: "1.15rem" },
                                    }}
                                >
                                    {t("TermsDescription", { defaultValue: "By using our services, you agree to these terms. Please read them carefully." })}
                                </Typography>
                                <LastUpdatedText>
                                    {t("LastUpdated", { defaultValue: "Last updated" })}: October 02, 2021
                                </LastUpdatedText>
                                <HeaderDivider />
                            </HeaderSection>
                            <PageTabs<typeof policyTabParams>
                                ariaLabel="privacy policy and terms tabs"
                                currTab={currTab}
                                fullWidth
                                onChange={handleTabChange}
                                tabs={tabs}
                                sx={enhancedTabsSx}
                            />
                            <ContentBox>
                                <MarkdownDisplay content={content} />
                            </ContentBox>
                        </Box>
                    </Fade>
                </StyledContainer>
                <FooterWrapper>
                    <Footer />
                </FooterWrapper>
            </ScrollBox>
        </PageContainer>
    );
}
