import { gql } from 'apollo-server-express';
import { ProjectSortBy } from '@local/shared';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, FindByIdInput, Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, Success, ProjectCountInput, ProjectSearchResult } from './types';
import { Context } from '../context';
import { createHelper, deleteOneHelper, ProjectModel, projectSearcher, readManyHelper, readOneHelper, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum ProjectSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input ProjectCreateInput {
        description: String
        name: String!
        parentId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        resourcesCreate: [ResourceCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    input ProjectUpdateInput {
        id: ID!
        description: String
        name: String
        parentId: ID
        userId: ID
        organizationId: ID
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
    }
    type Project {
        id: ID!
        created_at: Date!
        updated_at: Date!
        description: String
        name: String!
        stars: Int!
        isStarred: Boolean!
        role: MemberRole
        isUpvoted: Boolean
        score: Int!
        comments: [Comment!]!
        creator: Contributor
        forks: [Project!]!
        owner: Contributor
        parent: Project
        reports: [Report!]!
        resources: [Resource!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        wallets: [Wallet!]
    }

    input ProjectSearchInput {
        userId: ID
        organizationId: ID
        parentId: ID
        reportId: ID
        ids: [ID!]
        sortBy: ProjectSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type ProjectSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectEdge!]!
    }

    # Return type for search result edge
    type ProjectEdge {
        cursor: String!
        node: Project!
    }

    # Input for count
    input ProjectCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        project(input: FindByIdInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
        projectsCount(input: ProjectCountInput!): Int!
    }

    extend type Mutation {
        projectCreate(input: ProjectCreateInput!): Project!
        projectUpdate(input: ProjectUpdateInput!): Project!
        projectDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    ProjectSortBy: ProjectSortBy,
    Query: {
        project: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project> | null> => {
            return readOneHelper(req.userId, input, info, prisma);
        },
        projects: async (_parent: undefined, { input }: IWrap<ProjectSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ProjectSearchResult> => {
            return readManyHelper(req.userId, input, info, prisma, projectSearcher());
        },
        projectsCount: async (_parent: undefined, { input }: IWrap<ProjectCountInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<number> => {
            return await ProjectModel(prisma).count({}, input);
        },
    },
    Mutation: {
        projectCreate: async (_parent: undefined, { input }: IWrap<ProjectCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            return createHelper(req.userId, input, info, ProjectModel(prisma).cud);
        },
        projectUpdate: async (_parent: undefined, { input }: IWrap<ProjectUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Project>> => {
            return updateHelper(req.userId, input, info, ProjectModel(prisma).cud);
        },
        projectDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, ProjectModel(prisma).cud);
        },
    }
}