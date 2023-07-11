import { UserSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsUser, UserEndpoints } from "../logic";

export const typeDef = gql`
    enum UserSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
    }

    type User {
        id: ID!
        created_at: Date!
        updated_at: Date
        botSettings: String
        handle: String
        isBot: Boolean!
        isPrivate: Boolean!
        isPrivateApis: Boolean!
        isPrivateApisCreated: Boolean!
        isPrivateMemberships: Boolean!
        isPrivateOrganizationsCreated: Boolean!
        isPrivateProjects: Boolean!
        isPrivateProjectsCreated: Boolean!
        isPrivatePullRequests: Boolean!
        isPrivateQuestionsAnswered: Boolean!
        isPrivateQuestionsAsked: Boolean!
        isPrivateQuizzesCreated: Boolean!
        isPrivateRoles: Boolean!
        isPrivateRoutines: Boolean!
        isPrivateRoutinesCreated: Boolean!
        isPrivateSmartContracts: Boolean!
        isPrivateStandards: Boolean!
        isPrivateStandardsCreated: Boolean!
        isPrivateBookmarks: Boolean!
        isPrivateVotes: Boolean!
        name: String!
        theme: String
        status: AccountStatus
        bookmarks: Int!
        views: Int!
        apiKeys: [ApiKey!]
        apis: [Api!]!
        apisCount: Int!
        apisCreated: [Api!]
        awards: [Award!]
        comments: [Comment!]
        emails: [Email!]
        organizationsCreate: [Organization!]
        invitedByUser: User
        invitedUsers: [User!]
        issuesCreated: [Issue!]
        issuesClosed: [Issue!]
        labels: [Label!]
        translationLanguages: [String!]
        meetingsAttending: [Meeting!]
        meetingsInvited: [MeetingInvite!]
        memberships: [Member!]
        membershipsCount: Int!
        membershipsInvited: [MemberInvite!]
        notesCreated: [Note!]
        notes: [Note!]
        notesCount: Int!
        notifications: [Notification!]
        notificationSubscriptions: [NotificationSubscription!]
        notificationSettings: String
        paymentHistory: [Payment!]
        premium: Premium
        projects: [Project!]
        projectsCount: Int!
        projectsCreated: [Project!]
        pullRequests: [PullRequest!]
        pushDevices: [PushDevice!]
        questionsAnswered: [QuestionAnswer!]
        questionsAsked: [Question!]
        questionsAskedCount: Int!
        quizzesCreated: [Quiz!]
        quizzesTaken: [Quiz!]
        sentReports: [Report!]
        reportsCreated: [Report!]
        reportsReceived: [Report!]!
        reportsReceivedCount: Int!
        reportResponses: [ReportResponse!]
        reputationHistory: [ReputationHistory!]
        roles: [Role!]
        routines: [Routine!]
        routinesCount: Int!
        routinesCreated: [Routine!]
        runProjects: [RunProject!]
        runRoutines: [RunRoutine!]
        focusModes: [FocusMode!]
        smartContracts: [SmartContract!]
        smartContractsCount: Int!
        smartContractsCreated: [SmartContract!]
        standards: [Standard!]
        standardsCount: Int!
        standardsCreated: [Standard!]
        bookmarkedBy: [User!]!
        bookmarked: [Bookmark!]
        tags: [Tag!]
        transfersIncoming: [Transfer!]
        transfersOutgoing: [Transfer!]
        translations: [UserTranslation!]!
        viewed: [View!]
        viewedBy: [View!]
        reacted: [Reaction!]
        wallets: [Wallet!]
        you: UserYou!
    }

    type UserYou {
        canDelete: Boolean!
        canReport: Boolean!
        isBookmarked: Boolean!
        canUpdate: Boolean!
        isViewed: Boolean!
    }

    input UserTranslationCreateInput {
        id: ID!
        language: String!
        bio: String
    }
    input UserTranslationUpdateInput {
        id: ID!
        language: String!
        bio: String
    }
    type UserTranslation {
        id: ID!
        language: String!
        bio: String
    }

    input BotCreateInput {
        id: ID!
        botSettings: String!
        isPrivate: Boolean
        name: String!
        translationsCreate: [UserTranslationCreateInput!]
    }

    input BotUpdateInput {
        id: ID!
        botSettings: String
        isPrivate: Boolean
        name: String
        translationsDelete: [ID!]
        translationsCreate: [UserTranslationCreateInput!]
        translationsUpdate: [UserTranslationUpdateInput!]
    }

    input ProfileUpdateInput {
        handle: String
        name: String
        theme: String
        isPrivate: Boolean
        isPrivateApis: Boolean
        isPrivateApisCreated: Boolean
        isPrivateMemberships: Boolean
        isPrivateOrganizationsCreated: Boolean
        isPrivateProjects: Boolean
        isPrivateProjectsCreated: Boolean
        isPrivatePullRequests: Boolean
        isPrivateQuestionsAnswered: Boolean
        isPrivateQuestionsAsked: Boolean
        isPrivateQuizzesCreated: Boolean
        isPrivateRoles: Boolean
        isPrivateRoutines: Boolean
        isPrivateRoutinesCreated: Boolean
        isPrivateSmartContracts: Boolean
        isPrivateStandards: Boolean
        isPrivateStandardsCreated: Boolean
        isPrivateBookmarks: Boolean
        isPrivateVotes: Boolean
        notificationSettings: String
        languages: [String!]
        focusModesDelete: [ID!]
        focusModesCreate: [FocusModeCreateInput!]
        focusModesUpdate: [FocusModeUpdateInput!]
        translationsDelete: [ID!]
        translationsCreate: [UserTranslationCreateInput!]
        translationsUpdate: [UserTranslationUpdateInput!]
    }

    input ProfileEmailUpdateInput {
        emailsCreate: [EmailCreateInput!]
        emailsDelete: [ID!]
        currentPassword: String!
        newPassword: String
    }

    input UserDeleteInput {
        password: String!
        deletePublicData: Boolean!
    }

    input ImportCalendarInput {
        file: Upload!
    }

    input UserSearchInput {
        maxBookmarks: Int
        maxViews: Int
        memberInOrganizationId: ID
        minBookmarks: Int
        minViews: Int
        ids: [ID!]
        isBot: Boolean
        sortBy: UserSortBy
        searchString: String
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        after: String
        take: Int
        translationLanguages: [String!]
    }

    type UserSearchResult {
        pageInfo: PageInfo!
        edges: [UserEdge!]!
    }

    type UserEdge {
        cursor: String!
        node: User!
    }

    extend type Query {
        profile: User!
        user(input: FindByIdOrHandleInput!): User
        users(input: UserSearchInput!): UserSearchResult!
    }

    extend type Mutation {
        botCreate(input: BotCreateInput!): User!
        botUpdate(input: BotUpdateInput!): User!
        profileUpdate(input: ProfileUpdateInput!): User!
        profileEmailUpdate(input: ProfileEmailUpdateInput!): User!
        userDeleteOne(input: UserDeleteInput!): Success!
        importCalendar(input: ImportCalendarInput!): Success!
        # importUserData(input: ImportUserDataInput!): Success!
        exportCalendar: String!
        exportData: String!
    }
`;

export const resolvers: {
    UserSortBy: typeof UserSortBy;
    Query: EndpointsUser["Query"];
    Mutation: EndpointsUser["Mutation"];
} = {
    UserSortBy,
    ...UserEndpoints,
};
