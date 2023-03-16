import { useQuery } from '@apollo/client';
import { Stack } from '@mui/material';
import { HistoryInput, HistoryResult, LINKS, RunStatus } from '@shared/consts';
import { useLocation } from '@shared/route';
import { historyHistory } from 'api/generated/endpoints/history_history';
import { ListTitleContainer, PageTabs, SiteSearchBar, TopBar } from 'components';
import { PageTab } from 'components/types';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { centeredDiv } from 'styles';
import { AutocompleteOption, Wrap } from 'types';
import { getUserLanguages, HistorySearchPageTabOption, listToAutocomplete, listToListItems, openObject, useReactSearch } from 'utils';
import { HistoryViewProps } from '../types';

enum TabOptions {
    ForYou = "ForYou",
    History = "History",
}

const zIndex = 200;

/**
 * Shows items you've bookmarked, viewed, or run recently.
 */
export const HistoryView = ({
    display = 'page',
    session
}: HistoryViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [searchString, setSearchString] = useState<string>('');
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);

    const { data, refetch, loading } = useQuery<Wrap<HistoryResult, 'history'>, Wrap<HistoryInput, 'input'>>(historyHistory, { variables: { input: { searchString } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch]);


    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => ([{
        index: 0,
        href: LINKS.Home,
        label: t('ForYou'),
        value: TabOptions.ForYou,
    }, {
        index: 1,
        href: LINKS.History,
        label: t('History'),
        value: TabOptions.History,
    }]), [t]);
    const currTab = useMemo(() => tabs[1], [tabs])
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(tab.href!, { replace: true });
    }, [setLocation]);

    const activeRuns = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Run'),
        items: data?.history?.activeRuns,
        keyPrefix: 'active-runs-list-item',
        loading,
        session,
        zIndex,
    }), [data?.history?.activeRuns, loading, session])

    const completedRuns = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Run'),
        items: data?.history?.completedRuns,
        keyPrefix: 'completed-runs-list-item',
        loading,
        session,
        zIndex,
    }), [data?.history?.completedRuns, loading, session])

    const recent = useMemo(() => listToListItems({
        dummyItems: ['Organization', 'Project', 'Routine', 'Standard', 'User'],
        items: data?.history?.recentlyViewed,
        keyPrefix: 'recent-list-item',
        loading,
        session,
        zIndex,
    }), [data?.history?.recentlyViewed, loading, session])

    const bookmarked = useMemo(() => listToListItems({
        dummyItems: ['Organization', 'Project', 'Routine', 'Standard', 'User'],
        items: data?.history?.recentlyBookmarked,
        keyPrefix: 'bookmarked-list-item',
        loading,
        session,
        zIndex,
    }), [data?.history?.recentlyBookmarked, loading, session])

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const flattened = (Object.values(data?.history ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        return listToAutocomplete(flattened, languages);
    }, [data?.history, languages]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Replace current state with search string, so that search is not lost. 
        if (searchString) setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(LINKS.Home, { replace: true });
        // Navigate to item page
        openObject(newValue, setLocation);
    }, [searchString, setLocation]);

    const toSeeAllActiveRuns = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(LINKS.HistorySearch, {
            searchParams: {
                type: HistorySearchPageTabOption.Runs,
                status: RunStatus.InProgress
            }
        });
    }, [setLocation]);

    const toSeeAllCompletedRuns = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(LINKS.HistorySearch, {
            searchParams: {
                type: HistorySearchPageTabOption.Runs,
                status: RunStatus.Completed
            }
        });
    }, [setLocation]);

    const toSeeAllViewed = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(LINKS.HistorySearch, { searchParams: { type: HistorySearchPageTabOption.Viewed } });
    }, [setLocation]);

    const toSeeAllBookmarked = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(LINKS.HistorySearch, { searchParams: { type: HistorySearchPageTabOption.Bookmarked } });
    }, [setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
                titleData={{
                    titleKey: 'History',
                }}
                below={<PageTabs
                    ariaLabel="history-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Prompt stack */}
                <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '5vh' }}>
                    <SiteSearchBar
                        id="history-search"
                        placeholder='SearchHistory'
                        options={autocompleteOptions}
                        loading={loading}
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                        session={session}
                        showSecondaryLabel={true}
                        sxs={{ root: { width: 'min(100%, 600px)', paddingLeft: 2, paddingRight: 2 } }}
                    />
                </Stack>
                {/* Search results */}
                <ListTitleContainer
                    titleKey="RunsActive"
                    helpKey="RunsActiveHelp"
                    isEmpty={activeRuns.length === 0}
                    onClick={toSeeAllActiveRuns}
                    options={[['SeeAll', toSeeAllActiveRuns]]}
                >
                    {activeRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    titleKey="RunsCompleted"
                    helpKey="RunsCompletedHelp"
                    isEmpty={completedRuns.length === 0}
                    onClick={toSeeAllCompletedRuns}
                    options={[['SeeAll', toSeeAllCompletedRuns]]}
                >
                    {completedRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    titleKey="RecentlyViewed"
                    helpKey="RecentlyViewedHelp"
                    isEmpty={recent.length === 0}
                    onClick={toSeeAllViewed}
                    options={[['SeeAll', toSeeAllViewed]]}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    titleKey="Bookmarked"
                    helpKey="BookmarkedHelp"
                    isEmpty={bookmarked.length === 0}
                    onClick={toSeeAllBookmarked}
                    options={[['SeeAll', toSeeAllBookmarked]]}
                >
                    {bookmarked}
                </ListTitleContainer>
            </Stack>
        </>
    )
}