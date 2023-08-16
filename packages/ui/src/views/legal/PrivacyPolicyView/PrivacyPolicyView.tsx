import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import privacyMarkdown from "assets/policy/privacy.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { useMarkdown } from "utils/hooks/useMarkdown";
import { PageTab } from "utils/hooks/useTabs";
import { convertToDot, valueFromDot } from "utils/shape/general";
import { MarkdownDisplay } from "../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { PrivacyPolicyViewProps } from "../types";

const BUSINESS_DATA = {
    BUSINESS_NAME: "Vrooli",
    EMAIL: {
        Label: "info@vrooli.com",
        Link: "mailto:info@vrooli.com",
    },
    SUPPORT_EMAIL: {
        Label: "support@vrooli.com",
        Link: "mailto:support@vrooli.com",
    },
    SOCIALS: {
        Discord: "https://discord.gg/VyrDFzbmmF",
        GitHub: "https://github.com/MattHalloran/Vrooli",
        Twitter: "https://twitter.com/VrooliOfficial",
    },
    APP_URL: "https://vrooli.com",
};

enum TabOptions {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const PrivacyPolicyView = ({
    isOpen,
    onClose,
    zIndex,
}: PrivacyPolicyViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const privacy = useMarkdown(privacyMarkdown, (markdown) => {
        const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
        business_fields.forEach(f => markdown = markdown?.replaceAll(`<${f}>`, valueFromDot(BUSINESS_DATA, f) || "") ?? "");
        return markdown;
    });

    const tabs = useMemo<PageTab<TabOptions>[]>(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option as TabOptions),
        tabType: option as TabOptions,
    })), [t]);
    const currTab = useMemo(() => tabs[0], [tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(LINKS[tab.tabType], { replace: true });
    }, [setLocation]);

    return <>
        <TopBar
            display={display}
            onClose={onClose}
            title={t("Privacy")}
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
            <MarkdownDisplay content={privacy} zIndex={zIndex} />
        </Box>
    </>;
};
