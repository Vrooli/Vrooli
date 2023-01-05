import { Box, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { AutocompleteSearchBar, ListTitleContainer, PageContainer } from 'components';
import { useLocation } from '@shared/route';
import { APP_LINKS, HistoryInput, HistoryResult, RunStatus } from '@shared/consts';
import { HistoryPageProps } from '../types';
import { getUserLanguages, HistorySearchPageTabOption, listToAutocomplete, listToListItems, openObject, stringifySearchParams, useReactSearch } from 'utils';
import { AutocompleteOption, Wrap } from 'types';
import { centeredDiv } from 'styles';
import { historyEndpoint } from 'graphql/endpoints';

const activeRoutinesText = `Routines that you've started to execute, and have not finished.`;

const completedRoutinesText = `Routines that you've executed and completed`

const recentText = `Organizations, projects, routines, standards, and users that you've recently viewed.`;

const starredText = `Organizations, projects, routines, standards, and users that you've starred.`;

const tabOptions = [
    ['For You', APP_LINKS.Home],
    ['History', APP_LINKS.History],
];

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

    const [searchString, setSearchString] = useState<string>('');
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);

    const { data, refetch, loading } = useQuery<Wrap<HistoryResult, 'history'>, Wrap<HistoryInput, 'input'>>(historyEndpoint.history[0], { variables: { input: { searchString } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch]);

    // Handle tabs
    const tabIndex = useMemo(() => {
        if (window.location.pathname === APP_LINKS.History) return 1;
        return 0;
    }, []);
    const handleTabChange = (_e, newIndex) => {
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

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

    const starred = useMemo(() => listToListItems({
        dummyItems: ['Organization', 'Project', 'Routine', 'Standard', 'User'],
        items: data?.history?.recentlyStarred,
        keyPrefix: 'starred-list-item',
        loading,
        session,
        zIndex,
    }), [data?.history?.recentlyStarred, loading, session])

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const flattened = (Object.values(data?.history ?? [])).reduce((acc, curr) => acc.concat(curr), []);
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

    const toSeeAllStarred = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.HistorySearch}${stringifySearchParams({
            type: HistorySearchPageTabOption.Starred,
        })}`);
    }, [setLocation]);

    return (
        <PageContainer>
            {/* Navigate between normal home page (shows popular results) and history page (shows personalized results) */}
            <Box display="flex" justifyContent="center" width="100%">
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="home-pages"
                    sx={{
                        marginBottom: 1,
                    }}
                >
                    {tabOptions.map((option, index) => (
                        <Tab
                            key={index}
                            id={`for-you-tab-${index}`}
                            {...{
                                'aria-labelledby': `home-pages`,
                                'aria-label': `home page ${option[0]}`,
                            }}
                            label={option[0]}
                            color={index === 0 ? '#ce6c12' : 'default'}
                        />
                    ))}
                </Tabs>
            </Box>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Prompt stack */}
                <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '5vh' }}>
                    <Typography component="h1" variant="h3" textAlign="center">History</Typography>
                    <AutocompleteSearchBar
                        id="history-search"
                        placeholder='Search active/completed runs, stars, and views...'
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
                    title={"Active Routines"}
                    helpText={activeRoutinesText}
                    isEmpty={activeRuns.length === 0}
                    onClick={toSeeAllActiveRuns}
                    options={[['See all', toSeeAllActiveRuns]]}
                >
                    {activeRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Completed Routines"}
                    helpText={completedRoutinesText}
                    isEmpty={completedRuns.length === 0}
                    onClick={toSeeAllCompletedRuns}
                    options={[['See all', toSeeAllCompletedRuns]]}
                >
                    {completedRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Recently Viewed"}
                    helpText={recentText}
                    isEmpty={recent.length === 0}
                    onClick={toSeeAllViewed}
                    options={[['See all', toSeeAllViewed]]}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Starred"}
                    helpText={starredText}
                    isEmpty={starred.length === 0}
                    onClick={toSeeAllStarred}
                    options={[['See all', toSeeAllStarred]]}
                >
                    {starred}
                </ListTitleContainer>
            </Stack>
        </PageContainer>
    )
}