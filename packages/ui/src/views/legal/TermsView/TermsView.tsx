import { convertToDot, valueFromDot } from "@local/shared";
import { Box, styled } from "@mui/material";
import termsMarkdown from "assets/policy/terms.md";
import { PageTabs } from "components/PageTabs/PageTabs";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useMarkdown } from "hooks/useMarkdown";
import { PageTab, useTabs } from "hooks/useTabs";
import { ChangeEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { PolicyTabsInfo, policyTabParams } from "utils/search/objectToSearch";
import { BUSINESS_DATA } from "..";
import { PolicyTabOption } from "../PrivacyPolicyView/PrivacyPolicyView";
import { TermsViewProps } from "../types";

const OuterBox = styled(Box)(({ theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    padding: theme.spacing(4),
    borderRadius: "12px",
    overflow: "overlay",
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    marginTop: theme.spacing(2),
    [theme.breakpoints.down("sm")]: {
        marginTop: 0,
        borderRadius: 0,
    },
}));

export function TermsView({
    display,
    onClose,
}: TermsViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const content = useMarkdown(termsMarkdown, (markdown) => {
        const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
        business_fields.forEach(f => {
            const regex = new RegExp(`<${f}>`, "g");
            markdown = markdown?.replace(regex, valueFromDot(BUSINESS_DATA, f) || "") ?? "";
        });
        return markdown;
    });

    const { currTab, tabs } = useTabs({ id: "terms-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Terms, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<PolicyTabsInfo>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Terms")}
                titleBehaviorDesktop="Hide"
                titleBehaviorMobile="Hide"
                below={<PageTabs
                    ariaLabel="privacy policy and terms tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <OuterBox>
                <MarkdownDisplay content={content} />
            </OuterBox>
        </>
    );
}
