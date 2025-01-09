import { Bookmark } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const bookmark: ApiPartial<Bookmark> = {
    common: {
        id: true,
        list: async () => rel((await import("./bookmarkList")).bookmarkList, "common"),
        to: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Comment: async () => rel((await import("./comment")).comment, "common"),
                Issue: async () => rel((await import("./issue")).issue, "nav"),
                Note: async () => rel((await import("./note")).note, "list"),
                Post: async () => rel((await import("./post")).post, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                Question: async () => rel((await import("./question")).question, "list"),
                QuestionAnswer: async () => rel((await import("./questionAnswer")).questionAnswer, "list"),
                Quiz: async () => rel((await import("./quiz")).quiz, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
                Tag: async () => rel((await import("./tag")).tag, "list"),
                Team: async () => rel((await import("./team")).team, "list"),
                User: async () => rel((await import("./user")).user, "list"),
            },
        },
    },
    full: {
        list: async () => rel((await import("./bookmarkList")).bookmarkList, "common", { omit: "bookmarks" }),
    },
};
