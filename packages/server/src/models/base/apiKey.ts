import { apiKeyValidation, ApiKeySortBy, generatePK, MaxObjects } from "@vrooli/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CacheService } from "../../redisConn.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ApiKeyFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { type ApiKeyModelLogic } from "./types.js";

const __typename = "ApiKey" as const;

// Store for temporarily holding raw keys during creation
const rawKeyStore = new Map<string, string>();

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
            create: async ({ userData, data }) => {
                const rawKeyToShowUser = ApiKeyEncryptionService.generateSiteKey();
                const id = generatePK();
                // Store the raw key temporarily for the supplemental fields to pick up
                rawKeyStore.set(id, rawKeyToShowUser);
                // Clean up after a short delay (in case of errors)
                setTimeout(() => rawKeyStore.delete(id), 10000);
                return {
                    id,
                    creditsUsed: BigInt(0),
                    disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                    limitHard: BigInt(data.limitHard),
                    limitSoft: data.limitSoft ? BigInt(data.limitSoft) : null,
                    key: ApiKeyEncryptionService.get().hashSiteKey(rawKeyToShowUser),
                    name: data.name,
                    permissions: data.permissions,
                    stopAtLimit: data.stopAtLimit,
                    team: data.teamConnect ? { connect: { id: BigInt(data.teamConnect) } } : undefined,
                    user: data.teamConnect ? undefined : { connect: { id: BigInt(userData.id) } },
                };
            },
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
    search: {
        defaultSort: ApiKeySortBy.DateCreatedDesc,
        sortBy: ApiKeySortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            ids: true,
            teamIdRoot: true,
            userIdRoot: true,
        },
        searchStringQuery: () => ({
            name: {
                contains: "{{searchString}}",
                mode: "insensitive",
            },
        }),
        supplemental: {
            dbFields: ["id"],
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, objects }) => {
                // For each object, check if we have a raw key in the store
                const keys: (string | null)[] = [];
                for (const obj of objects) {
                    const id = obj.id.toString();
                    const rawKey = rawKeyStore.get(id);
                    if (rawKey) {
                        // Remove from store after retrieving
                        rawKeyStore.delete(id);
                        keys.push(rawKey);
                    } else {
                        keys.push(null);
                    }
                }
                return {
                    key: keys,
                };
            },
        },
    },
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
