import { Box, Button, Grid, Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { ActorListItem, OrganizationListItem, ProjectListItem, RoutineListItem, AutocompleteSearchBar, StandardListItem, TitleContainer, CreateNewDialog } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { HomePageProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { parseSearchParams } from 'utils/urlTools';
import {
    Add as CreateIcon,
    Search as SearchIcon,
} from '@mui/icons-material';
import { getTranslation } from 'utils';

const faqText =
`## What is This?
Vrooli is an automation platform built for the decentralized age. We are aiming to become the "missing piece" in the [Project Catalyst](https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e) ecosystem, which:  
- guides proposers through the process of validating, creating, and developing projects  
- helps developers discover and implement projects  
- reduces the work required of voters to validate a project's utility and impact  


## How does it work?
Vrooli contains various *routines* that can be executed to perform a variety of tasks. Some are very specific (e.g. "How to create a Project Catalyst proposal"), and some are more general (e.g "Validate your business idea").The more general routines are often constructed from a graph of *subroutines*. Each subroutine in turn can consist of multiple subroutines, and so on. This allows for routines to be built in a way that is flexible and scalable.

When a proposer is looking for a new project to tackle, they can search for existing routines which are both *complex* (i.e. consist of many steps) and *popular* (e.g. number of votes, stars, executions, and parent routines). This combination of complexity and popularity can be used to estimate the time and money that would be saved by the community as a whole if a simpler routine was created. Using the design of a simpler routine, you can also demonstrate to voters your project's user experience.


## What are the limitations of Vrooli?
Vrooli is a work in progress. This is the Minimum Viable product, which has been developed by a single person. While we have grand plans for Vrooli to become a robust and decentralized automation platform, this is not currently possible.

As of now, all routines are text-based, which - not surprisingly - limits their ability to "automate". They can link to external resources, but it is up to the user to step through the routine, follow the instructions, and fill in the requested fields. 
We have several important features in our [white paper](https://docs.google.com/document/d/1zHYdjAyy01SSFZX0O-YnZicef7t6sr1leOFnynQQOx4) that address these issues, such as smart contract execution, self-sovereign data storage, and more. Luckily, since routines are built on top of each other, automating one low-level routine can simplify the process of many other routines. 


## Why is This Important?
As more of our world is eaten by software, the opportunity to automate every aspect of our lives becomes increasingly tempting. Many optimists envision a future where they can spend most of their life pursuing what they love, and not have to worry about the drab of modern work and corporate bureaucracy. We can build this future together.


## How are we unique?
General-purpose automation platforms exist, but they are limited by their reliance on completely automated routines. Meaning, if *every* step in a task cannot be *100%* automated, then you are out of luck. We believe this is too restrictive, and prevents innovation from flourishing.

Automations don't pop into existence. They must be though of, designed, validated, and implemented. By combining the execution of automated routines with the design of the routines themselves, **the average person can contribute their ideas to the community, and let the natural process of discovery and implementation transform their visions into reality.**


## Who Created Vrooli?
Vrooli is being developed by Matt Halloran. He is a software developer and decentralization advocate, who believes that a carefully designed collaboration and automation platform can produce a new paradigm in human productivity. You can find his contact links [here](https://matthalloran.info).


## What can I do?
The simplest thing you can do right now is to participate! You can:  
- [Execute a routine](https://app.vrooli.com/search/routine)  
- [Create a routine](https://app.vrooli.com/create/routine)  
- [Familiarize yourself with our vision](https://docs.google.com/document/d/1zHYdjAyy01SSFZX0O-YnZicef7t6sr1leOFnynQQOx4)  
- [Join us on Discord](https://discord.gg/VyrDFzbmmF)  
- [Follow us on Twitter](https://twitter.com/VrooliOfficial)  

If you would like to contribute to the development of Vrooli, please contact us!
`

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

interface AutocompleteListItem {
    title: string | null;
    id: string;
    stars: number;
    objectType: string;
}

const examplesData: [string, string][] = [
    ['Start a new business', '5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'],
    ['Learn about Project Catalyst', ''], //TODO
    ['Fund your project', ''], //TODO
    ['Create a Cardano native asset token', '3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'],
]

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
        return search ?? '';
    });
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    // Search query removes words that start with a '!'. These are used for sorting results. TODO doesn't work
    const { data, refetch, loading } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } } });
    //const debouncedRefetch = useMemo(() => AwesomeDebouncePromise(refetch, 500), [refetch]);
    useEffect(() => { refetch() }, [refetch, searchString]);

    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

    const { routines, projects, organizations, standards, users } = useMemo(() => {
        if (!data) return { routines: [], projects: [], organizations: [], standards: [], users: [] };
        const { routines, projects, organizations, standards, users } = data.autocomplete;
        return { routines, projects, organizations, standards, users };
    }, [data]);

    const autocompleteOptions: AutocompleteListItem[] = useMemo(() => {
        if (!data) return [];
        const routines = data.autocomplete.routines.map(r => ({ title: getTranslation(r, 'title', languages, true), id: r.id, stars: r.stars, objectType: ObjectType.Routine }));
        const projects = data.autocomplete.projects.map(p => ({ title: getTranslation(p, 'name', languages, true), id: p.id, stars: p.stars, objectType: ObjectType.Project }));
        const organizations = data.autocomplete.organizations.map(o => ({ title: getTranslation(o, 'name', languages, true), id: o.id, stars: o.stars, objectType: ObjectType.Organization }));
        const standards = data.autocomplete.standards.map(s => ({ title: s.name, id: s.id, stars: s.stars, objectType: ObjectType.Standard }));
        const users = data.autocomplete.users.map(u => ({ title: u.name, id: u.id, stars: u.stars, objectType: ObjectType.User }));
        const options = [...routines, ...projects, ...organizations, ...standards, ...users].sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return options;
    }, [data, languages]);

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
        if (searchString) setLocation(`${APP_LINKS.Home}?search=${searchString}`, { replace: true });
        // Determine object from selected label
        const selectedItem = autocompleteOptions.find(o => o.id === newValue.id);
        if (!selectedItem) return;
        const linkBases = linkMap[selectedItem.objectType];
        setLocation(selectedItem.id ? `${linkBases[1]}/${selectedItem.id}` : linkBases[0]);
    }, [autocompleteOptions]);

    // Feed title is Popular when no search
    const getFeedTitle = useCallback((objectName: string) => {
        const prefix = !Boolean(searchString) ? 'Popular ' : '';
        return `${prefix}${objectName}`;
    }, [searchString]);

    // Opens correct search page
    const openSearch = useCallback((event: any, linkBases: [string, string], id?: string) => {
        event?.stopPropagation();
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
                            onClick={(e) => openSearch(e, linkMap[objectType], o.id)}
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
                            onClick={(e) => openSearch(e, linkMap[objectType], o.id)}
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
                            onClick={(e) => openSearch(e, linkMap[objectType], o.id)}
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
                            onClick={(e) => openSearch(e, linkMap[objectType], o.id)}
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
                            onClick={(e) => openSearch(e, linkMap[objectType], o.id)}
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
                        onClick={(e) => openSearch(e, linkMap[objectType])}
                        options={[['See more results', (e) => { openSearch(e, linkMap[objectType]) }]]}
                    >
                        {listFeedItems}
                    </TitleContainer>
                ))
            }
        }
        return listFeeds;
    }, [feedOrder, getFeedTitle, loading, organizations, projects, routines, session, standards, users, openSearch]);

    return (
        <Box id="page">
            {/* Create new popup */}
            <CreateNewDialog
                isOpen={isCreateDialogOpen}
                handleClose={closeCreateDialog}
            />
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
                    session={session}
                    sx={{ width: 'min(100%, 600px)' }}
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Examples stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '40px', paddingBottom: '40px' }}>
                <Typography component="h2" variant="h5" pb={1}>Examples</Typography>
                {
                    examplesData.map((example, index) => (
                        <Typography
                            key={`example-${index}`}
                            component="p"
                            variant="h6"
                            onClick={() => { setLocation(`${APP_LINKS.Run}/${example[1]}`) }}
                            sx={{
                                color: (t) => t.palette.text.secondary,
                                fontStyle: 'italic',
                                cursor: 'pointer',
                            }}
                        >"{example[0]}"</Typography>
                    ))
                }
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                {/* Search results */}
                {feeds}
                {/* Advanced search prompt TODO */}
                <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                    <Typography component="h3" variant="h4" textAlign="center">Can't find what you're looking for?</Typography>
                    <Grid container spacing={2} sx={{ width: 'min(100%, 500px)', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <Grid item xs={12} sm={6}>
                            <Button fullWidth onClick={openAdvancedSearch} startIcon={<SearchIcon />}>Advanced Search</Button>
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <Button fullWidth onClick={openCreateDialog} startIcon={<CreateIcon />}>Create New</Button>
                        </Grid>
                    </Grid>
                </Stack>
                {/* FAQ */}
                <TitleContainer
                    id="faq"
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