import { emailValidation, MaxObjects } from "@local/shared";
import { useVisibility } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { defaultPermissions } from "../../utils";
import { EmailFormat } from "../formats";
import { EmailModelLogic } from "./types";

const __typename = "Email" as const;
export const EmailModel: EmailModelLogic = ({
    __typename,
    dbTable: "email",
    display: () => ({
        label: {
            select: () => ({ id: true, emailAddress: true }),
            get: (select) => select.emailAddress,
        },
    }),
    format: EmailFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Delete, userData }) => {
                // Prevent creating emails if at least one is already in use
                if (Create.length) {
                    const emailAddresses = Create.map(x => x.input.emailAddress);
                    const existingEmails = await prismaInstance.email.findMany({
                        where: { emailAddress: { in: emailAddresses } },
                    });
                    if (existingEmails.length > 0) throw new CustomError("0044", "EmailInUse", { emailAddresses });
                }
                // Prevent deleting emails if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allEmails = await prismaInstance.email.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedEmailsCount = allEmails.filter(x => !Delete.some(d => d.input === x.id) && x.verified).length;
                    const verifiedPhonesCount = await prismaInstance.phone.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    const verifiedWalletsCount = await prismaInstance.wallet.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    if (remainingVerifiedEmailsCount + verifiedPhonesCount + verifiedWalletsCount < 1)
                        throw new CustomError("0049", "MustLeaveVerificationMethod");
                }
                return {};
            },
            create: async ({ data, userData }) => ({
                emailAddress: data.emailAddress,
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
        yup: emailValidation,
    },
    search: undefined,
    validate: () => ({
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
            own: function getOwn(data) {
                return {
                    user: { id: data.userId },
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Email", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Email", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
