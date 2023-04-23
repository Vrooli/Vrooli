import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BookmarkFor } from "@local/consts";
import { EditIcon, EllipsisIcon } from "@local/icons";
import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { projectVersionFindOne } from "../../../api/generated/endpoints/projectVersion_findOne";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton/BookmarkButton";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const ProjectView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { id, isLoading, object: projectVersion, permissions, setObject: setProjectVersion } = useObjectFromUrl({
        query: projectVersionFindOne,
        partialData,
    });
    const availableLanguages = useMemo(() => (projectVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [projectVersion?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { name, description, handle } = useMemo(() => {
        const { description, name } = getTranslation(projectVersion ?? partialData, [language]);
        return {
            name,
            description,
            handle: projectVersion?.root?.handle ?? partialData?.root?.handle,
        };
    }, [language, projectVersion, partialData]);
    useEffect(() => {
        if (handle)
            document.title = `${name} ($${handle}) | Vrooli`;
        else
            document.title = `${name} | Vrooli`;
    }, [handle, name]);
    const [moreMenuAnchor, setMoreMenuAnchor] = useState(null);
    const openMoreMenu = useCallback((ev) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const actionData = useObjectActions({
        object: projectVersion,
        objectType: "ProjectVersion",
        setLocation,
        setObject: setProjectVersion,
    });
    const overviewComponent = useMemo(() => (_jsxs(Box, { position: "relative", ml: 'auto', mr: 'auto', mt: 3, bgcolor: palette.background.paper, sx: {
            borderRadius: { xs: "0", sm: 2 },
            boxShadow: { xs: "none", sm: 12 },
            width: { xs: "100%", sm: "min(500px, 100vw)" },
        }, children: [_jsx(Tooltip, { title: "See all options", children: _jsx(IconButton, { "aria-label": "More", size: "small", onClick: openMoreMenu, sx: {
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                        paddingRight: "1em",
                    }, children: _jsx(EllipsisIcon, { fill: palette.background.textSecondary }) }) }), _jsxs(Stack, { direction: "column", spacing: 1, p: 1, alignItems: "center", justifyContent: "center", children: [isLoading ? (_jsx(Stack, { sx: { width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }, spacing: 2, children: _jsx(LinearProgress, { color: "inherit" }) })) : permissions.canUpdate ? (_jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", children: [_jsx(Typography, { variant: "h4", textAlign: "center", children: name }), _jsx(Tooltip, { title: "Edit project", children: _jsx(IconButton, { "aria-label": "Edit project", size: "small", onClick: () => actionData.onActionStart("Edit"), children: _jsx(EditIcon, { fill: palette.secondary.main }) }) })] })) : (_jsx(Typography, { variant: "h4", textAlign: "center", children: name })), handle && _jsx(Link, { href: `https://handle.me/${handle}`, underline: "hover", children: _jsxs(Typography, { variant: "h6", textAlign: "center", sx: {
                                color: palette.secondary.dark,
                                cursor: "pointer",
                            }, children: ["$", handle] }) }), _jsx(DateDisplay, { loading: isLoading, showIcon: true, textBeforeDate: "Created", timestamp: projectVersion?.created_at, width: "33%" }), isLoading && (_jsxs(Stack, { sx: { width: "85%", color: "grey.500" }, spacing: 2, children: [_jsx(LinearProgress, { color: "inherit" }), _jsx(LinearProgress, { color: "inherit" })] })), !isLoading && Boolean(description) && _jsx(Typography, { variant: "body1", sx: { color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary }, children: description }), _jsxs(Stack, { direction: "row", spacing: 2, alignItems: "center", children: [_jsx(ShareButton, { object: projectVersion, zIndex: zIndex }), _jsx(BookmarkButton, { disabled: !permissions.canBookmark, objectId: projectVersion?.id ?? "", bookmarkFor: BookmarkFor.Project, isBookmarked: projectVersion?.root?.you?.isBookmarked ?? false, bookmarks: projectVersion?.root?.bookmarks ?? 0, onChange: (isBookmarked) => { } })] })] })] })), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, handle, projectVersion, description, zIndex, actionData]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Project",
                } }), _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: moreMenuAnchor, object: projectVersion, onClose: closeMoreMenu, zIndex: zIndex + 1 }), _jsxs(Box, { sx: {
                    display: "flex",
                    paddingTop: 5,
                    paddingBottom: { xs: 0, sm: 2, md: 5 },
                    background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                    position: "relative",
                }, children: [_jsx(Box, { sx: {
                            position: "absolute",
                            top: 8,
                            right: 8,
                            paddingRight: "1em",
                        }, children: _jsx(SelectLanguageMenu, { currentLanguage: language, handleCurrent: setLanguage, languages: availableLanguages, zIndex: zIndex }) }), overviewComponent] }), _jsx(Box, {})] }));
};
//# sourceMappingURL=ProjectView.js.map