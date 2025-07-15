import { HttpResponse, http } from "msw";
import { AwardCategory } from "@vrooli/shared";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { getMockApiUrl } from "../../__test/helpers/storybookMocking.js";
import { AwardsView } from "./AwardsView.js";

// Generate mock award data with varying progress levels
const generateMockAwards = () => {
    const awardCategories = Object.values(AwardCategory);
    const awards = [];

    // Create some completed awards (progress should match the highest tier available)
    awards.push({
        __typename: "Award",
        id: "award-1",
        category: AwardCategory.RoutineCreate,
        progress: 1000, // Fully completed - this matches the highest tier for RoutineCreate
        createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Routine Creator Master",
        description: "Created 1000 routines",
        tierCompletedAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    });

    awards.push({
        __typename: "Award",
        id: "award-2", 
        category: AwardCategory.Reputation,
        progress: 10000, // Fully completed - this matches the highest tier for Reputation
        createdAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Reputation Legend",
        description: "Reached 10000 reputation points",
        tierCompletedAt: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString(),
    });

    // Create some in-progress awards with different progress levels
    awards.push({
        __typename: "Award",
        id: "award-3",
        category: AwardCategory.RunRoutine,
        progress: 87, // Almost completed next tier (target: 100)
        createdAt: new Date(Date.now() - 45 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Routine Runner",
        description: "Run routines to automate tasks",
        tierCompletedAt: null,
    });

    awards.push({
        __typename: "Award",
        id: "award-4",
        category: AwardCategory.CommentCreate,
        progress: 15, // Mid progress (target: 25)
        createdAt: new Date(Date.now() - 20 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Commenter",
        description: "Created helpful comments",
        tierCompletedAt: null,
    });

    awards.push({
        __typename: "Award",
        id: "award-5",
        category: AwardCategory.Streak,
        progress: 45, // Good progress (target: 100)
        createdAt: new Date(Date.now() - 50 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Streak Keeper",
        description: "Maintained daily activity streak",
        tierCompletedAt: null,
    });

    awards.push({
        __typename: "Award",
        id: "award-6",
        category: AwardCategory.ProjectCreate,
        progress: 2, // Early progress (target: 5)
        createdAt: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
        updatedAt: new Date().toISOString(),
        title: "Project Creator",
        description: "Created collaborative projects",
        tierCompletedAt: null,
    });

    return awards;
};

export default {
    title: "Views/AwardsView",
    component: AwardsView,
    parameters: {
        msw: {
            handlers: [
                // Awards endpoint
                http.get(`${getMockApiUrl("/awards")}*`, () => {
                    const mockAwards = generateMockAwards();
                    
                    return HttpResponse.json({
                        data: {
                            edges: mockAwards.map((award, index) => ({
                                cursor: index.toString(),
                                node: award,
                            })),
                            pageInfo: {
                                __typename: "PageInfo",
                                endCursor: (mockAwards.length - 1).toString(),
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
        <AwardsView />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <AwardsView />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <AwardsView />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <AwardsView />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <AwardsView />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};
