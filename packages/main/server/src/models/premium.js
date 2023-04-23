import { MaxObjects } from "@local/consts";
import i18next from "i18next";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "Premium";
const suppFields = [];
export const PremiumModel = ({
    __typename,
    delegate: (prisma) => prisma.payment,
    display: {
        select: () => ({ id: true, customPlan: true }),
        label: (select, languages) => {
            const lng = languages[0];
            if (select.customPlan)
                return i18next.t("common:PaymentPlanCustom", { lng });
            return i18next.t("common:PaymentPlanBasic", { lng });
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            user: "User",
        },
        countFields: {},
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
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
//# sourceMappingURL=premium.js.map