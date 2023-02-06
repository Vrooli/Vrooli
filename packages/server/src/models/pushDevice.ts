import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'PushDevice' as const;
const suppFields = [] as const;
export const PushDeviceModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PushDeviceCreateInput,
    GqlUpdate: PushDeviceUpdateInput,
    GqlModel: PushDevice,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.push_deviceUpsertArgs['create'],
    PrismaUpdate: Prisma.push_deviceUpsertArgs['update'],
    PrismaModel: Prisma.push_deviceGetPayload<SelectWrap<Prisma.push_deviceSelect>>,
    PrismaSelect: Prisma.push_deviceSelect,
    PrismaWhere: Prisma.push_deviceWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.push_device,
    display: {
        select: () => ({ id: true, name: true, p256dh: true }),
        label: (select) => {
            // Return name if it exists
            if (select.name) return select.name
            // Otherwise, return last 4 digits of p256dh
            return select.p256dh.length < 4 ? select.p256dh : `...${select.p256dh.slice(-4)}`
        }
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})