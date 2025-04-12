/**
 * Maintain references to all task queues for proper cleanup
 */
import { BaseQueue } from "./base/queue.js";

// Store references to all queues for proper cleanup
const allQueues: BaseQueue<any>[] = [];

/**
 * Setup the task queues
 */
export async function setupTaskQueues(): Promise<void> {
    const setupLlmTaskQueue = (await import("./llmTask/queue.js")).setupLlmTaskQueue;
    const setupEmailQueue = (await import("./email/queue.js")).setupEmailQueue;
    const setupExportQueue = (await import("./export/queue.js")).setupExportQueue;
    const setupImportQueue = (await import("./import/queue.js")).setupImportQueue;
    const setupLlmQueue = (await import("./llm/queue.js")).setupLlmQueue;
    const setupPushQueue = (await import("./push/queue.js")).setupPushQueue;
    const setupSandboxQueue = (await import("./sandbox/queue.js")).setupSandboxQueue;
    const setupSmsQueue = (await import("./sms/queue.js")).setupSmsQueue;
    const setupRunQueue = (await import("./run/queue.js")).setupRunQueue;

    await setupLlmTaskQueue();
    await setupEmailQueue();
    await setupExportQueue();
    await setupImportQueue();
    await setupLlmQueue();
    await setupPushQueue();
    await setupSandboxQueue();
    await setupSmsQueue();
    await setupRunQueue();

    // Add queues to the allQueues array for tracking
    allQueues.push(
        (await import("./llmTask/queue.js")).llmTaskQueue,
        (await import("./email/queue.js")).emailQueue,
        (await import("./export/queue.js")).exportQueue,
        (await import("./import/queue.js")).importQueue,
        (await import("./llm/queue.js")).llmQueue,
        (await import("./push/queue.js")).pushQueue,
        (await import("./sandbox/queue.js")).sandboxQueue,
        (await import("./sms/queue.js")).smsQueue,
        (await import("./run/queue.js")).runQueue
    );
}

/**
 * Close all task queues and their Redis connections
 * This is particularly important in test environments to prevent hanging
 */
export async function closeTaskQueues(): Promise<void> {
    console.info(`Closing ${allQueues.length} task queues...`);

    // Close all queues one by one to better track failures
    for (const queue of allQueues) {
        if (queue) {
            try {
                await queue.close();
            } catch (error) {
                console.error(`Error closing queue ${queue.getQueue()?.name} (continuing anyway):`, error);
            }
        }
    }

    // Clear the array
    console.info(`Finished closing all queues. Clearing queue references.`);
    allQueues.length = 0;
}