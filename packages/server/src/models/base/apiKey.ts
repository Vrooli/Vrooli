import { apiKeyValidation, MaxObjects, uuid } from "@local/shared";
import { randomString } from "../../auth/wallet";
import { noNull } from "../../builders/noNull";
import { defaultPermissions } from "../../utils";
import { ApiKeyFormat } from "../formats";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { ApiKeyModelLogic } from "./types";

const __typename = "ApiKey" as const;
const suppFields = [] as const;
export const ApiKeyModel: ModelLogic<ApiKeyModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.api_key,
    display: {
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
    },
    format: ApiKeyFormat,
    mutate: {
        shape: {
            create: async ({ userData, data }) => ({
                id: uuid(),
                key: randomString(64),
                creditsUsedBeforeLimit: data.creditsUsedBeforeLimit,
                stopAtLimit: data.stopAtLimit,
                absoluteMax: data.absoluteMax,
                user: data.organizationConnect ? undefined : { connect: { id: userData.id } },
                organization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,

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
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data?.organization,
            User: data?.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            organization: "Organization",
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
