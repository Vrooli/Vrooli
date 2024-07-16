import { Count, GqlModelType, lowercaseFirstLetter, ViewFor, ViewSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { ModelMap } from ".";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { PrismaDelegate } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { getLabels } from "../../getters/getLabels";
import { withRedis } from "../../redisConn";
import { SessionUserToken } from "../../types";
import { defaultPermissions } from "../../utils/defaultPermissions";
import { ViewFormat } from "../formats";
import { TeamModelLogic, ViewModelLogic } from "./types";

const toWhere = (key: string, nestedKey: string | null, id: string) => {
    if (nestedKey) return { [key]: { [nestedKey]: { some: { id } } } };
    return { [key]: { id } };
};

const toSelect = (key?: string) => {
    if (key) return { [key]: { select: { id: true, views: true } } };
    return { id: true, views: true };
};

const toData = (object: any, key?: string) => {
    if (key) return object[key];
    return object;
};

const toCreate = (object: any, relName: string, key?: string) => {
    if (key) return { [relName]: { connect: { id: object[key].id } } };
    return { [relName]: { connect: { id: object.id } } };
};

/**
 * Maps ViewFor types to partial query objects, used to determine 
 * if a view exists for a given object.
 */
const whereMapper = {
    Api: (id: string) => toWhere("api", null, id),
    ApiVersion: (id: string) => toWhere("api", "versions", id),
    Code: (id: string) => toWhere("code", null, id),
    CodeVersion: (id: string) => toWhere("code", "versions", id),
    Note: (id: string) => toWhere("note", null, id),
    NoteVersion: (id: string) => toWhere("note", "versions", id),
    Project: (id: string) => toWhere("project", null, id),
    ProjectVersion: (id: string) => toWhere("project", "versions", id),
    Question: (id: string) => toWhere("question", null, id),
    Routine: (id: string) => toWhere("routine", null, id),
    RoutineVersion: (id: string) => toWhere("routine", "versions", id),
    Standard: (id: string) => toWhere("standard", null, id),
    StandardVersion: (id: string) => toWhere("standard", "versions", id),
    Team: (id: string) => toWhere("team", null, id),
    User: (id: string) => toWhere("user", null, id),
} as const;

/**
 * Maps ViewFor types to partial query objects, used to find data required to create a view.
 */
const selectMapper = {
    Api: toSelect(),
    ApiVersion: toSelect("root"),
    Code: toSelect(),
    CodeVersion: toSelect("root"),
    Note: toSelect(),
    NoteVersion: toSelect("root"),
    Project: toSelect(),
    ProjectVersion: toSelect("root"),
    Question: toSelect(),
    Routine: toSelect(),
    RoutineVersion: toSelect("root"),
    Standard: toSelect(),
    StandardVersion: toSelect("root"),
    Team: toSelect(),
    User: toSelect(),
} as const;

/**
 * Maps object with selectMapper data to its corresponding id
 */
const dataMapper = {
    Api: (object: any) => toData(object),
    ApiVersion: (object: any) => toData(object, "root"),
    Code: (object: any) => toData(object),
    CodeVersion: (object: any) => toData(object, "root"),
    Note: (object: any) => toData(object),
    NoteVersion: (object: any) => toData(object, "root"),
    Project: (object: any) => toData(object),
    ProjectVersion: (object: any) => toData(object, "root"),
    Question: (object: any) => toData(object),
    Routine: (object: any) => toData(object),
    RoutineVersion: (object: any) => toData(object, "root"),
    Standard: (object: any) => toData(object),
    StandardVersion: (object: any) => toData(object, "root"),
    Team: (object: any) => toData(object),
    User: (object: any) => toData(object),
};

/**
 * Maps object with selectMapper data to a Prisma data object, to create a view.
 */
const createMapper = {
    Api: (object: any) => toCreate(object, "api"),
    ApiVersion: (object: any) => toCreate(object, "api", "root"),
    Code: (object: any) => toCreate(object, "code"),
    CodeVersion: (object: any) => toCreate(object, "code", "root"),
    Note: (object: any) => toCreate(object, "note"),
    NoteVersion: (object: any) => toCreate(object, "note", "root"),
    Project: (object: any) => toCreate(object, "project"),
    ProjectVersion: (object: any) => toCreate(object, "project", "root"),
    Question: (object: any) => toCreate(object, "question"),
    Routine: (object: any) => toCreate(object, "routine"),
    RoutineVersion: (object: any) => toCreate(object, "routine", "root"),
    Standard: (object: any) => toCreate(object, "standard"),
    StandardVersion: (object: any) => toCreate(object, "standard", "root"),
    Team: (object: any) => toCreate(object, "team"),
    User: (object: any) => toCreate(object, "user"),
};

interface ViewInput {
    forId: string;
    viewFor: ViewFor;
}

/**
 * Deletes views from user's view list, but does not affect view count or logs.
 */
const deleteViews = async (userId: string, ids: string[]): Promise<Count> => {
    return await prismaInstance.view.deleteMany({
        where: {
            AND: [
                { id: { in: ids } },
                { byId: userId },
            ],
        },
    }).then(({ count }) => ({ __typename: "Count" as const, count }));
};

/**
 * Removes all of user's views, but does not affect view count or logs.
 */
const clearViews = async (userId: string): Promise<Count> => {
    return await prismaInstance.view.deleteMany({
        where: { byId: userId },
    }).then(({ count }) => ({ __typename: "Count" as const, count }));
};

const displayMapper: { [key in ViewFor]?: keyof Prisma.viewUpsertArgs["create"] } = {
    Api: "api",
    Code: "code",
    Question: "question",
    Note: "note",
    Post: "post",
    Project: "project",
    Routine: "routine",
    Standard: "standard",
    Team: "team",
    User: "user",
};

const __typename = "View" as const;
export const ViewModel: ViewModelLogic = ({
    __typename,
    dbTable: "view",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(displayMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(displayMapper)) {
                    if (select[value]) return ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:View", { lng: languages[0], count: 1 });
            },
        },
    }),
    format: ViewFormat,
    search: {
        defaultSort: ViewSortBy.LastViewedDesc,
        sortBy: ViewSortBy,
        searchFields: {
            lastViewedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                ...Object.entries(displayMapper).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as GqlModelType).search.searchStringQuery() })),
            ],
        }),
    },
    query: {
        async getIsVieweds(
            userId: string | null | undefined,
            ids: string[],
            viewFor: keyof typeof ViewFor,
        ): Promise<Array<boolean | null>> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(false);
            // If userId not provided, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${lowercaseFirstLetter(viewFor)}Id`;
            const isViewedArray = await prismaInstance.view.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
            // Replace the nulls in the result array with true if viewed
            for (let i = 0; i < ids.length; i++) {
                // Try to find this id in the isViewed array
                result[i] = Boolean(isViewedArray.find((view: any) => view[fieldName] === ids[i]));
            }
            return result;
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: 10000000,
        owner: (data) => ({
            User: data?.by,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            by: "User",
        }),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    OR: [
                        ...Object.entries(displayMapper).map(([key, value]) => ({ [value]: ModelMap.get(key as GqlModelType).validate().visibility.private(...params) })),
                    ],
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    // Can use OR because only one relation will be present
                    OR: [
                        ...Object.entries(displayMapper).map(([key, value]) => ({ [value]: ModelMap.get(key as GqlModelType).validate().visibility.public(...params) })),
                    ],
                };
            },
            owner: (userId) => ({
                by: { id: userId },
            }),
        },
    }),
    /**
     * Marks objects as viewed. If view exists, updates last viewed time.
     * A user may view their own objects, but it does not count towards its view count.
     * @returns True if view updated correctly
     */
    view: async (userData: SessionUserToken, input: ViewInput): Promise<boolean> => {
        // Get db table for viewed object
        const { dbTable } = ModelMap.getLogic(["dbTable"], input.viewFor, true, "view 1");
        // Check if object being viewed on exists
        const objectToView: { [x: string]: any } | null = await (prismaInstance[dbTable] as PrismaDelegate).findUnique({
            where: { id: input.forId },
            select: selectMapper[input.viewFor],
        });
        if (!objectToView)
            throw new CustomError("0173", "NotFound", userData.languages);
        // Check if view exists
        let view = await prismaInstance.view.findFirst({
            where: {
                by: { id: userData.id },
                ...whereMapper[input.viewFor](input.forId),
            },
        });
        // If view already existed, update view time
        if (view) {
            await prismaInstance.view.update({
                where: { id: view.id },
                data: {
                    lastViewedAt: new Date(),
                },
            });
        }
        // If view did not exist, create it
        else {
            const labels = await getLabels([{ id: input.forId, languages: userData.languages }], input.viewFor, userData.languages, "view");
            view = await prismaInstance.view.create({
                data: {
                    by: { connect: { id: userData.id } },
                    name: labels[0],
                    ...createMapper[input.viewFor](objectToView),
                },
            });
        }
        // Check if a view from this user should increment the view count
        let isOwn = false;
        switch (input.viewFor) {
            case ViewFor.Team: {
                // Check if user is an admin or owner of the team
                const roles = await ModelMap.get<TeamModelLogic>("Team").query.hasRole(userData.id, [input.forId]);
                isOwn = Boolean(roles[0]);
                break;
            }
            case ViewFor.Api:
            case ViewFor.ApiVersion:
            case ViewFor.Code:
            case ViewFor.CodeVersion:
            case ViewFor.Note:
            case ViewFor.NoteVersion:
            case ViewFor.Project:
            case ViewFor.ProjectVersion:
            case ViewFor.Routine:
            case ViewFor.RoutineVersion:
            case ViewFor.Standard:
            case ViewFor.StandardVersion: {
                // Check if ROOT object is owned by this user or by a team they are a member of
                const { dbTable: rootDbTable } = ModelMap.getLogic(["dbTable"], input.viewFor.replace("Version", "") as GqlModelType, true, "view 2");
                const rootObject = await (prismaInstance[rootDbTable] as PrismaDelegate).findFirst({
                    where: {
                        AND: [
                            { id: dataMapper[input.viewFor](objectToView).id },
                            {
                                OR: [
                                    ModelMap.get<TeamModelLogic>("Team").query.isMemberOfTeamQuery(userData.id),
                                    { ownedByUser: { id: userData.id } },
                                ],
                            },
                        ],
                    },
                });
                if (rootObject) isOwn = true;
                break;
            }
            case ViewFor.Question: {
                // Check if question was created by this user
                const question = await prismaInstance.question.findFirst({ where: { id: input.forId, createdBy: { id: userData.id } } });
                if (question) isOwn = true;
                break;
            }
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Update view count
        await withRedis({
            process: async (redisClient) => {
                // Don't update view count when redis is unavailable
                if (!redisClient) return;
                // Check the last time the user viewed this object
                const redisKey = `view:${userData.id}_${dataMapper[input.viewFor](objectToView).id}_${input.viewFor}`;
                const lastViewed = await redisClient.get(redisKey);
                // If object viewed more than 1 hour ago, update view count
                if (!lastViewed || new Date(lastViewed).getTime() < new Date().getTime() - 3600000) {
                    // View counts don't exist on versioned objects, so we must make sure we are updating the root object
                    const { dbTable: rootDbTable } = ModelMap.getLogic(["dbTable"], input.viewFor.replace("Version", "") as GqlModelType, true, "view 3");
                    await (prismaInstance[rootDbTable] as PrismaDelegate).update({
                        where: { id: input.forId },
                        data: { views: dataMapper[input.viewFor](objectToView).views + 1 },
                    });
                }
                // Update last viewed time
                await redisClient.set(redisKey, new Date().toISOString());
            },
            trace: "0513",
        });
        return true;
    },
    deleteViews,
    clearViews,
});
