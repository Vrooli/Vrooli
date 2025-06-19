import { generatePK, TransferStatus } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

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
export const minimalTransferDb: Prisma.TransferCreateInput = {
    id: transferDbIds.transfer1,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    resource: { connect: { id: "resource_123" } },
    fromUser: { connect: { id: "user_from_123" } },
    toUser: { connect: { id: "user_to_123" } },
};

/**
 * Transfer with complete data including message
 */
export const completeTransferDb: Prisma.TransferCreateInput = {
    id: transferDbIds.transfer2,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "I would like to transfer this resource to you",
    resource: { connect: { id: "resource_456" } },
    fromUser: { connect: { id: "user_from_456" } },
    toUser: { connect: { id: "user_to_456" } },
};

/**
 * Transfer from team to user
 */
export const teamToUserTransferDb: Prisma.TransferCreateInput = {
    id: transferDbIds.transfer3,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "Transferring resource from team to user",
    resource: { connect: { id: "resource_789" } },
    fromTeam: { connect: { id: "team_from_789" } },
    toUser: { connect: { id: "user_to_789" } },
};

/**
 * Transfer from user to team
 */
export const userToTeamTransferDb: Prisma.TransferCreateInput = {
    id: transferDbIds.transfer4,
    status: TransferStatus.Pending,
    initializedByReceiver: false,
    message: "Transferring resource from user to team",
    resource: { connect: { id: "resource_012" } },
    fromUser: { connect: { id: "user_from_012" } },
    toTeam: { connect: { id: "team_to_012" } },
};

/**
 * Accepted transfer with closure date
 */
export const acceptedTransferDb: Prisma.TransferCreateInput = {
    id: transferDbIds.transfer5,
    status: TransferStatus.Accepted,
    initializedByReceiver: false,
    message: "Successfully accepted transfer",
    closedAt: new Date(),
    resource: { connect: { id: "resource_345" } },
    fromUser: { connect: { id: "user_from_345" } },
    toUser: { connect: { id: "user_to_345" } },
};

/**
 * Factory for creating transfer database fixtures with overrides
 */
export class TransferDbFactory {
    static createMinimal(overrides?: Partial<Prisma.TransferCreateInput>): Prisma.TransferCreateInput {
        return {
            ...minimalTransferDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.TransferCreateInput>): Prisma.TransferCreateInput {
        return {
            ...completeTransferDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createTeamToUser(
        fromTeamId: string,
        toUserId: string,
        resourceId: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
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
        fromUserId: string,
        toTeamId: string,
        resourceId: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
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
        fromUserId: string,
        toUserId: string,
        resourceId: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
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
        fromUserId: string,
        toUserId: string,
        resourceId: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
        const baseData = this.createUserToUser(fromUserId, toUserId, resourceId);
        
        let statusSpecificData: Partial<Prisma.TransferCreateInput> = {};
        
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
        fromUserId: string,
        toUserId: string,
        resourceId: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
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
        fromUserId: string,
        toUserId: string,
        resourceId: string,
        denyReason: string,
        overrides?: Partial<Prisma.TransferCreateInput>
    ): Prisma.TransferCreateInput {
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
        fromUserId?: string;
        fromTeamId?: string;
        toUserId?: string;
        toTeamId?: string;
        resourceIds: string[];
        status?: TransferStatus;
        count?: number;
    }
) {
    const transfers = [];
    const count = options.count || options.resourceIds.length;

    for (let i = 0; i < count; i++) {
        const resourceId = options.resourceIds[i % options.resourceIds.length];
        
        let transferData: Prisma.TransferCreateInput;

        // Determine transfer type based on provided IDs
        if (options.fromTeamId && options.toUserId) {
            transferData = TransferDbFactory.createTeamToUser(
                options.fromTeamId,
                options.toUserId,
                resourceId
            );
        } else if (options.fromUserId && options.toTeamId) {
            transferData = TransferDbFactory.createUserToTeam(
                options.fromUserId,
                options.toTeamId,
                resourceId
            );
        } else if (options.fromUserId && options.toUserId) {
            transferData = TransferDbFactory.createUserToUser(
                options.fromUserId,
                options.toUserId,
                resourceId
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
export async function cleanupTransfers(prisma: any, transferIds: string[]) {
    return prisma.transfer.deleteMany({
        where: {
            id: {
                in: transferIds,
            },
        },
    });
}