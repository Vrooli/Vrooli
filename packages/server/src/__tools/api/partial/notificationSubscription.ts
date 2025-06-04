import { type NotificationSubscription } from "@local/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const notificationSubscription: ApiPartial<NotificationSubscription> = {
    full: {
        id: true,
        createdAt: true,
        silent: true,
        object: {
            __union: {
                Comment: async () => rel((await import("./comment.js")).comment, "list"),
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Meeting: async () => rel((await import("./meeting.js")).meeting, "list"),
                PullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "list"),
                Report: async () => rel((await import("./report.js")).report, "list"),
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
                Team: async () => rel((await import("./team.js")).team, "list"),
            },
        },
    },
};
