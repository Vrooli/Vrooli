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
    paddingTop: theme.spacing(3),
    paddingBottom: theme.spacing(8),
}));

const ContentBox = styled(Box)(({ theme }) => ({
    padding: theme.spacing(6),
    borderRadius: theme.spacing(2),
    background: theme.palette.background.paper,
    boxShadow: theme.shadows[1],
    position: "relative",
    overflow: "hidden",
    "& .MuiTypography-root": {
        color: theme.palette.text.primary,
    },
    "& h1, & h2, & h3, & h4, & h5, & h6": {
        color: theme.palette.text.primary,
        fontWeight: 600,
    },
    "& a": {
        color: theme.palette.primary.main,
        textDecoration: "none",
        "&:hover": {
            textDecoration: "underline",
        },
    },
    "& code": {
        backgroundColor: theme.palette.mode === "dark" ? "rgba(255, 255, 255, 0.05)" : "rgba(0, 0, 0, 0.05)",
        padding: "2px 6px",
        borderRadius: "4px",
        fontSize: "0.875em",
    },
    "& blockquote": {
        borderLeft: `4px solid ${theme.palette.primary.main}`,
        paddingLeft: theme.spacing(2),
        marginLeft: 0,
        fontStyle: "italic",
        color: theme.palette.text.secondary,
    },
    [theme.breakpoints.down("md")]: {
        padding: theme.spacing(4),
    },
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(3),
        borderRadius: 0,
        boxShadow: "none",
    },
}));

const HeaderSection = styled(Box)(({ theme }) => ({
    textAlign: "center",
    marginBottom: theme.spacing(4),
    paddingTop: theme.spacing(2),
}));

const IconWrapper = styled(Box)(({ theme }) => ({
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 80,
    height: 80,
    borderRadius: "50%",
    backgroundColor: theme.palette.primary.main,
    marginBottom: theme.spacing(2),
    "& svg": {
        fontSize: 40,
        color: theme.palette.primary.contrastText,
    },
}));

const LastUpdatedText = styled(Typography)(({ theme }) => ({
    color: theme.palette.text.secondary,
    fontSize: "0.875rem",
    marginTop: theme.spacing(2),
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
                                    variant="h3"
                                    component="h1"
                                    sx={{
                                        fontWeight: 700,
                                        color: theme.palette.text.primary,
                                        mb: 1,
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
                                    }}
                                >
                                    {t("PrivacyPolicyDescription", { defaultValue: "We take your privacy seriously. This policy describes how we collect, use, and protect your information." })}
                                </Typography>
                                <LastUpdatedText>
                                    {t("LastUpdated", { defaultValue: "Last updated" })}: October 02, 2021
                                </LastUpdatedText>
                            </HeaderSection>
                            <PageTabs<typeof policyTabParams>
                                ariaLabel="privacy policy and terms tabs"
                                currTab={currTab}
                                fullWidth
                                onChange={handleTabChange}
                                tabs={tabs}
                                sx={{ mb: 3 }}
                            />
                            <ContentBox>
                                <MarkdownDisplay content={content} />
                            </ContentBox>
                        </Box>
                    </Fade>
                </StyledContainer>
                <Footer />
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
                                    variant="h3"
                                    component="h1"
                                    sx={{
                                        fontWeight: 700,
                                        color: theme.palette.text.primary,
                                        mb: 1,
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
                                    }}
                                >
                                    {t("TermsDescription", { defaultValue: "By using our services, you agree to these terms. Please read them carefully." })}
                                </Typography>
                                <LastUpdatedText>
                                    {t("LastUpdated", { defaultValue: "Last updated" })}: October 02, 2021
                                </LastUpdatedText>
                            </HeaderSection>
                            <PageTabs<typeof policyTabParams>
                                ariaLabel="privacy policy and terms tabs"
                                currTab={currTab}
                                fullWidth
                                onChange={handleTabChange}
                                tabs={tabs}
                                sx={{ mb: 3 }}
                            />
                            <ContentBox>
                                <MarkdownDisplay content={content} />
                            </ContentBox>
                        </Box>
                    </Fade>
                </StyledContainer>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}
