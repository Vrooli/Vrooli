/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK, TransferStatus } from "@vrooli/shared";

/**
 * Database fixtures for Transfer model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const transferDbIds = {
    transfer1: generatePK(),
    transfer2: generatePK(),
    transfer3: generatePK(),
    transfer4: generatePK(),
    transfer5: generatePK(),
};

/**
 * Minimal transfer data for database creation
 */
export const minimalTransferDb: Prisma.transferCreateInput = {
    id: transferDbIds.transfer1,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    resource: { connect: { id: BigInt(123) } },
    fromUser: { connect: { id: BigInt(456) } },
    toUser: { connect: { id: BigInt(789) } },
};

/**
 * Transfer with complete data including message
 */
export const completeTransferDb: Prisma.transferCreateInput = {
    id: transferDbIds.transfer2,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "I would like to transfer this resource to you",
    resource: { connect: { id: BigInt(234) } },
    fromUser: { connect: { id: BigInt(567) } },
    toUser: { connect: { id: BigInt(890) } },
};

/**
 * Transfer from team to user
 */
export const teamToUserTransferDb: Prisma.transferCreateInput = {
    id: transferDbIds.transfer3,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "Transferring resource from team to user",
    resource: { connect: { id: BigInt(345) } },
    fromTeam: { connect: { id: BigInt(678) } },
    toUser: { connect: { id: BigInt(901) } },
};

/**
 * Transfer from user to team
 */
export const userToTeamTransferDb: Prisma.transferCreateInput = {
    id: transferDbIds.transfer4,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "Transferring resource from user to team",
    resource: { connect: { id: BigInt(456) } },
    fromUser: { connect: { id: BigInt(789) } },
    toTeam: { connect: { id: BigInt(12) } },
};

/**
 * Accepted transfer with closure date
 */
export const acceptedTransferDb: Prisma.transferCreateInput = {
    id: transferDbIds.transfer5,
    status: TransferStatus.Accepted,
    initializedByReceiver: false,
    message: "Successfully accepted transfer",
    closedAt: new Date(),
    resource: { connect: { id: BigInt(567) } },
    fromUser: { connect: { id: BigInt(890) } },
    toUser: { connect: { id: BigInt(123) } },
};

/**
 * Factory for creating transfer database fixtures with overrides
 */
export class TransferDbFactory {
    static createMinimal(overrides?: Partial<Prisma.transferCreateInput>): Prisma.transferCreateInput {
        return {
            ...minimalTransferDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.transferCreateInput>): Prisma.transferCreateInput {
        return {
            ...completeTransferDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createTeamToUser(
        fromTeamId: bigint,
        toUserId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        return {
            ...teamToUserTransferDb,
            id: generatePK(),
            fromTeam: { connect: { id: fromTeamId } },
            toUser: { connect: { id: toUserId } },
            resource: { connect: { id: resourceId } },
            ...overrides,
        };
    }

    static createUserToTeam(
        fromUserId: bigint,
        toTeamId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        return {
            ...userToTeamTransferDb,
            id: generatePK(),
            fromUser: { connect: { id: fromUserId } },
            toTeam: { connect: { id: toTeamId } },
            resource: { connect: { id: resourceId } },
            ...overrides,
        };
    }

    static createUserToUser(
        fromUserId: bigint,
        toUserId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        return {
            ...minimalTransferDb,
            id: generatePK(),
            fromUser: { connect: { id: fromUserId } },
            toUser: { connect: { id: toUserId } },
            resource: { connect: { id: resourceId } },
            ...overrides,
        };
    }

    /**
     * Create transfer with specific status
     */
    static createWithStatus(
        status: TransferStatus,
        fromUserId: bigint,
        toUserId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        const baseData = this.createUserToUser(fromUserId, toUserId, resourceId);

        const statusSpecificData: Partial<Prisma.transferCreateInput> = {};

        if (status === TransferStatus.Accepted || status === TransferStatus.Denied) {
            statusSpecificData.closedAt = new Date();
        }

        if (status === TransferStatus.Denied) {
            statusSpecificData.denyReason = "Transfer request was denied";
        }

        return {
            ...baseData,
            status,
            ...statusSpecificData,
            ...overrides,
        };
    }

    /**
     * Create transfer initialized by receiver (incoming request)
     */
    static createInitializedByReceiver(
        fromUserId: bigint,
        toUserId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        return {
            ...this.createUserToUser(fromUserId, toUserId, resourceId),
            initializedByReceiver: true,
            message: "Requesting transfer of this resource",
            ...overrides,
        };
    }

    /**
     * Create denied transfer with reason
     */
    static createDenied(
        fromUserId: bigint,
        toUserId: bigint,
        resourceId: bigint,
        denyReason: string,
        overrides?: Partial<Prisma.transferCreateInput>,
    ): Prisma.transferCreateInput {
        return {
            ...this.createUserToUser(fromUserId, toUserId, resourceId),
            status: TransferStatus.Denied,
            closedAt: new Date(),
            denyReason,
            ...overrides,
        };
    }
}

/**
 * Helper to seed transfers for testing
 */
export async function seedTransfers(
    prisma: any,
    options: {
        fromUserId?: bigint;
        fromTeamId?: bigint;
        toUserId?: bigint;
        toTeamId?: bigint;
        resourceIds: bigint[];
        status?: TransferStatus;
        count?: number;
    },
) {
    const transfers: any[] = [];
    const count = options.count || options.resourceIds.length;

    for (let i = 0; i < count; i++) {
        const resourceId = options.resourceIds[i % options.resourceIds.length];

        let transferData: Prisma.transferCreateInput;

        // Determine transfer type based on provided IDs
        if (options.fromTeamId && options.toUserId) {
            transferData = TransferDbFactory.createTeamToUser(
                options.fromTeamId,
                options.toUserId,
                resourceId,
            );
        } else if (options.fromUserId && options.toTeamId) {
            transferData = TransferDbFactory.createUserToTeam(
                options.fromUserId,
                options.toTeamId,
                resourceId,
            );
        } else if (options.fromUserId && options.toUserId) {
            transferData = TransferDbFactory.createUserToUser(
                options.fromUserId,
                options.toUserId,
                resourceId,
            );
        } else {
            throw new Error("Must provide valid from/to combinations (user/team)");
        }

        // Apply status if specified
        if (options.status) {
            transferData.status = options.status;
            if (options.status === TransferStatus.Accepted || options.status === TransferStatus.Denied) {
                transferData.closedAt = new Date();
            }
            if (options.status === TransferStatus.Denied) {
                transferData.denyReason = "Transfer denied in test setup";
            }
        }

        const transfer = await prisma.transfer.create({
            data: transferData,
            include: {
                fromUser: true,
                fromTeam: true,
                toUser: true,
                toTeam: true,
                resource: true,
            },
        });

        transfers.push(transfer);
    }

    return transfers;
}

/**
 * Helper to clean up test transfers
 */
export async function cleanupTransfers(prisma: PrismaClient, transferIds: bigint[]) {
    return prisma.transfer.deleteMany({
        where: {
            id: {
                in: transferIds,
            },
        },
    });
}
