/**
 * Integration test for database factories
 * Tests that factories can be instantiated and basic operations work
 */
import { type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CreditAccountDbFactory } from "./CreditAccountDbFactory.js";
import { MemberDbFactory } from "./MemberDbFactory.js";

// Mock PrismaClient for testing
const mockPrisma = {
    credit_account: {
        create: vi.fn().mockImplementation(({ data }) => Promise.resolve({ ...data, id: data.id || generatePK() })),
        findUnique: vi.fn(),
        update: vi.fn(),
        delete: vi.fn(),
        deleteMany: vi.fn(),
    },
    member: {
        create: vi.fn().mockImplementation(({ data }) => Promise.resolve({
            ...data,
            id: data.id || generatePK(),
            publicId: "test12345678",
            createdAt: new Date(),
            updatedAt: new Date(),
            permissions: {},
        })),
        findUnique: vi.fn().mockResolvedValue(null),
        update: vi.fn(),
        delete: vi.fn(),
        deleteMany: vi.fn(),
        count: vi.fn().mockResolvedValue(0),
    },
    user: {
        create: vi.fn().mockImplementation(({ data }) => Promise.resolve({ ...data, id: data.id || generatePK() })),
    },
    team: {
        create: vi.fn().mockImplementation(({ data }) => Promise.resolve({ ...data, id: data.id || generatePK() })),
    },
    $transaction: vi.fn().mockImplementation((fn) => fn(mockPrisma)),
} as unknown as PrismaClient;

