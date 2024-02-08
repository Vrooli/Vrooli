import pkg from "../__mocks__/@prisma/client";
import { logger as mockLogger } from "../events/__mocks__/logger";
import { logger } from "../events/logger";
import { withPrisma } from "./withPrisma";

const { PrismaClient } = pkg;

jest.mock("@prisma/client");
jest.mock("../events/logger");

describe("withPrisma", () => {
    let mockProcess, prisma, originalLoggerMethods;

    beforeEach(() => {
        jest.clearAllMocks();
        mockProcess = jest.fn();
        prisma = new PrismaClient();
        // Save the original logger methods
        originalLoggerMethods = { ...logger };
        // Replace the logger methods with mocks
        Object.assign(logger, mockLogger);
    });

    afterEach(() => {
        // Restore the original logger methods after each test
        Object.assign(logger, originalLoggerMethods);
    });

    test("executes process successfully and disconnects from Prisma", async () => {
        mockProcess.mockResolvedValueOnce("some value"); // Simulate successful process execution
        const result = await withPrisma({ process: mockProcess, trace: "testTrace", traceObject: {} });

        expect(result).toBe(true);
        expect(mockProcess).toHaveBeenCalled();
        expect(prisma.$disconnect).toHaveBeenCalled();
    });

    test("handles process errors, logs them, and disconnects from Prisma", async () => {
        const testError = new Error("Test error");
        mockProcess.mockRejectedValueOnce(testError); // Simulate an error thrown by the process
        const result = await withPrisma({ process: mockProcess, trace: "testTrace", traceObject: {} });

        expect(result).toBe(false);
        expect(mockProcess).toHaveBeenCalled();
        expect(logger.error).toHaveBeenCalled();
        expect(prisma.$disconnect).toHaveBeenCalled();
    });
});
