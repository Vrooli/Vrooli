import { OrganizationSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsOrganization, OrganizationEndpoints } from "../logic";

export const typeDef = gql`
    enum OrganizationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
    }

    input OrganizationCreateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        permissions: String
        memberInvitesCreate: [MemberInviteCreateInput!]
        resourceListCreate: ResourceListCreateInput
        rolesCreate: [RoleCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [OrganizationTranslationCreateInput!]
    }
    input OrganizationUpdateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        permissions: String
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
        translationsCreate: [OrganizationTranslationCreateInput!]
        translationsUpdate: [OrganizationTranslationUpdateInput!]
    }
    type Organization {
        id: ID!
        created_at: Date!
        updated_at: Date!
        handle: String
        isOpenToNewMembers: Boolean!
        isPrivate: Boolean!
        bookmarks: Int!
        views: Int!
        translatedName: String!
        apis: [Api!]!
        apisCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        forks: [Organization!]!
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
        parent: Organization
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
        smartContracts: [SmartContract!]!
        smartContractsCount: Int!
        standards: [Standard!]!
        standardsCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsOrganization!]!
        tags: [Tag!]!
        transfersIncoming: [Transfer!]!
        transfersOutgoing: [Transfer!]!
        translations: [OrganizationTranslation!]!
        translationsCount: Int!
        wallets: [Wallet!]!
        you: OrganizationYou!
    }
        
    type OrganizationYou {
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

    input OrganizationTranslationCreateInput {
        id: ID!
        language: String!
        bio: String
        name: String!
    }
    input OrganizationTranslationUpdateInput {
        id: ID!
        language: String!
        bio: String
        name: String
    }
    type OrganizationTranslation {
        id: ID!
        language: String!
        bio: String
        name: String!
    }

    type Member {
        user: User!
        organization: Organization!
    }

    input OrganizationSearchInput {
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
        sortBy: OrganizationSortBy
        tags: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type OrganizationSearchResult {
        pageInfo: PageInfo!
        edges: [OrganizationEdge!]!
    }

    type OrganizationEdge {
        cursor: String!
        node: Organization!
    }

    extend type Query {
        organization(input: FindByIdOrHandleInput!): Organization
        organizations(input: OrganizationSearchInput!): OrganizationSearchResult!
    }

    extend type Mutation {
        organizationCreate(input: OrganizationCreateInput!): Organization!
        organizationUpdate(input: OrganizationUpdateInput!): Organization!
    }
`;

export const resolvers: {
    OrganizationSortBy: typeof OrganizationSortBy;
    Query: EndpointsOrganization["Query"];
    Mutation: EndpointsOrganization["Mutation"];
} = {
    OrganizationSortBy,
    ...OrganizationEndpoints,
};