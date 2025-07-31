// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for Wallet model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const walletDbIds = {
    wallet1: generatePK(),
    wallet2: generatePK(),
    wallet3: generatePK(),
    wallet4: generatePK(),
    wallet5: generatePK(),
};

/**
 * Minimal wallet data for database creation
 */
export const minimalWalletDb: Prisma.walletCreateInput = {
    id: walletDbIds.wallet1,
    stakingAddress: "addr1qyznc4mr2huxc0gg8l3hh4x5e0yxhj9jjxjh9r5v6f8z8n2q3l7k",
    name: "Test Wallet",
    verifiedAt: new Date("2024-01-01T00:00:00Z"),
};

/**
 * Wallet with public address
 */
export const walletWithPublicAddressDb: Prisma.walletCreateInput = {
    id: walletDbIds.wallet2,
    stakingAddress: "addr1qzfrx8jr5ug9sd9xf6tg2nh4p7r8q5l3m6v9w2k8j7h4n6c9b5x",
    publicAddress: "addr1_public_xyz789abc123def456ghi789jkl012mno345pqr678",
    name: "Wallet with Public Address",
    verifiedAt: new Date("2024-01-01T00:00:00Z"),
};

/**
 * Complete wallet with all features
 */
export const completeWalletDb: Prisma.walletCreateInput = {
    id: walletDbIds.wallet3,
    stakingAddress: "addr1qw8m9n2r5t6y7u8i9o0p1a2s3d4f5g6h7j8k9l0z1x2c3v4b5n",
    publicAddress: "addr1_complete_abc123def456ghi789jkl012mno345pqr678stu901",
    name: "Complete Test Wallet",
    nonce: "test-nonce-123456789",
    nonceCreationTime: new Date("2024-01-01T00:00:00Z"),
    verifiedAt: new Date("2024-01-01T00:00:00Z"),
    wasReported: false,
};

/**
 * Unverified wallet (pending verification)
 */
export const unverifiedWalletDb: Prisma.walletCreateInput = {
    id: walletDbIds.wallet4,
    stakingAddress: "addr1qr5t6y7u8i9o0p1a2s3d4f5g6h7j8k9l0z1x2c3v4b5n6m7p8q",
    name: "Unverified Wallet",
    nonce: "pending-nonce-987654321",
    nonceCreationTime: new Date(),
    verifiedAt: null,
    wasReported: false,
};

/**
 * Reported wallet (flagged for security)
 */
export const reportedWalletDb: Prisma.walletCreateInput = {
    id: walletDbIds.wallet5,
    stakingAddress: "addr1qp8q9r0s1t2u3v4w5x6y7z8a9b0c1d2e3f4g5h6i7j8k9l0m1n",
    name: "Reported Wallet",
    verifiedAt: new Date("2024-01-01T00:00:00Z"),
    wasReported: true,
};

/**
 * Factory for creating wallet database fixtures with overrides
 */
export class WalletDbFactory {
    static createMinimal(overrides?: Partial<Prisma.walletCreateInput>): Prisma.walletCreateInput {
        return {
            ...minimalWalletDb,
            id: generatePK(),
            stakingAddress: this.generateStakingAddress(),
            ...overrides,
        };
    }

