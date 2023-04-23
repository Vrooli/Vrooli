import { BlockFrostAPI } from "@blockfrost/blockfrost-js";
import { gql } from "apollo-server-express";
import { updateHelper } from "../actions";
import { getUser } from "../auth";
import { serializedAddressToBech32 } from "../auth/wallet";
import { onlyValidIds } from "../builders";
import { CustomError } from "../events/error";
import { rateLimit } from "../middleware";
export const typeDef = gql `
    input FindHandlesInput {
        organizationId: ID
    }

    input WalletUpdateInput {
        id: ID!
        name: String
    }

    type Wallet {
        id: ID!
        handles: [Handle!]!
        name: String
        publicAddress: String
        stakingAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    type Handle {
        id: ID!
        handle: String!
        wallet: Wallet!
    }

    extend type Query {
        findHandles(input: FindHandlesInput!): [String!]!
    }

    extend type Mutation {
        walletUpdate(input: WalletUpdateInput!): Wallet!
    }
`;
const objectType = "Wallet";
export const resolvers = {
    Query: {
        findHandles: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            const policyID = "f0ff48bbb7bbe9d59a40f1ce90e9e9d0ff5002ec48f232b49ca0fb9a";
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
            const bech32Wallets = wallets.map((wallet) => serializedAddressToBech32(wallet.stakingAddress));
            const Blockfrost = new BlockFrostAPI({
                projectId: process.env.BLOCKFROST_API_KEY ?? "",
                isTestnet: false,
                version: 0,
            });
            const handles = await Promise.all(bech32Wallets.map(async (bech32Wallet) => {
                let handles = [];
                try {
                    const data = await Blockfrost.accountsAddressesAssets(bech32Wallet);
                    handles = data
                        .filter(({ unit }) => unit.startsWith(policyID))
                        .map(({ unit }) => {
                        const hexName = unit.replace(policyID, "");
                        const utf8Name = Buffer.from(hexName, "hex").toString("utf8");
                        return utf8Name;
                    });
                }
                catch (err) {
                    if (err.status_code !== 404) {
                        throw new CustomError("0167", "ExternalServiceError", req.languages);
                    }
                }
                finally {
                    return handles;
                }
            }));
            for (let i = 0; i < handles.length; i++) {
                const currWallet = wallets[i];
                const currHandles = handles[i];
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
                const badLinks = existingHandles.filter(({ wallet }) => wallet !== null && wallet.id !== currWallet.id);
                if (badLinks.length > 0) {
                    await prisma.handle.updateMany({
                        where: { id: { in: badLinks.map(({ id }) => id) } },
                        data: { walletId: currWallet.id },
                    });
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
//# sourceMappingURL=wallet.js.map