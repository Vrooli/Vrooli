import { PopularObjectType, PopularSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsFeed, FeedEndpoints } from "../logic";

export const typeDef = gql`
    enum PopularSortBy {
        BookmarksAsc
        BookmarksDesc
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

    union Popular = Api | Note | Organization | Project | Question | Routine | SmartContract | Standard | User

    input PopularSearchInput {
        apiAfter: String
        createdTimeFrame: TimeFrame
        noteAfter: String
        objectType: PopularObjectType # To limit to a specific type
        organizationAfter: String
        projectAfter: String
        questionAfter: String
        routineAfter: String
        smartContractAfter: String
        sortBy: PopularSortBy
        searchString: String
        standardAfter: String
        take: Int
        updatedTimeFrame: TimeFrame
        userAfter: String
        visibility: VisibilityType
    }

    type PopularSearchResult {
        pageInfo: PopularPageInfo!
        edges: [PopularEdge!]!
    }
    type PopularPageInfo {
        hasNextPage: Boolean!
        endCursorApi: String
        endCursorNote: String
        endCursorOrganization: String
        endCursorProject: String
        endCursorQuestion: String
        endCursorRoutine: String
        endCursorSmartContract: String
        endCursorStandard: String
        endCursorUser: String
    }
    type PopularEdge {
        cursor: String!
        node: Popular!
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
        popular(input: PopularSearchInput!): PopularSearchResult!
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
