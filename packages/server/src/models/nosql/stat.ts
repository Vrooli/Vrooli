// Time series of site statistics (t.g. active users, routines completed, etc.)
import mongoose from 'mongoose';

const schemaPartial = {
    timestamp: {
        type: Date,
        required: true,
    },
    activeUsers: {
        type: Number,
        required: true,
    },
    organizations: {
        type: Number,
        required: true,
    },
    projects: {
        type: Number,
        required: true,
    },
    // projectsCompleted: {
    //     type: Number,
    //     required: true,
    // },
    // projectsCompletionTimeAverage: {
    //     type: Number,
    //     required: true,
    // },
    routines: {
        type: Number,
        required: true,
    },
    standards: {
        type: Number,
        required: true,
    },
    // routinesCompleted: {
    //     type: Number,
    //     required: true,
    // },
    // routinesCompletionTimeAggregate: {
    //     type: Number,
    //     required: true,
    // },
    // routinesCompletionTimeAverage: {
    //     type: Number,
    //     required: true,
    // },
    // routineOrchestrationComplexityAggregate: {
    //     type: Number,
    //     required: true,
    // },
    // routineOrchestrationComplexityAverage: {
    //     type: Number,
    //     required: true,
    // },
    // routinesStarted: {
    //     type: Number,
    //     required: true,
    // },
    verifiedEmails: {
        type: Number,
        required: true,
    },
    verifiedWallets: {
        type: Number,
        required: true,
    },
}

const Schema = mongoose.Schema;

// Models for daily, weekly, monthly, yearly, and all time
export const StatDay = mongoose.model('StatDay', new Schema(schemaPartial, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'minutes',
    }   
}));
export const StatWeek = mongoose.model('StatWeek', new Schema(schemaPartial, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'hours',
    }   
}));
export const StatMonth = mongoose.model('StatMonth', new Schema(schemaPartial, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'hours',
    }   
}));
export const StatYear = mongoose.model('StatYear', new Schema(schemaPartial, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'hours',
    }   
}));
export const StatAllTime = mongoose.model('StatAllTime', new Schema(schemaPartial, {
    timeseries: {
        timeField: 'timestamp',
        timeGranularity: 'hours',
    }   
}));
