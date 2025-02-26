import { convertToDot, valueFromDot } from "@local/shared";
import { Box, styled } from "@mui/material";
import privacyMarkdown from "assets/policy/privacy.md";
import { PageContainer } from "components/Page/Page.js";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Footer } from "components/navigation/Footer/Footer.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useMarkdown } from "hooks/useMarkdown";
import { PageTab, useTabs } from "hooks/useTabs";
import { ChangeEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ScrollBox } from "styles";
import { BUSINESS_DATA } from "utils/consts.js";
import { PolicyTabsInfo, TabParamBase, policyTabParams } from "utils/search/objectToSearch";
import { MarkdownDisplay } from "../../components/text/MarkdownDisplay/MarkdownDisplay";
import { PrivacyPolicyViewProps, TermsViewProps } from "../types.js";

export enum PolicyTabOption {
    Privacy = "Privacy",
    Terms = "Terms",
}

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

export function PrivacyPolicyView({
    display,
    onClose,
}: PrivacyPolicyViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const content = useMarkdown(privacyMarkdown, injectBusinessData);

    const { currTab, tabs } = useTabs({ id: "privacy-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Privacy, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<TabParamBase<PolicyTabsInfo>>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <PageContainer contentType="text" size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={t("Privacy")}
                    titleBehaviorDesktop="Hide"
                    titleBehaviorMobile="Hide"
                    below={<PageTabs<typeof policyTabParams>
                        ariaLabel="privacy policy and terms tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                <ContentBox>
                    <MarkdownDisplay content={content} />
                </ContentBox>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}

export function TermsView({
    display,
    onClose,
}: TermsViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const content = useMarkdown(privacyMarkdown, injectBusinessData);

    const { currTab, tabs } = useTabs({ id: "terms-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Terms, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<TabParamBase<PolicyTabsInfo>>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <PageContainer contentType="text" size="fullSize">
            <ScrollBox minHeight="100vh">
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={t("Terms")}
                    titleBehaviorDesktop="Hide"
                    titleBehaviorMobile="Hide"
                    below={<PageTabs<typeof policyTabParams>
                        ariaLabel="privacy policy and terms tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                <ContentBox>
                    <MarkdownDisplay content={content} />
                </ContentBox>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}
