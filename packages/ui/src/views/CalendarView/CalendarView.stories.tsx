import { http, HttpResponse } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { getMockApiUrl } from "../../__test/helpers/storybookMocking.js";
import { CalendarTabs, CalendarView } from "./CalendarView.js";

// Generate mock dates based on the current month
const now = new Date();
const calendarViewMonth = new Date(now.getFullYear(), now.getMonth(), 15); // Current month, 15th day

// Helper function to generate a date in the current month
const generateDate = (day, hour = 9, minute = 0) => {
    const date = new Date(now.getFullYear(), now.getMonth(), day);
    date.setHours(hour, minute, 0, 0);
    return date;
};

// Generate random duration between 30 and 120 minutes
const randomDuration = () => Math.floor(Math.random() * 90) + 30;

// Helper function to create schedule data
const createSchedule = (id, title, description, startTime, duration, scheduleFor, includeRecurrence = false) => {
    const endTime = new Date(startTime);
    endTime.setMinutes(endTime.getMinutes() + duration);

    // Create recurrence if requested
    const recurrences = [];
    if (includeRecurrence) {
        recurrences.push({
            __typename: "ScheduleRecurrence",
            id: `recurrence-${id}`,
            recurrenceType: "Weekly",
            interval: 1,
            dayOfWeek: startTime.getDay(),
            dayOfMonth: null,
            month: null,
            duration, // Duration in minutes
            endDate: null,
            schedule: { __typename: "Schedule", id: `schedule-${id}` },
        });
    }

    // Create the schedule object
    return {
        cursor: id.toString(),
        node: {
            __typename: "Schedule",
            id: `schedule-${id}`,
            startTime: startTime.toISOString(),
            endTime: endTime.toISOString(),
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            scheduleFor,
            recurrences,
            exceptions: [],
            meetings: scheduleFor === "Meeting" ? [{
                __typename: "Meeting",
                id: `meeting-${id}`,
                translations: [
                    {
                        __typename: "MeetingTranslation",
                        id: `meeting-translation-${id}`,
                        language: "en",
                        title,
                        description,
                    },
                ],
                you: {
                    __typename: "MeetingYou",
                    canBookmark: true,
                    canDelete: true,
                    canRead: true,
                    canUpdate: true,
                    canUseExisting: true,
                    isBookmarked: false,
                },
            }] : [],
            runs: scheduleFor === "Run" ? [{
                __typename: "Run",
                id: `routine-${id}`,
                translations: [
                    {
                        __typename: "RunTranslation",
                        id: `routine-translation-${id}`,
                        language: "en",
                        title,
                        description,
                    },
                ],
                you: {
                    __typename: "RunYou",
                    canBookmark: true,
                    canDelete: true,
                    canRead: true,
                    canUpdate: true,
                    canUseExisting: true,
                    isBookmarked: false,
                },
            }] : [],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            translations: [
                {
                    __typename: "ScheduleTranslation",
                    id: `schedule-translation-${id}`,
                    language: "en",
                    title,
                    description,
                },
            ],
            you: {
                __typename: "ScheduleYou",
                canBookmark: true,
                canDelete: true,
                canRead: true,
                canUpdate: true,
                canUseExisting: true,
                isBookmarked: false,
            },
        },
    };
};

// Generate schedules for the month
const generateSchedules = () => {
    const edges = [];
    let id = 1;

    // Get current month name
    const currentMonthName = now.toLocaleString("default", { month: "long" });
    
    // Meeting events - 2 per day for first week
    for (let day = 1; day <= 7; day++) {
        // Morning meeting
        edges.push(createSchedule(
            id++,
            `Team Standup - ${currentMonthName} ${day}`,
            "Daily team standup meeting",
            generateDate(day, 9, 30),
            30,
            "Meeting",
        ));

        // Afternoon meeting
        edges.push(createSchedule(
            id++,
            `Product Review - ${currentMonthName} ${day}`,
            "Product review with stakeholders",
            generateDate(day, 14, 0),
            60,
            "Meeting",
        ));
    }

    // Projects - 3 projects spread across mid-month
    const projectDays = [12, 17, 22];
    for (const day of projectDays) {
        edges.push(createSchedule(
            id++,
            `Project Sprint - ${currentMonthName} ${day}`,
            "Intensive work on project deliverables",
            generateDate(day, 11, 0),
            180,
            "Run",
        ));
    }

    // Routines - weekly recurring routines
    edges.push(createSchedule(
        id++,
        "Weekly Planning Routine",
        "Plan tasks and objectives for the week",
        generateDate(1, 8, 0),
        90,
        "Run",
        true, // Include recurrence
    ));

    edges.push(createSchedule(
        id++,
        "Weekly Retrospective",
        "Review progress and lessons learned",
        generateDate(5, 16, 0),
        60,
        "Run",
        true, // Include recurrence
    ));

    return edges;
};

