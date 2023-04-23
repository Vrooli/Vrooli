import { MaxObjects } from "@local/consts";
import { emailValidation } from "@local/validation";
import { CustomError, Trigger } from "../events";
import { defaultPermissions } from "../utils";
const __typename = "Email";
const suppFields = [];
export const EmailModel = ({
    __typename,
    delegate: (prisma) => prisma.email,
    display: {
        select: () => ({ id: true, emailAddress: true }),
        label: (select) => select.emailAddress,
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
            pre: async ({ createList, deleteList, prisma, userData }) => {
                if (createList.length) {
                    const existingEmails = await prisma.email.findMany({
                        where: { emailAddress: { in: createList.map(x => x.emailAddress) } },
                    });
                    if (existingEmails.length > 0)
                        throw new CustomError("0044", "EmailInUse", userData.languages);
                }
                if (deleteList.length) {
                    const allEmails = await prisma.email.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedEmailsCount = allEmails.filter(x => !deleteList.includes(x.id) && x.verified).length;
                    const verifiedWalletsCount = await prisma.wallet.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    if (remainingVerifiedEmailsCount + verifiedWalletsCount < 1)
                        throw new CustomError("0049", "MustLeaveVerificationMethod", userData.languages);
                }
                return {};
            },
            create: async ({ data, userData }) => ({
                emailAddress: data.emailAddress,
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
        yup: emailValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: () => false,
        profanityFields: ["emailAddress"],
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ user: { id: userId } }),
        },
    },
});
//# sourceMappingURL=email.js.map