import { MaxObjects, phoneValidation } from "@local/shared";
import { ModelMap } from ".";
import { Trigger } from "../../events/trigger";
import { defaultPermissions } from "../../utils";
import { PhoneFormat } from "../formats";
import { OrganizationModelLogic, PhoneModelLogic } from "./types";

const __typename = "Phone" as const;
export const PhoneModel: PhoneModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.phone,
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
            create: async ({ data, userData }) => ({
                phoneNumber: data.phoneNumber,
                user: { connect: { id: userData.id } },
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, prisma, userData }) => {
                for (const objectId of createdIds) {
                    await Trigger(prisma, userData.languages).objectCreated({
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
