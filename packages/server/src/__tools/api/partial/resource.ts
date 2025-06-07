import { type Resource, type ResourceYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const resourceYou: ApiPartial<ResourceYou> = {
    common: {
        canComment: true,
        canDelete: true,
        canBookmark: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
        isBookmarked: true,
        isViewed: true,
        reaction: true,
    },
};

export const resource: ApiPartial<Resource> = {
    common: {
        id: true,
        publicId: true,
        createdAt: true,
        updatedAt: true,
        bookmarks: true,
        isInternal: true,
        isPrivate: true,
        issuesCount: true,
        owner: {
            __union: {
                Team: async () => rel((await import("./team.js")).team, "nav"),
                User: async () => rel((await import("./user.js")).user, "nav"),
            },
        },
        permissions: true,
        resourceType: true,
        score: true,
        tags: async () => rel((await import("./tag.js")).tag, "list"),
        transfersCount: true,
        views: true,
        you: () => rel(resourceYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./resourceVersion.js")).resourceVersion, "nav"),
        versions: async () => rel((await import("./resourceVersion.js")).resourceVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./resourceVersion.js")).resourceVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isInternal: true,
        isPrivate: true,
    },
};
