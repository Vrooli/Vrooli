import { gql } from "apollo-server-express";
import { EndpointsTranslate, TranslateEndpoints } from "../logic/translate";

export const typeDef = gql`
    input TranslateInput {
        fields: String!
        languageSource: String!
        languageTarget: String!
    }

    type Translate {
        fields: String!
        language: String!
    }

    extend type Query {
        translate(input: TranslateInput!): Translate!
    }
`;

export const resolvers: {
    Query: EndpointsTranslate["Query"];
} = {
    ...TranslateEndpoints,
};
