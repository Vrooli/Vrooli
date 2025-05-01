import { apiKeyExternalValidation, MaxObjects, uuid } from "@local/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ApiKeyExternalFormat } from "../formats.js";
import { ApiKeyExternalModelLogic } from "./types.js";

const __typename = "ApiKeyExternal" as const;
export const ApiKeyExternalModel: ApiKeyExternalModelLogic = ({
    __typename,
    dbTable: "api_key_external",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => {
                return select.name;
            },
        },
    }),
    format: ApiKeyExternalFormat,
    mutate: {
        shape: {
            create: async ({ userData, data }) => ({
                id: uuid(),
                disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                key: ApiKeyEncryptionService.get().encryptExternal(data.key),
                name: data.name,
                service: data.service,
                team: data.teamConnect ? { connect: { id: data.teamConnect } } : undefined,
                user: data.teamConnect ? undefined : { connect: { id: userData.id } },

            }),
            update: async ({ data }) => ({
                disabledAt: data.disabled === true ? new Date() : data.disabled === false ? null : undefined,
                key: data.key ? ApiKeyEncryptionService.get().encryptExternal(data.key) : undefined,
                name: noNull(data.name),
                service: noNull(data.service),
            }),
        },
        yup: apiKeyExternalValidation,
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
