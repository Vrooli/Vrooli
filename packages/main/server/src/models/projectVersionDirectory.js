import { MaxObjects } from "@local/consts";
import { projectVersionDirectoryValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
const __typename = "ProjectVersionDirectory";
const suppFields = [];
export const ProjectVersionDirectoryModel = ({
    __typename,
    delegate: (prisma) => prisma.project_version_directory,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ProjectVersionModel.validate.isPublic(data.projectVersion, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ProjectVersionModel.validate.owner(data.projectVersion, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            projectVersion: "ProjectVersion",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                projectVersion: ProjectVersionModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=projectVersionDirectory.js.map