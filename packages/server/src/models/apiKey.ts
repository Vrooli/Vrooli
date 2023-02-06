import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiKey, ApiKeyCreateInput, ApiKeyUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { OrganizationModel } from "./organization";
import { defaultPermissions } from "../utils";

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
    mutate: {} as any,
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: {
            User: {
                private: {
                    noPremium: 1,
                    premium: 5,
                },
                public: {
                    noPremium: 1,
                    premium: 5,
                }
            },
            Organization: {
                private: {
                    noPremium: 1,
                    premium: 5,
                },
                public: {
                    noPremium: 1,
                    premium: 5,
                }
            },
        },
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isPublic }) => ({
            ...defaultPermissions({ isAdmin, isDeleted, isPublic }),
        }),
        permissionsSelect: () => ({
            id: true,
            organization: 'Organization',
            user: 'User',
        }),
        visibility: {
            private: { },
            public: { },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})