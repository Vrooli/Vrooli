import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.push_deviceSelect,
    Prisma.push_deviceGetPayload<SelectWrap<Prisma.push_deviceSelect>>
> => ({
    select: () => ({ id: true, name: true, p256dh: true }),
    label: (select) => {
        // Return name if it exists
        if (select.name) return select.name
        // Otherwise, return last 4 digits of p256dh
        return select.p256dh.length < 4 ? select.p256dh : `...${select.p256dh.slice(-4)}`
    }
})

export const PushDeviceModel = ({
    delegate: (prisma: PrismaType) => prisma.push_device,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'PushDevice' as GraphQLModelType,
    validate: {} as any,
})