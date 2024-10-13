
export const typeDef = `#graphql
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
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
        items: [NodeRoutineListItem!]!
        node: Node!
    }
`;

export const resolvers = {};
