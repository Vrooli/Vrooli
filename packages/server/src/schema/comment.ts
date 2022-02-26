import { gql } from 'apollo-server-express';
import { CommentFor } from '@local/shared';
import { Comment, CommentCreateInput, CommentUpdateInput, DeleteOneInput, Success } from './types';
import { IWrap, RecursivePartial } from 'types';
import { CommentModel, createHelper, deleteOneHelper, updateHelper } from '../models';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum CommentFor {
        Organization
        Project
        Routine
        Standard
        User
    }   

    input CommentCreateInput {
        text: String!
        createdFor: CommentFor!
        forId: ID!
    }
    input CommentUpdateInput {
        id: ID!
        text: String
    }

    union CommentedOn = Project | Routine | Standard

    type Comment {
        id: ID!
        text: String
        created_at: Date!
        updated_at: Date!
        creator: Contributor
        commentedOn: CommentedOn!
        reports: [Report!]!
        stars: Int
        isStarred: Boolean!
        starredBy: [User!]
        score: Int
        isUpvoted: Boolean
        role: MemberRole
    }

    extend type Mutation {
        commentCreate(input: CommentCreateInput!): Comment!
        commentUpdate(input: CommentUpdateInput!): Comment!
        commentDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    CommentFor: CommentFor,
    CommentedOn: {
        __resolveType(obj: any) {
            console.log('IN COMMENT __resolveType', obj);
            // Only a Project has a name field
            if (obj.hasOwnProperty('name')) return 'Project';
            // Only a Routine has a title field
            if (obj.hasOwnProperty('title')) return 'Routine';
            // Only a Standard has an isFile field
            if (obj.hasOwnProperty('isFile')) return 'Standard';
            return null; // GraphQLError is thrown
        },
    },
    Mutation: {
        commentCreate: async (_parent: undefined, { input }: IWrap<CommentCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            return createHelper(req.userId, input, info, CommentModel(prisma).cud);
        },
        commentUpdate: async (_parent: undefined, { input }: IWrap<CommentUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Comment>> => {
            return updateHelper(req.userId, input, info, CommentModel(prisma).cud);
        },
        commentDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, CommentModel(prisma).cud);
        },
    }
}