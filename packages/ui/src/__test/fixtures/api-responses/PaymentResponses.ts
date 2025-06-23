/**
 * Payment API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for payment endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from "msw";
import { 
    PaymentStatus,
    PaymentType} from "@vrooli/shared";
import type { 
    Payment,
    PaymentSortBy,
    User,
    Team,
} from "@vrooli/shared";

/**
 * Payment create input (mock structure since not available in shared)
 */
export interface PaymentCreateInput {
    amount: number;
    currency: string;
    description: string;
    paymentMethod: string;
    paymentType: PaymentType;
    teamId?: string;
}

/**
 * Payment update input (mock structure since not available in shared)
 */
export interface PaymentUpdateInput {
    id: string;
    status?: PaymentStatus;
    description?: string;
}

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, any>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * Payment API response factory
 */
export class PaymentResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate checkout ID
     */
    private generateCheckoutId(): string {
        return `cs_${Date.now()}_${Math.random().toString(36).substr(2, 16)}`;
    }
    
    /**
     * Create successful payment response
     */
    createSuccessResponse(payment: Payment): APIResponse<Payment> {
        return {
            data: payment,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/payment/${payment.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/${payment.user.id}`,
                        team: payment.team ? `${this.baseUrl}/api/team/${payment.team.id}` : undefined,
                        receipt: `${this.baseUrl}/api/payment/${payment.id}/receipt`,
                    },
                },
            },
        };
    }
    
    /**
     * Create payment list response
     */
    createPaymentListResponse(payments: Payment[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Payment> {
        const paginationData = pagination || {
            page: 1,
            pageSize: payments.length,
            totalCount: payments.length,
        };
        
        return {
            data: payments,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/payment?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: "VALIDATION_ERROR",
                message: "The payment request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(paymentId: string): APIErrorResponse {
        return {
            error: {
                code: "PAYMENT_NOT_FOUND",
                message: `Payment with ID '${paymentId}' was not found`,
                details: {
                    paymentId,
                    searchCriteria: { id: paymentId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/payment/${paymentId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this payment`,
                details: {
                    operation,
                    requiredPermissions: ["payment:write"],
                    userPermissions: ["payment:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment",
            },
        };
    }

    /**
     * Create payment failed error response
     */
    createPaymentFailedErrorResponse(reason: string): APIErrorResponse {
        return {
            error: {
                code: "PAYMENT_FAILED",
                message: `Payment processing failed: ${reason}`,
                details: {
                    reason,
                    retryable: true,
                    supportContact: "support@vrooli.com",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment",
            },
        };
    }

    /**
     * Create subscription error response
     */
    createSubscriptionErrorResponse(message: string): APIErrorResponse {
        return {
            error: {
                code: "SUBSCRIPTION_ERROR",
                message,
                details: {
                    billingPortalUrl: `${this.baseUrl}/billing`,
                    supportEmail: "billing@vrooli.com",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment/subscription",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred during payment processing",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                    contactSupport: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/payment",
            },
        };
    }
    
    /**
     * Create mock user data
     */
    createMockUser(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultUser: User = {
            __typename: "User",
            id: `user_${id}`,
            handle: "testuser",
            name: "Test User",
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: false,
            premiumExpiration: null,
            roles: [],
            wallets: [],
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "UserYou",
                isBlocked: false,
                isBlockedByYou: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isReacted: false,
                reactionSummary: {
                    __typename: "ReactionSummary",
                    emotion: null,
                    count: 0,
                },
            },
        };
        
        return {
            ...defaultUser,
            ...overrides,
        };
    }

    /**
     * Create mock team data
     */
    createMockTeam(overrides?: Partial<Team>): Team {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultTeam: Team = {
            __typename: "Team",
            id: `team_${id}`,
            handle: "test-team",
            name: "Test Team",
            createdAt: now,
            updatedAt: now,
            isOpenToNewMembers: true,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            translations: [],
            translationsCount: 0,
            paymentHistory: [],
            members: [],
            membersCount: 1,
            permissions: "{}",
            you: {
                __typename: "TeamYou",
                canDelete: false,
                canUpdate: false,
                canReport: false,
                isBookmarked: false,
                isReacted: false,
                reactionSummary: {
                    __typename: "ReactionSummary",
                    emotion: null,
                    count: 0,
                },
            },
        };
        
        return {
            ...defaultTeam,
            ...overrides,
        };
    }
    
    /**
     * Create mock payment data
     */
    createMockPayment(overrides?: Partial<Payment>): Payment {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultPayment: Payment = {
            __typename: "Payment",
            id: `payment_${id}`,
            amount: 2999, // $29.99 in cents
            cardExpDate: "12/25",
            cardLast4: "4242",
            cardType: "visa",
            checkoutId: this.generateCheckoutId(),
            createdAt: now,
            updatedAt: now,
            currency: "USD",
            description: "Premium Monthly Subscription",
            paymentMethod: "card",
            paymentType: PaymentType.PremiumMonthly,
            status: PaymentStatus.Paid,
            user: this.createMockUser(),
            team: this.createMockTeam(),
        };
        
        return {
            ...defaultPayment,
            ...overrides,
        };
    }
    
    /**
     * Create payment from API input
     */
    createPaymentFromInput(input: PaymentCreateInput): Payment {
        const payment = this.createMockPayment();
        
        // Update payment based on input
        payment.amount = input.amount;
        payment.currency = input.currency;
        payment.description = input.description;
        payment.paymentMethod = input.paymentMethod;
        payment.paymentType = input.paymentType;
        payment.status = PaymentStatus.Pending; // New payments start as pending
        
        if (input.teamId) {
            payment.team = this.createMockTeam({ id: input.teamId });
        }
        
        return payment;
    }
    
    /**
     * Create payments for different scenarios
     */
    createPaymentScenarios(): {
        subscription: Payment;
        credits: Payment;
        donation: Payment;
        failed: Payment;
        pending: Payment;
        refunded: Payment;
    } {
        const baseTime = new Date().toISOString();
        
        return {
            subscription: this.createMockPayment({
                paymentType: PaymentType.PremiumMonthly,
                amount: 999, // $9.99
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Paid,
            }),
            
            credits: this.createMockPayment({
                paymentType: PaymentType.Credits,
                amount: 2000, // $20.00
                description: "1000 Credits Purchase",
                status: PaymentStatus.Paid,
            }),
            
            donation: this.createMockPayment({
                paymentType: PaymentType.Donation,
                amount: 500, // $5.00
                description: "Support Vrooli Development",
                status: PaymentStatus.Paid,
            }),
            
            failed: this.createMockPayment({
                paymentType: PaymentType.PremiumMonthly,
                amount: 999,
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Failed,
                cardLast4: null,
                cardType: null,
                cardExpDate: null,
            }),
            
            pending: this.createMockPayment({
                paymentType: PaymentType.PremiumYearly,
                amount: 9999, // $99.99
                description: "Premium Yearly Subscription",
                status: PaymentStatus.Pending,
            }),
            
            refunded: this.createMockPayment({
                paymentType: PaymentType.Credits,
                amount: -1000, // Negative amount for refund
                description: "Refund for Credits Purchase",
                status: PaymentStatus.Paid,
            }),
        };
    }
    
    /**
     * Create payment history for different time periods
     */
    createPaymentHistory(): Payment[] {
        const now = new Date();
        const payments: Payment[] = [];
        
        // Create payments for the last 6 months
        for (let i = 0; i < 6; i++) {
            const date = new Date(now);
            date.setMonth(date.getMonth() - i);
            
            payments.push(this.createMockPayment({
                createdAt: date.toISOString(),
                updatedAt: date.toISOString(),
                paymentType: PaymentType.PremiumMonthly,
                amount: 999,
                description: `Premium Monthly Subscription - ${date.toLocaleDateString()}`,
            }));
        }
        
        // Add some credit purchases
        for (let i = 0; i < 3; i++) {
            const date = new Date(now);
            date.setMonth(date.getMonth() - (i * 2));
            date.setDate(15); // Mid-month
            
            payments.push(this.createMockPayment({
                createdAt: date.toISOString(),
                updatedAt: date.toISOString(),
                paymentType: PaymentType.Credits,
                amount: 2000,
                description: `1000 Credits Purchase - ${date.toLocaleDateString()}`,
            }));
        }
        
        return payments.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
    }
    
    /**
     * Mock validation for payment create input
     */
    async validateCreateInput(input: PaymentCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        // Amount validation
        if (!input.amount || input.amount <= 0) {
            errors.amount = "Amount must be greater than 0";
        }
        
        if (input.amount && input.amount > 100000000) { // $1M limit
            errors.amount = "Amount exceeds maximum limit";
        }
        
        // Currency validation
        if (!input.currency) {
            errors.currency = "Currency is required";
        } else if (!["USD", "EUR", "GBP"].includes(input.currency)) {
            errors.currency = "Unsupported currency";
        }
        
        // Description validation
        if (!input.description || input.description.trim().length === 0) {
            errors.description = "Description is required";
        } else if (input.description.length > 500) {
            errors.description = "Description must be 500 characters or less";
        }
        
        // Payment method validation
        if (!input.paymentMethod) {
            errors.paymentMethod = "Payment method is required";
        } else if (!["card", "bank_transfer", "digital_wallet"].includes(input.paymentMethod)) {
            errors.paymentMethod = "Invalid payment method";
        }
        
        // Payment type validation
        if (!input.paymentType) {
            errors.paymentType = "Payment type is required";
        } else if (!Object.values(PaymentType).includes(input.paymentType)) {
            errors.paymentType = "Invalid payment type";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }
}

/**
 * MSW handlers factory for payment endpoints
 */
export class PaymentMSWHandlers {
    private responseFactory: PaymentResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new PaymentResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all payment endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create payment
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, async (req, res, ctx) => {
                const body = await req.json() as PaymentCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create payment
                const payment = this.responseFactory.createPaymentFromInput(body);
                const response = this.responseFactory.createSuccessResponse(payment);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get payment by ID
            rest.get(`${this.responseFactory["baseUrl"]}/api/payment/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const payment = this.responseFactory.createMockPayment({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(payment);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update payment
            rest.put(`${this.responseFactory["baseUrl"]}/api/payment/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as PaymentUpdateInput;
                
                const payment = this.responseFactory.createMockPayment({ 
                    id: id as string,
                    status: body.status,
                    description: body.description,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(payment);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // List payments
            rest.get(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as PaymentStatus;
                const paymentType = url.searchParams.get("paymentType") as PaymentType;
                
                let payments = this.responseFactory.createPaymentHistory();
                
                // Filter by status if specified
                if (status) {
                    payments = payments.filter(p => p.status === status);
                }
                
                // Filter by payment type if specified
                if (paymentType) {
                    payments = payments.filter(p => p.paymentType === paymentType);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedPayments = payments.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createPaymentListResponse(
                    paginatedPayments,
                    {
                        page,
                        pageSize: limit,
                        totalCount: payments.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),

            // Get payment receipt
            rest.get(`${this.responseFactory["baseUrl"]}/api/payment/:id/receipt`, (req, res, ctx) => {
                const { id } = req.params;
                
                const receipt = {
                    paymentId: id,
                    receiptNumber: `RCP-${Date.now()}`,
                    downloadUrl: `${this.responseFactory["baseUrl"]}/api/payment/${id}/receipt/download`,
                    generatedAt: new Date().toISOString(),
                };
                
                return res(
                    ctx.status(200),
                    ctx.json({
                        data: receipt,
                        meta: {
                            timestamp: new Date().toISOString(),
                            requestId: this.responseFactory["generateRequestId"](),
                            version: "1.0",
                        },
                    }),
                );
            }),

            // Process refund
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment/:id/refund`, (req, res, ctx) => {
                const { id } = req.params;
                
                const refundPayment = this.responseFactory.createMockPayment({
                    id: `refund_${id}`,
                    amount: -999, // Negative amount for refund
                    description: "Refund for Premium Monthly Subscription",
                    status: PaymentStatus.Paid,
                });
                
                const response = this.responseFactory.createSuccessResponse(refundPayment);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        amount: "Amount must be greater than 0",
                        currency: "Currency is required",
                        paymentType: "Payment type must be specified",
                    })),
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory["baseUrl"]}/api/payment/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),

            // Payment failed error
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                return res(
                    ctx.status(402),
                    ctx.json(this.responseFactory.createPaymentFailedErrorResponse("Card declined")),
                );
            }),

            // Subscription error
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment/subscription`, (req, res, ctx) => {
                return res(
                    ctx.status(409),
                    ctx.json(this.responseFactory.createSubscriptionErrorResponse("User already has an active subscription")),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 3000): RestHandler[] {
        return [
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, async (req, res, ctx) => {
                const body = await req.json() as PaymentCreateInput;
                const payment = this.responseFactory.createPaymentFromInput(body);
                const response = this.responseFactory.createSuccessResponse(payment);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            rest.post(`${this.responseFactory["baseUrl"]}/api/payment`, (req, res, ctx) => {
                return res.networkError("Payment service unavailable");
            }),
            
            rest.get(`${this.responseFactory["baseUrl"]}/api/payment/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: any;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        return rest[method.toLowerCase() as keyof typeof rest](fullEndpoint, (req, res, ctx) => {
            const responseCtx = [ctx.status(status), ctx.json(response)];
            
            if (delay) {
                responseCtx.unshift(ctx.delay(delay));
            }
            
            return res(...responseCtx);
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const paymentResponseScenarios = {
    // Success scenarios
    createSuccess: (payment?: Payment) => {
        const factory = new PaymentResponseFactory();
        return factory.createSuccessResponse(
            payment || factory.createMockPayment(),
        );
    },
    
    listSuccess: (payments?: Payment[]) => {
        const factory = new PaymentResponseFactory();
        return factory.createPaymentListResponse(
            payments || factory.createPaymentHistory(),
        );
    },

    subscriptionSuccess: () => {
        const factory = new PaymentResponseFactory();
        const scenarios = factory.createPaymentScenarios();
        return factory.createSuccessResponse(scenarios.subscription);
    },

    creditsSuccess: () => {
        const factory = new PaymentResponseFactory();
        const scenarios = factory.createPaymentScenarios();
        return factory.createSuccessResponse(scenarios.credits);
    },

    donationSuccess: () => {
        const factory = new PaymentResponseFactory();
        const scenarios = factory.createPaymentScenarios();
        return factory.createSuccessResponse(scenarios.donation);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new PaymentResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                amount: "Amount must be greater than 0",
                currency: "Currency is required",
                paymentType: "Payment type must be specified",
            },
        );
    },
    
    notFoundError: (paymentId?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createNotFoundErrorResponse(
            paymentId || "non-existent-payment-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },

    paymentFailedError: (reason?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createPaymentFailedErrorResponse(
            reason || "Card declined",
        );
    },

    subscriptionError: (message?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createSubscriptionErrorResponse(
            message || "User already has an active subscription",
        );
    },
    
    serverError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new PaymentMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new PaymentMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new PaymentMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new PaymentMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const paymentResponseFactory = new PaymentResponseFactory();
export const paymentMSWHandlers = new PaymentMSWHandlers();
