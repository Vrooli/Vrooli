import { gql } from 'apollo-server-express';
import { UserSortBy, ViewSortBy, HistoryResult, HistoryInput, RunRoutineSortBy, RunStatus } from '@shared/consts';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { assertRequestFrom, getUser } from '../auth/request';
import { addSupplementalFieldsMultiTypes, toPartialGraphQLInfo } from '../builders';
import { PartialGraphQLInfo } from '../builders/types';
import { readManyAsFeedHelper } from '../actions';

export const typeDef = gql`
    input HistoryInput {
        searchString: String!
        take: Int
    }

    union AnyRun = RunProject | RunRoutine

    type HistoryResult {
        activeRuns: [AnyRun!]!
        completedRuns: [AnyRun!]!
        recentlyViewed: [View!]!
        recentlyStarred: [Star!]!
    }

    type Query {
        history(input: HistoryInput!): HistoryResult!
    }
`

export const resolvers: {
    Query: {
        history: GQLEndpoint<HistoryInput, HistoryResult>;
    },
} = {
    Query: {
        history: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'HistoryResult',
                activeRuns: {
                    RunProject: 'RunProject',
                    RunRoutine: 'RunRoutine',
                },
                completedRuns: {
                    RunProject: 'RunProject',
                    RunRoutine: 'RunRoutine',
                },
                recentlyViewed: 'View',
                recentlyStarred: 'Star',
            }, req.languages, true);
            const userId = userData.id;
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query for incomplete runs
            const { nodes: activeRunProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: (partial.activeRuns as PartialGraphQLInfo)?.RunProject as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.InProgress, sortBy: RunRoutineSortBy.DateUpdatedDesc },
                objectType: 'RunProject',
            });
            const { nodes: activeRunRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: (partial.activeRuns as PartialGraphQLInfo)?.RunRoutine as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.InProgress, sortBy: RunRoutineSortBy.DateUpdatedDesc },
                objectType: 'RunRoutine',
            });
            // Query for complete runs
            const { nodes: completedRunProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: (partial.completedRuns as PartialGraphQLInfo)?.RunProject as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.Completed, sortBy: RunRoutineSortBy.DateUpdatedDesc },
                objectType: 'RunProject',
            });
            const { nodes: completedRunRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: (partial.completedRuns as PartialGraphQLInfo)?.RunRoutine as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.Completed, sortBy: RunRoutineSortBy.DateUpdatedDesc },
                objectType: 'RunRoutine',
            });
            // Query recently viewed objects (of any type)
            const { nodes: recentlyViewed } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: partial.recentlyViewed as PartialGraphQLInfo,
                input: { take, ...input, sortBy: ViewSortBy.LastViewedDesc },
                objectType: 'View',
            });
            // Query recently starred objects (of any type). Make sure to ignore tags
            const { nodes: recentlyStarred } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: partial.recentlyStarred as PartialGraphQLInfo,
                input: { take, ...input, sortBy: UserSortBy.DateCreatedDesc, excludeTags: true },
                objectType: 'Star',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [activeRunProjects, activeRunRoutines, completedRunProjects, completedRunRoutines, recentlyViewed, recentlyStarred],
                [(partial.activeRuns as PartialGraphQLInfo)?.RunProject, (partial.activeRuns as PartialGraphQLInfo)?.RunRoutine, (partial.completedRuns as PartialGraphQLInfo)?.RunProject, (partial.completedRuns as PartialGraphQLInfo)?.RunRoutine, partial.recentlyViewed, partial.recentlyStarred] as PartialGraphQLInfo[],
                ['arp', 'arr', 'crp', 'crr', 'rv', 'rs'],
                getUser(req),
                prisma,
            )
            // Return results
            return {
                __typename: 'HistoryResult',
                // activeRuns combines projects and routines, and sorts by date started
                activeRuns: [...withSupplemental['arp'], ...withSupplemental['arr']].sort((a, b) => {
                    if (a.startedAt < b.startedAt) return -1;
                    if (a.startedAt > b.startedAt) return 1;
                    return 0;
                }),
                // completedRuns combines projects and routines, and sorts by date completed
                completedRuns: [...withSupplemental['crp'], ...withSupplemental['crr']].sort((a, b) => {
                    if (a.completedAt < b.completedAt) return -1;
                    if (a.completedAt > b.completedAt) return 1;
                    return 0;
                }),
                recentlyViewed: withSupplemental['rv'],
                recentlyStarred: withSupplemental['rs'],
            }
        },
    },
}