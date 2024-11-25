import { MaxObjects, walletValidation } from "@local/shared";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { defaultPermissions } from "../../utils";
import { WalletFormat } from "../formats";
import { TeamModelLogic, WalletModelLogic } from "./types";

const __typename = "Wallet" as const;
export const WalletModel: WalletModelLogic = ({
    __typename,
    dbTable: "wallet",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name ?? "",
        },
    }),
    format: WalletFormat,
    mutate: {
        shape: {
            pre: async ({ Delete, userData }) => {
                // Prevent deleting wallets if it will leave you with less than one verified authentication method
                if (Delete.length) {
                    const allWallets = await prismaInstance.wallet.findMany({
                        where: { user: { id: userData.id } },
                        select: { id: true, verified: true },
                    });
                    const remainingVerifiedWalletsCount = allWallets.filter(x => !Delete.some(d => d.input === x.id) && x.verified).length;
                    const verifiedPhonesCount = await prismaInstance.phone.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    const verifiedEmailsCount = await prismaInstance.email.count({
                        where: { user: { id: userData.id }, verified: true },
                    });
                    if (remainingVerifiedWalletsCount + verifiedPhonesCount + verifiedEmailsCount < 1)
                        throw new CustomError("0275", "MustLeaveVerificationMethod");
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
            team: "Team",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Team: data?.team,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: () => false, // Can make public in the future for donations, but for now keep private for security
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        // TODO will have to update how owners are determined in the future. All members shouldn't always be considered owners. Just members with certain roles.
                        { team: ModelMap.get<TeamModelLogic>("Team").query.hasRoleQuery(data.userId) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            // Always private, so it's the same as "own"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("Wallet", "Own", data);
            },
            // Always private, so it's the same as "own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Wallet", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
