import { Box } from "@mui/material";
import privacyMarkdown from "assets/policy/privacy.md";
import { PageTabs } from "components/PageTabs/PageTabs";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useMarkdown } from "hooks/useMarkdown";
import { PageTab, useTabs } from "hooks/useTabs";
import { ChangeEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { PolicyTabsInfo, policyTabParams } from "utils/search/objectToSearch";
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
        X: "https://x.com/VrooliOfficial",
    },
    APP_URL: "https://vrooli.com",
};

export enum PolicyTabOption {
    Privacy = "Privacy",
    Terms = "Terms",
}

export const PrivacyPolicyView = ({
    display,
    onClose,
}: PrivacyPolicyViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const privacy = useMarkdown(privacyMarkdown, (markdown) => {
        const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
        business_fields.forEach(f => {
            const regex = new RegExp(`<${f}>`, "g");
            markdown = markdown?.replace(regex, valueFromDot(BUSINESS_DATA, f) || "") ?? "";
        });
        return markdown;
    });


    const { currTab, tabs } = useTabs({ id: "privacy-tabs", tabParams: policyTabParams, defaultTab: PolicyTabOption.Privacy, display: "dialog" });
    const handleTabChange = useCallback((event: ChangeEvent<unknown>, tab: PageTab<PolicyTabsInfo>) => {
        event.preventDefault();
        setLocation(tab.href ?? "", { replace: true });
    }, [setLocation]);

    return (
        <>
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
            />
            <Box m={2}>
                <MarkdownDisplay content={privacy} />
            </Box>
        </>
    );
};
