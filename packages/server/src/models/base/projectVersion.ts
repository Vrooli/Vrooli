import { MaxObjects, ProjectVersionSortBy, projectVersionValidation } from "@local/shared";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { noNull } from "../../builders/noNull";
import { selectHelper } from "../../builders/selectHelper";
import { shapeHelper } from "../../builders/shapeHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { PartialGraphQLInfo } from "../../builders/types";
import { bestTranslation, defaultPermissions, getEmbeddableString, oneIsPublic } from "../../utils";
import { afterMutationsVersion, preShapeVersion, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, lineBreaksCheck, versionsCheck } from "../../validators";
import { ProjectVersionFormat } from "../formats";
import { ModelLogic } from "../types";
import { ProjectModel } from "./project";
import { RunProjectModel } from "./runProject";
import { ProjectModelLogic, ProjectVersionModelLogic } from "./types";

const __typename = "ProjectVersion" as const;
const suppFields = ["you"] as const;
export const ProjectVersionModel: ModelLogic<ProjectVersionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.project_version,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => bestTranslation(select.translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    name: trans?.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans?.description,
                }, languages[0]);
            },
        },
    },
    format: ProjectVersionFormat,
    mutate: {
        shape: {
            pre: async (params) => {
                const { Create, Update, Delete, prisma, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    prisma,
                    Update,
                    userData,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio", userData.languages));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isPrivate: data.isPrivate,
                isComplete: noNull(data.isComplete),
                versionLabel: data.versionLabel,
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directories", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, isRequired: true, objectType: "Project", parentRelationshipName: "versions", data, ...rest })),
                // ...(await shapeHelper({ relation: "suggestedNextByProject", relTypes: ['Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionEndNext', parentRelationshipName: 'fromProjectVersion', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                isPrivate: noNull(data.isPrivate),
                versionLabel: noNull(data.versionLabel),
                versionNotes: noNull(data.versionNotes),
                ...(await shapeHelper({ relation: "directories", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest })),
                ...(await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, isRequired: false, objectType: "Project", parentRelationshipName: "versions", data, ...rest })),
                // ...(await shapeHelper({ relation: "suggestedNextByProject", relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionEndNext', parentRelationshipName: 'fromProjectVersion', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        trigger: {
            afterMutations: async (params) => {
                await afterMutationsVersion({ ...params, objectType: __typename });
            },
        },
        yup: projectVersionValidation,
    },
    // query: {
    //     /**
    //      * Custom search query for querying items in a project version
    //      */
    //     async searchContents(
    //         prisma: PrismaType,
    //         req: Request,
    //         input: ProjectVersionContentsSearchInput,
    //         info: GraphQLInfo | PartialGraphQLInfo,
    //         nestLimit: number = 2,
    //     ): Promise<ProjectVersionContentsSearchResult> {
    //         // Partially convert info type
    //         const partial = toPartialGqlInfo(info, {
    //             __typename: "ProjectVersionContentsSearchResult",
    //             meetings: 'Meeting',
    //             notes: 'Note',
    //             reminders: 'Reminder',
    //             resources: 'Resource',
    //             runProjectSchedules: 'RunProjectSchedule',
    //             runRoutineSchedules: 'RunRoutineSchedule',
    //         }, req.session.languages, true);
    //         // Determine text search query
    //         const searchQuery = input.searchString ? getSearchStringQuery({ objectType: 'Comment', searchString: input.searchString }) : undefined;
    //         // Loop through search fields and add each to the search query, 
    //         // if the field is specified in the input
    //         const customQueries: { [x: string]: any }[] = [];
    //         for (const field of Object.keys(CommentModel.search.searchFields)) {
    //             if (input[field as string] !== undefined) {
    //                 customQueries.push(SearchMap[field as string](input, getUser(req.session), __typename));
    //             }
    //         }
    // Create query for visibility
    //const visibilityQuery = visibilityBuilder({ objectType: "Comment", userData: getUser(req.session), visibility: input.visibility ?? VisibilityType.Public });
    //         // Combine queries
    //         const where = combineQueries([searchQuery, visibilityQuery, ...customQueries]);
    //         // Determine sort order
    //         // Make sure sort field is valid
    //         const orderByField = input.sortBy ?? CommentModel.search.defaultSort;
    //         const orderByIsValid = CommentModel.search.sortBy[orderByField] === undefined
    //         const orderBy = orderByIsValid ? SortMap[input.sortBy ?? CommentModel.search.defaultSort] : undefined;
    //         // Find requested search array
    //         const searchResults = await prisma.comment.findMany({
    //             where,
    //             orderBy,
    //             take: input.take ?? 10,
    //             skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
    //             cursor: input.after ? {
    //                 id: input.after
    //             } : undefined,
    //             ...selectHelper(partialInfo)
    //         });
    //         // If there are no results
    //         if (searchResults.length === 0) return {
    //             __typename: "CommentSearchResult" as const,
    //             totalThreads: 0,
    //             threads: [],
    //         }
    //         // Query total in thread, if cursor is not provided (since this means this data was already given to the user earlier)
    //         const totalInThread = input.after ? undefined : await prisma.comment.count({
    //             where: { ...where }
    //         });
    //         // Calculate end cursor
    //         const endCursor = searchResults[searchResults.length - 1].id;
    //         // If not as nestLimit, recurse with all result IDs
    //         const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, getUser(req.session), {
    //             ids: searchResults.map(r => r.id),
    //             take: input.take ?? 10,
    //             sortBy: input.sortBy ?? CommentModel.search.defaultSort,
    //         }, info, nestLimit) : [];
    //         // Find every comment in "childThreads", and put into 1D array. This uses a helper function to handle recursion
    //         const flattenThreads = (threads: CommentThread[]) => {
    //             const result: Comment[] = [];
    //             for (const thread of threads) {
    //                 result.push(thread.comment);
    //                 result.push(...flattenThreads(thread.childThreads));
    //             }
    //             return result;
    //         }
    //         let comments: any = flattenThreads(childThreads);
    //         // Shape comments and add supplemental fields
    //         comments = comments.map((c: any) => modelToGql(c, partialInfo as PartialGraphQLInfo));
    //         comments = await addSupplementalFields(prisma, getUser(req.session), comments, partialInfo);
    //         // Put comments back into "threads" object, using another helper function. 
    //         // Comments can be matched by their ID
    //         const shapeThreads = (threads: CommentThread[]) => {
    //             const result: CommentThread[] = [];
    //             for (const thread of threads) {
    //                 // Find current-level comment
    //                 const comment = comments.find((c: any) => c.id === thread.comment.id);
    //                 // Recurse
    //                 const children = shapeThreads(thread.childThreads);
    //                 // Add thread to result
    //                 result.push({
    //                     __typename: "CommentThread" as const,
    //                     comment,
    //                     childThreads: children,
    //                     endCursor: thread.endCursor,
    //                     totalInThread: thread.totalInThread,
    //                 });
    //             }
    //             return result;
    //         }
    //         const threads = shapeThreads(childThreads);
    //         // Return result
    //         return {
    //             __typename: "CommentSearchResult" as const,
    //             totalThreads: totalInThread,
    //             threads,
    //             endCursor,
    //         }
    //     }
    // },
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
        },
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                { root: "tagsWrapped" },
                { root: "labelsWrapped" },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, objects, partial, prisma, userData }) => {
                const runs = async () => {
                    if (!userData || !partial.runs) return new Array(objects.length).fill([]);
                    // Find requested fields of runs. Also add projectVersionId, so we 
                    // can associate runs with their project
                    const runPartial: PartialGraphQLInfo = {
                        ...toPartialGqlInfo(partial.runs as PartialGraphQLInfo, RunProjectModel.format.gqlRelMap, userData.languages, true),
                        projectVersionId: true,
                    };
                    // Query runs made by user
                    let runs: any[] = await prisma.run_project.findMany({
                        where: {
                            AND: [
                                { projectVersion: { root: { id: { in: ids } } } },
                                { user: { id: userData.id } },
                            ],
                        },
                        ...selectHelper(runPartial),
                    });
                    // Format runs to GraphQL
                    runs = runs.map(r => modelToGql(r, runPartial));
                    // Add supplemental fields
                    runs = await addSupplementalFields(prisma, userData, runs, runPartial);
                    // Split runs by id
                    const projectRuns = ids.map((id) => runs.filter(r => r.projectVersionId === id));
                    return projectRuns;
                };
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        runs: await runs(),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<ProjectVersionModelLogic["PrismaSelect"]>([["root", "Project"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ProjectModel.validate.owner(data?.root as ProjectModelLogic["PrismaModel"], userId),
        permissionsSelect: () => ({
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
