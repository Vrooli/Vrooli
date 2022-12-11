import { projectsCreate, projectsUpdate } from "@shared/validation";
import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectVersionSortBy, SessionUser, RootPermission, ProjectVersionSearchInput, ProjectVersion } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Validator, Mutater, Displayer } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { OrganizationModel } from "./organization";
import { padSelect, permissionsSelectHelper, searchStringBuilder } from "../builders";
import { bestLabel, oneIsPublic } from "../utils";
import { SelectWrap } from "../builders/types";

const __typename = 'ProjectVersion' as const;

const suppFields = ['runs'] as const;
const formatter = (): Formatter<ProjectVersion, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        comments: 'Comment',
        directories: 'ProjectVersionDirectory',
        directoryListings: 'ProjectVersionDirectory',
        forks: 'Project',
        pullRequest: 'PullRequest',
        reports: 'Report',
        root: 'Project',
    },
    countFields: ['commentsCount', 'directoriesCount', 'directoryListingsCount', 'forksCount', 'reportsCount', 'runsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['runs', async () => {
                //TODO
                return {} as any;
            }],
        ],
    },
})

const searcher = (): Searcher<
    ProjectVersionSearchInput,
    ProjectVersionSortBy,
    Prisma.project_versionWhereInput
> => ({
    defaultSort: ProjectVersionSortBy.DateCompletedDesc,
    sortBy: ProjectVersionSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'directoryListingsId',
        'minScoreRoot',
        'minStarsRoot',
        'minViewsRoot',
        'ownedByOrganizationId',
        'ownedByUserId',
        'rootId',
        'tags',
        'translationLanguages',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: (params) => ({
        OR: [
            ...searchStringBuilder(['translationsDescription', 'translationsName'], params),
            { root: searchStringBuilder(['tags'], params)[0] },
        ]
    }),
})

const validator = (): Validator<
    ProjectCreateInput,
    ProjectUpdateInput,
    Prisma.projectGetPayload<SelectWrap<Prisma.projectSelect>>,
    RootPermission,
    Prisma.projectSelect,
    Prisma.projectWhereInput,
    true,
    true
> => ({
    validateMap: {
        __typename: 'Project',
        parent: 'Project',
        createdBy: 'User',
        ownedByOrganization: 'Organization',
        ownedByUser: 'User',
        versions: {
            select: {
                forks: 'Project',
            }
        },
    },
    isTransferable: true,
    hasOriginalOwner: ({ createdBy, ownedByUser }) => ownedByUser !== null && ownedByUser.id === createdBy?.id,
    maxObjects: {
        User: {
            private: {
                noPremium: 3,
                premium: 100,
            },
            public: {
                noPremium: 25,
                premium: 250,
            },
        },
        Organization: {
            private: {
                noPremium: 3,
                premium: 100,
            },
            public: {
                noPremium: 25,
                premium: 250,
            },
        },
    },
    permissionsSelect: (...params) => ({
        id: true,
        hasCompleteVersion: true,
        isDeleted: true,
        isPrivate: true,
        permissions: true,
        createdBy: padSelect({ id: true }),
        ...permissionsSelectHelper([
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], ...params),
        versions: {
            select: {
                isComplete: true,
                isDeleted: true,
                isPrivate: true,
            }
        },
    }),
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ([
        // ['canComment', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canDelete', async () => isAdmin && !isDeleted],
        // ['canEdit', async () => isAdmin && !isDeleted],
        // ['canReport', async () => !isAdmin && !isDeleted && isPublic],
        // // ['canRun', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canStar', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canView', async () => !isDeleted && (isAdmin || isPublic)],
        // ['canVote', async () => !isDeleted && (isAdmin || isPublic)],
    ]),
    owner: (data) => ({
        Organization: data.ownedByOrganization,
        User: data.ownedByUser,
    }),
    hasCompletedVersion: (data) => data.hasCompleteVersion === true,
    isDeleted: (data) => data.isDeleted,// || data.root.isDeleted,
    isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.projectSelect>(data, [
        ['ownedByOrganization', 'Organization'],
        ['ownedByUser', 'User'],
    ], languages),
    visibility: {
        private: {
            isPrivate: true,
            // OR: [
            //     { isPrivate: true },
            //     { root: { isPrivate: true } },
            // ]
        },
        public: {
            isPrivate: false,
            // AND: [
            //     { isPrivate: false },
            //     { root: { isPrivate: false } },
            // ]
        },
        owner: (userId) => ({
            OR: [
                { ownedByUser: { id: userId } },
                { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            ]
            // root: {
            //     OR: [
            //         { ownedByUser: { id: userId } },
            //         { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
            //     ]
            // }
        }),
    }
    // createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksDescription'));
    // for (const input of updateMany) {
})

const shapeBase = async (prisma: PrismaType, userData: SessionUser, data: ProjectCreateInput | ProjectUpdateInput, isAdd: boolean) => {
    return {
        id: data.id,
        // isComplete: data.isComplete ?? undefined,
        // isPrivate: data.isPrivate ?? undefined,
        // completedAt: (data.isComplete === true) ? new Date().toISOString() : (data.isComplete === false) ? null : undefined,
        permissions: JSON.stringify({}),
        // resourceList: await relBuilderHelper({ data, isAdd, isOneToOne: true, isRequired: false, relationshipName: 'resourceList', objectType: 'ResourceList', prisma, userData }),
        // tags: await tagRelationshipBuilder(prisma, userData, data, 'Project', isAdd),
        // translations: await translationRelationshipBuilder(prisma, userData, data, isAdd),
    }
}


const mutater = (): Mutater<
    Project,
    { graphql: ProjectCreateInput, db: Prisma.projectUpsertArgs['create'] },
    { graphql: ProjectUpdateInput, db: Prisma.projectUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, prisma, userData }) => {
            return {
                ...(await shapeBase(prisma, userData, data, true)),
                // parentId: data.parentId ?? undefined,
                // organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                // createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                // createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                // user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
            }
        },
        update: async ({ data, prisma, userData }) => {
            return {
                ...(await shapeBase(prisma, userData, data, false)),
                // organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                // user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
            }
        }
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).createProject(userData.id, c.id);
            }
        },
        onUpdated: ({ updated, prisma, userData }) => {
            // for (const u of updated) {
            //     Trigger(prisma, userData.languages).updateProject(userData.id, u.id as string);
            // }
        }
    },
    yup: { create: projectsCreate, update: projectsUpdate },
});

const displayer = (): Displayer<
    Prisma.project_versionSelect,
    Prisma.project_versionGetPayload<SelectWrap<Prisma.project_versionSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const ProjectVersionModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    validate: validator(),
})