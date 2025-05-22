import { apiKeyValidation, generatePK, MaxObjects } from "@local/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CacheService } from "../../redisConn.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ApiKeyFormat } from "../formats.js";
import { type ApiKeyModelLogic } from "./types.js";

const __typename = "ApiKey" as const;
export const ApiKeyModel: ApiKeyModelLogic = ({
    __typename,
    dbTable: "api_key",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => {
                return select.name;
            },
        },
    }),
    format: ApiKeyFormat,
    mutate: {
        shape: {
            create: async ({ userData, data }) => ({
                id: generatePK(),
                creditsUsedBeforeLimit: 0,
                disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                limitHard: BigInt(data.limitHard),
                limitSoft: data.limitSoft ? BigInt(data.limitSoft) : null,
                key: ApiKeyEncryptionService.generateSiteKey(),
                name: data.name,
                permissions: data.permissions,
                stopAtLimit: data.stopAtLimit,
                team: data.teamConnect ? { connect: { id: BigInt(data.teamConnect) } } : undefined,
                user: data.teamConnect ? undefined : { connect: { id: BigInt(userData.id) } },
            }),
            update: async ({ data }) => ({
                disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                limitHard: data.limitHard ? BigInt(data.limitHard) : undefined,
                limitSoft: data.limitSoft ? BigInt(data.limitSoft) : data.limitSoft === null ? null : undefined,
                name: noNull(data.name),
                permissions: noNull(data.permissions),
                stopAtLimit: noNull(data.stopAtLimit),
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, updatedIds, deletedIds }) => {
                const ids = [...createdIds, ...updatedIds, ...deletedIds];
                if (ids.length === 0) return;
                // Fetch the key strings for these API key IDs
                const records = await DbProvider.get().api_key.findMany({
                    where: { id: { in: ids.map(id => BigInt(id)) } },
                    select: { key: true },
                });
                // Clear cached permissions in Redis using CacheService
                const cacheService = CacheService.get();
                await Promise.all(
                    records.map(record => cacheService.del(`apiKeyPerm:${record.key}`)),
                );
            },
        },
        yup: apiKeyValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data?.team,
            User: data?.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            team: "Team",
            user: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { user: useVisibility("User", "Own", data) },
                        { team: useVisibility("Team", "Own", data) },
                    ],
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("ApiKey", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("ApiKey", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
