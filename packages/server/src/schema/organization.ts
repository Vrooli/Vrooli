import { gql } from 'apollo-server-express';
import { CODE, MemberRole, OrganizationSortBy } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationInput, OrganizationSearchInput, Success } from './types';
import { Context } from '../context';
import { organizationFormatter, OrganizationModel } from '../models';
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

    input OrganizationInput {
        id: ID
        name: String!
        bio: String
        resources: [ResourceInput!]
    }

    type Organization {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        bio: String
        stars: Int!
        isStarred: Boolean
        isAdmin: Boolean!
        comments: [Comment!]!
        resources: [Resource!]!
        projects: [Project!]!
        wallets: [Wallet!]!
        starredBy: [User!]!
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        tags: [Tag!]!
        reports: [Report!]!
        members: [Member!]!
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
        organizationAdd(input: OrganizationInput!): Organization!
        organizationUpdate(input: OrganizationInput!): Organization!
        organizationDeleteOne(input: DeleteOneInput): Success!
    }
`

export const resolvers = {
    OrganizationSortBy: OrganizationSortBy,
    MemberRole: MemberRole,
    Query: {
        organization: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization> | null> => {
            const data = await OrganizationModel(prisma).findOrganization(req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        organizations: async (_parent: undefined, { input }: IWrap<OrganizationSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<any> => {
            const data = await OrganizationModel(prisma).searchOrganizations({}, req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        organizationsCount: async (_parent: undefined, { input }: IWrap<OrganizationCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return await OrganizationModel(prisma).count({}, input);
        },
    },
    Mutation: {
        organizationAdd: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await OrganizationModel(prisma).addOrganization(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        organizationUpdate: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await OrganizationModel(prisma).updateOrganization(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        organizationDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await OrganizationModel(prisma).deleteOrganization(req.userId, input);
        },
    }
}