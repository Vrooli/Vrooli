import { Reaction, ReactionSummary } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const reaction: ApiPartial<Reaction> = {
    list: {
        id: true,
        to: {
            __union: {
                ChatMessage: async () => rel((await import("./chatMessage.js")).chatMessage, "list"),
                Comment: async () => rel((await import("./comment.js")).comment, "list"),
                Issue: async () => rel((await import("./issue.js")).issue, "list"),
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
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
