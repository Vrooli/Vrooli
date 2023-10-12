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
    delegate: (prisma) => prisma.project_version_directory,
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
                ...(await shapeHelper({ relation: "childApiVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childNoteVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "NoteVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childOrganizations", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "Organization", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childProjectVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childRoutineVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "RoutineVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childSmartContractVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "SmartContractVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childStandardVersions", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "parentDirectory", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "children", data, ...rest })),
                ...(await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "ProjectVersion", parentRelationshipName: "directories", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),

            }),
            update: async ({ data, ...rest }) => ({
                childOrder: noNull(data.childOrder),
                isRoot: noNull(data.isRoot),
                ...(await shapeHelper({ relation: "childApiVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ApiVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childNoteVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "NoteVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childOrganizations", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Organization", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childProjectVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childRoutineVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "RoutineVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childSmartContractVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "SmartContractVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "childStandardVersions", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "StandardVersion", parentRelationshipName: "directoryListings", data, ...rest })),
                ...(await shapeHelper({ relation: "parentDirectory", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "children", data, ...rest })),
                ...(await shapeHelper({ relation: "projectVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ProjectVersion", parentRelationshipName: "directories", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
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
