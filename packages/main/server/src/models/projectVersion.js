import { MaxObjects, ProjectVersionSortBy } from "@local/consts";
import { projectVersionValidation } from "@local/validation";
import { addSupplementalFields, modelToGql, noNull, selectHelper, shapeHelper, toPartialGqlInfo } from "../builders";
import { bestLabel, defaultPermissions, postShapeVersion, translationShapeHelper } from "../utils";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../validators";
import { ProjectModel } from "./project";
import { RunProjectModel } from "./runProject";
const __typename = "ProjectVersion";
const suppFields = ["you"];
export const ProjectVersionModel = ({
    __typename,
    delegate: (prisma) => prisma.project_version,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, "name", languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: "Comment",
            directories: "ProjectVersionDirectory",
            directoryListings: "ProjectVersionDirectory",
            forks: "Project",
            pullRequest: "PullRequest",
            reports: "Report",
            root: "Project",
        },
        prismaRelMap: {
            __typename,
            comments: "Comment",
            directories: "ProjectVersionDirectory",
            directoryListings: "ProjectVersionDirectory",
            pullRequest: "PullRequest",
            reports: "Report",
            resourceList: "ResourceList",
            root: "Project",
            forks: "Project",
            runProjects: "RunProject",
            suggestedNextByProject: "ProjectVersion",
        },
        joinMap: {
            suggestedNextByProject: "toProjectVersion",
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
                const runs = async () => {
                    if (!userData || !partial.runs)
                        return new Array(objects.length).fill([]);
                    const runPartial = {
                        ...toPartialGqlInfo(partial.runs, RunProjectModel.format.gqlRelMap, userData.languages, true),
                        projectVersionId: true,
                    };
                    let runs = await prisma.run_project.findMany({
                        where: {
                            AND: [
                                { projectVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } },
                            ],
                        },
                        ...selectHelper(runPartial),
                    });
                    runs = runs.map(r => modelToGql(r, runPartial));
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    const projectRuns = ids.map((id) => runs.filter(r => r.projectVersionId === id));
                    return projectRuns;
                };
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                        runs: await runs(),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, updateList, deleteList, prisma, userData }) => {
                await versionsCheck({
                    createList,
                    deleteList,
                    objectType: __typename,
                    prisma,
                    updateList,
                    userData,
                });
                const combined = [...createList, ...updateList.map(({ data }) => data)];
                combined.forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: noNull(data.isPrivate),
                isComplete: noNull(data.isComplete),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directories", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Project", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directories", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "Project", parentRelationshipName: "versions", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
            post: async (params) => {
                await postShapeVersion({ ...params, objectType: __typename });
            },
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
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, languages) => data.isPrivate === false &&
            data.isDeleted === false &&
            ProjectModel.validate.isPublic(data.root, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ProjectModel.validate.owner(data.root, userId),
        permissionsSelect: (...params) => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Project", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            private: {
                isDeleted: false,
                root: { isDeleted: false },
                OR: [
                    { isPrivate: true },
                    { root: { isPrivate: true } },
                ],
            },
            public: {
                isDeleted: false,
                root: { isDeleted: false },
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ],
            },
            owner: (userId) => ({
                root: ProjectModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=projectVersion.js.map