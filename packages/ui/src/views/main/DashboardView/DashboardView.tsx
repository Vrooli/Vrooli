import { calculateOccurrences, Chat, ChatCreateInput, ChatInviteStatus, ChatMessage, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatParticipant, ChatUpdateInput, DUMMY_ID, endpointGetChat, endpointGetChatMessageTree, endpointGetFeedHome, endpointPostChat, endpointPutChat, FindByIdInput, FocusMode, FocusModeStopCondition, HomeInput, HomeResult, LINKS, Reminder, ResourceList as ResourceListType, Schedule, uuid, VALYXA_ID } from "@local/shared";
import { Box, Button, IconButton, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode, socket } from "api";
import { ChatBubbleTree, TypingIndicator } from "components/ChatBubbleTree/ChatBubbleTree";
import { ListTitleContainer } from "components/containers/ListTitleContainer/ListTitleContainer";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ResourceList } from "components/lists/resource";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts/SessionContext";
import { useHistoryState } from "hooks/useHistoryState";
import { useKeyboardOpen } from "hooks/useKeyboardOpen";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useMessageTree } from "hooks/useMessageTree";
import { PageTab } from "hooks/useTabs";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, ChevronLeftIcon, ListIcon, MonthIcon, OpenInNewIcon, ReminderIcon, SendIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { CalendarEvent } from "types";
import { getCurrentUser, getFocusModeInfo } from "utils/authentication/session";
import { BranchMap, getCookieMatchingChat, getCookieMessageTree, setCookieMatchingChat, setCookieMessageTree } from "utils/cookies";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { MyStuffPageTabOption } from "utils/search/objectToSearch";
import { deleteArrayIndex, updateArray } from "utils/shape/general";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { chatInitialValues, transformChatValues, VALYXA_INFO, withoutOtherMessages } from "views/objects/chat";
import { DashboardViewProps } from "../types";

const CHAT_DEFAULTS = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    openToAnyoneWithInvite: false,
    invites: [{
        __typename: "ChatInvite" as const,
        id: DUMMY_ID,
        status: ChatInviteStatus.Pending,
        user: VALYXA_INFO,
    }],
} as unknown as Chat;

