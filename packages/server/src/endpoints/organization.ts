import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdOrHandleInput, Organization, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum OrganizationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    input OrganizationCreateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        resourceListCreate: ResourceListCreateInput
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [OrganizationTranslationCreateInput!]
        roles: [RoleCreateInput!]
        memberInvites: [MemberInviteCreateInput!]
    }
    input OrganizationUpdateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        membersDisconnect: [ID!]
        memberInvitesDelete: [ID!]
        memberInvitesCreate: [MemberInviteCreateInput!]
        resourceListUpdate: ResourceListUpdateInput
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [OrganizationTranslationCreateInput!]
        translationsUpdate: [OrganizationTranslationUpdateInput!]
        rolesDelete: [ID!]
        rolesCreate: [RoleCreateInput!]
        rolesUpdate: [RoleUpdateInput!]
    }
    type Organization {
        id: ID!
        created_at: Date!
        updated_at: Date!
        handle: String
        isOpenToNewMembers: Boolean!
        isPrivate: Boolean!
        stars: Int!
        views: Int!
        isStarred: Boolean!
        isViewed: Boolean!
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
        permissionsOrganization: OrganizationPermission
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
        starredBy: [User!]!
        stats: [StatsOrganization!]!
        tags: [Tag!]!
        transfersIncoming: [Transfer!]!
        transfersOutgoing: [Transfer!]!
        translations: [OrganizationTranslation!]!
        translationsCount: Int!
        wallets: [Wallet!]!
    }

    # Will beef this up later
    type OrganizationPermission {
        canAddMembers: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        isMember: Boolean!
    }

    input OrganizationTranslationCreateInput {
        id: ID!
        language: String!
        bio: String
        name: String!
    }
    input OrganizationTranslationUpdateInput {
        id: ID!
        language: String
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
        languages: [String!]
        minStars: Int
        minViews: Int
        projectId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        routineId: ID
        searchString: String
        standardId: ID
        sortBy: OrganizationSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
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
`

const objectType = 'Organization';
export const resolvers: {
    OrganizationSortBy: typeof OrganizationSortBy;
    Query: {
        organization: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<Organization>>;
        organizations: GQLEndpoint<OrganizationSearchInput, FindManyResult<Organization>>;
    },
    Mutation: {
        organizationCreate: GQLEndpoint<OrganizationCreateInput, CreateOneResult<Organization>>;
        organizationUpdate: GQLEndpoint<OrganizationUpdateInput, UpdateOneResult<Organization>>;
    }
} = {
    OrganizationSortBy: OrganizationSortBy,
    Query: {
        organization: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        organizations: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        organizationCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        organizationUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}