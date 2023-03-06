import { Button, Grid, Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { centeredDiv } from 'styles';
import { useQuery } from '@apollo/client';
import { SiteSearchBar, TitleContainer, ListMenu, PageTabs, TopBar } from 'components';
import { useLocation } from '@shared/route';
import { APP_LINKS, HomeInput, HomeResult } from '@shared/consts';
import { HomeViewProps } from '../types';
import { actionsItems, getUserLanguages, listToAutocomplete, listToListItems, openObject, SearchPageTabOption, shortcuts, useDisplayApolloError, useReactSearch } from 'utils';
import { AutocompleteOption, NavigableObject, ShortcutOption, Wrap } from 'types';
import { ListMenuItemData } from 'components/dialogs/types';
import { CreateIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SearchIcon, StandardIcon, UserIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { feedHome } from 'api/generated/endpoints/feed_home';
import { PageTab } from 'components/types';
import { useTranslation } from 'react-i18next';

enum TabOptions {
    ForYou = "ForYou",
    History = "History",
}

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

const zIndex = 200;

export const HomeView = ({
    display = 'page',
    session
}: HomeViewProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // TODO query should take the current user schedule into account somehow
    const [searchString, setSearchString] = useState<string>('');
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const { data, refetch, loading, error } = useQuery<Wrap<HomeResult, 'home'>, Wrap<HomeInput, 'input'>>(feedHome, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch, searchString]);
    useDisplayApolloError(error);
    const showTabs = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => ([{
        index: 0,
        href: APP_LINKS.Home,
        label: t('ForYou'),
        value: TabOptions.ForYou,
    }, {
        index: 1,
        href: APP_LINKS.History,
        label: t('History'),
        value: TabOptions.History,
    }]), [t]);
    const currTab = useMemo(() => tabs[0], [tabs])
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        setLocation(tab.href!, { replace: true });
    }, [setLocation]);

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const shortcutsItems = useMemo<ShortcutOption[]>(() => shortcuts.map(({ label, labelArgs, value }) => ({
        __typename: "Shortcut",
        label: t(label, { ...(labelArgs ?? {}), defaultValue: label }) as string,
        id: value,
    })), [t]);

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
        // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
        const flattened = (Object.values(data?.home ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [searchString, data?.home, languages, shortcutsItems]);

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

    /**
     * Opens search page for object type
     */
    const toSearchPage = useCallback((event: any, tab: SearchPageTabOption) => {
        event?.stopPropagation();
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(APP_LINKS.Home, { replace: true, searchParams: { search: searchString } });
        // Navigate to search page
        setLocation(APP_LINKS.Search, { searchParams: { type: tab } });
    }, [searchString, setLocation]);

    /**
     * Replaces current state with search string, so that search is not lost
     */
    const beforeNavigation = useCallback((item: NavigableObject) => {
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(APP_LINKS.Home, { replace: true, searchParams: { search: searchString } });
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

    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
                // Navigate between for you and history pages
                below={showTabs && (
                    <PageTabs
                        ariaLabel="home-tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />
                )}
            />
            {/* Prompt stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: { xs: '5vh', sm: '20vh' } }}>
                <Typography component="h1" variant="h3" textAlign="center">What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <SiteSearchBar
                    id="main-search"
                    placeholder='SearchHome'
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
            {/* Result feeds */}
            <Stack spacing={10} direction="column" mt={10}>
                {/* Resources */}
                <TitleContainer
                    titleKey="Resource"
                    titleVariables={{ count: 2 }}
                >
                    {/* TODO */}
                </TitleContainer>
                {/* Events */}
                <TitleContainer
                    titleKey="Schedule"
                >
                    {/* TODO */}
                </TitleContainer>
                {/* Reminders */}
                <TitleContainer
                    titleKey="ToDo"
                >
                    {/* TODO */}
                </TitleContainer>
                {/* Notes */}
                <TitleContainer
                    titleKey="Note"
                    titleVariables={{ count: 2 }}
                >
                    {/* TODO */}
                </TitleContainer>
            </Stack>
        </>
    )
}