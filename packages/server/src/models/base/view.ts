import { type Prisma } from "@prisma/client";
import { type Count, DEFAULT_LANGUAGE, HOURS_1_MS, MaxObjects, type ModelType, type SessionUser, ViewFor, ViewSortBy, generatePK, lowercaseFirstLetter } from "@vrooli/shared";
import i18next from "i18next";
import { onlyValidIds } from "../../builders/onlyValid.js";
import { type PrismaDelegate } from "../../builders/types.js";
import { useVisibility, useVisibilityMapper } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { getLabels } from "../../getters/getLabels.js";
import { CacheService } from "../../redisConn.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { ViewFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type ViewModelLogic } from "./types.js";

function toWhere(key: string, nestedKey: string | null, id: string) {
    if (nestedKey) return { [key]: { [nestedKey]: { some: { id } } } };
    return { [key]: { id: BigInt(id) } };
}

function toSelect(key?: string) {
    if (key) return { [key]: { select: { id: true, views: true } } };
    return { id: true, views: true };
}

function toData(object: object, key?: string) {
    if (key) return object[key];
    return object;
}

function toCreate(object: object, relName: string, key?: string) {
    if (key) return { [relName]: { connect: { id: BigInt(object[key].id) } } };
    return { [relName]: { connect: { id: BigInt((object as { id: string }).id) } } };
}

/**
 * Maps ViewFor types to partial query objects, used to determine 
 * if a view exists for a given object.
 */
const whereMapper = {
    Resource: (id: string) => toWhere("resource", null, id),
    ResourceVersion: (id: string) => toWhere("resource", "versions", id),
    Team: (id: string) => toWhere("team", null, id),
    User: (id: string) => toWhere("user", null, id),
} as const;

/**
 * Maps ViewFor types to partial query objects, used to find data required to create a view.
 */
const selectMapper = {
    Resource: toSelect(),
    ResourceVersion: toSelect("root"),
    User: toSelect(),
} as const;

/**
 * Maps object with selectMapper data to its corresponding id
 */
const dataMapper = {
    Resource: (object: object) => toData(object),
    ResourceVersion: (object: object) => toData(object, "root"),
    Team: (object: object) => toData(object),
    User: (object: object) => toData(object),
};

/**
 * Maps object with selectMapper data to a Prisma data object, to create a view.
 */
const createMapper = {
    Resource: (object: object) => toCreate(object, "resource"),
    ResourceVersion: (object: object) => toCreate(object, "resource", "root"),
    Team: (object: object) => toCreate(object, "team"),
    User: (object: object) => toCreate(object, "user"),
};

interface ViewInput {
    forId: string;
    viewFor: ViewFor;
}

/**
 * Deletes views from user's view list, but does not affect view count or logs.
 */
async function deleteViews(userId: string, ids: string[]): Promise<Count> {
    return await DbProvider.get().view.deleteMany({
        where: {
            AND: [
                { id: { in: ids.map(id => BigInt(id)) } },
                { byId: BigInt(userId) },
            ],
        },
    }).then(({ count }) => ({ __typename: "Count" as const, count }));
}

/**
 * Removes all of user's views, but does not affect view count or logs.
 */
async function clearViews(userId: string): Promise<Count> {
    return await DbProvider.get().view.deleteMany({
        where: { byId: BigInt(userId) },
    }).then(({ count }) => ({ __typename: "Count" as const, count }));
}

