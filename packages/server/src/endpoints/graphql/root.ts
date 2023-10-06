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
        ChatParticipant
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
        Organization
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
        ProjectOrOrganizationSearchResult
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
        RunRoutineStep
        Schedule
        ScheduleException
        ScheduleRecurrence
        Session
        SessionUser
        SmartContract
        SmartContractVersion
        Standard
        StandardVersion
        StatsApi
        StatsOrganization
        StatsProject
        StatsQuiz
        StatsRoutine
        StatsSite
        StatsSmartContract
        StatsStandard
        StatsUser
        Tag
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

    # Used for Projects, Standards, and Routines, since they can be owned
    # by either a User or an Organization.
    union Owner = User | Organization

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

    # Input for deleting multiple objects
    input DeleteManyInput {
        ids: [ID!]!
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
    Owner: { __resolveType(obj) { return resolveUnion(obj); } },
    ...RootEndpoints,
};
