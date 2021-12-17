import { gql } from 'apollo-server-express';
import { GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
import ogs from 'open-graph-scraper';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Return type for delete mutations,
    # which return the number of affected rows
    type Count {
        count: Int
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

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(files: [String!]!): [String]!
        readOpenGraph(url: String!): OpenGraphResponse!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(files: [Upload!]!): Boolean
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
        readAssets: async (_parent: undefined, args: any) => {
            return await readFiles(args.files);
        },
        readOpenGraph: async (_parent: undefined, args: any) => {
            return await ogs({ url: args.url })
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
        writeAssets: async (_parent: undefined, args: any) => {
            const data = await saveFiles(args.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}