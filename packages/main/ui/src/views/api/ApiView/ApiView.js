import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BookmarkFor } from "@local/consts";
import { ApiIcon, EditIcon, EllipsisIcon } from "@local/icons";
import { Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { apiVersionFindOne } from "../../../api/generated/endpoints/apiVersion_findOne";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "../../../components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "../../../components/lists/resource";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { placeholderColor } from "../../../utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const ApiView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const { id, isLoading, object: apiVersion, permissions, setObject: setApiVersion } = useObjectFromUrl({
        query: apiVersionFindOne,
        partialData,
    });
    const availableLanguages = useMemo(() => (apiVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [apiVersion?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { canBookmark, details, name, resourceList, summary } = useMemo(() => {
        const { canBookmark } = apiVersion?.root?.you ?? {};
        const resourceList = apiVersion?.resourceList;
        const { details, name, summary } = getTranslation(apiVersion ?? partialData, [language]);
        return {
            details,
            canBookmark,
            name,
            resourceList,
            summary,
        };
    }, [language, apiVersion, partialData]);
    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);
    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (_jsx(ResourceListVertical, { list: resourceList, canUpdate: permissions.canUpdate, handleUpdate: (updatedList) => {
            if (!apiVersion)
                return;
            setApiVersion({
                ...apiVersion,
                resourceList: updatedList,
            });
        }, loading: isLoading, mutate: true, zIndex: zIndex })) : null, [resourceList, permissions.canUpdate, isLoading, zIndex, apiVersion, setApiVersion]);
    const [moreMenuAnchor, setMoreMenuAnchor] = useState(null);
    const openMoreMenu = useCallback((ev) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const actionData = useObjectActions({
        object: apiVersion,
        objectType: "ApiVersion",
        setLocation,
        setObject: setApiVersion,
    });
    const overviewComponent = useMemo(() => (_jsxs(Box, { position: "relative", ml: 'auto', mr: 'auto', mt: 3, bgcolor: palette.background.paper, sx: {
            borderRadius: { xs: "0", sm: 2 },
            boxShadow: { xs: "none", sm: 12 },
            width: { xs: "100%", sm: "min(500px, 100vw)" },
        }, children: [_jsx(Box, { width: "min(100px, 25vw)", height: "min(100px, 25vw)", borderRadius: '100%', position: 'absolute', display: 'flex', justifyContent: 'center', alignItems: 'center', left: '50%', top: "-55px", sx: {
                    border: "1px solid black",
                    backgroundColor: profileColors[0],
                    transform: "translateX(-50%)",
                }, children: _jsx(ApiIcon, { fill: profileColors[1], width: '80%', height: '80%' }) }), _jsx(Tooltip, { title: "See all options", children: _jsx(IconButton, { "aria-label": "More", size: "small", onClick: openMoreMenu, sx: {
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }, children: _jsx(EllipsisIcon, { fill: palette.background.textSecondary }) }) }), _jsxs(Stack, { direction: "column", spacing: 1, p: 1, alignItems: "center", justifyContent: "center", children: [isLoading ? (_jsx(Stack, { sx: { width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }, spacing: 2, children: _jsx(LinearProgress, { color: "inherit" }) })) : permissions.canUpdate ? (_jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", children: [_jsx(Typography, { variant: "h4", textAlign: "center", children: name }), _jsx(Tooltip, { title: "Edit apiVersion", children: _jsx(IconButton, { "aria-label": "Edit apiVersion", size: "small", onClick: () => actionData.onActionStart("Edit"), children: _jsx(EditIcon, { fill: palette.secondary.main }) }) })] })) : (_jsx(Typography, { variant: "h4", textAlign: "center", children: name })), _jsx(DateDisplay, { loading: isLoading, showIcon: true, textBeforeDate: "Joined", timestamp: apiVersion?.created_at, width: "33%" }), isLoading ? (_jsxs(Stack, { sx: { width: "85%", color: "grey.500" }, spacing: 2, children: [_jsx(LinearProgress, { color: "inherit" }), _jsx(LinearProgress, { color: "inherit" })] })) : (_jsx(Typography, { variant: "body1", sx: { color: Boolean(summary) ? palette.background.textPrimary : palette.background.textSecondary }, children: summary ?? "No summary set" })), _jsxs(Stack, { direction: "row", spacing: 2, alignItems: "center", children: [_jsx(ShareButton, { object: apiVersion, zIndex: zIndex }), _jsx(ReportsLink, { object: apiVersion }), _jsx(BookmarkButton, { disabled: !canBookmark, objectId: apiVersion?.id ?? "", bookmarkFor: BookmarkFor.Api, isBookmarked: apiVersion?.root?.you?.isBookmarked ?? false, bookmarks: apiVersion?.root?.bookmarks ?? 0, onChange: (isBookmarked) => { } })] })] })] })), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, isLoading, permissions.canUpdate, name, apiVersion, summary, zIndex, canBookmark, actionData]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Api",
                } }), _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: moreMenuAnchor, object: apiVersion, onClose: closeMoreMenu, zIndex: zIndex + 1 }), _jsxs(Box, { sx: {
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
//# sourceMappingURL=ApiView.js.map