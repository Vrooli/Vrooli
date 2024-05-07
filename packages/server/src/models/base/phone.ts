import { MaxObjects, phoneValidation } from "@local/shared";
import { ModelMap } from ".";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { defaultPermissions } from "../../utils";
import { PhoneFormat } from "../formats";
import { OrganizationModelLogic, PhoneModelLogic } from "./types";

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
                if (select.phoneNumber.length < 4) return select.phoneNumber;
                return `...${select.phoneNumber.slice(-4)}`;
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
