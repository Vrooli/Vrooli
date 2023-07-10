import { LINKS, useLocation } from "@local/shared";
import { Box } from "@mui/material";
import termsMarkdown from "assets/policy/terms.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { PageTab } from "components/types";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useMarkdown } from "utils/hooks/useMarkdown";
import { TermsViewProps } from "../types";

enum TabOptions {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const TermsView = ({
    display = "page",
    onClose,
    zIndex,
}: TermsViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const terms = useMarkdown(termsMarkdown);

    const tabs = useMemo<PageTab<TabOptions>[]>(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option as TabOptions),
        value: option as TabOptions,
    })), [t]);
    const currTab = useMemo(() => tabs[1], [tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(LINKS[tab.value], { replace: true });
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
            zIndex={zIndex}
        />
        <Box m={2}>
            <MarkdownDisplay content={terms} zIndex={zIndex} />
        </Box>
    </>;
};
