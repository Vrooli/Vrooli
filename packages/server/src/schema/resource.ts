import { gql } from 'apollo-server-express';
import { CODE, ResourceFor, ResourceSortBy, ResourceUsedFor } from '@local/shared';
import { CustomError } from '../error';
import { ResourceModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteManyInput, FindByIdInput, Resource, ResourceCountInput, ResourceAddInput, ResourceUpdateInput, ResourceSearchInput, ResourceSearchResult } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ResourceFor {
        Organization
        Project
        RoutineContextual
        RoutineExternal
        User
    }

    enum ResourceSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum ResourceUsedFor {
        Community
        Context
        Donation
        Learning
        OfficialWebsite
        Related
        Social
        Tutorial
    }

    input ResourceAddInput {
        createdFor: ResourceFor!
        createdForId: ID!
        description: String
        link: String!
        title: String
        usedFor: ResourceUsedFor
    }
    input ResourceUpdateInput {
        id: ID!
        createdFor: ResourceFor
        createdForId: ID
        description: String
        link: String
        title: String
        usedFor: ResourceUsedFor
    }
    type Resource {
        id: ID!
        created_at: Date!
        updated_at: Date!
        createdFor: ResourceFor!
        createdForId: ID!
        description: String
        link: String!
        title: String!
        usedFor: ResourceUsedFor
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
        resourceAdd(input: ResourceAddInput!): Resource!
        resourceUpdate(input: ResourceUpdateInput!): Resource!
        resourceDeleteMany(input: DeleteManyInput!): Count!
    }
`

export const resolvers = {
    ResourceFor: ResourceFor,
    ResourceSortBy: ResourceSortBy,
    ResourceUsedFor: ResourceUsedFor,
    Query: {
        resource: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource> | null> => {
            const data = await ResourceModel(prisma).findResource(req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        resources: async (_parent: undefined, { input }: IWrap<ResourceSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ResourceSearchResult> => {
            const data = await ResourceModel(prisma).searchResources({}, req.userId, input, info);
            if (!data) throw new CustomError(CODE.ErrorUnknown);
            return data;
        },
        resourcesCount: async (_parent: undefined, { input }: IWrap<ResourceCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return await ResourceModel(prisma).count({}, input);
        },
    },
    Mutation: {
        resourceAdd: async (_parent: undefined, { input }: IWrap<ResourceAddInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await ResourceModel(prisma).addResource(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        resourceUpdate: async (_parent: undefined, { input }: IWrap<ResourceUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await ResourceModel(prisma).updateResource(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        resourceDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await ResourceModel(prisma).deleteResources(req.userId, input);
        },
    }
}