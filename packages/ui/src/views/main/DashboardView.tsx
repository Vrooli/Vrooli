import { CalendarEvent, ChatParticipantShape, DAYS_30_MS, DUMMY_ID, HomeResult, Reminder, ReminderList as ReminderListShape, Resource, ResourceList as ResourceListType, Schedule, calculateOccurrences, endpointsFeed, getAvailableModels, uuid, uuidToBase36 } from "@local/shared";
import { Box, IconButton, Typography, styled } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getExistingAIConfig } from "../../api/ai.js";
import { ChatBubbleTree } from "../../components/ChatBubbleTree/ChatBubbleTree.js";
import { ChatSettingsMenu, IntegrationSettings, ModelConfig, ShareSettings } from "../../components/ChatSettingsMenu/ChatSettingsMenu.js";
import { ChatMessageInput } from "../../components/inputs/ChatMessageInput/ChatMessageInput.js";
import { EventList } from "../../components/lists/EventList/EventList.js";
import { ReminderList } from "../../components/lists/ReminderList/ReminderList.js";
import { ResourceList } from "../../components/lists/ResourceList/ResourceList.js";
import { NavListBox, NavListInboxButton, NavListNewChatButton, NavListProfileButton, NavbarInner, SiteNavigatorButton } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useMessageInput } from "../../hooks/messages.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useHistoryState } from "../../hooks/useHistoryState.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { useActiveChat } from "../../stores/activeChatStore.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS, MAX_CHAT_INPUT_WIDTH } from "../../utils/consts.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { VALYXA_INFO } from "../objects/chat/ChatCrud.js";
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

const resourceListStyle = { list: { justifyContent: "flex-start" } } as const;

const resourceListFallback = {
    __typename: "ResourceList",
    created_at: 0,
    updated_at: 0,
    id: DUMMY_ID,
    resources: [],
    translations: [],
} as unknown as ResourceListType;

const greetingStyle = { mb: 2 } as const;

const NOON_HOUR = 12;
const EVENING_HOUR = 18;

/** Helper function to get the appropriate time of day greeting key */
function getTimeOfDayGreeting() {
    const hour = new Date().getHours();
    if (hour < NOON_HOUR) return "GoodMorning" as const;
    if (hour < EVENING_HOUR) return "GoodAfternoon" as const;
    return "GoodEvening" as const;
}

