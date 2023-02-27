/**
 * Search page for personal objects (active runs, completed runs, views, bookmarks)
 */
import { Stack, Typography } from "@mui/material";
import { PageTabs, SearchList } from "components";
import { useCallback, useMemo, useState } from "react";
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { SearchType, HistorySearchPageTabOption as TabOptions } from "utils";
import { PageTab } from "components/types";
import { useTranslation } from "react-i18next";
import { CommonKey } from "@shared/translations";
import { HistorySearchViewProps } from "../types";

// Tab data type
type BaseParams = {
    searchType: SearchType;
    tabType: TabOptions;
    titleKey: CommonKey;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: TabOptions.Runs,
    titleKey: 'Run',
    where: {},
}, {
    searchType: SearchType.View,
    tabType: TabOptions.Viewed,
    titleKey: 'View',
    where: {},
}, {
    searchType: SearchType.Bookmark,
    tabType: TabOptions.Bookmarked,
    titleKey: 'Bookmark',
    where: {},
}]

export function HistorySearchView({
    session,
}: HistorySearchViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            label: t(tab.titleKey, { count: 2 }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        return tabs[Math.max(0, index)];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
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
            <PageTabs
                ariaLabel="history-search-tabs"
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{title}</Typography>
            </Stack>
            {searchType && <SearchList
                id="history-search-page-list"
                take={20}
                searchType={searchType}
                session={session}
                zIndex={200}
                where={where}
            />}
        </>
    )
}