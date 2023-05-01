import { Reaction } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const reaction: GqlPartial<Reaction> = {
    __typename: "Reaction",
    list: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./chatMessage")).chatMessage, "list"),
            2: async () => rel((await import("./comment")).comment, "list"),
            3: async () => rel((await import("./issue")).issue, "list"),
            4: async () => rel((await import("./note")).note, "list"),
            5: async () => rel((await import("./post")).post, "list"),
            6: async () => rel((await import("./project")).project, "list"),
            7: async () => rel((await import("./question")).question, "list"),
            8: async () => rel((await import("./questionAnswer")).questionAnswer, "list"),
            9: async () => rel((await import("./quiz")).quiz, "list"),
            10: async () => rel((await import("./routine")).routine, "list"),
            11: async () => rel((await import("./smartContract")).smartContract, "list"),
            12: async () => rel((await import("./standard")).standard, "list"),
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                ChatMessage: 1,
                Comment: 2,
                Issue: 3,
                Note: 4,
                Post: 5,
                Project: 6,
                Question: 7,
                QuestionAnswer: 8,
                Quiz: 9,
                Routine: 10,
                SmartContract: 11,
                Standard: 12,
            },
        },
    },
};
