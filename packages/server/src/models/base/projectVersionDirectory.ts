import { MaxObjects, projectVersionDirectoryValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { ProjectVersionDirectoryFormat } from "../formats";
import { ProjectVersionDirectoryModelInfo, ProjectVersionDirectoryModelLogic, ProjectVersionModelInfo, ProjectVersionModelLogic } from "./types";

const __typename = "ProjectVersionDirectory" as const;
export const ProjectVersionDirectoryModel: ProjectVersionDirectoryModelLogic = ({
    __typename,
    dbTable: "project_version_directory",
    dbTranslationTable: "project_version_directory_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    }),
    format: ProjectVersionDirectoryFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                childOrder: noNull(data.childOrder),
                isRoot: noNull(data.isRoot),
                childApiVersions: await shapeHelper({ relation: "childApiVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childNoteVersions: await shapeHelper({ relation: "childNoteVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "NoteVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childOrganizations: await shapeHelper({ relation: "childOrganizations", relTypes: ["Connect"], isOneToOne: false, objectType: "Organization", parentRelationshipName: "directoryListings", data, ...rest }),
                childProjectVersions: await shapeHelper({ relation: "childProjectVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childRoutineVersions: await shapeHelper({ relation: "childRoutineVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "RoutineVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childSmartContractVersions: await shapeHelper({ relation: "childSmartContractVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "SmartContractVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childStandardVersions: await shapeHelper({ relation: "childStandardVersions", relTypes: ["Connect"], isOneToOne: false, objectType: "StandardVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                parentDirectory: await shapeHelper({ relation: "parentDirectory", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersionDirectory", parentRelationshipName: "children", data, ...rest }),
                projectVersion: await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "directories", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),

            }),
            update: async ({ data, ...rest }) => ({
                childOrder: noNull(data.childOrder),
                isRoot: noNull(data.isRoot),
                childApiVersions: await shapeHelper({ relation: "childApiVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ApiVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childNoteVersions: await shapeHelper({ relation: "childNoteVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "NoteVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childOrganizations: await shapeHelper({ relation: "childOrganizations", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Organization", parentRelationshipName: "directoryListings", data, ...rest }),
                childProjectVersions: await shapeHelper({ relation: "childProjectVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "ProjectVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childRoutineVersions: await shapeHelper({ relation: "childRoutineVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "RoutineVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childSmartContractVersions: await shapeHelper({ relation: "childSmartContractVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "SmartContractVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                childStandardVersions: await shapeHelper({ relation: "childStandardVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "StandardVersion", parentRelationshipName: "directoryListings", data, ...rest }),
                parentDirectory: await shapeHelper({ relation: "parentDirectory", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "ProjectVersionDirectory", parentRelationshipName: "children", data, ...rest }),
                projectVersion: await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersion", parentRelationshipName: "directories", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: projectVersionDirectoryValidation,
    },
    search: {} as any,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ProjectVersionDirectoryModelInfo["PrismaSelect"]>([["projectVersion", "ProjectVersion"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").validate().owner(data?.projectVersion as ProjectVersionModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            projectVersion: "ProjectVersion",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                projectVersion: ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").validate().visibility.owner(userId),
            }),
        },
    }),
});
