import { gql } from 'apollo-server-express';
import { OrganizationSortBy, ProjectSortBy, ResourceUsedFor, RoutineSortBy, StandardSortBy, UserSortBy, Project, Routine, PopularInput, PopularResult, LearnResult, ResearchResult, DevelopResult } from '@shared/consts';
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
        organizations: [Organization!]!
        projects: [Project!]!
        routines: [Routine!]!
        standards: [Standard!]!
        users: [User!]!
    }
 
    type LearnResult {
        courses: [Project!]!
        tutorials: [Routine!]!
    }
 
    type ResearchResult {
        processes: [Routine!]!
        newlyCompleted: [ProjectOrRoutine!]!
        needVotes: [Project!]!
        needInvestments: [Project!]!
        needMembers: [Organization!]!
    }
 
    type DevelopResult {
        completed: [ProjectOrRoutine!]!
        inProgress: [ProjectOrRoutine!]!
        recent: [ProjectOrRoutine!]!
    }

    type Query {
        popular(input: PopularInput!): PopularResult!
        learn: LearnResult!
        research: ResearchResult!
        develop: DevelopResult!
    }
 `

export const resolvers: {
    Query: {
        popular: GQLEndpoint<PopularInput, PopularResult>;
        learn: GQLEndpoint<{}, LearnResult>;
        research: GQLEndpoint<{}, ResearchResult>;
        develop: GQLEndpoint<{}, DevelopResult>;
    }
} = {
    Query: {
        popular: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'PopularResult',
                organizations: 'Organization',
                projects: 'Project',
                routines: 'Routine',
                standards: 'Standard',
                users: 'User',
            }, req.languages, true);
            const MinimumStars = 0; // Minimum stars required to show up in results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query organizations
            const { nodes: organizations } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.organizations as PartialGraphQLInfo,
                input: { ...input, take, sortBy: OrganizationSortBy.StarsDesc },
                objectType: 'Organization',
            });
            // Query projects
            const { nodes: projects } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.projects as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.StarsDesc, isComplete: true },
                objectType: 'Project',
            });
            // Query routines
            const { nodes: routines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.routines as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.StarsDesc, isComplete: true, isInternal: false },
                objectType: 'Routine',
            });
            // Query standards
            const { nodes: standards } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.standards as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.StarsDesc, type: 'JSON' },
                objectType: 'Standard',
            });
            // Query users
            const { nodes: users } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery },
                info: partial.users as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.StarsDesc },
                objectType: 'User',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [organizations, projects, routines, standards, users],
                [partial.organizations, partial.projects, partial.routines, partial.standards, partial.users] as PartialGraphQLInfo[],
                ['o', 'p', 'r', 's', 'u'],
                getUser(req),
                prisma,
            )
            // Return results
            return {
                __typename: 'PopularResult' as const,
                organizations: withSupplemental['o'],
                projects: withSupplemental['p'],
                routines: withSupplemental['r'],
                standards: withSupplemental['s'],
                users: withSupplemental['u'],
            }
        },
        learn: async (_p, _d, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'LearnResult',
                courses: 'Project',
                tutorials: 'Routine',
            }, req.languages, true);
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query courses
            const { nodes: courses } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.courses as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.ScoreDesc, tags: ['Learn'], isComplete: true, },
                objectType: 'Project',
            });
            // Query tutorials
            const { nodes: tutorials } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.tutorials as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.ScoreDesc, tags: ['Learn'], isComplete: true, },
                objectType: 'Routine',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [courses, tutorials],
                [partial.courses, partial.tutorials] as PartialGraphQLInfo[],
                ['c', 't'],
                getUser(req),
                prisma,
            )
            // Return data
            return {
                __typename: 'LearnResult' as const,
                courses: withSupplemental['c'],
                tutorials: withSupplemental['t'],
            }
        },
        research: async (_p, _d, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'ResearchResult',
                processes: 'Routine',
                newlyCompleted: {
                    Routine: 'Routine',
                    Project: 'Project',
                },
                needVotes: 'Project',
                needInvestments: 'Project',
                needMembers: 'Organization',
            }, req.languages, true);
            const MinimumStars = 0; // Minimum stars required to show up in autocomplete results. Will increase in the future.
            const starsQuery = { stars: { gte: MinimumStars } };
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query processes
            const { nodes: processes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.processes as PartialGraphQLInfo,
                input: { take, sortBy: RoutineSortBy.ScoreDesc, tags: ['Research'], isComplete: true, isInternal: false },
                objectType: 'Routine',
            });
            // Query newlyCompleted
            const { nodes: newlyCompletedProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: (partial.newlyCompleted as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, sortBy: ProjectSortBy.DateCompletedAsc, isComplete: true },
                objectType: 'Project',
            });
            const { nodes: newlyCompletedRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: (partial.newlyCompleted as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, isComplete: true, isInternal: false, sortBy: RoutineSortBy.DateCompletedAsc },
                objectType: 'Routine',
            });
            // Query needVotes
            const { nodes: needVotes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.needVotes as PartialGraphQLInfo,
                input: { take, isComplete: false, resourceTypes: [ResourceUsedFor.Proposal] },
                objectType: 'Project',
            });
            // Query needInvestments
            const { nodes: needInvestments } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...starsQuery, isPrivate: false },
                info: partial.needInvestments as PartialGraphQLInfo,
                input: { take, isComplete: false, resourceTypes: [ResourceUsedFor.Donation] },
                objectType: 'Project',
            });
            // Query needMembers
            const { nodes: needMembers } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: partial.needMembers as PartialGraphQLInfo,
                input: { take, isOpenToNewMembers: true, sortBy: OrganizationSortBy.StarsDesc },
                objectType: 'Organization',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [processes, newlyCompletedProjects, newlyCompletedRoutines, needVotes, needInvestments, needMembers],
                [partial.processes, (partial.newlyCompleted as PartialGraphQLInfo)?.Project, (partial.newlyCompleted as PartialGraphQLInfo)?.Routine, partial.needVotes, partial.needInvestments, partial.needMembers] as PartialGraphQLInfo[],
                ['p', 'ncp', 'ncr', 'nv', 'ni', 'nm'],
                getUser(req),
                prisma,
            )
            // Return data
            return {
                __typename: 'ResearchResult' as const,
                processes: withSupplemental['p'],
                // newlyCompleted combines projects and routines, and sorts by date completed
                newlyCompleted: [...withSupplemental['ncp'], ...withSupplemental['ncr']].sort((a, b) => {
                    if (a.completedAt < b.completedAt) return -1;
                    if (a.completedAt > b.completedAt) return 1;
                    return 0;
                }),
                needVotes: withSupplemental['nv'],
                // needInvestments combines projects and organizations, and sorts by stars
                needInvestments: withSupplemental['ni'].sort((a, b) => {
                    if (a.stars < b.stars) return 1;
                    if (a.stars > b.stars) return -1;
                    return 0;
                }),
                needMembers: withSupplemental['nm'],
            }
        },
        develop: async (_p, _d, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGraphQLInfo(info, {
                __typename: 'DevelopResult',
                completed: {
                    Routine: 'Routine',
                    Project: 'Project',
                },
                inProgress: {
                    Routine: 'Routine',
                    Project: 'Project',
                },
                recent: {
                    Routine: 'Routine',
                    Project: 'Project',
                },
            }, req.languages, true);
            // If not signed in, return empty data
            const userId = getUser(req)?.id ?? null;
            if (!userId) return {
                __typename: 'DevelopResult' as const,
                completed: [],
                inProgress: [],
                recent: [],
            }
            const take = 5;
            const commonReadParams = { prisma, req }
            // Query for routines you've completed
            const { nodes: completedRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.completed as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, hasCompleteVersion: true, isInternal: false, userId, sortBy: RoutineSortBy.DateCompletedAsc },
                objectType: 'Routine',
            });
            // Query for projects you've completed
            const { nodes: completedProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.completed as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, hasCompleteVersion: true, userId, sortBy: ProjectSortBy.DateCompletedAsc },
                objectType: 'Project',
            });
            // Query for routines you're currently working on
            const { nodes: inProgressRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.inProgress as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, hasCompleteVersion: false, isInternal: false, userId, sortBy: RoutineSortBy.DateCreatedAsc },
                objectType: 'Routine',
            });
            // Query for projects you're currently working on
            const { nodes: inProgressProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.inProgress as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, hasCompleteVersion: false, userId, sortBy: ProjectSortBy.DateCreatedAsc },
                objectType: 'Project',
            });
            // Query recently created/updated routines
            const { nodes: recentRoutines } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.recent as PartialGraphQLInfo)?.Routine as PartialGraphQLInfo,
                input: { take, userId, sortBy: RoutineSortBy.DateUpdatedAsc, isInternal: false },
                objectType: 'Routine',
            });
            // Query recently created/updated projects
            const { nodes: recentProjects } = await readManyAsFeedHelper({
                ...commonReadParams,
                info: (partial.recent as PartialGraphQLInfo)?.Project as PartialGraphQLInfo,
                input: { take, userId, sortBy: ProjectSortBy.DateUpdatedAsc },
                objectType: 'Project',
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes(
                [completedRoutines, completedProjects, inProgressRoutines, inProgressProjects, recentRoutines, recentProjects],
                [(partial.completed as PartialGraphQLInfo)?.Routine, (partial.completed as PartialGraphQLInfo)?.Project, (partial.inProgress as PartialGraphQLInfo)?.Routine, (partial.inProgress as PartialGraphQLInfo)?.Project, (partial.recent as PartialGraphQLInfo)?.Routine, (partial.recent as PartialGraphQLInfo)?.Project] as PartialGraphQLInfo[],
                ['cr', 'cp', 'ipr', 'ipp', 'rr', 'rp'],
                getUser(req),
                prisma,
            )
            // Combine arrays
            const completed: Array<Project | Routine> = [...withSupplemental['cr'], ...withSupplemental['cp']];
            const inProgress: Array<Project | Routine> = [...withSupplemental['ipr'], ...withSupplemental['ipp']];
            const recent: Array<Project | Routine> = [...withSupplemental['rr'], ...withSupplemental['rp']];
            // Sort arrays
            completed.sort((a, b) => {
                if (a.updated_at < b.updated_at) return -1;
                if (a.updated_at > b.updated_at) return 1;
                return 0;
            });
            inProgress.sort((a, b) => {
                if (a.updated_at < b.updated_at) return -1;
                if (a.updated_at > b.updated_at) return 1;
                return 0;
            });
            recent.sort((a, b) => {
                if (a.updated_at < b.updated_at) return -1;
                if (a.updated_at > b.updated_at) return 1;
                return 0;
            });
            // Return data
            return {
                __typename: 'DevelopResult' as const,
                completed,
                inProgress,
                recent,
            }
        },
    },
}