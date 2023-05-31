import { ActiveFocusMode, FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeSortBy, FocusModeStopCondition, FocusModeUpdateInput, SetActiveFocusModeInput, VisibilityType } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom, focusModeSelect, updateSessionCurrentUser } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export const typeDef = gql`
    enum FocusModeSortBy {
        NameAsc
        NameDesc
    }

    enum FocusModeStopCondition {
        AfterStopTime
        Automatic
        Never
        NextBegins
    }

    input FocusModeCreateInput {
        id: ID!
        name: String!
        description: String
        filtersCreate: [FocusModeFilterCreateInput!]
        reminderListConnect: ID
        reminderListCreate: ReminderListCreateInput
        resourceListCreate: ResourceListCreateInput
        scheduleCreate: ScheduleCreateInput
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input FocusModeUpdateInput {
        id: ID!
        name: String
        description: String
        filtersCreate: [FocusModeFilterCreateInput!]
        filtersDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        reminderListConnect: ID
        reminderListDisconnect: Boolean
        reminderListCreate: ReminderListCreateInput
        reminderListUpdate: ReminderListUpdateInput
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
    }
    type FocusMode {
        id: ID!
        created_at: Date!
        updated_at: Date!
        name: String!
        description: String
        filters: [FocusModeFilter!]!
        labels: [Label!]!
        reminderList: ReminderList
        resourceList: ResourceList
        schedule: Schedule
    }

    input SetActiveFocusModeInput {
        id: ID!
        stopCondition: FocusModeStopCondition!
        stopTime: Date
    }
    type ActiveFocusMode {
        mode: FocusMode!
        stopCondition: FocusModeStopCondition!
        stopTime: Date    
    }

    input FocusModeSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        scheduleStartTimeFrame: TimeFrame
        scheduleEndTimeFrame: TimeFrame
        searchString: String
        sortBy: FocusModeSortBy
        labelsIds: [ID!]
        take: Int
        timeZone: String
        updatedTimeFrame: TimeFrame
    }

    type FocusModeSearchResult {
        pageInfo: PageInfo!
        edges: [FocusModeEdge!]!
    }

    type FocusModeEdge {
        cursor: String!
        node: FocusMode!
    }

    extend type Query {
        focusMode(input: FindByIdInput!): FocusMode
        focusModes(input: FocusModeSearchInput!): FocusModeSearchResult!
    }

    extend type Mutation {
        focusModeCreate(input: FocusModeCreateInput!): FocusMode!
        focusModeUpdate(input: FocusModeUpdateInput!): FocusMode!
        setActiveFocusMode(input: SetActiveFocusModeInput!): ActiveFocusMode!
    }
`;

const objectType = "FocusMode";
export const resolvers: {
    FocusModeSortBy: typeof FocusModeSortBy;
    FocusModeStopCondition: typeof FocusModeStopCondition;
    Query: {
        focusMode: GQLEndpoint<FindByIdInput, FindOneResult<FocusMode>>;
        focusModes: GQLEndpoint<FocusModeSearchInput, FindManyResult<FocusMode>>;
    },
    Mutation: {
        focusModeCreate: GQLEndpoint<FocusModeCreateInput, CreateOneResult<FocusMode>>;
        focusModeUpdate: GQLEndpoint<FocusModeUpdateInput, UpdateOneResult<FocusMode>>;
        setActiveFocusMode: GQLEndpoint<SetActiveFocusModeInput, ActiveFocusMode>;
    }
} = {
    FocusModeSortBy,
    FocusModeStopCondition,
    Query: {
        focusMode: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        focusModes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        focusModeCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        focusModeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        setActiveFocusMode: async (_, { input }, { prisma, req, res }) => {
            // Unlink other objects, active focus mode is only stored in the user's session. 
            const userData = assertRequestFrom(req, { isUser: true });
            // Create time frame to limit focus mode's schedule data
            const now = new Date();
            const startDate = now;
            const endDate = new Date(now.setDate(now.getDate() + 7));
            // Find focus mode data & verify that it belongs to the user
            const focusMode = await prisma.focus_mode.findFirst({
                where: {
                    id: input.id,
                    user: { id: userData.id },
                },
                select: focusModeSelect(startDate, endDate),
            });
            if (!focusMode) throw new CustomError("0448", "NotFound", userData.languages);
            const activeFocusMode = {
                __typename: "ActiveFocusMode" as const,
                mode: focusMode as any as FocusMode,
                stopCondition: input.stopCondition,
                stopTime: input.stopTime,
            };
            // Set active focus mode in user's session token
            updateSessionCurrentUser(req, res, { activeFocusMode });
            return activeFocusMode;
        },
    },
};
