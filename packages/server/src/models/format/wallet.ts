import { MaxObjects, Wallet, WalletUpdateInput, walletValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { CustomError } from "../../events";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Wallet" as const;
export const WalletFormat: Formatter<ModelWalletLogic> = {
        gqlRelMap: {
            __typename,
            handles: "Handle",
            user: "User",
            organization: "Organization",
        },
        prismaRelMap: {
            __typename,
            handles: "Handle",
            user: "User",
            organization: "Organization",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            pre: async ({ deleteList, prisma, userData }) => {
                // Prevent deleting wallets if it will leave you with less than one verified authentication method
                if (deleteList.length) {
                    const allWallets = await prisma.wallet.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedWalletsCount = allWallets.filter(x => !deleteList.includes(x.id) && x.verified).length;
                    const verifiedEmailsCount = await prisma.email.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    if (remainingVerifiedWalletsCount + verifiedEmailsCount < 1)
                        throw new CustomError("0049", "MustLeaveVerificationMethod", userData.languages);
                }
            },
            update: async ({ data }) => data,
        },
        yup: walletValidation,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: (...params) => ({
            id: true,
            organization: "Organization",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.walletSelect>(data, [
            ["organization", "Organization"],
            ["user", "User"],
        ], languages),
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
};
