/**
 * Premium API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for premium subscription endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 * Premium objects represent subscription plans, premium features, and tier management.
 */

import { http, type RestHandler } from "msw";
import {
    PaymentType} from "@vrooli/shared";
import type { 
    Premium,
    CheckSubscriptionResponse,
    CheckCreditsPaymentResponse,
    SubscriptionPricesResponse,
    CreateCheckoutSessionResponse,
    CreatePortalSessionResponse,
    CreateCheckoutSessionParams,
    CreatePortalSessionParams,
    User,
} from "@vrooli/shared";

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
 * Premium subscription statuses
 */
export enum PremiumStatus {
    ACTIVE = "active",
    EXPIRED = "expired",
    CANCELLED = "cancelled",
    PENDING = "pending",
    TRIALING = "trialing",
    PAST_DUE = "past_due",
    UNPAID = "unpaid"
}

/**
 * Premium feature access levels
 */
export enum PremiumTier {
    FREE = "free",
    PREMIUM_MONTHLY = "premium_monthly",
    PREMIUM_YEARLY = "premium_yearly",
    CUSTOM = "custom"
}

/**
 * Extended Premium type with additional fields for testing
 */
export interface ExtendedPremium extends Premium {
    status: PremiumStatus;
    tier: PremiumTier;
    billingCycle?: "monthly" | "yearly";
    stripeCustomerId?: string;
    stripeSubscriptionId?: string;
    lastPaymentAt?: string;
    nextBillingAt?: string;
    cancelledAt?: string;
    features: {
        maxProjects: number;
        maxTeamMembers: number;
        maxApiCalls: number;
        advancedAnalytics: boolean;
        prioritySupport: boolean;
        customBranding: boolean;
    };
}

/**
 * Premium subscription input for upgrades/downgrades
 */
export interface PremiumSubscriptionInput {
    paymentType: PaymentType.PremiumMonthly | PaymentType.PremiumYearly;
    customPlan?: string;
    promoCode?: string;
}

/**
 * Premium API response factory
 */
