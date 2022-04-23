import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, Log as LogReturn, Routine, RoutineCountInput, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, Success, RoutineSearchResult, RoutineSortBy, RoutineStartInput, RoutineCompleteInput, RoutineCancelInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteOneHelper, GraphQLModelType, Log, LogType, readManyHelper, readOneHelper, RoutineModel, updateHelper } from '../models';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE } from '@local/shared';
import { genErrorCode } from '../logger';

export const typeDef = gql`
    enum RoutineSortBy {
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input RoutineCreateInput {
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        version: String
        parentId: ID
        projectId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [InputItemCreateInput!]
        outputsCreate: [OutputItemCreateInput!]
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [RoutineTranslationCreateInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        version: String
        parentId: ID
        userId: ID
        organizationId: ID
        nodesDelete: [ID!]
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodeLinksDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        inputsCreate: [InputItemCreateInput!]
        inputsUpdate: [InputItemUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [OutputItemCreateInput!]
        outputsUpdate: [OutputItemUpdateInput!]
        outputsDelete: [ID!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [RoutineTranslationCreateInput!]
        translationsUpdate: [RoutineTranslationUpdateInput!]
    }
    type Routine {
        id: ID!
        completedAt: Date
        complexity: Int!
        created_at: Date!
        updated_at: Date!
        isAutomatable: Boolean
        isComplete: Boolean!
        isInternal: Boolean
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        score: Int!
        simplicity: Int!
        stars: Int!
        version: String
        comments: [Comment!]!
        creator: Contributor
        forks: [Routine!]!
        inputs: [InputItem!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        nodesCount: Int
        nodeLinks: [NodeLink!]!
        outputs: [OutputItem!]!
        owner: Contributor
        parent: Routine
        project: Project
        reports: [Report!]!
        resourceLists: [ResourceList!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [RoutineTranslation!]!
    }

    input RoutineTranslationCreateInput {
        language: String!
        description: String
        instructions: String!
        title: String!
    }
    input RoutineTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        instructions: String
        title: String
    }
    type RoutineTranslation {
        id: ID!
        language: String!
        description: String
        instructions: String!
        title: String!
    }

    input InputItemCreateInput {
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    input InputItemUpdateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    type InputItem {
        id: ID!
        isRequired: Boolean
        name: String
        routine: Routine!
        standard: Standard
        translations: [InputItemTranslation!]!
    }

    input InputItemTranslationCreateInput {
        language: String!
        description: String
    }
    input InputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type InputItemTranslation {
        id: ID!
        language: String!
        description: String
    }

    input OutputItemCreateInput {
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsCreate: [OutputItemTranslationCreateInput!]
    }
    input OutputItemUpdateInput {
        id: ID!
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [OutputItemTranslationCreateInput!]
        translationsUpdate: [OutputItemTranslationUpdateInput!]
    }
    type OutputItem {
        id: ID!
        name: String
        routine: Routine!
        standard: Standard
        translations: [OutputItemTranslation!]!
    }

    input OutputItemTranslationCreateInput {
        language: String!
        description: String
    }
    input OutputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
    }
    type OutputItemTranslation {
        id: ID!
        language: String!
        description: String
    }

    input RoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        isComplete: Boolean
        languages: [String!]
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        minScore: Int
        minStars: Int
        organizationId: ID
        projectId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
    }

    # Return type for search result
    type RoutineSearchResult {
        pageInfo: PageInfo!
        edges: [RoutineEdge!]!
    }

    # Return type for search result edge
    type RoutineEdge {
        cursor: String!
        node: Routine!
    }

    # Input for count
    input RoutineCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    input RoutineStartInput {
        id: ID!
    }

    input RoutineCompleteInput {
        id: ID!
    }

    input RoutineCancelInput {
        id: ID!
    }

    extend type Query {
        routine(input: FindByIdInput!): Routine
        routines(input: RoutineSearchInput!): RoutineSearchResult!
        routinesCount(input: RoutineCountInput!): Int!
    }

    extend type Mutation {
        routineCreate(input: RoutineCreateInput!): Routine!
        routineUpdate(input: RoutineUpdateInput!): Routine!
        routineDeleteOne(input: DeleteOneInput!): Success!
        routineStart(input: RoutineStartInput!): Log!
        routineComplete(input: RoutineCompleteInput!): Log!
        routineCancel(input: RoutineCancelInput!): Log!
    }
`