describe("Database Factory Integration Tests", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("CreditAccountDbFactory", () => {
        it("should instantiate correctly", () => {
            const factory = new CreditAccountDbFactory(mockPrisma);
            expect(factory).toBeInstanceOf(CreditAccountDbFactory);
        });

        it("should create minimal account", async () => {
            const factory = new CreditAccountDbFactory(mockPrisma);
            const account = await factory.createMinimal();

            expect(mockPrisma.credit_account.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    currentBalance: BigInt(1000),
                    user: { connect: expect.objectContaining({ id: expect.any(BigInt) }) },
                }),
                include: expect.anything(),
            });

            expect(account).toHaveProperty("id");
            expect(account).toHaveProperty("currentBalance", BigInt(1000));
        });

        it("should create account with specific balance", async () => {
            const factory = new CreditAccountDbFactory(mockPrisma);
            const balance = BigInt(5000);
            const account = await factory.createWithBalance(balance);

            expect(mockPrisma.credit_account.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    currentBalance: balance,
                }),
                include: expect.anything(),
            });
        });

        it("should create account for user", async () => {
            const factory = new CreditAccountDbFactory(mockPrisma);
            const userId = generatePK();
            const account = await factory.createForUser(userId);

            expect(mockPrisma.credit_account.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    currentBalance: BigInt(1000),
                    user: { connect: { id: userId } },
                    team: undefined,
                }),
                include: expect.anything(),
            });
        });

        it("should create account for team", async () => {
            const factory = new CreditAccountDbFactory(mockPrisma);
            const teamId = generatePK();
            const account = await factory.createForTeam(teamId);

            expect(mockPrisma.credit_account.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    currentBalance: BigInt(1000),
                    team: { connect: { id: teamId } },
                    user: undefined,
                }),
            });
        });
    });

    describe("MemberDbFactory", () => {
        it("should instantiate correctly", () => {
            const factory = new MemberDbFactory(mockPrisma);
            expect(factory).toBeInstanceOf(MemberDbFactory);
        });

        it("should create minimal member", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const member = await factory.createMinimal();

            expect(mockPrisma.member.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    isAdmin: false,
                    user: { connect: expect.objectContaining({ id: expect.any(BigInt) }) },
                    team: { connect: expect.objectContaining({ id: expect.any(BigInt) }) },
                }),
                include: expect.anything(),
            });

            expect(member).toHaveProperty("id");
            expect(member).toHaveProperty("isAdmin", false);
        });

        it("should create owner", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const userId = generatePK();
            const teamId = generatePK();
            const owner = await factory.createOwner(userId, teamId);

            expect(mockPrisma.member.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    isAdmin: true,
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                }),
                include: expect.anything(),
            });
        });

        it("should create admin", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const userId = generatePK();
            const teamId = generatePK();
            const admin = await factory.createAdmin(userId, teamId);

            expect(mockPrisma.member.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    isAdmin: true,
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                }),
                include: expect.anything(),
            });
        });

        it("should verify admin status correctly", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const memberId = generatePK();
            const expectedRole = "Admin";

            (mockPrisma.member.findUnique as any).mockResolvedValueOnce({
                id: memberId,
                publicId: "test12345678",
                createdAt: new Date(),
                updatedAt: new Date(),
                isAdmin: true,
                permissions: {},
                userId: generatePK(),
                teamId: generatePK(),
            });

            const member = await factory.verifyRole(memberId, expectedRole);
            expect(member.isAdmin).toBe(true);
        });

        it("should throw error for admin status mismatch", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const memberId = generatePK();

            (mockPrisma.member.findUnique as any).mockResolvedValueOnce({
                id: memberId,
                publicId: "test12345678",
                createdAt: new Date(),
                updatedAt: new Date(),
                isAdmin: false,
                permissions: {},
                userId: generatePK(),
                teamId: generatePK(),
            });

            await expect(factory.verifyRole(memberId, "Owner"))
                .rejects.toThrow("Role mismatch: expected Owner, got Member");
        });

        it("should verify team member count", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const teamId = generatePK();
            const expectedCount = 5;

            (mockPrisma.member.count as any).mockResolvedValueOnce(expectedCount);

            const count = await factory.verifyTeamMemberCount(teamId, expectedCount);
            expect(count).toBe(expectedCount);
        });

        it("should create multiple team members", async () => {
            const factory = new MemberDbFactory(mockPrisma);
            const teamId = generatePK();
            const members = [
                { userId: generatePK(), isAdmin: true }, // Owner-like
                { userId: generatePK(), isAdmin: true }, // Admin
                { userId: generatePK(), isAdmin: false }, // Regular member
            ];

            await factory.createTeamMembers(teamId, members);

            expect(mockPrisma.member.create).toHaveBeenCalledTimes(3);
            expect(mockPrisma.$transaction).toHaveBeenCalled();
        });
    });

    describe("Factory Fixtures", () => {
        it("should provide valid minimal fixtures", () => {
            const creditFactory = new CreditAccountDbFactory(mockPrisma);
            const creditFixtures = creditFactory["getFixtures"]();

            expect(creditFixtures.minimal).toHaveProperty("currentBalance");
            expect(creditFixtures.minimal.currentBalance).toBe(BigInt(1000));

            const memberFactory = new MemberDbFactory(mockPrisma);
            const memberFixtures = memberFactory["getFixtures"]();

            expect(memberFixtures.minimal).toHaveProperty("isAdmin");
            expect(memberFixtures.minimal.isAdmin).toBe(false);
        });

        it("should provide edge case fixtures", () => {
            const creditFactory = new CreditAccountDbFactory(mockPrisma);
            const creditFixtures = creditFactory["getFixtures"]();

            expect(creditFixtures.edgeCases).toHaveProperty("zeroBalance");
            expect(creditFixtures.edgeCases.zeroBalance.currentBalance).toBe(BigInt(0));
            expect(creditFixtures.edgeCases).toHaveProperty("highBalance");
            expect(creditFixtures.edgeCases.highBalance.currentBalance).toBe(BigInt(10000000));

            const memberFactory = new MemberDbFactory(mockPrisma);
            const memberFixtures = memberFactory["getFixtures"]();

            // Member factory may still use variants - check what it actually provides
            expect(memberFixtures).toHaveProperty("minimal");
            expect(memberFixtures.minimal.isAdmin).toBe(false);
        });

        it("should provide update fixtures", () => {
            const creditFactory = new CreditAccountDbFactory(mockPrisma);
            const creditFixtures = creditFactory["getFixtures"]();

            expect(creditFixtures.updates).toHaveProperty("addCredits");
            expect(creditFixtures.updates?.addCredits.currentBalance).toBe(BigInt(5000));
            expect(creditFixtures.updates).toHaveProperty("zeroOut");
            expect(creditFixtures.updates?.zeroOut.currentBalance).toBe(BigInt(0));

            const memberFactory = new MemberDbFactory(mockPrisma);
            const memberFixtures = memberFactory["getFixtures"]();

            // Member factory may still use different structure - will check separately
            expect(memberFixtures).toHaveProperty("minimal");
        });
    });
});
