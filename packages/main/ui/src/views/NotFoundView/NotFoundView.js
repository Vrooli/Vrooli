import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { Box, Button } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { Link } from "../../utils/route";
export const NotFoundView = () => {
    const { t } = useTranslation();
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: "page", onClose: () => { }, titleData: {
                    title: t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" }),
                } }), _jsxs(Box, { sx: {
                    position: "absolute",
                    top: "50%",
                    left: "50%",
                    transform: "translateX(-50%) translateY(-50%)",
                }, children: [_jsx("h1", { children: t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" }) }), _jsx("h3", { children: t("PageNotFoundDetails", { ns: "error", defaultValue: "PageNotFoundDetails" }) }), _jsx("br", {}), _jsx(Link, { to: LINKS.Home, children: _jsx(Button, { children: t("GoToHome") }) })] })] }));
};
//# sourceMappingURL=NotFoundView.js.map