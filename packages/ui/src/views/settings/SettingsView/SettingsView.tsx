import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { SettingsSearchBar } from "components/inputs/search";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { ApiIcon, HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ObjectIcon, ProfileIcon, VisibleIcon, WalletIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { translateSearchItems } from "utils/search/siteToSearch";
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

/** Search bar options */
const searchItems: PreSearchItem[] = [
    {
        label: "Profile",
        keywords: ["Bio", "Handle", "Name"],
        value: "Profile",
    },
    {
        label: "Privacy",
        keywords: ["History"],
        value: "Privacy",
    },
    {
        label: "Authentication",
        keywords: [{ key: "Wallet", count: 1 }, { key: "Wallet", count: 2 }, { key: "Email", count: 1 }, { key: "Email", count: 2 }, "LogOut", "Security"],
        value: "authentication",
    },
    {
        label: "Display",
        keywords: ["Theme", "Light", "Dark", "Interests", "Hidden", { key: "Tag", count: 1 }, { key: "Tag", count: 2 }, "History"],
        value: "Display",
    },
    {
        label: "Notification",
        labelArgs: { count: 2 },
        keywords: [{ key: "Alert", count: 1 }, { key: "Alert", count: 2 }, { key: "PushNotification", count: 1 }, { key: "PushNotification", count: 2 }],
        value: "Notification",
    },
    {
        label: "Schedule",
        keywords: [],
        value: "Schedule",
    },
];

export const SettingsView = ({
    isOpen,
    onClose,
}: SettingsViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const onSelect = useCallback((link: LINKS) => {
        if (!link) return;
        setLocation(link);
    }, [setLocation]);

    const [searchString, setSearchString] = useState<string>("");

    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue); }, []);
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        setLocation(newValue);
    }, [setLocation]);

    const searchOptions = useMemo(() => translateSearchItems(searchItems, session), [session]);

    return (
        <>
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
                        options={searchOptions}
                    />
                </Box>}
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("Settings")}
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
                <CardGrid minWidth={300} sx={{ paddingBottom: "64px" }}>
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
        </>
    );
};
