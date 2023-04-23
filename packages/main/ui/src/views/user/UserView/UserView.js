import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { BookmarkFor, LINKS, VisibilityType } from "@local/consts";
import { EditIcon, EllipsisIcon, HelpIcon, OrganizationIcon, ProjectIcon, UserIcon } from "@local/icons";
import { uuidValidate } from "@local/uuid";
import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useCustomLazyQuery } from "../../../api";
import { userFindOne } from "../../../api/generated/endpoints/user_findOne";
import { userProfile } from "../../../api/generated/endpoints/user_profile";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "../../../components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "../../../components/lists/resource";
import { SearchList } from "../../../components/lists/SearchList/SearchList";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { defaultYou, getYou, placeholderColor, toSearchListData } from "../../../utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useDisplayApolloError } from "../../../utils/hooks/useDisplayApolloError";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { PubSub } from "../../../utils/pubsub";
import { getLastUrlPart, useLocation } from "../../../utils/route";
import { SearchType } from "../../../utils/search/objectToSearch";
import { SessionContext } from "../../../utils/SessionContext";
var TabOptions;
(function (TabOptions) {
    TabOptions["Resource"] = "Resource";
    TabOptions["Project"] = "Project";
    TabOptions["Organization"] = "Organization";
})(TabOptions || (TabOptions = {}));
const tabParams = [{
        Icon: HelpIcon,
        searchType: SearchType.Resource,
        tabType: TabOptions.Resource,
        where: {},
    }, {
        Icon: ProjectIcon,
        searchType: SearchType.Project,
        tabType: TabOptions.Project,
        where: {},
    }, {
        Icon: OrganizationIcon,
        searchType: SearchType.Organization,
        tabType: TabOptions.Organization,
        where: {},
    }];
