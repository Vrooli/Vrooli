import { RunStatus, StatPeriodType, VisibilityType } from "@local/shared";
import { gql } from "apollo-server-express";
import { GraphQLScalarType } from "graphql";
import { GraphQLUpload } from "graphql-upload";
import { UnionResolver } from "../../types";
import { EndpointsRoot, RootEndpoints } from "../logic/root";
import { resolveUnion } from "./resolvers";

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    enum GqlModelType {
        Api
        ApiKey
        ApiVersion
        Award
        Bookmark
        BookmarkList
        Chat
        ChatInvite
        ChatMessage
        ChatMessageSearchTreeResult
        ChatParticipant
        Code
        CodeVersion
        Comment
        Copy
        Email
        FocusMode
        FocusModeFilter
        Fork
        Handle
        HomeResult
        Issue
        Label
        Meeting
        MeetingInvite
        Member
        MemberInvite
        Node
        NodeEnd
        NodeLink
        NodeLinkWhen
        NodeLoop
        NodeLoopWhile
        NodeRoutineList
        NodeRoutineListItem
        Note
        NoteVersion
        Notification
        NotificationSubscription
        Payment
        Phone
        PopularResult
        Post
        Premium
        Project
        ProjectVersion
        ProjectVersionContentsSearchResult
        ProjectVersionDirectory
        ProjectOrRoutineSearchResult
        ProjectOrTeamSearchResult
        PullRequest
        PushDevice
        Question
        QuestionAnswer
        Quiz
        QuizAttempt
        QuizQuestion
        QuizQuestionResponse
        Reaction
        ReactionSummary
        Reminder
        ReminderItem
        ReminderList
        Report
        ReportResponse
        ReputationHistory
        Resource
        ResourceList
        Role
        Routine
        RoutineVersion
        RoutineVersionInput
        RoutineVersionOutput
        RunProject
        RunProjectOrRunRoutineSearchResult
        RunProjectStep
        RunRoutine
        RunRoutineInput
        RunRoutineOutput
        RunRoutineStep
        Schedule
        ScheduleException
        ScheduleRecurrence
        Session
        SessionUser
        Standard
        StandardVersion
        StatsApi
        StatsCode
        StatsProject
        StatsQuiz
        StatsRoutine
        StatsSite
        StatsStandard
        StatsTeam
        StatsUser
        Tag
        Team
        Transfer
        User
        View
        Wallet
    }

    enum VisibilityType {
        All
        Own
        Public
        Private
    }

    scalar Date
    scalar Upload
    scalar JSON

    # Used for Projects, Standards, and Routines, since they can be owned
    # by either a User or an Team.
    union Owner = User | Team

    # Used for filtering by date created/updated, as well as fetching metrics (e.g. monthly active users)
    input TimeFrame {
        after: Date
        before: Date
    }

    # Return type for a cursor-based pagination's pageInfo response
    type PageInfo {
        hasNextPage: Boolean!
        endCursor: String
    }
    # Return type for delete mutations,
    # which return the number of affected rows
    type Count {
        count: Int!
    }
    # Return type for mutations with a success boolean
    # Could return just the boolean, but this makes it clear what the result means
    type Success {
        success: Boolean!
    }
    # Return type for error messages
    type Response {
        code: Int
        message: String!
    }

    type VersionYou {
        canComment: Boolean!
        canCopy: Boolean!
        canDelete: Boolean!
        canReport: Boolean!
        canUpdate: Boolean!
        canUse: Boolean! # In run, project, etc. Not always applicable
        canRead: Boolean!
    }

    enum RunStatus {
        Scheduled
        InProgress
        Completed
        Failed
        Cancelled
    }

    enum StatPeriodType {
        Hourly
        Daily
        Weekly
        Monthly
        Yearly
    }

    input ReadAssetsInput {
        files: [String!]!
    }

    input WriteAssetsInput {
        files: [Upload!]!
    }

    # Input for finding object by its ID
    input FindByIdInput {
        id: ID!
    }

    # Input for finding object by its ID or handle
    input FindByIdOrHandleInput {
        id: ID
        handle: String
    }

    # Input for finding a versioned object
    input FindVersionInput {
        id: ID
        idRoot: ID # If using root, finds the latest public version
        handleRoot: String # Not always applicable
    }

    # Input for an exception to a search query parameter
    input SearchException {
        field: String!
        value: String! # JSON string
    }

    # Base query. Must contain something,
    # which can be as simple as '_empty: String'
    type Query {
        _empty: String
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        _empty: String
    }
`;

export const resolvers: {
    RunStatus: typeof RunStatus;
    StatPeriodType: typeof StatPeriodType;
    VisibilityType: typeof VisibilityType;
    Upload: typeof GraphQLUpload;
    Date: GraphQLScalarType;
    JSON: GraphQLScalarType;
    Owner: UnionResolver;
    Query: EndpointsRoot["Query"];
    Mutation: EndpointsRoot["Mutation"];
} = {
    RunStatus,
    StatPeriodType,
    VisibilityType,
    Upload: GraphQLUpload,
    Date: new GraphQLScalarType({
        name: "Date",
        description: "Custom description for the date scalar",
        // Assumes data is either Unix timestamp or Date object
        parseValue(value) {
            return new Date(value).toISOString(); // value from the client
        },
        serialize(value) {
            return new Date(value).getTime(); // value sent to the client
        },
        parseLiteral(ast: any) {
            return new Date(ast).toDateString(); // ast value is always in string format
        },
    }),
    JSON: new GraphQLScalarType({
        name: "JSONObject",
        description: "Arbitrary JSON object",
        // Serialize data sent to the client
        serialize(value) {
            // Assuming value is already a JSON object
            return value; // No transformation needed
        },
        // Parse data from the client
        parseValue(value) {
            return typeof value === "string" ? JSON.parse(value) : value; // Convert JSON string to object
        },
        parseLiteral(ast: any) {
            try {
                // Assuming ast.value is a JSON string for simplicity; you might need to handle different AST types
                return JSON.parse(ast.value);
            } catch (error) {
                throw new Error(`JSONObject cannot represent non-JSON value: ${ast.value}`);
            }
        },
    }),
    Owner: { __resolveType(obj) { return resolveUnion(obj); } },
    ...RootEndpoints,
};
