import { describe, it, expect, beforeEach, afterEach, vi, type Mock } from "vitest";
import { Job } from "bullmq";
import nodemailer from "nodemailer";
import { emailProcess, setupTransporter } from "./process.js";
import { EmailTask, QueueTaskType } from "../taskTypes.js";
import { logger } from "../../events/logger.js";

// Mock nodemailer
vi.mock("nodemailer", () => ({
    default: {
        createTransport: vi.fn(),
    },
}));

describe("emailProcess", () => {
    let mockTransporter: any;
    let mockSendMail: Mock;
    let loggerErrorSpy: Mock;
    let loggerWarnSpy: Mock;
    let loggerInfoSpy: Mock;
    
    // Store original env values
    const originalEnv = {
        SITE_EMAIL_USERNAME: process.env.SITE_EMAIL_USERNAME,
        SITE_EMAIL_PASSWORD: process.env.SITE_EMAIL_PASSWORD,
        SITE_EMAIL_FROM: process.env.SITE_EMAIL_FROM,
        SITE_EMAIL_ALIAS: process.env.SITE_EMAIL_ALIAS,
    };

    beforeEach(() => {
        // Reset mocks
        vi.clearAllMocks();
        
        // Setup logger spies
        loggerErrorSpy = vi.spyOn(logger, "error").mockImplementation(() => {});
        loggerWarnSpy = vi.spyOn(logger, "warn").mockImplementation(() => {});
        loggerInfoSpy = vi.spyOn(logger, "info").mockImplementation(() => {});
        
        // Setup default env vars
        process.env.SITE_EMAIL_USERNAME = "test@example.com";
        process.env.SITE_EMAIL_PASSWORD = "test-password";
        process.env.SITE_EMAIL_FROM = "Test Site";
        process.env.SITE_EMAIL_ALIAS = "noreply@example.com";
        
        // Setup mock transporter
        mockSendMail = vi.fn().mockResolvedValue({
            messageId: "test-message-id",
            accepted: ["recipient@example.com"],
            rejected: [],
            pending: [],
        });
        
        mockTransporter = {
            sendMail: mockSendMail,
        };
        
        (nodemailer.createTransport as Mock).mockReturnValue(mockTransporter);
        
        // Reset the transporter to null before each test
        (setupTransporter as any).transporter = null;
        (setupTransporter as any).criticalSetupFailed = false;
    });

    afterEach(() => {
        // Restore original env values
        Object.entries(originalEnv).forEach(([key, value]) => {
            if (value === undefined) {
                delete process.env[key];
            } else {
                process.env[key] = value;
            }
        });
        
        // Restore spies
        loggerErrorSpy.mockRestore();
        loggerWarnSpy.mockRestore();
        loggerInfoSpy.mockRestore();
    });

    const createMockJob = (data: Partial<EmailTask> = {}): Job<EmailTask> => {
        const defaultData: EmailTask = {
            taskType: QueueTaskType.Email,
            to: ["recipient@example.com"],
            subject: "Test Subject",
            text: "Test email body",
            html: "<p>Test email body</p>",
            ...data,
        };
        
        return {
            id: "test-job-id",
            data: defaultData,
            name: "email",
            attemptsMade: 0,
            opts: {},
        } as Job<EmailTask>;
    };

    describe("successful email sending", () => {
        it("should send email successfully with all fields", async () => {
            const job = createMockJob();
            
            const result = await emailProcess(job);
            
            expect(result.success).toBe(true);
            expect(result.info).toBeDefined();
            expect(mockSendMail).toHaveBeenCalledWith({
                from: '"Test Site" <noreply@example.com>',
                to: "recipient@example.com",
                subject: "Test Subject",
                text: "Test email body",
                html: "<p>Test email body</p>",
            });
            expect(loggerInfoSpy).toHaveBeenCalledWith(
                "Email transporter initialized successfully.",
                expect.any(Object)
            );
        });

        it("should send email to multiple recipients", async () => {
            const job = createMockJob({
                to: ["user1@example.com", "user2@example.com", "user3@example.com"],
            });
            
            const result = await emailProcess(job);
            
            expect(result.success).toBe(true);
            expect(mockSendMail).toHaveBeenCalledWith(
                expect.objectContaining({
                    to: "user1@example.com, user2@example.com, user3@example.com",
                })
            );
        });

        it("should send email without HTML content", async () => {
            const job = createMockJob({
                html: undefined,
            });
            
            const result = await emailProcess(job);
            
            expect(result.success).toBe(true);
            expect(mockSendMail).toHaveBeenCalledWith(
                expect.objectContaining({
                    text: "Test email body",
                    html: undefined,
                })
            );
        });

        it("should use SITE_EMAIL_USERNAME when SITE_EMAIL_ALIAS is not set", async () => {
            delete process.env.SITE_EMAIL_ALIAS;
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(true);
            expect(mockSendMail).toHaveBeenCalledWith(
                expect.objectContaining({
                    from: '"Test Site" <test@example.com>',
                })
            );
        });

        it("should reuse existing transporter on subsequent calls", async () => {
            const job = createMockJob();
            
            // First call
            await emailProcess(job);
            expect(nodemailer.createTransport).toHaveBeenCalledTimes(1);
            
            // Second call should reuse transporter
            await emailProcess(job);
            expect(nodemailer.createTransport).toHaveBeenCalledTimes(1);
            expect(mockSendMail).toHaveBeenCalledTimes(2);
        });
    });

    describe("email sending failures", () => {
        it("should handle rejected recipients", async () => {
            mockSendMail.mockResolvedValueOnce({
                messageId: "test-message-id",
                accepted: [],
                rejected: ["rejected@example.com"],
                pending: [],
            });
            
            const job = createMockJob({
                to: ["rejected@example.com"],
            });
            
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(result.info.rejected).toContain("rejected@example.com");
        });

        it("should handle nodemailer sendMail errors", async () => {
            const sendError = new Error("SMTP connection failed");
            mockSendMail.mockRejectedValueOnce(sendError);
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                "Failed to process email job",
                expect.objectContaining({
                    jobId: "test-job-id",
                    errorDetails: expect.objectContaining({
                        message: "SMTP connection failed",
                    }),
                })
            );
        });

        it("should handle network errors with error codes", async () => {
            const networkError = new Error("Connection timeout") as any;
            networkError.code = "ETIMEDOUT";
            mockSendMail.mockRejectedValueOnce(networkError);
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                "Failed to process email job",
                expect.objectContaining({
                    errorDetails: expect.objectContaining({
                        message: "Connection timeout",
                        code: "ETIMEDOUT",
                    }),
                })
            );
        });
    });

    describe("configuration errors", () => {
        it("should fail when SITE_EMAIL_USERNAME is missing", async () => {
            delete process.env.SITE_EMAIL_USERNAME;
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("SITE_EMAIL_USERNAME or SITE_EMAIL_PASSWORD is not defined"),
                expect.any(Object)
            );
            expect(mockSendMail).not.toHaveBeenCalled();
        });

        it("should fail when SITE_EMAIL_PASSWORD is missing", async () => {
            delete process.env.SITE_EMAIL_PASSWORD;
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("SITE_EMAIL_USERNAME or SITE_EMAIL_PASSWORD is not defined"),
                expect.any(Object)
            );
        });

        it("should fail when SITE_EMAIL_FROM is missing", async () => {
            delete process.env.SITE_EMAIL_FROM;
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("SITE_EMAIL_FROM is not defined"),
                expect.any(Object)
            );
        });

        it("should fail when both SITE_EMAIL_ALIAS and SITE_EMAIL_USERNAME are missing for from address", async () => {
            delete process.env.SITE_EMAIL_ALIAS;
            delete process.env.SITE_EMAIL_USERNAME;
            process.env.SITE_EMAIL_PASSWORD = "test-password"; // Keep password to pass first check
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("SITE_EMAIL_USERNAME or SITE_EMAIL_PASSWORD is not defined"),
                expect.any(Object)
            );
        });
    });

    describe("transporter initialization", () => {
        it("should handle transporter creation errors", async () => {
            const createError = new Error("Invalid auth credentials");
            (nodemailer.createTransport as Mock).mockImplementationOnce(() => {
                throw createError;
            });
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("Email transporter setup failed during nodemailer.createTransport"),
                expect.objectContaining({
                    error: "Invalid auth credentials",
                })
            );
        });

        it("should recover when credentials become available after initial failure", async () => {
            // First attempt without credentials
            delete process.env.SITE_EMAIL_PASSWORD;
            let job = createMockJob();
            let result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                expect.stringContaining("SITE_EMAIL_USERNAME or SITE_EMAIL_PASSWORD is not defined"),
                expect.any(Object)
            );
            
            // Second attempt without credentials (should log warning)
            loggerErrorSpy.mockClear();
            result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerWarnSpy).toHaveBeenCalledWith(
                expect.stringContaining("Email transporter setup skipped: critical credential issue persists"),
                expect.any(Object)
            );
            
            // Restore credentials
            process.env.SITE_EMAIL_PASSWORD = "test-password";
            loggerInfoSpy.mockClear();
            
            // Third attempt with restored credentials
            job = createMockJob();
            result = await emailProcess(job);
            
            expect(result.success).toBe(true);
            expect(loggerInfoSpy).toHaveBeenCalledWith(
                "Email credentials are now available. Attempting to initialize or reinitialize transporter.",
                expect.any(Object)
            );
        });
    });

    describe("error handling edge cases", () => {
        it("should handle non-Error objects thrown", async () => {
            mockSendMail.mockRejectedValueOnce({ message: "Custom error object", code: "CUSTOM" });
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                "Failed to process email job",
                expect.objectContaining({
                    errorDetails: expect.objectContaining({
                        message: "Custom error object",
                        code: "CUSTOM",
                    }),
                })
            );
        });

        it("should handle string errors", async () => {
            mockSendMail.mockRejectedValueOnce("String error message");
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                "Failed to process email job",
                expect.objectContaining({
                    errorDetails: expect.objectContaining({
                        message: "String error message",
                    }),
                })
            );
        });

        it("should handle null/undefined errors", async () => {
            mockSendMail.mockRejectedValueOnce(null);
            
            const job = createMockJob();
            const result = await emailProcess(job);
            
            expect(result.success).toBe(false);
            expect(loggerErrorSpy).toHaveBeenCalledWith(
                "Failed to process email job",
                expect.objectContaining({
                    errorDetails: expect.objectContaining({
                        message: "An unknown error occurred during email transport.",
                    }),
                })
            );
        });
    });
});