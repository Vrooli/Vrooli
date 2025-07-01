// AI_CHECK: TYPE_SAFETY=replaced-2-any-types-with-proper-types | LAST: 2025-06-28
import { Box } from "@mui/material";
import { LINKS } from "@vrooli/shared";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { SettingsSearchBar } from "../../components/inputs/search/SettingsSearchBar.js";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid.js";
import { TIDCard } from "../../components/lists/TIDCard/TIDCard.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { Title } from "../../components/text/Title.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { type ViewProps } from "../../types.js";
import { type SearchItem } from "../../utils/search/siteToSearch.js";
import { type SettingsData } from "./types.js";

export const accountSettingsData: SettingsData[] = [
    {
        title: "Profile",
        description: "ProfileSettingsDescription",
        link: LINKS.SettingsProfile,
        iconInfo: { name: "Profile", type: "Common" },
    },
    {
        title: "Privacy",
        description: "PrivacySettingsDescription",
        link: LINKS.SettingsPrivacy,
        iconInfo: { name: "Visible", type: "Common" },
    },
    {
        title: "Data",
        description: "DataSettingsDescription",
        link: LINKS.SettingsData,
        iconInfo: { name: "Object", type: "Common" },
    },
    {
        title: "Authentication",
        description: "AuthenticationSettingsDescription",
        link: LINKS.SettingsAuthentication,
        iconInfo: { name: "Lock", type: "Common" },
    },
    {
        title: "Payment",
        titleVariables: { count: 2 },
        description: "PaymentsSettingsDescription",
        link: LINKS.SettingsPayments,
        iconInfo: { name: "Wallet", type: "Common" },
    },
    {
        title: "Api",
        description: "ApiSettingsDescription",
        link: LINKS.SettingsApi,
        iconInfo: { name: "Api", type: "Common" },
    },
];

export const displaySettingsData: SettingsData[] = [
    {
        title: "Display",
        description: "DisplaySettingsDescription",
        link: LINKS.SettingsDisplay,
        iconInfo: { name: "LightMode", type: "Common" },
    },
    {
        title: "Notification",
        titleVariables: { count: 2 },
        description: "NotificationSettingsDescription",
        link: LINKS.SettingsNotifications,
        iconInfo: { name: "NotificationsCustomized", type: "Common" },
    },
];

export function SettingsView(_props: ViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    const [searchString, setSearchString] = useState<string>("");

    const updateSearch = useCallback((newValue: string) => {
        setSearchString(newValue);
    }, []);
    const onInputSelect = useCallback((newValue: SearchItem) => {
        if (!newValue) return;
        setLocation(newValue.value);
    }, [setLocation]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("Settings")} />
                <Box
                    margin="auto"
                    marginTop={2}
                    marginBottom={2}
                    maxWidth={800}
                    paddingLeft={2}
                    paddingRight={2}
                >
                    <SettingsSearchBar
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                    />
                </Box>
                <Box maxWidth={800} margin="auto">
                    <Title
                        title={t("Account")}
                        variant="header"
                    />
                    <CardGrid minWidth={300}>
                        {accountSettingsData.map(({ title, titleVariables, description, iconInfo, link }, index) => {
                            function handleClick() {
                                onSelect(link);
                            }

                            return (
                                <TIDCard
                                    buttonText={t("Open")}
                                    description={t(description)}
                                    iconInfo={iconInfo}
                                    key={index}
                                    onClick={handleClick}
                                    title={t(title, { ...titleVariables, defaultValue: title })}
                                />
                            );
                        })}
                    </CardGrid>
                    <Title
                        title={t("Display")}
                        sxs={{ text: { paddingTop: 2 } }}
                        variant="header"
                    />
                    <CardGrid minWidth={300}>
                        {displaySettingsData.map(({ title, titleVariables, description, iconInfo, link }, index) => {
                            function handleClick() {
                                onSelect(link);
                            }

                            return (
                                <TIDCard
                                    buttonText={t("Open")}
                                    description={t(description)}
                                    iconInfo={iconInfo}
                                    key={index}
                                    onClick={handleClick}
                                    title={t(title, { ...titleVariables, defaultValue: title })}
                                />
                            );
                        })}
                    </CardGrid>
                </Box>
            </ScrollBox>
        </PageContainer>
    );
}
