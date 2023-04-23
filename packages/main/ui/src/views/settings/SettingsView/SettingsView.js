import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ProfileIcon, VisibleIcon } from "@local/icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CardGrid } from "../../../components/lists/CardGrid/CardGrid";
import { TIDCard } from "../../../components/lists/TIDCard/TIDCard";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { Header } from "../../../components/text/Header/Header";
import { useLocation } from "../../../utils/route";
export const accountSettingsData = [
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
export const displaySettingsData = [
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
export const SettingsView = ({ display = "page", }) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const onSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        setLocation(LINKS[newValue]);
    }, [setLocation]);
    return (_jsxs(_Fragment, { children: [_jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Settings",
                } }), _jsx(Header, { title: t("Account") }), _jsx(CardGrid, { minWidth: 300, children: accountSettingsData.map(({ title, description, link, Icon }, index) => (_jsx(TIDCard, { buttonText: t("Open"), description: t(description), Icon: Icon, onClick: () => onSelect(link), title: t(title, { count: 2 }) }, index))) }), _jsx(Header, { title: t("Display"), sxs: { text: { paddingTop: 2 } } }), _jsx(CardGrid, { minWidth: 300, children: displaySettingsData.map(({ title, description, link, Icon }, index) => (_jsx(TIDCard, { buttonText: t("Open"), description: t(description), Icon: Icon, onClick: () => onSelect(link), title: t(title, { count: 2 }) }, index))) })] }));
};
//# sourceMappingURL=SettingsView.js.map