import { CommonKey, LINKS } from "@local/shared";
import { Box } from "@mui/material";
import privacyMarkdown from "assets/policy/privacy.md";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { ChangeEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { useMarkdown } from "utils/hooks/useMarkdown";
import { PageTab, useTabs } from "utils/hooks/useTabs";
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

export enum PolicyTabOption {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const policyTabParams = [
    {
        titleKey: "Privacy" as CommonKey,
        href: LINKS.Privacy,
        tabType: PolicyTabOption.Privacy,
    }, {
        titleKey: "Terms" as CommonKey,
        href: LINKS.Terms,
        tabType: PolicyTabOption.Terms,
    },
];

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

    const { currTab, tabs } = useTabs<PolicyTabOption, false>({ tabParams: policyTabParams, defaultTab: 0, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<PolicyTabOption, false>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
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
