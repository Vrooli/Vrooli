import { BlockFrostAPI } from "@blockfrost/blockfrost-js";
import { FindHandlesInput, Wallet, WalletUpdateInput } from "@local/shared";
import { updateHelper } from "../../actions";
import { getUser, serializedAddressToBech32 } from "../../auth";
import { onlyValidIds } from "../../builders";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsWallet = {
    Query: {
        findHandles: GQLEndpoint<FindHandlesInput, string[]>;
    },
    Mutation: {
        walletUpdate: GQLEndpoint<WalletUpdateInput, UpdateOneResult<Wallet>>;
    }
}

const objectType = "Wallet";
export const WalletEndpoints: EndpointsWallet = {
    Query: {
        /**
         * Finds all ADA handles for your profile, or an organization you belong to.
         */
        findHandles: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            // const policyID = 'de95598bb370b6d289f42dfc1de656d65c250ec4cdc930d32b1dc0e5'; // Fake policy ID for testing
            const policyID = "f0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9a"; // Mainnet ADA Handle policy ID
            const walletFields = {
                id: true,
                stakingAddress: true,
                publicAddress: true,
                name: true,
                verified: true,
                handles: {
                    select: { handle: true },
                },
            };
            // Find all wallets associated with user or organization
            let wallets;
            if (input.organizationId) {
                wallets = await prisma.wallet.findMany({
                    where: { organizationId: input.organizationId },
                    select: walletFields,
                });
            }
            else {
                const userId = getUser(req)?.id;
                if (!userId)
                    throw new CustomError("0166", "NotLoggedIn", req.languages);
                wallets = await prisma.wallet.findMany({
                    where: { userId },
                    select: walletFields,
                });
            }
            // Convert wallet staking addresses to Bech32
            const bech32Wallets = wallets.map((wallet) => serializedAddressToBech32(wallet.stakingAddress));
            // Initialize blockfrost client
            const Blockfrost = new BlockFrostAPI({
                projectId: process.env.BLOCKFROST_API_KEY ?? "",
                isTestnet: false,
                version: 0,
            });
            // Query each wallet with blockfrost to get handles (/accounts/{stake_address}/addresses/assets)
            const handles: string[][] = await Promise.all(bech32Wallets.map(async (bech32Wallet: string) => {
                let handles: string[] = [];
                try {
                    const data = await Blockfrost.accountsAddressesAssets(bech32Wallet);
                    // Find handles. Each asset will look something like this:
                    // {"unit":"de95598bb370b6d289f42dfc1de656d65c250ec4cdc930d32b1dc0e5474f4f5345","quantity":"1"}
                    // The unit is the concatenation of the policy ID (56 characters long) and the asset name (hex encoded)
                    handles = data
                        .filter(({ unit }: any) => unit.startsWith(policyID))
                        .map(({ unit }: any) => {
                            const hexName = unit.replace(policyID, "");
                            const utf8Name = Buffer.from(hexName, "hex").toString("utf8");
                            return utf8Name;
                        });
                } catch (err: any) {
                    // If code is 404, then resource does not exist. This means that the wallet has no transactions.
                    // In this case, we shouldn't throw an error
                    if (err.status_code !== 404) {
                        throw new CustomError("0167", "ExternalServiceError", req.languages);
                    }
                } finally {
                    return handles;
                }
            }));
            // Store handles in database, so we can verify them later
            for (let i = 0; i < handles.length; i++) {
                const currWallet = wallets[i];
                const currHandles = handles[i];
                // Find existing handles
                const existingHandles = await prisma.handle.findMany({
                    where: { handle: { in: currHandles } },
                    select: {
                        id: true,
                        handle: true,
                        wallet: {
                            select: {
                                id: true,
                                userId: true,
                                organizationId: true,
                            },
                        },
                    },
                });
                // If any of the existing handles are not associated with the current wallet, change that
                const badLinks = existingHandles.filter(({ wallet }) => wallet !== null && wallet.id !== currWallet.id);
                if (badLinks.length > 0) {
                    await prisma.handle.updateMany({
                        where: { id: { in: badLinks.map(({ id }) => id) } },
                        data: { walletId: currWallet.id },
                    });
                    // Remove the handle from the user or organization linked to a wallet with one of the bad handles
                    const userIds = onlyValidIds(badLinks.map(({ wallet }) => wallet?.userId));
                    await prisma.user.updateMany({
                        where: { id: { in: userIds } },
                        data: { handle: null },
                    });
                    const organizationIds = onlyValidIds(badLinks.map(({ wallet }) => wallet?.organizationId));
                    await prisma.organization.updateMany({
                        where: { id: { in: organizationIds } },
                        data: { handle: null },
                    });
                }
                // If any handles did not exist in the database already, add them
                const newHandles = currHandles.filter(handle => !existingHandles.some(({ handle: existingHandle }) => existingHandle === handle));
                if (newHandles.length > 0) {
                    await prisma.handle.createMany({
                        data: newHandles.map(handle => ({
                            handle,
                            walletId: currWallet.id,
                        })),
                    });
                }
            }
            // Flatten handles
            const flatHandles = handles.reduce((acc, curr) => acc.concat(curr), []);
            return flatHandles;
        },
    },
    Mutation: {
        walletUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
