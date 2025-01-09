import { User, UserTranslation, UserYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const userTranslation: ApiPartial<UserTranslation> = {
    common: {
        id: true,
        language: true,
        bio: true,
    },
};

export const userYou: ApiPartial<UserYou> = {
    common: {
        canDelete: true,
        canReport: true,
        canUpdate: true,
        isBookmarked: true,
        isViewed: true,
    },
};

export const user: ApiPartial<User> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        bannerImage: true,
        handle: true,
        isBot: true,
        isBotDepictingPerson: true,
        name: true,
        profileImage: true,
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
        created_at: true,
        updated_at: true,
        bannerImage: true,
        handle: true,
        isBot: true,
        isBotDepictingPerson: true,
        name: true,
        profileImage: true,
    },
};

export const profile: ApiPartial<User> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        bannerImage: true,
        handle: true,
        isPrivate: true,
        isPrivateApis: true,
        isPrivateApisCreated: true,
        isPrivateMemberships: true,
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
        isPrivateTeamsCreated: true,
        isPrivateBookmarks: true,
        isPrivateVotes: true,
        name: true,
        notificationSettings: true,
        profileImage: true,
        theme: true,
        emails: async () => rel((await import("./email")).email, "full"),
        focusModes: async () => rel((await import("./focusMode")).focusMode, "full"),
        phones: async () => rel((await import("./phone")).phone, "full"),
        pushDevices: async () => rel((await import("./pushDevice")).pushDevice, "full"),
        wallets: async () => rel((await import("./wallet")).wallet, "common"),
        notifications: async () => rel((await import("./notification")).notification, "full"),
        translations: () => rel(userTranslation, "full"),
        you: () => rel(userYou, "full"),
    },
};
