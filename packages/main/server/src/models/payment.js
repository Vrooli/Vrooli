import { MaxObjects, PaymentSortBy } from "@local/consts";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "Payment";
const suppFields = [];
export const PaymentModel = ({
    __typename,
    delegate: (prisma) => prisma.payment,
    display: {
        select: () => ({ id: true, description: true }),
        label: (select) => select.description.length > 20 ? select.description.slice(0, 20) + "..." : select.description,
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
        },
        countFields: {},
    },
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
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            organization: "Organization",
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
//# sourceMappingURL=payment.js.map