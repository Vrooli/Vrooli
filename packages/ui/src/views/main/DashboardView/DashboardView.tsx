import { useQuery } from "@apollo/client";
import { AddIcon, calculateOccurrences, DUMMY_ID, feedHome, FocusMode, FocusModeStopCondition, HomeInput, HomeResult, LINKS, MonthIcon, Note, NoteIcon, NoteVersion, OpenInNewIcon, Reminder, ResourceList, useLocation, uuid } from "@local/shared";
import { Stack } from "@mui/material";
import { ListTitleContainer } from "components/containers/ListTitleContainer/ListTitleContainer";
import { PageContainer } from "components/containers/PageContainer/PageContainer";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "components/inputs/search";
import { ReminderList } from "components/lists/reminder";
import { ResourceListHorizontal } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { HomePrompt } from "components/text/HomePrompt/HomePrompt";
import { PageTab } from "components/types";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { centeredDiv } from "styles";
import { AutocompleteOption, CalendarEvent, ShortcutOption, Wrap } from "types";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { getDisplay, listToAutocomplete, listToListItems } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useReactSearch } from "utils/hooks/useReactSearch";
import { openObject } from "utils/navigation/openObject";
import { actionsItems, shortcuts } from "utils/navigation/quickActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { NoteUpsert } from "views/objects/note";
import { DashboardViewProps } from "../types";

const zIndex = 200;

/**
 * View displayed for Home page when logged in
 */
