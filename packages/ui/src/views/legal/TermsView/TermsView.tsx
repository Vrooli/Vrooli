import { Box } from "@mui/material";
import termsMarkdown from "assets/policy/terms.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useMarkdown } from "hooks/useMarkdown";
import { PageTab, useTabs } from "hooks/useTabs";
import { ChangeEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { policyTabParams } from "utils/search/objectToSearch";
import { PolicyTabOption } from "../PrivacyPolicyView/PrivacyPolicyView";
import { TermsViewProps } from "../types";

export const TermsView = ({
    isOpen,
    onClose,
}: TermsViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const terms = useMarkdown(termsMarkdown);

    const { currTab, tabs } = useTabs<PolicyTabOption, false>({ id: "terms-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Terms, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<PolicyTabOption, false>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return <>
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
        <Box m={2}>
            <MarkdownDisplay content={terms} />
        </Box>
    </>;
};
