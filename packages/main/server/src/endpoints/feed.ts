import { ApiSortBy, HomeInput, HomeResult, NoteSortBy, OrganizationSortBy, PopularInput, PopularObjectType, PopularResult, PopularSortBy, ProjectSortBy, QuestionSortBy, ReminderSortBy, ResourceSortBy, RoutineSortBy, ScheduleSortBy, SmartContractSortBy, StandardSortBy, UserSortBy } from ":/consts";
import { exists } from ":/utils";
import { gql } from "apollo-server-express";
import { readManyAsFeedHelper } from "../actions";
import { assertRequestFrom, getUser } from "../auth/request";
import { addSupplementalFieldsMultiTypes, toPartialGqlInfo } from "../builders";
import { PartialGraphQLInfo } from "../builders/types";
import { schedulesWhereInTimeframe } from "../events";
import { rateLimit } from "../middleware";
import { GQLEndpoint } from "../types";

export const typeDef = gql`
    enum PopularSortBy {
        StarsAsc
        StarsDesc
        ViewsAsc
        ViewsDesc
    }

    enum PopularObjectType {
        Api
        Note
        Organization
        Project
        Question
        Routine
        SmartContract
        Standard
        User
    }  

    input PopularInput {
        objectType: PopularObjectType # To limit to a specific type
        searchString: String!
        sortBy: PopularSortBy
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
    }

    type HomeResult {
        notes: [Note!]!
        reminders: [Reminder!]!
        resources: [Resource!]!
        schedules: [Schedule!]!
    }

    type Query {
        home(input: HomeInput!): HomeResult!
        popular(input: PopularInput!): PopularResult!
    }
 `;

