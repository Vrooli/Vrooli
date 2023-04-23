import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { FocusModeStopCondition, LINKS } from "@local/consts";
import { calculateOccurrences } from "@local/utils";
import { DUMMY_ID, uuid } from "@local/uuid";
import { Stack } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { feedHome } from "../../../api/generated/endpoints/feed_home";
import { ListTitleContainer } from "../../../components/containers/ListTitleContainer/ListTitleContainer";
import { PageContainer } from "../../../components/containers/PageContainer/PageContainer";
import { LargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "../../../components/inputs/search";
import { ReminderList } from "../../../components/lists/reminder";
import { ResourceListHorizontal } from "../../../components/lists/resource";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../../components/PageTabs/PageTabs";
import { HomePrompt } from "../../../components/text/HomePrompt/HomePrompt";
import { centeredDiv } from "../../../styles";
import { getCurrentUser, getFocusModeInfo } from "../../../utils/authentication/session";
import { getDisplay, listToAutocomplete, listToListItems } from "../../../utils/display/listTools";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { useDisplayApolloError } from "../../../utils/hooks/useDisplayApolloError";
import { useReactSearch } from "../../../utils/hooks/useReactSearch";
import { openObject } from "../../../utils/navigation/openObject";
import { actionsItems, shortcuts } from "../../../utils/navigation/quickActions";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { NoteUpsert } from "../../note";
const zIndex = 200;
export const DashboardView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { active: activeFocusMode, all: allFocusModes } = useMemo(() => getFocusModeInfo(session), [session]);
    const tabs = useMemo(() => allFocusModes.map((mode, index) => ({
        index,
        label: mode.name,
        value: mode,
    })), [allFocusModes]);
    const currTab = useMemo(() => {
        const match = tabs.find(tab => tab.value.id === activeFocusMode?.mode?.id);
        if (match)
            return match;
        if (tabs.length)
            return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        PubSub.get().publishFocusMode({
            __typename: "ActiveFocusMode",
            mode: tab.value,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    }, []);
    const [searchString, setSearchString] = useState("");
    const searchParams = useReactSearch();
    useEffect(() => {
        if (typeof searchParams.search === "string")
            setSearchString(searchParams.search);
    }, [searchParams]);
    const updateSearch = useCallback((newValue) => { setSearchString(newValue); }, []);
    const { data, refetch, loading, error } = useQuery(feedHome, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, "") } }, errorPolicy: "all" });
    useEffect(() => { refetch(); }, [refetch, searchString, activeFocusMode]);
    useDisplayApolloError(error);
    const showTabs = useMemo(() => Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [session, allFocusModes.length, currTab]);
    const [resourceList, setResourceList] = useState({
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
        if (activeFocusMode?.mode?.resourceList?.id && activeFocusMode.mode?.resourceList.id !== DUMMY_ID) {
            setResourceList(activeFocusMode.mode?.resourceList);
        }
    }, [activeFocusMode]);
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const shortcutsItems = useMemo(() => shortcuts.map(({ label, labelArgs, value }) => ({
        __typename: "Shortcut",
        label: t(label, { ...(labelArgs ?? {}), defaultValue: label }),
        id: value,
    })), [t]);
    const autocompleteOptions = useMemo(() => {
        const firstResults = [];
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
        const flattened = (Object.values(data?.home ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a, b) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [searchString, data?.home, languages, shortcutsItems]);
    const onInputSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        if (newValue.__typename === "Action") {
            return;
        }
        if (newValue.__typename !== "Shortcut" && searchString)
            setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
        else
            setLocation(LINKS.Home, { replace: true });
        if (newValue.__typename === "Shortcut") {
            setLocation(newValue.id);
        }
        else {
            openObject(newValue, setLocation);
        }
    }, [searchString, setLocation]);
    const openSchedule = useCallback(() => {
        setLocation(LINKS.Calendar);
    }, [setLocation]);
    const [isCreateNoteOpen, setIsCreateNoteOpen] = useState(false);
    const openCreateNote = useCallback(() => { setIsCreateNoteOpen(true); }, []);
    const closeCreateNote = useCallback(() => { setIsCreateNoteOpen(false); }, []);
    const onNoteCreated = useCallback((note) => {
    }, []);
    const beforeNavigation = useCallback(() => {
        if (searchString)
            setLocation(`${LINKS.Home}?search="${searchString}"`, { replace: true });
    }, [searchString, setLocation]);
    const notes = useMemo(() => listToListItems({
        beforeNavigation,
        dummyItems: new Array(5).fill("Note"),
        items: data?.home?.notes ?? [],
        keyPrefix: "note-list-item",
        loading,
        zIndex,
    }), [beforeNavigation, data?.home?.notes, loading]);
    const upcomingEvents = useMemo(() => {
        const schedules = data?.home?.schedules ?? [];
        const result = [];
        schedules.forEach((schedule) => {
            const occurrences = calculateOccurrences(schedule, new Date(), new Date(Date.now() + 30 * 24 * 60 * 60 * 1000));
            const events = occurrences.map(occurrence => ({
                __typename: "CalendarEvent",
                id: uuid(),
                title: getDisplay(schedule, getUserLanguages(session)).title,
                start: occurrence.start,
                end: occurrence.end,
                allDay: false,
                schedule,
            }));
            result.push(...events);
        });
        result.sort((a, b) => a.start.getTime() - b.start.getTime());
        const first10 = result.slice(0, 10);
        return listToListItems({
            beforeNavigation,
            dummyItems: new Array(5).fill("Event"),
            items: first10,
            keyPrefix: "event-list-item",
            loading,
            zIndex,
        });
    }, [beforeNavigation, data?.home?.schedules, loading, session]);
    const [reminders, setReminders] = useState([]);
    useEffect(() => {
        if (data?.home?.reminders) {
            setReminders(data.home.reminders);
        }
    }, [data]);
    const handleReminderUpdate = useCallback((updatedReminders) => {
        setReminders(updatedReminders);
    }, []);
    return (_jsxs(PageContainer, { children: [_jsx(LargeDialog, { id: "add-note-dialog", onClose: closeCreateNote, isOpen: isCreateNoteOpen, zIndex: 201, children: _jsx(NoteUpsert, { display: "dialog", isCreate: true, onCancel: closeCreateNote, onCompleted: onNoteCreated, zIndex: 201 }) }), _jsx(TopBar, { display: display, onClose: () => { }, below: showTabs && (_jsx(PageTabs, { ariaLabel: "home-tabs", currTab: currTab, fullWidth: true, onChange: handleTabChange, tabs: tabs })) }), _jsxs(Stack, { spacing: 2, direction: "column", sx: { ...centeredDiv, paddingTop: { xs: "5vh", sm: "20vh" } }, children: [_jsx(HomePrompt, {}), _jsx(SiteSearchBar, { id: "main-search", placeholder: 'SearchHome', options: autocompleteOptions, loading: loading, value: searchString, onChange: updateSearch, onInputChange: onInputSelect, showSecondaryLabel: true, sxs: { root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } } })] }), _jsxs(Stack, { spacing: 10, direction: "column", mt: 10, children: [_jsx(ResourceListHorizontal, { list: resourceList, canUpdate: true, handleUpdate: setResourceList, loading: loading, mutate: true, zIndex: zIndex }), _jsx(ListTitleContainer, { isEmpty: upcomingEvents.length === 0, titleKey: "Schedule", options: [["Open", openSchedule]], children: upcomingEvents }), _jsx(ReminderList, { handleUpdate: handleReminderUpdate, loading: loading, listId: data?.home?.reminders.find((r) => r.reminderList?.focusMode?.id === activeFocusMode?.mode?.id)?.reminderList?.id, reminders: reminders, zIndex: zIndex }), _jsx(ListTitleContainer, { isEmpty: notes.length === 0, titleKey: "Note", titleVariables: { count: 2 }, options: [["Create", openCreateNote]], children: notes })] })] }));
};
//# sourceMappingURL=DashboardView.js.map