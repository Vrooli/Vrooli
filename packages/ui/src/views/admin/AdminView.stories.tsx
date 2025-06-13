import { HttpResponse, http } from "msw";
import { SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import type { Session, SessionUser } from "@vrooli/shared";
import { API_URL, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { AdminView } from "./AdminView.js";

export default {
    title: "Views/Admin/AdminView",
    component: AdminView,
    parameters: {
        msw: {
            handlers: [
                // Site statistics endpoint
                http.get(`${API_URL}/v2/stats/site*`, ({ request }) => {
                    const url = new URL(request.url);
                    const params = new URLSearchParams(url.search);
                    
                    // Generate realistic trending data based on time
                    const generateTrendingValue = (baseValue: number, index: number, trend: "up" | "down" | "stable") => {
                        const variation = Math.random() * 20 - 10; // ±10% variation
                        const trendMultiplier = trend === "up" ? (1 + index * 0.02) : 
                                              trend === "down" ? (1 - index * 0.01) : 1;
                        return Math.max(0, Math.floor(baseValue * trendMultiplier + variation));
                    };

                    return HttpResponse.json({
                        data: {
                            edges: Array.from({ length: 24 }, (_, i) => ({
                                cursor: i.toString(),
                                node: {
                                    __typename: "StatsSite",
                                    id: `stats-${i}`,
                                    periodStart: new Date(Date.now() - (23 - i) * 60 * 60 * 1000).toISOString(),
                                    periodEnd: new Date(Date.now() - (22 - i) * 60 * 60 * 1000).toISOString(),
                                    activeUsers: generateTrendingValue(75, i, "up"),
                                    verifiedEmailsCreated: generateTrendingValue(8, i, "stable"),
                                    verifiedWalletsCreated: generateTrendingValue(3, i, "up"),
                                    teamsCreated: generateTrendingValue(2, i, "up"),
                                    runsStarted: generateTrendingValue(35, i, "up"),
                                    runsCompleted: generateTrendingValue(28, i, "up"),
                                    routinesCreated: generateTrendingValue(5, i, "up"),
                                    resourcesCreatedByType: JSON.stringify({
                                        Routine: generateTrendingValue(5, i, "up"),
                                        Api: generateTrendingValue(3, i, "stable"),
                                        Project: generateTrendingValue(4, i, "up"),
                                    }),
                                    resourcesCompletedByType: JSON.stringify({
                                        Routine: generateTrendingValue(4, i, "up"),
                                        Api: generateTrendingValue(2, i, "stable"),
                                        Project: generateTrendingValue(3, i, "up"),
                                    }),
                                },
                            })),
                            pageInfo: {
                                __typename: "PageInfo",
                                endCursor: "23",
                                hasNextPage: false,
                            },
                        },
                    });
                }),

                // Admin site stats endpoint (used by CreditStatsPanel)
                http.get(`${API_URL}/v2/admin/siteStats`, () => {
                    return HttpResponse.json({
                        data: {
                            totalCreditsEarned: 1234567,
                            totalCreditsSpent: 987654,
                            totalDonations: 45000,
                            totalSubscriptions: 156,
                            monthlyRecurringRevenue: 3900,
                            averageCreditsPerUser: 2500,
                            topSpendingCategories: [
                                { category: "AI Processing", amount: 456789 },
                                { category: "Data Storage", amount: 234567 },
                                { category: "API Calls", amount: 123456 },
                            ],
                        },
                    });
                }),

                // User management endpoints
                http.get(`${API_URL}/v2/admin/users*`, () => {
                    return HttpResponse.json({
                        data: {
                            edges: Array.from({ length: 10 }, (_, i) => ({
                                cursor: i.toString(),
                                node: {
                                    __typename: "User",
                                    id: `user-${i}`,
                                    name: `User ${i + 1}`,
                                    handle: `user${i + 1}`,
                                    isBot: i % 5 === 0,
                                    isPrivate: i % 3 === 0,
                                    isDeleted: false,
                                    isBanned: i === 9,
                                    isAdmin: i === 0,
                                    credits: Math.floor(Math.random() * 10000).toString(),
                                    reportsReceivedCount: Math.floor(Math.random() * 5),
                                    createdAt: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString(),
                                    updatedAt: new Date().toISOString(),
                                },
                            })),
                            pageInfo: {
                                __typename: "PageInfo",
                                endCursor: "9",
                                hasNextPage: false,
                            },
                        },
                    });
                }),

                // Reports endpoint
                http.get(`${API_URL}/v2/reports*`, () => {
                    return HttpResponse.json({
                        data: {
                            edges: Array.from({ length: 5 }, (_, i) => ({
                                cursor: i.toString(),
                                node: {
                                    __typename: "Report",
                                    id: `report-${i}`,
                                    reason: ["Spam", "Harassment", "Inappropriate Content", "Copyright", "Other"][i],
                                    details: `This is a sample report ${i + 1}`,
                                    status: ["Active", "Under Review", "Resolved", "Dismissed"][i % 4],
                                    createdAt: new Date(Date.now() - i * 24 * 60 * 60 * 1000).toISOString(),
                                    from: {
                                        __typename: "User",
                                        id: `reporter-${i}`,
                                        name: `Reporter ${i + 1}`,
                                        handle: `reporter${i + 1}`,
                                    },
                                    responses: [],
                                },
                            })),
                            pageInfo: {
                                __typename: "PageInfo",
                                endCursor: "4",
                                hasNextPage: false,
                            },
                        },
                    });
                }),

                // Admin user management endpoints
                http.put(`${API_URL}/v2/admin/user/status`, async ({ request }) => {
                    const body = await request.json();
                    return HttpResponse.json({
                        data: {
                            success: true,
                            message: `User status updated to ${body.status}`,
                        },
                    });
                }),

                http.post(`${API_URL}/v2/admin/user/resetPassword`, async ({ request }) => {
                    const body = await request.json();
                    return HttpResponse.json({
                        data: {
                            success: true,
                            message: `Password reset email sent to user ${body.userId}`,
                        },
                    });
                }),

                // General user update endpoint
                http.put(`${API_URL}/v2/users/:id`, async ({ params, request }) => {
                    const body = await request.json();
                    return HttpResponse.json({
                        data: {
                            id: params.id,
                            ...body,
                            updatedAt: new Date().toISOString(),
                        },
                    });
                }),

                // System settings endpoints
                http.get(`${API_URL}/v2/admin/settings`, () => {
                    return HttpResponse.json({
                        data: {
                            maintenanceMode: false,
                            registrationEnabled: true,
                            maxUsersPerTeam: 50,
                            defaultCredits: 1000,
                            premiumPrice: 2500,
                        },
                    });
                }),

                http.put(`${API_URL}/v2/admin/settings`, async ({ request }) => {
                    const body = await request.json();
                    return HttpResponse.json({
                        data: body,
                    });
                }),
            ],
        },
    },
};

// Create an admin session using the seeded admin ID
const adminSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        id: SEEDED_PUBLIC_IDS.Admin,
        credits: "999999",
        handle: "admin",
        hasPremium: true,
        name: "Admin User",
        languages: ["en"],
        isAdmin: true,
    }] as SessionUser[],
};

// Create a non-admin session for comparison
const nonAdminSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        id: "regular-user-id",
        credits: "100",
        handle: "regularuser",
        hasPremium: false,
        name: "Regular User",
        languages: ["en"],
    }] as SessionUser[],
};

/**
 * Admin user viewing the admin dashboard with full page navigation
 */
export function AdminUserView() {
    return <AdminView display="Page" />;
}
AdminUserView.parameters = {
    session: adminSession,
};

/**
 * Non-admin user attempting to access admin dashboard
 * Should show access denied message
 */
export function NonAdminUserView() {
    return <AdminView display="Page" />;
}
NonAdminUserView.parameters = {
    session: nonAdminSession,
};

/**
 * Loading state while checking admin status
 */
export function LoadingState() {
    return <AdminView display="Page" />;
}
LoadingState.parameters = {
    session: {
        ...adminSession,
        loading: true,
    },
};

/**
 * Admin view in dialog mode
 */
export function AdminDialogView() {
    return <AdminView display="Dialog" />;
}
AdminDialogView.parameters = {
    session: signedInPremiumWithCreditsSession,
};