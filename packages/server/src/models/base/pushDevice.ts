import { MaxObjects, pushDeviceValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { PushDeviceFormat } from "../formats.js";
import { type PushDeviceModelLogic } from "./types.js";

const __typename = "PushDevice" as const;
export const PushDeviceModel: PushDeviceModelLogic = ({
    __typename,
    dbTable: "push_device",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true, p256dh: true }),
            get: (select) => {
                // Return name if it exists
                if (select.name) return select.name;
                const DIGITS_TO_SHOW = 4;
                // Otherwise, return last 4 digits of p256dh
                return select.p256dh.length < DIGITS_TO_SHOW ? select.p256dh : `...${select.p256dh.slice(-DIGITS_TO_SHOW)}`;
            },
        },
    }),
    format: PushDeviceFormat,
    mutate: {
        shape: {
            create: async ({ data, userData }) => ({
                endpoint: data.endpoint,
                expires: noNull(data.expires),
                auth: data.keys.auth,
                p256dh: data.keys.p256dh,
                name: noNull(data.name),
                user: { connect: { id: BigInt(userData.id) } },
            }),
            update: async ({ data }) => ({
                name: noNull(data.name),
            }),
        },
        yup: pushDeviceValidation,
    },
    search: undefined,
    validate: () => ({
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
            own: function getOwn(data) {
                return {
                    user: { id: data.userId },
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("PushDevice", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("PushDevice", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
