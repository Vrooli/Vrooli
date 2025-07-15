import { type User, type UserTranslation, type UserYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

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
        publicId: true,
        createdAt: true,
        updatedAt: true,
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
        // @ts-expect-error - JSONB field - select entire JSON object
        botSettings: true,
        translations: () => rel(userTranslation, "full"),
    },
    list: {
        translations: () => rel(userTranslation, "list"),
    },
    nav: {
        id: true,
        createdAt: true,
        updatedAt: true,
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
        createdAt: true,
        updatedAt: true,
        bannerImage: true,
        handle: true,
        isPrivate: true,
        isPrivateMemberships: true,
        isPrivatePullRequests: true,
        isPrivateResources: true,
        isPrivateResourcesCreated: true,
        isPrivateTeamsCreated: true,
        isPrivateBookmarks: true,
        isPrivateVotes: true,
        name: true,
        notificationSettings: true,
        profileImage: true,
        theme: true,
        apiKeys: async () => rel((await import("./apiKey.js")).apiKey, "full"),
        apiKeysExternal: async () => rel((await import("./apiKeyExternal.js")).apiKeyExternal, "full"),
        emails: async () => rel((await import("./email.js")).email, "full"),
        phones: async () => rel((await import("./phone.js")).phone, "full"),
        pushDevices: async () => rel((await import("./pushDevice.js")).pushDevice, "full"),
        wallets: async () => rel((await import("./wallet.js")).wallet, "common"),
        notifications: async () => rel((await import("./notification.js")).notification, "full"),
        translations: () => rel(userTranslation, "full"),
        you: () => rel(userYou, "full"),
    },
};