/** View displayed for Home page when logged in */
export function DashboardView({
    display,
}: DashboardViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const isLeftHanded = useIsLeftHanded();

    const [refetch, { data: feedData, loading: isFeedLoading }] = useLazyFetch<Record<string, never>, HomeResult>(endpointsFeed.home);

    const [resourceList, setResourceList] = useState<ResourceListType>(resourceListFallback);
    const feedResourcesRef = useRef<Resource[]>([]);

    const [reminders, setReminders] = useState<Reminder[]>([]);
    const feedRemindersRef = useRef<Reminder[]>([]);

    const [schedules, setSchedules] = useState<Schedule[]>([]);
    const feedSchedulesRef = useRef<Schedule[]>([]);

    // State for the ChatSettingsMenu
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);

    // Get available models for default config
    const availableModels = useMemo(() => {
        const config = getExistingAIConfig()?.service?.config;
        return config ? getAvailableModels(config) : [];
    }, []);

    // Default model config state
    const [modelConfigs, setModelConfigs] = useState<ModelConfig[]>(() => {
        const firstModel = availableModels.length > 0 ? availableModels[0] : null;
        if (!firstModel) return [];
        return [{
            model: firstModel,
            toolSettings: { siteActions: true, webSearch: true, fileRetrieval: true },
            requireConfirmation: false,
        }];
    });

    // Participants for dashboard context (e.g., just Valyxa)
    const [participants, setParticipants] = useState<ChatParticipantShape[]>([
        {
            __typename: "ChatParticipant",
            id: uuid(),
            user: { ...VALYXA_INFO, __typename: "User" },
            // chat property is usually added by the backend or context
        } as ChatParticipantShape // Type assertion needed as 'chat' is missing
    ]);

    // Default share settings for dashboard
    const [shareSettings, setShareSettings] = useState<ShareSettings>({ enabled: false });

    // Default integration settings for dashboard
    const [integrationSettings, setIntegrationSettings] = useState<IntegrationSettings>({ projects: [] });

    // Placeholder chat object for the menu props
    const settingsChat = useMemo(() => ({
        __typename: "Chat" as const,
        id: DUMMY_ID,
        translations: [{
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "Dashboard Settings",
            description: "Configure default model settings for new chats.",
        }]
    }), [session]);

    // Update local state when model config changes
    const handleModelConfigChange = useCallback((newConfigs: ModelConfig[]) => {
        // TODO: Persist these settings (e.g., to user profile or local storage)
        setModelConfigs(newConfigs);
        console.log("Dashboard model config updated:", newConfigs);
    }, []);

    // Placeholder callbacks (remain the same)
    const handleAddParticipant = useCallback((id: string) => { console.log("Dashboard: Add participant:", id); }, []);
    const handleRemoveParticipant = useCallback((id: string) => { console.log("Dashboard: Remove participant:", id); }, []);
    const handleUpdateDetails = useCallback((data: { name: string; description?: string }) => { console.log("Dashboard: Update details:", data); }, []);
    const handleToggleShare = useCallback((enabled: boolean) => {
        // Generate invite link when enabled, clear on disable
        const link = enabled ? `${window.location.origin}/chat/${uuidToBase36(settingsChat.id)}` : undefined;
        setShareSettings({ enabled, link });
    }, [settingsChat.id]);
    const handleIntegrationSettingsChange = useCallback((newSettings: IntegrationSettings) => {
        // TODO: Persist integration settings
        setIntegrationSettings(newSettings);
        console.log("Dashboard integration settings updated:", newSettings);
    }, []);
    // Placeholder for delete chat - not applicable on dashboard
    const handleDeleteChat = useCallback(() => {
        console.log("Dashboard: Delete Chat requested (no-op)");
    }, []);

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

    const [upcomingEvents, setUpcomingEvents] = useState<CalendarEvent[]>([]);
    useEffect(() => {
        const languages = getUserLanguages(session);
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
    }, [schedules, session]);

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
        languages: getUserLanguages(session),
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
            <ScrollBox>
                <NavbarInner>
                    <SiteNavigatorButton />
                    {/* Button to open settings menu */}
                    <IconButton onClick={() => setIsSettingsOpen(true)} aria-label={t("Settings")}>
                        <IconCommon name="Settings" />
                    </IconButton>
                    <NavListBox isLeftHanded={isLeftHanded}>
                        <NavListNewChatButton handleNewChat={resetActiveChat} />
                        <NavListInboxButton />
                        <NavListProfileButton />
                    </NavListBox>
                </NavbarInner>
                {/* TODO for morning: work on changes needed for a chat to track active and inactive tasks. Might need to link them to their own reminder list */}
                {!hasMessages && <Box
                    display="flex"
                    flexDirection="column"
                    gap={4}
                    width="100%"
                    maxWidth={MAX_CHAT_INPUT_WIDTH}
                    margin="auto"
                >
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
                        sx={resourceListStyle}
                    />
                    <EventList
                        id={ELEMENT_IDS.DashboardEventList}
                        list={upcomingEvents}
                        canUpdate={true}
                        handleUpdate={setUpcomingEvents}
                        loading={isFeedLoading}
                        mutate={true}
                    />
                    <ReminderList
                        id={ELEMENT_IDS.DashboardReminderList}
                        list={{
                            __typename: "ReminderList",
                            id: DUMMY_ID,
                            reminders: reminders,
                            created_at: Date.now(),
                            updated_at: Date.now(),
                        } as ReminderListShape}
                        canUpdate={true}
                        handleUpdate={(updatedList: ReminderListShape) => setReminders(updatedList.reminders)}
                        loading={isFeedLoading}
                        mutate={true}
                        parent={{ id: DUMMY_ID }}
                    />
                </Box>}
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
            </ScrollBox>
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

            {/* Render the settings menu dialog */}
            <ChatSettingsMenu
                open={isSettingsOpen}
                onClose={() => setIsSettingsOpen(false)}
                chat={settingsChat}
                participants={participants}
                modelConfigs={modelConfigs}
                shareSettings={shareSettings}
                integrationSettings={integrationSettings}
                onModelConfigChange={handleModelConfigChange}
                onAddParticipant={handleAddParticipant}
                onRemoveParticipant={handleRemoveParticipant}
                onUpdateDetails={handleUpdateDetails}
                onToggleShare={handleToggleShare}
                onIntegrationSettingsChange={handleIntegrationSettingsChange}
                onDeleteChat={handleDeleteChat}
            />
        </DashboardBox>
    );
}
