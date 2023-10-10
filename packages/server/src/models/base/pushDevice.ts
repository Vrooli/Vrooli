import { MaxObjects, pushDeviceValidation } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { defaultPermissions } from "../../utils";
import { PushDeviceFormat } from "../formats";
import { PushDeviceModelLogic } from "./types";

const __typename = "PushDevice" as const;
export const PushDeviceModel: PushDeviceModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.push_device,
    display: {
        label: {
            select: () => ({ id: true, name: true, p256dh: true }),
            get: (select) => {
                // Return name if it exists
                if (select.name) return select.name;
                // Otherwise, return last 4 digits of p256dh
                return select.p256dh.length < 4 ? select.p256dh : `...${select.p256dh.slice(-4)}`;
            },
        },
    },
    format: PushDeviceFormat,
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
    search: undefined,
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
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
