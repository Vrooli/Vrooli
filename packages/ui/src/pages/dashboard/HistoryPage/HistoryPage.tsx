import { Box, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { historyPage, historyPageVariables } from 'graphql/generated/historyPage';
import { useQuery } from '@apollo/client';
import { historyPageQuery } from 'graphql/query';
import { AutocompleteSearchBar, ListTitleContainer } from 'components';
import { useLocation } from '@local/route';
import { APP_LINKS } from '@local/shared';
import { HistoryPageProps } from '../types';
import { listToAutocomplete, listToListItems, openObject, OpenObjectProps, useReactSearch } from 'utils';
import { AutocompleteOption } from 'types';
import { centeredDiv } from 'styles';

const activeRoutinesText = `Routines that you've started to execute, and have not finished.`;

const completedRoutinesText = `Routines that you've executed and completed`

const recentText = `Organizations, projects, routines, standards, and users that you've recently viewed.`;

const starredText = `Organizations, projects, routines, standards, and users that you've starred.`;

const tabOptions = [
    ['For You', APP_LINKS.Home],
    ['History', APP_LINKS.History],
];

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

    const { data, refetch, loading } = useQuery<historyPage, historyPageVariables>(historyPageQuery, { variables: { input: { searchString } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch]);

    // Handle tabs
    const tabIndex = useMemo(() => {
        if (window.location.pathname === APP_LINKS.History) return 1;
        return 0;
    }, []);
    const handleTabChange = (_e, newIndex) => {
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: OpenObjectProps['object'], event: any) => {
        event?.stopPropagation();
        // Navigate to item page
        openObject(item, setLocation);
    }, [setLocation]);

    const activeRuns = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Run'),
        items: data?.historyPage?.activeRuns,
        keyPrefix: 'active-runs-list-item',
        loading,
        session,
    }), [data?.historyPage?.activeRuns, loading, session])

    const completedRuns = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Run'),
        items: data?.historyPage?.completedRuns,
        keyPrefix: 'completed-runs-list-item',
        loading,
        session,
    }), [data?.historyPage?.completedRuns, loading, session])

    const recent = useMemo(() => listToListItems({
        dummyItems: ['Organization', 'Project', 'Routine', 'Standard', 'User'],
        items: data?.historyPage?.recentlyViewed,
        keyPrefix: 'recent-list-item',
        loading,
        onClick: toItemPage,
        session,
    }), [data?.historyPage?.recentlyViewed, loading, session, toItemPage])

    const starred = useMemo(() => listToListItems({
        dummyItems: ['Organization', 'Project', 'Routine', 'Standard', 'User'],
        items: data?.historyPage?.recentlyStarred,
        keyPrefix: 'starred-list-item',
        loading,
        onClick: toItemPage,
        session,
    }), [data?.historyPage?.recentlyStarred, loading, session, toItemPage])

    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const flattened = (Object.values(data?.historyPage ?? [])).reduce((acc, curr) => acc.concat(curr), []);
        return listToAutocomplete(flattened, languages);
    }, [data?.historyPage, languages]);

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

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
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
                        id="main-search"
                        placeholder='Search active/completed runs, stars, and views...'
                        options={autocompleteOptions}
                        loading={loading}
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                        session={session}
                        showSecondaryLabel={true}
                        sx={{ width: 'min(100%, 600px)' }}
                    />
                </Stack>
                {/* Search results */}
                <ListTitleContainer
                    title={"Active Routines"}
                    helpText={activeRoutinesText}
                    isEmpty={activeRuns.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {activeRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Completed Routines"}
                    helpText={completedRoutinesText}
                    isEmpty={completedRuns.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {completedRuns}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Recently Viewed"}
                    helpText={recentText}
                    isEmpty={recent.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Starred"}
                    helpText={starredText}
                    isEmpty={starred.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {starred}
                </ListTitleContainer>
            </Stack>
        </Box>
    )
}