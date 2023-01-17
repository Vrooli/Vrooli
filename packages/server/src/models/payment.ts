import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Payment } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'Payment' as const;
const suppFields = [] as const;
export const PaymentModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: {
        select: () => ({ id: true, description: true }),
        // Cut off the description at 20 characters
        label: (select) =>  select.description.length > 20 ? select.description.slice(0, 20) + '...' : select.description,
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})