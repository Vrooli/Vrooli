import { BookmarkFilledIcon, RoutineActiveIcon, RoutineCompleteIcon, SvgProps, VisibleIcon } from '@shared/icons';
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';
import { SearchList } from 'components/lists/SearchList/SearchList';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { PageTab } from 'components/types';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { HistoryPageTabOption, SearchType } from 'utils/search/objectToSearch';
import { HistoryViewProps } from '../types';


// Tab data type
type BaseParams = {
    Icon: (props: SvgProps) => JSX.Element,
    titleKey: CommonKey;
    searchType: SearchType;
    tabType: HistoryPageTabOption;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    Icon: VisibleIcon,
    titleKey: 'View',
    searchType: SearchType.View,
    tabType: HistoryPageTabOption.Viewed,
    where: {},
}, {
    Icon: BookmarkFilledIcon,
    titleKey: 'Bookmark',
    searchType: SearchType.BookmarkList,
    tabType: HistoryPageTabOption.Bookmarked,
    where: {},
}, {
    Icon: RoutineActiveIcon,
    titleKey: 'Active',
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsActive,
    where: {},
}, {
    Icon: RoutineCompleteIcon,
    titleKey: 'Complete',
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsCompleted,
    where: {},
}];

/**
 * Shows items you've bookmarked, viewed, or run recently.
 */
export const HistoryView = ({
    display = 'page',
}: HistoryViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Handle tabs
    const tabs = useMemo<PageTab<HistoryPageTabOption>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<HistoryPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to bookmarked tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<HistoryPageTabOption>) => {
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
                    ariaLabel="history-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="history-page-list"
                take={20}
                searchType={searchType}
                zIndex={200}
                sxs={{
                    search: {
                        marginTop: 2,
                    }
                }}
                where={where}
            />}
        </>
    )
}