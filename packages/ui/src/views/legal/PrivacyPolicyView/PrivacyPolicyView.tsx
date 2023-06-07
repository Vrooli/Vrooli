import { LINKS, useLocation } from "@local/shared";
import { Box } from "@mui/material";
import privacyMarkdown from "assets/policy/privacy.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useMarkdown } from "utils/hooks/useMarkdown";
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
    display = "page",
    onClose,
}: PrivacyPolicyViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const privacy = useMarkdown(privacyMarkdown, (markdown) => {
        const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
        business_fields.forEach(f => markdown = markdown?.replaceAll(`<${f}>`, valueFromDot(BUSINESS_DATA, f) || "") ?? "");
        return markdown;
    });

    const tabs = useMemo<PageTab<TabOptions>[]>(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option as TabOptions),
        value: option as TabOptions,
    })), [t]);
    const currTab = useMemo(() => tabs[0], [tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(LINKS[tab.value], { replace: true });
    }, [setLocation]);

    return <>
        <TopBar
            display={display}
            onClose={onClose}
            titleData={{
                titleKey: "Privacy",
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
            <MarkdownDisplay content={privacy} />
        </Box>
    </>;
};
