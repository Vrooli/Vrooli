import { gql } from 'apollo-server-express';
import { GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
import ogs from 'open-graph-scraper';
import { OpenGraphResponse } from './types';
import { METRIC_TIME_FRAME } from '@local/shared';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Enums for counting objects recently created or updated (for metrics)
    enum MetricTimeFrame {
        Daily
        Weekly
        Monthly
        Yearly
    }

    # Return type for a cursor-based pagination's pageInfo response
    type PageInfo {
        hasNextPage: Boolean!
        endCursor: String
    }
    # Return type for delete mutations,
    # which return the number of affected rows
    type Count {
        count: Int
    }
    # Return type for mutations with a success boolean
    # Could return just the boolean, but this makes it clear what the result means
    type Success {
        success: Boolean
    }
    # Return type for error messages
    type Response {
        code: Int
        message: String!
    }

    # Return type for Open Graph queries
    type OpenGraphResponse {
        site: String
        title: String
        description: String
        imageUrl: String
    }

    input ReadAssetsInput {
        files: [String!]!
    }

    input ReadOpenGraphInput {
        url: String!
    }

    input WriteAssetsInput {
        files: [Upload!]!
    }

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
        readOpenGraph(input: ReadOpenGraphInput!): OpenGraphResponse!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
    }

    # Input for finding object by id
    input FindByIdInput {
        id: ID!
    }

    # Input for deleting one object
    input DeleteOneInput {
        id: ID!
    }

    # Input for deleting multiple objects
    input DeleteManyInput {
        ids: [ID!]!
    }

    # Input for reporting an object
    input ReportInput {
        id: ID!
        reason: String
    }
`

export const resolvers = {
    Upload: GraphQLUpload,
    MetricTimeFrame: METRIC_TIME_FRAME,
    Date: new GraphQLScalarType({
        name: "Date",
        description: "Custom description for the date scalar",
        // Assumes data is either Unix timestamp or Date object
        parseValue(value) {
            return new Date(value).toISOString(); // value from the client
        },
        serialize(value) {
            return new Date(value).getTime(); // value sent to the client
        },
        parseLiteral(ast: any) {
            return new Date(ast).toDateString(); // ast value is always in string format
        }
    }),
    Query: {
        readAssets: async (_parent: undefined, { input }: any): Promise<Array<String | null>> => {
            return await readFiles(input.files);
        },
        readOpenGraph: async (_parent: undefined, { input }: any): Promise<OpenGraphResponse> => {
            return await ogs({ url: input.url })
                .then((data: any) => {
                    const { result } = data;
                    return {
                        site: result?.ogSiteName,
                        title: result?.ogTitle,
                        description: result?.ogDescription,
                        imageUrl: result?.ogImage?.url,
                    };
                }).catch(err => {
                    console.error('Caught error fetching Open Graph url', err);
                    return {};
                }).finally(() => {return {}})
        }
    },
    Mutation: {
        writeAssets: async (_parent: undefined, { input }: any): Promise<boolean> => {
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}