import Box from "@mui/material/Box";
import { IconButton } from "../../components/buttons/IconButton.js";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material/styles";
import { DAYS_30_MS, DUMMY_ID, calculateOccurrences, endpointsFeed, generatePK, type CalendarEvent, type HomeResult, type Reminder, type ReminderList as ReminderListShape, type Resource, type ResourceList as ResourceListType, type Schedule } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChatInterface } from "../../components/ChatInterface/ChatInterface.js";
import { EventList } from "../../components/lists/EventList/EventList.js";
import { ReminderList } from "../../components/lists/ReminderList/ReminderList.js";
import { ResourceList } from "../../components/lists/ResourceList/ResourceList.js";
import { NavListBox, NavListInboxButton, NavListNewChatButton, NavListProfileButton, NavbarInner, SiteNavigatorButton } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useIsLeftHanded } from "../../hooks/subscriptions.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { useActiveChat } from "../../stores/activeChatStore.js";
import { ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS, MAX_CHAT_INPUT_WIDTH } from "../../utils/consts.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { type DashboardViewProps } from "./types.js";

const MAX_EVENTS_SHOWN = 10;

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
    createdAt: 0,
    updatedAt: 0,
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

    // State for the ChatSettingsMenu - kept for external settings button
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);

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
                    id: generatePK(),
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

    const { resetActiveChat } = useActiveChat({ setMessage: () => {} });

    // Create the dashboard content that appears when no messages exist
    const dashboardContent = useMemo(() => (
        <Box
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
                    reminders,
                    createdAt: Date.now(),
                    updatedAt: Date.now(),
                } as ReminderListShape}
                canUpdate={true}
                handleUpdate={(updatedList: ReminderListShape) => setReminders(updatedList.reminders)}
                loading={isFeedLoading}
                mutate={true}
                parent={{ id: DUMMY_ID }}
            />
        </Box>
    ), [resourceList, upcomingEvents, reminders, isFeedLoading, t, session]);

    return (
        <DashboardBox>
            <ScrollBox>
                <NavbarInner>
                    <SiteNavigatorButton />
                    {/* Button to open chat settings menu */}
                    <IconButton onClick={() => setIsSettingsOpen(true)} aria-label={t("Settings")} variant="transparent">
                        <IconCommon name="Settings" />
                    </IconButton>
                    <NavListBox isLeftHanded={isLeftHanded}>
                        <NavListNewChatButton handleNewChat={resetActiveChat} />
                        <NavListInboxButton />
                        <NavListProfileButton />
                    </NavListBox>
                </NavbarInner>
            </ScrollBox>
            
            <ChatInterface
                display={display}
                placeholder={t("WhatWouldYouLikeToDo")}
                noMessagesContent={dashboardContent}
                showSettingsButton={false} // We handle settings in the navbar
                onSettingsOpen={() => setIsSettingsOpen(true)}
                onSettingsClose={() => setIsSettingsOpen(false)}
            />
        </DashboardBox>
    );
}
