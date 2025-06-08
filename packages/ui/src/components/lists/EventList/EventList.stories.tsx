/* eslint-disable no-magic-numbers */
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import type { Meta } from "@storybook/react";
import { ScheduleRecurrenceType, generatePK, uuid, type CalendarEvent, type Meeting, type Run, type Schedule } from "@vrooli/shared";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { EventList } from "./EventList.js";

const meta = {
    title: "Components/Lists/EventList",
    component: EventList,
    parameters: {
        docs: {
            description: {
                component: "A list component for displaying and managing calendar events.",
            },
        },
    },
} satisfies Meta<typeof EventList>;

export default meta;

// Mock data for meetings and routines
const mockMeetingId = uuid();
const mockMeetingTranslationId = uuid();
const mockMeeting: Meeting = {
    __typename: "Meeting",
    id: mockMeetingId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    translations: [{
        __typename: "MeetingTranslation",
        id: mockMeetingTranslationId,
        language: "en",
        name: "Team Sync",
        description: "Weekly team sync meeting",
    }],
    schedules: [],
    participants: [],
    invites: [],
    you: {
        __typename: "MeetingYou",
        canDelete: true,
        canUpdate: true,
        isInvited: true,
        isParticipant: true,
    },
};

const mockRun: Run = {
    __typename: "Run",
    id: generatePK().toString(),
    name: "Weekly Review",
    you: {
        __typename: "RunYou",
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

// Mock data for the story
const mockScheduleId = uuid();
const mockRecurrenceId = uuid();
const mockSchedule2Id = uuid();
const mockRecurrence2Id = uuid();
const mockSchedule3Id = uuid();
const mockRecurrence3Id = uuid();
const baseMockSchedules: Schedule[] = [
    {
        __typename: "Schedule",
        id: mockScheduleId,
        startTime: new Date("2024-03-20T10:00:00Z"),
        endTime: new Date("2024-03-20T11:00:00Z"),
        timezone: "UTC",
        exceptions: [],
        meetings: [mockMeeting],
        recurrences: [{
            __typename: "ScheduleRecurrence",
            id: mockRecurrenceId,
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            duration: 3600000, // 1 hour in milliseconds
            dayOfWeek: 3, // Wednesday
            schedule: { __typename: "Schedule", id: mockScheduleId } as Schedule,
        }],
        runs: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Schedule",
        id: mockSchedule2Id,
        startTime: new Date("2024-03-21T00:00:00Z"),
        endTime: new Date("2024-03-21T23:59:59Z"),
        timezone: "UTC",
        exceptions: [],
        meetings: [mockMeeting],
        recurrences: [{
            __typename: "ScheduleRecurrence",
            id: mockRecurrence2Id,
            recurrenceType: ScheduleRecurrenceType.Monthly,
            interval: 1,
            duration: 86400000, // 24 hours in milliseconds
            dayOfMonth: 21,
            schedule: { __typename: "Schedule", id: mockSchedule2Id } as Schedule,
        }],
        runs: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Schedule",
        id: mockSchedule3Id,
        startTime: new Date("2024-03-22T15:00:00Z"),
        endTime: new Date("2024-03-22T16:00:00Z"),
        timezone: "UTC",
        exceptions: [],
        meetings: [],
        recurrences: [{
            __typename: "ScheduleRecurrence",
            id: mockRecurrence3Id,
            recurrenceType: ScheduleRecurrenceType.Weekly,
            interval: 1,
            duration: 3600000, // 1 hour in milliseconds
            dayOfWeek: 5, // Friday
            schedule: { __typename: "Schedule", id: mockSchedule3Id } as Schedule,
        }],
        runs: [mockRun],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
];

const mockEvent1Id = uuid();
const mockEvent2Id = uuid();
const mockEvent3Id = uuid();
const mockEvent4Id = uuid();
const mockEvent5Id = uuid();
const mockEvent6Id = uuid();
const mockEvent7Id = uuid();

// Helper to create a date relative to now
function getRelativeDate(minutes: number): Date {
    const date = new Date();
    date.setMinutes(date.getMinutes() + minutes);
    return date;
}

const baseMockEvents: CalendarEvent[] = [
    {
        __typename: "CalendarEvent",
        id: mockEvent1Id,
        title: "Past Meeting",
        start: getRelativeDate(-30), // 30 minutes ago
        end: getRelativeDate(-15), // 15 minutes ago
        allDay: false,
        schedule: baseMockSchedules[0],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent2Id,
        title: "Current Meeting",
        start: getRelativeDate(-5), // Started 5 minutes ago
        end: getRelativeDate(25), // Ends in 25 minutes
        allDay: false,
        schedule: baseMockSchedules[0],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent3Id,
        title: "Upcoming Meeting",
        start: getRelativeDate(60), // 1 hour from now
        end: getRelativeDate(120), // 2 hours from now
        allDay: false,
        schedule: baseMockSchedules[0],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent4Id,
        title: "Tomorrow's Review",
        start: getRelativeDate(30 * 24), // 30 hours from now
        end: getRelativeDate(31 * 24), // 31 hours from now
        allDay: false,
        schedule: baseMockSchedules[1],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent5Id,
        title: "Weekly Planning",
        start: getRelativeDate(4 * 24 * 60), // 4 days from now
        end: getRelativeDate((4 * 24 * 60) + 60), // 4 days + 1 hour from now
        allDay: false,
        schedule: baseMockSchedules[2],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent6Id,
        title: "Monthly Review",
        start: getRelativeDate(30 * 24 * 60), // ~1 month from now
        end: getRelativeDate((30 * 24 * 60) + 120), // ~1 month + 2 hours from now
        allDay: false,
        schedule: baseMockSchedules[1],
    },
    {
        __typename: "CalendarEvent",
        id: mockEvent7Id,
        title: "Annual Planning",
        start: getRelativeDate(365 * 24 * 60), // ~1 year from now
        end: getRelativeDate((365 * 24 * 60) + 240), // ~1 year + 4 hours from now
        allDay: false,
        schedule: baseMockSchedules[2],
    },
];

// Update the extended mock events to use the new base events
const extendedMockEvents: CalendarEvent[] = Array.from({ length: 3 }).flatMap((_, i) =>
    baseMockEvents.map((event, j) => {
        // Add weeks to the dates
        const start = new Date(event.start);
        start.setDate(start.getDate() + i * 7);
        const end = new Date(event.end);
        end.setDate(end.getDate() + i * 7);

        // Create new meeting or runRoutine for this event
        const newMeeting: Meeting = event.schedule.meetings[0] ? {
            ...event.schedule.meetings[0],
            id: `meeting-${i * baseMockEvents.length + j + 1}`,
            translations: [{
                ...event.schedule.meetings[0].translations[0],
                id: `meeting-${i * baseMockEvents.length + j + 1}-en`,
            }],
        } : undefined;

        const newRun: Run = event.schedule.runs[0] ? {
            ...event.schedule.runs[0],
            id: `run-${i * baseMockEvents.length + j + 1}`,
            translations: [{
                ...event.schedule.runs[0].resourceVersion?.translations[0],
                id: `run-${i * baseMockEvents.length + j + 1}-en`,
            }],
        } : undefined;

        // Create a new schedule for this event
        const newSchedule: Schedule = {
            ...event.schedule,
            id: `schedule-${i * baseMockEvents.length + j + 1}`,
            startTime: start,
            endTime: end,
            meetings: newMeeting ? [newMeeting] : [],
            runs: newRun ? [newRun] : [],
            recurrences: event.schedule.recurrences.map(rec => ({
                ...rec,
                id: `recurrence-${i * baseMockEvents.length + j + 1}`,
                schedule: { __typename: "Schedule", id: `schedule-${i * baseMockEvents.length + j + 1}` } as Schedule,
            })),
        };

        return {
            ...event,
            id: `${i * baseMockEvents.length + j + 1}`,
            title: `${event.title} ${i + 1}`,
            start,
            end,
            schedule: newSchedule,
        };
    }),
);

const controlsContainerStyle = {
    marginBottom: 4,
    padding: 2,
    border: 1,
    borderColor: "divider",
    borderRadius: 1,
} as const;

const controlsRowStyle = {
    display: "flex",
    gap: 2,
    alignItems: "center",
    marginBottom: 2,
} as const;

const controlLabelStyle = {
    minWidth: 120,
} as const;

/**
 * Interactive story showcasing EventList with configurable options
 */
export function Interactive() {
    const [canUpdate, setCanUpdate] = useState(true);
    const [loading, setLoading] = useState(false);
    const [listState, setListState] = useState<"short" | "long" | "empty">("short");
    const [list, setList] = useState<CalendarEvent[]>(baseMockEvents);

    function handleUpdate(newList: CalendarEvent[]) {
        setList(newList);
    }

    function handleCanUpdateClick() {
        setCanUpdate(!canUpdate);
    }
    function handleLoadingClick() {
        setLoading(!loading);
    }
    function handleListLengthClick() {
        // Cycle through states: short -> long -> empty -> short
        const nextState = listState === "short" ? "long" : listState === "long" ? "empty" : "short";
        setListState(nextState);
        setList(
            nextState === "long" ? extendedMockEvents :
                nextState === "short" ? baseMockEvents :
                    [],
        );
    }

    return (
        <PageContainer>
            <ScrollBox>
                {/* Controls */}
                <Box sx={controlsContainerStyle}>
                    <Typography variant="h6" gutterBottom>Controls</Typography>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>List Length:</Typography>
                        <Button
                            variant="contained"
                            onClick={handleListLengthClick}
                            color={listState === "empty" ? "error" : "primary"}
                        >
                            {listState === "long" ? "Long List" :
                                listState === "short" ? "Short List" :
                                    "Empty List"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Can Update:</Typography>
                        <Button
                            variant={canUpdate ? "contained" : "outlined"}
                            onClick={handleCanUpdateClick}
                        >
                            {canUpdate ? "Enabled" : "Disabled"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Loading:</Typography>
                        <Button
                            variant={loading ? "contained" : "outlined"}
                            onClick={handleLoadingClick}
                        >
                            {loading ? "Loading" : "Loaded"}
                        </Button>
                    </Box>
                </Box>

                {/* EventList */}
                <EventList
                    canUpdate={canUpdate}
                    loading={loading}
                    list={loading ? [] : list}
                    handleUpdate={handleUpdate}
                />
            </ScrollBox>
        </PageContainer>
    );
}

Interactive.parameters = {
    docs: {
        description: {
            story: "An interactive demo of the EventList component with configurable list length, update permissions, and loading states.",
        },
    },
}; 
