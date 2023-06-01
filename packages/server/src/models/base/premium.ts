import { MaxObjects } from "@local/shared";
import i18next from "i18next";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic, PremiumModelLogic } from "./types";

const __typename = "Premium" as const;
const suppFields = [] as const;
export const PremiumModel: ModelLogic<PremiumModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.payment,
    display: {
        label: {
            select: () => ({ id: true, customPlan: true }),
            get: (select, languages) => {
                const lng = languages[0];
                if (select.customPlan) return i18next.t("common:PaymentPlanCustom", { lng });
                return i18next.t("common:PaymentPlanBasic", { lng });
            },
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
