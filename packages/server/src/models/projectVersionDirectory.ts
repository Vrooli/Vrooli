import { MaxObjects, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySortBy, ProjectVersionDirectoryUpdateInput, projectVersionDirectoryValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
import { ModelLogic } from "./types";

const __typename = "ProjectVersionDirectory" as const;
const suppFields = [] as const;
export const ProjectVersionDirectoryModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ProjectVersionDirectoryCreateInput,
    GqlUpdate: ProjectVersionDirectoryUpdateInput,
    GqlModel: ProjectVersionDirectory,
    GqlPermission: object,
    GqlSearch: ProjectVersionDirectorySearchInput,
    GqlSort: ProjectVersionDirectorySortBy,
    PrismaCreate: Prisma.project_version_directoryUpsertArgs["create"],
    PrismaUpdate: Prisma.project_version_directoryUpsertArgs["update"],
    PrismaModel: Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>,
    PrismaSelect: Prisma.project_version_directorySelect,
    PrismaWhere: Prisma.project_version_directoryWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
    },
    format: {
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
    },
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ProjectVersionModel.validate!.isPublic(data.projectVersion as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ProjectVersionModel.validate!.owner(data.projectVersion as any, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            projectVersion: "ProjectVersion",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                projectVersion: ProjectVersionModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
