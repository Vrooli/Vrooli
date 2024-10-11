import { calculateOccurrences, CalendarEvent, Chat, ChatCreateInput, ChatParticipantShape, ChatShape, ChatUpdateInput, DAYS_30_MS, deleteArrayIndex, DUMMY_ID, endpointGetChat, endpointGetFeedHome, endpointPostChat, endpointPutChat, FindByIdInput, FocusMode, FocusModeStopCondition, HomeInput, HomeResult, LINKS, MyStuffPageTabOption, Reminder, ResourceList as ResourceListType, Schedule, updateArray, uuid, VALYXA_ID } from "@local/shared";
import { Box, Button, styled, useTheme } from "@mui/material";
import { hasErrorCode } from "api/errorParser";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { ChatBubbleTree } from "components/ChatBubbleTree/ChatBubbleTree";
import { ListTitleContainer } from "components/containers/ListTitleContainer/ListTitleContainer";
import { ChatMessageInput } from "components/inputs/ChatMessageInput/ChatMessageInput";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ResourceList } from "components/lists/resource";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts";
import { useMessageActions, useMessageInput, useMessageTree } from "hooks/messages";
import { useChatTasks } from "hooks/tasks";
import { useHistoryState } from "hooks/useHistoryState";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSocketChat } from "hooks/useSocketChat";
import { PageTab } from "hooks/useTabs";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, ChevronLeftIcon, MonthIcon, OpenInNewIcon, ReminderIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { DUMMY_LIST_LENGTH } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getCookieMatchingChat, setCookieMatchingChat } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { CHAT_DEFAULTS, chatInitialValues, transformChatValues, withModifiableMessages, withYourMessages } from "views/objects/chat";
import { DashboardViewProps } from "../types";

const MAX_EVENTS_SHOWN = 10;
const MESSAGE_LIST_ID = "dashboardMessage";

type DashboardTabsInfo = {
    IsSearchable: false;
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
    background: theme.palette.primary.main,
}));

const NewChatButton = styled(Button)(({ theme }) => ({
    margin: theme.spacing(1),
    borderRadius: theme.spacing(8),
    padding: "4px 8px",
}));

