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
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        commentsCount: true,
        status: true,
        from: {
            __union: {
                ApiVersion: async () => rel((await import("./apiVersion.js")).apiVersion, "list"),
                CodeVersion: async () => rel((await import("./codeVersion.js")).codeVersion, "list"),
                NoteVersion: async () => rel((await import("./noteVersion.js")).noteVersion, "list"),
                ProjectVersion: async () => rel((await import("./projectVersion.js")).projectVersion, "list"),
                RoutineVersion: async () => rel((await import("./resourceVersion.js")).routineVersion, "list"),
                StandardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
            },
        },
        to: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "list"),
                Code: async () => rel((await import("./code.js")).code, "list"),
                Note: async () => rel((await import("./note.js")).note, "list"),
                Project: async () => rel((await import("./project.js")).project, "list"),
                Routine: async () => rel((await import("./resource.js")).routine, "list"),
                Standard: async () => rel((await import("./standard.js")).standard, "list"),
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
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
    },
};
