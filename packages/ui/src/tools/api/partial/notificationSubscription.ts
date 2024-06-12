import { NotificationSubscription } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const notificationSubscription: GqlPartial<NotificationSubscription> = {
    __typename: "NotificationSubscription",
    full: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./comment")).comment, "list"),
            2: async () => rel((await import("./issue")).issue, "list"),
            3: async () => rel((await import("./meeting")).meeting, "list"),
            4: async () => rel((await import("./note")).note, "list"),
            5: async () => rel((await import("./team")).team, "list"),
            6: async () => rel((await import("./project")).project, "list"),
            7: async () => rel((await import("./pullRequest")).pullRequest, "list"),
            8: async () => rel((await import("./question")).question, "list"),
            9: async () => rel((await import("./quiz")).quiz, "list"),
            10: async () => rel((await import("./report")).report, "list"),
            11: async () => rel((await import("./routine")).routine, "list"),
            12: async () => rel((await import("./code")).code, "list"),
            13: async () => rel((await import("./standard")).standard, "list"),
        },
        id: true,
        created_at: true,
        silent: true,
        object: {
            __union: {
                Api: 0,
                Code: 12,
                Comment: 1,
                Issue: 2,
                Meeting: 3,
                Note: 4,
                Project: 6,
                PullRequest: 7,
                Question: 8,
                Quiz: 9,
                Report: 10,
                Routine: 11,
                Standard: 13,
                Team: 5,
            },
        },
    },
};
