/* eslint-disable @typescript-eslint/ban-ts-comment */
import { generatePK, type Session, type SessionUser } from "@vrooli/shared";
import { DEFAULT_THEME } from "../utils/display/theme.js";

export const API_URL = "http://localhost:5329/api";

export const signedInUserId = generatePK().toString();

export const baseSession = {
    __typename: "Session" as const,
    isLoggedIn: true,
    users: [{ theme: DEFAULT_THEME }],
    // Add other properties if your components require them
} as const;

export const loggedOutSession: Partial<Session> = {
    isLoggedIn: false,
    users: [],
};

export const signedInNoPremiumNoCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "0",
        handle: "mr_anderson420",
        hasPremium: false,
        id: signedInUserId,
        name: "Joe Anderson",
    }] as SessionUser[],
};

export const signedInNoPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "1234567",
        handle: "bugsbunnyofficial",
        hasPremium: false,
        id: signedInUserId,
        name: "Bugs Bunny",
    }] as SessionUser[],
};

export const signedInPremiumNoCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "0",
        handle: "daffyduck",
        hasPremium: true,
        id: signedInUserId,
        name: "Daffy Duck",
    }] as SessionUser[],
};


export const signedInPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        handle: "ElmerFudd",
        hasPremium: true,
        id: signedInUserId,
        name: "Elmer Fudd",
    }] as SessionUser[],
};

export const multipleUsersSession: Partial<Session> = {
    isLoggedIn: true,
    users: [
        // @ts-ignore - We know these users exist in the test context
        signedInPremiumWithCreditsSession.users?.[0],
        // @ts-ignore - We know these users exist in the test context
        { ...signedInPremiumNoCreditsSession.users?.[0], id: generatePK().toString() },
        // @ts-ignore - We know these users exist in the test context
        { ...signedInNoPremiumWithCreditsSession.users?.[0], id: generatePK().toString() },
        // @ts-ignore - We know these users exist in the test context
        { ...signedInNoPremiumNoCreditsSession.users?.[0], id: generatePK().toString() },
    ],
};
