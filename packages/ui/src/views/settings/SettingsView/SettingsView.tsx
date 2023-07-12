import { HistoryIcon, LightModeIcon, LINKS, LockIcon, NotificationsCustomizedIcon, ProfileIcon, useLocation, VisibleIcon } from "@local/shared";
import { Box } from "@mui/material";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { SettingsData, SettingsViewProps } from "../types";


export const accountSettingsData: SettingsData[] = [
    {
        title: "Profile",
        description: "ProfileSettingsDescription",
        link: LINKS.SettingsProfile,
        Icon: ProfileIcon,
    },
    {
        title: "Privacy",
        description: "PrivacySettingsDescription",
        link: LINKS.SettingsPrivacy,
        Icon: VisibleIcon,
    },
    {
        title: "Authentication",
        description: "AuthenticationSettingsDescription",
        link: LINKS.SettingsAuthentication,
        Icon: LockIcon,
    },
];

export const displaySettingsData: SettingsData[] = [
    {
        title: "Display",
        description: "DisplaySettingsDescription",
        link: LINKS.SettingsDisplay,
        Icon: LightModeIcon,
    },
    {
        title: "Notification",
        description: "NotificationSettingsDescription",
        link: LINKS.SettingsNotifications,
        Icon: NotificationsCustomizedIcon,
    },
    {
        title: "FocusMode",
        description: "FocusModeSettingsDescription",
        link: LINKS.SettingsFocusModes,
        Icon: HistoryIcon,
    },
];

export const SettingsView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Settings")}
                zIndex={zIndex}
            />
            <Box p={2}>
                <Title
                    title={t("Account")}
                    variant="header"
                    zIndex={zIndex}
                />
                <CardGrid minWidth={300}>
                    {accountSettingsData.map(({ title, description, link, Icon }, index) => (
                        <TIDCard
                            buttonText={t("Open")}
                            description={t(description)}
                            Icon={Icon}
                            key={index}
                            onClick={() => onSelect(link)}
                            title={t(title, { count: 2 })}
                        />
                    ))}
                </CardGrid>
                <Title
                    title={t("Display")}
                    sxs={{ text: { paddingTop: 2 } }}
                    variant="header"
                    zIndex={zIndex}
                />
                <CardGrid minWidth={300} sx={{ paddingBottom: "64px" }}>
                    {displaySettingsData.map(({ title, description, link, Icon }, index) => (
                        <TIDCard
                            buttonText={t("Open")}
                            description={t(description)}
                            Icon={Icon}
                            key={index}
                            onClick={() => onSelect(link)}
                            title={t(title, { count: 2 })}
                        />
                    ))}
                </CardGrid>
            </Box>
        </>
    );
};
