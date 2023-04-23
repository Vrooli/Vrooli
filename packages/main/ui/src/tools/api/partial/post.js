import { rel } from "../utils";
export const postTranslation = {
    __typename: "PostTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};
export const post = {
    __typename: "Post",
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
        resourceList: async () => rel((await import("./resourceList")).resourceList, "full"),
        translations: () => rel(postTranslation, "full"),
    },
    list: {
        resourceList: async () => rel((await import("./resourceList")).resourceList, "list"),
        translations: () => rel(postTranslation, "list"),
    },
    nav: {
        id: true,
        translations: () => rel(postTranslation, "list"),
    },
};
//# sourceMappingURL=post.js.map