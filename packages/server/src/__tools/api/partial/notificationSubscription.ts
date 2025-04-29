import { NotificationSubscription } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const notificationSubscription: ApiPartial<NotificationSubscription> = {
    full: {
        id: true,
        created_at: true,
        silent: true,
        object: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Comment: async () => rel((await import("./comment.js")).comment, "list"),
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Meeting: async () => rel((await import("./meeting.js")).meeting, "list"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                PullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "list"),
                Report: async () => rel((await import("./report.js")).report, "list"),
                Routine: async () => rel((await import("./resource.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
                Team: async () => rel((await import("./team.js")).team, "list"),
            },
        },
    },
};
