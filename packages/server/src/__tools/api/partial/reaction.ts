import { Reaction, ReactionSummary } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reaction: GqlPartial<Reaction> = {
    list: {
        id: true,
        to: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                ChatMessage: async () => rel((await import("./chatMessage")).chatMessage, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Comment: async () => rel((await import("./comment")).comment, "list"),
                Issue: async () => rel((await import("./issue")).issue, "list"),
                Note: async () => rel((await import("./note")).note, "list"),
                Post: async () => rel((await import("./post")).post, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                Question: async () => rel((await import("./question")).question, "list"),
                QuestionAnswer: async () => rel((await import("./questionAnswer")).questionAnswer, "list"),
                Quiz: async () => rel((await import("./quiz")).quiz, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
            },
        },
    },
};

export const reactionSummary: GqlPartial<ReactionSummary> = {
    list: {
        emoji: true,
        count: true,
    },
};
