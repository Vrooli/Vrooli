import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, Success, OrganizationSearchResult, OrganizationSortBy, MemberRole } from './types';
import { Context } from '../context';
import { countHelper, createHelper, deleteOneHelper, OrganizationModel, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';

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
        isStarred: Boolean!
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
        organization(input: FindByIdInput!): Organization
        organizations(input: OrganizationSearchInput!): OrganizationSearchResult!
        organizationsCount(input: OrganizationCountInput!): Int!
    }

    extend type Mutation {
        organizationCreate(input: OrganizationCreateInput!): Organization!
        organizationUpdate(input: OrganizationUpdateInput!): Organization!
        organizationDeleteOne(input: DeleteOneInput): Success!
    }
`

export const resolvers = {
    OrganizationSortBy: OrganizationSortBy,
    MemberRole: MemberRole,
    Query: {
        organization: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization> | null> => {
            return readOneHelper(req.userId, input, info, OrganizationModel(prisma));
        },
        organizations: async (_parent: undefined, { input }: IWrap<OrganizationSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<OrganizationSearchResult> => {
            return readManyHelper(req.userId, input, info, OrganizationModel(prisma));
        },
        organizationsCount: async (_parent: undefined, { input }: IWrap<OrganizationCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return countHelper(input, OrganizationModel(prisma));
        },
    },
    Mutation: {
        organizationCreate: async (_parent: undefined, { input }: IWrap<OrganizationCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            return createHelper(req.userId, input, info, OrganizationModel(prisma));
        },
        organizationUpdate: async (_parent: undefined, { input }: IWrap<OrganizationUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            return updateHelper(req.userId, input, info, OrganizationModel(prisma));
        },
        organizationDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, OrganizationModel(prisma));
        },
    }
}