import Box from "@mui/material/Box";
import { styled } from "@mui/material";
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

const ContentBox = styled(Box)(({ theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    padding: theme.spacing(4),
    borderRadius: "12px",
    overflow: "overlay",
    minHeight: "100vh",
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    marginTop: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        marginTop: 0,
        borderRadius: 0,
    },
}));

export function PrivacyPolicyView(_props: ViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

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
                <PageTabs<typeof policyTabParams>
                    ariaLabel="privacy policy and terms tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                <ContentBox>
                    <MarkdownDisplay content={content} />
                </ContentBox>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}

export function TermsView(_props: ViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

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
                <PageTabs<typeof policyTabParams>
                    ariaLabel="privacy policy and terms tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                <ContentBox>
                    <MarkdownDisplay content={content} />
                </ContentBox>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}