const displayMapper: { [key in ViewFor]?: keyof Prisma.viewUpsertArgs["create"] } = {
    Resource: "resource",
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
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(displayMapper)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:View", { lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE, count: 1 });
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
                ...Object.entries(displayMapper).map(([key, value]) => ({ [value]: ModelMap.getLogic(["search"], key as ModelType).search.searchStringQuery() })),
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
            const isViewedArray = await DbProvider.get().view.findMany({
                where: {
                    byId: BigInt(userId),
                    [fieldName]: { in: idsFiltered.map(id => BigInt(id)) },
                },
            });
            // Replace the nulls in the result array with true if viewed
            for (let i = 0; i < ids.length; i++) {
                // Try to find this id in the isViewed array
                result[i] = Boolean(isViewedArray.find((view) => view[fieldName].toString() === ids[i]));
            }
            return result;
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (_data, _getParentInfo?) => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => ({
            User: data?.by,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            by: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    by: { id: BigInt(data.userId) },
                    // Any non-public, non-owned objects should be filtered out
                    // Can use OR because only one relation will be present
                    OR: [
                        ...useVisibilityMapper("OwnOrPublic", data, displayMapper, false),
                    ],
                };
            },
            // Not useful for this object type
            ownOrPublic: null,
            // Not useful for this object type
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("View", "Own", data);
            },
            // Not useful for this object type
            ownPublic: function getOwnPublic(data) {
                return useVisibility("View", "Own", data);
            },
            public: function getPublic(data) {
                return {
                    // Can use OR because only one relation will be present
                    OR: [
                        ...useVisibilityMapper("Public", data, displayMapper, false),
                    ],
                };
            },
        },
    }),
    /**
     * Marks objects as viewed. If view exists, updates last viewed time.
     * A user may view their own objects, but it does not count towards its view count.
     * @returns True if view updated correctly
     */
    view: async (userData: SessionUser, input: ViewInput): Promise<boolean> => {
        // Get db table for viewed object
        const { dbTable } = ModelMap.getLogic(["dbTable"], input.viewFor, true, "view 1");
        // Check if object being viewed on exists
        const objectToView: { [x: string]: any } | null = await (DbProvider.get()[dbTable] as PrismaDelegate).findUnique({
            where: { id: BigInt(input.forId) },
            select: selectMapper[input.viewFor],
        });
        if (!objectToView)
            throw new CustomError("0173", "NotFound");
        // Check if view exists
        let view = await DbProvider.get().view.findFirst({
            where: {
                by: { id: BigInt(userData.id) },
                ...whereMapper[input.viewFor](input.forId),
            },
        });
        // If view already existed, update view time
        if (view) {
            await DbProvider.get().view.update({
                where: { id: view.id },
                data: {
                    lastViewedAt: new Date(),
                },
            });
        }
        // If view did not exist, create it
        else {
            const labels = await getLabels([input.forId], input.viewFor, userData.languages, "view");
            view = await DbProvider.get().view.create({
                data: {
                    id: generatePK(),
                    by: { connect: { id: BigInt(userData.id) } },
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
                const adminData = await DbProvider.get().team.findFirst({
                    where: {
                        id: BigInt(input.forId),
                        members: { some: { isAdmin: true, user: { id: BigInt(userData.id) } } },
                    },
                });
                isOwn = Boolean(adminData);
                break;
            }
            case ViewFor.Resource:
            case ViewFor.ResourceVersion: {
                // Check if ROOT object is owned by this user or by a team they are a member of
                const { dbTable: rootDbTable } = ModelMap.getLogic(["dbTable"], input.viewFor.replace("Version", "") as ModelType, true, "view 2");
                const rootObject = await (DbProvider.get()[rootDbTable] as PrismaDelegate).findFirst({
                    where: {
                        AND: [
                            { id: BigInt(dataMapper[input.viewFor](objectToView).id) },
                            {
                                OR: [
                                    { ownedByTeam: { members: { some: { user: { id: BigInt(userData.id) } } } } },
                                    { ownedByUser: { id: BigInt(userData.id) } },
                                ],
                            },
                        ],
                    },
                });
                if (rootObject) isOwn = true;
                break;
            }
            case ViewFor.User:
                isOwn = userData.id === input.forId;
                break;
        }
        // If user is owner, don't do anything else
        if (isOwn) return true;
        // Update view count and cache last viewed time using CacheService
        const cacheService = CacheService.get();
        const redisKey = `view:${userData.id}_${dataMapper[input.viewFor](objectToView).id}_${input.viewFor}`;
        const lastViewed = await cacheService.get<string>(redisKey);
        if (!lastViewed || new Date(lastViewed).getTime() < Date.now() - HOURS_1_MS) {
            const { dbTable: rootDbTable } = ModelMap.getLogic(["dbTable"], input.viewFor.replace("Version", "") as ModelType, true, "view 3");
            await (DbProvider.get()[rootDbTable] as PrismaDelegate).update({
                where: { id: BigInt(input.forId) },
                data: { views: dataMapper[input.viewFor](objectToView).views + 1 },
            });
        }
        await cacheService.set(redisKey, new Date().toISOString());
        return true;
    },
    deleteViews,
    clearViews,
});
