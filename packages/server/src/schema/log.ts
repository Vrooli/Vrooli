/**
 * User log (e.g. created a project, completed a routine)
 */
import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { FindByIdInput, Count, LogSortBy, LogType, DeleteManyInput, Log, LogSearchResult, LogSearchInput } from './types';
import { Context } from '../context';
import { deleteManyHelper, LogModel, readManyHelper, readOneHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

export const typeDef = gql`
    enum LogSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum LogType {
        Complete
        Create
        Update
        Delete
        Start 
        Run
        RunComplete
        Cancel
        Join
        Leave
        AddMember
        RemoveMember
        UpdateMember 
    }
 
    input LogCreateInput {
        log: LogType!
        data: String
        table1: String
        table2: String
        object1Id: ID
        object2Id: ID
        userId: ID!
    }
    type Log {
        id: ID!
        created_at: Date!
        log: LogType!
        data: String
        table1: String
        table2: String
        object1Id: ID
        object2Id: ID
    }

    input LogSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: LogSortBy
        table1: String
        table2: String
        object1Id: ID
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
        log(input: FindByIdInput!): Log
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
        log: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Log> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, LogModel(context.prisma));
        },
        logs: async (_parent: undefined, { input }: IWrap<LogSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<LogSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, LogModel(context.prisma));
        },
    },
    // Logs are created automatically, so the only mutation is to delete them
    Mutation: {
        logDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            return deleteManyHelper(context.req.userId, input, LogModel(context.prisma));
        },
    }
}