/**
 * User log (e.g. created a project, completed a routine)
 */
import { gql } from 'apollo-server-express';
import { IWrap } from 'types';
import { Count, LogSortBy, DeleteManyInput, LogSearchResult, LogSearchInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { CustomError } from '../error';
import { CODE } from '@local/shared';
import { Log, logSearcher, LogType } from '../models';

export const typeDef = gql`
    enum LogSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum LogType {
        Create
        Delete
        OrganizationAddMember
        OrganizationJoin
        OrganizationLeave
        OrganizationRemoveMember
        OrganizationUpdateMember
        RoutineCancel
        RoutineComplete
        RoutineStart
        ProjectComplete
        Update
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
        action: String
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
        logs: async (_parent: undefined, { input }: IWrap<LogSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<LogSearchResult> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            // Initialize results
            let paginatedResults = {
                pageInfo: {
                    endCursor: null,
                    hasNextPage: false,
                },
                edges: []
            }
            // Attempt to query MongoDB
            try {
                // Create the search and sort queries
                const findQuery = logSearcher().getFindQuery(context.req.userId ?? '', input);
                console.log('logs findQuery', JSON.stringify(findQuery));
                const sortQuery = logSearcher().getSortQuery(input.sortBy ?? LogSortBy.DateCreatedDesc);
                console.log('logs sortQuery', JSON.stringify(sortQuery));
                // Perform a cursor-based paginated query, using input.after as the starting point
                const cursor = Log.find(findQuery).sort(sortQuery).skip(input.after ? parseInt(input.after) : 0).limit(input.take ?? 10).cursor();
                // const searchResults = await Log.find(findQuery).sort(sortQuery);
                console.log('SEARCHED LOGS', JSON.stringify(cursor));
                // Create the paginated results by iterating through the cursor
                for (let doc = await cursor.next(); doc != null; doc = await cursor.next()) {
                    console.log('GOT SERARCH DOC', JSON.stringify(doc));
                }
                //TODO
                // if (searchResults. > 0) {
                //     // Find cursor
                //     const cursor = searchResults[searchResults.length - 1]._id;
                //     // Query after the cursor to check if there are more results
                //     const hasNextPage = await PrismaMap[objectType](model.prisma).findMany({
                //         take: 1,
                //         cursor: {
                //             id: cursor
                //         }
                //     });
                //     paginatedResults = {
                //         pageInfo: {
                //             hasNextPage: hasNextPage.length > 0,
                //             endCursor: cursor,
                //         },
                //         edges: searchResults.map((result: any) => ({
                //             cursor: result.id,
                //             node: result,
                //         }))
                //     }
                // }
            } catch (e) {
                throw new CustomError(CODE.InternalError, 'Error searching logs');
            } finally {
                return paginatedResults;
            }
        },
    },
    // Logs are created automatically, so the only mutation is to delete them
    Mutation: {
        logDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            throw new CustomError(CODE.NotImplemented);
            // return deleteManyHelper(context.req.userId, input, LogModel(context.prisma));
        },
    }
}