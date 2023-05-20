import { MaxObjects, Payment, PaymentSearchInput, PaymentSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";

const __typename = "Payment" as const;
const suppFields = [] as const;
export const PaymentModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Payment,
    GqlPermission: object,
    GqlSearch: PaymentSearchInput,
    GqlSort: PaymentSortBy,
    PrismaCreate: Prisma.paymentUpsertArgs["create"],
    PrismaUpdate: Prisma.paymentUpsertArgs["update"],
    PrismaModel: Prisma.paymentGetPayload<SelectWrap<Prisma.paymentSelect>>,
    PrismaSelect: Prisma.paymentSelect,
    PrismaWhere: Prisma.paymentWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: {
        label: {
            select: () => ({ id: true, description: true }),
            // Cut off the description at 20 characters TODO should do this for every label
            get: (select) => select.description.length > 20 ? select.description.slice(0, 20) + "..." : select.description,
        },
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
