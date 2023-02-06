import { gql } from 'apollo-server-express';
import { GraphQLScalarType } from "graphql";
import { GraphQLUpload } from 'graphql-upload';
import { readFiles, saveFiles } from '../utils';
// import ogs from 'open-graph-scraper';
import { rateLimit } from '../middleware';
import { CustomError } from '../events/error';
import { resolveUnion } from './resolvers';
import { ReadAssetsInput, RunStatus, StatPeriodType, VisibilityType, WriteAssetsInput } from '@shared/consts';
import { GQLEndpoint, UnionResolver } from '../types';

// Defines common inputs, outputs, and types for all GraphQL queries and mutations.
export const typeDef = gql`
    enum GqlModelType {
        Api
        ApiKey
        ApiVersion
        Comment
        Copy
        DevelopResult
        Email
        Fork
        Handle
        HistoryResult
        Issue
        Label
        LearnResult
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
        Reminder
        ReminderItem
        ReminderList
        Report
        ReportResponse
        ReputationHistory
        ResearchResult
        Resource
        ResourceList
        Role
        Routine
        RoutineVersion
        RoutineVersionInput
        RoutineVersionOutput
        RunProject
        RunProjectOrRunRoutineSearchResult
        RunProjectSchedule
        RunProjectStep
        RunRoutine
        RunRoutineInput
        RunRoutineSchedule
        RunRoutineStep
        Session
        SessionUser
        SmartContract
        SmartContractVersion
        Standard
        StandardVersion
        Star
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
        UserSchedule
        UserScheduleFilter
        View
        Vote
        Wallet
    }

    enum VisibilityType {
        All
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
        # _empty: String
        readAssets(input: ReadAssetsInput!): [String]!
    }
    # Base mutation. Must contain something,
    # which can be as simple as '_empty: String'
    type Mutation {
        # _empty: String
        writeAssets(input: WriteAssetsInput!): Boolean
    }
`

export const resolvers: {
    RunStatus: typeof RunStatus;
    StatPeriodType: typeof StatPeriodType;
    VisibilityType: typeof VisibilityType;
    Upload: typeof GraphQLUpload;
    Date: GraphQLScalarType;
    Owner: UnionResolver;
    Query: {
        readAssets: GQLEndpoint<ReadAssetsInput, (string | null)[]>;
    },
    Mutation: {
        writeAssets: GQLEndpoint<WriteAssetsInput, boolean>;
    }
} = {
    RunStatus,
    StatPeriodType,
    VisibilityType: VisibilityType,
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
        }
    }),
    Owner: { __resolveType(obj) { return resolveUnion(obj) } },
    Query: {
        readAssets: async (_parent, { input }, { req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return await readFiles(input.files);
        },
    },
    Mutation: {
        writeAssets: async (_parent, { input }, { req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            throw new CustomError('0327', 'NotImplemented', req.languages); // TODO add safety checks before allowing uploads
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null)
        },
    }
}