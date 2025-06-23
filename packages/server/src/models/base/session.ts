import { MaxObjects } from "@vrooli/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { SessionFormat } from "../formats.js";
import { type SessionModelLogic } from "./types.js";

const __typename = "Session" as const;
export const SessionModel: SessionModelLogic = ({
    __typename,
    dbTable: "session",
    display: () => ({
        label: {
            select: () => ({ id: true, device_info: true }),
            get: (select) => select.device_info ?? "",
        },
    }),
    format: SessionFormat,
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: () => false,
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { user: { id: data.userId } },
                    ],
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Session", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Session", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
