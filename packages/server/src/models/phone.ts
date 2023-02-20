import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Phone, PhoneCreateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { OrganizationModel } from "./organization";
import { defaultPermissions } from "../utils";

const __typename = 'Phone' as const;
const suppFields = [] as const;
export const PhoneModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PhoneCreateInput,
    GqlUpdate: undefined,
    GqlModel: Phone,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.phoneUpsertArgs['create'],
    PrismaUpdate: Prisma.phoneUpsertArgs['update'],
    PrismaModel: Prisma.phoneGetPayload<SelectWrap<Prisma.phoneSelect>>,
    PrismaSelect: Prisma.phoneSelect,
    PrismaWhere: Prisma.phoneWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.phone,
    display: {
        select: () => ({ id: true, phoneNumber: true }),
        // Only display last 4 digits of phone number
        label: (select) => {
            // Make sure number is at least 4 digits long
            if (select.phoneNumber.length < 4) return select.phoneNumber
            return `...${select.phoneNumber.slice(-4)}`
        }
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: 'User',
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
        permissionResolvers: defaultPermissions,
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