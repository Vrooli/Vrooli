import { FocusModeSortBy, FocusModeStopCondition } from "@local/shared";
import { EndpointsFocusMode, FocusModeEndpoints } from "../logic/focusMode";

export const typeDef = `#graphql
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
        you: FocusModeYou!
    }

    type FocusModeYou {
        canDelete: Boolean!
        canRead: Boolean!
        canUpdate: Boolean!
    }

    input SetActiveFocusModeInput {
        id: ID
        stopCondition: FocusModeStopCondition
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

    type ActiveFocusModeFocusMode {
        id: String!
        reminderListId: String
    }

    type ActiveFocusMode {
        focusMode: ActiveFocusModeFocusMode!
        stopCondition: FocusModeStopCondition!
        stopTime: Date
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
        setActiveFocusMode(input: SetActiveFocusModeInput!): ActiveFocusMode
    }
`;

export const resolvers: {
    FocusModeSortBy: typeof FocusModeSortBy;
    FocusModeStopCondition: typeof FocusModeStopCondition;
    Query: EndpointsFocusMode["Query"];
    Mutation: EndpointsFocusMode["Mutation"];
} = {
    FocusModeSortBy,
    FocusModeStopCondition,
    ...FocusModeEndpoints,
};
