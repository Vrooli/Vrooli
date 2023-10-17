import { calculateOccurrences, DUMMY_ID, endpointGetFeedHome, FocusMode, FocusModeStopCondition, HomeResult, LINKS, Reminder, ResourceList, Schedule, uuid } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
import { ListTitleContainer } from "components/containers/ListTitleContainer/ListTitleContainer";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ResourceListHorizontal } from "components/lists/resource";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Resizable, useDimensionContext } from "components/Resizable/Resizable";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { PageTab } from "hooks/useTabs";
import { AddIcon, ListIcon, MonthIcon, OpenInNewIcon, ReminderIcon, SearchIcon } from "icons";
import { Dispatch, SetStateAction, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { CalendarEvent, ShortcutOption } from "types";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { shortcuts } from "utils/navigation/quickActions";
import { PubSub } from "utils/pubsub";
import { MyStuffPageTabOption } from "utils/search/objectToSearch";
import { deleteArrayIndex, updateArray } from "utils/shape/general";
import { DashboardViewProps } from "../types";

const NewMessageContainer = ({
    message,
    handleSubmit,
    setMessage,
}: {
    message: string,
    setMessage: Dispatch<SetStateAction<string>>,
    handleSubmit: (text: string) => unknown,
}) => {
    const { t } = useTranslation();
    const dimensions = useDimensionContext();

    return (
        <RichInputBase
            actionButtons={[{
                Icon: SearchIcon, // Should be SearchIcon by default. But if search result is focused, then can change to that item's icon
                onClick: () => {
                    //TODO
                },
            }]}
            disableAssistant={true}
            fullWidth
            getTaggableItems={async (message) => {
                // TODO should be able to tag any public or owned object (e.g. "Create routine like @some_existing_routine, but change a to b")
                return [];
            }}
            maxChars={1500}
            minRows={4}
            maxRows={15}
            name="search"
            onChange={setMessage}
            placeholder={t("WhatWouldYouLikeToDo")}
            sxs={{
                root: {
                    height: dimensions.height,
                    width: "100%",
                    // When BottomNav is shown, need to make room for it
                    paddingBottom: { xs: pagePaddingBottom, md: 0 },
                    willChange: "height",
                },
                bar: { borderRadius: 0 },
                textArea: { paddingRight: 4, border: "none", height: "100%" },
            }}
            value={message}
        />
    );
};

/** View displayed for Home page when logged in */
export const DashboardView = ({
    isOpen,
    onClose,
}: DashboardViewProps) => {
    const { palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const [message, setMessage] = useState<string>("");
    const [refetch, { data, loading }] = useLazyFetch<any, HomeResult>(endpointGetFeedHome);

    // Handle focus modes
    const { active: activeFocusMode, all: allFocusModes } = useMemo(() => getFocusModeInfo(session), [session]);

    // Handle tabs
    const tabs = useMemo<PageTab<FocusMode, false>[]>(() => allFocusModes.map((mode, index) => ({
        index,
        label: mode.name,
        tabType: mode,
    })), [allFocusModes]);
    const currTab = useMemo(() => {
        const match = tabs.find(tab => tab.tabType.id === activeFocusMode?.mode?.id);
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e: any, tab: PageTab<FocusMode, false>) => {
        e.preventDefault();
        PubSub.get().publishFocusMode({
            __typename: "ActiveFocusMode" as const,
            mode: tab.tabType,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    }, []);
    useEffect(() => {
        refetch({ activeFocusModeId: activeFocusMode?.mode?.id });
    }, [activeFocusMode, refetch]);


    /** Only show tabs if:
    * 1. The user is logged in 
    * 2. The user has at least two focusModes
    **/
    const showTabs = useMemo(() => Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [session, allFocusModes.length, currTab]);

    // Converts resources to a resource list
    const [resourceList, setResourceList] = useState<ResourceList>({
        __typename: "ResourceList",
        created_at: 0,
        updated_at: 0,
        id: DUMMY_ID,
        resources: [],
        translations: [],
    } as any);
    useEffect(() => {
        if (data?.resources) {
            setResourceList(r => ({ ...r, resources: data.resources }));
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

    const openSchedule = useCallback(() => {
        setLocation(LINKS.Calendar);
    }, [setLocation]);

    const [reminders, setReminders] = useState<Reminder[]>([]);
    useEffect(() => {
        if (data?.reminders) {
            setReminders(data.reminders);
        }
    }, [data]);
    const onReminderAction = useCallback((action: keyof ObjectListActions<Reminder>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const id = data[0] as string;
                setReminders(curr => deleteArrayIndex(curr, curr.findIndex(item => item.id === id)));
                break;
            }
            case "Updated": {
                const updated = data[0] as Reminder;
                setReminders(curr => updateArray(curr, curr.findIndex(item => item.id === updated.id), updated));
                break;
            }
        }
    }, []);

    // Calculate upcoming events using schedules 
    const [schedules, setSchedules] = useState<Schedule[]>([]);
    useEffect(() => {
        if (data?.schedules) {
            setSchedules(data.schedules);
        }
    }, [data]);
    const upcomingEvents = useMemo(() => {
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
        return result.slice(0, 10);
    }, [schedules, session]);
    const onEventAction = useCallback((action: keyof ObjectListActions<CalendarEvent>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const eventId = data[0] as string;
                const event = upcomingEvents.find(event => event.id === eventId);
                if (!event) return;
                const schedule = event.schedule;
                setSchedules(curr => deleteArrayIndex(curr, curr.findIndex(item => item.id === schedule.id)));
                break;
            }
            case "Updated": {
                const updatedEvent = data[0] as CalendarEvent;
                const schedule = updatedEvent.schedule;
                setSchedules(curr => updateArray(curr, curr.findIndex(item => item.id === schedule.id), schedule));
                break;
            }
        }
    }, [upcomingEvents]);

    const openSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", isOpen: true }); }, []);
    const closeSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", isOpen: false }); }, []);
    useEffect(() => {
        return () => {
            closeSideMenu();
        };
    }, [closeSideMenu]);

    return (
        <>
            {/* Main content */}
            <TopBar
                display={display}
                onClose={onClose}
                startComponent={<IconButton
                    aria-label="Open chat menu"
                    onClick={openSideMenu}
                    sx={{
                        width: "48px",
                        height: "48px",
                        marginLeft: 1,
                        marginRight: 1,
                        cursor: "pointer",
                    }}
                >
                    <ListIcon fill={palette.primary.contrastText} width="100%" height="100%" />
                </IconButton>}
                // Navigate between for you and history pages
                below={showTabs && (
                    <PageTabs
                        ariaLabel="home-tabs"
                        id="home-tabs"
                        currTab={currTab!}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />
                )}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                margin: "auto",
                minHeight: "100vh",
                gap: 2,
            }}>
                {/* Resources */}
                <Box p={1}>
                    <ResourceListHorizontal
                        id="main-resource-list"
                        list={resourceList}
                        canUpdate={true}
                        handleUpdate={setResourceList}
                        loading={loading}
                        mutate={true}
                        parent={{ __typename: "FocusMode", id: activeFocusMode?.mode?.id ?? "" }}
                        title={t("Resource", { count: 2 })}
                    />
                </Box>
                {/* Events */}
                {upcomingEvents.length > 0 && <ListTitleContainer
                    Icon={MonthIcon}
                    id="main-event-list"
                    isEmpty={upcomingEvents.length === 0 && !loading}
                    title={t("Schedule", { count: 1 })}
                    options={[{
                        Icon: OpenInNewIcon,
                        label: t("Open"),
                        onClick: openSchedule,
                    }]}
                >
                    <ObjectList
                        dummyItems={new Array(5).fill("Event")}
                        items={upcomingEvents}
                        keyPrefix="event-list-item"
                        loading={loading}
                        onAction={onEventAction}
                    />
                </ListTitleContainer>}
                {/* Reminders */}
                {reminders.length > 0 && <ListTitleContainer
                    Icon={ReminderIcon}
                    id="main-reminder-list"
                    isEmpty={reminders.length === 0 && !loading}
                    title={t("Reminder", { count: 2 })}
                    options={[{
                        Icon: OpenInNewIcon,
                        label: t("SeeAll"),
                        onClick: () => { setLocation(`${LINKS.MyStuff}?type=${MyStuffPageTabOption.Reminder}`); },
                    }, {
                        Icon: AddIcon,
                        label: t("Create"),
                        onClick: () => { setLocation(`${LINKS.Reminder}/add`); },
                    }]}
                >
                    <ObjectList
                        dummyItems={new Array(5).fill("Reminder")}
                        items={reminders}
                        keyPrefix="reminder-list-item"
                        loading={loading}
                        onAction={onReminderAction}
                    />
                </ListTitleContainer>}
            </Box>
            <Resizable
                id="chat-message-input"
                min={150}
                max={"50vh"}
                position="top"
                sx={{
                    position: "sticky",
                    bottom: 0,
                    height: "min(50vh, 250px)",
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                }}>
                <NewMessageContainer
                    handleSubmit={() => { }}
                    message={message}
                    setMessage={setMessage}
                />
            </Resizable>
            <ChatSideMenu />
        </>
    );
};
