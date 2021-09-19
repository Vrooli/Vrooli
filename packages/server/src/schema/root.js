import { gql } from 'apollo-server-express';
import { GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
import _ from 'lodash';

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
    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        # _empty: String
        readAssets(files: [String!]!): [String]!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(files: [Upload!]!): Boolean
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
        parseLiteral(ast) {
            return new Date(value).toDateString(); // ast value is always in string format
        }
    }),
    Query: {
        readAssets: async (_, args) => {
            return await readFiles(args.files);
        },
    },
    Mutation: {
        writeAssets: async (_, args) => {
            const data = await saveFiles(args.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}