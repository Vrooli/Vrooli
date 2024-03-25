import { MaxObjects } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions } from "../../utils";
import { PremiumFormat } from "../formats";
import { OrganizationModelLogic, PremiumModelLogic } from "./types";

const __typename = "Premium" as const;
export const PremiumModel: PremiumModelLogic = ({
    __typename,
    delegate: (p) => p.payment,
    display: () => ({
        label: {
            select: () => ({ id: true, customPlan: true }),
            get: (select, languages) => {
                const lng = languages[0];
                if (select.customPlan) return i18next.t("common:PaymentPlanCustom", { lng });
                return i18next.t("common:PaymentPlanBasic", { lng });
            },
        },
    }),
    format: PremiumFormat,
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data?.organization,
            User: data?.user,
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
                    { organization: ModelMap.get<OrganizationModelLogic>("Organization").query.hasRoleQuery(userId) },
                ],
            }),
        },
    }),
});
