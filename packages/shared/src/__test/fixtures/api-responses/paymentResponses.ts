/* c8 ignore start */
/**
 * Payment API Response Fixtures
 * 
 * Comprehensive fixtures for payment and billing endpoints including
 * subscription management, credit purchases, and transaction history.
 */

import type {
    Payment,
    PaymentStatus,
    PaymentType,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import { teamResponseFactory } from "./teamResponses.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MIN_AMOUNT = 100; // $1.00 in cents
const MAX_AMOUNT = 100000000; // $1M in cents
const MAX_DESCRIPTION_LENGTH = 500;

// Payment create/update input types (since they may not exist in shared)
interface PaymentCreateInput {
    amount: number;
    currency: string;
    description: string;
    paymentMethod: string;
    paymentType: PaymentType;
    teamId?: string;
}

interface PaymentUpdateInput {
    id: string;
    status?: PaymentStatus;
    description?: string;
}

/**
 * Payment API response factory
 */
export class PaymentResponseFactory extends BaseAPIResponseFactory<
    Payment,
    PaymentCreateInput,
    PaymentUpdateInput
> {
    protected readonly entityName = "payment";

    /**
     * Create mock payment data
     */
    createMockData(options?: MockDataOptions): Payment {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const paymentId = options?.overrides?.id || generatePK().toString();

        const basePayment: Payment = {
            __typename: "Payment",
            id: paymentId,
            createdAt: now,
            updatedAt: now,
            amount: 2999, // $29.99
            cardExpDate: "12/25",
            cardLast4: "4242",
            cardType: "visa",
            checkoutId: `cs_${generatePK().toString()}`,
            currency: "USD",
            description: "Premium Monthly Subscription",
            paymentMethod: "card",
            paymentType: "PremiumMonthly",
            status: "Paid",
            user: userResponseFactory.createMockData(),
            team: null,
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...basePayment,
                amount: 9999, // $99.99
                description: "Premium Annual Subscription with Advanced Features",
                paymentType: "PremiumYearly",
                cardType: "mastercard",
                cardLast4: "5678",
                cardExpDate: "03/26",
                team: teamResponseFactory.createMockData({ scenario: "complete" }),
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                ...options?.overrides,
            };
        }

        return {
            ...basePayment,
            ...options?.overrides,
        };
    }

    /**
     * Create payment from input
     */
    createFromInput(input: PaymentCreateInput): Payment {
        const now = new Date().toISOString();
        const paymentId = generatePK().toString();

        return {
            __typename: "Payment",
            id: paymentId,
            createdAt: now,
            updatedAt: now,
            amount: input.amount,
            cardExpDate: "12/25", // Default for new payments
            cardLast4: "4242", // Default for new payments
            cardType: "visa", // Default for new payments
            checkoutId: `cs_${generatePK().toString()}`,
            currency: input.currency,
            description: input.description,
            paymentMethod: input.paymentMethod,
            paymentType: input.paymentType as PaymentType,
            status: "Pending", // New payments start as pending
            user: userResponseFactory.createMockData(),
            team: input.teamId ? teamResponseFactory.createMockData({ overrides: { id: input.teamId } }) : null,
        };
    }

    /**
     * Update payment from input
     */
    updateFromInput(existing: Payment, input: PaymentUpdateInput): Payment {
        const updates: Partial<Payment> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.status !== undefined) updates.status = input.status;
        if (input.description !== undefined) updates.description = input.description;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: PaymentCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.amount || input.amount < MIN_AMOUNT) {
            errors.amount = "Amount must be at least $1.00";
        } else if (input.amount > MAX_AMOUNT) {
            errors.amount = "Amount exceeds maximum limit of $1,000,000";
        }

        if (!input.currency) {
            errors.currency = "Currency is required";
        } else if (!["USD", "EUR", "GBP", "CAD"].includes(input.currency)) {
            errors.currency = "Unsupported currency";
        }

        if (!input.description || input.description.trim().length === 0) {
            errors.description = "Description is required";
        } else if (input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = "Description must be 500 characters or less";
        }

        if (!input.paymentMethod) {
            errors.paymentMethod = "Payment method is required";
        } else if (!["card", "bank_transfer", "digital_wallet"].includes(input.paymentMethod)) {
            errors.paymentMethod = "Invalid payment method";
        }

        if (!input.paymentType) {
            errors.paymentType = "Payment type is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: PaymentUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.status !== undefined) {
            const validStatuses: PaymentStatus[] = ["Pending", "Paid", "Failed", "Cancelled", "Refunded"];
            if (!validStatuses.includes(input.status)) {
                errors.status = "Invalid payment status";
            }
        }

        if (input.description !== undefined && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = "Description must be 500 characters or less";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create payments for different scenarios
     */
    createPaymentScenarios(): Record<string, Payment> {
        return {
            subscription: this.createMockData({
                overrides: {
                    paymentType: "PremiumMonthly",
                    amount: 999, // $9.99
                    description: "Premium Monthly Subscription",
                    status: "Paid",
                },
            }),

            credits: this.createMockData({
                overrides: {
                    paymentType: "Credits",
                    amount: 2000, // $20.00
                    description: "1000 Credits Purchase",
                    status: "Paid",
                },
            }),

            donation: this.createMockData({
                overrides: {
                    paymentType: "Donation",
                    amount: 500, // $5.00
                    description: "Support Vrooli Development",
                    status: "Paid",
                },
            }),

            failed: this.createMockData({
                overrides: {
                    paymentType: "PremiumMonthly",
                    amount: 999,
                    description: "Premium Monthly Subscription",
                    status: "Failed",
                    cardLast4: null,
                    cardType: null,
                    cardExpDate: null,
                },
            }),

            pending: this.createMockData({
                overrides: {
                    paymentType: "PremiumYearly",
                    amount: 9999, // $99.99
                    description: "Premium Yearly Subscription",
                    status: "Pending",
                },
            }),

            refund: this.createMockData({
                overrides: {
                    paymentType: "Credits",
                    amount: -1000, // Negative amount for refund
                    description: "Refund for Credits Purchase",
                    status: "Refunded",
                },
            }),
        };
    }

    /**
     * Create payment history
     */
    createPaymentHistory(months = 6): Payment[] {
        const now = new Date().toISOString();
        const payments: Payment[] = [];

        // Create monthly subscription payments
        for (let i = 0; i < months; i++) {
            const date = new Date(now).toISOString();
            date.setMonth(date.getMonth() - i);

            payments.push(this.createMockData({
                overrides: {
                    createdAt: date.toISOString(),
                    updatedAt: date.toISOString(),
                    paymentType: "PremiumMonthly",
                    amount: 999, // $9.99
                    description: `Premium Monthly Subscription - ${date.toLocaleDateString("en-US", { month: "long", year: "numeric" })}`,
                    status: "Paid",
                },
            }));
        }

        // Add some credit purchases
        for (let i = 0; i < 3; i++) {
            const date = new Date(now).toISOString();
            date.setMonth(date.getMonth() - (i * 2));
            date.setDate(15); // Mid-month

            payments.push(this.createMockData({
                overrides: {
                    createdAt: date.toISOString(),
                    updatedAt: date.toISOString(),
                    paymentType: "Credits",
                    amount: 2000, // $20.00
                    description: "1000 Credits Purchase",
                    status: "Paid",
                },
            }));
        }

        return payments.sort((a, b) => new Date(b.created_at).toISOString().getTime() - new Date(a.created_at).toISOString().getTime());
    }

    /**
     * Create team payment
     */
    createTeamPayment(): Payment {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                paymentType: "PremiumYearly",
                amount: 19999, // $199.99 for team
                description: "Team Premium Annual Subscription (5 members)",
                team: teamResponseFactory.createMockData({ scenario: "complete" }),
            },
        });
    }

    /**
     * Create large payment
     */
    createLargePayment(): Payment {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                paymentType: "Credits",
                amount: 50000, // $500.00
                description: "Bulk Credits Purchase - 50,000 Credits",
                status: "Paid",
            },
        });
    }

    /**
     * Create payment failed error response
     */
    createPaymentFailedErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("payment_failed", {
            reason,
            retryable: true,
            supportContact: "billing@vrooli.com",
            message: `Payment processing failed: ${reason}`,
        });
    }

    /**
     * Create subscription error response
     */
    createSubscriptionErrorResponse(message: string) {
        return this.createBusinessErrorResponse("subscription_error", {
            message,
            billingPortalUrl: "/billing",
            supportEmail: "billing@vrooli.com",
        });
    }

    /**
     * Create insufficient funds error response
     */
    createInsufficientFundsErrorResponse() {
        return this.createPaymentFailedErrorResponse("Insufficient funds on payment method");
    }

    /**
     * Create card declined error response
     */
    createCardDeclinedErrorResponse() {
        return this.createPaymentFailedErrorResponse("Payment method declined by issuer");
    }

    /**
     * Create duplicate payment error response
     */
    createDuplicatePaymentErrorResponse(originalPaymentId: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "payment",
            originalPaymentId,
            message: "A payment with identical details was already processed",
        });
    }

    /**
     * Create payment receipt response
     */
    createPaymentReceiptResponse(paymentId: string) {
        return {
            data: {
                paymentId,
                receiptNumber: `RCP-${Date.now()}`,
                downloadUrl: `/api/payment/${paymentId}/receipt/download`,
                generatedAt: new Date().toISOString(),
                format: "PDF",
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: `/api/payment/${paymentId}/receipt`,
                    download: `/api/payment/${paymentId}/receipt/download`,
                },
            },
        };
    }
}

