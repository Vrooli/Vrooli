import { apiKeyValidation, MaxObjects, uuid } from "@local/shared";
import { ModelMap } from ".";
import { randomString } from "../../auth/codes";
import { noNull } from "../../builders/noNull";
import { defaultPermissions } from "../../utils";
import { ApiKeyFormat } from "../formats";
import { ApiKeyModelLogic, TeamModelLogic } from "./types";

const __typename = "ApiKey" as const;
export const ApiKeyModel: ApiKeyModelLogic = ({
    __typename,
    dbTable: "api_key",
    display: () => ({
        label: {
            select: () => ({ id: true, key: true }),
            // Of the form "1234...5678"
            get: (select) => {
                // Make sure key is at least 8 characters long
                // (should always be, but you never know)
                if (select.key.length < 8) return select.key;
                return select.key.slice(0, 4) + "..." + select.key.slice(-4);
            },
        },
    }),
    format: ApiKeyFormat,
    mutate: {
        shape: {
            create: async ({ userData, data }) => ({
                id: uuid(),
                key: randomString(64),
                creditsUsedBeforeLimit: data.creditsUsedBeforeLimit,
                stopAtLimit: data.stopAtLimit,
                absoluteMax: data.absoluteMax,
                team: data.teamConnect ? { connect: { id: data.teamConnect } } : undefined,
                user: data.teamConnect ? undefined : { connect: { id: userData.id } },

            }),
            update: async ({ data }) => ({
                creditsUsedBeforeLimit: noNull(data.creditsUsedBeforeLimit),
                stopAtLimit: noNull(data.stopAtLimit),
                absoluteMax: noNull(data.absoluteMax),
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
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
