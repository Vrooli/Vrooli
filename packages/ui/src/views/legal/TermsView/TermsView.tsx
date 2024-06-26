import { Box, useTheme } from "@mui/material";
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
import { convertToDot, valueFromDot } from "utils/shape/general";
import { BUSINESS_DATA } from "..";
import { PolicyTabOption } from "../PrivacyPolicyView/PrivacyPolicyView";
import { TermsViewProps } from "../types";

export const TermsView = ({
    display,
    onClose,
}: TermsViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const terms = useMarkdown(termsMarkdown, (markdown) => {
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
                below={<PageTabs
                    ariaLabel="privacy policy and terms tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Box p={2} sx={{ background: palette.background.paper }}>
                <MarkdownDisplay content={terms} />
            </Box>
        </>
    );
};
