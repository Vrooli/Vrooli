import { PopularObjectType, PopularSortBy } from "@local/shared";
import { EndpointsFeed, FeedEndpoints } from "../logic/feed";

export const typeDef = `#graphql
    enum PopularSortBy {
        BookmarksAsc
        BookmarksDesc
        ViewsAsc
        ViewsDesc
    }

    enum PopularObjectType {
        Api
        Code
        Note
        Project
        Question
        Routine
        Standard
        Team
        User
    }  

    union Popular = Api | Code | Note | Project | Question | Routine | Standard | Team | User

    input PopularSearchInput {
        apiAfter: String
        codeAfter: String
        createdTimeFrame: TimeFrame
        noteAfter: String
        objectType: PopularObjectType # To limit to a specific type
        projectAfter: String
        questionAfter: String
        routineAfter: String
        sortBy: PopularSortBy
        searchString: String
        standardAfter: String
        take: Int
        teamAfter: String
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
        endCursorCode: String
        endCursorNote: String
        endCursorProject: String
        endCursorQuestion: String
        endCursorRoutine: String
        endCursorStandard: String
        endCursorTeam: String
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