export const resolvers: {
    PopularSortBy: typeof PopularSortBy;
    PopularObjectType: typeof PopularObjectType;
    Query: {
        home: GQLEndpoint<HomeInput, HomeResult>;
        popular: GQLEndpoint<PopularInput, PopularResult>;
    }
} = {
    PopularSortBy,
    PopularObjectType,
    Query: {
        home: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 5000, req });
            const activeFocusMode = userData.activeFocusMode;
            const partial = toPartialGqlInfo(info, {
                __typename: "HomeResult",
                notes: "Note",
                reminders: "Reminder",
                resources: "Resource",
                schedules: "Schedule",
            }, req.languages, true);
            const take = 5;
            const commonReadParams = { prisma, req };
            // Query notes
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ownedByUser: { id: userData.id } },
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: NoteSortBy.DateUpdatedDesc },
                objectType: "Note",
            });
            // Query reminders
            const { nodes: reminders } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    reminderList: {
                        focusMode: {
                            ...(activeFocusMode ? { id: activeFocusMode.mode.id } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.reminders as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ReminderSortBy.DateCreatedAsc, isComplete: false },
                objectType: "Reminder",
            });
            // Query resources
            const { nodes: resources } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    list: {
                        focusMode: {
                            ...(activeFocusMode ? { id: activeFocusMode.mode.id } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.resources as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ResourceSortBy.IndexAsc },
                objectType: "Resource",
            });
            // Query schedules that might occur in the next 7 days. 
            // Need to perform calculations on the client side to determine which ones are actually relevant.
            const now = new Date();
            const startDate = now;
            const endDate = new Date(now.setDate(now.getDate() + 7));
            const { nodes: schedules } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    ...schedulesWhereInTimeframe(startDate, endDate),
                    OR: [
                        {
                            meetings: {
                                some: {
                                    attendees: {
                                        some: {
                                            user: {
                                                id: userData.id,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                        {
                            runProjects: {
                                some: {
                                    user: {
                                        id: userData.id,
                                    },
                                },
                            },
                        },
                        {
                            runRoutines: {
                                some: {
                                    user: {
                                        id: userData.id,
                                    },
                                },
                            },
                        },
                    ],
                },
                info: partial.schedules as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ScheduleSortBy.EndTimeAsc },
                objectType: "Schedule",
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                notes,
                reminders,
                resources,
                schedules,
            }, partial as any, prisma, getUser(req));
            // Return results
            return {
                __typename: "HomeResult" as const,
                ...withSupplemental,
            };
        },
        popular: async (_, { input }, { prisma, req }, info) => {
            //TODO implement sorting. Must do in UI too if combining multiple types.
            await rateLimit({ info, maxUser: 5000, req });
            const partial = toPartialGqlInfo(info, {
                __typename: "PopularResult",
                apis: "Api",
                notes: "Note",
                organizations: "Organization",
                projects: "Project",
                questions: "Question",
                routines: "Routine",
                smartContracts: "SmartContract",
                standards: "Standard",
                users: "User",
            }, req.languages, true);
            const bookmarksQuery = { bookmarks: { gte: 0 } };  // Minimum bookmarks required to show up in results. Should increase in the future.
            const take = 5;
            const commonReadParams = { prisma, req };
            // Checks if object type should be included in results
            const shouldInclude = (objectType: PopularObjectType | `${PopularObjectType}`) => {
                if (exists(input.objectType)) return input.objectType === objectType;
                return true;
            };
            // Query apis
            const { nodes: apis } = shouldInclude("Api") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.apis as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ApiSortBy.ScoreDesc },
                objectType: "Api",
            }) : { nodes: [] };
            // Query notes
            const { nodes: notes } = shouldInclude("Note") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.notes as PartialGraphQLInfo,
                input: { ...input, take, sortBy: NoteSortBy.ScoreDesc },
                objectType: "Note",
            }) : { nodes: [] };
            // Query organizations
            const { nodes: organizations } = shouldInclude("Organization") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.organizations as PartialGraphQLInfo,
                input: { ...input, take, sortBy: OrganizationSortBy.BookmarksDesc },
                objectType: "Organization",
            }) : { nodes: [] };
            // Query projects
            const { nodes: projects } = shouldInclude("Project") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.projects as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ProjectSortBy.ScoreDesc, isComplete: true },
                objectType: "Project",
            }) : { nodes: [] };
            // Query questions
            const { nodes: questions } = shouldInclude("Question") ? await readManyAsFeedHelper({
                ...commonReadParams,
                // Make sure question is not attached to any objects (i.e. standalone)
                additionalQueries: { ...bookmarksQuery, api: null, note: null, organization: null, project: null, routine: null, smartContract: null, standard: null },
                info: partial.questions as PartialGraphQLInfo,
                input: { ...input, take, sortBy: QuestionSortBy.ScoreDesc, isComplete: true },
                objectType: "Question",
            }) : { nodes: [] };
            // Query routines
            const { nodes: routines } = shouldInclude("Routine") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.routines as PartialGraphQLInfo,
                input: { ...input, take, sortBy: RoutineSortBy.ScoreDesc, isComplete: true, isInternal: false },
                objectType: "Routine",
            }) : { nodes: [] };
            // Query smart contracts
            const { nodes: smartContracts } = shouldInclude("SmartContract") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.smartContracts as PartialGraphQLInfo,
                input: { ...input, take, sortBy: SmartContractSortBy.ScoreDesc, isComplete: true },
                objectType: "SmartContract",
            }) : { nodes: [] };
            // Query standards
            const { nodes: standards } = shouldInclude("Standard") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.standards as PartialGraphQLInfo,
                input: { ...input, take, sortBy: StandardSortBy.ScoreDesc, type: "JSON" },
                objectType: "Standard",
            }) : { nodes: [] };
            // Query users
            const { nodes: users } = shouldInclude("User") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery },
                info: partial.users as PartialGraphQLInfo,
                input: { ...input, take, sortBy: UserSortBy.BookmarksDesc },
                objectType: "User",
            }) : { nodes: [] };
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                apis,
                notes,
                organizations,
                projects,
                questions,
                routines,
                smartContracts,
                standards,
                users,
            }, partial as any, prisma, getUser(req));
            // Return results
            return {
                __typename: "PopularResult" as const,
                ...withSupplemental,
            };
        },
    },
};
