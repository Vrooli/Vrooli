import { gql } from 'apollo-server-express';
import { OrganizationSortBy, ProjectSortBy, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, Project, Routine, PopularInput, PopularResult, ApiSortBy, NoteSortBy, SmartContractSortBy, HomeInput, HomeResult, MeetingSortBy } from '@shared/consts';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { getUser } from '../auth/request';
import { addSupplementalFieldsMultiTypes, toPartialGraphQLInfo } from '../builders';
import { PartialGraphQLInfo } from '../builders/types';
import { readManyAsFeedHelper } from '../actions';

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
        showOnlyRelevantToSchedule: Boolean
        take: Int
    }

    type HomeResult {
        meetings: [Meeting!]!
        notes: [Note!]!
        reminders: [Reminder!]!
        resources: [Resource!]!
        runProjectSchedules: [RunProjectSchedule!]!
        runRoutineSchedules: [RunRoutineSchedule!]!
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
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'HomeResult',
                meetings: 'Meeting',
                notes: 'Note',
                reminders: 'Reminder',
                resources: 'Resource',
                runProjectSchedules: 'RunProjectSchedule',
                runRoutineSchedules: 'RunRoutineSchedule',
            }, req.languages, true);
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query meetings
            const { nodes: meetings } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find meetings that have not ended yet, and you are invited to (or you are the owner)
                additionalQueries: {  },
                info: partial.meetings as PartialGraphQLInfo,
                input: { ...input, take, sortBy: MeetingSortBy.EventEndAsc },
                objectType: 'Meeting',
            });
            // Query notes
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find your own notes only
                additionalQueries: {  },
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.BookmarksDesc, isComplete: true },
                objectType: 'Note',
            });
            // Query reminders
            const { nodes: reminders } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find all of your reminders if "showOnlyRelevantToSchedule" is false, otherwise find reminders associated with your active schedule(s)
                additionalQueries: { },
                info: partial.reminders as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.BookmarksDesc, isComplete: true, isInternal: false },
                objectType: 'Reminder',
            });
            // Query resources
            const { nodes: resources } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find all of your resources if "showOnlyRelevantToSchedule" is false, otherwise find resources associated with your active schedule(s)
                additionalQueries: {  },
                info: partial.resources as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.BookmarksDesc, type: 'JSON' },
                objectType: 'Resource',
            });
            // Query runProjectSchedules
            const { nodes: runProjectSchedules } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find all of your runProjectSchedules that have not ended yet, or which are recurring and the recurr period has not ended yet
                additionalQueries: { },
                info: partial.runProjectSchedules as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.BookmarksDesc },
                objectType: 'RunProjectSchedule',
            });
            // Query runRoutineSchedules
            const { nodes: runRoutineSchedules } = await readManyAsFeedHelper({
                ...commonReadParams,
                // TODO Find all of your runRoutineSchedules that have not ended yet, or which are recurring and the recurr period has not ended yet
                additionalQueries: { },
                info: partial.runRoutineSchedules as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.BookmarksDesc },
                objectType: 'RunRoutineSchedule',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [meetings, notes, reminders, resources, runProjectSchedules, runRoutineSchedules],
                [partial.meetings, partial.notes, partial.reminders, partial.resources, partial.runProjectSchedules, partial.runRoutineSchedules] as PartialGraphQLInfo[],
                ['m', 'n', 'rem', 'res', 'rps', 'rrs'],
                getUser(req),
                prisma,
            )
            // Return results
            return {
                __typename: 'HomeResult' as const,
                meetings: withSupplemental['m'],
                notes: withSupplemental['n'],
                reminders: withSupplemental['rem'],
                resources: withSupplemental['res'],
                runProjectSchedules: withSupplemental['rps'],
                runRoutineSchedules: withSupplemental['rrs'],
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
                input: { ...input, take, sortBy: ApiSortBy.BookmarksDesc },
                objectType: 'Api',
            });
            // Query notes
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: NoteSortBy.BookmarksDesc },
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
                input: { ...input, take, sortBy: ProjectSortBy.BookmarksDesc, isComplete: true },
                objectType: 'Project',
            });
            // Query questions
            const { nodes: questions } = await readManyAsFeedHelper({
                ...commonReadParams,
                // Make sure question is not attached to any objects (i.e. standalone)
                additionalQueries: { ...bookmarksQuery, api: null, note: null, organization: null, project: null, routine: null, smartContract: null, standard: null},
                info: partial.questions as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.BookmarksDesc, isComplete: true },
                objectType: 'Project',
            });
            // Query routines
            const { nodes: routines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.routines as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.BookmarksDesc, isComplete: true, isInternal: false },
                objectType: 'Routine',
            });
            // Query smart contracts
            const { nodes: smartContracts } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.smartContracts as PartialGraphQLInfo,
                input: { ...input, take, sortBy: SmartContractSortBy.BookmarksDesc, isComplete: true },
                objectType: 'SmartContract',
            });
            // Query standards
            const { nodes: standards } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.standards as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.BookmarksDesc, type: 'JSON' },
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