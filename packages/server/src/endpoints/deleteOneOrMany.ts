import { Count, DeleteManyInput, DeleteOneInput, DeleteType, Success } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { deleteManyHelper, deleteOneHelper } from '../actions';
import { rateLimit } from '../middleware';
import { GQLEndpoint } from '../types';

export const typeDef = gql`
    enum DeleteType {
        Api
        ApiVersion
        Bookmark
        Chat
        ChatInvite
        ChatMessage
        ChatParticipant
        Comment
        Email
        FocusMode
        Issue
        Meeting
        MeetingInvite
        Node
        Note
        NoteVersion
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
        Resource
        Routine
        RoutineVersion
        RunProject
        RunRoutine
        Schedule
        SmartContract
        SmartContractVersion
        Standard
        StandardVersion
        Transfer
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