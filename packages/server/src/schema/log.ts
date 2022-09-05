/**
 * User log (e.g. created a project, completed a routine)
 */
import { gql } from 'apollo-server-express';
import { IWrap } from '../types';
import { Count, LogSortBy, DeleteManyInput, LogSearchResult, LogSearchInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE } from '@shared/consts';
import { logSearcher, LogType, paginatedMongoSearch } from '../models';

export const typeDef = gql`
    enum LogSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum LogType {
        Copy
        Create
        Delete
        Downvote
        Fork
        OrganizationAddMember
        OrganizationJoin
        OrganizationLeave
        OrganizationRemoveMember
        OrganizationUpdateMember
        ProjectComplete
        RemoveStar
        RemoveVote
        RoutineCancel
        RoutineComplete
        RoutineStartIncomplete
        RoutineStartCanceled
        RoutineStartCompleted
        Star
        Update
        Upvote
        View
    }
 
    input LogCreateInput {
        action: LogType!
        object1Type: String
        object1Id: ID
        object2Type: String
        object2Id: ID
        data: String
    }
    type Log {
        id: ID!
        timestamp: Date!
        action: LogType!
        object1Type: String
        object1Id: ID
        object2Type: String
        object2Id: ID
        data: String
    }

    input LogSearchInput {
        actions: [String!]
        after: String
        createdTimeFrame: TimeFrame
        data: String
        ids: [ID!]
        sortBy: LogSortBy
        object1Type: String
        object1Id: ID
        object2Type: String
        object2Id: ID
        take: Int
    }
 
    # Return type for search result
    type LogSearchResult {
        pageInfo: PageInfo!
        edges: [LogEdge!]!
    }
 
    # Return type for search result edge
    type LogEdge {
        cursor: String!
        node: Log!
    }
 
    extend type Query {
        logs(input: LogSearchInput!): LogSearchResult!
    }
 
    extend type Mutation {
        logDeleteMany(input: DeleteManyInput): Count!
    }
 `

export const resolvers = {
    LogSortBy: LogSortBy,
    LogType: LogType,
    Query: {
        logs: async (_parent: undefined, { input }: IWrap<LogSearchInput>, { req }: Context, info: GraphQLResolveInfo): Promise<LogSearchResult> => {
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            // Create the search and sort queries
            const findQuery = logSearcher().getFindQuery(req.userId ?? '', input);
            const sortQuery = logSearcher().getSortQuery(input.sortBy ?? LogSortBy.DateCreatedDesc);
            const project = logSearcher().defaultProjection;
            return paginatedMongoSearch<LogSearchResult>({
                findQuery,
                sortQuery,
                take: input.take ?? 10,
                after: input.after ?? undefined,
                project,
            });
        },
    },
    // Logs are created automatically, so the only mutation is to delete them
    Mutation: {
        logDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            throw new CustomError(CODE.NotImplemented);
            // return deleteManyHelper(context.req.userId, input, LogModel(context.prisma));
        },
    }
}