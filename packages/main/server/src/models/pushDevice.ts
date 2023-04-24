import { MaxObjects, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, pushDeviceValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = "PushDevice" as const;
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
    PrismaCreate: Prisma.push_deviceUpsertArgs["create"],
    PrismaUpdate: Prisma.push_deviceUpsertArgs["update"],
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
            if (select.name) return select.name;
            // Otherwise, return last 4 digits of p256dh
            return select.p256dh.length < 4 ? select.p256dh : `...${select.p256dh.slice(-4)}`;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: "User",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => ({
                endpoint: data.endpoint,
                expires: noNull(data.expires),
                auth: data.keys.auth,
                p256dh: data.keys.p256dh,
                name: noNull(data.name),
                user: { connect: { id: userData.id } },
            }),
            update: async ({ data }) => ({
                name: noNull(data.name),
            }),
        },
        yup: pushDeviceValidation,
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
});
