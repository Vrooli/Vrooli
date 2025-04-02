import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { SettingsSearchBar } from "../../components/inputs/search/SettingsSearchBar.js";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid.js";
import { TIDCard } from "../../components/lists/TIDCard/TIDCard.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { Title } from "../../components/text/Title.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { SettingsData, SettingsViewProps } from "./types.js";

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
    // {
    //     title: "FocusMode",
    //     titleVariables: { count: 2 },
    //     description: "FocusModeSettingsDescription",
    //     link: LINKS.SettingsFocusModes,
    //     iconInfo: { name: "History", type: "Common" },
    // },
];

export function SettingsView({
    display,
    onClose,
}: SettingsViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    const [searchString, setSearchString] = useState<string>("");

    const updateSearch = useCallback((newValue: any) => {
        setSearchString(newValue);
    }, []);
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(newValue?.value);
    }, [setLocation]);

    return (
        <ScrollBox>
            <TopBar
                below={<Box
                    width="min(100%, 700px)"
                    margin="auto"
                    marginTop={2}
                    marginBottom={2}
                    paddingLeft={2}
                    paddingRight={2}
                >
                    <SettingsSearchBar
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                    />
                </Box>}
                display={display}
                onClose={onClose}
                title={t("Settings")}
                titleBehaviorDesktop="ShowIn"
            />
            <Box>
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
    );
}
