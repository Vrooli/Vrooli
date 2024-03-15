import { DeleteType } from "@local/shared";
import { gql } from "apollo-server-express";
import { DeleteOneOrManyEndpoints, EndpointsDeleteOneOrMany } from "../logic/deleteOneOrMany";

export const typeDef = gql`
    enum DeleteType {
        Api
        ApiKey
        ApiVersion
        Bookmark
        BookmarkList
        Chat
        ChatInvite
        ChatMessage
        ChatParticipant
        Comment
        Email
        FocusMode
        Issue
        Member
        MemberInvite
        Meeting
        MeetingInvite
        Node
        Note
        NoteVersion
        Notification
        Organization
        Phone
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
        Role
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
        User # Should only delete bots this way, not your account. Doing so won't allow you to select what happens to your data.
        Wallet
    }   

    input DeleteOneInput {
        id: ID!
        objectType: DeleteType!
    }

    input DeleteManyInput {
        objects: [DeleteOneInput!]!
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
