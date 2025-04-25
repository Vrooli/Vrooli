import { Routine, RoutineYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const routineYou: ApiPartial<RoutineYou> = {
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

export const routine: ApiPartial<Routine> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isInternal: true,
        isPrivate: true,
        issuesCount: true,
        labels: async () => rel((await import("./label.js")).label, "list"),
        owner: {
            __union: {
                Team: async () => rel((await import("./team.js")).team, "nav"),
                User: async () => rel((await import("./user.js")).user, "nav"),
            },
        },
        permissions: true,
        score: true,
        bookmarks: true,
        tags: async () => rel((await import("./tag.js")).tag, "list"),
        transfersCount: true,
        views: true,
        you: () => rel(routineYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./routineVersion.js")).routineVersion, "nav"),
        versions: async () => rel((await import("./routineVersion.js")).routineVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./routineVersion.js")).routineVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isInternal: true,
        isPrivate: true,
    },
};
