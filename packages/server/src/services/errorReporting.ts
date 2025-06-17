import { HttpStatus, SECONDS_1_MS } from "@vrooli/shared";
import type { Request, Response } from "express";
import { type Express } from "express";
import { RequestService } from "../auth/request.js";
import { logger } from "../events/logger.js";

interface ErrorReport {
    error: string;
    stack?: string;
    userAgent: string;
    url: string;
    timestamp: string;
    userId?: string;
    sessionId?: string;
}

/**
 * Validates and sanitizes error report data
 */
function validateErrorReport(data: any): ErrorReport | null {
    if (!data || typeof data !== "object") {
        return null;
    }

    // Required fields
    if (typeof data.error !== "string" || !data.error.trim()) {
        return null;
    }
    if (typeof data.userAgent !== "string" || !data.userAgent.trim()) {
        return null;
    }
    if (typeof data.url !== "string" || !data.url.trim()) {
        return null;
    }
    if (typeof data.timestamp !== "string" || !data.timestamp.trim()) {
        return null;
    }

    // Sanitize and validate timestamp
    const timestamp = new Date(data.timestamp);
    if (isNaN(timestamp.getTime())) {
        return null;
    }

    // Optional fields
    const stack = typeof data.stack === "string" ? data.stack : undefined;
    const userId = typeof data.userId === "string" ? data.userId : undefined;
    const sessionId = typeof data.sessionId === "string" ? data.sessionId : undefined;

    // Truncate long strings to prevent log spam
    const MAX_ERROR_LENGTH = 2000;
    const MAX_STACK_LENGTH = 5000;
    const MAX_URL_LENGTH = 500;
    const MAX_USER_AGENT_LENGTH = 500;

    return {
        error: data.error.substring(0, MAX_ERROR_LENGTH),
        stack: stack ? stack.substring(0, MAX_STACK_LENGTH) : undefined,
        userAgent: data.userAgent.substring(0, MAX_USER_AGENT_LENGTH),
        url: data.url.substring(0, MAX_URL_LENGTH),
        timestamp: timestamp.toISOString(),
        userId,
        sessionId,
    };
}

/**
 * Setup the error reporting endpoint
 */
export function setupErrorReporting(app: Express): void {
    app.post("/api/error-reports", async (req: Request, res: Response) => {
        const ERROR_PROCESSING_TIMEOUT_MS = 5000; // 5 seconds

        const timeoutPromise = new Promise((_, reject) => {
            setTimeout(() => {
                reject(new Error("Error report processing timed out"));
            }, ERROR_PROCESSING_TIMEOUT_MS);
        });

        try {
            const processPromise = (async () => {
                // Apply rate limiting - 10 error reports per user per day
                await RequestService.get().rateLimit({ maxUser: 10, req });

                // Validate and sanitize the error report
                const errorReport = validateErrorReport(req.body);
                if (!errorReport) {
                    return res.status(HttpStatus.BadRequest).json({
                        success: false,
                        error: "Invalid error report data",
                        timestamp: Date.now(),
                    });
                }

                // Add user context if available from session
                if (req.session?.userId && !errorReport.userId) {
                    errorReport.userId = req.session.userId;
                }
                if (req.sessionID && !errorReport.sessionId) {
                    errorReport.sessionId = req.sessionID;
                }

                // Log the error report using winston
                logger.error("Client error report received", {
                    trace: "client-error-report",
                    clientError: {
                        error: errorReport.error,
                        stack: errorReport.stack,
                        userAgent: errorReport.userAgent,
                        url: errorReport.url,
                        timestamp: errorReport.timestamp,
                        userId: errorReport.userId,
                        sessionId: errorReport.sessionId,
                    },
                    ip: req.ip,
                    forwardedFor: req.headers["x-forwarded-for"],
                });

                return res.status(HttpStatus.Ok).json({
                    success: true,
                    message: "Error report received",
                    timestamp: Date.now(),
                });
            })();

            await Promise.race([processPromise, timeoutPromise]);

        } catch (error) {
            let errorMessage = "Failed to process error report";
            let statusCode = HttpStatus.InternalServerError;

            if (error instanceof Error && error.message === "Error report processing timed out") {
                errorMessage = "Error report processing timed out after " + (ERROR_PROCESSING_TIMEOUT_MS / SECONDS_1_MS) + " seconds";
                statusCode = HttpStatus.RequestTimeout;
            } else if (error instanceof Error) {
                errorMessage = `Failed to process error report: ${error.message}`;
                // Check if it's a rate limit error
                if (error.message.includes("RateLimitExceeded") || error.message.includes("rate limit")) {
                    statusCode = HttpStatus.TooManyRequests;
                    errorMessage = "Too many error reports. Please try again later.";
                }
            }

            // Log the error processing failure (but don't spam logs if it's just rate limiting)
            if (statusCode !== HttpStatus.TooManyRequests) {
                logger.error(`🚨 ${errorMessage}`, { 
                    trace: "error-reporting-failure", 
                    error,
                    requestBody: req.body,
                });
            }

            res.status(statusCode).json({
                success: false,
                error: errorMessage,
                timestamp: Date.now(),
            });
        }
    });
}