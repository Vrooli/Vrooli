import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeRoutineListCreateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        itemsCreate: [NodeRoutineListItemCreateInput!]
        nodeConnect: ID!
    }
    input NodeRoutineListUpdateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        itemsDelete: [ID!]
        itemsCreate: [NodeRoutineListItemCreateInput!]
        itemsUpdate: [NodeRoutineListItemUpdateInput!]
    }
    type NodeRoutineList {
        type: GqlModelType!
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
        items: [NodeRoutineListItem!]!
    }
`
export const resolvers: {
} = {
}