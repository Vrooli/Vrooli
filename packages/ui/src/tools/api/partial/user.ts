import { User, UserTranslation, UserYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const userTranslation: GqlPartial<UserTranslation> = {
    __typename: "UserTranslation",
    common: {
        id: true,
        language: true,
        bio: true,
    },
    full: {},
    list: {},
};

export const userYou: GqlPartial<UserYou> = {
    __typename: "UserYou",
    common: {
        canDelete: true,
        canReport: true,
        canUpdate: true,
        isBookmarked: true,
        isViewed: true,
    },
    full: {},
    list: {},
};

export const user: GqlPartial<User> = {
    __typename: "User",
    common: {
        id: true,
        created_at: true,
        handle: true,
        isBot: true,
        name: true,
        bookmarks: true,
        reportsReceivedCount: true,
        you: () => rel(userYou, "full"),
    },
    full: {
        botSettings: true,
        translations: () => rel(userTranslation, "full"),
    },
    list: {
        translations: () => rel(userTranslation, "list"),
    },
    nav: {
        id: true,
        isBot: true,
        name: true,
        handle: true,
    },
};

export const profile: GqlPartial<User> = {
    __typename: "User",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        handle: true,
        isPrivate: true,
        isPrivateApis: true,
        isPrivateApisCreated: true,
        isPrivateMemberships: true,
        isPrivateOrganizationsCreated: true,
        isPrivateProjects: true,
        isPrivateProjectsCreated: true,
        isPrivatePullRequests: true,
        isPrivateQuestionsAnswered: true,
        isPrivateQuestionsAsked: true,
        isPrivateQuizzesCreated: true,
        isPrivateRoles: true,
        isPrivateRoutines: true,
        isPrivateRoutinesCreated: true,
        isPrivateStandards: true,
        isPrivateStandardsCreated: true,
        isPrivateBookmarks: true,
        isPrivateVotes: true,
        name: true,
        theme: true,
        emails: async () => rel((await import("./email")).email, "full"),
        focusModes: async () => rel((await import("./focusMode")).focusMode, "full"),
        pushDevices: async () => rel((await import("./pushDevice")).pushDevice, "full"),
        wallets: async () => rel((await import("./wallet")).wallet, "common"),
        notifications: async () => rel((await import("./notification")).notification, "full"),
        notificationSettings: true,
        translations: () => rel(userTranslation, "full"),
        you: () => rel(userYou, "full"),
    },
    full: {},
    list: {},
};
