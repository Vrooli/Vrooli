import { MaxObjects, ProjectVersionSortBy, getTranslation, projectVersionValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { afterMutationsVersion } from "../../utils/shapes/afterMutationsVersion.js";
import { preShapeVersion, type PreShapeVersionResult } from "../../utils/shapes/preShapeVersion.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { lineBreaksCheck } from "../../validators/lineBreaksCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { versionsCheck } from "../../validators/versionsCheck.js";
import { ProjectVersionFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ProjectModelInfo, ProjectModelLogic, ProjectVersionModelInfo, ProjectVersionModelLogic } from "./types.js";

type ProjectVersionPre = PreShapeVersionResult;

const __typename = "ProjectVersion" as const;
export const ProjectVersionModel: ProjectVersionModelLogic = ({
    __typename,
    dbTable: "project_version",
    dbTranslationTable: "project_version_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: (select, languages) => getTranslation(select, languages).name ?? "",
        },
        embed: {
            select: () => ({
                id: true,
                root: { select: { tags: { select: { tag: true } } } },
                translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } },
            }),
            get: ({ root, translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    name: trans.name,
                    tags: (root as any).tags.map(({ tag }) => tag),
                    description: trans.description,
                }, languages?.[0]);
            },
        },
    }),
    format: ProjectVersionFormat,
    mutate: {
        shape: {
            pre: async (params): Promise<ProjectVersionPre> => {
                const { Create, Update, Delete, userData } = params;
                await versionsCheck({
                    Create,
                    Delete,
                    objectType: __typename,
                    Update,
                    userData,
                });
                [...Create, ...Update].map(d => d.input).forEach(input => lineBreaksCheck(input, ["description"], "LineBreaksBio"));
                const maps = preShapeVersion<"id">({ Create, Update, objectType: __typename });
                return { ...maps };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ProjectVersionPre;
                return {
                    id: data.id,
                    isPrivate: data.isPrivate,
                    isComplete: noNull(data.isComplete),
                    versionLabel: data.versionLabel,
                    versionNotes: noNull(data.versionNotes),
                    directories: await shapeHelper({ relation: "directories", relTypes: ["Create"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Connect", "Create"], isOneToOne: true, objectType: "Project", parentRelationshipName: "versions", data, ...rest }),
                    // ...(await shapeHelper({ relation: "suggestedNextByProject", relTypes: ['Connect'], isOneToOne: false,   objectType: 'ProjectVersionEndNext', parentRelationshipName: 'fromProjectVersion', data, ...rest })), //TODO needs join table
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ProjectVersionPre;
                return {
                    isPrivate: noNull(data.isPrivate),
                    versionLabel: noNull(data.versionLabel),
                    versionNotes: noNull(data.versionNotes),
                    directories: await shapeHelper({ relation: "directories", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ProjectVersionDirectory", parentRelationshipName: "projectVersion", data, ...rest }),
                    root: await shapeHelper({ relation: "root", relTypes: ["Update"], isOneToOne: true, objectType: "Project", parentRelationshipName: "versions", data, ...rest }),
                    // ...(await shapeHelper({ relation: "suggestedNextByProject", relTypes: ['Connect', 'Disconnect'], isOneToOne: false,   objectType: 'ProjectVersionEndNext', parentRelationshipName: 'fromProjectVersion', data, ...rest })), needs join table
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
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
    //         req: Request,
    //         input: ProjectVersionContentsSearchInput,
    //         info: PartialApiInfo,
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
    //         }, true);
    //         // Determine text search query
    //         const searchQuery = input.searchString ? getSearchStringQuery({ objectType: 'Comment', searchString: input.searchString }) : undefined;
    //         // Loop through search fields and add each to the search query, 
    //         // if the field is specified in the input
    //         const customQueries: { [x: string]: any }[] = [];
    //         for (const field of Object.keys(ModelMap.get<CommentModelLogic>("Comment").search.searchFields)) {
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
    //         const orderByField = input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort;
    //         const orderByIsValid = ModelMap.get<CommentModelLogic>("Comment").search.sortBy[orderByField] === undefined
    //         const orderBy = orderByIsValid ? SortMap[input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort] : undefined;
    //         // Find requested search array
    //         const searchResults = await DbProvider.get().comment.findMany({
    //             where,
    //             orderBy,
    //             take: input.take ?? 10,
    //             skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
    //             cursor: input.after ? {
    //                 id: input.after
    //             } : undefined,
    //             ...InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo)
    //         });
    //         // If there are no results
    //         if (searchResults.length === 0) return {
    //             __typename: "CommentSearchResult" as const,
    //             totalThreads: 0,
    //             threads: [],
    //         }
    //         // Query total in thread, if cursor is not provided (since this means this data was already given to the user earlier)
    //         const totalInThread = input.after ? undefined : await DbProvider.get().comment.count({
    //             where: { ...where }
    //         });
    //         // Calculate end cursor
    //         const endCursor = searchResults[searchResults.length - 1].id;
    //         // If not as nestLimit, recurse with all result IDs
    //         const childThreads = nestLimit > 0 ? await this.searchThreads(getUser(req.session), {
    //             ids: searchResults.map(r => r.id),
    //             take: input.take ?? 10,
    //             sortBy: input.sortBy ?? ModelMap.get<CommentModelLogic>("Comment").search.defaultSort,
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
    //         comments = comments.map((c: any) => InfoConverter.get().fromDbToApi(c, partialInfo as PartialApiInfo));
    //         comments = await addSupplementalFields(getUser(req.session), comments, partialInfo);
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
            isCompleteWithRootExcludeOwnedByTeamId: true,
            isCompleteWithRootExcludeOwnedByUserId: true,
            isLatest: true,
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
            ownedByTeamIdRoot: true,
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ProjectVersionModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: (data) => data.isDeleted || data.root.isDeleted,
        isPublic: (data, ...rest) => data.isPrivate === false &&
            data.isDeleted === false &&
            oneIsPublic<ProjectVersionModelInfo["DbSelect"]>([["root", "Project"]], data, ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<ProjectModelLogic>("Project").validate().owner(data?.root as ProjectModelInfo["DbModel"], userId),
        permissionsSelect: () => ({
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: ["Project", ["versions"]],
        }),
        permissionResolvers: defaultPermissions,
        visibility: {
            own: function getOwn(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    root: useVisibility("Project", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Objects you own
                        {
                            root: useVisibility("Project", "Own", data),
                        },
                        // Public objects
                        {
                            isPrivate: false, // Can't be private
                            root: (useVisibility("ProjectVersion", "Public", data) as { root: object }).root,
                        },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Private versions you own
                        {
                            isPrivate: true, // Version is private
                            root: useVisibility("Project", "Own", data),
                        },
                        // Private roots you own
                        {
                            root: {
                                isPrivate: true, // Root is private
                                ...useVisibility("Project", "Own", data),
                            },
                        },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    OR: [
                        // Public versions you own
                        {
                            isPrivate: false, // Version is public
                            root: useVisibility("Project", "Own", data),
                        },
                        // Public roots you own
                        {
                            root: {
                                isPrivate: false, // Root is public
                                ...useVisibility("Project", "Own", data),
                            },
                        },
                    ],
                };
            },
            public: function getPublic(data) {
                return {
                    isDeleted: false, // Can't be deleted
                    isPrivate: false, // Version can't be private
                    root: {
                        isDeleted: false, // Root can't be deleted
                        isPrivate: false, // Root can't be private
                        OR: [
                            // Unowned
                            { ownedByTeam: null, ownedByUser: null },
                            // Owned by public teams
                            { ownedByTeam: useVisibility("Team", "Public", data) },
                            // Owned by public users
                            { ownedByUser: { isPrivate: false, isPrivateProjects: false } },
                        ],
                    },
                };
            },
        },
    }),
});