// Create mock schedule data for testing
const mockSchedules = {
    data: {
        __typename: "ScheduleSearchResult",
        edges: generateSchedules(),
        pageInfo: {
            __typename: "PageInfo",
            endCursor: "30",
            hasNextPage: false,
        },
    },
};

// Ensure proper Date objects instead of strings for the schedules
const processScheduleDate = (scheduleData) => {
    const processedData = JSON.parse(JSON.stringify(scheduleData));

    for (const edge of processedData.data.edges) {
        const node = edge.node;
        // Convert ISO strings to Date objects where needed
        node.startTime = new Date(node.startTime);
        node.endTime = new Date(node.endTime);
        node.createdAt = new Date(node.createdAt);
        node.updatedAt = new Date(node.updatedAt);

        // Process recurrences if any
        if (node.recurrences && node.recurrences.length > 0) {
            for (const recurrence of node.recurrences) {
                if (recurrence.endDate) {
                    recurrence.endDate = new Date(recurrence.endDate);
                }
            }
        }

        // Process exceptions if any
        if (node.exceptions && node.exceptions.length > 0) {
            for (const exception of node.exceptions) {
                exception.originalStartTime = new Date(exception.originalStartTime);
                if (exception.newStartTime) exception.newStartTime = new Date(exception.newStartTime);
                if (exception.newEndTime) exception.newEndTime = new Date(exception.newEndTime);
            }
        }
    }

    return processedData;
};

// Empty schedule data for the trigger tab
const emptySchedules = {
    data: {
        __typename: "ScheduleSearchResult",
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            endCursor: null,
            hasNextPage: false,
        },
    },
};

