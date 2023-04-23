import { MaxObjects } from "@local/consts";
import { pushDeviceValidation } from "@local/validation";
import { noNull } from "../builders";
import { defaultPermissions } from "../utils";
const __typename = "PushDevice";
const suppFields = [];
export const PushDeviceModel = ({
    __typename,
    delegate: (prisma) => prisma.push_device,
    display: {
        select: () => ({ id: true, name: true, p256dh: true }),
        label: (select) => {
            if (select.name)
                return select.name;
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
//# sourceMappingURL=pushDevice.js.map