const pageTabsStyle = { width: "min(700px, 100%)", minWidth: undefined, margin: "auto" } as const;
const exitChatButtonStyle = { margin: 1, borderRadius: 8, padding: "4px 8px" } as const;
const resourceListStyle = { list: { justifyContent: "flex-start" } } as const;

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
    const [refetch, { data, loading: isFeedLoading }] = useLazyFetch<HomeInput, HomeResult>(endpointGetFeedHome);

    const {
        fetch,
        fetchCreate,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Chat, ChatCreateInput, ChatUpdateInput>({
        isCreate: false, // We create chats automatically, so this should always be false
        isMutate: true,
        endpointCreate: endpointPostChat,
        endpointUpdate: endpointPutChat,
    });

    const [getChat, { data: loadedChat, loading: isChatLoading }] = useLazyFetch<FindByIdInput, Chat>(endpointGetChat);
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
        setCookieMatchingChat(loadedChat.id, [userId, VALYXA_ID]);
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
                if (userId) setCookieMatchingChat(data.id, [userId, VALYXA_ID]);
            },
            onCompleted: () => {
                chatCreateStatus.current = "complete";
            },
        });
    }, [chat, languages, fetchCreate, session, t]);
    const resetChat = useCallback(() => { createChat(true); }, [createChat]);

    // Create chats automatically
    const chatCreateStatus = useRef<"notStarted" | "inProgress" | "complete">("notStarted");
    useEffect(() => {
        const userId = getCurrentUser(session).id;
        if (!userId) return;
        // Unlike the chat view, we look for the chat by local storage data rather than the ID in the URL
        const existingChatId = getCookieMatchingChat([userId, VALYXA_ID]);
        const isChatValid = chat.id !== DUMMY_ID && chat.participants?.every(p => [userId, VALYXA_ID].includes(p.user.id));
        if (chat.id === DUMMY_ID && existingChatId) {
            fetchLazyWrapper<FindByIdInput, Chat>({
                fetch: getChat,
                inputs: { id: existingChatId },
                onSuccess: (data) => {
                    setChat({ ...data, messages: [] });
                },
                onError: (response) => {
                    if (hasErrorCode(response, "NotFound")) {
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
    const { active: activeFocusMode, all: allFocusModes } = useMemo(() => getFocusModeInfo(session), [session]);

    // Handle tabs
    const tabs = useMemo<PageTab<DashboardTabsInfo>[]>(() => {
        const activeColor = isMobile ? palette.primary.contrastText : palette.background.textPrimary;
        const inactiveColor = isMobile ? palette.primary.contrastText : palette.background.textSecondary;
        return [
            ...allFocusModes.map((mode, index) => ({
                color: mode.id === activeFocusMode?.mode?.id ? activeColor : inactiveColor,
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
                index: allFocusModes.length,
                key: "Add",
                label: "Add",
                searchPlaceholder: "",
            },
        ];
    }, [activeFocusMode?.mode?.id, allFocusModes, isMobile, palette.background.textPrimary, palette.background.textSecondary, palette.primary.contrastText]);
    const currTab = useMemo(() => {
        if (typeof activeFocusMode === "string") return "Add";
        const match = tabs.find(tab => typeof tab.data === "object" && tab.data.id === activeFocusMode?.mode?.id);
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e: any, tab: PageTab<DashboardTabsInfo>) => {
        e.preventDefault();
        // If "Add", go to the add focus mode page
        if (tab.key === "Add" || !tab.data) {
            setLocation(LINKS.SettingsFocusModes);
            return;
        }
        // Otherwise, publish the focus mode
        PubSub.get().publish("focusMode", {
            __typename: "ActiveFocusMode" as const,
            mode: tab.data,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    }, [setLocation]);
    useEffect(() => {
        refetch({ activeFocusModeId: activeFocusMode?.mode?.id });
    }, [activeFocusMode, refetch]);

    // Converts resources to a resource list
    const [resourceList, setResourceList] = useState<ResourceListType>({
        __typename: "ResourceList",
        created_at: 0,
        updated_at: 0,
        id: DUMMY_ID,
        resources: [],
        translations: [],
    } as unknown as ResourceListType);
    useEffect(() => {
        if (data?.resources) {
            setResourceList(r => ({ ...r, resources: data.resources }));
        }
    }, [data]);
    useEffect(() => {
        // Resources are added to the focus mode's resource list
        if (activeFocusMode?.mode?.resourceList?.id && activeFocusMode.mode?.resourceList.id !== DUMMY_ID) {
            setResourceList({
                ...activeFocusMode.mode.resourceList,
                __typename: "ResourceList" as const,
                listFor: {
                    ...activeFocusMode.mode,
                    __typename: "FocusMode" as const,
                    resourceList: undefined, // Avoid circular reference
                },
            });
        }
    }, [activeFocusMode]);

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

    const showTabs = useMemo(() => view === "home" && Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [session, allFocusModes.length, currTab, view]);

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
                    {showTabs && currTab && <PageTabs
                        ariaLabel="home-tabs"
                        id="home-tabs"
                        currTab={currTab as PageTab<DashboardTabsInfo>}
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
                            id="main-resource-list"
                            list={resourceList}
                            canUpdate={true}
                            handleUpdate={setResourceList}
                            horizontal
                            loading={isFeedLoading}
                            mutate={true}
                            parent={activeFocusMode?.mode ?
                                {
                                    ...activeFocusMode.mode,
                                    __typename: "FocusMode" as const,
                                    resourceList: undefined,
                                } as FocusMode : // Avoid circular reference
                                { __typename: "FocusMode", id: DUMMY_ID }
                            }
                            title={t("Resource", { count: 2 })}
                            sxs={resourceListStyle}
                        />
                    </Box>
                    {/* Events */}
                    <ListTitleContainer
                        Icon={MonthIcon}
                        id="main-event-list"
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
                        id="main-reminder-list"
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
