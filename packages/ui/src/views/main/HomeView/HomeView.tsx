import { useQuery } from '@apollo/client';
import { Stack } from '@mui/material';
import { HomeInput, HomeResult, LINKS, ResourceList } from '@shared/consts';
import { useLocation } from '@shared/route';
import { DUMMY_ID } from '@shared/uuid';
import { feedHome } from 'api/generated/endpoints/feed_home';
import { TitleContainer } from 'components/containers/TitleContainer/TitleContainer';
import { SiteSearchBar } from 'components/inputs/search';
import { ResourceListHorizontal } from 'components/lists/resource';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { HomePrompt } from 'components/text/HomePrompt/HomePrompt';
import { PageTab } from 'components/types';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { centeredDiv } from 'styles';
import { AutocompleteOption, NavigableObject, ShortcutOption, Wrap } from 'types';
import { getCurrentUser } from 'utils/authentication/session';
import { listToAutocomplete } from 'utils/display/listTools';
import { getUserLanguages } from 'utils/display/translationTools';
import { useDisplayApolloError } from 'utils/hooks/useDisplayApolloError';
import { useReactSearch } from 'utils/hooks/useReactSearch';
import { openObject } from 'utils/navigation/openObject';
import { actionsItems, shortcuts } from 'utils/navigation/quickActions';
import { HomeViewProps } from '../types';

enum TabOptions {
    ForYou = "ForYou",
    History = "History",
}

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

    // Converts resources to a resource list
    const [resourceList, setResourceList] = useState<ResourceList>({
        __typename: 'ResourceList',
        created_at: 0,
        updated_at: 0,
        id: DUMMY_ID,
        resources: [],
        translations: [],
    });
    useEffect(() => {
        if (data?.home?.resources) {
            setResourceList({
                __typename: 'ResourceList',
                created_at: 0,
                updated_at: 0,
                id: DUMMY_ID,
                resources: data.home.resources,
                translations: [],
            });
        }
    }, [data]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => ([{
        index: 0,
        href: LINKS.Home,
        label: t('ForYou'),
        value: TabOptions.ForYou,
    }, {
        index: 1,
        href: LINKS.History,
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
                id: LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: 'Help - FAQ',
                id: LINKS.FAQ,
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
        if (newValue.__typename !== 'Shortcut' && searchString) setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(LINKS.Home, { replace: true });
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
     * Replaces current state with search string, so that search is not lost
     */
    const beforeNavigation = useCallback((item: NavigableObject) => {
        // Replace current state with search string, so that search is not lost
        if (searchString) setLocation(LINKS.Home, { replace: true, searchParams: { search: searchString } });
    }, [searchString, setLocation]);

    const openSchedule = useCallback(() => {
        setLocation(LINKS.Calendar);
    }, [setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
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
                <HomePrompt />
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
            </Stack>
            {/* Result feeds */}
            <Stack spacing={10} direction="column" mt={10}>
                {/* Resources */}
                <ResourceListHorizontal
                    list={resourceList}
                    session={session}
                    canUpdate={true}
                    handleUpdate={setResourceList}
                    loading={loading}
                    mutate={true}
                    zIndex={zIndex}
                />
                {/* Events */}
                <TitleContainer
                    titleKey="Schedule"
                    options={[['Open', openSchedule]]}
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