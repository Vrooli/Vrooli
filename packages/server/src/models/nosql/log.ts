// Logs user actions, such as viewing an item, starting/completing a routine, 
// joining/leaving an organization, etc.
import { GraphQLModelType } from '../../models/sql';
import mongoose from 'mongoose';
import { LogSearchInput, LogSortBy } from '../../schema/types';

// TODO ✅ = implemented
export enum LogType {
    Create = 'Create', // Create on object1Type ✅
    Delete = 'Delete', // Delete on object1Type ✅
    Downvote = 'Downvote', // Downvote on object1Type ✅
    OrganizationAddMember = 'OrganizationAddMember', // Added member object2Id to organization object1Id
    OrganizationJoin = 'OrganizationJoin', // Joined organization object1Id
    OrganizationLeave = 'OrganizationLeave', // Left organization object1Id
    OrganizationRemoveMember = 'OrganizationRemoveMember', // Removed member object2Id from organization object1Id
    OrganizationUpdateMember = 'OrganizationUpdateMember', // Updated member object2Id in organization object1Id
    ProjectComplete = 'ProjectComplete', // Completed project
    RemoveStar = 'RemoveStar', // Unstarred object1Id
    RemoveVote = 'RemoveVote', // Unvoted object1Id
    RoutineCancel = 'RoutineCancel', // Canceled running routine object1Id
    RoutineComplete = 'RoutineComplete', // Finished running routine object1Id ✅
    RoutineStartIncomplete = 'RoutineStartIncomplete', // Started routine object1Id, and it has not been completed yet ✅
    RoutineStartCanceled = 'RoutineStartCanceled', // Started routine object1Id, but it was canceled. This is used to update a RoutineStartIncomplete, which simplifies querying
    RoutineStartCompleted = 'RoutineStartCompleted', // Started routine object1Id, and it has been completed. This is used to update a RoutineStartIncomplete, which simplifies querying ✅
    Star = 'Star', // Starred object1Id
    Update = 'Update', // Update on object1Type ✅
    Upvote = 'Upvote', // Upvoted object1Id
    View = 'View', // Viewed object1Type
}

// Structure of data
const Schema = mongoose.Schema;
const logSchema = new Schema({
    // The time the log was created
    timestamp: {
        type: Date,
        required: true,
    },
    // The user the log is for. This should always be a search parameter, 
    // so you don't see logs for other users.
    userId: {
        type: String,
        required: true,
    },
    // The type of log
    action: {
        type: String,
        enum: Object.values(LogType),
        required: true,
    },
    // The main type of object the log is for.
    object1Type: {
        type: String,
        enum: [
            GraphQLModelType.Organization,
            GraphQLModelType.Project,
            GraphQLModelType.Routine,
            GraphQLModelType.Standard,
            GraphQLModelType.User,
        ],
        required: false,
    },
    // The id of the main object the log is for.
    object1Id: {
        type: String,
        required: false,
    },
    // An optional secondary object type that the log is for.
    object2Type: {
        type: String,
        enum: [
            GraphQLModelType.Comment,
            GraphQLModelType.Organization,
            GraphQLModelType.Project,
            GraphQLModelType.Routine,
            GraphQLModelType.Standard,
            GraphQLModelType.User,
        ],
        required: false,
    },
    // An optional secondary object id that the log is for.
    object2Id: {
        type: String,
        required: false,
    },
    // An optional session string that the log is for, to link logs together. 
    // For example, an associate routine start and complete logs together, so 
    // the duration is easy to calculate.
    session: {
        type: String,
        required: false,
    },
    // Additional data that may be useful for the log. Basically a catch-all field.
    // Doesn't have to be JSON, but probably should be.
    data: {
        type: String,
        required: false,
    },
}, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'seconds',
    }
});

// Model for logs. Logs during development are sent to their own collection.
const collectionName = process.env.NODE_ENV === 'development' ? 'LogDev' : 'Log';
export const Log = mongoose.model(collectionName, logSchema);

// Search for logs
export const logSearcher = () => ({
    defaultSort: LogSortBy.DateCreatedDesc,
    defaultProjection: { _id: 1, timestamp: 1, userId: 1, action: 1, object1Type: 1, object1Id: 1, object2Type: 1, object2Id: 1, session: 1, data: 1 },
    getSortQuery: (sortBy: string): any => {
        return {
            [LogSortBy.DateCreatedAsc]: { timestamp: 1, _id: 1 },
            [LogSortBy.DateCreatedDesc]: { timestamp: -1, _id: 1 },
        }[sortBy]
    },
    getFindQuery(userId: string, input: LogSearchInput): { [x: string]: any } {
        const queries = [
            input.createdTimeFrame ? { timestamp: { $gte: new Date(input.createdTimeFrame.after), $lte: new Date(input.createdTimeFrame.before) } } : undefined,
            input.ids ? { _id: { $in: input.ids } } : undefined,
            input.object1Id ? { object1Id: input.object1Id } : undefined,
            input.object2Id ? { object2Id: input.object2Id } : undefined,
            input.object1Type ? { object1Type: input.object1Type } : undefined,
            input.object2Type ? { object2Type: input.object2Type } : undefined,
            input.actions ? (input.actions.length !== 1 ? { action: { $in: input.actions } } : { action: input.actions[0] }) : undefined,
            input.data ? { data: input.data } : undefined,
        ]
        // Combine all the queries
        const existingQueries = queries.filter(q => q);
        if (existingQueries.length > 0) {
            return { $and: [{ userId }, ...existingQueries] };
        } else {
            return { userId };
        }
    },
})