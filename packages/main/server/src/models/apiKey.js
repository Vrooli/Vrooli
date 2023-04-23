import { MaxObjects } from "@local/consts";
import { uuid } from "@local/uuid";
import { apiKeyValidation } from "@local/validation";
import { randomString } from "../auth";
import { noNull } from "../builders";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "ApiKey";
const suppFields = [];
export const ApiKeyModel = ({
    __typename,
    delegate: (prisma) => prisma.api_key,
    display: {
        select: () => ({ id: true, key: true }),
        label: (select) => {
            if (select.key.length < 8)
                return select.key;
            return select.key.slice(0, 4) + "..." + select.key.slice(-4);
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
        },
        countFields: {},
    },
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
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
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
//# sourceMappingURL=apiKey.js.map