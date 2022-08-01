import { Box, Button, Grid, Stack, Tab, Tabs, Typography, useTheme } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv } from 'styles';
import { homePage, homePageVariables } from 'graphql/generated/homePage';
import { useQuery } from '@apollo/client';
import { homePageQuery } from 'graphql/query';
import { AutocompleteSearchBar, ListTitleContainer, TitleContainer, ListMenu } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { HomePageProps } from '../types';
import Markdown from 'markdown-to-jsx';
import {
    Add as CreateIcon,
    Search as SearchIcon,
} from '@mui/icons-material';
import { listToAutocomplete, listToListItems, ObjectType, openObject, OpenObjectProps, openSearchPage, useReactSearch } from 'utils';
import { AutocompleteOption } from 'types';
import { ListMenuItemData } from 'components/dialogs/types';

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
Vrooli is being developed by Matt Halloran. He is a software developer and decentralization advocate, who believes that a carefully designed collaboration and automation platform can produce a new paradigm in human productivity. Here are [his contact links](https://matthalloran.info).


## What can I do?
The simplest thing you can do right now is to participate! You can:  
- [Execute a routine](https://app.vrooli.com/search/routine)  
- [Create a routine](https://app.vrooli.com/create/routine)  
- [Familiarize yourself with our vision](https://docs.google.com/document/d/1zHYdjAyy01SSFZX0O-YnZicef7t6sr1leOFnynQQOx4)  
- [Join us on Discord](https://discord.gg/VyrDFzbmmF)  
- [Follow us on Twitter](https://twitter.com/VrooliOfficial)  

If you would like to contribute to the development of Vrooli, please contact us!
`

const advancedSearchPopupOptions: ListMenuItemData<string>[] = [
    { label: 'Organization', value: `${APP_LINKS.SearchOrganizations}?advanced=true` },
    { label: 'Project', value: `${APP_LINKS.SearchProjects}?advanced=true` },
    { label: 'Routine', value: `${APP_LINKS.SearchRoutines}?advanced=true` },
    { label: 'Standard', value: `${APP_LINKS.SearchStandards}?advanced=true` },
    { label: 'User', value: `${APP_LINKS.SearchUsers}?advanced=true` },
]

const createNewPopupOptions: ListMenuItemData<string>[] = [
    { label: 'Organization', value: `${APP_LINKS.Organization}/add` },
    { label: 'Project', value: `${APP_LINKS.Project}/add` },
    { label: 'Routine (Single Step)', value: `${APP_LINKS.Routine}/add` },
    { label: 'Routine (Multi Step)', value: `${APP_LINKS.Routine}/add?build=true` },
    { label: 'Standard', value: `${APP_LINKS.Standard}/add` },
]

const tabOptions = [
    ['For You', APP_LINKS.Home],
    ['History', APP_LINKS.History],
];

const examplesData: [string, string][] = [
    ['Start a new business', '5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'],
    // ['Learn about Project Catalyst', ''], //TODO
    // ['Fund your project', ''], //TODO
    ['Create a Cardano native asset token', '3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9'],
]

interface ShortcutItem {
    label: string;
    link: string;
}
/**
 * Shortcuts that can appear in the search bar
 */
const shortcuts: ShortcutItem[] = [
    {
        label: 'Create new organization',
        link: `${APP_LINKS.Organization}/add`,
    },
    {
        label: 'Create new project',
        link: `${APP_LINKS.Project}/add`,
    },
    {
        label: 'Create new single-step routine',
        link: `${APP_LINKS.Routine}/add`,
    },
    {
        label: 'Create new multi-step routine',
        link: `${APP_LINKS.Routine}/add?build=true`,
    },
    {
        label: 'Create new standard',
        link: `${APP_LINKS.Standard}/add`,
    },
    {
        label: 'View learn dashboard',
        link: `${APP_LINKS.Learn}`,
    },
    {
        label: 'View research dashboard',
        link: `${APP_LINKS.Research}`,
    },
    {
        label: 'View develop dashboard',
        link: `${APP_LINKS.Develop}`,
    },
    {
        label: 'View history page',
        link: `${APP_LINKS.History}`,
    },
    {
        label: 'Search organizations',
        link: `${APP_LINKS.SearchOrganizations}`,
    },
    {
        label: 'Search projects',
        link: `${APP_LINKS.SearchProjects}`,
    },
    {
        label: 'Search routines',
        link: `${APP_LINKS.SearchRoutines}`,
    },
    {
        label: 'Search standards',
        link: `${APP_LINKS.SearchStandards}`,
    },
    {
        label: 'Search users',
        link: `${APP_LINKS.SearchUsers}`,
    },
    {
        label: 'Search organizations advanced',
        link: `${APP_LINKS.SearchOrganizations}?advanced=true`,
    },
    {
        label: 'Search projects advanced',
        link: `${APP_LINKS.SearchProjects}?advanced=true`,
    },
    {
        label: 'Search routines advanced',
        link: `${APP_LINKS.SearchRoutines}?advanced=true`,
    },
    {
        label: 'Search standards advanced',
        link: `${APP_LINKS.SearchStandards}?advanced=true`,
    },
    {
        label: 'Search users advanced',
        link: `${APP_LINKS.SearchUsers}?advanced=true`,
    },
    {
        label: `Beginner's Guide`,
        link: `${APP_LINKS.Welcome}`,
    },
    {
        label: 'FAQ',
        link: `${APP_LINKS.FAQ}`,
    },
]
// Shape shortcuts to match AutoCompleteListItem format
const shortcutsItems: AutocompleteOption[] = shortcuts.map(({ label, link }) => ({
    __typename: "Shortcut",
    label,
    id: link,
}))

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
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>('');
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const { data, refetch, loading } = useQuery<homePage, homePageVariables>(homePageQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } } });
    useEffect(() => { refetch() }, [refetch, searchString]);
    const showHistoryTab = useMemo(() => session?.isLoggedIn === true, [session?.isLoggedIn]);

    // Handle tabs
    const tabIndex = useMemo(() => {
        if (window.location.pathname === APP_LINKS.History) return 1;
        return 0;
    }, []);
    const handleTabChange = (_e, newIndex) => {
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // If "help" typed, add help and faq shortcuts as first result
        if (searchString.toLowerCase().startsWith('help')) {
            firstResults.push({
                __typename: "Shortcut",
                label: `Help - Beginner's Guide`,
                id: APP_LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: 'Help - FAQ',
                id: APP_LINKS.FAQ,
            });
        }
        // Group all query results and sort by number of stars
        const flattened = (Object.values(data?.homePage ?? [])).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems];
    }, [languages, data, searchString]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Replace current state with search string, so that search is not lost. 
        // Only do this if the selected item is not a shortcut
        if (newValue.__typename !== 'Shortcut' && searchString) setLocation(`${APP_LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(APP_LINKS.Home, { replace: true });
        // If selected item is a shortcut, navigate to it
        if (newValue.__typename === 'Shortcut') {
            setLocation(newValue.id);
        }
        // Otherwise, navigate to item page
        else {
            openObject(newValue, setLocation);
        }
    }, [searchString, setLocation]);

    // Feed title is Popular when no search
    const getFeedTitle = useCallback((objectName: string) => {
        const prefix = !Boolean(searchString) ? 'Popular ' : '';
        return `${prefix}${objectName}`;
    }, [searchString]);

    /**
     * Opens search page for object type
     */
    const toSearchPage = useCallback((event: any, objectType: ObjectType) => {
        event?.stopPropagation();
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search="${searchString}"`, { replace: true });
        // Navigate to search page
        openSearchPage(objectType, setLocation);
    }, [searchString, setLocation]);

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: OpenObjectProps['object'], event: any) => {
        event?.stopPropagation();
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search=${searchString}`, { replace: true });
        // Navigate to item page
        openObject(item, setLocation);
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
            let currentList: any[] = [];
            let dummyType: string = '';
            switch (objectType) {
                case ObjectType.Organization:
                    currentList = data?.homePage?.organizations ?? [];
                    dummyType = 'Organization';
                    break;
                case ObjectType.Project:
                    currentList = data?.homePage?.projects ?? [];
                    dummyType = 'Project';
                    break;
                case ObjectType.Routine:
                    currentList = data?.homePage?.routines ?? [];
                    dummyType = 'Routine';
                    break;
                case ObjectType.Standard:
                    currentList = data?.homePage?.standards ?? [];
                    dummyType = 'Standard';
                    break;
                case ObjectType.User:
                    currentList = data?.homePage?.users ?? [];
                    dummyType = 'User';
                    break;
            }
            const listFeedItems: JSX.Element[] = listToListItems({
                dummyItems: new Array(5).fill(dummyType),
                items: currentList,
                keyPrefix: `feed-list-item-${objectType}`,
                loading,
                onClick: toItemPage,
                session,
            });
            if (loading || listFeedItems.length > 0) {
                listFeeds.push((
                    <ListTitleContainer
                        key={`feed-list-${objectType}`}
                        isEmpty={listFeedItems.length === 0}
                        title={getFeedTitle(`${objectType}s`)}
                        onClick={(e) => toSearchPage(e, objectType)}
                        options={[['See more results', (e) => { toSearchPage(e, objectType) }]]}
                    >
                        {listFeedItems}
                    </ListTitleContainer>
                ))
            }
        }
        return listFeeds;
    }, [data, feedOrder, getFeedTitle, loading, session, toItemPage, toSearchPage]);

    // Menu for opening an advanced search page
    const [advancedSearchAnchor, setAdvancedSearchAnchor] = useState<any>(null);
    const openAdvancedSearch = useCallback((ev: React.MouseEvent<any>) => {
        setAdvancedSearchAnchor(ev.currentTarget)
    }, [setAdvancedSearchAnchor]);
    const closeAdvancedSearch = useCallback(() => setAdvancedSearchAnchor(null), []);
    const handleAdvancedSearchSelect = useCallback((path: string) => {
        setLocation(path);
    }, [setLocation]);

    // Menu for opening a create page
    const [createNewAnchor, setCreateNewAnchor] = useState<any>(null);
    const openCreateNew = useCallback((ev: React.MouseEvent<any>) => {
        setCreateNewAnchor(ev.currentTarget)
    }, [setCreateNewAnchor]);
    const closeCreateNew = useCallback(() => setCreateNewAnchor(null), []);
    const handleCreateNewSelect = useCallback((path: string) => {
        setLocation(path);
    }, [setLocation]);

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
            {/* Navigate between normal home page (shows popular results) and for you page (shows personalized results) */}
            {showHistoryTab && (
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
                                id={`home-tab-${index}`}
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
            )}
            {/* Advanced search dialog */}
            <ListMenu
                id={`open-advanced-search-menu`}
                anchorEl={advancedSearchAnchor}
                title='Select Object Type'
                data={advancedSearchPopupOptions}
                onSelect={handleAdvancedSearchSelect}
                onClose={closeAdvancedSearch}
                zIndex={200}
            />
            {/* Create new dialog */}
            <ListMenu
                id={`open-advanced-search-menu`}
                anchorEl={createNewAnchor}
                title='Create New...'
                data={createNewPopupOptions}
                onSelect={handleCreateNewSelect}
                onClose={closeCreateNew}
                zIndex={200}
            />
            {/* Prompt stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: { xs: '5vh', sm: '20vh' } }}>
                <Typography component="h1" variant="h3" textAlign="center">What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <AutocompleteSearchBar
                    id="main-search"
                    placeholder='Search routines, projects, and more...'
                    options={autocompleteOptions}
                    loading={loading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    session={session}
                    showSecondaryLabel={true}
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
                            onClick={() => { setLocation(`${APP_LINKS.Routine}/${example[1]}`) }}
                            sx={{
                                color: palette.text.secondary,
                                fontStyle: 'italic',
                                cursor: 'pointer',
                                textAlign: 'center',
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
                            <Button fullWidth onClick={openCreateNew} startIcon={<CreateIcon />}>Create New</Button>
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