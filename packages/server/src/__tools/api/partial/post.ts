import { Post, PostTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const postTranslation: ApiPartial<PostTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const post: ApiPartial<Post> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        commentsCount: true,
        repostsCount: true,
        score: true,
        bookmarks: true,
        views: true,
    },
    full: {
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common"),
        translations: () => rel(postTranslation, "full"),
    },
    list: {
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common"),
        translations: () => rel(postTranslation, "list"),
    },
    nav: {
        id: true,
        translations: () => rel(postTranslation, "list"),
    },
};
