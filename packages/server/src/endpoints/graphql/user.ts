import { UserSortBy } from "@local/shared";
import { EndpointsUser, UserEndpoints } from "../logic/user";

export const typeDef = `#graphql
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
        bannerImage: String
        botSettings: String
        handle: String
        isBot: Boolean!
        isBotDepictingPerson: Boolean!
        isPrivate: Boolean!
        isPrivateApis: Boolean!
        isPrivateApisCreated: Boolean!
        isPrivateBookmarks: Boolean!
        isPrivateCodes: Boolean!
        isPrivateCodesCreated: Boolean!
        isPrivateMemberships: Boolean!
        isPrivateProjects: Boolean!
        isPrivateProjectsCreated: Boolean!
        isPrivatePullRequests: Boolean!
        isPrivateQuestionsAnswered: Boolean!
        isPrivateQuestionsAsked: Boolean!
        isPrivateQuizzesCreated: Boolean!
        isPrivateRoles: Boolean!
        isPrivateRoutines: Boolean!
        isPrivateRoutinesCreated: Boolean!
        isPrivateStandards: Boolean!
        isPrivateStandardsCreated: Boolean!
        isPrivateTeamsCreated: Boolean!
        isPrivateVotes: Boolean!
        name: String!
        profileImage: String
        theme: String
        status: AccountStatus
        bookmarks: Int!
        views: Int!
        apiKeys: [ApiKey!]
        apis: [Api!]!
        apisCount: Int!
        apisCreated: [Api!]
        awards: [Award!]
        codes: [Code!]
        codesCount: Int!
        codesCreated: [Code!]
        comments: [Comment!]
        emails: [Email!]
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
        phones: [Phone!]
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
        standards: [Standard!]
        standardsCount: Int!
        standardsCreated: [Standard!]
        bookmarkedBy: [User!]!
        bookmarked: [Bookmark!]
        tags: [Tag!]
        teamsCreated: [Team!]
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
        bannerImage: Upload
        botSettings: String!
        handle: String
        isBotDepictingPerson: Boolean!
        isPrivate: Boolean!
        name: String!
        profileImage: Upload
        translationsCreate: [UserTranslationCreateInput!]
    }

    input BotUpdateInput {
        id: ID!
        bannerImage: Upload
        botSettings: String
        handle: String
        isBotDepictingPerson: Boolean
        isPrivate: Boolean
        name: String
        profileImage: Upload
        translationsDelete: [ID!]
        translationsCreate: [UserTranslationCreateInput!]
        translationsUpdate: [UserTranslationUpdateInput!]
    }

    input ProfileUpdateInput {
        bannerImage: Upload
        handle: String
        name: String
        profileImage: Upload
        theme: String
        isPrivate: Boolean
        isPrivateApis: Boolean
        isPrivateApisCreated: Boolean
        isPrivateBookmarks: Boolean
        isPrivateCodes: Boolean
        isPrivateCodesCreated: Boolean
        isPrivateMemberships: Boolean
        isPrivateProjects: Boolean
        isPrivateProjectsCreated: Boolean
        isPrivatePullRequests: Boolean
        isPrivateQuestionsAnswered: Boolean
        isPrivateQuestionsAsked: Boolean
        isPrivateQuizzesCreated: Boolean
        isPrivateRoles: Boolean
        isPrivateRoutines: Boolean
        isPrivateRoutinesCreated: Boolean
        isPrivateStandards: Boolean
        isPrivateStandardsCreated: Boolean
        isPrivateTeamsCreated: Boolean
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
        excludeIds: [ID!]
        maxBookmarks: Int
        maxViews: Int
        memberInTeamId: ID
        minBookmarks: Int
        minViews: Int
        notInChatId: ID
        notInvitedToTeamId: ID
        notMemberInTeamId: ID
        ids: [ID!]
        isBot: Boolean
        isBotDepictingPerson: Boolean
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
        userDeleteOne(input: UserDeleteInput!): Session!
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
