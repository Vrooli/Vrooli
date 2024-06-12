import { ProjectVersionDirectory, ProjectVersionDirectoryTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const projectVersionDirectoryTranslation: GqlPartial<ProjectVersionDirectoryTranslation> = {
    __typename: "ProjectVersionDirectoryTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const projectVersionDirectory: GqlPartial<ProjectVersionDirectory> = {
    __typename: "ProjectVersionDirectory",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        childOrder: true,
        isRoot: true,
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "nav", { omit: "directories" }),
    },
    full: {
        children: () => rel(projectVersionDirectory, "nav", { omit: ["parentDirectory", "children"] }),
        childApiVersions: async () => rel((await import("./apiVersion")).apiVersion, "list"),
        childCodeVersions: async () => rel((await import("./codeVersion")).codeVersion, "list"),
        childNoteVersions: async () => rel((await import("./noteVersion")).noteVersion, "list"),
        childProjectVersions: async () => rel((await import("./projectVersion")).projectVersion, "list"),
        childRoutineVersions: async () => rel((await import("./routineVersion")).routineVersion, "list"),
        childStandardVersions: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        childTeams: async () => rel((await import("./team")).team, "list"),
        parentDirectory: () => rel(projectVersionDirectory, "nav", { omit: ["parentDirectory", "children"] }),
        translations: () => rel(projectVersionDirectoryTranslation, "full"),
    },
    list: {
        translations: () => rel(projectVersionDirectoryTranslation, "list"),
    },
};
