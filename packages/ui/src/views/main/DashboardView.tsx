import { calculateOccurrences, CalendarEvent, Chat, ChatCreateInput, ChatParticipantShape, ChatShape, ChatUpdateInput, DAYS_30_MS, deleteArrayIndex, DUMMY_ID, endpointsChat, endpointsFeed, FindByIdInput, FocusMode, FocusModeStopCondition, HomeResult, LINKS, MyStuffPageTabOption, Reminder, Resource, ResourceList as ResourceListType, Schedule, SEEDED_IDS, updateArray, uuid } from "@local/shared";
import { Box, Button, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper.js";
import { ServerResponseParser } from "api/responseParser.js";
import { ChatBubbleTree } from "components/ChatBubbleTree/ChatBubbleTree.js";
import { ListTitleContainer } from "components/containers/ListTitleContainer/ListTitleContainer.js";
import { ChatMessageInput } from "components/inputs/ChatMessageInput/ChatMessageInput.js";
import { FocusModeInfo, useFocusModesStore } from "components/inputs/FocusModeSelector/FocusModeSelector.js";
import { ObjectList } from "components/lists/ObjectList/ObjectList.js";
import { ResourceList } from "components/lists/ResourceList/ResourceList.js";
import { ObjectListActions } from "components/lists/types.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { PageTabs } from "components/PageTabs/PageTabs.js";
import { useMessageActions, useMessageInput, useMessageTree } from "hooks/messages.js";
import { useChatTasks } from "hooks/tasks.js";
import { useHistoryState } from "hooks/useHistoryState.js";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { useSocketChat } from "hooks/useSocketChat.js";
import { PageTab } from "hooks/useTabs.js";
import { useUpsertFetch } from "hooks/useUpsertFetch.js";
import { useWindowSize } from "hooks/useWindowSize.js";
import { AddIcon, ChevronLeftIcon, MonthIcon, OpenInNewIcon, ReminderIcon } from "icons/common.js";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session.js";
import { DUMMY_LIST_LENGTH, ELEMENT_IDS } from "utils/consts.js";
import { getDisplay } from "utils/display/listTools.js";
import { getUserLanguages } from "utils/display/translationTools.js";
import { getCookieMatchingChat, setCookieMatchingChat } from "utils/localStorage.js";
import { TabParamPayload } from "utils/search/objectToSearch.js";
import { CHAT_DEFAULTS, chatInitialValues, transformChatValues, withModifiableMessages, withYourMessages } from "views/objects/chat/ChatCrud.js";
import { SessionContext } from "../../contexts.js";
import { DashboardViewProps } from "./types.js";

const MAX_EVENTS_SHOWN = 10;
const MESSAGE_LIST_ID = "dashboardMessage";

type DashboardTabsInfo = {
    Key: string; // "All" of focus mode's ID
    Payload: FocusMode | undefined;
    WhereParams: undefined;
}

const DashboardBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    maxHeight: "100vh",
    height: "calc(100vh - env(safe-area-inset-bottom))",
    overflow: "hidden",
}));

const DashboardInnerBox = styled(Box)(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    flexGrow: 1,
    gap: 2,
    margin: "auto",
    paddingBottom: 4,
    overflowY: "auto",
    width: "min(700px, 100%)",
}));

const ChatViewOptionsBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "space-around",
    alignItems: "center",
    maxWidth: "min(100vw, 1000px)",
    margin: "auto",
    background: "none",
}));

const NewChatButton = styled(Button)(({ theme }) => ({
    margin: theme.spacing(1),
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(8),
    padding: "4px 8px",
}));

const pageTabsStyle = { width: "min(700px, 100%)", minWidth: undefined, margin: "auto" } as const;
const exitChatButtonStyle = { margin: 1, borderRadius: 8, padding: "4px 8px" } as const;
const resourceListStyle = { list: { justifyContent: "flex-start" } } as const;

const activeFocusModeFallback = { __typename: "FocusMode", id: DUMMY_ID } as const;
const resourceListFallback = {
    __typename: "ResourceList",
    created_at: 0,
    updated_at: 0,
    id: DUMMY_ID,
    resources: [],
    translations: [],
} as unknown as ResourceListType;

