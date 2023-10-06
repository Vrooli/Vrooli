import { gql } from "apollo-server-express";
import { EndpointsReminderList, ReminderListEndpoints } from "../logic/reminderList";

export const typeDef = gql`
    input ReminderListCreateInput {
        id: ID!
        focusModeConnect: ID
        remindersCreate: [ReminderCreateInput!]
    }
    input ReminderListUpdateInput {
        id: ID!
        focusModeConnect: ID
        remindersCreate: [ReminderCreateInput!]
        remindersUpdate: [ReminderUpdateInput!]
        remindersDelete: [ID!]
    }
    type ReminderList {
        id: ID!
        created_at: Date!
        updated_at: Date!
        focusMode: FocusMode
        reminders: [Reminder!]!
    }

    extend type Mutation {
        reminderListCreate(input: ReminderListCreateInput!): ReminderList!
        reminderListUpdate(input: ReminderListUpdateInput!): ReminderList!
    }
`;

export const resolvers: {
    Mutation: EndpointsReminderList["Mutation"];
} = {
    ...ReminderListEndpoints,
};
