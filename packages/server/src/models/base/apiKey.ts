import { apiKeyValidation, MaxObjects, uuid } from "@local/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ApiKeyFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { ApiKeyModelLogic, TeamModelLogic } from "./types.js";

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
                id: uuid(),
                creditsUsedBeforeLimit: 0,
                disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                limitHard: BigInt(data.limitHard),
                limitSoft: data.limitSoft ? BigInt(data.limitSoft) : null,
                key: ApiKeyEncryptionService.generateSiteKey(),
                name: data.name,
                permissions: data.permissions,
                stopAtLimit: data.stopAtLimit,
                team: data.teamConnect ? { connect: { id: data.teamConnect } } : undefined,
                user: data.teamConnect ? undefined : { connect: { id: userData.id } },

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
                        { user: { id: data.userId } },
                        { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId) },
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
