import { emailValidation, generatePK, MaxObjects } from "@local/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { Trigger } from "../../events/trigger.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { EmailFormat } from "../formats.js";
import { type EmailModelLogic } from "./types.js";

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
                    const existingEmails = await DbProvider.get().email.findMany({
                        where: { emailAddress: { in: emailAddresses } },
                    });
                    if (existingEmails.length > 0) throw new CustomError("0044", "EmailInUse", { emailAddresses });
                }
                // Prevent deleting emails if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allEmails = await DbProvider.get().email.findMany({
                        where: { user: { id: BigInt(userData.id) } },
                        select: { id: true, verifiedAt: true },
                    });
                    const remainingVerifiedEmailsCount = allEmails.filter(x => !Delete.some(d => d.input === x.id.toString()) && x.verifiedAt).length;
                    const verifiedPhonesCount = await DbProvider.get().phone.count({
                        where: { user: { id: BigInt(userData.id) }, verifiedAt: { not: null } },
                    });
                    const verifiedWalletsCount = await DbProvider.get().wallet.count({
                        where: { user: { id: BigInt(userData.id) }, verifiedAt: { not: null } },
                    });
                    if (remainingVerifiedEmailsCount + verifiedPhonesCount + verifiedWalletsCount < 1)
                        throw new CustomError("0049", "MustLeaveVerificationMethod");
                }
                return {};
            },
            create: async ({ data, userData }) => ({
                id: generatePK(),
                emailAddress: data.emailAddress,
                user: { connect: { id: BigInt(userData.id) } },
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
                    user: { id: BigInt(data.userId) },
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
