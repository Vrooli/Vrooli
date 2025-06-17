import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { StatsSiteView } from "./StatsSiteView.js";

export default {
    title: "Views/StatsSiteView",
    component: StatsSiteView,
    parameters: {
        msw: {
            handlers: [
                // Site statistics endpoint (reuse the same mock data from AdminView)
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
