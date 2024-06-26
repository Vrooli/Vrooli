import { MaxObjects, PaymentSortBy } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions } from "../../utils";
import { PaymentFormat } from "../formats";
import { PaymentModelLogic, TeamModelLogic } from "./types";

const __typename = "Payment" as const;
export const PaymentModel: PaymentModelLogic = ({
    __typename,
    dbTable: "payment",
    display: () => ({
        label: {
            select: () => ({ id: true, description: true }),
            // Cut off the description at 20 characters TODO should do this for every label
            get: (select) => select.description.length > 20 ? select.description.slice(0, 20) + "..." : select.description,
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
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
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
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(userId) },
                    { user: { id: userId } },
                ],
            }),
        },
    }),
});
