import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS, WHITE_PAPER_URL } from "@local/consts";
import { ArticleIcon, LearnIcon, PlayIcon, ProfileIcon } from "@local/icons";
import { Box, Button, Link, Stack, Typography, useTheme } from "@mui/material";
import { useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { clickSize } from "../../styles";
import { checkIfLoggedIn } from "../../utils/authentication/session";
import { PubSub } from "../../utils/pubsub";
import { openLink, useLocation } from "../../utils/route";
import { SessionContext } from "../../utils/SessionContext";
const buttonProps = {
    height: "48px",
    background: "white",
    color: "black",
    borderRadius: "10px",
    width: "20em",
    display: "flex",
    marginBottom: "5px",
    transition: "0.3s ease-in-out",
    "&:hover": {
        filter: "brightness(120%)",
        border: "1px solid white",
    },
};
export const WelcomeView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    useEffect(() => {
        const firstTime = localStorage.getItem("firstTime");
        if (firstTime === null) {
            PubSub.get().publishCelebration();
            localStorage.setItem("firstTime", "false");
        }
    }, []);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "WelcomeToVrooli",
                } }), _jsx(Box, { sx: {
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    textAlign: "center",
                    marginTop: 4,
                }, children: _jsxs(Box, { children: [_jsx(Typography, { component: "h2", variant: "h4", mb: 3, children: t("NotSureWhereToStart") }), _jsxs(Stack, { direction: "column", spacing: 1, mb: 2, sx: { alignItems: "center" }, children: [_jsx(Button, { onClick: () => setLocation(LINKS.Tutorial), startIcon: _jsx(LearnIcon, { fill: "black" }), sx: { ...buttonProps, marginBottom: 0 }, children: t("Tutorial") }), _jsx(Button, { onClick: () => setLocation(LINKS.Example), startIcon: _jsx(PlayIcon, { fill: "black" }), sx: { ...buttonProps, marginBottom: 0 }, children: t("RunExample") }), checkIfLoggedIn(session) && _jsx(Button, { onClick: () => setLocation(LINKS.SettingsProfile), startIcon: _jsx(ProfileIcon, { fill: "black" }), sx: { ...buttonProps, marginBottom: 0 }, children: t("SetUpProfile") }), _jsx(Button, { onClick: () => openLink(setLocation, WHITE_PAPER_URL), startIcon: _jsx(ArticleIcon, { fill: "black" }), sx: { ...buttonProps, marginBottom: 0 }, children: t("ReadWhitePaper") })] }), _jsx(Box, { sx: {
                                ...clickSize,
                                justifyContent: "end",
                            }, children: _jsx(Link, { onClick: () => setLocation(LINKS.Home), sx: {
                                    cursor: "pointer",
                                    "&:hover": {
                                        brightness: "120%",
                                    },
                                }, children: _jsx(Typography, { sx: { marginRight: 2, color: palette.secondary.dark }, children: t("IKnowWhatImDoing") }) }) })] }) })] }));
};
//# sourceMappingURL=WelcomeView.js.map