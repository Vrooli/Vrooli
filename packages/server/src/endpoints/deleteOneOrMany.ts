import { gql } from 'apollo-server-express';
import { Count, DeleteManyInput, DeleteOneInput, Success } from '@shared/consts';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { DeleteType } from '@shared/consts';
import { deleteManyHelper, deleteOneHelper } from '../actions';

export const typeDef = gql`
    enum DeleteType {
        Api
        ApiVersion
        Bookmark
        Comment
        Email
        Issue
        Meeting
        MeetingInvite
        Node
        Organization
        Post
        Project
        ProjectVersion
        PullRequest
        PushDevice
        Question
        QuestionAnswer
        Quiz
        Reminder
        ReminderList
        Report
        Routine
        RoutineVersion
        RunProject
        RunRoutine
        SmartContract
        SmartContractVersion
        Standard
        StandardVersion
        Transfer
        UserSchedule
        Wallet
    }   

    input DeleteOneInput {
        id: ID!
        objectType: DeleteType!
    }

    input DeleteManyInput {
        ids: [ID!]!
        objectType: DeleteType!
    }

    extend type Mutation {
        deleteOne(input: DeleteOneInput!): Success!
        deleteMany(input: DeleteManyInput!): Count!
    }
`

export const resolvers: {
    DeleteType: typeof DeleteType;
    Mutation: {
        deleteOne: GQLEndpoint<DeleteOneInput, Success>;
        deleteMany: GQLEndpoint<DeleteManyInput, Count>;
    }
} = {
    DeleteType,
    Mutation: {
        deleteOne: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return deleteOneHelper({ input, objectType: input.objectType, prisma, req });
        },
        deleteMany: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return deleteManyHelper({ input, objectType: input.objectType, prisma, req });
        }
    }
}