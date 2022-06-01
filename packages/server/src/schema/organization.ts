import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { FindByIdOrHandleInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, OrganizationSearchResult, OrganizationSortBy, MemberRole } from './types';
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

    enum MemberRole {
        Admin
        Member
        Owner
    }

    input OrganizationCreateInput {
        isOpenToNewMembers: Boolean
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [OrganizationTranslationCreateInput!]
    }
    input OrganizationUpdateInput {
        id: ID!
        handle: String
        isOpenToNewMembers: Boolean
        membersConnect: [ID!]
        membersDisconnect: [ID!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
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
        stars: Int!
        views: Int!
        isStarred: Boolean!
        isViewed: Boolean!
        role: MemberRole
        comments: [Comment!]!
        members: [Member!]!
        projects: [Project!]!
        reports: [Report!]!
        resourceLists: [ResourceList!]!
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [OrganizationTranslation!]!
        wallets: [Wallet!]!
    }

    input OrganizationTranslationCreateInput {
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
        role: MemberRole!
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
    MemberRole: MemberRole,
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