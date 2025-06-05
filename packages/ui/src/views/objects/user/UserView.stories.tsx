/* eslint-disable no-magic-numbers */
import { endpointsUser, generatePK, getObjectUrl, type User, type UserTranslation } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { UserView } from "./UserView.js";

// Create simplified mock data for User responses
const mockUserBase: Omit<User, "id" | "handle" | "name" | "isBot" | "translations" | "createdAt" | "updatedAt" | "you" | "bookmarks" | "reportsReceivedCount"> = {
    __typename: "User" as const,
    profileImage: null,
    bannerImage: null,
    tags: [],
    resources: [],
    comments: [],
    reportsReceived: [],
    views: Math.floor(Math.random() * 100_000),
};

const mockUserTranslationBase: Omit<UserTranslation, "id" | "language" | "name" | "bio"> = {
    __typename: "UserTranslation" as const,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    occupation: `Occupation ${Math.floor(Math.random() * 1000)}`,
    persona: `Persona ${Math.floor(Math.random() * 1000)}`,
    startingMessage: `Start Message ${Math.floor(Math.random() * 1000)}`,
    tone: `Tone ${Math.floor(Math.random() * 1000)}`,
    keyPhrases: `Key Phrases ${Math.floor(Math.random() * 1000)}`,
    domainKnowledge: `Domain Knowledge ${Math.floor(Math.random() * 1000)}`,
    bias: `Bias ${Math.floor(Math.random() * 1000)}`,
    creativity: Math.random(),
    verbosity: Math.random(),
};

function createMockUser(isBot: boolean, isOwner: boolean): User {
    const userId = generatePK().toString();
    const handle = `user${Math.floor(Math.random() * 1000)}`;
    const name = `User ${Math.floor(Math.random() * 1000)}`;
    const createdAt = new Date(Date.now() - Math.random() * 1000 * 60 * 60 * 24 * 365).toISOString(); // Random date in the last year
    const updatedAt = new Date().toISOString();

    return {
        ...mockUserBase,
        id: userId,
        handle,
        name,
        isBot,
        createdAt,
        updatedAt,
        bookmarks: Math.floor(Math.random() * 100),
        reportsReceivedCount: Math.floor(Math.random() * 10),
        translations: [{
            ...mockUserTranslationBase,
            id: generatePK().toString(),
            language: "en",
            name,
            bio: `This is the bio for ${name}. Lorem ipsum dolor sit amet. ${isBot ? "I am a bot." : ""}`,
        }],
        you: {
            canBookmark: true,
            canDelete: isOwner,
            canUpdate: isOwner,
            canRead: true,
            isBookmarked: Math.random() > 0.5,
            isOwner,
            isStarred: Math.random() > 0.5,
        },
        // Add other necessary fields or simplify based on what UserView actually uses
    };
}

const mockUserDataRegular = createMockUser(false, false);
const mockUserDataBot = createMockUser(true, false);
const mockUserDataOwner = createMockUser(false, true);


export default {
    title: "Views/Objects/User/UserView",
    component: UserView,
    parameters: {
        // Default session for most stories unless overridden
        session: signedInNoPremiumNoCreditsSession,
    },
};

export function NoResult() {
    return (
        <UserView display="Page" />
    );
}
NoResult.parameters = {
    msw: {
        handlers: [
            // Ensure it returns a 404 or empty response for any user findOne request
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return new HttpResponse(null, { status: 404 });
            }),
        ],
    },
    route: {
        // Use a non-existent user ID or handle
        path: `/user/nonexistentuser-${generatePK().toString()}`,
    },
};

export function Loading() {
    return (
        <UserView display="Page" />
    );
}
Loading.parameters = {
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120_000));
                // Return regular user data after delay
                return HttpResponse.json({ data: mockUserDataRegular });
            }),
        ],
    },
    route: {
        path: getObjectUrl(mockUserDataRegular), // Route for the regular user
    },
};

export function SignInWithRegularUser() {
    return (
        <UserView display="Page" />
    );
}
SignInWithRegularUser.parameters = {
    session: signedInPremiumWithCreditsSession, // User is signed in
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockUserDataRegular });
            }),
        ],
    },
    route: {
        path: getObjectUrl(mockUserDataRegular),
    },
};

export function SignInWithBotUser() {
    return (
        <UserView display="Page" />
    );
}
SignInWithBotUser.parameters = {
    session: signedInPremiumWithCreditsSession, // User is signed in
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockUserDataBot });
            }),
        ],
    },
    route: {
        path: getObjectUrl(mockUserDataBot),
    },
};


export function LoggedOutWithRegularUser() {
    return (
        <UserView display="Page" />
    );
}
LoggedOutWithRegularUser.parameters = {
    session: loggedOutSession, // User is logged out
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockUserDataRegular });
            }),
        ],
    },
    route: {
        path: getObjectUrl(mockUserDataRegular),
    },
};

export function OwnProfile() {
    return (
        <UserView display="Page" />
    );
}
OwnProfile.parameters = {
    session: signedInPremiumWithCreditsSession, // User is signed in
    msw: {
        handlers: [
            // Mock the /profile endpoint specificially if UserView uses it
            // Otherwise, mock findOne with owner data
            http.get(`${API_URL}/v2${endpointsUser.profile.endpoint}`, () => {
                // Assuming profile endpoint returns the owner's data
                return HttpResponse.json({ data: mockUserDataOwner });
            }),
            // Fallback for findOne if needed, returning owner data
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockUserDataOwner });
            }),
        ],
    },
    route: {
        // Use a specific path that UserView interprets as the current user's profile,
        // e.g., "/profile" or the owner's actual user path
        path: "/profile",
        // Alternatively, if it resolves based on ID/handle:
        // path: getObjectUrl(mockUserDataOwner),
    },
};

// Example of Dialog display
export function DialogDisplay() {
    return (
        <UserView display="Dialog" onClose={() => alert("Close Dialog")} />
    );
}
DialogDisplay.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsUser.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockUserDataBot }); // Show a bot in the dialog
            }),
        ],
    },
    route: {
        path: getObjectUrl(mockUserDataBot),
    },
}; 
