import { ProjectVersionDirectory, ProjectVersionDirectoryTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const projectVersionDirectoryTranslation: ApiPartial<ProjectVersionDirectoryTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const projectVersionDirectory: ApiPartial<ProjectVersionDirectory> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        childOrder: true,
        isRoot: true,
        projectVersion: async () => rel((await import("./projectVersion.js")).projectVersion, "nav", { omit: "directories" }),
    },
    full: {
        children: () => rel(projectVersionDirectory, "nav", { omit: ["parentDirectory", "children"] }),
        childApiVersions: async () => rel((await import("./apiVersion.js")).apiVersion, "list"),
        childCodeVersions: async () => rel((await import("./codeVersion.js")).codeVersion, "list"),
        childNoteVersions: async () => rel((await import("./noteVersion.js")).noteVersion, "list"),
        childProjectVersions: async () => rel((await import("./projectVersion.js")).projectVersion, "list"),
        childRoutineVersions: async () => rel((await import("./routineVersion.js")).routineVersion, "list"),
        childStandardVersions: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
        childTeams: async () => rel((await import("./team.js")).team, "list"),
        parentDirectory: () => rel(projectVersionDirectory, "nav", { omit: ["parentDirectory", "children"] }),
        translations: () => rel(projectVersionDirectoryTranslation, "full"),
    },
    list: {
        translations: () => rel(projectVersionDirectoryTranslation, "list"),
    },
};