/**
 * Pre-configured payment response scenarios
 */
export const paymentResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<PaymentCreateInput>) => {
        const factory = new PaymentResponseFactory();
        const defaultInput: PaymentCreateInput = {
            amount: 999, // $9.99
            currency: "USD",
            description: "Premium Monthly Subscription",
            paymentMethod: "card",
            paymentType: "PremiumMonthly",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (payment?: Payment) => {
        const factory = new PaymentResponseFactory();
        return factory.createSuccessResponse(
            payment || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new PaymentResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Payment, updates?: Partial<PaymentUpdateInput>) => {
        const factory = new PaymentResponseFactory();
        const payment = existing || factory.createMockData({ scenario: "complete" });
        const input: PaymentUpdateInput = {
            id: payment.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(payment, input),
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

    teamPaymentSuccess: () => {
        const factory = new PaymentResponseFactory();
        return factory.createSuccessResponse(
            factory.createTeamPayment(),
        );
    },

    largePaymentSuccess: () => {
        const factory = new PaymentResponseFactory();
        return factory.createSuccessResponse(
            factory.createLargePayment(),
        );
    },

    refundSuccess: (originalPaymentId?: string) => {
        const factory = new PaymentResponseFactory();
        const scenarios = factory.createPaymentScenarios();
        return factory.createSuccessResponse({
            ...scenarios.refund,
            id: `refund_${originalPaymentId || generatePK().toString()}`,
        });
    },

    listSuccess: (payments?: Payment[]) => {
        const factory = new PaymentResponseFactory();
        return factory.createPaginatedResponse(
            payments || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: payments?.length || DEFAULT_COUNT },
        );
    },

    historySuccess: (months?: number) => {
        const factory = new PaymentResponseFactory();
        const history = factory.createPaymentHistory(months);
        return factory.createPaginatedResponse(
            history,
            { page: 1, totalCount: history.length },
        );
    },

    statusFilteredSuccess: (status: PaymentStatus) => {
        const factory = new PaymentResponseFactory();
        const scenarios = factory.createPaymentScenarios();
        const payments = Object.values(scenarios).filter(p => p.status === status);
        return factory.createPaginatedResponse(
            payments,
            { page: 1, totalCount: payments.length },
        );
    },

    receiptSuccess: (paymentId?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createPaymentReceiptResponse(paymentId || generatePK().toString());
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createValidationErrorResponse({
            amount: "Amount must be at least $1.00",
            currency: "Currency is required",
            description: "Description is required",
            paymentType: "Payment type is required",
        });
    },

    notFoundError: (paymentId?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createNotFoundErrorResponse(
            paymentId || "non-existent-payment",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["payment:create"],
        );
    },

    paymentFailedError: (reason = "Card declined") => {
        const factory = new PaymentResponseFactory();
        return factory.createPaymentFailedErrorResponse(reason);
    },

    insufficientFundsError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createInsufficientFundsErrorResponse();
    },

    cardDeclinedError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createCardDeclinedErrorResponse();
    },

    subscriptionExistsError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createSubscriptionErrorResponse("User already has an active subscription");
    },

    duplicatePaymentError: (originalPaymentId?: string) => {
        const factory = new PaymentResponseFactory();
        return factory.createDuplicatePaymentErrorResponse(originalPaymentId || generatePK().toString());
    },

    amountTooLargeError: () => {
        const factory = new PaymentResponseFactory();
        return factory.createValidationErrorResponse({
            amount: "Amount exceeds maximum limit of $1,000,000",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new PaymentResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new PaymentResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new PaymentResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const paymentResponseFactory = new PaymentResponseFactory();
