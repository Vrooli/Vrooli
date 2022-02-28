import { gql } from 'apollo-server-express';
import { CODE, VoteFor } from '@local/shared';
import { CustomError } from '../error';
import { VoteInput, Success } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { GraphQLModelType, VoteModel } from '../models';

export const typeDef = gql`
    enum VoteFor {
        Comment
        Project
        Routine
        Standard
        Tag
    }   

    union VoteTo = Comment | Project | Routine | Standard | Tag

    input VoteInput {
        isUpvote: Boolean
        voteFor: VoteFor!
        forId: ID!
    }
    type Vote {
        isUpvote: Boolean
        from: User!
        to: VoteTo!
    }

    extend type Mutation {
        vote(input: VoteInput!): Success!
    }
`

export const resolvers = {
    VoteFor: VoteFor,
    Vote: {
        __resolveType(obj: any) {
            console.log('IN VOTE __resolveType', obj);
            // Only a Standard has an isFile field
            if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
            // Only a Project has a name and description field
            if (obj.hasOwnProperty('name')) return GraphQLModelType.Project;
            // Only a Routine has a title and description field
            if (obj.hasOwnProperty('title')) return GraphQLModelType.Routine;
            return null; // GraphQLError is thrown
        },
    },
    Mutation: {
        /**
         * Adds or removes a vote to an object. A user can only cast one vote per object. So if a user re-votes, 
         * their previous vote is overruled. A user may vote on their own project/routine/etc.
         * @returns 
         */
        vote: async (_parent: undefined, { input }: IWrap<VoteInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await VoteModel(prisma).vote(req.userId, input);
            return { success };
        },
    }
}