import { MaxObjects, PaymentSortBy } from "@vrooli/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { PaymentFormat } from "../formats.js";
import { type PaymentModelLogic } from "./types.js";

const __typename = "Payment" as const;
export const PaymentModel: PaymentModelLogic = ({
    __typename,
    dbTable: "payment",
    display: () => ({
        label: {
            select: () => ({ id: true, description: true }),
            get: (select) => select.description,
        },
    }),
    format: PaymentFormat,
    search: {
        defaultSort: PaymentSortBy.DateCreatedDesc,
        sortBy: PaymentSortBy,
        searchFields: {
            cardLast4: true,
            createdTimeFrame: true,
            currency: true,
            maxAmount: true,
            minAmount: true,
            status: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
            ],
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (_data, _getParentInfo?) => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => ({
            Team: data?.team,
            User: data?.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            team: "Team",
            user: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { team: useVisibility("Team", "Own", data) },
                        { user: useVisibility("User", "Own", data) },
                    ],
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Payment", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Payment", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