export const UserView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const [getUserData, { data: userData, error: userError, loading: isUserLoading }] = useCustomLazyQuery(userFindOne, { errorPolicy: "all" });
    const [getProfileData, { data: profileData, error: profileError, loading: isProfileLoading }] = useCustomLazyQuery(userProfile, { errorPolicy: "all" });
    const [user, setUser] = useState(null);
    useDisplayApolloError(userError ?? profileError);
    useEffect(() => {
        const urlEnding = getLastUrlPart();
        if (urlEnding && uuidValidate(urlEnding))
            getUserData({ variables: { id: urlEnding } });
        else if (typeof urlEnding === "string" && urlEnding.toLowerCase() === "profile")
            getProfileData();
        else
            PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getUserData, getProfileData]);
    useEffect(() => {
        setUser(userData ?? profileData ?? partialData);
    }, [userData, profileData, partialData]);
    const permissions = useMemo(() => user ? getYou(user) : defaultYou, [user]);
    const isLoading = useMemo(() => isUserLoading || isProfileLoading, [isUserLoading, isProfileLoading]);
    const availableLanguages = useMemo(() => (user?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [user?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { bio, name, handle, resourceList } = useMemo(() => {
        const resourceList = undefined;
        const { bio } = getTranslation(user ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            name: user?.name ?? partialData?.name,
            handle: user?.handle ?? partialData?.handle,
            resourceList,
        };
    }, [language, partialData, user]);
    useEffect(() => {
        if (handle)
            document.title = `${name} ($${handle}) | Vrooli`;
        else
            document.title = `${name} | Vrooli`;
    }, [handle, name]);
    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (_jsx(ResourceListVertical, { list: resourceList, canUpdate: permissions.canUpdate, handleUpdate: (updatedList) => {
            if (!user)
                return;
            setUser({
                ...user,
            });
        }, loading: isLoading, mutate: true, zIndex: zIndex })) : null, [isLoading, permissions.canUpdate, resourceList, setUser, user, zIndex]);
    const tabs = useMemo(() => {
        let tabs = tabParams;
        if (!resources && !permissions.canUpdate)
            tabs = tabs.filter(t => t.tabType !== TabOptions.Resource);
        return tabs.map((tab, i) => ({
            color: tab.tabType === TabOptions.Resource ? "#8e6b00" : palette.secondary.dark,
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [palette.secondary.dark, permissions.canUpdate, resources, t]);
    const [currTab, setCurrTab] = useState(tabs[0]);
    const handleTabChange = useCallback((_, value) => setCurrTab(value), []);
    const onEdit = useCallback(() => {
        setLocation(LINKS.SettingsProfile);
    }, [setLocation]);
    const { searchType, placeholder, where } = useMemo(() => {
        if (currTab.value === TabOptions.Organization)
            return toSearchListData("Organization", "SearchOrganization", { memberUserIds: [user?.id], visibility: VisibilityType.All });
        return toSearchListData("Project", "SearchProject", { ownedByUserId: user?.id, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
    }, [currTab.value, user?.id, permissions.canUpdate]);
    const [moreMenuAnchor, setMoreMenuAnchor] = useState(null);
    const openMoreMenu = useCallback((ev) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const actionData = useObjectActions({
        object: user,
        objectType: "User",
        setLocation,
        setObject: setUser,
    });
    const overviewComponent = useMemo(() => (_jsxs(Box, { position: "relative", ml: 'auto', mr: 'auto', mt: 3, bgcolor: palette.background.paper, sx: {
            borderRadius: { xs: "0", sm: 2 },
            boxShadow: { xs: "none", sm: 12 },
            width: { xs: "100%", sm: "min(500px, 100vw)" },
        }, children: [_jsx(Box, { width: "min(100px, 25vw)", height: "min(100px, 25vw)", borderRadius: '100%', border: `4px solid ${palette.primary.dark}`, position: 'absolute', display: 'flex', justifyContent: 'center', alignItems: 'center', left: '50%', top: "-55px", sx: {
                    border: "1px solid black",
                    backgroundColor: profileColors[0],
                    transform: "translateX(-50%)",
                }, children: _jsx(UserIcon, { fill: profileColors[1], width: '80%', height: '80%' }) }), _jsx(Tooltip, { title: "See all options", children: _jsx(IconButton, { "aria-label": "More", size: "small", onClick: openMoreMenu, sx: {
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                        paddingRight: "1em",
                    }, children: _jsx(EllipsisIcon, { fill: palette.background.textSecondary }) }) }), _jsxs(Stack, { direction: "column", spacing: 1, p: 1, alignItems: "center", justifyContent: "center", children: [isLoading ? (_jsx(Stack, { sx: { width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }, spacing: 2, children: _jsx(LinearProgress, { color: "inherit" }) })) : permissions.canUpdate ? (_jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", children: [_jsx(Typography, { variant: "h4", textAlign: "center", children: name }), _jsx(Tooltip, { title: "Edit profile", children: _jsx(IconButton, { "aria-label": "Edit profile", size: "small", onClick: onEdit, children: _jsx(EditIcon, { fill: palette.secondary.main }) }) })] })) : (_jsx(Typography, { variant: "h4", textAlign: "center", children: name })), handle && _jsx(Link, { href: `https://handle.me/${handle}`, underline: "hover", children: _jsxs(Typography, { variant: "h6", textAlign: "center", sx: {
                                color: palette.secondary.dark,
                                cursor: "pointer",
                            }, children: ["$", handle] }) }), _jsx(DateDisplay, { loading: isLoading, showIcon: true, textBeforeDate: "Joined", timestamp: user?.created_at, width: "33%" }), isLoading ? (_jsxs(Stack, { sx: { width: "85%", color: "grey.500" }, spacing: 2, children: [_jsx(LinearProgress, { color: "inherit" }), _jsx(LinearProgress, { color: "inherit" })] })) : (_jsx(Typography, { variant: "body1", sx: { color: Boolean(bio) ? palette.background.textPrimary : palette.background.textSecondary }, children: bio ?? "No bio set" })), _jsxs(Stack, { direction: "row", spacing: 2, alignItems: "center", children: [_jsx(ShareButton, { object: user, zIndex: zIndex }), _jsx(BookmarkButton, { disabled: permissions.canUpdate, objectId: user?.id ?? "", bookmarkFor: BookmarkFor.User, isBookmarked: user?.you?.isBookmarked ?? false, bookmarks: user?.bookmarks ?? 0, onChange: (isBookmarked) => { } }), _jsx(ReportsLink, { object: user ? { ...user, reportsCount: user.reportsReceivedCount } : undefined })] })] })] })), [bio, handle, permissions.canUpdate, isLoading, name, onEdit, openMoreMenu, palette.background.paper, palette.background.textPrimary, palette.background.textSecondary, palette.primary.dark, palette.secondary.dark, palette.secondary.main, profileColors, user, zIndex]);
    const toAddNew = useCallback((event) => {
        setLocation(`${LINKS[currTab.value]}/add`);
    }, [currTab.value, setLocation]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "User",
                } }), _jsx(ObjectActionMenu, { actionData: actionData, anchorEl: moreMenuAnchor, object: user, onClose: closeMoreMenu, zIndex: zIndex + 1 }), _jsxs(Box, { sx: {
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
                        }, children: _jsx(SelectLanguageMenu, { currentLanguage: language, handleCurrent: setLanguage, languages: availableLanguages, zIndex: zIndex }) }), overviewComponent] }), _jsxs(Box, { children: [_jsx(PageTabs, { ariaLabel: "user-tabs", currTab: currTab, onChange: handleTabChange, tabs: tabs }), _jsx(Box, { p: 2, children: currTab.value === TabOptions.Resource ? resources : (_jsx(SearchList, { canSearch: Boolean(user?.id) && uuidValidate(user?.id), handleAdd: permissions.canUpdate ? toAddNew : undefined, hideUpdateButton: true, id: "user-view-list", searchType: searchType, searchPlaceholder: placeholder, take: 20, where: where, zIndex: zIndex })) })] })] }));
};
//# sourceMappingURL=UserView.js.map