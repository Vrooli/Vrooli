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
        childApiVersions: async () => rel((await import("./apiVersion")).apiVersion, "common"),
        childNoteVersions: async () => rel((await import("./noteVersion")).noteVersion, "common"),
        childOrganizations: async () => rel((await import("./organization")).organization, "common"),
        childProjectVersions: async () => rel((await import("./projectVersion")).projectVersion, "common"),
        childRoutineVersions: async () => rel((await import("./routineVersion")).routineVersion, "common"),
        childSmartContractVersions: async () => rel((await import("./smartContractVersion")).smartContractVersion, "common"),
        childStandardVersions: async () => rel((await import("./standardVersion")).standardVersion, "common"),
        parentDirectory: () => rel(projectVersionDirectory, "nav", { omit: ["parentDirectory", "children"] }),
        translations: () => rel(projectVersionDirectoryTranslation, "full"),
    },
    list: {
        translations: () => rel(projectVersionDirectoryTranslation, "list"),
    },
};
