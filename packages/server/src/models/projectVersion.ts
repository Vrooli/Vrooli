import { projectVersionValidation } from "@shared/validation";
import { ProjectCreateInput, ProjectUpdateInput, ProjectVersionSortBy, SessionUser, ProjectVersionSearchInput, ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput, VersionYou, PrependString, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { Trigger } from "../events";
import { addSupplementalFields, modelToGraphQL, selPad, selectHelper, toPartialGraphQLInfo } from "../builders";
import { bestLabel, defaultPermissions, oneIsPublic } from "../utils";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { RunProjectModel } from "./runProject";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { ProjectModel } from "./project";

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

const __typename = 'ProjectVersion' as const;
type Permissions = Pick<VersionYou, 'canCopy' | 'canDelete' | 'canUpdate' | 'canReport' | 'canUse' | 'canRead'>;
const suppFields = ['you'] as const;
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
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
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
            __typename,
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
            runProjectsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                console.log('in projectversion tographql a');
                const runs = async () => {
                    console.log('in projectversion tographql b', userData);
                    if (!userData || !partial.runs) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add projectVersionId, so we 
                    // can associate runs with their project
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGraphQLInfo(partial.runs as PartialGraphQLInfo, RunProjectModel.format.gqlRelMap, userData.languages, true),
                        projectVersionId: true
                    }
                    console.log('in projectversion tographql c');
                    // Query runs made by user
                    console.log('in projectversion tographql before runs', selectHelper(partial));
                    let runs: any[] = await prisma.run_project.findMany({
                        where: {
                            AND: [
                                { projectVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } }
                            ]
                        },
                        ...selectHelper(runPartial)
                    });
                    console.log('in projectversion tographql d');
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGraphQL(r, runPartial));
                    // Add supplemental fields
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    // Split runs by id
                    const projectRuns = ids.map((id) => runs.filter(r => r.projectVersionId === id));
                    return projectRuns;
                };
                console.log('in projectversion tographql e');
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        runs: await runs(),
                    }
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
            createdByIdRoot: true,
            createdTimeFrame: true,
            directoryListingsId: true,
            isCompleteWithRoot: true,
            isCompleteWithRootExcludeOwnedByOrganizationId: true,
            isCompleteWithRootExcludeOwnedByUserId: true,
            maxComplexity: true,
            maxSimplicity: true,
            maxTimesCompleted: true,
            minComplexity: true,
            minSimplicity: true,
            minTimesCompleted: true,
            maxBookmarksRoot: true,
            maxScoreRoot: true,
            maxViewsRoot: true,
            minBookmarksRoot: true,
            minScoreRoot: true,
            minViewsRoot: true,
            ownedByOrganizationIdRoot: true,
            ownedByUserIdRoot: true,
            rootId: true,
            tagsRoot: true,
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
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            ProjectModel.validate!.isPublic(data.root as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ProjectModel.validate!.owner(data.root as any),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ['Project', ['versions']],
        }),
        permissionResolvers: defaultPermissions,
        validations: {
            async common({ createMany, deleteMany, languages, prisma, updateMany }) {
                await versionsCheck({
                    createMany,
                    deleteMany,
                    languages,
                    objectType: 'Project',
                    prisma,
                    updateMany: updateMany as any,
                });
            },
            async create({ createMany, languages }) {
                createMany.forEach(input => lineBreaksCheck(input, ['description'], 'LineBreaksBio', languages))
            },
            async update({ languages, updateMany }) {
                updateMany.forEach(({ data }) => lineBreaksCheck(data, ['description'], 'LineBreaksBio', languages));
            },
        },
        visibility: {
            private: {
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ]
            },
            public: {
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ]
            },
            owner: (userId) => ({
                root: ProjectModel.validate!.visibility.owner(userId),
            }),
        },
    },
})