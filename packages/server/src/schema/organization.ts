import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { FindByIdOrHandleInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSearchResult, OrganizationSortBy } from './types';
import { Context } from '../context';
import { countHelper, createHelper, OrganizationModel, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

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
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [OrganizationTranslationCreateInput!]
        roles: [RoleCreateInput!]
    }
    input OrganizationUpdateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        isPrivate: Boolean
        membersConnect: [ID!]
        membersDisconnect: [ID!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
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
        comments: [Comment!]!
        commentsCount: Int!
        members: [Member!]!
        membersCount: Int!
        permissionsOrganization: OrganizationPermission
        projects: [Project!]!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        roles: [Role!]
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [OrganizationTranslation!]!
        wallets: [Wallet!]!
    }

    # Will beef this up later
    type OrganizationPermission {
        canAddMembers: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
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
    }

    # Return type for search result
    type OrganizationSearchResult {
        pageInfo: PageInfo!
        edges: [OrganizationEdge!]!
    }

    # Return type for search result edge
    type OrganizationEdge {
        cursor: String!
        node: Organization!
    }

    # Input for count
    input OrganizationCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        organization(input: FindByIdOrHandleInput!): Organization
        organizations(input: OrganizationSearchInput!): OrganizationSearchResult!
        organizationsCount(input: OrganizationCountInput!): Int!
    }

    extend type Mutation {
        organizationCreate(input: OrganizationCreateInput!): Organization!
        organizationUpdate(input: OrganizationUpdateInput!): Organization!
    }
`

export const resolvers = {
    OrganizationSortBy: OrganizationSortBy,
    Query: {
        organization: async (_parent: undefined, { input }: IWrap<FindByIdOrHandleInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
        },
        organizations: async (_parent: undefined, { input }: IWrap<OrganizationSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<OrganizationSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
        },
        organizationsCount: async (_parent: undefined, { input }: IWrap<OrganizationCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, OrganizationModel(context.prisma));
        },
    },
    Mutation: {
        organizationCreate: async (_parent: undefined, { input }: IWrap<OrganizationCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            await rateLimit({ context, info, max: 100, byAccount: true });
            return createHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
        },
        organizationUpdate: async (_parent: undefined, { input }: IWrap<OrganizationUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            await rateLimit({ context, info, max: 250, byAccount: true });
            return updateHelper(context.req.userId, input, info, OrganizationModel(context.prisma));
        },
    }
}