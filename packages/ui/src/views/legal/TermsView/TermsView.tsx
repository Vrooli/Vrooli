import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import termsMarkdown from "assets/policy/terms.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { ChangeEvent, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { useMarkdown } from "utils/hooks/useMarkdown";
import { PageTab } from "utils/hooks/useTabs";
import { TermsViewProps } from "../types";

enum TabOptions {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const TermsView = ({
    isOpen,
    onClose,
    zIndex,
}: TermsViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const terms = useMarkdown(termsMarkdown);

    const tabs = useMemo<PageTab<TabOptions>[]>(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option as TabOptions),
        tabType: option as TabOptions,
    })), [t]);
    const currTab = useMemo(() => tabs[1], [tabs]);
    const handleTabChange = useCallback((e: ChangeEvent<unknown>, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(LINKS[tab.tabType], { replace: true });
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
