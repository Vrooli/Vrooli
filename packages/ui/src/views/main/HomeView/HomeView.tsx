import { useQuery } from '@apollo/client';
import { Stack } from '@mui/material';
import { FocusMode, FocusModeStopCondition, HomeInput, HomeResult, LINKS, ResourceList } from '@shared/consts';
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
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { centeredDiv } from 'styles';
import { AutocompleteOption, NavigableObject, ShortcutOption, Wrap } from 'types';
import { getCurrentUser, getFocusModeInfo } from 'utils/authentication/session';
import { listToAutocomplete } from 'utils/display/listTools';
import { getUserLanguages } from 'utils/display/translationTools';
import { useDisplayApolloError } from 'utils/hooks/useDisplayApolloError';
import { useReactSearch } from 'utils/hooks/useReactSearch';
import { openObject } from 'utils/navigation/openObject';
import { actionsItems, shortcuts } from 'utils/navigation/quickActions';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { HomeViewProps } from '../types';

const zIndex = 200;

export const HomeView = ({
    display = 'page',
}: HomeViewProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Handle focus modes
    const { active: activeFocusMode, all: allFocusModes } = useMemo(() => getFocusModeInfo(session), [session]);

    // Handle tabs
    const tabs = useMemo<PageTab<FocusMode>[]>(() => allFocusModes.map((mode, index) => ({
        index,
        label: mode.name,
        value: mode,
    })), [allFocusModes]);
    const currTab = useMemo(() => {
        const match = tabs.find(tab => tab.value.id === activeFocusMode?.mode?.id)
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e: any, tab: PageTab<FocusMode>) => {
        e.preventDefault();
        PubSub.get().publishFocusMode({
            __typename: 'ActiveFocusMode' as const,
            mode: tab.value,
            stopCondition: FocusModeStopCondition.Automatic,
        });
    }, []);

    const [searchString, setSearchString] = useState<string>('');
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const { data, refetch, loading, error } = useQuery<Wrap<HomeResult, 'home'>, Wrap<HomeInput, 'input'>>(feedHome, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch, searchString, activeFocusMode]);
    useDisplayApolloError(error);

    // Only show tabs if:
    // 1. The user is logged in 
    // 2. The user has at least two focusModes
    const showTabs = useMemo(() => Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [session, allFocusModes.length, currTab]);

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
        console.log('setting resources?', data)
        if (data?.home?.resources) {
            setResourceList(r => ({ ...r, resources: data.home.resources }));
        }
    }, [data]);
    useEffect(() => {
        // Resources are added to the focus mode's resource list
        if (activeFocusMode?.mode?.reminderList?.id && activeFocusMode.mode?.reminderList.id !== DUMMY_ID) {
            setResourceList(r => ({ ...r, id: activeFocusMode!.mode?.reminderList!.id }));
        }
    }, [activeFocusMode]);

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
                // Navigate between for you and history pages
                below={showTabs && (
                    <PageTabs
                        ariaLabel="home-tabs"
                        currTab={currTab!}
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
                    showSecondaryLabel={true}
                    sxs={{ root: { width: 'min(100%, 600px)', paddingLeft: 2, paddingRight: 2 } }}
                />
            </Stack>
            {/* Result feeds */}
            <Stack spacing={10} direction="column" mt={10}>
                {/* Resources */}
                <ResourceListHorizontal
                    list={resourceList}
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