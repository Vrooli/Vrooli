import { Session, SessionUser, uuid } from "@local/shared";
import { DEFAULT_THEME } from "../utils/display/theme.js";

export const API_URL = "http://localhost:5329/api";

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
        hasPremium: false,
        id: uuid(),
    }] as SessionUser[],
};

export const signedInNoPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "1234567",
        hasPremium: false,
        id: uuid(),
    }] as SessionUser[],
};

export const signedInPremiumNoCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "0",
        hasPremium: true,
        id: uuid(),
    }] as SessionUser[],
};


export const signedInPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        hasPremium: true,
        id: uuid(),
    }] as SessionUser[],
};
