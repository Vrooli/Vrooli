import { ApiSortBy, NoteSortBy, OrganizationSortBy, PopularObjectType, PopularSortBy, ProjectSortBy, QuestionSortBy, ReminderSortBy, ResourceSortBy, RoutineSortBy, ScheduleSortBy, SmartContractSortBy, StandardSortBy, UserSortBy } from "@local/consts";
import { exists } from "@local/utils";
import { gql } from "apollo-server-express";
import { readManyAsFeedHelper } from "../actions";
import { assertRequestFrom, getUser } from "../auth/request";
import { addSupplementalFieldsMultiTypes, toPartialGqlInfo } from "../builders";
import { schedulesWhereInTimeframe } from "../events";
import { rateLimit } from "../middleware";
export const typeDef = gql `
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
export const resolvers = {
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
            const { nodes: notes } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ownedByUser: { id: userData.id } },
                info: partial.notes,
                input: { ...input, take, sortBy: NoteSortBy.DateUpdatedDesc },
                objectType: "Note",
            });
            const { nodes: reminders } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    reminderList: {
                        focusMode: {
                            ...(activeFocusMode ? { id: activeFocusMode.mode.id } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.reminders,
                input: { ...input, take, sortBy: ReminderSortBy.DateCreatedAsc, isComplete: false },
                objectType: "Reminder",
            });
            const { nodes: resources } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    list: {
                        focusMode: {
                            ...(activeFocusMode ? { id: activeFocusMode.mode.id } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.resources,
                input: { ...input, take, sortBy: ResourceSortBy.IndexAsc },
                objectType: "Resource",
            });
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
                info: partial.schedules,
                input: { ...input, take, sortBy: ScheduleSortBy.EndTimeAsc },
                objectType: "Schedule",
            });
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                notes,
                reminders,
                resources,
                schedules,
            }, partial, prisma, getUser(req));
            return {
                __typename: "HomeResult",
                ...withSupplemental,
            };
        },
        popular: async (_, { input }, { prisma, req }, info) => {
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
            const bookmarksQuery = { bookmarks: { gte: 0 } };
            const take = 5;
            const commonReadParams = { prisma, req };
            const shouldInclude = (objectType) => {
                if (exists(input.objectType))
                    return input.objectType === objectType;
                return true;
            };
            const { nodes: apis } = shouldInclude("Api") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.apis,
                input: { ...input, take, sortBy: ApiSortBy.ScoreDesc },
                objectType: "Api",
            }) : { nodes: [] };
            const { nodes: notes } = shouldInclude("Note") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.notes,
                input: { ...input, take, sortBy: NoteSortBy.ScoreDesc },
                objectType: "Note",
            }) : { nodes: [] };
            const { nodes: organizations } = shouldInclude("Organization") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.organizations,
                input: { ...input, take, sortBy: OrganizationSortBy.BookmarksDesc },
                objectType: "Organization",
            }) : { nodes: [] };
            const { nodes: projects } = shouldInclude("Project") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.projects,
                input: { ...input, take, sortBy: ProjectSortBy.ScoreDesc, isComplete: true },
                objectType: "Project",
            }) : { nodes: [] };
            const { nodes: questions } = shouldInclude("Question") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, api: null, note: null, organization: null, project: null, routine: null, smartContract: null, standard: null },
                info: partial.questions,
                input: { ...input, take, sortBy: QuestionSortBy.ScoreDesc, isComplete: true },
                objectType: "Question",
            }) : { nodes: [] };
            const { nodes: routines } = shouldInclude("Routine") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.routines,
                input: { ...input, take, sortBy: RoutineSortBy.ScoreDesc, isComplete: true, isInternal: false },
                objectType: "Routine",
            }) : { nodes: [] };
            const { nodes: smartContracts } = shouldInclude("SmartContract") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.smartContracts,
                input: { ...input, take, sortBy: SmartContractSortBy.ScoreDesc, isComplete: true },
                objectType: "SmartContract",
            }) : { nodes: [] };
            const { nodes: standards } = shouldInclude("Standard") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery, isPrivate: false },
                info: partial.standards,
                input: { ...input, take, sortBy: StandardSortBy.ScoreDesc, type: "JSON" },
                objectType: "Standard",
            }) : { nodes: [] };
            const { nodes: users } = shouldInclude("User") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { ...bookmarksQuery },
                info: partial.users,
                input: { ...input, take, sortBy: UserSortBy.BookmarksDesc },
                objectType: "User",
            }) : { nodes: [] };
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
            }, partial, prisma, getUser(req));
            return {
                __typename: "PopularResult",
                ...withSupplemental,
            };
        },
    },
};
//# sourceMappingURL=feed.js.map