import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Payment } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Payment,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.paymentUpsertArgs['create'],
    PrismaUpdate: Prisma.paymentUpsertArgs['update'],
    PrismaModel: Prisma.paymentGetPayload<SelectWrap<Prisma.paymentSelect>>,
    PrismaSelect: Prisma.paymentSelect,
    PrismaWhere: Prisma.paymentWhereInput,
}

const __typename = 'Payment' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, description: true }),
    // Cut off the description at 20 characters
    label: (select) =>  select.description.length > 20 ? select.description.slice(0, 20) + '...' : select.description,
})

export const PaymentModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})