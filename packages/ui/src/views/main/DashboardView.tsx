import { CalendarEvent, DAYS_30_MS, DUMMY_ID, FocusMode, HomeResult, LlmModel, Reminder, Resource, ResourceList as ResourceListType, Schedule, calculateOccurrences, endpointsFeed, getAvailableModels, uuid } from "@local/shared";
import { Box, IconButton, InputAdornment, ListItemIcon, Menu, MenuItem, TextField, Typography, styled } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getExistingAIConfig } from "../../api/ai.js";
import { ChatBubbleTree } from "../../components/ChatBubbleTree/ChatBubbleTree.js";
import { MenuTitle } from "../../components/dialogs/MenuTitle/MenuTitle.js";
import { ChatMessageInput } from "../../components/inputs/ChatMessageInput/ChatMessageInput.js";
import { EventList } from "../../components/lists/EventList/EventList.js";
import { ReminderList } from "../../components/lists/ReminderList/ReminderList.js";
import { ResourceList } from "../../components/lists/ResourceList/ResourceList.js";
import { NavListBox, NavListInboxButton, NavListProfileButton, NavbarInner, NavbarSpacer, SiteNavigatorButton } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useMessageInput } from "../../hooks/messages.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useHistoryState } from "../../hooks/useHistoryState.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { useActiveChat } from "../../stores/activeChatStore.js";
import { useFocusModes } from "../../stores/focusModeStore.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS, MAX_CHAT_INPUT_WIDTH } from "../../utils/consts.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { DashboardViewProps } from "./types.js";

const MAX_EVENTS_SHOWN = 10;
const MESSAGE_LIST_ID = "dashboardMessage";

const DashboardBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    maxHeight: "100vh",
    height: "calc(100vh - env(safe-area-inset-bottom))",
    overflow: "hidden",
    paddingBottom: theme.spacing(2),
    [`@media (max-width: ${MAX_CHAT_INPUT_WIDTH}px)`]: {
        paddingBottom: 0,
    },
}));

const DashboardInnerBox = styled(Box)(({ theme }) => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    flexGrow: 1,
    gap: theme.spacing(2),
    justifyContent: "end",
    margin: "auto",
    overflowY: "auto",
    padding: theme.spacing(2),
    width: `min(${MAX_CHAT_INPUT_WIDTH}px, 100%)`,
}));

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

const ModelTitleBox = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    cursor: "pointer",
    color: theme.palette.background.textSecondary,
}));
const modelMenuAnchorOrigin = {
    vertical: "bottom",
    horizontal: "left",
} as const;
const modelMenuTransformOrigin = {
    vertical: "top",
    horizontal: "left",
} as const;
const greetingStyle = { mb: 2 } as const;
const modelMenuPaperProps = {
    style: {
        maxHeight: 400,
        width: "350px",
    },
} as const;
const searchBoxStyle = { p: 2 } as const;
const noResultsStyle = { p: 2, textAlign: "center" } as const;
const searchInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                name="Search"
                size={20}
            />
        </InputAdornment>
    ),
} as const;
const menuListStyle = {
    "& .MuiMenu-list": {
        padding: 0, // Remove default padding
    },
} as const;

function ModelTitleComponent() {
    const { t } = useTranslation();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const [searchQuery, setSearchQuery] = useState("");
    const searchInputRef = useRef<HTMLInputElement>(null);
    const open = Boolean(anchorEl);

    const availableModels = useMemo(() => {
        const config = getExistingAIConfig()?.service?.config;
        return config ? getAvailableModels(config) : [];
    }, []);

    const [selectedModel, setSelectedModel] = useState<LlmModel | null>(
        availableModels.length > 0 ? availableModels[0] : null,
    );

    const handleClick = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
        setSearchQuery("");
    }, []);

    const handleClose = useCallback(() => {
        setAnchorEl(null);
    }, []);

    const handleModelSelect = useCallback((model: LlmModel) => {
        setSelectedModel(model);
        handleClose();
    }, [handleClose]);

    const handleSearchChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(event.target.value);
    }, []);

    // Focus the search input when the menu opens
    useEffect(() => {
        if (open && searchInputRef.current) {
            setTimeout(() => {
                searchInputRef.current?.focus();
            }, 100);
        }
    }, [open]);

    const filteredModels = useMemo(() => {
        if (!searchQuery) return availableModels;
        const query = searchQuery.toLowerCase();
        return availableModels.filter(model =>
            t(model.name, { ns: "service" }).toLowerCase().includes(query) ||
            (model.description && t(model.description, { ns: "service", defaultValue: "" }).toLowerCase().includes(query)),
        );
    }, [availableModels, searchQuery, t]);

    const modelHelpText = "Select an AI model to use for your conversations. Different models have different capabilities, strengths, and pricing.";

    return (
        <ModelTitleBox
            onClick={handleClick}
        >
            <Typography variant="body1" color="inherit" component="div">
                {selectedModel ?
                    t(selectedModel.name, { ns: "service" }) :
                    t("SelectModel")
                }
            </Typography>
            <IconCommon
                fill="background.textSecondary"
                name="ChevronRight"
            />
            <Menu
                id="model-selection-menu"
                anchorEl={anchorEl}
                open={open}
                onClose={handleClose}
                anchorOrigin={modelMenuAnchorOrigin}
                transformOrigin={modelMenuTransformOrigin}
                PaperProps={modelMenuPaperProps}
                disableAutoFocusItem
                sx={menuListStyle}
            >
                <MenuTitle
                    title={t("SelectModel")}
                    onClose={handleClose}
                    help={modelHelpText}
                />
                <Box sx={searchBoxStyle}>
                    <TextField
                        fullWidth
                        placeholder="Search models"
                        value={searchQuery}
                        onChange={handleSearchChange}
                        variant="outlined"
                        size="small"
                        InputProps={searchInputProps}
                        inputRef={searchInputRef}
                    />
                </Box>
                {filteredModels.length === 0 ? (
                    <Box sx={noResultsStyle}>
                        <Typography variant="body2" color="textSecondary">
                            No models found
                        </Typography>
                    </Box>
                ) : (
                    filteredModels.map((model) => {
                        function handleClick() {
                            handleModelSelect(model);
                        }
                        return (
                            <MenuItem
                                key={model.value}
                                onClick={handleClick}
                                selected={selectedModel?.value === model.value}
                            >
                                <ListItemIcon>
                                    {selectedModel?.value === model.value && (
                                        <IconCommon
                                            fill="background.textPrimary"
                                            name="Complete"
                                            size={20}
                                        />
                                    )}
                                </ListItemIcon>
                                <Box>
                                    <Typography variant="body1">
                                        {t(model.name, { ns: "service" })}
                                    </Typography>
                                    <Typography variant="caption" color="textSecondary">
                                        {t(model.description || "", { ns: "service", defaultValue: "" })}
                                    </Typography>
                                </Box>
                            </MenuItem>
                        );
                    })
                )}
            </Menu>
        </ModelTitleBox>
    );
}

