import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { getMockApiUrl } from "../../__test/helpers/storybookMocking.js";
import { StatsSiteView } from "./StatsSiteView.js";

export default {
    title: "Views/StatsSiteView",
    component: StatsSiteView,
    parameters: {
        msw: {
            handlers: [
                // Site statistics endpoint (reuse the same mock data from AdminView)
                http.get(`${getMockApiUrl("/stats/site")}*`, ({ request }) => {
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

                // Health check endpoint
                http.get(`${getMockApiUrl("")}/healthcheck`, () => {
                    return HttpResponse.json({
                        status: "Operational",
                        version: "1.0.0",
                        services: {
                            api: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            bus: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            cronJobs: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            database: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            i18n: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            llm: {
                                "openai": { healthy: true, status: "Operational", lastChecked: Date.now() },
                                "anthropic": { healthy: false, status: "Degraded", lastChecked: Date.now() },
                            },
                            mcp: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            memory: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            queues: {
                                "email": { healthy: true, status: "Operational", lastChecked: Date.now() },
                                "jobs": { healthy: true, status: "Operational", lastChecked: Date.now() },
                            },
                            redis: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            ssl: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            stripe: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            system: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            websocket: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            imageStorage: { healthy: true, status: "Operational", lastChecked: Date.now() },
                            embeddingService: { healthy: true, status: "Operational", lastChecked: Date.now() },
                        },
                        timestamp: Date.now(),
                    });
                }),

                // Metrics endpoint
                http.get(`${getMockApiUrl("")}/metrics`, () => {
                    return HttpResponse.json({
                        timestamp: Date.now(),
                        uptime: 7200, // 2 hours
                        system: {
                            cpu: {
                                usage: Math.floor(Math.random() * 50) + 20, // 20-70%
                                cores: 4,
                                loadAvg: [0.5, 0.7, 0.9],
                            },
                            memory: {
                                heapUsed: 256 * 1024 * 1024, // 256MB
                                heapTotal: 512 * 1024 * 1024, // 512MB
                                heapUsedPercent: 50,
                                rss: 300 * 1024 * 1024,
                                external: 10 * 1024 * 1024,
                                arrayBuffers: 5 * 1024 * 1024,
                            },
                            disk: {
                                total: 100 * 1024 * 1024 * 1024, // 100GB
                                used: 45 * 1024 * 1024 * 1024, // 45GB
                                usagePercent: 45,
                            },
                        },
                        application: {
                            nodeVersion: "v18.17.0",
                            pid: 12345,
                            platform: "linux",
                            environment: "development",
                            websockets: {
                                connections: Math.floor(Math.random() * 100),
                                rooms: Math.floor(Math.random() * 20),
                            },
                            queues: {
                                email: {
                                    waiting: Math.floor(Math.random() * 10),
                                    active: Math.floor(Math.random() * 5),
                                    completed: Math.floor(Math.random() * 1000),
                                    failed: Math.floor(Math.random() * 10),
                                    delayed: Math.floor(Math.random() * 5),
                                    total: 1050,
                                },
                                jobs: {
                                    waiting: Math.floor(Math.random() * 20),
                                    active: Math.floor(Math.random() * 10),
                                    completed: Math.floor(Math.random() * 500),
                                    failed: Math.floor(Math.random() * 5),
                                    delayed: Math.floor(Math.random() * 2),
                                    total: 537,
                                },
                            },
                            llmServices: {
                                "openai": { state: "Active" },
                                "anthropic": { state: "Cooldown", cooldownUntil: Date.now() + 300000 },
                                "local": { state: "Active" },
                            },
                        },
                        database: {
                            connected: true,
                            poolSize: 10,
                        },
                        redis: {
                            connected: true,
                            memory: {
                                used: 64 * 1024 * 1024, // 64MB
                                peak: 128 * 1024 * 1024, // 128MB
                            },
                            stats: {
                                connections: 25,
                                commands: 10000,
                            },
                        },
                        api: {
                            requestsTotal: 5000,
                            responseTimes: {
                                min: 10,
                                max: 500,
                                avg: 85,
                            },
                        },
                    });
                }),
            ],
        },
    },
};

export function LoggedOut() {
    return (
        <StatsSiteView display="Page" />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <StatsSiteView display="Page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <StatsSiteView display="Page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <StatsSiteView display="Page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <StatsSiteView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
