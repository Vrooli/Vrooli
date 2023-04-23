import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BookmarkFor } from "@local/consts";
import { EditIcon, EllipsisIcon, SmartContractIcon } from "@local/icons";
import { Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { smartContractVersionFindOne } from "../../../api/generated/endpoints/smartContractVersion_findOne";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "../../../components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { placeholderColor } from "../../../utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const SmartContractView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const { id, isLoading, object: smartContractVersion, permissions, setObject: setSmartContractVersion } = useObjectFromUrl({
        query: smartContractVersionFindOne,
        partialData,
    });
    const availableLanguages = useMemo(() => (smartContractVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [smartContractVersion?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(smartContractVersion ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
        };
    }, [language, smartContractVersion, partialData]);
    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);
    const [moreMenuAnchor, setMoreMenuAnchor] = useState(null);
    const openMoreMenu = useCallback((ev) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const actionData = useObjectActions({
        object: smartContractVersion,
        objectType: "SmartContractVersion",
        setLocation,
        setObject: setSmartContractVersion,
    });
    const overviewComponent = useMemo(() => (_jsxs(Box, { position: "relative", ml: 'auto', mr: 'auto', mt: 3, bgcolor: palette.background.paper, sx: {
            borderRadius: { xs: "0", sm: 2 },
            boxShadow: { xs: "none", sm: 12 },
            width: { xs: "100%", sm: "min(500px, 100vw)" },
        }, children: [_jsx(Box, { width: "min(100px, 25vw)", height: "min(100px, 25vw)", borderRadius: '100%', position: 'absolute', display: 'flex', justifyContent: 'center', alignItems: 'center', left: '50%', top: "-55px", sx: {
                    border: "1px solid black",
                    backgroundColor: profileColors[0],
                    transform: "translateX(-50%)",
                }, children: _jsx(SmartContractIcon, { fill: profileColors[1], width: '80%', height: '80%' }) }), _jsx(Tooltip, { title: "See all options", children: _jsx(IconButton, { "aria-label": "More", size: "small", onClick: openMoreMenu, sx: {
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }, children: _jsx(EllipsisIcon, { fill: palette.background.textSecondary }) }) }), _jsxs(Stack, { direction: "column", spacing: 1, p: 1, alignItems: "center", justifyContent: "center", children: [isLoading ? (_jsx(Stack, { sx: { width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }, spacing: 2, children: _jsx(LinearProgress, { color: "inherit" }) })) : permissions.canUpdate ? (_jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", children: [_jsx(Typography, { variant: "h4", textAlign: "center", children: name }), _jsx(Tooltip, { title: "Edit smartContractVersion", children: _jsx(IconButton, { "aria-label": "Edit smartContractVersion", size: "small", onClick: () => actionData.onActionStart("Edit"), children: _jsx(EditIcon, { fill: palette.secondary.main }) }) })] })) : (_jsx(Typography, { variant: "h4", textAlign: "center", children: name })), _jsx(DateDisplay, { loading: isLoading, showIcon: true, textBeforeDate: "Joined", timestamp: smartContractVersion?.created_at, width: "33%" }), isLoading ? (_jsxs(Stack, { sx: { width: "85%", color: "grey.500" }, spacing: 2, children: [_jsx(LinearProgress, { color: "inherit" }), _jsx(LinearProgress, { color: "inherit" })] })) : (_jsx(Typography, { variant: "body1", sx: { color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary }, children: description ?? "No description set" })), _jsxs(Stack, { direction: "row", spacing: 2, alignItems: "center", children: [_jsx(ShareButton, { object: smartContractVersion, zIndex: zIndex }), _jsx(ReportsLink, { object: smartContractVersion }), _jsx(BookmarkButton, { disabled: !permissions.canBookmark, objectId: smartContractVersion?.id ?? "", bookmarkFor: BookmarkFor.SmartContract, isBookmarked: smartContractVersion?.root?.you?.isBookmarked ?? false, bookmarks: smartContractVersion?.root?.bookmarks ?? 0, onChange: (isBookmarked) => { } })] })] })] })), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, smartContractVersion, description, zIndex, actionData]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "SmartContract",
                } }), _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: moreMenuAnchor, object: smartContractVersion, onClose: closeMoreMenu, zIndex: zIndex + 1 }), _jsxs(Box, { sx: {
                    background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                    display: "flex",
                    paddingTop: 5,
                    paddingBottom: { xs: 0, sm: 2, md: 5 },
                    position: "relative",
                }, children: [_jsx(Box, { sx: {
                            position: "absolute",
                            top: 8,
                            right: 8,
                        }, children: _jsx(SelectLanguageMenu, { currentLanguage: language, handleCurrent: setLanguage, languages: availableLanguages, zIndex: zIndex }) }), overviewComponent] })] }));
};
//# sourceMappingURL=SmartContractView.js.map