const NOON_HOUR = 12;
const EVENING_HOUR = 18;
/** Helper function to get the appropriate time of day greeting key */
function getTimeOfDayGreeting() {
    const hour = new Date().getHours();
    if (hour < NOON_HOUR) return "GoodMorning" as const;
    if (hour < EVENING_HOUR) return "GoodAfternoon" as const;
    return "GoodEvening" as const;
}

/**
 * Button to create a new chat
 */
export function NavListNewChatButton({
    handleNewChat,
}: {
    handleNewChat: () => unknown;
}) {
    const { t } = useTranslation();

    return (
        <IconButton
            aria-label={t("NewChat")}
            onClick={handleNewChat}
            title={t("NewChat")}
        >
            <IconCommon
                decorative
                name="ChatNew"
                size={32}
            />
        </IconButton>
    );
}

/** View displayed for Home page when logged in */
export function DashboardView({
    display,
}: DashboardViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const isLeftHanded = useIsLeftHanded();

    const [refetch, { data: feedData, loading: isFeedLoading }] = useLazyFetch<Record<string, never>, HomeResult>(endpointsFeed.home);

    const focusModeInfo = useFocusModes(session);

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

    const [message, setMessage] = useHistoryState<string>(`${MESSAGE_LIST_ID}-message`, "");
    const {
        chat,
        isBotOnlyChat,
        isLoading: isChatLoading,
        messageActions,
        messageStream,
        messageTree,
        usersTyping: participantsTyping,
        resetActiveChat,
        taskInfo,
    } = useActiveChat({ setMessage });
    const messageInput = useMessageInput({
        id: MESSAGE_LIST_ID,
        languages,
        message,
        postMessage: messageActions.postMessage,
        putMessage: messageActions.putMessage,
        replyToMessage: messageActions.replyToMessage,
        setMessage,
    });
    const hasMessages = useMemo(function hasMessagesMemo() {
        return messageTree.tree.getMessagesCount() > 0;
    }, [messageTree]);

    return (
        <DashboardBox>
            <NavbarSpacer />
            <NavbarInner>
                <SiteNavigatorButton />
                <ModelTitleComponent />
                <NavListBox isLeftHanded={isLeftHanded}>
                    <NavListNewChatButton handleNewChat={resetActiveChat} />
                    <NavListInboxButton />
                    <NavListProfileButton />
                </NavListBox>
            </NavbarInner>
            <DashboardInnerBox>
                {/* TODO for morning: work on changes needed for a chat to track active and inactive tasks. Might need to link them to their own reminder list */}
                {!hasMessages && <>
                    <Typography
                        variant="h4"
                        textAlign="center"
                        color="textPrimary"
                        sx={greetingStyle}
                    >
                        {t(getTimeOfDayGreeting())}{getCurrentUser(session)?.name || getCurrentUser(session)?.handle ? `, ${getCurrentUser(session)?.name || getCurrentUser(session)?.handle}` : ""}
                    </Typography>
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
                    <EventList
                        id={ELEMENT_IDS.DashboardEventList}
                        list={upcomingEvents}
                        canUpdate={true}
                        handleUpdate={setUpcomingEvents}
                        loading={isFeedLoading}
                        mutate={true}
                        title={t("Schedule", { count: 1 })}
                    />
                    <ReminderList
                        id={ELEMENT_IDS.DashboardReminderList}
                        list={reminders}
                        canUpdate={true}
                        handleUpdate={setReminders}
                        loading={isFeedLoading}
                        mutate={true}
                        title={t("Reminder", { count: 2 })}
                    />
                </>}
                {hasMessages && <ChatBubbleTree
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
                isLoading={isChatLoading}
                message={message}
                messageBeingEdited={messageInput.messageBeingEdited}
                messageBeingRepliedTo={messageInput.messageBeingRepliedTo}
                participantsTyping={participantsTyping}
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
