import { noNull, shapeHelper } from "../../builders";
import { postShapeVersion, translationShapeHelper } from "../../utils";
import { Formatter } from "../types";
import { ProjectModel } from "./project";

const __typename = "ProjectVersion" as const;
export const ProjectVersionFormat: Formatter<ModelProjectVersionLogic> = {
    gqlRelMap: {
        __typename,
        comments: "Comment",
        directories: "ProjectVersionDirectory",
        directoryListings: "ProjectVersionDirectory",
        forks: "Project",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Project",
        // 'runs.project': 'RunProject', //TODO
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
}'Connect'], isOneToOne: false, isRequired: false, objectType: 'ProjectVersionEndNext', parentRelationshipName: 'fromProjectVersion', data, ...rest })),
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
    post: async (params) => {
        await postShapeVersion({ ...params, objectType: __typename });
        //         const partial = toPartialGqlInfo(info, {
        //             __typename: "ProjectVersionContentsSearchResult",
        //             meetings: 'Meeting',
        //             notes: 'Note',
        //             reminders: 'Reminder',
        //             resources: 'Resource',
        //             runProjectSchedules: 'RunProjectSchedule',
        //             runRoutineSchedules: 'RunRoutineSchedule',
        //         for (const field of Object.keys(CommentModel.search!.searchFields)) {
        //             if (input[field as string] !== undefined) {
        //                 customQueries.push(SearchMap[field as string](input, getUser(req), __typename));
        //             }
        //         const searchResults = await prisma.comment.findMany({
        //             where,
        //             orderBy,
        //             take: input.take ?? 10,
        //             skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        //             cursor: input.after ? {
        //                 id: input.after
        //             } : undefined,
        //             ...selectHelper(partialInfo)
        //         if (searchResults.length === 0) return {
        //             __typename: "CommentSearchResult" as const,
        //             totalThreads: 0,
        //             threads: [],
        //         const totalInThread = input.after ? undefined : await prisma.comment.count({
        //             where: { ...where }
        //         const childThreads = nestLimit > 0 ? await this.searchThreads(prisma, getUser(req), {
        //             ids: searchResults.map(r => r.id),
        //             take: input.take ?? 10,
        //             sortBy: input.sortBy ?? CommentModel.search!.defaultSort,
        //         const flattenThreads = (threads: CommentThread[]) => {
        //             const result: Comment[] = [];
        //             for (const thread of threads) {
        //                 result.push(thread.comment);
        //                 result.push(...flattenThreads(thread.childThreads));
        //             }
        //             return result;
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
        //         return {
        //             __typename: "CommentSearchResult" as const,
        //             totalThreads: totalInThread,
        //             threads,
        //             endCursor,
        private: {
            isDeleted: false,
                root: { isDeleted: false },
            OR: [
                { isPrivate: true },
                { root: { isPrivate: true } },
            ],
                public: {
                isDeleted: false,
                    root: { isDeleted: false },
                AND: [
                    { isPrivate: false },
                    { root: { isPrivate: false } },
                ],
                    owner: (userId) => ({
                        root: ProjectModel.validate!.visibility.owner(userId),
                    };
