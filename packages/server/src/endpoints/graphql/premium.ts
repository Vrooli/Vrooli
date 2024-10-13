
export const typeDef = `#graphql
    type Premium {
        id: ID!
        credits: Int!
        customPlan: String
        enabledAt: Date
        expiresAt: Date
        isActive: Boolean!
    }
`;

export const resolvers = {};
