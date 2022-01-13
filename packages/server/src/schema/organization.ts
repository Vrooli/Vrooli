import { gql } from 'apollo-server-express';
import { CODE, MemberRole } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteOneInput, FindByIdInput, Organization, OrganizationCountInput, OrganizationInput, OrganizationSearchInput, OrganizationSearchResult, OrganizationSortBy, ReportInput, Success } from './types';
import { Context } from '../context';
import { OrganizationModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum OrganizationSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    enum MemberRole {
        Admin
        Member
        Owner
    }

    input OrganizationInput {
        id: ID
        name: String!
        description: String
        resources: [ResourceInput!]
    }

    type Organization {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        stars: Int!
        comments: [Comment!]!
        resources: [Resource!]!
        donationResources: [Resource!]!
        projects: [Project!]!
        wallets: [Wallet!]!
        starredBy: [User!]!
        routines: [Routine!]!
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
        organizationReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    OrganizationSortBy: OrganizationSortBy,
    MemberRole: MemberRole,
    Query: {
        organization: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization> | null> => {
            // Query database
            const dbModel = await OrganizationModel(prisma).findById(input, info);
            // Format data
            return dbModel ? OrganizationModel().toGraphQL(dbModel) : null;
        },
        organizations: async (_parent: undefined, { input }: IWrap<OrganizationSearchInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<any> => {
            // Return search query
            return await OrganizationModel(prisma).search({}, input, info);
        },
        organizationsCount: async (_parent: undefined, { input }: IWrap<OrganizationCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await OrganizationModel(prisma).count({}, input);
        },
    },
    Mutation: {
        organizationAdd: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            // Create object
            const dbModel = await OrganizationModel(prisma).create(input, info);
            // Format object to GraphQL type
            return OrganizationModel().toGraphQL(dbModel);
        },
        organizationUpdate: async (_parent: undefined, { input }: IWrap<OrganizationInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Organization>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be updating your own
            // Update object
            const dbModel = await OrganizationModel(prisma).update(input, info);
            // Format to GraphQL type
            return OrganizationModel().toGraphQL(dbModel);
        },
        organizationDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO must be deleting your own
            const success = await OrganizationModel(prisma).delete(input);
            return { success };
        },
        /**
         * Reports an organization. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
        organizationReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await OrganizationModel(prisma).report(input);
            return { success };
        }
    }
}