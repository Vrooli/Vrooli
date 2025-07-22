import { BUSINESS_NAME, generatePK } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { clearRedisCache } from "../queueFactory.js";
import { QueueService } from "../queues.js";
import { QueueTaskType } from "../taskTypes.js";
import { AUTH_EMAIL_TEMPLATES } from "./queue.js";
import { createValidTaskData } from "../taskFactory.js";


describe("Email Queue Tests (BullMQ)", () => {
    let queueService: QueueService;
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";
    const originalSiteEmailUsername = process.env.SITE_EMAIL_USERNAME;

    beforeEach(async () => {
        // Set test email for admin
        process.env.SITE_EMAIL_USERNAME = "admin@example.com";

        // Get fresh instance and initialize
        queueService = QueueService.get();
        await queueService.init(redisUrl);
    });

    afterEach(async () => {
        // Restore env variable
        process.env.SITE_EMAIL_USERNAME = originalSiteEmailUsername;

        // Clean shutdown - order is critical to prevent "Connection is closed" errors
        try {
            await queueService.shutdown();
            // Wait for shutdown to fully complete and event handlers to detach
            await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
            console.log("Shutdown error (ignored):", error);
        }
        // Clear singleton before clearing cache to prevent any access during cleanup
        (QueueService as any).instance = null;
        // Clear Redis cache last to avoid disconnecting connections still in use
        clearRedisCache();
        // Final delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    /**
     * Helper function to get the latest job from email queue and verify its properties
     * @param expectedData The expected data object to validate against.
     */
    async function expectEmailToBeEnqueuedWith(expectedData) {
        // Get all jobs from the email queue
        const jobs = await queueService.email.queue.getJobs(["waiting", "active", "completed", "failed"]);
        expect(jobs.length).toBeGreaterThan(0);

        // Get the most recent job
        const latestJob = jobs[jobs.length - 1];
        const actualData = latestJob.data;

        // Check if it matches expected data
        if (expectedData) {
            if (expectedData.to) {
                expect(actualData.to).toEqual(expectedData.to);
            }
            if (expectedData.subject) {
                expect(actualData.subject).toBe(expectedData.subject);
            }
            if (expectedData.text) {
                expect(actualData.text).toBe(expectedData.text);
            }
            // Special handling for html field, which can be undefined
            if ("html" in expectedData) {
                if (expectedData.html === undefined) {
                    expect(actualData.html).toBeUndefined();
                } else {
                    expect(actualData.html).toBe(expectedData.html);
                }
            }
        }

        return actualData;
    }

    describe("Direct email sending tests", () => {
        it("enqueues an email with all parameters provided", async () => {
            const testEmails = ["user1@example.com", "user2@example.com"];
            const subject = "Test Subject";
            const text = "Test email body";
            const html = "<p>Test email body</p>";
            const delay = 5000; // 5 seconds

            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: testEmails,
                    subject,
                    text,
                    html,
                }), 
                { delay },
            );

            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            await expectEmailToBeEnqueuedWith({
                to: testEmails,
                subject,
                text,
                html,
            });

            // Check if the delay option is set correctly
            const job = await queueService.email.queue.getJob(result.data!.id);
            expect(job?.opts.delay).toBe(delay);
        });

        it("enqueues an email without the html parameter", async () => {
            const testEmails = ["user@example.com"];
            const subject = "No HTML Subject";
            const text = "Email body without HTML";

            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: testEmails,
                    subject,
                    text,
                }),
            );

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: testEmails,
                subject,
                text,
                html: undefined,
            });

            // Check that default delay is 0
            const job = await queueService.email.queue.getJob(result.data!.id);
            expect(job?.opts.delay).toBeUndefined(); // BullMQ doesn't set delay if it's 0
        });

        it("enqueues an email with a delay", async () => {
            const delay = 10000; // 10 seconds

            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["delayed@example.com"],
                    subject: "Delayed Email",
                    text: "This is a delayed email.",
                }), 
                { delay },
            );

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: ["delayed@example.com"],
                subject: "Delayed Email",
                text: "This is a delayed email.",
                html: undefined,
            });

            // Check if the delay option is set correctly
            const job = await queueService.email.queue.getJob(result.data!.id);
            expect(job?.opts.delay).toBe(delay);
        });

        it("handles empty recipient list", async () => {
            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: [],
                    subject: "Subject",
                    text: "Text body",
                }),
            );

            // The task should be added but may fail during processing
            expect(result.success).toBe(true);
        });
    });

    describe("Email template tests", () => {
        const testEmail = "valid.email@example.com";

        it("correctly enqueues email for ResetPassword template", async () => {
            const userId = "user123";
            const link = "https://example.com/reset/user123:code123";

            const template = AUTH_EMAIL_TEMPLATES.ResetPassword(userId, link);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            const actualData = await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });

            // Check template properties
            expect(actualData.subject).toBe(`${BUSINESS_NAME} Password Reset`);
            expect(actualData.text).toContain(link);
            expect(actualData.html).toContain(link);
        });

        it("correctly enqueues email for VerificationLink template", async () => {
            const userId = "user123";
            const link = "https://example.com/verify/user123:code123";

            const template = AUTH_EMAIL_TEMPLATES.VerificationLink(userId, link);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            const actualData = await expectEmailToBeEnqueuedWith({
                to: [testEmail],
            });

            // Check template properties
            expect(actualData.subject).toBe(`Verify ${BUSINESS_NAME} Account`);
            expect(actualData.text).toContain(link);
            if (actualData.html) {
                expect(actualData.html).toContain(link);
            }
        });

        it("correctly enqueues feedback notification email for the admin", async () => {
            const feedbackText = "This is a feedback message.";
            const userId = "user123";

            const template = AUTH_EMAIL_TEMPLATES.FeedbackNotifyAdmin(userId, feedbackText);
            const result = await queueService.email.addTask({
                to: [process.env.SITE_EMAIL_USERNAME!],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [process.env.SITE_EMAIL_USERNAME],
                subject: "Received Vrooli Feedback!",
                text: `Feedback received: ${feedbackText}`,
            });
        });

        it("correctly handles payment thank you email", async () => {
            const userId = "user123";
            const isDonation = true;

            const template = AUTH_EMAIL_TEMPLATES.PaymentThankYou(userId, isDonation);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Thank you for your donation!",
                text: expect.stringContaining("Thank you for your donation"),
            });
        });

        it("correctly enqueues email for purchase thank you", async () => {
            const userId = "user123";
            const isDonation = false;

            const template = AUTH_EMAIL_TEMPLATES.PaymentThankYou(userId, isDonation);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Thank you for your purchase!",
                text: expect.stringContaining("premium subscription"),
            });
        });

        it("correctly enqueues email for PaymentFailed template", async () => {
            const userId = "user123";
            const isDonation = true;

            const template = AUTH_EMAIL_TEMPLATES.PaymentFailed(userId, isDonation);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Your donation failed",
                text: expect.stringContaining(`donation to ${BUSINESS_NAME} failed`),
            });
        });

        it("correctly enqueues email for CreditCardExpiringSoon template", async () => {
            const userId = "user123";

            const template = AUTH_EMAIL_TEMPLATES.CreditCardExpiringSoon(userId);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Your credit card is expiring soon!",
                text: expect.stringContaining("credit card is expiring soon"),
            });
        });

        it("correctly enqueues email for SubscriptionCanceled template", async () => {
            const userId = "user123";

            const template = AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(userId);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Sorry to see you go!",
                text: expect.stringContaining("canceled your subscription"),
            });
        });

        it("correctly enqueues email for SubscriptionEnded template", async () => {
            const userId = "user123";

            const template = AUTH_EMAIL_TEMPLATES.SubscriptionEnded(userId);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Your subscription has ended",
                text: expect.stringContaining("subscription has ended"),
            });
        });

        it("correctly enqueues email for TrialEndingSoon template", async () => {
            const userId = "user123";

            const template = AUTH_EMAIL_TEMPLATES.TrialEndingSoon(userId);
            const result = await queueService.email.addTask({
                to: [testEmail],
                ...template,
            });

            expect(result.success).toBe(true);

            await expectEmailToBeEnqueuedWith({
                to: [testEmail],
                subject: "Your trial is ending soon!",
                text: expect.stringContaining("free trial is ending soon"),
            });
        });
    });

    describe("BullMQ-specific features", () => {
        it("should handle job priorities", async () => {
            // Add multiple jobs with different priorities
            const highPriorityResult = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["high@example.com"],
                    subject: "High Priority",
                    text: "This is high priority",
                }), 
                { priority: 10 },
            );

            const lowPriorityResult = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["low@example.com"],
                    subject: "Low Priority",
                    text: "This is low priority",
                }), 
                { priority: 100 },
            );

            expect(highPriorityResult.success).toBe(true);
            expect(lowPriorityResult.success).toBe(true);

            // Verify priorities
            const highPriorityJob = await queueService.email.queue.getJob(highPriorityResult.data!.id);
            const lowPriorityJob = await queueService.email.queue.getJob(lowPriorityResult.data!.id);

            expect(highPriorityJob?.opts.priority).toBe(10);
            expect(lowPriorityJob?.opts.priority).toBe(100);
        });

        it("should handle job deduplication with IDs", async () => {
            const duplicateId = "unique-email-123";

            // Add first job
            const result1 = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["test@example.com"],
                    subject: "First Version",
                    text: "This is the first version",
                }, { id: duplicateId }),
            );

            // Try to add duplicate
            const result2 = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["test@example.com"],
                    subject: "Second Version",
                    text: "This is the second version",
                }, { id: duplicateId }),
            );

            expect(result1.success).toBe(true);
            expect(result2.success).toBe(true);

            // Should only have one job with this ID
            const job = await queueService.email.queue.getJob(duplicateId);
            expect(job).toBeDefined();
            // The second add should have replaced the first
            expect(job?.data.subject).toBe("Second Version");
        });

        it("should handle concurrent job additions", async () => {
            const promises = [];
            for (let i = 0; i < 10; i++) {
                promises.push(
                    queueService.email.addTask(
                        createValidTaskData(QueueTaskType.EMAIL_SEND, {
                            to: [`user${i}@example.com`],
                            subject: `Email ${i}`,
                            text: `This is email number ${i}`,
                        }),
                    ),
                );
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(10);
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            });

            // Verify all jobs were added
            const jobCounts = await queueService.email.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active + jobCounts.completed).toBeGreaterThanOrEqual(10);
        });

        it("should retrieve job statuses", async () => {
            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["status@example.com"],
                    subject: "Status Test",
                    text: "Testing job status",
                }),
            );

            expect(result.success).toBe(true);
            const jobId = result.data!.id;

            // Get job status
            const statuses = await queueService.getTaskStatuses([jobId], "email");
            expect(statuses).toHaveLength(1);
            expect(statuses[0].id).toBe(jobId);
            expect(statuses[0].status).toBeDefined();
            expect(statuses[0].queueName).toBe("email");
        });

        it("should handle job removal", async () => {
            const result = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["remove@example.com"],
                    subject: "To Be Removed",
                    text: "This job will be removed",
                }),
            );

            expect(result.success).toBe(true);
            const jobId = result.data!.id;

            // Verify job exists
            let job = await queueService.email.queue.getJob(jobId);
            expect(job).toBeDefined();

            // Remove the job
            await job!.remove();

            // Verify job is removed
            job = await queueService.email.queue.getJob(jobId);
            expect(job).toBeNull();
        });
    });

    describe("Email processing integration", () => {
        it("should handle email template with complex data", async () => {
            const userId = "complex-user-123";
            const link = "https://example.com/verify/complex-user-123:very-long-verification-code-here";

            // Test with all template types to ensure they work with real queue
            const templates = [
                AUTH_EMAIL_TEMPLATES.ResetPassword(userId, link),
                AUTH_EMAIL_TEMPLATES.VerificationLink(userId, link),
                AUTH_EMAIL_TEMPLATES.PaymentThankYou(userId, true),
                AUTH_EMAIL_TEMPLATES.PaymentThankYou(userId, false),
                AUTH_EMAIL_TEMPLATES.PaymentFailed(userId, true),
                AUTH_EMAIL_TEMPLATES.PaymentFailed(userId, false),
                AUTH_EMAIL_TEMPLATES.CreditCardExpiringSoon(userId),
                AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(userId),
                AUTH_EMAIL_TEMPLATES.SubscriptionEnded(userId),
                AUTH_EMAIL_TEMPLATES.TrialEndingSoon(userId),
            ];

            for (const template of templates) {
                const result = await queueService.email.addTask({
                    to: ["template-test@example.com"],
                    ...template,
                });
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            }
        });

        it("should maintain job data integrity", async () => {
            const complexData = createValidTaskData(QueueTaskType.EMAIL_SEND, {
                to: ["data-integrity@example.com", "second@example.com"],
                subject: "Complex Subject with 特殊字符 and émojis 🎉",
                text: "This is a test with\nmultiple lines\nand special chars: <>&\"",
                html: "<p>HTML with <strong>tags</strong> and &entities;</p>",
                customField: "This should be preserved",
            });

            const result = await queueService.email.addTask(complexData);
            expect(result.success).toBe(true);

            const job = await queueService.email.queue.getJob(result.data!.id);
            expect(job?.data).toMatchObject(complexData);
        });

        it("should handle queue shutdown and restart", async () => {
            // Add a job before shutdown
            const beforeResult = await queueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["before-shutdown@example.com"],
                    subject: "Before Shutdown",
                    text: "Added before shutdown",
                }),
            );
            expect(beforeResult.success).toBe(true);
            const beforeJobId = beforeResult.data!.id;

            // Shutdown the queue service
            await queueService.shutdown();

            // Clear instance to force new one
            (QueueService as any).instance = null;

            // Get new instance and reinitialize
            const newQueueService = QueueService.get();
            await newQueueService.init(process.env.REDIS_URL || "redis://localhost:6379");

            // Verify the job still exists after restart
            const job = await newQueueService.email.queue.getJob(beforeJobId);
            expect(job).toBeDefined();
            expect(job?.data.subject).toBe("Before Shutdown");

            // Add a new job after restart
            const afterResult = await newQueueService.email.addTask(
                createValidTaskData(QueueTaskType.EMAIL_SEND, {
                    to: ["after-restart@example.com"],
                    subject: "After Restart",
                    text: "Added after restart",
                }),
            );
            expect(afterResult.success).toBe(true);

            // Clean up
            await newQueueService.shutdown();
        });
    });
});
