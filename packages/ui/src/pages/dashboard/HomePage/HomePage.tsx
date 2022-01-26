import { Box, Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv, containerShadow } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { ActorListItem, FeedList, OrganizationListItem, ProjectListItem, RoutineListItem, AutocompleteSearchBar, StandardListItem } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { HomePageProps } from '../types';
import Markdown from 'markdown-to-jsx';
import faqMarkdown from './faq.md';

const ObjectType = {
    Organization: 'Organization',
    Project: 'Project',
    Routine: 'Routine',
    Standard: 'Standard',
    User: 'User',
}

const linkMap = {
    [ObjectType.Organization]: APP_LINKS.SearchOrganizations,
    [ObjectType.Project]: APP_LINKS.SearchProjects,
    [ObjectType.Routine]: APP_LINKS.SearchRoutines,
    [ObjectType.Standard]: APP_LINKS.SearchStandards,
    [ObjectType.User]: APP_LINKS.SearchUsers,
}

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
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => { console.log('update search'); setSearchString(newValue) }, []);
    // Search query removes words that start with a '!'. These are used for sorting results. TODO doesn't work
    const { data, refetch } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } } });
    //const debouncedRefetch = useMemo(() => AwesomeDebouncePromise(refetch, 500), [refetch]);
    useEffect(() => { console.log('refetching...', session); refetch() }, [refetch, searchString]);

    const { routines, projects, organizations, standards, users } = useMemo(() => {
        if (!data) return { routines: [], projects: [], organizations: [], standards: [], users: [] };
        const { routines, projects, organizations, standards, users } = data.autocomplete;
        return { routines, projects, organizations, standards, users };
    }, [data]);

    const autocompleteOptions = useMemo(() => {
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
    const onInputSelect = useCallback((_e: any, newValue: any) => {
        console.log('onInputSelect', newValue);
        if (!newValue) return;
        // Determine object from selected label
        const selectedItem = autocompleteOptions.find(o => `${o.title} | ${o.objectType}` === newValue);
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
    const openSearch = useCallback((linkBase: string, id?: string) => {
        return setLocation(id ? `${linkBase}/${id}` : linkBase);
    }, [])

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
                    listFeedItems = organizations.map(o => (
                        <OrganizationListItem
                            key={`feed-list-item-${o.id}`}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.Project:
                    listFeedItems = projects.map(o => {console.log('p', o); return (
                        <ProjectListItem
                            key={`feed-list-item-${o.id}`}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    )})
                    break;
                case ObjectType.Routine:
                    listFeedItems = routines.map(o => (
                        <RoutineListItem
                            key={`feed-list-item-${o.id}`}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.Standard:
                    listFeedItems = standards.map(o => (
                        <StandardListItem
                            key={`feed-list-item-${o.id}`}
                            session={session}
                            data={o}
                            isOwn={false}
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
                case ObjectType.User:
                    listFeedItems = users.map(o => (
                        <ActorListItem 
                            key={`feed-list-item-${o.id}`} 
                            session={session} 
                            data={o} 
                            isOwn={false} 
                            onClick={() => openSearch(linkMap[objectType], o.id)}
                        />
                    ))
                    break;
            }
            listFeeds.push((
                <FeedList
                    key={`feed-list-${objectType}`}
                    title={getFeedTitle(`${objectType}s`)}
                    onClick={(id?: string) => openSearch(linkMap[objectType], id)}
                >
                    {listFeedItems}
                </FeedList>
            ))
        }
        return listFeeds;
    }, [feedOrder, organizations, projects, routines, standards, users, openSearch]);

    // Parse FAQ markdown from .md file
    const [faqText, setFaqText] = useState<string>('');
    useEffect(() => {
        fetch(faqMarkdown)
            .then((response) => response.text())
            .then((text) => {
                setFaqText(text);
            });
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
                    getOptionLabel={(o: any) => `${o.title} | ${o.objectType}`}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    sx={{ width: 'min(100%, 600px)' }}
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Examples stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '40px', paddingBottom: '40px' }}>
                <Typography component="h2" variant="h5">Examples</Typography>
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {feeds}
                {/* FAQ */}
                <Box
                    sx={{
                        ...containerShadow,
                        borderRadius: '8px',
                        background: (t) => t.palette.background.default,

                        '& h2': {
                            fontSize: '2rem',
                            fontWeight: '500',
                            textAlign: 'center',
                        },
                    }}
                >
                    <Box sx={{
                        background: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        borderRadius: '8px 8px 0 0',
                        padding: 0.5,
                    }}>
                        <Typography component="h2" variant="h4" textAlign="center">FAQ</Typography>
                    </Box>
                    <Box pl={2} pr={2}><Markdown>{faqText}</Markdown></Box>
                </Box>
            </Stack>
        </Box>
    )
}