import { Box, Button, Grid, Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv, containerShadow } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { ActorListItem, OrganizationListItem, ProjectListItem, RoutineListItem, AutocompleteSearchBar, StandardListItem, TitleContainer } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { HomePageProps } from '../types';
import Markdown from 'markdown-to-jsx';
import faqMarkdown from './faq.md';
import { parseSearchParams } from 'utils/urlTools';
import {
    Add as CreateIcon,
    Search as SearchIcon,
} from '@mui/icons-material';

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
    [ObjectType.Routine]: [APP_LINKS.SearchRoutines, APP_LINKS.Routine],
    [ObjectType.Standard]: [APP_LINKS.SearchStandards, APP_LINKS.Standard],
    [ObjectType.User]: [APP_LINKS.SearchUsers, APP_LINKS.Profile],
}

interface AutocompleteListItem {
    title: string | null;
    id: string;
    stars: number;
    objectType: string;
}

const examples = [
    'Start a new business',
    'Learn about Project Catalyst',
    'Fund your project',
    'Create a Cardano native asset token',
].map(example => (
    <Typography component="p" variant="h6" sx={{
        color: (t) => t.palette.text.secondary,
        fontStyle: 'italics'
    }}>"{example}"</Typography>
))

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HomePage = ({
    session
}: HomePageProps) => {
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>(() => {
        const { search } = parseSearchParams(window.location.search);
        console.log('in calculate search string', search);
        return search ?? '';
    });
    const updateSearch = useCallback((newValue: any) => { console.log('update search'); setSearchString(newValue) }, []);
    // Search query removes words that start with a '!'. These are used for sorting results. TODO doesn't work
    const { data, refetch, loading } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } } });
    //const debouncedRefetch = useMemo(() => AwesomeDebouncePromise(refetch, 500), [refetch]);
    useEffect(() => { console.log('refetching...', session); refetch() }, [refetch, searchString]);

    const { routines, projects, organizations, standards, users } = useMemo(() => {
        if (!data) return { routines: [], projects: [], organizations: [], standards: [], users: [] };
        const { routines, projects, organizations, standards, users } = data.autocomplete;
        return { routines, projects, organizations, standards, users };
    }, [data]);

    const autocompleteOptions: AutocompleteListItem[] = useMemo(() => {
        if (!data) return [];
        const routines = data.autocomplete.routines.map(r => ({ title: r.title, id: r.id, stars: r.stars, objectType: ObjectType.Routine }));
        const projects = data.autocomplete.projects.map(p => ({ title: p.name, id: p.id, stars: p.stars, objectType: ObjectType.Project }));
        const organizations = data.autocomplete.organizations.map(o => ({ title: o.name, id: o.id, stars: o.stars, objectType: ObjectType.Organization }));
        const standards = data.autocomplete.standards.map(s => ({ title: s.name, id: s.id, stars: s.stars, objectType: ObjectType.Standard }));
        const users = data.autocomplete.users.map(u => ({ title: u.username, id: u.id, stars: u.stars, objectType: ObjectType.User }));
        const options = [...routines, ...projects, ...organizations, ...standards, ...users].sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return options;
    }, [data]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteListItem) => {
        console.log('onInputSelect', newValue);
        if (!newValue) return;
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search=${searchString}`, { replace: true });
        // Determine object from selected label
        const selectedItem = autocompleteOptions.find(o => o.id === newValue.id);
        if (!selectedItem) return;
        console.log('selectedItem', selectedItem);
        return openSearch(linkMap[selectedItem.objectType], selectedItem.id);
    }, [autocompleteOptions]);

    // Feed title is Popular when no search
    const getFeedTitle = useCallback((objectName: string) => {
        const prefix = !Boolean(searchString) ? 'Popular ' : '';
        return `${prefix}${objectName}`;
    }, [searchString]);

    // Opens correct search page
    const openSearch = useCallback((linkBases: [string, string], id?: string) => {
        console.log('open search', searchString, id ? `${linkBases[1]}/${id}` : linkBases[0])
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search=${searchString}`, { replace: true });
        setLocation(id ? `${linkBases[1]}/${id}` : linkBases[0]);
    }, [searchString, setLocation]);

    /**
     * Determine the order that the feed lists should be displayed in.
     * If a key word (e.g. "Routine", "Organization", etc.) is in the search string, then 
     * the list of that type should be displayed first.
     */
    const feedOrder = useMemo(() => {
        // Helper method for checking if a word (NOT a substring) is in the search string
        const containsWord = (str: string, word: string) => str.toLowerCase().match(new RegExp("\\b" + `!${word}`.toLowerCase() + "\\b")) != null;
        // Set default order
        let defaultOrder = [ObjectType.Routine, ObjectType.Project, ObjectType.Organization, ObjectType.Standard, ObjectType.User];
        // Loop through keywords, and move ones which appear in the search string to the front
        // A keyword is only counted as a match if it has an exclamation point (!) at the beginning
        for (const keyword of Object.keys(ObjectType)) {
            if (containsWord(searchString, keyword)) {
                defaultOrder = [ObjectType[keyword], ...defaultOrder.filter(o => o !== ObjectType[keyword])];
            }
        }
        return defaultOrder;
    }, [searchString]);

    const feeds = useMemo(() => {
        console.log('updating feeds');
        let listFeeds: JSX.Element[] = [];
        for (const objectType of feedOrder) {
            let listFeedItems: JSX.Element[] = [];
            switch (objectType) {
                case ObjectType.Organization:
                    listFeedItems = organizations.map((o, index) => (
                        <OrganizationListItem
                            key={`feed-list-item-${o.id}`}
                            index={index}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.Project:
                    listFeedItems = projects.map((o, index) => (
                        <ProjectListItem
                            key={`feed-list-item-${o.id}`}
                            index={index}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.Routine:
                    listFeedItems = routines.map((o, index) => (
                        <RoutineListItem
                            key={`feed-list-item-${o.id}`}
                            index={index}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.Standard:
                    listFeedItems = standards.map((o, index) => (
                        <StandardListItem
                            key={`feed-list-item-${o.id}`}
                            index={index}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.User:
                    listFeedItems = users.map((o, index) => (
                        <ActorListItem
                            key={`feed-list-item-${o.id}`}
                            index={index}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
            }
            if (loading || listFeedItems.length > 0) {
                listFeeds.push((
                    <TitleContainer
                        key={`feed-list-${objectType}`}
                        title={getFeedTitle(`${objectType}s`)}
                        loading={loading}
                        onClick={() => openSearch(linkMap[objectType])}
                        options={[['See more results', () => openSearch(linkMap[objectType])]]}
                    >
                        {listFeedItems}
                    </TitleContainer>
                ))
            }
        }
        return listFeeds;
    }, [feedOrder, loading, organizations, projects, routines, standards, users, openSearch]);

    // Parse FAQ markdown from .md file
    const [faqText, setFaqText] = useState<string>('');
    useEffect(() => {
        fetch(faqMarkdown).then((r) => r.text()).then((text) => { setFaqText(text) });
    }, []);

    return (
        <Box id="page">
            {/* Prompt stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: { xs: '5vh', sm: '30vh' } }}>
                <Typography component="h1" variant="h2" textAlign="center">What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <AutocompleteSearchBar
                    id="main-search"
                    placeholder='Search routines, projects, and more...'
                    options={autocompleteOptions}
                    getOptionKey={(option) => option.id}
                    getOptionLabel={(option) => option.title ?? ''}
                    getOptionLabelSecondary={(option) => option.objectType}
                    loading={loading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    sx={{ width: 'min(100%, 600px)' }}
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Examples stack TODO */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '40px', paddingBottom: '40px' }}>
                <Typography component="h2" variant="h5" pb={1}>Examples</Typography>
                {examples}
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Search results */}
                {feeds}
                {/* Advanced search prompt TODO */}
                <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                    <Typography variant="h4" textAlign="center">Can't find what you're looking for?</Typography>
                    <Grid container spacing={2} sx={{ width: 'min(100%, 500px)', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <Grid item xs={12} sm={6}>
                            <Button fullWidth startIcon={<SearchIcon />}>Advanced Search</Button>
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <Button fullWidth startIcon={<CreateIcon />}>Create New</Button>
                        </Grid>
                    </Grid>
                </Stack>
                {/* FAQ */}
                <TitleContainer
                    key={`faq-container`}
                    title={'FAQ'}
                    sx={{
                        '& h2': {
                            fontSize: '2rem',
                            fontWeight: '500',
                            textAlign: 'center',
                        },
                    }}
                >
                    <Box pl={2} pr={2}><Markdown>{faqText}</Markdown></Box>
                </TitleContainer>
            </Stack>
        </Box>
    )
}