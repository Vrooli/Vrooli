import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'Premium' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.premiumSelect,
    Prisma.premiumGetPayload<SelectWrap<Prisma.premiumSelect>>
> => ({
    select: () => ({ id: true, customPlan: true }),
    label: (select, languages) => {
        const lng = languages[0];
        if (select.customPlan) return i18next.t(`common:PaymentPlanCustom`, { lng });
        return i18next.t(`common:PaymentPlanBasic`, { lng });
    }
})

export const PremiumModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})