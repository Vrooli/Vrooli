import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input ProjectVersionDirectoryCreateInput {
        id: ID!
    }
    input ProjectVersionDirectoryUpdateInput {
        id: ID!
    }
    type ProjectVersionDirectory {
        id: ID!
    }
`

export const resolvers: {
} = {
}