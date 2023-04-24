import { LINKS } from "@local/consts";
import { HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, VisibleIcon } from "@local/icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CardGrid } from "../../../components/lists/CardGrid/CardGrid";
import { TIDCard } from "../../../components/lists/TIDCard/TIDCard";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { Header } from "../../../components/text/Header/Header";
import { useLocation } from "../../../utils/route";
import { SettingsData, SettingsViewProps } from "../types";


export const accountSettingsData: SettingsData[] = [
    {
        title: "Profile",
        description: "ProfileSettingsDescription",
        link: "SettingsProfile",
        Icon: ProfileIcon,
    },
    {
        title: "Privacy",
        description: "PrivacySettingsDescription",
        link: "SettingsPrivacy",
        Icon: VisibleIcon,
    },
    {
        title: "Authentication",
        description: "AuthenticationSettingsDescription",
        link: "SettingsAuthentication",
        Icon: LockIcon,
    },
];

export const displaySettingsData: SettingsData[] = [
    {
        title: "Display",
        description: "DisplaySettingsDescription",
        link: "SettingsDisplay",
        Icon: LightModeIcon,
    },
    {
        title: "Notification",
        description: "NotificationSettingsDescription",
        link: "SettingsNotifications",
        Icon: NotificationsCustomizedIcon,
    },
    {
        title: "FocusMode",
        description: "FocusModeSettingsDescription",
        link: "SettingsFocusModes",
        Icon: HistoryIcon,
    },
];

export const SettingsView = ({
    display = "page",
}: SettingsViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(LINKS[newValue]);
    }, [setLocation]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "Settings",
                }}
            />
            <Header title={t("Account")} />
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
            <Header title={t("Display")} sxs={{ text: { paddingTop: 2 } }} />
            <CardGrid minWidth={300}>
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
        </>
    );
};