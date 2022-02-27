import { gql } from 'apollo-server-express';
import { MemberRole, OrganizationSortBy } from '@local/shared';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationCreateInput, OrganizationUpdateInput, OrganizationSearchInput, Success, OrganizationSearchResult } from './types';
import { Context } from '../context';
import { countHelper, createHelper, deleteOneHelper, OrganizationModel, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum OrganizationSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
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
        bio: String
        name: String!
        resourcesCreate: [ResourceCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    input OrganizationUpdateInput {
        id: ID!
        bio: String
        name: String
        membersConnect: [ID!]
        membersDisconnect: [ID!]
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    type Organization {
        id: ID!
        created_at: Date!
        updated_at: Date!
        bio: String
        name: String!
        stars: Int!
        isStarred: Boolean!
        role: MemberRole
        comments: [Comment!]!
        members: [Member!]!
        projects: [Project!]!
        reports: [Report!]!
        resources: [Resource!]!
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        wallets: [Wallet!]!
    }

    type Member {
        user: User!
        role: MemberRole!
    }

    input OrganizationSearchInput {
        userId: ID
        projectId: ID
        routineId: ID
        standardId: ID
        reportId: ID
        ids: [ID!]
        sortBy: OrganizationSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
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