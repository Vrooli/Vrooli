import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

export const projectVersionTranslation: GqlPartial<ProjectVersionTranslation> = {
    __typename: "ProjectVersionTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const projectVersionYou: GqlPartial<ProjectVersionYou> = {
    __typename: "ProjectVersionYou",
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

export const projectVersion: GqlPartial<ProjectVersion> = {
    __typename: "ProjectVersion",
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
        directories: async () => rel((await import("./projectVersionDirectory")).projectVersionDirectory, "full", { omit: "projectVersion " }),
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./project")).project, "full", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "full"),
        versionNotes: true,
    },
    list: {
        root: async () => rel((await import("./project")).project, "list", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "list"),
    },
    nav: {
        id: true,
        complexity: true, // Used by RunProject
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./project")).project, "nav", { omit: "versions" }),
        translations: () => rel(projectVersionTranslation, "list"),
    },
};
