import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Premium } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { defaultPermissions } from "../utils";

const __typename = 'Premium' as const;
const suppFields = [] as const;
export const PremiumModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Premium,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.premiumUpsertArgs['create'],
    PrismaUpdate: Prisma.premiumUpsertArgs['update'],
    PrismaModel: Prisma.premiumGetPayload<SelectWrap<Prisma.premiumSelect>>,
    PrismaSelect: Prisma.premiumSelect,
    PrismaWhere: Prisma.premiumWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: {
        select: () => ({ id: true, customPlan: true }),
        label: (select, languages) => {
            const lng = languages[0];
            if (select.customPlan) return i18next.t(`common:PaymentPlanCustom`, { lng });
            return i18next.t(`common:PaymentPlanBasic`, { lng });
        }
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        countFields: {},
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            user: 'User',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
})