// Mock trigger data for the trigger tab
const mockTriggers = {
    data: {
        __typename: "TriggerSearchResult",
        edges: [
            {
                cursor: "1",
                node: {
                    __typename: "Trigger",
                    id: "trigger-1",
                    name: "Weekly Planning Notification",
                    description: "Send reminder to plan the week every Monday",
                    enabled: true,
                    triggerEvent: "WeeklyOn",
                    triggerConditions: [
                        {
                            field: "dayOfWeek",
                            operator: "equals",
                            value: 1, // Monday
                        },
                        {
                            field: "timeOfDay",
                            operator: "equals",
                            value: "09:00",
                        },
                    ],
                    action: "SendNotification",
                    actionConfig: {
                        title: "Weekly Planning Time",
                        message: "Don't forget to plan your week ahead!",
                        type: "reminder",
                    },
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    you: {
                        __typename: "TriggerYou",
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    },
                },
            },
            {
                cursor: "2", 
                node: {
                    __typename: "Trigger",
                    id: "trigger-2",
                    name: "Project Completion Follow-up",
                    description: "Start retrospective routine when a project is completed",
                    enabled: true,
                    triggerEvent: "RunCompleted",
                    triggerConditions: [
                        {
                            field: "routineType",
                            operator: "equals",
                            value: "project",
                        },
                    ],
                    action: "StartRun",
                    actionConfig: {
                        routineId: "retrospective-routine-id",
                        delay: 30, // minutes
                    },
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    you: {
                        __typename: "TriggerYou",
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    },
                },
            },
            {
                cursor: "3",
                node: {
                    __typename: "Trigger", 
                    id: "trigger-3",
                    name: "Team Meeting After Sprint",
                    description: "Schedule team meeting automatically after sprint completion",
                    enabled: false,
                    triggerEvent: "RunCompleted", 
                    triggerConditions: [
                        {
                            field: "tags",
                            operator: "contains",
                            value: "sprint",
                        },
                    ],
                    action: "CreateMeeting",
                    actionConfig: {
                        title: "Sprint Retrospective Meeting",
                        duration: 60,
                        teamId: "team-id",
                        scheduledOffset: "1 day",
                    },
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    you: {
                        __typename: "TriggerYou",
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    },
                },
            },
            {
                cursor: "4",
                node: {
                    __typename: "Trigger",
                    id: "trigger-4", 
                    name: "Daily Standup Auto-Start",
                    description: "Automatically start daily standup routine every weekday",
                    enabled: true,
                    triggerEvent: "DailyAt",
                    triggerConditions: [
                        {
                            field: "timeOfDay",
                            operator: "equals",
                            value: "09:30",
                        },
                        {
                            field: "weekdaysOnly",
                            operator: "equals",
                            value: true,
                        },
                    ],
                    action: "StartRun",
                    actionConfig: {
                        routineId: "daily-standup-routine-id",
                    },
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    you: {
                        __typename: "TriggerYou",
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    },
                },
            },
            {
                cursor: "5",
                node: {
                    __typename: "Trigger",
                    id: "trigger-5",
                    name: "Note Update Documentation",
                    description: "Create documentation note when important project notes are updated",
                    enabled: true,
                    triggerEvent: "NoteUpdated", 
                    triggerConditions: [
                        {
                            field: "tags",
                            operator: "contains", 
                            value: "important",
                        },
                    ],
                    action: "CreateNote",
                    actionConfig: {
                        title: "Documentation Update Required",
                        template: "project-doc-template",
                        parentProject: "auto-detect",
                    },
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    you: {
                        __typename: "TriggerYou",
                        canDelete: true,
                        canUpdate: true,
                        canRead: true,
                    },
                },
            },
        ],
        pageInfo: {
            __typename: "PageInfo",
            endCursor: "5",
            hasNextPage: false,
        },
    },
};

