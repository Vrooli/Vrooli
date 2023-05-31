import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

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

const objectType = "ReminderList";
export const resolvers: {
    Mutation: {
        reminderListCreate: GQLEndpoint<ReminderListCreateInput, CreateOneResult<ReminderList>>;
        reminderListUpdate: GQLEndpoint<ReminderListUpdateInput, UpdateOneResult<ReminderList>>;
    }
} = {
    Mutation: {
        reminderListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reminderListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
