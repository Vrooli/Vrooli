import { MaxObjects, walletValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { CustomError } from "../../events/error";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { WalletFormat } from "../formats";
import { OrganizationModelLogic, WalletModelLogic } from "./types";

const __typename = "Wallet" as const;
export const WalletModel: WalletModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.wallet,
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name ?? "",
        },
    }),
    format: WalletFormat,
    mutate: {
        shape: {
            pre: async ({ Delete, prisma, userData }) => {
                // Prevent deleting wallets if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allWallets = await prisma.wallet.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedWalletsCount = allWallets.filter(x => !Delete.some(d => d.input === x.id) && x.verified).length;
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
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            organization: "Organization",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data?.organization,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<Prisma.walletSelect>([
            ["organization", "Organization"],
            ["user", "User"],
        ], ...rest),
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
