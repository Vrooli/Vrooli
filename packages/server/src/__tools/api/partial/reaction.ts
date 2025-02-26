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
                Post: async () => rel((await import("./post.js")).post, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Question: async () => rel((await import("./question.js")).question, "list"),
                QuestionAnswer: async () => rel((await import("./questionAnswer.js")).questionAnswer, "list"),
                Quiz: async () => rel((await import("./quiz.js")).quiz, "list"),
                Routine: async () => rel((await import("./routine.js")).routine, "list"),
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
