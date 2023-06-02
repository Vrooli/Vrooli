import { ProjectVersionDirectoryModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ProjectVersionDirectory" as const;
export const ProjectVersionDirectoryFormat: Formatter<ProjectVersionDirectoryModelLogic> = {
    gqlRelMap: {
        __typename,
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
        __typename,
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
