import { NotificationSubscription } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const notificationSubscription: ApiPartial<NotificationSubscription> = {
    full: {
        id: true,
        created_at: true,
        silent: true,
        object: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Comment: async () => rel((await import("./comment")).comment, "list"),
                Issue: async () => rel((await import("./issue")).issue, "list"),
                Meeting: async () => rel((await import("./meeting")).meeting, "list"),
                Note: async () => rel((await import("./note")).note, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                PullRequest: async () => rel((await import("./pullRequest")).pullRequest, "list"),
                Question: async () => rel((await import("./question")).question, "list"),
                Quiz: async () => rel((await import("./quiz")).quiz, "list"),
                Report: async () => rel((await import("./report")).report, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
                Team: async () => rel((await import("./team")).team, "list"),
            },
        },
    },
};
