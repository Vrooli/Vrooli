import { Box, Button, Grid, Stack, Tab, Tabs, Typography, useTheme } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv, linkColors } from 'styles';
import { useQuery } from '@apollo/client';
import { popularQuery } from 'graphql/query';
import { AutocompleteSearchBar, ListTitleContainer, TitleContainer, ListMenu, PageContainer } from 'components';
import { useLocation } from '@shared/route';
import { APP_LINKS } from '@shared/consts';
import { HomePageProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { actionsItems, getUserLanguages, listToAutocomplete, listToListItems, openObject, SearchPageTabOption, shortcutsItems, stringifySearchParams, useReactSearch } from 'utils';
import { AutocompleteOption, NavigableObject } from 'types';
import { ListMenuItemData } from 'components/dialogs/types';
import { CreateIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SearchIcon, StandardIcon, UserIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { popular, popularVariables } from 'graphql/generated/popular';

const faqText =
    `## What is This?
Vrooli is an automation platform built for the decentralized age. We are aiming to become the "missing piece" in the [Project Catalyst](https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e) ecosystem, which:  
- guides proposers through the process of validating, creating, and developing projects  
- helps developers discover and implement projects  
- reduces the work required of voters to validate a project's utility and impact  


## How does it work?
Vrooli contains various *routines* that can be executed to perform a variety of tasks. Some are very specific (e.g. *How to create a Project Catalyst proposal*), and some are more general (e.g *Validate your business idea*).The more general routines are often constructed from a graph of *subroutines*. Each subroutine in turn can consist of multiple subroutines, and so on. This allows for routines to be built in a way that is flexible and scalable.

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
    { label: 'Organization', Icon: OrganizationIcon, value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Organizations}&advanced=true` },
    { label: 'Project', Icon: ProjectIcon, value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Projects}&advanced=true` },
    { label: 'Routine', Icon: RoutineIcon, value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Routines}&advanced=true` },
    { label: 'Standard', Icon: StandardIcon, value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Standards}&advanced=true` },
    { label: 'User', Icon: UserIcon, value: `${APP_LINKS.Search}?type=${SearchPageTabOption.Users}&advanced=true` },
]

const createNewPopupOptions: ListMenuItemData<string>[] = [
    { label: 'Organization', Icon: OrganizationIcon, value: `${APP_LINKS.Organization}/add` },
    { label: 'Project', Icon: ProjectIcon, value: `${APP_LINKS.Project}/add` },
    { label: 'Routine', Icon: RoutineIcon, value: `${APP_LINKS.Routine}/add` },
    { label: 'Standard', Icon: StandardIcon, value: `${APP_LINKS.Standard}/add` },
]

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
    const { data, refetch, loading } = useQuery<popular, popularVariables>(popularQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch, searchString]);
    const showHistoryTab = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    // Handle tabs
    const tabIndex = useMemo(() => {
        if (window.location.pathname === APP_LINKS.History) return 1;
        return 0;
    }, []);
    const handleTabChange = (e, newIndex) => {
        e.preventDefault();
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

    const languages = useMemo(() => getUserLanguages(session), [session]);

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
        const flattened = (Object.values(data?.popular ?? [])).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [languages, data, searchString]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // If selected item is an action (i.e. no navigation required), do nothing 
        // (search bar performs actions automatically)
        if (newValue.__typename === 'Action') {
            return;
        }
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
    const toSearchPage = useCallback((event: any, tab: SearchPageTabOption) => {
        event?.stopPropagation();
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search="${searchString}"`, { replace: true });
        // Navigate to search page
        setLocation(`${APP_LINKS.Search}?type=${tab}`);
    }, [searchString, setLocation]);

    /**
     * Replaces current state with search string, so that search is not lost
     */
    const beforeNavigation = useCallback((item: NavigableObject) => {
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(`${APP_LINKS.Home}?search=${searchString}`, { replace: true });
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
        let defaultOrder = [SearchPageTabOption.Routines, SearchPageTabOption.Projects, SearchPageTabOption.Organizations, SearchPageTabOption.Standards, SearchPageTabOption.Users];
        // Loop through keywords, and move ones which appear in the search string to the front
        // A keyword is only counted as a match if it has an exclamation point (!) at the beginning
        for (const keyword of Object.keys(SearchPageTabOption)) {
            if (containsWord(searchString, keyword)) {
                defaultOrder = [SearchPageTabOption[keyword], ...defaultOrder.filter(o => o !== SearchPageTabOption[keyword])];
            }
        }
        return defaultOrder;
    }, [searchString]);

    const feeds = useMemo(() => {
        let listFeeds: JSX.Element[] = [];
        for (const tab of feedOrder) {
            let currentList: any[] = [];
            let dummyType: string = '';
            switch (tab) {
                case SearchPageTabOption.Organizations:
                    currentList = data?.popular?.organizations ?? [];
                    dummyType = 'Organization';
                    break;
                case SearchPageTabOption.Projects:
                    currentList = data?.popular?.projects ?? [];
                    dummyType = 'Project';
                    break;
                case SearchPageTabOption.Routines:
                    currentList = data?.popular?.routines ?? [];
                    dummyType = 'Routine';
                    break;
                case SearchPageTabOption.Standards:
                    currentList = data?.popular?.standards ?? [];
                    dummyType = 'Standard';
                    break;
                case SearchPageTabOption.Users:
                    currentList = data?.popular?.users ?? [];
                    dummyType = 'User';
                    break;
            }
            const listFeedItems: JSX.Element[] = listToListItems({
                dummyItems: new Array(5).fill(dummyType),
                items: currentList,
                keyPrefix: `feed-list-item-${tab}`,
                loading,
                beforeNavigation,
                session,
                zIndex,
            });
            if (loading || listFeedItems.length > 0) {
                listFeeds.push((
                    <ListTitleContainer
                        key={`feed-list-${tab}`}
                        isEmpty={listFeedItems.length === 0}
                        title={getFeedTitle(`${tab}`)}
                        onClick={(e) => toSearchPage(e, tab)}
                        options={[['See more results', (e) => { toSearchPage(e, tab) }]]}
                    >
                        {listFeedItems}
                    </ListTitleContainer>
                ))
            }
        }
        return listFeeds;
    }, [beforeNavigation, data, feedOrder, getFeedTitle, loading, session, toSearchPage]);

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
        // If not logged in, redirect to login page
        if (!getCurrentUser(session).id) {
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: path
            })}`);
        }
        // Otherwise, navigate to create page
        else setLocation(path);
    }, [session, setLocation]);

    return (
        <PageContainer>
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
                                component='a'
                                href={option[1]}
                            />
                        ))}
                    </Tabs>
                </Box>
            )}
            {/* Advanced search dialog */}
            <ListMenu
                id={`open-advanced-search-menu`}
                anchorEl={advancedSearchAnchor}
                data={advancedSearchPopupOptions}
                onSelect={handleAdvancedSearchSelect}
                onClose={closeAdvancedSearch}
                zIndex={200}
            />
            {/* Create new dialog */}
            <ListMenu
                id={`create-new-object-menu`}
                anchorEl={createNewAnchor}
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
                    sxs={{ root: { width: 'min(100%, 600px)', paddingLeft: 2, paddingRight: 2 } }}
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column" mt={10}>
                {/* Search results */}
                {feeds}
                {/* Advanced search prompt TODO */}
                <Stack direction="column" spacing={2} justifyContent="center" alignItems="center">
                    <Typography component="h3" variant="h4" textAlign="center">Can't find what you're looking for?</Typography>
                    <Grid container spacing={2} sx={{ width: 'min(100%, 500px)', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                        <Grid item xs={6}>
                            <Button fullWidth onClick={openAdvancedSearch} startIcon={<SearchIcon />}>Advanced</Button>
                        </Grid>
                        <Grid item xs={6}>
                            <Button fullWidth onClick={openCreateNew} startIcon={<CreateIcon />}>Create</Button>
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
                        ...linkColors(palette)
                    }}
                >
                    <Box pl={2} pr={2}><Markdown>{faqText}</Markdown></Box>
                </TitleContainer>
            </Stack>
        </PageContainer>
    )
}