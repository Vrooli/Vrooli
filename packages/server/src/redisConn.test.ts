import { RedisClientMock } from "./__mocks__/redis";
import { logger as mockLogger } from "./events/__mocks__/logger";
import { logger } from "./events/logger";
import { createRedisClient, initializeRedis, withRedis } from "./redisConn";

jest.mock("./events/logger");

const originalEnv = process.env;
const REDIS_URL = "redis:6379";

describe("createRedisClient function tests", () => {
    beforeEach(() => {
        jest.clearAllMocks();
        process.env = { ...originalEnv, REDIS_URL };
    });

    afterAll(() => {
        process.env = originalEnv;
    });

    it("should create and connect a Redis client successfully", async () => {
        await expect(createRedisClient()).resolves.not.toThrow();
        expect(RedisClientMock.instance?.url).toBe(`redis://${REDIS_URL}`);
    });

    it("should handle 'error' events correctly", async () => {
        const testError = new Error("Test error");
        await createRedisClient();

        // Simulate an 'error' event being emitted
        RedisClientMock.instance?.__emit("error", testError);

        // Verify that the logger.error was called with the correct parameters
        expect(logger.error).toHaveBeenCalledWith("Error occured while connecting or accessing redis server", expect.objectContaining({ trace: "0002", error: testError }));
    });

    it("should handle 'end' events correctly", async () => {
        await createRedisClient();

        // Simulate an 'end' event being emitted
        RedisClientMock.instance?.__emit("end");

        // Verify that the logger.info was called with the correct parameters
        expect(logger.info).toHaveBeenCalledWith("Redis client closed.", expect.objectContaining({ trace: "0208" }));
    });
});

describe("initializeRedis function tests", () => {
    beforeEach(() => {
        // Resetting RedisClientMock before each test
        RedisClientMock.resetMock();
        jest.clearAllMocks();
    });

    it("should initialize a new Redis client when not already initialized", async () => {
        expect(RedisClientMock.instantiationCount).toBe(0);
        await initializeRedis();
        expect(RedisClientMock.instantiationCount).toBe(1);
    });

    it("should not initialize a new Redis client if one is already initialized", async () => {
        expect(RedisClientMock.instantiationCount).toBe(0);
        await initializeRedis();
        await initializeRedis();
        expect(RedisClientMock.instantiationCount).toBe(1);
    });

    it("should return the existing Redis client when already initialized", async () => {
        const redisClient1 = await initializeRedis();
        const redisClient2 = await initializeRedis();
        expect(redisClient1).toBe(redisClient2);
    });
});

describe("withRedis", () => {
    let mockProcess;
    let originalLoggerMethods;

    beforeEach(() => {
        jest.clearAllMocks();
        mockProcess = jest.fn();
        // Save the original logger methods
        originalLoggerMethods = { ...logger };
        // Replace the logger methods with mocks
        Object.assign(logger, mockLogger); // Ensure mockLogger is defined or use jest.fn() for each method you need to mock
    });

    afterEach(() => {
        // Restore the original logger methods after each test
        Object.assign(logger, originalLoggerMethods);
    });

    test("executes process successfully", async () => {
        mockProcess.mockResolvedValueOnce("some value"); // Simulate successful process execution
        const result = await withRedis({ process: mockProcess, trace: "testTrace", traceObject: {} });

        expect(result).toBe(true);
        expect(mockProcess).toHaveBeenCalled();
    });

    test("handles process errors, logs them", async () => {
        const testError = new Error("Test error");
        mockProcess.mockRejectedValueOnce(testError); // Simulate an error thrown by the process
        const result = await withRedis({ process: mockProcess, trace: "testTrace", traceObject: {} });

        expect(result).toBe(false);
        expect(mockProcess).toHaveBeenCalled();
        expect(logger.error).toHaveBeenCalled();
    });
});