export class PremiumResponseFactory {
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
        return `premium_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful premium response
     */
    createSuccessResponse(premium: ExtendedPremium): APIResponse<ExtendedPremium> {
        return {
            data: premium,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/premium/${premium.id}`,
                    related: {
                        pricing: `${this.baseUrl}/api/subscription-prices`,
                        portal: `${this.baseUrl}/api/create-portal-session`,
                        checkout: `${this.baseUrl}/api/create-checkout-session`,
                    },
                },
            },
        };
    }
    
    /**
     * Create subscription status response
     */
    createSubscriptionStatusResponse(response: CheckSubscriptionResponse): APIResponse<CheckSubscriptionResponse> {
        return {
            data: response,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/check-subscription`,
                },
            },
        };
    }
    
    /**
     * Create subscription prices response
     */
    createPricingResponse(prices: SubscriptionPricesResponse): APIResponse<SubscriptionPricesResponse> {
        return {
            data: prices,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/subscription-prices`,
                },
            },
        };
    }
    
    /**
     * Create checkout session response
     */
    createCheckoutResponse(response: CreateCheckoutSessionResponse): APIResponse<CreateCheckoutSessionResponse> {
        return {
            data: response,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
            },
        };
    }
    
    /**
     * Create portal session response
     */
    createPortalResponse(response: CreatePortalSessionResponse): APIResponse<CreatePortalSessionResponse> {
        return {
            data: response,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
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
                message: "The request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
            },
        };
    }
    
    /**
     * Create subscription not found error response
     */
    createSubscriptionNotFoundErrorResponse(userId: string): APIErrorResponse {
        return {
            error: {
                code: "SUBSCRIPTION_NOT_FOUND",
                message: `No active subscription found for user '${userId}'`,
                details: {
                    userId,
                    availablePlans: ["premium_monthly", "premium_yearly"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/premium/${userId}`,
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
                    supportedMethods: ["card", "bank_transfer"],
                    retryAfter: 3600,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
            },
        };
    }
    
    /**
     * Create subscription limit error response
     */
    createSubscriptionLimitErrorResponse(feature: string, limit: number): APIErrorResponse {
        return {
            error: {
                code: "SUBSCRIPTION_LIMIT_EXCEEDED",
                message: `Your current plan has reached the limit for ${feature}`,
                details: {
                    feature,
                    currentLimit: limit,
                    upgradeRequired: true,
                    availableUpgrades: ["premium_monthly", "premium_yearly"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
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
                message: `You do not have permission to ${operation} premium subscriptions`,
                details: {
                    operation,
                    requiredPermissions: ["premium:manage"],
                    userPermissions: ["user:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
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
                message: "Payment service temporarily unavailable",
                details: {
                    reason: "Stripe service timeout",
                    retryable: true,
                    retryAfter: 30000,
                    estimatedResolution: "5 minutes",
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
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
                message: "An unexpected error occurred while processing subscription",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                    contactSupport: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/premium",
            },
        };
    }
    
    /**
     * Create mock premium subscription data
     */
    createMockPremium(overrides?: Partial<ExtendedPremium>): ExtendedPremium {
        const now = new Date();
        const expiresAt = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000); // 30 days from now
        const id = this.generateId();
        
        const defaultPremium: ExtendedPremium = {
            __typename: "Premium",
            id,
            credits: 10000,
            customPlan: null,
            enabledAt: now.toISOString(),
            expiresAt: expiresAt.toISOString(),
            status: PremiumStatus.ACTIVE,
            tier: PremiumTier.PREMIUM_MONTHLY,
            billingCycle: "monthly",
            stripeCustomerId: `cus_${Math.random().toString(36).substr(2, 14)}`,
            stripeSubscriptionId: `sub_${Math.random().toString(36).substr(2, 14)}`,
            lastPaymentAt: now.toISOString(),
            nextBillingAt: new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000).toISOString(),
            features: {
                maxProjects: 100,
                maxTeamMembers: 25,
                maxApiCalls: 10000,
                advancedAnalytics: true,
                prioritySupport: true,
                customBranding: true,
            },
        };
        
        return {
            ...defaultPremium,
            ...overrides,
        };
    }
    
    /**
     * Create premium subscription from input
     */
    createPremiumFromInput(input: PremiumSubscriptionInput): ExtendedPremium {
        const premium = this.createMockPremium();
        
        // Update premium based on input
        if (input.paymentType === PaymentType.PremiumMonthly) {
            premium.tier = PremiumTier.PREMIUM_MONTHLY;
            premium.billingCycle = "monthly";
            premium.credits = 10000;
            premium.features.maxProjects = 100;
        } else if (input.paymentType === PaymentType.PremiumYearly) {
            premium.tier = PremiumTier.PREMIUM_YEARLY;
            premium.billingCycle = "yearly";
            premium.credits = 120000;
            premium.features.maxProjects = 500;
        }
        
        if (input.customPlan) {
            premium.customPlan = input.customPlan;
            premium.tier = PremiumTier.CUSTOM;
        }
        
        return premium;
    }
    
    /**
     * Create premium subscriptions for different statuses
     */
    createPremiumForAllStatuses(): ExtendedPremium[] {
        return Object.values(PremiumStatus).map(status => 
            this.createMockPremium({
                status,
                id: `premium_${status}_${this.generateId()}`,
            }),
        );
    }
    
    /**
     * Create subscription prices
     */
    createMockPricing(): SubscriptionPricesResponse {
        return {
            monthly: 9.99,
            yearly: 99.99,
        };
    }
    
    /**
     * Create checkout session URL
     */
    createMockCheckoutUrl(): CreateCheckoutSessionResponse {
        return {
            url: `https://checkout.stripe.com/pay/cs_test_${Math.random().toString(36).substr(2, 32)}`,
        };
    }
    
    /**
     * Create portal session URL
     */
    createMockPortalUrl(): CreatePortalSessionResponse {
        return {
            url: `https://billing.stripe.com/p/session/${Math.random().toString(36).substr(2, 32)}`,
        };
    }
    
    /**
     * Validate premium subscription input
     */
    async validateSubscriptionInput(input: PremiumSubscriptionInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        // Validate payment type
        if (!input.paymentType) {
            errors.paymentType = "Payment type is required";
        } else if (![PaymentType.PremiumMonthly, PaymentType.PremiumYearly].includes(input.paymentType)) {
            errors.paymentType = "Invalid payment type for premium subscription";
        }
        
        // Validate custom plan if provided
        if (input.customPlan && input.customPlan.length > 100) {
            errors.customPlan = "Custom plan description too long";
        }
        
        // Validate promo code if provided
        if (input.promoCode && !/^[A-Z0-9_-]{1,20}$/i.test(input.promoCode)) {
            errors.promoCode = "Invalid promo code format";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }
}

/**
 * MSW handlers factory for premium subscription endpoints
 */
export class PremiumMSWHandlers {
    private responseFactory: PremiumResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new PremiumResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all premium endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get subscription status
            http.get(`${this.responseFactory["baseUrl"]}/api/check-subscription`, (req, res, ctx) => {
                const response: CheckSubscriptionResponse = {
                    paymentType: PaymentType.PremiumMonthly,
                    status: "already_subscribed",
                };
                
                return res(
                    ctx.status(200),
                    ctx.json(this.responseFactory.createSubscriptionStatusResponse(response)),
                );
            }),
            
            // Get subscription prices
            http.get(`${this.responseFactory["baseUrl"]}/api/subscription-prices`, (req, res, ctx) => {
                const prices = this.responseFactory.createMockPricing();
                const response = this.responseFactory.createPricingResponse(prices);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Create checkout session
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, async (req, res, ctx) => {
                const body = await req.json() as CreateCheckoutSessionParams;
                
                // Basic validation
                if (!body.variant || !body.userId) {
                    const validation = this.responseFactory.createValidationErrorResponse({
                        variant: !body.variant ? "Payment type is required" : "",
                        userId: !body.userId ? "User ID is required" : "",
                    });
                    return res(
                        ctx.status(400),
                        ctx.json(validation),
                    );
                }
                
                const checkoutUrl = this.responseFactory.createMockCheckoutUrl();
                const response = this.responseFactory.createCheckoutResponse(checkoutUrl);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Create portal session
            http.post(`${this.responseFactory["baseUrl"]}/api/create-portal-session`, async (req, res, ctx) => {
                const body = await req.json() as CreatePortalSessionParams;
                
                if (!body.userId || !body.returnUrl) {
                    const validation = this.responseFactory.createValidationErrorResponse({
                        userId: !body.userId ? "User ID is required" : "",
                        returnUrl: !body.returnUrl ? "Return URL is required" : "",
                    });
                    return res(
                        ctx.status(400),
                        ctx.json(validation),
                    );
                }
                
                const portalUrl = this.responseFactory.createMockPortalUrl();
                const response = this.responseFactory.createPortalResponse(portalUrl);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get premium details by user ID
            http.get(`${this.responseFactory["baseUrl"]}/api/premium/:userId`, (req, res, ctx) => {
                const { userId } = req.params;
                
                const premium = this.responseFactory.createMockPremium({ 
                    id: `premium_${userId}`,
                    stripeCustomerId: `cus_${userId}`,
                });
                const response = this.responseFactory.createSuccessResponse(premium);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update subscription
            http.put(`${this.responseFactory["baseUrl"]}/api/premium/:userId`, async (req, res, ctx) => {
                const { userId } = req.params;
                const body = await req.json() as PremiumSubscriptionInput;
                
                // Validate input
                const validation = await this.responseFactory.validateSubscriptionInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                const premium = this.responseFactory.createPremiumFromInput(body);
                premium.id = `premium_${userId}`;
                
                const response = this.responseFactory.createSuccessResponse(premium);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Cancel subscription
            http.delete(`${this.responseFactory["baseUrl"]}/api/premium/:userId`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // Check credits payment
            http.get(`${this.responseFactory["baseUrl"]}/api/check-credits-payment`, (req, res, ctx) => {
                const response: CheckCreditsPaymentResponse = {
                    status: "new_credits_received",
                };
                
                return res(
                    ctx.status(200),
                    ctx.json({
                        data: response,
                        meta: {
                            timestamp: new Date().toISOString(),
                            requestId: this.responseFactory["generateRequestId"](),
                            version: "1.0",
                        },
                    }),
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
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        variant: "Payment type is required",
                        userId: "User ID is required",
                    })),
                );
            }),
            
            // Subscription not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/premium/:userId`, (req, res, ctx) => {
                const { userId } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createSubscriptionNotFoundErrorResponse(userId as string)),
                );
            }),
            
            // Payment failed error
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, (req, res, ctx) => {
                return res(
                    ctx.status(402),
                    ctx.json(this.responseFactory.createPaymentFailedErrorResponse("Insufficient funds")),
                );
            }),
            
            // Subscription limit error
            http.post(`${this.responseFactory["baseUrl"]}/api/premium/check-limit`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createSubscriptionLimitErrorResponse("projects", 100)),
                );
            }),
            
            // Permission error
            http.put(`${this.responseFactory["baseUrl"]}/api/premium/:userId`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("manage")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, (req, res, ctx) => {
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
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, async (req, res, ctx) => {
                const body = await req.json() as CreateCheckoutSessionParams;
                const checkoutUrl = this.responseFactory.createMockCheckoutUrl();
                const response = this.responseFactory.createCheckoutResponse(checkoutUrl);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/create-checkout-session`, (req, res, ctx) => {
                return res.networkError("Payment service connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/subscription-prices`, (req, res, ctx) => {
                return res.networkError("Pricing service timeout");
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
export const premiumResponseScenarios = {
    // Success scenarios
    activeSubscription: (premium?: ExtendedPremium) => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            premium || factory.createMockPremium({ status: PremiumStatus.ACTIVE }),
        );
    },
    
    expiredSubscription: (premium?: ExtendedPremium) => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            premium || factory.createMockPremium({ 
                status: PremiumStatus.EXPIRED,
                expiresAt: new Date(Date.now() - 86400000).toISOString(), // Yesterday
            }),
        );
    },
    
    pricingSuccess: (prices?: SubscriptionPricesResponse) => {
        const factory = new PremiumResponseFactory();
        return factory.createPricingResponse(
            prices || factory.createMockPricing(),
        );
    },
    
    checkoutSuccess: (url?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createCheckoutResponse(
            url ? { url } : factory.createMockCheckoutUrl(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new PremiumResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                paymentType: "Payment type is required",
                userId: "User ID is required",
            },
        );
    },
    
    subscriptionNotFoundError: (userId?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createSubscriptionNotFoundErrorResponse(
            userId || "non-existent-user",
        );
    },
    
    paymentFailedError: (reason?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createPaymentFailedErrorResponse(
            reason || "Card declined",
        );
    },
    
    subscriptionLimitError: (feature?: string, limit?: number) => {
        const factory = new PremiumResponseFactory();
        return factory.createSubscriptionLimitErrorResponse(
            feature || "projects",
            limit || 10,
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "manage",
        );
    },
    
    serverError: () => {
        const factory = new PremiumResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new PremiumMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new PremiumMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new PremiumMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new PremiumMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const premiumResponseFactory = new PremiumResponseFactory();
export const premiumMSWHandlers = new PremiumMSWHandlers();
