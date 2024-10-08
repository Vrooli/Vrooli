import { MaxObjects, phoneValidation } from "@local/shared";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { defaultPermissions } from "../../utils";
import { PhoneFormat } from "../formats";
import { PhoneModelLogic, TeamModelLogic } from "./types";

const __typename = "Phone" as const;
export const PhoneModel: PhoneModelLogic = ({
    __typename,
    dbTable: "phone",
    display: () => ({
        label: {
            select: () => ({ id: true, phoneNumber: true }),
            // Only display last 4 digits of phone number
            get: (select) => {
                // Make sure number is at least 4 digits long
                const DIGITS_TO_SHOW = 4;
                if (select.phoneNumber.length < DIGITS_TO_SHOW) return select.phoneNumber;
                return `...${select.phoneNumber.slice(-DIGITS_TO_SHOW)}`;
            },
        },
    }),
    format: PhoneFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Delete, userData }) => {
                // Prevent creating phones if at least one is already in use
                if (Create.length) {
                    const phoneNumbers = Create.map(x => x.input.phoneNumber);
                    const existingPhones = await prismaInstance.phone.findMany({
                        where: { phoneNumber: { in: phoneNumbers } },
                    });
                    if (existingPhones.length > 0) {
                        throw new CustomError("0147", "PhoneInUse", userData.languages, { phoneNumbers });
                    }
                }
                // Prevent deleting phones if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allPhones = await prismaInstance.phone.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedPhonesCount = allPhones.filter(x => !Delete.some(d => d.input === x.id) && x.verified).length;
                    const verifiedEmailsCount = await prismaInstance.email.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    const verifiedWalletsCount = await prismaInstance.wallet.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    if (remainingVerifiedPhonesCount + verifiedEmailsCount + verifiedWalletsCount < 1)
                        throw new CustomError("0153", "MustLeaveVerificationMethod", userData.languages);
                }
                return {};
            },
            create: async ({ data, userData }) => ({
                phoneNumber: data.phoneNumber,
                user: { connect: { id: userData.id } },
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                for (const objectId of createdIds) {
                    await Trigger(userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                }
            },
        },
        yup: phoneValidation,
    },
    search: undefined,
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
            own: function getOwn(data) {
                return {
                    OR: [
                        { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Phone", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Phone", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
