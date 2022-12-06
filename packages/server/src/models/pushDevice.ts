import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const PushDeviceModel = ({
    delegate: (prisma: PrismaType) => prisma.push_device,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'PushDevice' as GraphQLModelType,
    validate: {} as any,
})