/** View displayed for Home page when logged in */
export const DashboardView = ({
    display,
    isOpen,
    onClose,
}: DashboardViewProps) => {
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isKeyboardOpen = useKeyboardOpen();

    const [message, setMessage] = useHistoryState<string>("dashboardMessage", "");
    const [usersTyping, setUsersTyping] = useState<ChatParticipant[]>([]);
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
    const [chat, setChat] = useState<ChatShape>(chatInitialValues(session, undefined, t, languages[0], CHAT_DEFAULTS));
    // When a chat is loaded, store chat ID by participant and task
    useEffect(() => {
        if (!loadedChat?.id) return;
        const userId = getCurrentUser(session).id;
        if (!userId) return;
        setCookieMatchingChat(loadedChat.id, [userId, VALYXA_ID]);
    }, [loadedChat, loadedChat?.id, session]);

    const createChat = useCallback((resetChat = false) => {
        chatCreateStatus.current = "inProgress";
        const chatToUse = resetChat ? chatInitialValues(session, undefined, t, languages[0], CHAT_DEFAULTS) : chat;
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: fetchCreate,
            inputs: transformChatValues(withoutOtherMessages(chatToUse, session), withoutOtherMessages(chatToUse, session), true),
            onSuccess: (data) => {
                console.log("created new chat!", data);
                setChat(data);
                const userId = getCurrentUser(session).id;
                if (userId) setCookieMatchingChat(data.id, [userId, VALYXA_ID]);
            },
            onCompleted: () => {
                chatCreateStatus.current = "complete";
            },
        });
    }, [chat, languages, fetchCreate, session, t]);

    // Create chats automatically
    const chatCreateStatus = useRef<"notStarted" | "inProgress" | "complete">("notStarted");
    useEffect(() => {
        const userId = getCurrentUser(session).id;
        if (!userId) return;
        // Unlike the chat view, we look for the chat by local storage data rather than the ID in the URL
        const existingChatId = getCookieMatchingChat([userId, VALYXA_ID]);
        const isChatValid = chat.id !== DUMMY_ID && chat.participants.every(p => [userId, VALYXA_ID].includes(p.user.id));
        if (chat.id === DUMMY_ID && existingChatId) {
            console.log("fetching chattttt", existingChatId, [userId, VALYXA_ID]);
            fetchLazyWrapper<FindByIdInput, Chat>({
                fetch: getChat,
                inputs: { id: existingChatId },
                onSuccess: (data) => {
                    console.log("fetched chattttt", data);
                    setChat(data);
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
    const tabs = useMemo<PageTab<FocusMode | "Add", false>[]>(() => ([
        ...allFocusModes.map((mode, index) => ({
            index,
            label: mode.name,
            tabType: mode,
        })),
        {
            index: allFocusModes.length,
            label: "Add",
            Icon: AddIcon,
            tabType: "Add",
        },
    ]), [allFocusModes]);
    const currTab = useMemo(() => {
        if (typeof activeFocusMode === "string") return "Add";
        const match = tabs.find(tab => typeof tab.tabType !== "string" && tab.tabType.id === activeFocusMode?.mode?.id);
        if (match) return match;
        if (tabs.length) return tabs[0];
        return null;
    }, [tabs, activeFocusMode]);
    const handleTabChange = useCallback((e: any, tab: PageTab<FocusMode | "Add", false>) => {
        e.preventDefault();
        // If "Add", go to the add focus mode page
        if (tab.tabType === "Add") {
            setLocation(LINKS.SettingsFocusModes);
            return;
        }
        // Otherwise, publish the focus mode
        PubSub.get().publish("focusMode", {
            __typename: "ActiveFocusMode" as const,
            mode: tab.tabType,
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

        const fetchUpcomingEvents = async () => {
            const result: CalendarEvent[] = [];
            for (const schedule of schedules) {
                const occurrences = await calculateOccurrences(
                    schedule,
                    new Date(),
                    new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) // 30 days
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
                setUpcomingEvents(result.slice(0, 10));
            }
        };

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

    const openSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: "chat-side-menu", isOpen: true }); }, []);
    const closeSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: "chat-side-menu", isOpen: false }); }, []);
    useEffect(() => {
        return () => {
            closeSideMenu();
        };
    }, [closeSideMenu]);

    const [showChat, setShowChat] = useState(false);
    const showTabs = useMemo(() => !showChat && Boolean(getCurrentUser(session).id) && allFocusModes.length > 1 && currTab !== null, [showChat, session, allFocusModes.length, currTab]);

    const { addMessages, clearMessages, editMessage, messagesCount, removeMessages, tree } = useMessageTree<ChatMessageShape>([]);
    const [branches, setBranches] = useState<BranchMap>(getCookieMessageTree(chat.id)?.branches ?? {});

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => {
        setInputFocused(true);
        setShowChat(true);
    }, []);

    // We query messages separate from the chat, since we must traverse the message tree
    const [getTreeData, { data: searchTreeData }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);

    // When chatId changes, clear the message tree and branches, and fetch new data
    useEffect(() => {
        if (chat.id === DUMMY_ID) return;
        clearMessages();
        setBranches({});
        getTreeData({ chatId: chat.id });
    }, [chat.id, clearMessages, getTreeData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        console.log("got search tree messages!!", searchTreeData.messages);
        addMessages(searchTreeData.messages);
    }, [addMessages, searchTreeData]);

    useEffect(() => {
        // Update the cookie with current branches
        setCookieMessageTree(chat.id, { branches, locationId: "someLocationId" }); // TODO locationId should be last chat message in view
    }, [branches, chat.id]);

    const onSubmit = useCallback((updatedChat: ChatShape) => {
        fetchLazyWrapper<ChatUpdateInput, Chat>({
            fetch,
            inputs: transformChatValues(withoutOtherMessages(updatedChat, session), withoutOtherMessages(chat, session), false),
            onSuccess: (data) => {
                console.log("update success!", data);
                setChat(data);
                setMessage("");
            },
            spinnerDelay: null,
        });
    }, [chat, fetch, setChat, session, setMessage]);

    // Handle websocket connection/disconnection
    useEffect(() => {
        // Only connect to the websocket if the chat exists
        if (!chat?.id || chat.id === DUMMY_ID) return;
        socket.emit("joinRoom", chat.id, (response) => {
            if (response.error) {
                console.error("Failed to join chat room", response?.error);
            } else {
                console.info("Joined chat room");
            }
        });
        // Leave the chat room when the component is unmounted
        return () => {
            socket.emit("leaveRoom", chat.id, (response) => {
                if (response.error) {
                    console.error("Failed to leave chat room", response?.error);
                } else {
                    console.info("Left chat room");
                }
            });
        };
    }, [chat.id]);
    // Handle websocket events
    useEffect(() => {
        // When a message is received, add it to the chat
        socket.on("message", (message: ChatMessage) => {
            console.log("got websocket message!!!", message);
            addMessages([message]);
        });
        // When a message is updated, update it in the chat
        socket.on("editMessage", (message: ChatMessage) => {
            editMessage(message);
        });
        // When a message is deleted, remove it from the chat
        socket.on("deleteMessage", (id: string) => {
            removeMessages([id]);
        });
        // Show the status of users typing
        socket.on("typing", ({ starting, stopping }: { starting?: string[], stopping?: string[] }) => {
            // Add every user that's typing
            const newTyping = [...usersTyping];
            for (const id of starting ?? []) {
                // Never add yourself
                if (id === getCurrentUser(session).id) continue;
                if (newTyping.some(p => p.user.id === id)) continue;
                const participant = chat.participants?.find(p => p.user.id === id);
                if (!participant) continue;
                newTyping.push(participant);
            }
            // Remove every user that stopped typing
            for (const id of stopping ?? []) {
                const index = newTyping.findIndex(p => p.user.id === id);
                if (index === -1) continue;
                newTyping.splice(index, 1);
            }
            setUsersTyping(newTyping);
        });
        return () => {
            // Remove chat-specific event handlers
            socket.off("message");
            socket.off("typing");
        };
    }, [chat.participants, setChat, message, session, usersTyping, addMessages, editMessage, removeMessages]);

    const addMessage = useCallback((text: string) => {
        const newMessage: ChatMessageShape = {
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            isUnsent: true,
            chat: {
                __typename: "Chat" as const,
                id: chat.id,
            },
            reactionSummaries: [],
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language: languages[0],
                text,
            }],
            user: {
                __typename: "User" as const,
                id: getCurrentUser(session).id ?? "",
                isBot: false,
                name: getCurrentUser(session).name ?? undefined,
            },
        };
        onSubmit({ ...chat, messages: [...(chat.messages ?? []), newMessage] });
    }, [chat, languages, onSubmit, session]);

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            maxHeight: "100vh",
            height: "calc(100vh - env(safe-area-inset-bottom))",
            overflow: "hidden",
        }}>
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
                below={<>
                    {showTabs && currTab && <PageTabs
                        ariaLabel="home-tabs"
                        id="home-tabs"
                        currTab={currTab as PageTab<FocusMode, false>}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                        sx={{ width: "min(700px, 100%)", minWidth: undefined, margin: "auto" }}
                    />}
                    {showChat && <Box
                        display="flex"
                        flexDirection="row"
                        justifyContent="space-around"
                        alignItems="center"
                    >
                        <Button
                            color="primary"
                            onClick={() => { setShowChat(false); }}
                            variant="contained"
                            sx={{ margin: 1, borderRadius: 8, padding: "4px 8px" }}
                            startIcon={<ChevronLeftIcon />}
                        >
                            {t("Dashboard")}
                        </Button>
                        <Button
                            color="primary"
                            onClick={() => { createChat(true); }}
                            variant="contained"
                            sx={{ margin: 1, borderRadius: 8, padding: "4px 8px" }}
                            startIcon={<AddIcon />}
                        >
                            {t("NewChat")}
                        </Button>
                    </Box>}
                </>}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                flexGrow: 1,
                gap: 2,
                margin: "auto",
                paddingBottom: 4,
                overflowY: "auto",
                width: "min(700px, 100%)",
            }}>
                {/* TODO for morning: work on changes needed for a chat to track active and inactive tasks. Might need to link them to their own reminder list */}
                {!showChat && <>
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
                            sxs={{ list: { justifyContent: "flex-start" } }}
                        />
                    </Box>
                    {/* Events */}
                    {upcomingEvents.length > 0 && <ListTitleContainer
                        Icon={MonthIcon}
                        id="main-event-list"
                        isEmpty={upcomingEvents.length === 0 && !isFeedLoading}
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
                            loading={isFeedLoading}
                            onAction={onEventAction}
                        />
                    </ListTitleContainer>}
                    {/* Reminders */}
                    {reminders.length > 0 && <ListTitleContainer
                        Icon={ReminderIcon}
                        id="main-reminder-list"
                        isEmpty={reminders.length === 0 && !isFeedLoading}
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
                            loading={isFeedLoading}
                            onAction={onReminderAction}
                        />
                    </ListTitleContainer>}
                </>}
                {showChat && <ChatBubbleTree
                    branches={branches}
                    editMessage={editMessage}
                    handleReply={() => { }}
                    handleRetry={() => { }}
                    removeMessages={removeMessages}
                    setBranches={setBranches}
                    tree={tree}
                />}
            </Box>
            <TypingIndicator participants={usersTyping} />
            <RichInputBase
                actionButtons={[{
                    disabled: isChatLoading || isCreateLoading || isUpdateLoading,
                    Icon: SendIcon,
                    onClick: () => {
                        const trimmed = message.trim();
                        if (trimmed.length === 0) return;
                        addMessage(trimmed);
                    },
                }]}
                disableAssistant={true}
                fullWidth
                getTaggableItems={async (message) => {
                    // TODO should be able to tag any public or owned object (e.g. "Create routine like @some_existing_routine, but change a to b")
                    return [];
                }}
                maxChars={1500}
                maxRows={inputFocused ? 10 : 2}
                minRows={1}
                name="search"
                onChange={setMessage}
                onFocus={onFocus}
                onSubmit={(m) => {
                    const trimmed = m.trim();
                    if (trimmed.length === 0) return;
                    addMessage(trimmed);
                }}
                placeholder={t("WhatWouldYouLikeToDo")}
                sxs={{
                    root: {
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        maxHeight: "min(75vh, 500px)",
                        width: "min(700px, 100%)",
                        margin: "auto",
                        marginBottom: { xs: (display === "page" && !isKeyboardOpen) ? pagePaddingBottom : "0", md: "0" },
                    },
                    topBar: { borderRadius: 0, paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
                    bottomBar: { paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
                    textArea: {
                        border: "none",
                        background: palette.background.paper,
                    },
                }}
                value={message}
            />
            <ChatSideMenu />
        </Box>
    );
};
