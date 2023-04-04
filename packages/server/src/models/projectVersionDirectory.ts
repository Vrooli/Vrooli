import { Prisma } from "@prisma/client";
import { MaxObjects, ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from '@shared/consts';
import { projectVersionDirectoryValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, translationShapeHelper } from "../utils";
import { ProjectVersionModel } from "./projectVersion";
import { ModelLogic } from "./types";

const __typename = 'ProjectVersionDirectory' as const;
const suppFields = [] as const;
export const ProjectVersionDirectoryModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ProjectVersionDirectoryCreateInput,
    GqlUpdate: ProjectVersionDirectoryUpdateInput,
    GqlModel: ProjectVersionDirectory,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.project_version_directoryUpsertArgs['create'],
    PrismaUpdate: Prisma.project_version_directoryUpsertArgs['update'],
    PrismaModel: Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>,
    PrismaSelect: Prisma.project_version_directorySelect,
    PrismaWhere: Prisma.project_version_directoryWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            parentDirectory: 'ProjectVersionDirectory',
            projectVersion: 'ProjectVersion',
            children: 'ProjectVersionDirectory',
            childApiVersions: 'ApiVersion',
            childNoteVersions: 'NoteVersion',
            childOrganizations: 'Organization',
            childProjectVersions: 'ProjectVersion',
            childRoutineVersions: 'RoutineVersion',
            childSmartContractVersions: 'SmartContractVersion',
            childStandardVersions: 'StandardVersion',
            runProjectSteps: 'RunProjectStep',
        },
        prismaRelMap: {
            __typename,
            parentDirectory: 'ProjectVersionDirectory',
            projectVersion: 'ProjectVersion',
            children: 'ProjectVersionDirectory',
            childApiVersions: 'ApiVersion',
            childNoteVersions: 'NoteVersion',
            childOrganizations: 'Organization',
            childProjectVersions: 'ProjectVersion',
            childRoutineVersions: 'RoutineVersion',
            childSmartContractVersions: 'SmartContractVersion',
            childStandardVersions: 'StandardVersion',
            runProjectSteps: 'RunProjectStep',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                childOrder: noNull(data.childOrder),
                isRoot: noNull(data.isRoot),
                ...(await shapeHelper({ relation: 'parentDirectory', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'children', data, ...rest })),
                ...(await shapeHelper({ relation: 'projectVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'ProjectVersion', parentRelationshipName: 'directories', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),

            }),
            update: async ({ data, ...rest }) => ({
                childOrder: noNull(data.childOrder),
                isRoot: noNull(data.isRoot),
                ...(await shapeHelper({ relation: 'parentDirectory', relTypes: ['Connect', 'Disconnect'], isOneToOne: true, isRequired: false, objectType: 'ProjectVersionDirectory', parentRelationshipName: 'children', data, ...rest })),
                ...(await shapeHelper({ relation: 'projectVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'ProjectVersion', parentRelationshipName: 'directories', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            })
        },
        yup: projectVersionDirectoryValidation,
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ProjectVersionModel.validate!.isPublic(data.projectVersion as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ProjectVersionModel.validate!.owner(data.projectVersion as any, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            projectVersion: 'ProjectVersion',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                projectVersion: ProjectVersionModel.validate!.visibility.owner(userId),
            })
        }
    },
})