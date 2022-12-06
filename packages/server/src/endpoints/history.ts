import { gql } from 'apollo-server-express';
import { UserSortBy, RunStatus, ViewSortBy, HistoryResult, HistoryInput, RunRoutineSortBy } from './types';
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

    type HistoryResult {
        activeRuns: [RunRoutine!]!
        completedRuns: [RunRoutine!]!
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
                '__typename': 'HistoryResult',
                'activeRuns': 'RunRoutine',
                'completedRuns': 'RunRoutine',
                'recentlyViewed': 'View',
                'recentlyStarred': 'Star',
            }, req.languages, true);
            const userId = userData.id;
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query for incomplete runs
            const { nodes: activeRuns } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: partial.activeRuns as PartialGraphQLInfo,
                input: { take, ...input, status: RunStatus.InProgress, sortBy: RunRoutineSortBy.DateUpdatedDesc },
                objectType: 'RunRoutine',
            });
            // Query for complete runs
            const { nodes: completedRuns } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { userId },
                info: partial.completedRuns as PartialGraphQLInfo,
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
                [activeRuns, completedRuns, recentlyViewed, recentlyStarred],
                [partial.activeRuns, partial.completedRuns, partial.recentlyViewed, partial.recentlyStarred] as PartialGraphQLInfo[],
                ['ar', 'cr', 'rv', 'rs'],
                getUser(req),
                prisma,
            )
            // Return results
            return {
                activeRuns: withSupplemental['ar'],
                completedRuns: withSupplemental['cr'],
                recentlyViewed: withSupplemental['rv'],
                recentlyStarred: withSupplemental['rs'],
            } as any
        },
    },
}