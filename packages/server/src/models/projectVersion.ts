import { projectVersionValidation } from "@shared/validation";
import { ProjectCreateInput, ProjectUpdateInput, ProjectVersionSortBy, SessionUser, ProjectVersionSearchInput, ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput, VersionYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { addSupplementalFields, modelToGraphQL, padSelect, permissionsSelectHelper, selectHelper, toPartialGraphQLInfo } from "../builders";
import { bestLabel, oneIsPublic } from "../utils";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { RunProjectModel } from "./runProject";
import { getSingleTypePermissions } from "../validators";

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

const type = 'ProjectVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canEdit' | 'canReport' | 'canUse' | 'canView'>;
const suppFields = ['you.canCopy', 'you.canDelete', 'you.canEdit', 'you.canReport', 'you.canUse', 'you.canView', 'you.runs'] as const;
export const ProjectVersionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ProjectVersionCreateInput,
    GqlUpdate: ProjectVersionUpdateInput,
    GqlModel: ProjectVersion,
    GqlSearch: ProjectVersionSearchInput,
    GqlSort: ProjectVersionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.project_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.project_versionUpsertArgs['update'],
    PrismaModel: Prisma.project_versionGetPayload<SelectWrap<Prisma.project_versionSelect>>,
    PrismaSelect: Prisma.project_versionSelect,
    PrismaWhere: Prisma.project_versionWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.project_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            type,
            comments: 'Comment',
            directories: 'ProjectVersionDirectory',
            directoryListings: 'ProjectVersionDirectory',
            forks: 'Project',
            pullRequest: 'PullRequest',
            reports: 'Report',
            root: 'Project',
            // 'runs.project': 'RunProject', //TODO
        },
        prismaRelMap: {
            type,
            comments: 'Comment',
            directories: 'ProjectVersionDirectory',
            directoryListings: 'ProjectVersionDirectory',
            pullRequest: 'PullRequest',
            reports: 'Report',
            resourceList: 'ResourceList',
            root: 'Project',
            forks: 'Project',
            runProjects: 'RunProject',
            suggestedNextByProject: 'ProjectVersion',
        },
        joinMap: {
            suggestedNextByProject: 'toProjectVersion',
        },
        countFields: {
            commentsCount: true,
            directoriesCount: true,
            directoryListingsCount: true,
            forksCount: true,
            reportsCount: true,
            runsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                const runs = async () => {
                    if (!userData) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add projectVersionId, so we 
                    // can associate runs with their project
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, RunProjectModel.format.gqlRelMap, userData.languages, true),
                        projectVersionId: true
                    }
                    // Query runs made by user
                    let runs: any[] = await prisma.run_project.findMany({
                        where: {
                            AND: [
                                { projectVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } }
                            ]
                        },
                        ...selectHelper(runPartial)
                    });
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGraphQL(r, runPartial));
                    // Add supplemental fields
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    // Split runs by id
                    const projectRuns = ids.map((id) => runs.filter(r => r.projectVersionId === id));
                    return projectRuns;
                };
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.runs': await runs(),
                }
            },
        }
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                // parentId: data.parentId ?? undefined,
                // organization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                // createdByOrganization: data.createdByOrganizationId ? { connect: { id: data.createdByOrganizationId } } : undefined,
                // createdByUser: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
                // user: data.createdByUserId ? { connect: { id: data.createdByUserId } } : undefined,
            } as any),
            update: async ({ data, prisma, userData }) => ({
                // organization: data.organizationId ? { connect: { id: data.organizationId } } : data.userId ? { disconnect: true } : undefined,
                // user: data.userId ? { connect: { id: data.userId } } : data.organizationId ? { disconnect: true } : undefined,
            } as any)
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
        yup: projectVersionValidation,
    },
    search: {
        defaultSort: ProjectVersionSortBy.DateCompletedDesc,
        sortBy: ProjectVersionSortBy,
        searchFields: {
            createdById: true,
            createdTimeFrame: true,
            directoryListingsId: true,
            isCompleteWithRoot: true,
            isCompleteWithRootExcludeOwnedByOrganizationId: true,
            isCompleteWithRootExcludeOwnedByUserId: true,
            maxComplexity: true,
            maxSimplicity: true,
            maxTimesCompleted: true,
            minComplexity: true,
            minScoreRoot: true,
            minSimplicity: true,
            minStarsRoot: true,
            minTimesCompleted: true,
            minViewsRoot: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            rootId: true,
            tags: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'transNameWrapped',
                { root: 'tagsWrapped' },
                { root: 'labelsWrapped' },
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 1000000,
        permissionsSelect: (...params) => ({
            id: true,
            hasCompleteVersion: true,
            isDeleted: true,
            isPrivate: true,
            permissions: true,
            createdBy: padSelect({ id: true }),
            ...permissionsSelectHelper({
                ownedByOrganization: 'Organization',
                ownedByUser: 'User',
            }, ...params),
            versions: {
                select: {
                    isComplete: true,
                    isDeleted: true,
                    isPrivate: true,
                }
            },
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            canComment: () => !isDeleted && (isAdmin || isPublic),
            canCopy: () => !isDeleted && (isAdmin || isPublic),
            canDelete: () => isAdmin && !isDeleted,
            canEdit: () => isAdmin && !isDeleted,
            canReport: () => !isAdmin && !isDeleted && isPublic,
            canRun: () => !isDeleted && (isAdmin || isPublic),
            canStar: () => !isDeleted && (isAdmin || isPublic),
            canUse: () => !isDeleted && (isAdmin || isPublic),
            canView: () => !isDeleted && (isAdmin || isPublic),
            canVote: () => !isDeleted && (isAdmin || isPublic),
        }),
        owner: (data) => ({
            // Organization: data.ownedByOrganization,
            // User: data.ownedByUser,
        } as any),
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
                root: ProjectVersionModel.validate!.visibility.owner(userId),
            }),
        }
        // createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksDescription'));
        // for (const input of updateMany) {
    },
})