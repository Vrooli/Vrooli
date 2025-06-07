import { DEFAULT_LANGUAGE, MaxObjects } from "@vrooli/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { PremiumFormat } from "../formats.js";
import { type PremiumModelLogic } from "./types.js";

const __typename = "Premium" as const;
export const PremiumModel: PremiumModelLogic = ({
    __typename,
    dbTable: "payment",
    display: () => ({
        label: {
            select: () => ({ id: true, customPlan: true }),
            get: (select, languages) => {
                const lng = languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE;
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
                return useVisibility("Premium", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Premium", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
