import { gql } from 'apollo-server-express';
import { CODE, ResourceFor, ResourceSortBy } from '@local/shared';
import { CustomError } from '../error';
import { ResourceModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteManyInput, FindByIdInput, ReportInput, Resource, ResourceCountInput, ResourceInput, ResourceSearchInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ResourceFor {
        Organization
        Project
        RoutineContextual
        RoutineExternal
        RoutineDonation
        Actor
    }

    enum ResourceSortBy {
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
    }

    input ResourceInput {
        id: ID
        title: String!
        description: String
        link: String!
        displayUrl: String
        createdFor: ResourceFor!
        forId: ID!
    }

    type Resource {
        id: ID!
        created_at: Date!
        updated_at: Date!
        title: String!
        description: String
        link: String!
        displayUrl: String
        organization_resources: [Organization!]!
        project_resources: [Project!]!
        routine_resources_contextual: [Routine!]!
        routine_resources_external: [Routine!]!
        routine_resources_donation: [Routine!]!
        user_resources: [User!]!
        starredBy: [User!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input ResourceSearchInput {
        forId: ID
        forType: ResourceFor
        ids: [ID!]
        sortBy: ResourceSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type ResourceSearchResult {
        pageInfo: PageInfo!
        edges: [ResourceEdge!]!
    }

    # Return type for search result edge
    type ResourceEdge {
        cursor: String!
        node: Resource!
    }

    # Input for count
    input ResourceCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        resource(input: FindByIdInput!): Resource
        resources(input: ResourceSearchInput!): ResourceSearchResult!
        resourcesCount(input: ResourceCountInput!): Int!
    }

    extend type Mutation {
        resourceAdd(input: ResourceInput!): Resource!
        resourceUpdate(input: ResourceInput!): Resource!
        resourceDeleteMany(input: DeleteManyInput!): Count!
        resourceReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    ResourceFor: ResourceFor,
    ResourceSortBy: ResourceSortBy,
    Query: {
        resource: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource> | null> => {
            // Query database
            const dbModel = await ResourceModel(prisma).findById(input, info);
            // Format data
            return dbModel ? ResourceModel().toGraphQL(dbModel) : null;
        },
        resources: async (_parent: undefined, { input }: IWrap<ResourceSearchInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>[]> => {
            // Create query for specified object
            //const forQuery = (input.forId && input.forType) ? { user: { id: input.userId } } : undefined; TODO
            // return search query
            //return await ResourceModel(prisma).search({...forQuery,}, input, info);
            throw new CustomError(CODE.NotImplemented);
        },
        resourcesCount: async (_parent: undefined, { input }: IWrap<ResourceCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            // Return count query
            return await ResourceModel(prisma).count({}, input);
        },
    },
    Mutation: {
        resourceAdd: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await ResourceModel(prisma).create(input, info);
            // Format object to GraphQL type
            return ResourceModel().toGraphQL(dbModel);
        },
        resourceUpdate: async (_parent: undefined, { input }: IWrap<ResourceInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Update object
            const dbModel = await ResourceModel(prisma).update(input, info);
            // Format to GraphQL type
            return ResourceModel().toGraphQL(dbModel);
        },
        resourceDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).deleteMany(input);
        },
        /**
         * Reports a resource. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         resourceReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: any): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await ResourceModel(prisma).report(input);
            return { success };
        }
    }
}