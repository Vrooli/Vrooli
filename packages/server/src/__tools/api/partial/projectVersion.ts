import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";
import { versionYou } from "./root.js";

export const projectVersionTranslation: ApiPartial<ProjectVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const projectVersionYou: ApiPartial<ProjectVersionYou> = {
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
        canRead: true,
    },
};

export const projectVersion: ApiPartial<ProjectVersion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        directoriesCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        runProjectsCount: true,
        simplicity: true,
        versionIndex: true,
        versionLabel: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        directories: async () => rel((await import("./projectVersionDirectory.js")).projectVersionDirectory, "full", { omit: "projectVersion " }),
        pullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./project.js")).project, "full", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "full"),
        versionNotes: true,
    },
    list: {
        root: async () => rel((await import("./project.js")).project, "list", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "list"),
    },
    nav: {
        id: true,
        complexity: true, // Used by RunProject
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./project.js")).project, "nav", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "list"),
    },
};
