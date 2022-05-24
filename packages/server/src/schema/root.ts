import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo, GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
// import ogs from 'open-graph-scraper';
import { CODE } from '@local/shared';
import { Context } from '../context';
import { GraphQLModelType } from '../models';
import { CustomError } from '../error';
import { rateLimit } from '../rateLimit';
import { resolveContributor } from './resolvers';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    scalar Date
    scalar Upload

    # Used for Projects, Standards, and Routines, since they can be created 
    # by either a User or an Organization.
    union Contributor = User | Organization

    # Used for filtering by date created/updated, as well as fetching metrics (e.g. monthly active users)
    input TimeFrame {
        after: Date
        before: Date
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

    input ReadAssetsInput {
        files: [String!]!
    }

    input WriteAssetsInput {
        files: [Upload!]!
    }

    # Input for finding object by id
    input FindByIdInput {
        id: ID!
    }

    # Input for finding object by id OR handle
    input FindByIdOrHandleInput {
        id: ID
        handle: String
    }

    # Input for deleting one object
    input DeleteOneInput {
        id: ID!
    }

    # Input for deleting multiple objects
    input DeleteManyInput {
        ids: [ID!]!
    }

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
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
    Contributor: {
        __resolveType(obj: any) { return resolveContributor(obj) },
    },
    Query: {
        readAssets: async (_parent: undefined, { input }: any, context: Context, info: GraphQLResolveInfo): Promise<Array<String | null>> => {
            await rateLimit({ context, info, max: 1000 });
            return await readFiles(input.files);
        },
    },
    Mutation: {
        writeAssets: async (_parent: undefined, { input }: any, context: Context, info: GraphQLResolveInfo): Promise<boolean> => {
            await rateLimit({ context, info, max: 500 });
            throw new CustomError(CODE.NotImplemented); // TODO add safety checks before allowing uploads
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}