/** View displayed for Home page when logged in */
export function DashboardView({
    display,
    isOpen,
    onClose,
}: DashboardViewProps) {
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [participants, setParticipants] = useState<Omit<ChatParticipantShape, "chat">[]>([]);
    const [usersTyping, setUsersTyping] = useState<Omit<ChatParticipantShape, "chat">[]>([]);
    const [refetch, { data: feedData, loading: isFeedLoading }] = useLazyFetch<Record<string, never>, HomeResult>(endpointsFeed.home);

    const {
        fetchCreate,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Chat, ChatCreateInput, ChatUpdateInput>({
        isCreate: false, // We create chats automatically, so this should always be false
        isMutate: true,
        endpointCreate: endpointsChat.createOne,
        endpointUpdate: endpointsChat.updateOne,
    });

    const [getChat, { data: loadedChat, loading: isChatLoading }] = useLazyFetch<FindByIdInput, Chat>(endpointsChat.findOne);
    const [chat, setChat] = useState<ChatShape>(chatInitialValues(session, t, languages[0], CHAT_DEFAULTS));
    useEffect(() => {
        if (loadedChat?.participants) {
            setParticipants(loadedChat.participants);
        }
    }, [loadedChat?.participants]);
    // When a chat is loaded, store chat ID by participant and task
    useEffect(() => {
        if (!loadedChat?.id) return;
        const userId = getCurrentUser(session).id;
        if (!userId) return;
        setCookieMatchingChat(loadedChat.id, [userId, SEEDED_IDS.User.Valyxa]);
    }, [loadedChat, loadedChat?.id, session]);

    const createChat = useCallback((resetChat = false) => {
        chatCreateStatus.current = "inProgress";
        const chatToUse = resetChat ? chatInitialValues(session, t, languages[0], CHAT_DEFAULTS) : chat;
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: fetchCreate,
            inputs: transformChatValues(withModifiableMessages(chatToUse, session), withYourMessages(chatToUse, session), true),
            onSuccess: (data) => {
                setChat({ ...data, messages: [] });
                setUsersTyping([]);
                const userId = getCurrentUser(session).id;
                if (userId) setCookieMatchingChat(data.id, [userId, SEEDED_IDS.User.Valyxa]);
            },
            onCompleted: () => {
                chatCreateStatus.current = "complete";
            },
        });
    }, [chat, languages, fetchCreate, session, t]);
    const resetChat = useCallback(() => { createChat(true); }, [createChat]);

    const chatCreateStatus = useRef<"notStarted" | "inProgress" | "complete">("notStarted");
    useEffect(function autoCreateMatchingChatEffect() {
        const userId = getCurrentUser(session).id;
        if (!userId) return;
        // Unlike the chat view, we look for the chat by local storage data rather than the ID in the URL
        const existingChatId = getCookieMatchingChat([userId, SEEDED_IDS.User.Valyxa]);
        const isChatValid = chat.id !== DUMMY_ID && chat.participants?.every(p => [userId, SEEDED_IDS.User.Valyxa].includes(p.user.id));
        if (chat.id === DUMMY_ID && existingChatId) {
            fetchLazyWrapper<FindByIdInput, Chat>({
                fetch: getChat,
                inputs: { id: existingChatId },
                onSuccess: (data) => {
                    setChat({ ...data, messages: [] });
                },
                onError: (response) => {
                    if (ServerResponseParser.hasErrorCode(response, "NotFound")) {
                        createChat();
                    }
                },
            });
        }
        else if (!isChatValid && chatCreateStatus.current === "notStarted") {
            createChat();
        }
    }, [chat, createChat, display, fetchCreate, getChat, isOpen, session, setLocation]);

    // Handle focus modes
    const getFocusModeInfo = useFocusModesStore(state => state.getFocusModeInfo);
    const putActiveFocusMode = useFocusModesStore(state => state.putActiveFocusMode);
    const [focusModeInfo, setFocusModeInfo] = useState<FocusModeInfo>({ active: null, all: [] });
    useEffect(function fetchFocusModeInfoEffect() {
        const abortController = new AbortController();

        async function fetchFocusModeInfo() {
            const info = await getFocusModeInfo(session, abortController.signal);
            setFocusModeInfo(info);
        }

        fetchFocusModeInfo();

        return () => {
            abortController.abort();
        };
    }, [getFocusModeInfo, session]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabParamPayload<DashboardTabsInfo>>[]>(() => {
        const activeColor = isMobile ? palette.primary.contrastText : palette.background.textPrimary;
        const inactiveColor = isMobile ? palette.primary.contrastText : palette.background.textSecondary;
        return [
            ...focusModeInfo.all.map((mode, index) => ({
                color: mode.id === focusModeInfo.active?.focusMode?.id ? activeColor : inactiveColor,
                data: mode,
                index,
                key: mode.id,
                label: mode.name,
                searchPlaceholder: "",
            })),
            {
                color: inactiveColor,
                data: undefined,
                Icon: AddIcon,
                index: focusModeInfo.all.length,
                key: "Add",
                label: "Add",
                searchPlaceholder: "",
            },
        ];
    }, [focusModeInfo.active?.focusMode?.id, focusModeInfo.all, isMobile, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText]);
    const currTab = useMemo(function calculateCurrTabMemo() {
        const match = tabs.find(tab => typeof tab.data === "object" && tab.data.id === focusModeInfo.active?.focusMode?.id);
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, focusModeInfo.active]);
    const handleTabChange = useCallback((e: any, tab: PageTab<TabParamPayload<DashboardTabsInfo>>) => {
        e.preventDefault();
        // If "Add", go to the add focus mode page
        if (tab.key === "Add" || !tab.data) {
            setLocation(LINKS.SettingsFocusModes);
            return;
        }
        // Otherwise, publish the focus mode
        putActiveFocusMode({
            __typename: "ActiveFocusMode" as const,
            focusMode: {
                ...tab.data,
                __typename: "ActiveFocusModeFocusMode" as const,
            },
            stopCondition: FocusModeStopCondition.NextBegins,
        }, session);
    }, [putActiveFocusMode, session, setLocation]);
    useEffect(() => {
        refetch({});
    }, [focusModeInfo.active, refetch]);

    const [resourceList, setResourceList] = useState<ResourceListType>(resourceListFallback);
    const feedResourcesRef = useRef<Resource[]>([]);

    const [reminders, setReminders] = useState<Reminder[]>([]);
    const feedRemindersRef = useRef<Reminder[]>([]);

    const [schedules, setSchedules] = useState<Schedule[]>([]);
    const feedSchedulesRef = useRef<Schedule[]>([]);

    useEffect(function parseFeedData() {
        const feedResources = feedData?.resources;
        if (feedResources) {
            feedResourcesRef.current = feedResources;
            setResourceList(r => {
                // Add feed resources without duplicates
                const newResources = feedResources.filter(resource => !r.resources.some(r => r.id === resource.id));
                return {
                    ...r,
                    resources: [...r.resources, ...newResources],
                };
            });
        }

        const feedReminders = feedData?.reminders;
        if (feedReminders) {
            feedRemindersRef.current = feedReminders;
            setReminders(feedReminders);
        }

        const feedSchedules = feedData?.schedules;
        if (feedSchedules) {
            feedSchedulesRef.current = feedSchedules;
            setSchedules(feedSchedules);
        }
    }, [feedData?.reminders, feedData?.resources, feedData?.schedules]);

    useEffect(function setActiveFocusModeData() {
        const activeFocusModeId = focusModeInfo.active?.focusMode?.id;
        const fullActiveFocusModeInfo = focusModeInfo.all.find(focusMode => focusMode.id === activeFocusModeId);
        // If no active focus mode set
        if (!fullActiveFocusModeInfo || !fullActiveFocusModeInfo.resourceList) {
            // Use feed resources only
            setResourceList({
                ...resourceListFallback,
                resources: feedResourcesRef.current,
            });
            // Use feed reminders only
            setReminders(feedRemindersRef.current);
        }
        // If active mode set
        else {
            // Override full resource list, while combining resources from feed without duplicates
            setResourceList(r => {
                // Add feed resources without duplicates
                const newResources = feedResourcesRef.current.filter(resource => !r.resources.some(r => r.id === resource.id));
                return {
                    ...fullActiveFocusModeInfo.resourceList,
                    __typename: "ResourceList" as const,
                    listFor: {
                        ...fullActiveFocusModeInfo,
                        __typename: "FocusMode" as const,
                        resourceList: undefined, // Avoid circular reference
                    },
                    resources: [...r.resources, ...newResources],
                } as ResourceListType;
            });
            // Combine feed reminders with active mode reminders without duplicates
            setReminders(r => {
                // Add feed reminders without duplicates
                const newReminders = feedRemindersRef.current.filter(reminder => !r.some(r => r.id === reminder.id));
                return [...r, ...newReminders];
            });
        }
    }, [focusModeInfo, focusModeInfo.active, focusModeInfo.all]);

    const openSchedule = useCallback(() => {
        setLocation(LINKS.Calendar);
    }, [setLocation]);

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

    const [upcomingEvents, setUpcomingEvents] = useState<CalendarEvent[]>([]);
    useEffect(() => {
        let isCancelled = false;

        async function fetchUpcomingEvents() {
            const result: CalendarEvent[] = [];
            for (const schedule of schedules) {
                const occurrences = await calculateOccurrences(
                    schedule,
                    new Date(),
                    new Date(Date.now() + DAYS_30_MS),
                );
                const events: CalendarEvent[] = occurrences.map(occurrence => ({
                    __typename: "CalendarEvent",
                    id: uuid(),
                    title: getDisplay(schedule, languages).title,
                    start: occurrence.start,
                    end: occurrence.end,
                    allDay: false,
                    schedule,
                }));
                if (!isCancelled) {
                    result.push(...events);
                }
            }
            if (!isCancelled) {
                // Sort events by start date, and set the first 10
                result.sort((a, b) => a.start.getTime() - b.start.getTime());
                setUpcomingEvents(result.slice(0, MAX_EVENTS_SHOWN));
            }
        }

        fetchUpcomingEvents();

        return () => {
            isCancelled = true; // Cleanup function to avoid setting state on unmounted component
        };
    }, [languages, schedules]);
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

    const [view, setView] = useState<"chat" | "home">("home");
    const showChat = useCallback(function showChatCallback() {
        setView("chat");
    }, []);
    const showHome = useCallback(function showHomeCallback() {
        setView("home");
    }, []);

    const showTabs = useMemo(() => view === "home" && Boolean(getCurrentUser(session).id) && focusModeInfo.all.length > 1 && currTab !== null, [session, focusModeInfo.all.length, currTab, view]);

    const [message, setMessage] = useHistoryState<string>(`${MESSAGE_LIST_ID}-message`, "");
    const messageTree = useMessageTree(chat.id);
    const taskInfo = useChatTasks({ chatId: chat.id });
    const isBotOnlyChat = chat?.participants?.every(p => p.user?.isBot || p.user?.id === getCurrentUser(session).id) ?? false;
    const messageActions = useMessageActions({
        activeTask: taskInfo.activeTask,
        addMessages: messageTree.addMessages,
        chat,
        contexts: taskInfo.contexts[taskInfo.activeTask.taskId] || [],
        editMessage: messageTree.editMessage,
        isBotOnlyChat,
        language: languages[0],
        setMessage,
    });
    const messageInput = useMessageInput({
        id: MESSAGE_LIST_ID,
        languages,
        message,
        postMessage: messageActions.postMessage,
        putMessage: messageActions.putMessage,
        replyToMessage: messageActions.replyToMessage,
        setMessage,
    });
    const { messageStream } = useSocketChat({
        addMessages: messageTree.addMessages,
        chat,
        editMessage: messageTree.editMessage,
        participants,
        removeMessages: messageTree.removeMessages,
        setParticipants,
        setUsersTyping,
        usersTyping,
    });

    const scheduleContainerOptions = useMemo(function scheduleContainerOptionsMemo() {
        return [{
            Icon: OpenInNewIcon,
            label: t("Open"),
            onClick: openSchedule,
        }];
    }, [openSchedule, t]);

    const reminderContainerOptions = useMemo(function reminderContainerOptionsMemo() {
        return [{
            Icon: OpenInNewIcon,
            label: t("SeeAll"),
            onClick: () => { setLocation(`${LINKS.MyStuff}?type="${MyStuffPageTabOption.Reminder}"`); },
        }, {
            Icon: AddIcon,
            label: t("Create"),
            onClick: () => { setLocation(`${LINKS.Reminder}/add`); },
        }];
    }, [setLocation, t]);

    return (
        <DashboardBox>
            {/* Main content */}
            <TopBar
                display={display}
                onClose={onClose}
                below={<>
                    {showTabs && currTab && <PageTabs<TabParamPayload<DashboardTabsInfo>[]>
                        ariaLabel="Focus modes"
                        id={ELEMENT_IDS.DashboardFocusModeTabs}
                        currTab={currTab as PageTab<TabParamPayload<DashboardTabsInfo>>}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                        sx={pageTabsStyle}
                    />}
                    {view === "chat" && <ChatViewOptionsBox>
                        <Button
                            color="primary"
                            onClick={showHome}
                            variant="contained"
                            sx={exitChatButtonStyle}
                            startIcon={<ChevronLeftIcon />}
                        >
                            {t("Dashboard")}
                        </Button>
                        <NewChatButton
                            color="primary"
                            onClick={resetChat}
                            variant="contained"
                            startIcon={<AddIcon />}
                        >
                            {t("NewChat")}
                        </NewChatButton>
                    </ChatViewOptionsBox>}
                </>}
            />
            <DashboardInnerBox>
                {/* TODO for morning: work on changes needed for a chat to track active and inactive tasks. Might need to link them to their own reminder list */}
                {view === "home" && <>
                    <Box p={1}>
                        <ResourceList
                            id={ELEMENT_IDS.DashboardResourceList}
                            list={resourceList}
                            canUpdate={true}
                            handleUpdate={setResourceList}
                            horizontal
                            loading={isFeedLoading}
                            mutate={true}
                            parent={focusModeInfo.active?.focusMode ?
                                {
                                    ...focusModeInfo.active.focusMode,
                                    __typename: "FocusMode" as const,
                                    resourceList: undefined, // Avoid circular reference
                                } as FocusMode
                                : activeFocusModeFallback
                            }
                            title={t("Resource", { count: 2 })}
                            sxs={resourceListStyle}
                        />
                    </Box>
                    {/* Events */}
                    <ListTitleContainer
                        Icon={MonthIcon}
                        id={ELEMENT_IDS.DashboardEventList}
                        isEmpty={upcomingEvents.length === 0 && !isFeedLoading}
                        emptyText="No upcoming events"
                        loading={isFeedLoading}
                        title={t("Schedule", { count: 1 })}
                        options={scheduleContainerOptions}
                    >
                        <ObjectList
                            dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Event")}
                            items={upcomingEvents}
                            keyPrefix="event-list-item"
                            loading={isFeedLoading}
                            onAction={onEventAction}
                        />
                    </ListTitleContainer>
                    {/* Reminders */}
                    <ListTitleContainer
                        Icon={ReminderIcon}
                        id={ELEMENT_IDS.DashboardReminderList}
                        isEmpty={reminders.length === 0 && !isFeedLoading}
                        emptyText="No reminders"
                        loading={isFeedLoading}
                        title={t("Reminder", { count: 2 })}
                        options={reminderContainerOptions}
                    >
                        <ObjectList
                            dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Reminder")}
                            items={reminders}
                            keyPrefix="reminder-list-item"
                            loading={isFeedLoading}
                            onAction={onReminderAction}
                        />
                    </ListTitleContainer>
                </>}
                {view === "chat" && <ChatBubbleTree
                    branches={messageTree.branches}
                    handleEdit={messageInput.startEditingMessage}
                    handleRegenerateResponse={messageActions.regenerateResponse}
                    handleReply={messageInput.startReplyingToMessage}
                    handleRetry={messageActions.retryPostMessage}
                    isBotOnlyChat={isBotOnlyChat}
                    isEditingMessage={Boolean(messageInput.messageBeingEdited)}
                    isReplyingToMessage={Boolean(messageInput.messageBeingRepliedTo)}
                    messageStream={messageStream}
                    removeMessages={messageTree.removeMessages}
                    setBranches={messageTree.setBranches}
                    tree={messageTree.tree}
                />}
            </DashboardInnerBox>
            <ChatMessageInput
                disabled={!chat}
                display={display}
                isLoading={isCreateLoading || isUpdateLoading || isChatLoading}
                message={message}
                messageBeingEdited={messageInput.messageBeingEdited}
                messageBeingRepliedTo={messageInput.messageBeingRepliedTo}
                onFocused={showChat}
                participantsAll={participants}
                participantsTyping={usersTyping}
                messagesCount={messageTree.messagesCount}
                placeholder={t("WhatWouldYouLikeToDo")}
                setMessage={setMessage}
                stopEditingMessage={messageInput.stopEditingMessage}
                stopReplyingToMessage={messageInput.stopReplyingToMessage}
                submitMessage={messageInput.submitMessage}
                taskInfo={taskInfo}
            />
        </DashboardBox>
    );
}
