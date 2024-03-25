import { ApiSortBy, FocusModeStopCondition, HomeInput, HomeResult, NoteSortBy, OrganizationSortBy, PageInfo, Popular, PopularSearchInput, PopularSearchResult, ProjectSortBy, QuestionSortBy, ReminderSortBy, ResourceSortBy, RoutineSortBy, ScheduleSortBy, SmartContractSortBy, StandardSortBy, UserSortBy, VisibilityType } from "@local/shared";
import { GraphQLResolveInfo } from "graphql";
import { readManyAsFeedHelper } from "../../actions/reads";
import { assertRequestFrom, getUser } from "../../auth/request";
import { addSupplementalFieldsMultiTypes } from "../../builders/addSupplementalFieldsMultiTypes";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { PartialGraphQLInfo } from "../../builders/types";
import { schedulesWhereInTimeframe } from "../../events/schedule";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";
import { FocusModeEndpoints } from "./focusMode";

export type EndpointsFeed = {
    Query: {
        home: GQLEndpoint<HomeInput, HomeResult>;
        popular: GQLEndpoint<PopularSearchInput, PopularSearchResult>;
    }
}

export const FeedEndpoints: EndpointsFeed = {
    Query: {
        home: async (_, { input }, { req, res }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 5000, req });
            // Find focus mode to use
            const activeFocusModeId = input.activeFocusModeId ?? userData.activeFocusMode?.mode?.id;
            // If input provided the focus mode, update session
            if (input.activeFocusModeId) {
                FocusModeEndpoints.Mutation.setActiveFocusMode(_,
                    { input: { id: input.activeFocusModeId, stopCondition: FocusModeStopCondition.NextBegins } },
                    { req, res },
                    {} as unknown as GraphQLResolveInfo,
                );
            }
            const partial = toPartialGqlInfo(info, {
                __typename: "HomeResult",
                recommended: "Resource",
                reminders: "Reminder",
                resources: "Resource",
                schedules: "Schedule",
            }, req.session.languages, true);
            const take = 10;
            const commonReadParams = { req };
            // Query recommended TODO
            const recommended: object[] = [];
            // Query reminders
            const { nodes: reminders } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    reminderList: {
                        focusMode: {
                            ...(activeFocusModeId ? { id: activeFocusModeId } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.reminders as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ReminderSortBy.DateCreatedAsc, isComplete: false, visibility: VisibilityType.Public }, // VisibilityType.Own clashes with additionalQueries
                objectType: "Reminder",
            });
            // Query resources
            const { nodes: resources } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: {
                    list: {
                        focusMode: {
                            ...(activeFocusModeId ? { id: activeFocusModeId } : { user: { id: userData.id } }),
                        },
                    },
                },
                info: partial.resources as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ResourceSortBy.IndexAsc, visibility: VisibilityType.Public }, // VisibilityType.Own clashes with additionalQueries
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
                },
                info: partial.schedules as PartialGraphQLInfo,
                input: { ...input, take, sortBy: ScheduleSortBy.EndTimeAsc, visibility: VisibilityType.Own },
                objectType: "Schedule",
            });
            // Add supplemental fields to every result
            const withSupplemental = await addSupplementalFieldsMultiTypes({
                recommended,
                reminders,
                resources,
                schedules,
            }, partial as any, getUser(req.session));
            // Return results
            return {
                __typename: "HomeResult" as const,
                ...withSupplemental,
            };
        },
        popular: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 5000, req });
            const partial = toPartialGqlInfo(info, {
                __typename: "PopularResult",
                Api: "Api",
                Note: "Note",
                Organization: "Organization",
                Project: "Project",
                Question: "Question",
                Routine: "Routine",
                SmartContract: "SmartContract",
                Standard: "Standard",
                User: "User",
            }, req.session.languages, true);
            // If any "after" cursor is provided, we can assume that missing cursors mean that we've reached the end for that object type
            const aftersCount = Object.entries(input).filter(([key, value]) => key.endsWith("After") && typeof value === "string" && value.trim() !== "").length;
            const anyAfters = aftersCount > 0;
            // Checks if object type should be included in results
            const shouldInclude = (objectType: `${PopularSearchInput["objectType"]}`) => {
                if (anyAfters && (input[`${objectType.toLowerCase()}After`]?.trim() ?? "") === "") return false;
                return input.objectType ? input.objectType === objectType : true;
            };
            // Split take between each requested object type.
            // If input.objectType is provided, take stays the same (as it's only querying one object type).
            // If there are any "after" cursors, take is split evenly between each object type.
            // If neither of the above are true, take is split evenly between each object type
            const totalTake = 25;
            const take = input.objectType ? totalTake : anyAfters ? Math.ceil(totalTake / aftersCount) : Math.ceil(totalTake / 9);
            const commonReadParams = { req };
            const commonInputParams = {
                createdTimeFrame: input.createdTimeFrame,
                take,
                updatedTimeFrame: input.updatedTimeFrame,
                visibility: input.visibility,
            };
            // Query apis
            const { nodes: apis, pageInfo: apisInfo } = shouldInclude("Api") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Api as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.apiAfter,
                    sortBy: (input.sortBy ?? ApiSortBy.ScoreDesc) as ApiSortBy,
                },
                objectType: "Api",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query notes
            const { nodes: notes, pageInfo: notesInfo } = shouldInclude("Note") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Note as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.noteAfter,
                    sortBy: (input.sortBy ?? NoteSortBy.DateUpdatedDesc) as NoteSortBy,
                },
                objectType: "Note",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query organizations
            const { nodes: organizations, pageInfo: organizationsInfo } = shouldInclude("Organization") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Organization as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.organizationAfter,
                    sortBy: (input.sortBy ?? OrganizationSortBy.BookmarksDesc) as OrganizationSortBy,
                },
                objectType: "Organization",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query projects
            const { nodes: projects, pageInfo: projectsInfo } = shouldInclude("Project") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Project as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.projectAfter,
                    sortBy: (input.sortBy ?? ProjectSortBy.BookmarksDesc) as ProjectSortBy,
                },
                objectType: "Project",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query questions
            const { nodes: questions, pageInfo: questionsInfo } = shouldInclude("Question") ? await readManyAsFeedHelper({
                ...commonReadParams,
                // Make sure question is not attached to any objects (i.e. standalone)
                additionalQueries: { api: null, note: null, organization: null, project: null, routine: null, smartContract: null, standard: null },
                info: partial.Question as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.questionAfter,
                    sortBy: (input.sortBy ?? QuestionSortBy.BookmarksDesc) as QuestionSortBy,
                },
                objectType: "Question",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query routines
            const { nodes: routines, pageInfo: routinesInfo } = shouldInclude("Routine") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Routine as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.routineAfter,
                    sortBy: (input.sortBy ?? RoutineSortBy.ScoreDesc) as RoutineSortBy,
                    isInternal: false,
                },
                objectType: "Routine",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query smart contracts
            const { nodes: smartContracts, pageInfo: smartContractsInfo } = shouldInclude("SmartContract") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.SmartContract as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.smartContractAfter,
                    sortBy: (input.sortBy ?? SmartContractSortBy.ScoreDesc) as SmartContractSortBy,
                },
                objectType: "SmartContract",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query standards
            const { nodes: standards, pageInfo: standardsInfo } = shouldInclude("Standard") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Standard as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.standardAfter,
                    sortBy: (input.sortBy ?? StandardSortBy.BookmarksDesc) as StandardSortBy,
                    isInternal: false,
                    type: "JSON",
                },
                objectType: "Standard",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
            // Query users
            const { nodes: users, pageInfo: usersInfo } = shouldInclude("User") ? await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.User as PartialGraphQLInfo,
                input: {
                    ...commonInputParams,
                    after: input.userAfter,
                    sortBy: (input.sortBy ?? UserSortBy.DateUpdatedDesc) as UserSortBy,
                },
                objectType: "User",
            }) : { nodes: [], pageInfo: {} } as { nodes: object[], pageInfo: Partial<PageInfo> };
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
            }, {
                apis: { type: "Api", ...(partial.Api as PartialGraphQLInfo) },
                notes: { type: "Note", ...(partial.Note as PartialGraphQLInfo) },
                organizations: { type: "Organization", ...(partial.Organization as PartialGraphQLInfo) },
                projects: { type: "Project", ...(partial.Project as PartialGraphQLInfo) },
                questions: { type: "Question", ...(partial.Question as PartialGraphQLInfo) },
                routines: { type: "Routine", ...(partial.Routine as PartialGraphQLInfo) },
                smartContracts: { type: "SmartContract", ...(partial.SmartContract as PartialGraphQLInfo) },
                standards: { type: "Standard", ...(partial.Standard as PartialGraphQLInfo) },
                users: { type: "User", ...(partial.User as PartialGraphQLInfo) },
            }, getUser(req.session));
            // Combine nodes, alternating between each type
            const properties = Object.values(withSupplemental);
            const maxLen = Math.max(...properties.map(arr => arr.length));
            const nodes: Popular[] = Array.from({ length: maxLen })
                .flatMap((_, i) => properties.map(prop => prop[i]))
                .filter(Boolean);
            // Combine pageInfo
            const combined: PopularSearchResult = {
                __typename: "PopularSearchResult" as const,
                pageInfo: {
                    __typename: "PopularPageInfo" as const,
                    hasNextPage: apisInfo.hasNextPage || notesInfo.hasNextPage || organizationsInfo.hasNextPage || projectsInfo.hasNextPage || questionsInfo.hasNextPage || routinesInfo.hasNextPage || smartContractsInfo.hasNextPage || standardsInfo.hasNextPage || usersInfo.hasNextPage || false,
                    endCursorApi: apisInfo.endCursor ?? "",
                    endCursorNote: notesInfo.endCursor ?? "",
                    endCursorOrganization: organizationsInfo.endCursor ?? "",
                    endCursorProject: projectsInfo.endCursor ?? "",
                    endCursorQuestion: questionsInfo.endCursor ?? "",
                    endCursorRoutine: routinesInfo.endCursor ?? "",
                    endCursorSmartContract: smartContractsInfo.endCursor ?? "",
                    endCursorStandard: standardsInfo.endCursor ?? "",
                    endCursorUser: usersInfo.endCursor ?? "",
                },
                edges: nodes.map((node) => ({ __typename: "PopularEdge" as const, cursor: node.id, node })),
            };
            return combined;
        },
    },
};