    static createWithPublicAddress(overrides?: Partial<Prisma.walletCreateInput>): Prisma.walletCreateInput {
        return {
            ...walletWithPublicAddressDb,
            id: generatePK(),
            stakingAddress: this.generateStakingAddress(),
            publicAddress: this.generatePublicAddress(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.walletCreateInput>): Prisma.walletCreateInput {
        return {
            ...completeWalletDb,
            id: generatePK(),
            stakingAddress: this.generateStakingAddress(),
            publicAddress: this.generatePublicAddress(),
            nonce: this.generateNonce(),
            ...overrides,
        };
    }

    static createUnverified(overrides?: Partial<Prisma.walletCreateInput>): Prisma.walletCreateInput {
        return {
            ...unverifiedWalletDb,
            id: generatePK(),
            stakingAddress: this.generateStakingAddress(),
            nonce: this.generateNonce(),
            nonceCreationTime: new Date(),
            verifiedAt: null,
            ...overrides,
        };
    }

    static createReported(overrides?: Partial<Prisma.walletCreateInput>): Prisma.walletCreateInput {
        return {
            ...reportedWalletDb,
            id: generatePK(),
            stakingAddress: this.generateStakingAddress(),
            wasReported: true,
            ...overrides,
        };
    }

    /**
     * Create wallet for specific user
     */
    static createForUser(
        userId: string,
        overrides?: Partial<Prisma.walletCreateInput>,
    ): Prisma.walletCreateInput {
        return {
            ...this.createComplete(overrides),
            user: { connect: { id: BigInt(userId) } },
        };
    }

    /**
     * Create wallet for specific team
     */
    static createForTeam(
        teamId: string,
        overrides?: Partial<Prisma.walletCreateInput>,
    ): Prisma.walletCreateInput {
        return {
            ...this.createComplete(overrides),
            team: { connect: { id: BigInt(teamId) } },
        };
    }

    /**
     * Create wallet with both user and team associations
     */
    static createForUserAndTeam(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.walletCreateInput>,
    ): Prisma.walletCreateInput {
        return {
            ...this.createComplete(overrides),
            user: { connect: { id: BigInt(userId) } },
            team: { connect: { id: BigInt(teamId) } },
        };
    }

    /**
     * Generate a realistic staking address
     */
    private static generateStakingAddress(): string {
        const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        let address = "addr1q";
        for (let i = 0; i < 50; i++) {
            address += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return address;
    }

    /**
     * Generate a realistic public address
     */
    private static generatePublicAddress(): string {
        const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        let address = "addr1_";
        for (let i = 0; i < 50; i++) {
            address += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return address;
    }

    /**
     * Generate a test nonce
     */
    private static generateNonce(): string {
        return `nonce-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }
}

/**
 * Helper to create a wallet that matches verification flow requirements
 */
export function createWalletForVerification(
    stakingAddress: string,
    overrides?: Partial<Prisma.walletCreateInput>,
) {
    return {
        ...WalletDbFactory.createUnverified(overrides),
        stakingAddress,
        nonce: WalletDbFactory["generateNonce"](),
        nonceCreationTime: new Date(),
        verifiedAt: null,
    };
}

/**
 * Helper to seed multiple test wallets
 */
export async function seedTestWallets(
    prisma: any,
    count = 3,
    options?: {
        userId?: string;
        teamId?: string;
        verified?: boolean;
        withPublicAddress?: boolean;
    },
) {
    const wallets = [];

    for (let i = 0; i < count; i++) {
        let walletData: Prisma.walletCreateInput;

        if (options?.verified === false) {
            walletData = WalletDbFactory.createUnverified({
                name: `Test Wallet ${i + 1}`,
            });
        } else if (options?.withPublicAddress) {
            walletData = WalletDbFactory.createWithPublicAddress({
                name: `Test Wallet ${i + 1}`,
            });
        } else {
            walletData = WalletDbFactory.createComplete({
                name: `Test Wallet ${i + 1}`,
            });
        }

        // Add user/team associations if provided
        if (options?.userId) {
            walletData.user = { connect: { id: BigInt(options.userId) } };
        }
        if (options?.teamId) {
            walletData.team = { connect: { id: BigInt(options.teamId) } };
        }

        wallets.push(await prisma.wallet.create({ data: walletData }));
    }

    return wallets;
}

/**
 * Helper to clean up test wallets
 */
export async function cleanupTestWallets(prisma: PrismaClient, walletIds: string[]) {
    await prisma.wallet.deleteMany({
        where: {
            id: {
                in: walletIds.map(id => BigInt(id)),
            },
        },
    });
}