export default {
    title: "Views/CalendarView",
    component: CalendarView,
    parameters: {
        msw: {
            handlers: [
                // Mock the exact query we saw in the request
                http.post(getMockApiUrl("/schedules"), async ({ request }) => {
                    console.log("Intercepted POST schedule request", request.url);
                    // Get the request body and examine it
                    let body;
                    try {
                        body = await request.json();
                        console.log("POST Request body:", body);
                    } catch (e) {
                        console.error("Error parsing request body:", e);
                    }

                    const processedData = processScheduleDate(mockSchedules);
                    console.log("Returning processed mock schedule data for POST request:", processedData);

                    // Debug the dates to ensure they're Date objects, not strings
                    console.log("Debug - First schedule start time type:",
                        typeof processedData.data.edges[0].node.startTime,
                        "Value:", processedData.data.edges[0].node.startTime);

                    return HttpResponse.json(processedData);
                }),

                // Also handle GET requests with query parameters
                http.get(`${getMockApiUrl("/schedules")}*`, ({ request }) => {
                    console.log("Intercepted GET schedule request", request.url);
                    const url = new URL(request.url);
                    console.log("GET request parameters:",
                        Array.from(url.searchParams.entries())
                            .map(([k, v]) => `${k}: ${v}`)
                            .join(", "));

                    // Parse the date ranges from the query parameters to check them
                    let startTimeFrame, endTimeFrame;
                    try {
                        if (url.searchParams.has("startTimeFrame")) {
                            startTimeFrame = JSON.parse(url.searchParams.get("startTimeFrame"));
                            console.log("Parsed startTimeFrame:", startTimeFrame);
                        }
                        if (url.searchParams.has("endTimeFrame")) {
                            endTimeFrame = JSON.parse(url.searchParams.get("endTimeFrame"));
                            console.log("Parsed endTimeFrame:", endTimeFrame);
                        }
                    } catch (e) {
                        console.error("Error parsing time frames:", e);
                    }

                    // If this has the schedule parameters, return our mock data
                    if (url.searchParams.has("scheduleForUserId")) {
                        const processedData = processScheduleDate(mockSchedules);
                        console.log("Returning processed mock schedule data for GET request:", processedData);

                        // Debug the dates to ensure they're Date objects, not strings
                        console.log("Debug - First schedule start time type:",
                            typeof processedData.data.edges[0].node.startTime,
                            "Value:", processedData.data.edges[0].node.startTime);

                        return HttpResponse.json(processedData);
                    }

                    return HttpResponse.json({ data: { edges: [] } });
                }),

                // Mock trigger endpoints for the trigger tab
                http.post(`${API_URL}/triggers`, async ({ request }) => {
                    console.log("Intercepted POST trigger request", request.url);
                    let body;
                    try {
                        body = await request.json();
                        console.log("POST Trigger Request body:", body);
                    } catch (e) {
                        console.error("Error parsing trigger request body:", e);
                    }

                    console.log("Returning mock trigger data for POST request:", mockTriggers);
                    return HttpResponse.json(mockTriggers);
                }),

                http.get(`${API_URL}/triggers*`, ({ request }) => {
                    console.log("Intercepted GET trigger request", request.url);
                    const url = new URL(request.url);
                    console.log("GET trigger request parameters:",
                        Array.from(url.searchParams.entries())
                            .map(([k, v]) => `${k}: ${v}`)
                            .join(", "));

                    console.log("Returning mock trigger data for GET request:", mockTriggers);
                    return HttpResponse.json(mockTriggers);
                }),

                // Handle any GraphQL requests that might be happening
                http.post(getMockApiUrl("/graphql"), async ({ request }) => {
                    console.log("Intercepted GraphQL request");
                    const body = await request.json();

                    // If this query mentions schedules, return our mock data
                    if (body?.query?.includes("Schedule")) {
                        const processedData = processScheduleDate(mockSchedules);
                        console.log("Returning processed mock schedule data in GraphQL format:", processedData);
                        return HttpResponse.json({
                            data: {
                                scheduleSearch: processedData.data,
                            },
                        });
                    }

                    // If this query mentions triggers, return trigger mock data
                    if (body?.query?.includes("Trigger")) {
                        console.log("Returning mock trigger data in GraphQL format:", mockTriggers);
                        return HttpResponse.json({
                            data: {
                                triggerSearch: mockTriggers.data,
                            },
                        });
                    }

                    return HttpResponse.json({ data: {} });
                }),
            ],
        },
    },
    decorators: [
        (Story) => {
            // Add a debug helper to window object to monitor schedule processing
            if (typeof window !== "undefined") {
                window.debugCalendar = {
                    getMockData: () => processScheduleDate(mockSchedules),
                    originalMockData: mockSchedules,
                };
            }

            return <Story />;
        },
    ],
};

export function SignedInNoPremiumNoCredits() {
    return (
        <CalendarView display="Page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <CalendarView display="Page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <CalendarView display="Page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <CalendarView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

export function CalendarTabView() {
    // Create a debugging div with helpful instructions
    const debugStyles = {
        position: "fixed",
        bottom: "10px",
        left: "10px",
        background: "rgba(0,0,0,0.7)",
        color: "white",
        padding: "10px",
        borderRadius: "5px",
        fontSize: "12px",
        zIndex: 9999,
        maxWidth: "350px",
    };

    return (
        <>
            <div style={debugStyles}>
                <h3 style={{ margin: "0 0 8px 0" }}>Calendar Debug Info</h3>
                <div>Open browser console and type:</div>
                <code>window.debugCalendar.getMockData()</code>
                <div style={{ marginTop: "8px" }}>to see the processed mock data</div>
            </div>
            <CalendarView
                display="Page"
                initialTab={CalendarTabs.CALENDAR}
            />
        </>
    );
}
CalendarTabView.storyName = "Calendar Tab View";
CalendarTabView.parameters = {
    session: signedInPremiumWithCreditsSession,
};

export function TriggerTabView() {
    return (
        <CalendarView display="Page" initialTab={CalendarTabs.TRIGGER} />
    );
}
TriggerTabView.storyName = "Trigger Tab View";
TriggerTabView.parameters = {
    session: signedInPremiumWithCreditsSession,
};
