import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { AddIcon, ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, UserIcon } from "@local/icons";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ShareSiteDialog } from "../../components/dialogs/ShareSiteDialog/ShareSiteDialog";
import { SearchList } from "../../components/lists/SearchList/SearchList";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../components/PageTabs/PageTabs";
import { centeredDiv } from "../../styles";
import { getCurrentUser } from "../../utils/authentication/session";
import { getObjectUrlBase } from "../../utils/navigation/openObject";
import { PubSub } from "../../utils/pubsub";
import { addSearchParams, parseSearchParams, useLocation } from "../../utils/route";
import { SearchPageTabOption, SearchType } from "../../utils/search/objectToSearch";
import { SessionContext } from "../../utils/SessionContext";
const tabParams = [{
        Icon: RoutineIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Routine,
        tabType: SearchPageTabOption.Routines,
        where: { isInternal: false },
    }, {
        Icon: ProjectIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Project,
        tabType: SearchPageTabOption.Projects,
        where: {},
    }, {
        Icon: HelpIcon,
        popupTitleKey: "Invite",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Question,
        tabType: SearchPageTabOption.Questions,
        where: {},
    }, {
        Icon: NoteIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Note,
        tabType: SearchPageTabOption.Notes,
        where: {},
    }, {
        Icon: OrganizationIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Organization,
        tabType: SearchPageTabOption.Organizations,
        where: {},
    }, {
        Icon: UserIcon,
        popupTitleKey: "Invite",
        popupTooltipKey: "InviteTooltip",
        searchType: SearchType.User,
        tabType: SearchPageTabOption.Users,
        where: {},
    }, {
        Icon: StandardIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Standard,
        tabType: SearchPageTabOption.Standards,
        where: {},
    }, {
        Icon: ApiIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.Api,
        tabType: SearchPageTabOption.Apis,
        where: {},
    }, {
        Icon: SmartContractIcon,
        popupTitleKey: "Add",
        popupTooltipKey: "AddTooltip",
        searchType: SearchType.SmartContract,
        tabType: SearchPageTabOption.SmartContracts,
        where: {},
    }];
export const SearchView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [popupButton, setPopupButton] = useState(false);
    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => setShareDialogOpen(false), []);
    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        if (index === -1)
            return tabs[0];
        return tabs[index];
    });
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [setLocation]);
    const { popupTitleKey, popupTooltipKey, searchType, where } = useMemo(() => {
        document.title = `${t("Search")} | ${currTab.label}`;
        return tabParams[currTab.index];
    }, [currTab.index, currTab.label, t]);
    const onAddClick = useCallback((ev) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType })}/add`;
        if (!getCurrentUser(session).id) {
            PubSub.get().publishSnack({ messageKey: "MustBeLoggedIn", severity: "Error" });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        if (searchType === SearchType.Routine) {
            setLocation(`${LINKS.Routine}/add`);
        }
        else if (searchType === SearchType.User) {
            setShareDialogOpen(true);
        }
        else {
            setLocation(addUrl);
        }
    }, [searchType, session, setLocation]);
    const onPopupButtonClick = useCallback((ev) => {
        if ([SearchPageTabOption.Users].includes(currTab.value)) {
            setShareDialogOpen(true);
        }
        else {
            onAddClick(ev);
        }
    }, [currTab.value, onAddClick]);
    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (_jsx(Box, { sx: { ...centeredDiv, paddingTop: 1 }, children: _jsx(Tooltip, { title: t(popupTooltipKey), children: _jsx(Button, { onClick: onPopupButtonClick, size: "large", sx: {
                    zIndex: 100,
                    minWidth: "min(100%, 200px)",
                    height: "48px",
                    borderRadius: 3,
                    position: "fixed",
                    bottom: "calc(5em + env(safe-area-inset-bottom))",
                    transform: popupButton ? "translateY(0)" : "translateY(calc(10em + env(safe-area-inset-bottom)))",
                    transition: "transform 1s ease-in-out",
                }, children: t(popupTitleKey) }) }) })), [onPopupButtonClick, popupButton, popupTitleKey, popupTooltipKey, t]);
    console.log("search typeeee", searchType, currTab);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    hideOnDesktop: true,
                    titleKey: "Search",
                }, below: _jsx(PageTabs, { ariaLabel: "search-tabs", currTab: currTab, onChange: handleTabChange, tabs: tabs }) }), _jsx(ShareSiteDialog, { onClose: closeShareDialog, open: shareDialogOpen, zIndex: 200 }), _jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", sx: { paddingTop: 2 }, children: [_jsx(Typography, { component: "h2", variant: "h4", children: t(searchType, { count: 2, defaultValue: searchType }) }), _jsx(Tooltip, { title: "Add new", placement: "top", children: _jsx(IconButton, { size: "medium", onClick: onAddClick, sx: {
                                padding: 1,
                            }, children: _jsx(AddIcon, { fill: palette.secondary.main, width: '1.5em', height: '1.5em' }) }) })] }), searchType && _jsx(SearchList, { id: "main-search-page-list", take: 20, searchType: searchType, onScrolledFar: handleScrolledFar, zIndex: 200, where: where }), popupButtonContainer] }));
};
//# sourceMappingURL=SearchView.js.map