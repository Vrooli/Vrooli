import { MaxObjects } from "@local/consts";
import { phoneValidation } from "@local/validation";
import { Trigger } from "../events";
import { defaultPermissions } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "Phone";
const suppFields = [];
export const PhoneModel = ({
    __typename,
    delegate: (prisma) => prisma.phone,
    display: {
        select: () => ({ id: true, phoneNumber: true }),
        label: (select) => {
            if (select.phoneNumber.length < 4)
                return select.phoneNumber;
            return `...${select.phoneNumber.slice(-4)}`;
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
                        hasCompleteAndPublic: true,
                        hasParent: true,
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
//# sourceMappingURL=phone.js.map