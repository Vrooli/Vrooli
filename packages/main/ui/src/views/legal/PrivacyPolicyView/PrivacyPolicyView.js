import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { Box } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import privacyMarkdown from "../../../assets/policy/privacy.md";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { useMarkdown } from "../../../utils/hooks/useMarkdown";
import { useLocation } from "../../../utils/route";
import { convertToDot, valueFromDot } from "../../../utils/shape/general";
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
var TabOptions;
(function (TabOptions) {
    TabOptions["Privacy"] = "Privacy";
    TabOptions["Terms"] = "Terms";
})(TabOptions || (TabOptions = {}));
export const PrivacyPolicyView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const privacy = useMarkdown(privacyMarkdown, (markdown) => {
        const business_fields = Object.keys(convertToDot(BUSINESS_DATA));
        business_fields.forEach(f => markdown = markdown?.replaceAll(`<${f}>`, valueFromDot(BUSINESS_DATA, f) || "") ?? "");
        return markdown;
    });
    const tabs = useMemo(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option),
        value: option,
    })), [t]);
    const currTab = useMemo(() => tabs[0], [tabs]);
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        setLocation(LINKS[tab.value], { replace: true });
    }, [setLocation]);
    return _jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Privacy",
                }, below: _jsx(PageTabs, { ariaLabel: "privacy policy and terms tabs", currTab: currTab, fullWidth: true, onChange: handleTabChange, tabs: tabs }) }), _jsx(Box, { m: 2, children: _jsx(Markdown, { children: privacy }) })] });
};
//# sourceMappingURL=PrivacyPolicyView.js.map