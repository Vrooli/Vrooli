import { ResourceVersion, ResourceVersionTranslation, ResourceVersionYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const resourceVersionTranslation: ApiPartial<ResourceVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        details: true,
        instructions: true,
        name: true,
    },
};

export const routineVersionYou: ApiPartial<ResourceVersionYou> = {
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canBookmark: true,
        canReport: true,
        canRun: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
    },
};

export const resourceVersion: ApiPartial<ResourceVersion> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
        codeLanguage: true,
        completedAt: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        resourceSubType: true,
        timesStarted: true,
        timesCompleted: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => rel(routineVersionYou, "common"),
    },
    full: {
        config: true,
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./resource.js")).resource, "full", { omit: "versions" }),
        relatedVersions: async () => rel((await import("./resourceVersionRelation.js")).resourceVersionRelation, "full"),
        translations: () => rel(resourceVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./resource.js")).resource, "list", { omit: "versions" }),
        translations: () => rel(resourceVersionTranslation, "list"),
    },
    nav: {
        id: true,
        complexity: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        root: async () => rel((await import("./resource.js")).resource, "nav", { omit: "versions" }),
        resourceSubType: true,
        translations: () => rel(resourceVersionTranslation, "list"),
        versionIndex: true,
        versionLabel: true,
    },
};
