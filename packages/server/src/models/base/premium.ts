import { MaxObjects } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions } from "../../utils";
import { PremiumFormat } from "../formats";
import { PremiumModelLogic, TeamModelLogic } from "./types";

const __typename = "Premium" as const;
export const PremiumModel: PremiumModelLogic = ({
    __typename,
    dbTable: "payment",
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
