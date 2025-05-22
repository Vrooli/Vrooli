import { InboxPageTabOption, type Notification, type NotificationSearchResult, type Success, endpointsNotification, generatePKString } from "@local/shared";
import { HttpResponse, http } from "msw";
import { signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { apiUrlBase, restBase } from "../../utils/consts.js";
import { InboxView } from "./InboxView.js";

// Constants for magic numbers
const MILLISECONDS_IN_SECOND = 1000;
const SECONDS_IN_MINUTE = 60;
const MINUTES_IN_HOUR = 60;
const HOURS_IN_DAY = 24;
const ONE_DAY_MS = HOURS_IN_DAY * MINUTES_IN_HOUR * SECONDS_IN_MINUTE * MILLISECONDS_IN_SECOND;
const TWO_DAYS_MS = 2 * ONE_DAY_MS;
const DEFAULT_PAGE_SIZE = 20; // Matching the actual call observed

// Mock Notification Data
const mockNotifications: Notification[] = [
    {
        __typename: "Notification",
        id: generatePKString(),
        createdAt: new Date().toISOString(),
        title: "Agent Task Completed",
        description: "Your agent 'Alpha' completed its task successfully.",
        category: "AGENT",
        isRead: false,
        link: "/agent/task/...",
        imgLink: undefined,
    },
    {
        __typename: "Notification",
        id: generatePKString(),
        createdAt: new Date(Date.now() - ONE_DAY_MS).toISOString(),
        title: "New Comment",
        description: "New comment on your document 'Project Proposal'.",
        category: "DOCUMENT",
        isRead: false,
        link: "/document/...",
        imgLink: undefined,
    },
    {
        __typename: "Notification",
        id: generatePKString(),
        createdAt: new Date(Date.now() - TWO_DAYS_MS).toISOString(),
        title: "Low Credit Balance",
        description: "Credit balance is low. Please top up.",
        category: "BILLING",
        isRead: true,
        link: "/billing",
        imgLink: undefined,
    },
];

// MSW Handlers
const handlers = [
    // Mock findMany for Notifications (GET request)
    http.get(`${apiUrlBase}${restBase}/notifications`, ({ request }) => {
        const url = new URL(request.url);
        const take = parseInt(url.searchParams.get("take") || `${DEFAULT_PAGE_SIZE}`, 10);
        const skip = parseInt(url.searchParams.get("skip") || "0", 10);

        const paginatedNotifications = mockNotifications.slice(skip, skip + take);

        // Construct the correct SearchResult structure
        const responseData: NotificationSearchResult = {
            __typename: "NotificationSearchResult",
            edges: paginatedNotifications.map(notification => ({
                __typename: "NotificationEdge", // Add edge typename
                node: notification,
                cursor: notification.id, // Use notification ID as cursor
            })),
            pageInfo: { // Add basic PageInfo
                __typename: "PageInfo",
                hasNextPage: skip + take < mockNotifications.length,
                hasPreviousPage: skip > 0,
                startCursor: paginatedNotifications[0]?.id ?? null,
                endCursor: paginatedNotifications[paginatedNotifications.length - 1]?.id ?? null,
            },
        };

        // Log the correctly formatted response
        console.log("[notificationDisplayFix] MSW returning formatted response:", responseData);

        // Return data directly (fetchData expects the raw response data)
        return HttpResponse.json(responseData);
    }),
    // Mock markAllAsRead (PUT request)
    http.put(`${apiUrlBase}${endpointsNotification.markAllAsRead.endpoint}`, () => {
        mockNotifications.forEach(n => n.isRead = true);
        const response: Success = { __typename: "Success", success: true };
        // Return data directly for Success endpoint
        return HttpResponse.json(response);
    }),
    // Mock findMany for Chats (POST request - assuming this one is correct)
    http.post(`${apiUrlBase}${restBase}/chat/findMany`, () => {
        return HttpResponse.json({
            __typename: "ChatSearchResult", // Example
            edges: [],
            pageInfo: { __typename: "PageInfo", hasNextPage: false, hasPreviousPage: false, startCursor: null, endCursor: null },
        });
    }),
];

export default {
    title: "Views/Main/InboxView",
    component: InboxView,
    parameters: {
        // Apply handlers globally to all stories in this file
        msw: {
            handlers,
        },
        // Pass initial tab state via URL search params for specific stories
        initialRoute: {
            search: `?tab=${InboxPageTabOption.Notification}`,
        },
    },
};

export function SignedInNoPremiumNoCredits() {
    return (
        <InboxView display="Page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    initialRoute: {
        search: `?tab=${InboxPageTabOption.Notification}`,
    },
    msw: { handlers },
};

export function SignedInNoPremiumWithCredits() {
    return (
        <InboxView display="Page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
    initialRoute: {
        search: `?tab=${InboxPageTabOption.Notification}`,
    },
    msw: { handlers },
};

export function SignedInPremiumNoCredits() {
    return (
        <InboxView display="Page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
    initialRoute: {
        search: `?tab=${InboxPageTabOption.Notification}`,
    },
    msw: { handlers },
};

export function SignedInPremiumWithCredits() {
    return (
        <InboxView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
    initialRoute: {
        search: `?tab=${InboxPageTabOption.Notification}`,
    },
    msw: { handlers },
};
