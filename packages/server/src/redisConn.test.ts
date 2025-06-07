import { describe, it, expect, beforeAll, afterAll } from "vitest";
import sinon from "sinon";
import { logger } from "./events/logger.js";
import { createRedisClient, initializeRedis, withRedis } from "./redisConn.js";

describe("Redis Integration Tests", function redisIntegrationTests() {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    beforeAll(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    afterAll(() => {
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("createRedisClient function tests", () => {
        it("should create and connect a Redis client successfully", async () => {
            const client = await createRedisClient();
            expect(client).to.not.be.null;
            expect(client?.isReady).to.be.true;
            await client?.quit();
        });

        it("should handle 'error' events correctly", async () => {
            loggerErrorStub.reset();

            const client = await createRedisClient();
            const testError = new Error("Test error");
            client?.emit("error", testError);

            expect(loggerErrorStub.calledOnce).to.be.true;
            expect(loggerErrorStub.firstCall.args[0]).to.equal("Error occured while connecting or accessing redis server");
            await client?.quit();
        });

        it("should handle 'end' events correctly", async () => {
            loggerInfoStub.reset();

            const client = await createRedisClient();
            client?.emit("end");

            // Check last call
            const calls = loggerInfoStub.getCalls();
            const lastCall = calls[calls.length - 1];
            expect(lastCall?.args[0]).to.equal("Redis client closed.");
            await client?.quit();
        });
    });

    describe("initializeRedis function tests", () => {
        it("should initialize a new Redis client when not already initialized", async () => {
            const client = await initializeRedis();
            expect(client).to.not.be.null;
            expect(client?.isReady).to.be.true;
            await client?.quit();
        });

        it("should not initialize a new Redis client if one is already initialized", async () => {
            const client1 = await initializeRedis();
            const client2 = await initializeRedis();
            expect(client1).to.equal(client2);
            await client1?.quit();
        });

        it("should return the existing Redis client when already initialized", async () => {
            const client1 = await initializeRedis();
            const client2 = await initializeRedis();
            expect(client1).to.equal(client2);
            await client1?.quit();
        });
    });

    describe("withRedis", () => {
        it("executes process successfully", async () => {
            const processStub = sinon.stub().resolves("some value");
            const success = await withRedis({ process: processStub, trace: "testTrace", traceObject: {} });

            expect(success).to.be.true;
            expect(processStub.calledOnce).to.be.true;
            await (await createRedisClient())?.quit();
        });

        it("handles process errors, logs them", async () => {
            loggerErrorStub.reset();

            const testError = new Error("Test error");
            const processStub = sinon.stub().rejects(testError);

            const success = await withRedis({ process: processStub, trace: "testTrace", traceObject: {} });

            expect(success).to.be.false;
            expect(processStub.calledOnce).to.be.true;
            expect(loggerErrorStub.calledOnce).to.be.true;
            await (await createRedisClient())?.quit();
        });
    });
});
