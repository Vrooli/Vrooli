import { emailValidation, MaxObjects } from "@local/shared";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { defaultPermissions } from "../../utils";
import { EmailFormat } from "../formats";
import { ModelLogic } from "../types";
import { EmailModelLogic } from "./types";

const __typename = "Email" as const;
const suppFields = [] as const;
export const EmailModel: ModelLogic<EmailModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.email,
    display: {
        label: {
            select: () => ({ id: true, emailAddress: true }),
            get: (select) => select.emailAddress,
        },
    },
    format: EmailFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Delete, prisma, userData }) => {
                // Prevent creating emails if at least one is already in use
                if (Create.length) {
                    const existingEmails = await prisma.email.findMany({
                        where: { emailAddress: { in: Create.map(x => x.input.emailAddress) } },
                    });
                    if (existingEmails.length > 0) throw new CustomError("0044", "EmailInUse", userData.languages);
                }
                // Prevent deleting emails if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allEmails = await prisma.email.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedEmailsCount = allEmails.filter(x => !Delete.some(d => d.input === x.id) && x.verified).length;
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
        yup: emailValidation,
    },
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            User: data?.user,
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
