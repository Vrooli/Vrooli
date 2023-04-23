import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { Box } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import termsMarkdown from "../../../assets/policy/terms.md";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { useMarkdown } from "../../../utils/hooks/useMarkdown";
import { useLocation } from "../../../utils/route";
var TabOptions;
(function (TabOptions) {
    TabOptions["Privacy"] = "Privacy";
    TabOptions["Terms"] = "Terms";
})(TabOptions || (TabOptions = {}));
export const TermsView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const terms = useMarkdown(termsMarkdown);
    const tabs = useMemo(() => Object.keys(TabOptions).map((option, index) => ({
        index,
        label: t(option),
        value: option,
    })), [t]);
    const currTab = useMemo(() => tabs[1], [tabs]);
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        setLocation(LINKS[tab.value], { replace: true });
    }, [setLocation]);
    return _jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Terms",
                }, below: _jsx(PageTabs, { ariaLabel: "privacy policy and terms tabs", currTab: currTab, fullWidth: true, onChange: handleTabChange, tabs: tabs }) }), _jsx(Box, { m: 2, children: _jsx(Markdown, { children: terms }) })] });
};
//# sourceMappingURL=TermsView.js.map