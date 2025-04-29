import { Reaction, ReactionSummary } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reaction: ApiPartial<Reaction> = {
    list: {
        id: true,
        to: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                ChatMessage: async () => rel((await import("./chatMessage.js")).chatMessage, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Comment: async () => rel((await import("./comment.js")).comment, "list"),
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Routine: async () => rel((await import("./resource.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
            },
        },
    },
};

export const reactionSummary: ApiPartial<ReactionSummary> = {
    list: {
        emoji: true,
        count: true,
    },
};
