import { Routine, RoutineYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const routineYou: GqlPartial<RoutineYou> = {
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

export const routine: GqlPartial<Routine> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isInternal: true,
        isPrivate: true,
        issuesCount: true,
        labels: async () => rel((await import("./label")).label, "list"),
        owner: {
            __union: {
                Team: async () => rel((await import("./team")).team, "nav"),
                User: async () => rel((await import("./user")).user, "nav"),
            },
        },
        permissions: true,
        questionsCount: true,
        score: true,
        bookmarks: true,
        tags: async () => rel((await import("./tag")).tag, "list"),
        transfersCount: true,
        views: true,
        you: () => rel(routineYou, "full"),
    },
    full: {
        versionsCount: true,
        parent: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
        versions: async () => rel((await import("./routineVersion")).routineVersion, "full", { omit: "root" }),
    },
    list: {
        versions: async () => rel((await import("./routineVersion")).routineVersion, "list", { omit: "root" }),
    },
    nav: {
        id: true,
        isInternal: true,
        isPrivate: true,
    },
};
