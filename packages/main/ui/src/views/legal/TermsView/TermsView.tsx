import { LINKS } from ":local/consts";
import { Box } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import termsMarkdown from "../../../assets/policy/terms.md";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { PageTab } from "../../../components/types";
import { useMarkdown } from "../../../utils/hooks/useMarkdown";
import { useLocation } from "../../../utils/route";
import { TermsViewProps } from "../types";

enum TabOptions {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const TermsView = ({
    display = "page",
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
            onClose={() => { }}
            titleData={{
                titleKey: "Terms",
            }}
            below={<PageTabs
                ariaLabel="privacy policy and terms tabs"
                currTab={currTab}
                fullWidth
                onChange={handleTabChange}
                tabs={tabs}
            />}
        />
        <Box m={2}>
            <Markdown>{terms}</Markdown>
        </Box>
    </>;
};
