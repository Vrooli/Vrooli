
export const typeDef = `#graphql
    input NodeEndCreateInput {
        id: ID!
        wasSuccessful: Boolean
        nodeConnect: ID!
        suggestedNextRoutineVersionsConnect: [ID!]
    }
    input NodeEndUpdateInput {
        id: ID!
        wasSuccessful: Boolean
        suggestedNextRoutineVersionsConnect: [ID!]
        suggestedNextRoutineVersionsDisconnect: [ID!]
    }
    type NodeEnd {
        id: ID!
        node: Node!
        wasSuccessful: Boolean!
        suggestedNextRoutineVersions: [RoutineVersion!]
    }
`;

export const resolvers = {};
