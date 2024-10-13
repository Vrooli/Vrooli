import { TeamSortBy } from "@local/shared";
import { EndpointsTeam, TeamEndpoints } from "../logic/team";

export const typeDef = `#graphql
    enum TeamSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
    }

    input TeamCreateInput {
        id: ID!
        bannerImage: Upload
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean!
        permissions: String
        profileImage: Upload
        memberInvitesCreate: [MemberInviteCreateInput!]
        resourceListCreate: ResourceListCreateInput
        rolesCreate: [RoleCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [TeamTranslationCreateInput!]
    }
    input TeamUpdateInput {
        id: ID!
        bannerImage: Upload
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        permissions: String
        profileImage: Upload
        membersDelete: [ID!]
        memberInvitesDelete: [ID!]
        memberInvitesCreate: [MemberInviteCreateInput!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rolesDelete: [ID!]
        rolesCreate: [RoleCreateInput!]
        rolesUpdate: [RoleUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [TeamTranslationCreateInput!]
        translationsUpdate: [TeamTranslationUpdateInput!]
    }
    type Team {
        id: ID!
        created_at: Date!
        updated_at: Date!
        bannerImage: String
        handle: String
        isOpenToNewMembers: Boolean!
        isPrivate: Boolean!
        bookmarks: Int!
        profileImage: String
        views: Int!
        translatedName: String!
        apis: [Api!]!
        apisCount: Int!
        codes: [Code!]!
        codesCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        forks: [Team!]!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        meetings: [Meeting!]!
        meetingsCount: Int!
        members: [Member!]!
        membersCount: Int!
        notes: [Note!]!
        notesCount: Int!
        parent: Team
        paymentHistory: [Payment!]!
        permissions: String!
        posts: [Post!]!
        postsCount: Int!
        premium: Premium
        projects: [Project!]!
        projectsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        reports: [Report!]!
        reportsCount: Int!
        resourceList: ResourceList
        roles: [Role!]
        rolesCount: Int!
        routines: [Routine!]!
        routinesCount: Int!
        standards: [Standard!]!
        standardsCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsTeam!]!
        tags: [Tag!]!
        transfersIncoming: [Transfer!]!
        transfersOutgoing: [Transfer!]!
        translations: [TeamTranslation!]!
        translationsCount: Int!
        wallets: [Wallet!]!
        you: TeamYou!
    }
        
    type TeamYou {
        canAddMembers: Boolean!
        canDelete: Boolean!
        canBookmark: Boolean!
        canReport: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        isBookmarked: Boolean!
        isViewed: Boolean!
        yourMembership: Member
    }

    input TeamTranslationCreateInput {
        id: ID!
        language: String!
        bio: String
        name: String!
    }
    input TeamTranslationUpdateInput {
        id: ID!
        language: String!
        bio: String
        name: String
    }
    type TeamTranslation {
        id: ID!
        language: String!
        bio: String
        name: String!
    }

    type Member {
        team: Team!
        user: User!
    }

    input TeamSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isOpenToNewMembers: Boolean
        maxBookmarks: Int
        maxViews: Int
        memberUserIds: [ID!]
        minBookmarks: Int
        minViews: Int
        projectId: ID
        reportId: ID
        routineId: ID
        searchString: String
        standardId: ID
        sortBy: TeamSortBy
        tags: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type TeamSearchResult {
        pageInfo: PageInfo!
        edges: [TeamEdge!]!
    }

    type TeamEdge {
        cursor: String!
        node: Team!
    }

    extend type Query {
        team(input: FindByIdOrHandleInput!): Team
        teams(input: TeamSearchInput!): TeamSearchResult!
    }

    extend type Mutation {
        teamCreate(input: TeamCreateInput!): Team!
        teamUpdate(input: TeamUpdateInput!): Team!
    }
`;

export const resolvers: {
    TeamSortBy: typeof TeamSortBy;
    Query: EndpointsTeam["Query"];
    Mutation: EndpointsTeam["Mutation"];
} = {
    TeamSortBy,
    ...TeamEndpoints,
};
