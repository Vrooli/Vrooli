import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { SelectWrap } from "../builders/types";
import { Premium } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Premium,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.premiumUpsertArgs['create'],
    PrismaUpdate: Prisma.premiumUpsertArgs['update'],
    PrismaModel: Prisma.premiumGetPayload<SelectWrap<Prisma.premiumSelect>>,
    PrismaSelect: Prisma.premiumSelect,
    PrismaWhere: Prisma.premiumWhereInput,
}

const __typename = 'Premium' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, customPlan: true }),
    label: (select, languages) => {
        const lng = languages[0];
        if (select.customPlan) return i18next.t(`common:PaymentPlanCustom`, { lng });
        return i18next.t(`common:PaymentPlanBasic`, { lng });
    }
})

export const PremiumModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})