// All logs related to executing routines. Used to query for last status of routine.
const routineLogs = [LogType.RoutineCancel, LogType.RoutineComplete, LogType.RoutineStartCanceled, LogType.RoutineStartCompleted, LogType.RoutineStartIncomplete]

export const resolvers = {
    RoutineSortBy: RoutineSortBy,
    Query: {
        routine: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, RoutineModel(context.prisma));
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutineSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<RoutineSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, RoutineModel(context.prisma));
        },
        routinesCount: async (_parent: undefined, { input }: IWrap<RoutineCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, RoutineModel(context.prisma));
        },
    },
    Mutation: {
        routineCreate: async (_parent: undefined, { input }: IWrap<RoutineCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            return createHelper(context.req.userId, input, info, RoutineModel(context.prisma));
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            return updateHelper(context.req.userId, input, info, RoutineModel(context.prisma));
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return deleteOneHelper(context.req.userId, input, RoutineModel(context.prisma));
        },
        routineStart: async (_parent: undefined, { input }: IWrap<RoutineStartInput>, context: Context, info: GraphQLResolveInfo): Promise<LogReturn> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Cannot log routine start if you are not logged in', { code: genErrorCode('0153') });
            // Check if log already exists
            const lastLog = await Log.find({
                action: { $in: routineLogs },
                input1Type: GraphQLModelType.Routine,
                input1Id: input.id,
                userId: context.req.userId,
            }).sort({ timestamp: -1 }).limit(1).lean().exec();
            // If last log exists and is start, return it
            if (lastLog.length > 0 && lastLog[0].action === LogType.RoutineStartIncomplete) return lastLog[0];
            // Otherwise, create new start log
            else {
                const logData = {
                    timestamp: Date.now(),
                    userId: context.req.userId,
                    action: LogType.RoutineStartIncomplete,
                    object1Type: GraphQLModelType.Routine,
                    object1Id: input.id,
                }
                const log = await Log.collection.insertOne(logData)
                return { id: log.insertedId.toString(), ...logData } as any; //TODO remove any
            }
        },
        routineComplete: async (_parent: undefined, { input }: IWrap<RoutineCompleteInput>, context: Context, info: GraphQLResolveInfo): Promise<LogReturn> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Cannot log routine complete if you are not logged in', { code: genErrorCode('0154') });
            // Make sure start log exists
            const lastLog = await Log.find({
                action: { $in: routineLogs },
                input1Type: GraphQLModelType.Routine,
                input1Id: input.id,
                userId: context.req.userId,
            }).sort({ timestamp: -1 }).limit(1).lean().exec();
            if (!lastLog || lastLog.length === 0 || lastLog[0].action !== LogType.RoutineStartIncomplete) throw new CustomError(CODE.InternalError, 'Could not find start log for routine');
            // Update old log from incomplete to complete
            await Log.updateOne({ _id: lastLog[0]._id }, { action: LogType.RoutineStartCompleted }).exec();
            // Insert new complete log
            const logData = {
                timestamp: Date.now(),
                userId: context.req.userId,
                action: LogType.RoutineComplete,
                object1Type: GraphQLModelType.Routine,
                object1Id: input.id,
            }
            const log = await Log.collection.insertOne(logData)
            return { id: log.insertedId.toString(), ...logData } as any; //TODO remove any
        },
        routineCancel: async (_parent: undefined, { input }: IWrap<RoutineCancelInput>, context: Context, info: GraphQLResolveInfo): Promise<LogReturn> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Cannot log routine cancel if you are not logged in', { code: genErrorCode('0155') });
            // Make sure start log exists
            const lastLog = await Log.find({
                action: { $in: routineLogs },
                input1Type: GraphQLModelType.Routine,
                input1Id: input.id,
                userId: context.req.userId,
            }).sort({ timestamp: -1 }).limit(1).lean().exec();
            if (!lastLog || lastLog.length === 0 || lastLog[0].action !== LogType.RoutineStartIncomplete) 
                throw new CustomError(CODE.InternalError, 'Could not find start log for routine', { code: genErrorCode('0156') });
            // Update old log from incomplete to canceled
            await Log.updateOne({ _id: lastLog[0]._id }, { action: LogType.RoutineStartCanceled }).exec();
            // Insert new canceled log
            const logData = {
                timestamp: Date.now(),
                userId: context.req.userId,
                action: LogType.RoutineCancel,
                object1Type: GraphQLModelType.Routine,
                object1Id: input.id,
            }
            const log = await Log.collection.insertOne(logData)
            return { id: log.insertedId.toString(), ...logData } as any; //TODO remove any
        },
    }
}