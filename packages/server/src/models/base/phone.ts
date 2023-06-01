import { MaxObjects, phoneValidation } from "@local/shared";
import { Trigger } from "../events";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic, PhoneModelLogic } from "./types";

const __typename = "Phone" as const;
const suppFields = [] as const;
export const PhoneModel: ModelLogic<PhoneModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.phone,
    display: {
        label: {
            select: () => ({ id: true, phoneNumber: true }),
            // Only display last 4 digits of phone number
            get: (select) => {
                // Make sure number is at least 4 digits long
                if (select.phoneNumber.length < 4) return select.phoneNumber;
                return `...${select.phoneNumber.slice(-4)}`;
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: "User",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => ({
                phoneNumber: data.phoneNumber,
                user: { connect: { id: userData.id } },
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
                for (const { id: objectId } of created) {
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
