import { Email, EmailCreateInput, emailValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { CustomError, Trigger } from "../events";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = "Email" as const;
const suppFields = [] as const;
export const EmailModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: EmailCreateInput,
    GqlUpdate: undefined,
    GqlModel: Email,
    GqlPermission: object,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.emailUpsertArgs["create"],
    PrismaUpdate: Prisma.emailUpsertArgs["update"],
    PrismaModel: Prisma.emailGetPayload<SelectWrap<Prisma.emailSelect>>,
    PrismaSelect: Prisma.emailSelect,
    PrismaWhere: Prisma.emailWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.email,
    display: {
        label: {
            select: () => ({ id: true, emailAddress: true }),
            get: (select) => select.emailAddress,
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
            pre: async ({ createList, deleteList, prisma, userData }) => {
                // Prevent creating emails if at least one is already in use
                if (createList.length) {
                    const existingEmails = await prisma.email.findMany({
                        where: { emailAddress: { in: createList.map(x => x.emailAddress) } },
                    });
                    if (existingEmails.length > 0) throw new CustomError("0044", "EmailInUse", userData.languages);
                }
                // Prevent deleting emails if it will leave you with less than one verified authentication method
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
