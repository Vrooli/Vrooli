import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { BookmarkFilledIcon, RoutineActiveIcon, RoutineCompleteIcon, VisibleIcon } from "@local/icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SearchList } from "../../../components/lists/SearchList/SearchList";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { addSearchParams, parseSearchParams, useLocation } from "../../../utils/route";
import { HistoryPageTabOption, SearchType } from "../../../utils/search/objectToSearch";
const tabParams = [{
        Icon: VisibleIcon,
        titleKey: "View",
        searchType: SearchType.View,
        tabType: HistoryPageTabOption.Viewed,
        where: {},
    }, {
        Icon: BookmarkFilledIcon,
        titleKey: "Bookmark",
        searchType: SearchType.BookmarkList,
        tabType: HistoryPageTabOption.Bookmarked,
        where: {},
    }, {
        Icon: RoutineActiveIcon,
        titleKey: "Active",
        searchType: SearchType.RunProjectOrRunRoutine,
        tabType: HistoryPageTabOption.RunsActive,
        where: {},
    }, {
        Icon: RoutineCompleteIcon,
        titleKey: "Complete",
        searchType: SearchType.RunProjectOrRunRoutine,
        tabType: HistoryPageTabOption.RunsCompleted,
        where: {},
    }];
export const HistoryView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
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
    const { searchType, title, where } = useMemo(() => {
        document.title = `${t("Search")} | ${currTab.label}`;
        return {
            ...tabParams[currTab.index],
            title: currTab.label,
        };
    }, [currTab.index, currTab.label, t]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    hideOnDesktop: true,
                    title,
                }, below: _jsx(PageTabs, { ariaLabel: "history-tabs", currTab: currTab, fullWidth: true, onChange: handleTabChange, tabs: tabs }) }), searchType && _jsx(SearchList, { id: "history-page-list", take: 20, searchType: searchType, zIndex: 200, sxs: {
                    search: {
                        marginTop: 2,
                    },
                }, where: where })] }));
};
//# sourceMappingURL=HistoryView.js.map