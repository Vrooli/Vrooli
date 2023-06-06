import { DeleteType } from "@local/shared";
import { gql } from "apollo-server-express";
import { DeleteOneOrManyEndpoints, EndpointsDeleteOneOrMany } from "../logic";

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
        Notification
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
`;

export const resolvers: {
    DeleteType: typeof DeleteType;
    Mutation: EndpointsDeleteOneOrMany["Mutation"];
} = {
    DeleteType,
    ...DeleteOneOrManyEndpoints,
};
