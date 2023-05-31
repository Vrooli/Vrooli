import { PopularObjectType, PopularSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsFeed, FeedEndpoints } from "../logic";

export const typeDef = gql`
    enum PopularSortBy {
        StarsAsc
        StarsDesc
        ViewsAsc
        ViewsDesc
    }

    enum PopularObjectType {
        Api
        Note
        Organization
        Project
        Question
        Routine
        SmartContract
        Standard
        User
    }  

    input PopularInput {
        objectType: PopularObjectType # To limit to a specific type
        searchString: String!
        sortBy: PopularSortBy
        take: Int
    }
 
    type PopularResult {
        apis: [Api!]!
        notes: [Note!]!
        organizations: [Organization!]!
        projects: [Project!]!
        questions: [Question!]!
        routines: [Routine!]!
        smartContracts: [SmartContract!]!
        standards: [Standard!]!
        users: [User!]!
    }

    input HomeInput {
        searchString: String!
        take: Int
    }

    type HomeResult {
        notes: [Note!]!
        reminders: [Reminder!]!
        resources: [Resource!]!
        schedules: [Schedule!]!
    }

    type Query {
        home(input: HomeInput!): HomeResult!
        popular(input: PopularInput!): PopularResult!
    }
 `;

export const resolvers: {
    PopularSortBy: typeof PopularSortBy;
    PopularObjectType: typeof PopularObjectType;
    Query: EndpointsFeed["Query"];
} = {
    PopularSortBy,
    PopularObjectType,
    ...FeedEndpoints,
};
