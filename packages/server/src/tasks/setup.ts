
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
}