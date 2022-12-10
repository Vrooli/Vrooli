import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'Payment' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.paymentSelect,
    Prisma.paymentGetPayload<SelectWrap<Prisma.paymentSelect>>
> => ({
    select: () => ({ id: true, description: true }),
    // Cut off the description at 20 characters
    label: (select) =>  select.description.length > 20 ? select.description.slice(0, 20) + '...' : select.description,
})

export const PaymentModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})