import { Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { SiteSearchBar, ListTitleContainer, PageContainer, PageTabs } from 'components';
import { stringifySearchParams, useLocation } from '@shared/route';
import { APP_LINKS, HistoryInput, HistoryResult, RunStatus } from '@shared/consts';
import { HistoryPageProps } from '../types';
import { getUserLanguages, HistorySearchPageTabOption, listToAutocomplete, listToListItems, openObject, useReactSearch } from 'utils';
import { AutocompleteOption, Wrap } from 'types';
import { centeredDiv } from 'styles';
import { historyHistory } from 'api/generated/endpoints/history';
import { useTranslation } from 'react-i18next';
import { PageTab } from 'components/types';

enum TabOptions {
    ForYou = "ForYou",
    History = "History",
}

const activeRoutinesText = `Routines that you've started to execute, and have not finished.`;

const completedRoutinesText = `Routines that you've executed and completed`

const recentText = `Organizations, projects, routines, standards, and users that you've recently viewed.`;

const bookmarkedText = `Organizations, projects, routines, standards, and users that you've bookmarked.`;

const zIndex = 200;

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HistoryPage = ({
    session
}: HistoryPageProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

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
        href: APP_LINKS.Home,
        label: t('common:ForYou', { lng }),
        value: TabOptions.ForYou,
    }, {
        index: 1,
        href: APP_LINKS.History,
        label: t('common:History', { lng }),
        value: TabOptions.History,
    }]), [t, lng]);
    const currTab = useMemo(() => tabs[1], [tabs])
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(tab.href, { replace: true });
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
        if (searchString) setLocation(`${APP_LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(APP_LINKS.Home, { replace: true });
        // Navigate to item page
        openObject(newValue, setLocation);
    }, [searchString, setLocation]);

    const toSeeAllActiveRuns = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.HistorySearch}${stringifySearchParams({
            type: HistorySearchPageTabOption.Runs,
            status: RunStatus.InProgress
        })}`);
    }, [setLocation]);

    const toSeeAllCompletedRuns = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.HistorySearch}${stringifySearchParams({
            type: HistorySearchPageTabOption.Runs,
            status: RunStatus.Completed
        })}`);
    }, [setLocation]);

    const toSeeAllViewed = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.HistorySearch}${stringifySearchParams({
            type: HistorySearchPageTabOption.Viewed,
        })}`);
    }, [setLocation]);

    const toSeeAllBookmarked = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.HistorySearch}${stringifySearchParams({
            type: HistorySearchPageTabOption.Bookmarked,
        })}`);
    }, [setLocation]);

    return (
        <PageContainer>
            <PageTabs
                ariaLabel="history-tabs"
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Prompt stack */}
                <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '5vh' }}>
                    <Typography component="h1" variant="h3" textAlign="center">History</Typography>
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
                    title={t(`common:RunsActive`, { lng })}
                    helpText={activeRoutinesText}
                    isEmpty={activeRuns.length === 0}
                    onClick={toSeeAllActiveRuns}
                    options={[['See all', toSeeAllActiveRuns]]}
                    session={session}
                >
                    {activeRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={t(`common:RunsCompleted`, { lng })}
                    helpText={completedRoutinesText}
                    isEmpty={completedRuns.length === 0}
                    onClick={toSeeAllCompletedRuns}
                    options={[['See all', toSeeAllCompletedRuns]]}
                    session={session}
                >
                    {completedRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Recently Viewed"}
                    helpText={recentText}
                    isEmpty={recent.length === 0}
                    onClick={toSeeAllViewed}
                    options={[['See all', toSeeAllViewed]]}
                    session={session}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Bookmarked"}
                    helpText={bookmarkedText}
                    isEmpty={bookmarked.length === 0}
                    onClick={toSeeAllBookmarked}
                    options={[['See all', toSeeAllBookmarked]]}
                    session={session}
                >
                    {bookmarked}
                </ListTitleContainer>
            </Stack>
        </PageContainer>
    )
}