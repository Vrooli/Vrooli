import { Session, SessionUser, uuid } from "@local/shared";
import { DEFAULT_THEME } from "../utils/display/theme.js";

export const API_URL = "http://localhost:5329/api";

export const signedInUserId = uuid();

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
        handle: "myHandle",
        hasPremium: false,
        id: signedInUserId,
        name: "Joe Anderson",
    }] as SessionUser[],
};

export const signedInNoPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "1234567",
        handle: "myHandle",
        hasPremium: false,
        id: signedInUserId,
        name: "Joe Anderson",
    }] as SessionUser[],
};

export const signedInPremiumNoCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "0",
        handle: "myHandle",
        hasPremium: true,
        id: signedInUserId,
        name: "Joe Anderson",
    }] as SessionUser[],
};


export const signedInPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        handle: "myHandle",
        hasPremium: true,
        id: signedInUserId,
        name: "Joe Anderson",
    }] as SessionUser[],
};

export const multipleUsersSession: Partial<Session> = {
    isLoggedIn: true,
    users: [
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        signedInPremiumWithCreditsSession.users[0]!,
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        { ...signedInPremiumNoCreditsSession.users[0]!, id: uuid() },
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        { ...signedInNoPremiumWithCreditsSession.users[0]!, id: uuid() },
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        { ...signedInNoPremiumNoCreditsSession.users[0]!, id: uuid() },
    ],
};
