import { ApiSortBy, FocusModeSortBy, HomeInput, HomeResult, NoteSortBy, OrganizationSortBy, PopularInput, PopularResult, ProjectSortBy, QuestionSortBy, ReminderSortBy, ResourceSortBy, RoutineSortBy, ScheduleSortBy, SmartContractSortBy, StandardSortBy, UserSortBy } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { readManyAsFeedHelper } from '../actions';
import { assertRequestFrom, getUser } from '../auth/request';
import { addSupplementalFieldsMultiTypes, toPartialGraphQLInfo } from '../builders';
import { PartialGraphQLInfo } from '../builders/types';
import { rateLimit } from '../middleware';
import { GQLEndpoint } from '../types';

export const typeDef = gql`
    input PopularInput {
        searchString: String!
        take: Int
    }
 
    type PopularResult {
        apis: [Api!]!
        notes: [Note!]!
        organizations: [Organization!]!
        projects: [Project!]!
        questions: [Question!]!
        routines: [Routine!]!
        smartContracts: [SmartContract!]!
        standards: [Standard!]!
        users: [User!]!
    }

    input HomeInput {
        searchString: String!
        take: Int
        focusModeId: ID
    }

    type HomeResult {
        notes: [Note!]!
        reminders: [Reminder!]!
        schedules: [Schedule!]!
        resources: [Resource!]!
        focusModes: [FocusMode!]!
    }

    type Query {
        home(input: HomeInput!): HomeResult!
        popular(input: PopularInput!): PopularResult!
    }
 `

export const resolvers: {
    Query: {
        home: GQLEndpoint<HomeInput, HomeResult>;
        popular: GQLEndpoint<PopularInput, PopularResult>;
    }
} = {
    Query: {
        home: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 5000, req });
            // If no focus mode specified, try to find the user's active focus modes
            const allFocusModes = await prisma.focus_mode.findMany({
                where: { userId: userData.id },
            });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'HomeResult',
                focusModes: 'FocusMode',
                notes: 'Note',
                reminders: 'Reminder',
                resources: 'Resource',
                schedules: 'Schedule',
            }, req.languages, true);
            const take = 5;
            const commonReadParams = { prisma, req }
            const activeScheduleIds = [];//TODO
            // Query notes
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find your own notes only
                additionalQueries: {},
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: NoteSortBy.DateUpdatedDesc },
                objectType: 'Note',
            });
            // Query reminders
            const { nodes: reminders } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find all of your reminders if "showOnlyRelevantToSchedule" is false, otherwise find reminders associated with your active schedule(s)
                additionalQueries: {},
                info: partial.reminders as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ReminderSortBy.DateCreatedAsc, isComplete: false },
                objectType: 'Reminder',
            });
            // Query resources
            const { nodes: resources } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    // list: {
                    //     userSchedule: input.showOnlyRelevantToSchedule ? {
                    //         id: { in: activeScheduleIds }
                    //     } : {
                    //         user: {
                    //             id: userData.id
                    //         }
                    //     }
                    // }
                },
                info: partial.resources as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ResourceSortBy.IndexAsc },
                objectType: 'Resource',
            });
            // Query schedules
            const { nodes: schedules } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find meetings that have not ended yet, and you are invited to (or you are the owner)
                // TODO Find all of your runProjectSchedules that have not ended yet, or which are recurring and the recurr period has not ended yet
                // TODO Find all of your runRoutineSchedules that have not ended yet, or which are recurring and the recurr period has not ended yet
                additionalQueries: {},
                info: partial.schedules as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ScheduleSortBy.EndTimeAsc },
                objectType: 'Schedule',
            });
            // Query focusModes
            const { nodes: focusModes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { user: { id: userData.id } },
                info: partial.focusModes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: FocusModeSortBy.NameAsc },
                objectType: 'FocusMode',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [focusModes, notes, reminders, resources, schedules],
                [partial.focusModes, partial.notes, partial.reminders, partial.resources, partial.schedules] as PartialGraphQLInfo[],
                ['f', 'n', 'rem', 'res', 's'],
                getUser(req),
                prisma,
            )
            // Return results
            return {
                __typename: 'HomeResult' as const,
                focusModes: withSupplemental['f'],
                notes: withSupplemental['n'],
                reminders: withSupplemental['rem'],
                resources: withSupplemental['res'],
                schedules: withSupplemental['s'],
            }
        },
        popular: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'PopularResult',
                apis: 'Api',
                notes: 'Note',
                organizations: 'Organization',
                projects: 'Project',
                questions: 'Question',
                routines: 'Routine',
                smartContracts: 'SmartContract',
                standards: 'Standard',
                users: 'User',
            }, req.languages, true);
            const bookmarksQuery = { bookmarks: { gte: 0 } };  // Minimum bookmarks required to show up in results. Should increase in the future.
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query apis
            const { nodes: apis } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.apis as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ApiSortBy.ScoreDesc },
                objectType: 'Api',
            });
            // Query notes
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: NoteSortBy.ScoreDesc },
                objectType: 'Note',
            });
            // Query organizations
            const { nodes: organizations } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.organizations as PartialGraphQLInfo,
                input: { ...input, take, sortBy: OrganizationSortBy.BookmarksDesc },
                objectType: 'Organization',
            });
            // Query projects
            const { nodes: projects } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.projects as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.ScoreDesc, isComplete: true },
                objectType: 'Project',
            });
            // Query questions
            const { nodes: questions } = await readManyAsFeedHelper({
                ...commonReadParams,
                // Make sure question is not attached to any objects (i.e. standalone)
                additionalQueries: { ...bookmarksQuery, api: null, note: null, organization: null, project: null, routine: null, smartContract: null, standard: null },
                info: partial.questions as PartialGraphQLInfo,
                input: { ...input, take, sortBy: QuestionSortBy.ScoreDesc, isComplete: true },
                objectType: 'Question',
            });
            // Query routines
            const { nodes: routines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.routines as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.ScoreDesc, isComplete: true, isInternal: false },
                objectType: 'Routine',
            });
            // Query smart contracts
            const { nodes: smartContracts } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.smartContracts as PartialGraphQLInfo,
                input: { ...input, take, sortBy: SmartContractSortBy.ScoreDesc, isComplete: true },
                objectType: 'SmartContract',
            });
            // Query standards
            const { nodes: standards } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.standards as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.ScoreDesc, type: 'JSON' },
                objectType: 'Standard',
            });
            // Query users
            const { nodes: users } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery },
                info: partial.users as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.BookmarksDesc },
                objectType: 'User',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [apis, notes, organizations, projects, questions, routines, smartContracts, standards, users],
                [partial.apis, partial.notes, partial.organizations, partial.projects, partial.questions, partial.routines, partial.smartContracts, partial.standards, partial.users] as PartialGraphQLInfo[],
                ['a', 'n', 'o', 'p', 'q', 'r', 'sc', 'st', 'u'],
                getUser(req),
                prisma,
            )
            console.log('feeed made it heere 2', JSON.stringify(withSupplemental), '\n\n')
            // Return results
            return {
                __typename: 'PopularResult' as const,
                apis: withSupplemental['a'],
                notes: withSupplemental['n'],
                organizations: withSupplemental['o'],
                projects: withSupplemental['p'],
                questions: withSupplemental['q'],
                routines: withSupplemental['r'],
                smartContracts: withSupplemental['sc'],
                standards: withSupplemental['st'],
                users: withSupplemental['u'],
            }
        },
    },
}