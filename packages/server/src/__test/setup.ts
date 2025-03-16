import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { GenericContainer } from "testcontainers";

// Enable chai-as-promised for async tests
chai.use(chaiAsPromised);

let redisContainer;

before(async function beforeAllTests() {
    this.timeout(60_000); // Allow extra time for container startup

    // Start the Redis container
    redisContainer = await new GenericContainer("redis")
        .withExposedPorts(6379)
        .start();
    // Set the REDIS_URL environment variable
    const redisHost = redisContainer.getHost();
    const redisPort = redisContainer.getMappedPort(6379);
    process.env.REDIS_URL = `redis://${redisHost}:${redisPort}`;
});

after(async function afterAllTests() {
    // Stop the Redis container
    if (redisContainer) {
        await redisContainer.stop();
    }
});
