import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiKey, ApiKeyCreateInput, ApiKeyUpdateInput, MaxObjects } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { OrganizationModel } from "./organization";
import { defaultPermissions } from "../utils";
import { apiKeyValidation } from "@shared/validation";
import { uuid } from "@shared/uuid";
import { randomString } from "../auth";
import { noNull } from "../builders";

const __typename = 'ApiKey' as const;
const suppFields = [] as const;
export const ApiKeyModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ApiKeyCreateInput,
    GqlUpdate: ApiKeyUpdateInput,
    GqlPermission: {},
    GqlModel: ApiKey,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.api_keyUpsertArgs['create'],
    PrismaUpdate: Prisma.api_keyUpsertArgs['update'],
    PrismaModel: Prisma.api_keyGetPayload<SelectWrap<Prisma.api_keySelect>>,
    PrismaSelect: Prisma.api_keySelect,
    PrismaWhere: Prisma.api_keyWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api_key,
    display: {
        select: () => ({ id: true, key: true }),
        // Label should be first 4 characters of key, an ellipsis, and last 4 characters of key
        label: (select) => {
            // Make sure key is at least 8 characters long
            // (should always be, but you never know)
            if (select.key.length < 8) return select.key
            return select.key.slice(0, 4) + '...' + select.key.slice(-4)
        }
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
            organization: 'Organization',
            user: 'User',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})