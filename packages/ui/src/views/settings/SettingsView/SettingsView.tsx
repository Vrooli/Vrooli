import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { SettingsSearchBar } from "components/inputs/search";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { ApiIcon, HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ObjectIcon, ProfileIcon, VisibleIcon, WalletIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ScrollBox } from "styles";
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
        title: "Data",
        description: "DataSettingsDescription",
        link: LINKS.SettingsData,
        Icon: ObjectIcon,
    },
    {
        title: "Authentication",
        description: "AuthenticationSettingsDescription",
        link: LINKS.SettingsAuthentication,
        Icon: LockIcon,
    },
    {
        title: "Payment",
        titleVariables: { count: 2 },
        description: "PaymentsSettingsDescription",
        link: LINKS.SettingsPayments,
        Icon: WalletIcon,
    },
    {
        title: "Api",
        description: "ApiSettingsDescription",
        link: LINKS.SettingsApi,
        Icon: ApiIcon,
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
        titleVariables: { count: 2 },
        description: "NotificationSettingsDescription",
        link: LINKS.SettingsNotifications,
        Icon: NotificationsCustomizedIcon,
    },
    {
        title: "FocusMode",
        titleVariables: { count: 2 },
        description: "FocusModeSettingsDescription",
        link: LINKS.SettingsFocusModes,
        Icon: HistoryIcon,
    },
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

    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue); }, []);
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(newValue?.value);
    }, [setLocation]);

    return (
        <ScrollBox>
            <TopBar
                below={<Box sx={{
                    width: "min(100%, 700px)",
                    margin: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    paddingLeft: 2,
                    paddingRight: 2,
                }}>
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
                    {accountSettingsData.map(({ title, titleVariables, description, link, Icon }, index) => (
                        <TIDCard
                            buttonText={t("Open")}
                            description={t(description)}
                            Icon={Icon}
                            key={index}
                            onClick={() => onSelect(link)}
                            title={t(title, { ...titleVariables, defaultValue: title })}
                        />
                    ))}
                </CardGrid>
                <Title
                    title={t("Display")}
                    sxs={{ text: { paddingTop: 2 } }}
                    variant="header"
                />
                <CardGrid minWidth={300}>
                    {displaySettingsData.map(({ title, titleVariables, description, link, Icon }, index) => (
                        <TIDCard
                            buttonText={t("Open")}
                            description={t(description)}
                            Icon={Icon}
                            key={index}
                            onClick={() => onSelect(link)}
                            title={t(title, { ...titleVariables, defaultValue: title })}
                        />
                    ))}
                </CardGrid>
            </Box>
        </ScrollBox>
    );
}