export const DashboardView = ({
    display = "page",
    onClose,
    zIndex,
}: DashboardViewProps) => {
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
        const match = tabs.find(tab => tab.value.id === activeFocusMode?.mode?.id);
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e: any, tab: PageTab<FocusMode>) => {
        e.preventDefault();
        PubSub.get().publishFocusMode({
            __typename: "ActiveFocusMode" as const,
            mode: tab.value,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    }, []);

    const [searchString, setSearchString] = useState<string>("");
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === "string") setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue); }, []);
    const { data, refetch, loading, errors } = useQuery<Wrap<HomeResult, "home">, Wrap<HomeInput, "input">>(feedHome, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, "") } }, errorPolicy: "all" });
    useEffect(() => { refetch(); }, [refetch, searchString, activeFocusMode]);
    useDisplayServerError(errors);

    // Only show tabs if:
    // 1. The user is logged in 
    // 2. The user has at least two focusModes
    const showTabs = useMemo(() => Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [session, allFocusModes.length, currTab]);

    // Converts resources to a resource list
    const [resourceList, setResourceList] = useState<ResourceList>({
        __typename: "ResourceList",
        created_at: 0,
        updated_at: 0,
        id: DUMMY_ID,
        resources: [],
        translations: [],
    });
    useEffect(() => {
        if (data?.home?.resources) {
            setResourceList(r => ({ ...r, resources: data.home.resources }));
        }
    }, [data]);
    useEffect(() => {
        // Resources are added to the focus mode's resource list
        if (activeFocusMode?.mode?.resourceList?.id && activeFocusMode.mode?.resourceList.id !== DUMMY_ID) {
            setResourceList(activeFocusMode!.mode?.resourceList);
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
        if (searchString.toLowerCase().startsWith("help")) {
            firstResults.push({
                __typename: "Shortcut",
                label: "Help - Beginner's Guide",
                id: LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: "Help - FAQ",
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
        if (newValue.__typename === "Action") {
            return;
        }
        // Replace current state with search string, so that search is not lost. 
        // Only do this if the selected item is not a shortcut
        if (newValue.__typename !== "Shortcut" && searchString) setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(LINKS.Home, { replace: true });
        // If selected item is a shortcut, navigate to it
        if (newValue.__typename === "Shortcut") {
            setLocation(newValue.id);
        }
        // Otherwise, navigate to item page
        else {
            openObject(newValue, setLocation);
        }
    }, [searchString, setLocation]);

    const openSchedule = useCallback(() => {
        setLocation(LINKS.Calendar);
    }, [setLocation]);

    const onClick = useCallback(() => {
        if (searchString) setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
    }, [searchString, setLocation]);

    const [reminders, setReminders] = useState<Reminder[]>([]);
    useEffect(() => {
        if (data?.home?.reminders) {
            setReminders(data.home.reminders);
        }
    }, [data]);
    const handleReminderUpdate = useCallback((updatedReminders: Reminder[]) => {
        setReminders(updatedReminders);
    }, []);

    const reminderListId = useMemo(() => {
        // First, try to find list using foccus mode
        const sessionReminderListId = activeFocusMode?.mode?.reminderList?.id;
        // If that doesn't work, try to find list using the reminders and hope for the best
        const reminderList = reminders.length > 0 ? reminders[0].reminderList.id : null;
        return sessionReminderListId ?? reminderList ?? "";
    }, [activeFocusMode, reminders]);

    const [notes, setNotes] = useState<Note[]>([]);
    useEffect(() => {
        if (data?.home?.notes) {
            setNotes(data.home.notes);
        }
    }, [data]);

    const noteItems = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill("Note"),
        items: notes,
        keyPrefix: "note-list-item",
        loading,
        onClick,
        zIndex,
    }), [onClick, notes, loading]);

    const [isCreateNoteOpen, setIsCreateNoteOpen] = useState(false);
    const openCreateNote = useCallback(() => { setIsCreateNoteOpen(true); }, []);
    const closeCreateNote = useCallback(() => { setIsCreateNoteOpen(false); }, []);
    const onNoteCreated = useCallback((note: NoteVersion) => {
        closeCreateNote();
        // Convert NoteVersion to Note before adding to list
        const { root, ...rest } = note;
        const asRoot = { ...root, versions: [note] };
        setNotes(n => [asRoot, ...n]);
    }, []);

    // Calculate upcoming events using schedules 
    const upcomingEvents = useMemo(() => {
        const schedules = data?.home?.schedules ?? [];
        // Initialize result
        const result: CalendarEvent[] = [];
        // Loop through schedules
        schedules.forEach((schedule: any) => {
            // Get occurrences in the upcoming 30 days
            const occurrences = calculateOccurrences(schedule, new Date(), new Date(Date.now() + 30 * 24 * 60 * 60 * 1000));
            // Create events
            const events: CalendarEvent[] = occurrences.map(occurrence => ({
                __typename: "CalendarEvent",
                id: uuid(),
                title: getDisplay(schedule, getUserLanguages(session)).title,
                start: occurrence.start,
                end: occurrence.end,
                allDay: false,
                schedule,
            }));
            // Add events to result
            result.push(...events);
        });
        // Sort events by start date, and return the first 10
        result.sort((a, b) => a.start.getTime() - b.start.getTime());
        const first10 = result.slice(0, 10);
        // Convert to list items
        return listToListItems({
            dummyItems: new Array(5).fill("Event"),
            items: first10,
            keyPrefix: "event-list-item",
            loading,
            onClick,
            zIndex,
        });
    }, [onClick, data?.home?.schedules, loading, session]);

    return (
        <PageContainer>
            {/* Create note dialog */}
            <LargeDialog
                id="add-note-dialog"
                onClose={closeCreateNote}
                isOpen={isCreateNoteOpen}
                zIndex={zIndex + 1}
            >
                <NoteUpsert
                    display="dialog"
                    isCreate={true}
                    onCancel={closeCreateNote}
                    onCompleted={onNoteCreated}
                    zIndex={zIndex + 1}
                />
            </LargeDialog>
            <TopBar
                display={display}
                onClose={onClose}
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
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: { xs: "5vh", sm: "20vh" } }}>
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
                    sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                    zIndex={zIndex}
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
                <ListTitleContainer
                    Icon={MonthIcon}
                    isEmpty={upcomingEvents.length === 0}
                    titleKey="Schedule"
                    options={[{
                        Icon: OpenInNewIcon,
                        key: "Open",
                        onClick: openSchedule,
                    }]}
                >
                    {upcomingEvents}
                </ListTitleContainer>
                {/* Reminders */}
                <ReminderList
                    handleUpdate={handleReminderUpdate}
                    loading={loading}
                    listId={reminderListId}
                    reminders={reminders}
                    zIndex={zIndex}
                />
                {/* Notes */}
                <ListTitleContainer
                    Icon={NoteIcon}
                    isEmpty={noteItems.length === 0}
                    titleKey="Note"
                    titleVariables={{ count: 2 }}
                    options={[{
                        Icon: AddIcon,
                        key: "Create",
                        onClick: openCreateNote,
                    }]}
                >
                    {noteItems}
                </ListTitleContainer>
            </Stack>
        </PageContainer>
    );
};
