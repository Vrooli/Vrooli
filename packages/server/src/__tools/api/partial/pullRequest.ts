import { PullRequest, PullRequestTranslation, PullRequestYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const pullRequestYou: GqlPartial<PullRequestYou> = {
    common: {
        canComment: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
    },
};

export const pullRequestTranslation: GqlPartial<PullRequestTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

export const pullRequest: GqlPartial<PullRequest> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        commentsCount: true,
        status: true,
        from: {
            __union: {
                ApiVersion: async () => rel((await import("./apiVersion")).apiVersion, "list"),
                CodeVersion: async () => rel((await import("./codeVersion")).codeVersion, "list"),
                NoteVersion: async () => rel((await import("./noteVersion")).noteVersion, "list"),
                ProjectVersion: async () => rel((await import("./projectVersion")).projectVersion, "list"),
                RoutineVersion: async () => rel((await import("./routineVersion")).routineVersion, "list"),
                StandardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
            },
        },
        to: {
            __union: {
                Api: async () => rel((await import("./api")).api, "list"),
                Code: async () => rel((await import("./code")).code, "list"),
                Note: async () => rel((await import("./note")).note, "list"),
                Project: async () => rel((await import("./project")).project, "list"),
                Routine: async () => rel((await import("./routine")).routine, "list"),
                Standard: async () => rel((await import("./standard")).standard, "list"),
            },
        },
        createdBy: async () => rel((await import("./user")).user, "nav"),
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
