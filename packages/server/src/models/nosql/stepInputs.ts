/**
 * Stores user inputs for a step. Contains: 
 * - For each step:
 *    - The step id
 *    - The run id
 *    - The node id
 *    - The main routine id
 *    - The subroutine id
 *    - The user id
 *    - For each input:
 *       - The input id
 *       - The standard id
 *       - The input name
 *       - The input value
 */
import mongoose from 'mongoose';
// import { LogSearchInput, LogSortBy } from '../../schema/types';

// Structure of data
const Schema = mongoose.Schema;
const stepInputsSchema = new Schema({
    stepId: {
        type: String,
        required: true,
    },
    runId: {
        type: String,
        required: true,
    },
    nodeId: {
        type: String,
        required: true,
    },
    routineId: {
        type: String,
        required: true,
    },
    subroutineId: {
        type: String,
        required: false,
    },
    userId: {
        type: String,
        required: true,
    },
    inputs: [{
        inputId: {
            type: String,
            required: true,
        },
        standardId: {
            type: String,
            required: true,
        },
        name: {
            type: String,
            required: true,
        },
        value: {
            type: String,
            required: true,
        },
    }],
});

const collectionName = process.env.NODE_ENV === 'development' ? 'StepInputsDev' : 'StepInputs';
export const StepInputs = mongoose.model(collectionName, stepInputsSchema);

// Search for inputs
export const stepInputsSearcher = () => ({
    // defaultSort: LogSortBy.DateCreatedDesc,
    // defaultProjection: { _id: 1, timestamp: 1, userId: 1, action: 1, object1Type: 1, object1Id: 1, object2Type: 1, object2Id: 1, session: 1, data: 1 },
    // getSortQuery: (sortBy: string): any => {
    //     return {
    //         [LogSortBy.DateCreatedAsc]: { timestamp: 1, _id: 1 },
    //         [LogSortBy.DateCreatedDesc]: { timestamp: -1, _id: 1 },
    //     }[sortBy]
    // },
    // getFindQuery(userId: string, input: LogSearchInput): { [x: string]: any } {
    //     const queries = [
    //         input.createdTimeFrame ? { timestamp: { $gte: new Date(input.createdTimeFrame.after), $lte: new Date(input.createdTimeFrame.before) } } : undefined,
    //         input.ids ? { _id: { $in: input.ids } } : undefined,
    //         input.object1Id ? { object1Id: input.object1Id } : undefined,
    //         input.object2Id ? { object2Id: input.object2Id } : undefined,
    //         input.object1Type ? { object1Type: input.object1Type } : undefined,
    //         input.object2Type ? { object2Type: input.object2Type } : undefined,
    //         input.actions ? (input.actions.length !== 1 ? { action: { $in: input.actions } } : { action: input.actions[0] }) : undefined,
    //         input.data ? { data: input.data } : undefined,
    //     ]
    //     // Combine all the queries
    //     const existingQueries = queries.filter(q => q);
    //     if (existingQueries.length > 0) {
    //         return { $and: [{ userId }, ...existingQueries] };
    //     } else {
    //         return { userId };
    //     }
    // },
})