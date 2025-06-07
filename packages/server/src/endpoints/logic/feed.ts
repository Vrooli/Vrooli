import { type HomeResult, type PageInfo, type Popular, type PopularSearchInput, type PopularSearchResult, ReminderSortBy, ResourceSortBy, ScheduleSortBy, TeamSortBy, UserSortBy, VisibilityType } from "@vrooli/shared";
import { readManyAsFeedHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { InfoConverter, addSupplementalFieldsMultiTypes } from "../../builders/infoConverter.js";
import { type PartialApiInfo } from "../../builders/types.js";
import { schedulesWhereInTimeframe } from "../../events/schedule.js";
import { type ApiEndpoint } from "../../types.js";

const SCHEDULES_DAYS_TO_LOOK_AHEAD = 7;

export type EndpointsFeed = {
    home: ApiEndpoint<never, HomeResult>;
    popular: ApiEndpoint<PopularSearchInput, PopularSearchResult>;
}

export const feed: EndpointsFeed = {
    home: async (_i, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 5000, req });
        const partial = InfoConverter.get().fromApiToPartialApi(info, {
            __typename: "HomeResult",
            reminders: "Reminder",
            resources: "Resource",
            schedules: "Schedule",
        }, true);
        const take = 10;
        const commonReadParams = { req };
        // Query reminders
        const { nodes: reminders } = await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.reminders as PartialApiInfo,
            input: { take, sortBy: ReminderSortBy.DateCreatedAsc, isComplete: false, visibility: VisibilityType.Own },
            objectType: "Reminder",
        });
        // Query resources
        const { nodes: resources } = await readManyAsFeedHelper({
            ...commonReadParams,
            info: partial.resources as PartialApiInfo,
            input: { take, visibility: VisibilityType.Own },
            objectType: "Resource",
        });
        // Query schedules that might occur in the next 7 days. 
        // Need to perform calculations on the client side to determine which ones are actually relevant.
        const now = new Date();
        const startDate = now;
        const endDate = new Date(now.setDate(now.getDate() + SCHEDULES_DAYS_TO_LOOK_AHEAD));
        const { nodes: schedules } = await readManyAsFeedHelper({
            ...commonReadParams,
            additionalQueries: {
                ...schedulesWhereInTimeframe(startDate, endDate),
            },
            info: partial.schedules as PartialApiInfo,
            input: { take, sortBy: ScheduleSortBy.EndTimeAsc, visibility: VisibilityType.Own },
            objectType: "Schedule",
        });
        // Add supplemental fields to every result
        const withSupplemental = await addSupplementalFieldsMultiTypes({
            reminders,
            resources,
            schedules,
        }, partial as any, SessionService.getUser(req));
        // Return results
        return {
            __typename: "HomeResult" as const,
            ...withSupplemental,
        };
    },
    popular: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 5000, req });
        const partial = InfoConverter.get().fromApiToPartialApi(info, {
            __typename: "PopularResult",
            Resource: "Resource",
            Team: "Team",
            User: "User",
        }, true);
        // Split take between each requested object type, favoring resources because they have more variations.
        const totalTake = 50;
        const resourceTake = input.objectType ? Math.ceil(totalTake * 4 / 6) : totalTake;
        const teamTake = input.objectType ? Math.ceil(totalTake * 1 / 6) : totalTake;
        const userTake = input.objectType ? Math.ceil(totalTake * 1 / 6) : totalTake;
        const commonReadParams = { req };
        const commonInputParams = {
            createdTimeFrame: input.createdTimeFrame,
            updatedTimeFrame: input.updatedTimeFrame,
            visibility: input.visibility,
        };
        // Query resources
        let resources: object[] = [];
        let resourcePageInfo: Partial<PageInfo> = {};
        if (!input.objectType || input.objectType === "Resource") {
            const { nodes, pageInfo } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isDeleted: false, isInternal: false, isPrivate: false },
                info: partial.Resource as PartialApiInfo,
                input: {
                    ...commonInputParams,
                    after: input.resourceAfter,
                    sortBy: input.sortBy ?? ResourceSortBy.ScoreDesc,
                    take: resourceTake,
                },
                objectType: "Resource",
            });
            resources = nodes;
            resourcePageInfo = pageInfo;
        }
        // Query teams
        let teams: object[] = [];
        let teamPageInfo: Partial<PageInfo> = {};
        if (!input.objectType || input.objectType === "Team") {
            const { nodes, pageInfo } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.Team as PartialApiInfo,
                input: {
                    ...commonInputParams,
                    after: input.teamAfter,
                    sortBy: input.sortBy ?? TeamSortBy.BookmarksDesc,
                    take: teamTake,
                },
                objectType: "Team",
            });
            teams = nodes;
            teamPageInfo = pageInfo;
        }
        // Query users
        let users: object[] = [];
        let userPageInfo: Partial<PageInfo> = {};
        if (!input.objectType || input.objectType === "User") {
            const { nodes, pageInfo } = await readManyAsFeedHelper({
                ...commonReadParams,
                additionalQueries: { isPrivate: false },
                info: partial.User as PartialApiInfo,
                input: {
                    ...commonInputParams,
                    after: input.userAfter,
                    sortBy: input.sortBy ?? UserSortBy.DateUpdatedDesc,
                    take: userTake,
                },
                objectType: "User",
            });
            users = nodes;
            userPageInfo = pageInfo;
        }
        // Add supplemental fields to every result
        const withSupplemental = await addSupplementalFieldsMultiTypes({
            resources,
            teams,
            users,
        }, {
            resources: { type: "Resource", ...(partial.Resource as PartialApiInfo) },
            teams: { type: "Team", ...(partial.Team as PartialApiInfo) },
            users: { type: "User", ...(partial.User as PartialApiInfo) },
        }, SessionService.getUser(req));
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
                hasNextPage:
                    resourcePageInfo.hasNextPage
                    || teamPageInfo.hasNextPage
                    || userPageInfo.hasNextPage
                    || false,
                endCursorResource: resourcePageInfo.endCursor ?? "",
                endCursorTeam: teamPageInfo.endCursor ?? "",
                endCursorUser: userPageInfo.endCursor ?? "",
            },
            edges: nodes.map((node) => ({ __typename: "PopularEdge" as const, cursor: node.id, node })),
        };
        return combined;
    },
};
