import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { AddIcon, ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "@local/icons";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SearchList } from "../../../components/lists/SearchList/SearchList";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { centeredDiv } from "../../../styles";
import { getCurrentUser } from "../../../utils/authentication/session";
import { getObjectUrlBase } from "../../../utils/navigation/openObject";
import { PubSub } from "../../../utils/pubsub";
import { addSearchParams, parseSearchParams, useLocation } from "../../../utils/route";
import { SearchPageTabOption, SearchType } from "../../../utils/search/objectToSearch";
import { SessionContext } from "../../../utils/SessionContext";
const tabParams = [{
        Icon: RoutineIcon,
        searchType: SearchType.Routine,
        tabType: SearchPageTabOption.Routines,
        where: (userId) => ({ isInternal: false, ownedByUserId: userId }),
    }, {
        Icon: ProjectIcon,
        searchType: SearchType.Project,
        tabType: SearchPageTabOption.Projects,
        where: (userId) => ({ ownedByUserId: userId }),
    }, {
        Icon: HelpIcon,
        searchType: SearchType.Question,
        tabType: SearchPageTabOption.Questions,
        where: (userId) => ({ createdById: userId }),
    }, {
        Icon: NoteIcon,
        searchType: SearchType.Note,
        tabType: SearchPageTabOption.Notes,
        where: (userId) => ({ ownedByUserId: userId }),
    }, {
        Icon: OrganizationIcon,
        searchType: SearchType.Organization,
        tabType: SearchPageTabOption.Organizations,
        where: (userId) => ({ memberUserIds: [userId] }),
    }, {
        Icon: StandardIcon,
        searchType: SearchType.Standard,
        tabType: SearchPageTabOption.Standards,
        where: (userId) => ({ ownedByUserId: userId }),
    }, {
        Icon: ApiIcon,
        searchType: SearchType.Api,
        tabType: SearchPageTabOption.Apis,
        where: (userId) => ({ ownedByUserId: userId }),
    }, {
        Icon: SmartContractIcon,
        searchType: SearchType.SmartContract,
        tabType: SearchPageTabOption.SmartContracts,
        where: (userId) => ({ ownedByUserId: userId }),
    }];
export const MyStuffView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id: userId, apisCount, membershipsCount, questionsAskedCount, smartContractsCount, standardsCount, } = useMemo(() => getCurrentUser(session), [session]);
    const [popupButton, setPopupButton] = useState(false);
    const tabs = useMemo(() => {
        const filteredTabParams = tabParams.filter(tab => {
            switch (tab.tabType) {
                case SearchPageTabOption.Apis:
                    return Boolean(apisCount);
                case SearchPageTabOption.Organizations:
                    return Boolean(membershipsCount);
                case SearchPageTabOption.Questions:
                    return Boolean(questionsAskedCount);
                case SearchPageTabOption.SmartContracts:
                    return Boolean(smartContractsCount);
                case SearchPageTabOption.Standards:
                    return Boolean(standardsCount);
            }
            return true;
        });
        return filteredTabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            searchType: tab.searchType,
            tabType: tab.tabType,
            value: tab.tabType,
            where: tab.where,
        }));
    }, [apisCount, membershipsCount, questionsAskedCount, smartContractsCount, standardsCount, t]);
    const [currTab, setCurrTab] = useState(() => {
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.tabType === searchParams.type);
        if (index === -1)
            return tabs[0];
        return tabs[index];
    });
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [setLocation]);
    const { searchType, where } = useMemo(() => {
        document.title = `${t("Search")} | ${currTab.label}`;
        const params = tabs[currTab.index];
        return {
            ...params,
            where: params.where(userId),
        };
    }, [currTab.index, currTab.label, t, tabs, userId]);
    const onAddClick = useCallback((ev) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType })}/add`;
        if (!userId) {
            PubSub.get().publishSnack({ messageKey: "MustBeLoggedIn", severity: "Error" });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        if (searchType === SearchType.Routine) {
            setLocation(`${LINKS.Routine}/add`);
        }
        else if (searchType === SearchType.User) {
            setLocation(`${LINKS.Start}`);
        }
        else {
            setLocation(addUrl);
        }
    }, [searchType, setLocation, userId]);
    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (_jsx(Box, { sx: { ...centeredDiv, paddingTop: 1 }, children: _jsx(Tooltip, { title: t("AddTooltip"), children: _jsx(Button, { onClick: onAddClick, size: "large", sx: {
                    zIndex: 100,
                    minWidth: "min(100%, 200px)",
                    height: "48px",
                    borderRadius: 3,
                    position: "fixed",
                    bottom: "calc(5em + env(safe-area-inset-bottom))",
                    transform: popupButton ? "translateY(0)" : "translateY(calc(10em + env(safe-area-inset-bottom)))",
                    transition: "transform 1s ease-in-out",
                }, children: t("Add") }) }) })), [onAddClick, popupButton, t]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    hideOnDesktop: true,
                    titleKey: "MyStuff",
                }, below: _jsx(PageTabs, { ariaLabel: "search-tabs", currTab: currTab, onChange: handleTabChange, tabs: tabs }) }), _jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", sx: { paddingTop: 2 }, children: [_jsx(Typography, { component: "h2", variant: "h4", children: t(searchType, { count: 2, defaultValue: searchType }) }), _jsx(Tooltip, { title: "Add new", placement: "top", children: _jsx(IconButton, { size: "medium", onClick: onAddClick, sx: {
                                padding: 1,
                            }, children: _jsx(AddIcon, { fill: palette.secondary.main, width: '1.5em', height: '1.5em' }) }) })] }), searchType && _jsx(SearchList, { id: "main-search-page-list", take: 20, searchType: searchType, onScrolledFar: handleScrolledFar, zIndex: 200, where: where }), popupButtonContainer] }));
};
//# sourceMappingURL=MyStuffView.js.map