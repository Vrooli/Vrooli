import { Email, EmailCreateInput, emailValidation, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { CustomError, Trigger } from "../../events";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Email" as const;
export const EmailFormat: Formatter<ModelEmailLogic> = {
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
};
