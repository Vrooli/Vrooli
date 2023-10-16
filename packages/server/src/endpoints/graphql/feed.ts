import { PopularObjectType, PopularSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsFeed, FeedEndpoints } from "../logic/feed";

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
        activeFocusModeId: ID # Updates active focus mode when provided
    }

    type HomeResult {
        recommended: [Resource!]! # Not real resources (i.e. pulled from a feed instead of queried from database), but mimics the Resource shape
        reminders: [Reminder!]! # Should only show reminders for the current focus mode which are almost due or overdue
        resources: [Resource!]!
        schedules: [Schedule!]! # Should only show schedules with upcoming or current events
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
