import { ProjectVersionDirectoryModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ProjectVersionDirectoryFormat: Formatter<ProjectVersionDirectoryModelLogic> = {
    gqlRelMap: {
        __typename: "ProjectVersionDirectory",
        parentDirectory: "ProjectVersionDirectory",
        projectVersion: "ProjectVersion",
        children: "ProjectVersionDirectory",
        childApiVersions: "ApiVersion",
        childNoteVersions: "NoteVersion",
        childOrganizations: "Organization",
        childProjectVersions: "ProjectVersion",
        childRoutineVersions: "RoutineVersion",
        childSmartContractVersions: "SmartContractVersion",
        childStandardVersions: "StandardVersion",
        runProjectSteps: "RunProjectStep",
    },
    prismaRelMap: {
        __typename: "ProjectVersionDirectory",
        parentDirectory: "ProjectVersionDirectory",
        projectVersion: "ProjectVersion",
        children: "ProjectVersionDirectory",
        childApiVersions: "ApiVersion",
        childNoteVersions: "NoteVersion",
        childOrganizations: "Organization",
        childProjectVersions: "ProjectVersion",
        childRoutineVersions: "RoutineVersion",
        childSmartContractVersions: "SmartContractVersion",
        childStandardVersions: "StandardVersion",
        runProjectSteps: "RunProjectStep",
    },
    countFields: {},
};
