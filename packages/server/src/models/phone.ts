import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const __typename = 'Phone' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.phoneSelect,
    Prisma.phoneGetPayload<SelectWrap<Prisma.phoneSelect>>
> => ({
    select: () => ({ id: true, phoneNumber: true }),
    // Only display last 4 digits of phone number
    label: (select) => {
        // Make sure number is at least 4 digits long
        if (select.phoneNumber.length < 4) return select.phoneNumber
        return `...${select.phoneNumber.slice(-4)}`
    }
})

export const PhoneModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.phone,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})