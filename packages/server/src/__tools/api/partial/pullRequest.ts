import { PullRequest, PullRequestTranslation, PullRequestYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const pullRequestYou: ApiPartial<PullRequestYou> = {
    common: {
        canComment: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
    },
};

export const pullRequestTranslation: ApiPartial<PullRequestTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

export const pullRequest: ApiPartial<PullRequest> = {
    common: {
        id: true,
        publicId: true,
        createdAt: true,
        updatedAt: true,
        mergedOrRejectedAt: true,
        commentsCount: true,
        status: true,
        from: {
            __union: {
                ResourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "list"),
            },
        },
        to: {
            __union: {
                Resource: async () => rel((await import("./resource.js")).resource, "list"),
            },
        },
        createdBy: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(pullRequestYou, "full"),
    },
    full: {
        translations: () => rel(pullRequestTranslation, "full"),
    },
    list: {
        translations: () => rel(pullRequestTranslation, "list"),
    },
    nav: {
        id: true,
        createdAt: true,
        updatedAt: true,
        mergedOrRejectedAt: true,
        status: true,
    },
};
