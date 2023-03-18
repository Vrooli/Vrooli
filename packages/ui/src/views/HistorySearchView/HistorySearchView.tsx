import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { CommonKey } from "@shared/translations";
import { SearchList } from 'components/lists/SearchList/SearchList';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { PageTab } from "components/types";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HistorySearchPageTabOption, SearchType } from 'utils/search/objectToSearch';
import { HistorySearchViewProps } from "../types";

// Tab data type
type BaseParams = {
    searchType: SearchType;
    tabType: HistorySearchPageTabOption;
    titleKey: CommonKey;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistorySearchPageTabOption.Runs,
    titleKey: 'Run',
    where: {},
}, {
    searchType: SearchType.View,
    tabType: HistorySearchPageTabOption.Viewed,
    titleKey: 'View',
    where: {},
}, {
    searchType: SearchType.BookmarkList,
    tabType: HistorySearchPageTabOption.Bookmarked,
    titleKey: 'Bookmark',
    where: {},
}]

/**
 * Search page for personal objects (active runs, completed runs, views, bookmarks)
 */
export const HistorySearchView = ({
    display = 'page',
}: HistorySearchViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Handle tabs
    const tabs = useMemo<PageTab<HistorySearchPageTabOption>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            label: t(tab.titleKey, { count: 2 }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<HistorySearchPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        return tabs[Math.max(0, index)];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<HistorySearchPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab)
    }, [setLocation]);

    // On tab change, update BaseParams, document title, where, and URL
    const { searchType, title, where } = useMemo(() => {
        // Update tab title
        document.title = `${t(`Search`)} | ${currTab.label}`;
        return {
            ...tabParams[currTab.index],
            title: currTab.label,
        }
    }, [currTab.index, currTab.label, t]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    hideOnDesktop: true,
                    title,
                }}
                below={<PageTabs
                    ariaLabel="history-search-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="history-search-page-list"
                take={20}
                searchType={searchType}
                zIndex={200}
                where={where}
            />}
        </>
    )
}