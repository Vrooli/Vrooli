import { Box, Stack, Tab, Tabs, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv } from 'styles';
import { forYouPage, forYouPageVariables } from 'graphql/generated/forYouPage';
import { useQuery } from '@apollo/client';
import { forYouPageQuery } from 'graphql/query';
import { AutocompleteSearchBar, TitleContainer, CreateNewDialog } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { ForYouPageProps } from '../types';
import { parseSearchParams } from 'utils/urlTools';
import { AutocompleteListItem, listToAutocomplete, listToListItems } from 'utils';
import _ from 'lodash';

const activeRoutinesText = `Routines that you've started to execute, and have not finished.`;

const completedRoutinesText = `Routines that you've executed and completed`

const recentText = `Organizations, projects, routines, standards, and users that you've recently viewed.`;

const starredText = `Organizations, projects, routines, standards, and users that you've starred.`;

const tabOptions = [
    ['Popular', APP_LINKS.Home],
    ['For You', APP_LINKS.ForYou],
];

const ObjectType = {
    Organization: 'Organization',
    Project: 'Project',
    Routine: 'Routine',
    Standard: 'Standard',
    User: 'User',
}

const linkMap: { [x: string]: [string, string] } = {
    [ObjectType.Organization]: [APP_LINKS.SearchOrganizations, APP_LINKS.Organization],
    [ObjectType.Project]: [APP_LINKS.SearchProjects, APP_LINKS.Project],
    [ObjectType.Routine]: [APP_LINKS.SearchRoutines, APP_LINKS.Run],
    [ObjectType.Standard]: [APP_LINKS.SearchStandards, APP_LINKS.Standard],
    [ObjectType.User]: [APP_LINKS.SearchUsers, APP_LINKS.Profile],
}

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const ForYouPage = ({
    session
}: ForYouPageProps) => {
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>(() => {
        const { search } = parseSearchParams(window.location.search);
        return search ?? '';
    });
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const { data, refetch, loading } = useQuery<forYouPage, forYouPageVariables>(forYouPageQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } } });
    useEffect(() => { refetch() }, [refetch, searchString]);

    // Handle tabs
    const tabIndex = useMemo(() => {
        if (window.location.pathname === APP_LINKS.ForYou) return 1;
        return 0;
    }, []);
    const handleTabChange = (_e, newIndex) => {
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

    const autocompleteOptions: AutocompleteListItem[] = useMemo(() => {
        return listToAutocomplete(_.flatten(Object.values(data?.forYouPage ?? [])), languages).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
    }, [languages, data]);

    const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
    const openCreateDialog = useCallback(() => { setCreateDialogOpen(true) }, [setCreateDialogOpen]);
    const closeCreateDialog = useCallback(() => { setCreateDialogOpen(false) }, [setCreateDialogOpen]);
    const openAdvancedSearch = useCallback(() => {
        //TODO
    }, [setLocation]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteListItem) => {
        if (!newValue) return;
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.ForYou}?search=${searchString}`, { replace: true });
        // Determine object from selected label
        const selectedItem = autocompleteOptions.find(o => o.id === newValue.id);
        if (!selectedItem) return;
        const linkBases = linkMap[selectedItem.objectType];
        setLocation(selectedItem.id ? `${linkBases[1]}/${selectedItem.id}` : linkBases[0]);
    }, [autocompleteOptions]);

    // Opens correct search page
    const openSearch = useCallback((event: any, linkBases: [string, string], id?: string) => {
        event?.stopPropagation();
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.ForYou}?search=${searchString}`, { replace: true });
        setLocation(id ? `${linkBases[1]}/${id}` : linkBases[0]);
    }, [searchString, setLocation]);

    const activeRoutines = useMemo(() => listToListItems(
        data?.forYouPage?.activeRoutines ?? [],
        session,
        'active-routines-list-item',
        () => {}
    ), [data, session])

    const completedRoutines = useMemo(() => listToListItems(
        data?.forYouPage?.completedRoutines ?? [],
        session,
        'completed-routines-list-item',
        () => {}
    ), [data, session])

    const recent = useMemo(() => listToListItems(
        data?.forYouPage?.recent ?? [],
        session,
        'recently-viewed-list-item',
        () => {}
    ), [data, session])

    const starred = useMemo(() => listToListItems(
        data?.forYouPage?.starred ?? [],
        session,
        'starred-list-item',
        () => {}
    ), [data, session])

    return (
        <Box id="page">
            {/* Navigate between normal home page (shows popular results) and for you page (shows personalized results) */}
            <Tabs
                value={tabIndex}
                onChange={handleTabChange}
                indicatorColor="secondary"
                textColor="inherit"
                variant="scrollable"
                scrollButtons="auto"
                allowScrollButtonsMobile
                aria-label="search-type-tabs"
                sx={{
                    marginBottom: 2,
                    '& .MuiTabs-flexContainer': {
                        justifyContent: 'center',
                    },
                }}
            >
                {tabOptions.map((option, index) => (
                    <Tab
                        key={index}
                        id={`home-tab-${index}`}
                        {...{ 'aria-controls': `home-tabpanel-${index}` }}
                        label={option[0]}
                        color={index === 0 ? '#ce6c12' : 'default'}
                    />
                ))}
            </Tabs>
            {/* Create new popup */}
            <CreateNewDialog
                isOpen={isCreateDialogOpen}
                handleClose={closeCreateDialog}
            />
            {/* Prompt stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '5vh', paddingBottom: '5vh' }}>
                <Typography component="h1" variant="h2" textAlign="center">What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <AutocompleteSearchBar
                    id="main-search"
                    placeholder='Search routines, projects, and more...'
                    options={autocompleteOptions}
                    getOptionKey={(option) => option.id}
                    getOptionLabel={(option) => option.label ?? ''}
                    getOptionLabelSecondary={(option) => option.objectType}
                    loading={loading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    session={session}
                    sx={{ width: 'min(100%, 600px)' }}
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Search results */}
                <TitleContainer
                    title={"Active Routines"}
                    helpText={activeRoutinesText}
                    loading={loading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {activeRoutines}
                </TitleContainer>
                <TitleContainer
                    title={"Completed Routines"}
                    helpText={completedRoutinesText}
                    loading={loading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {completedRoutines}
                </TitleContainer>
                <TitleContainer
                    title={"Recently Viewed"}
                    helpText={recentText}
                    loading={loading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {recent}
                </TitleContainer>
                <TitleContainer
                    title={"Starred"}
                    helpText={starredText}
                    loading={loading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {starred}
                </TitleContainer>
            </Stack>
        </Box>
    )
}