import { Bookmark } from ":local/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const bookmark: GqlPartial<Bookmark> = {
    __typename: "Bookmark",
    common: {
        __define: {
            0: async () => rel((await import("./api")).api, "list"),
            1: async () => rel((await import("./comment")).comment, "list"),
            2: async () => rel((await import("./issue")).issue, "list"),
            3: async () => rel((await import("./note")).note, "list"),
            4: async () => rel((await import("./organization")).organization, "list"),
            5: async () => rel((await import("./post")).post, "list"),
            6: async () => rel((await import("./project")).project, "list"),
            7: async () => rel((await import("./question")).question, "list"),
            8: async () => rel((await import("./questionAnswer")).questionAnswer, "list"),
            9: async () => rel((await import("./quiz")).quiz, "list"),
            10: async () => rel((await import("./routine")).routine, "list"),
            11: async () => rel((await import("./smartContract")).smartContract, "list"),
            12: async () => rel((await import("./standard")).standard, "list"),
            13: async () => rel((await import("./tag")).tag, "list"),
            14: async () => rel((await import("./user")).user, "list"),
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Comment: 1,
                Issue: 2,
                Note: 3,
                Organization: 4,
                Post: 5,
                Project: 6,
                Question: 7,
                QuestionAnswer: 8,
                Quiz: 9,
                Routine: 10,
                SmartContract: 11,
                Standard: 12,
                Tag: 13,
                User: 14,
            },
        },
    },
    full: {
        __define: {
            0: async () => rel((await import("./bookmarkList")).bookmarkList, "common", { omit: "bookmarks" }),
        },
        list: { __use: 0 },
    },
};
