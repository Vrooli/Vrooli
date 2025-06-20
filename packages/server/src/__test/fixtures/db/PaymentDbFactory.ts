import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type payment } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { 
    DbTestFixtures, 
} from "./types.js";

/**
 * Enhanced database fixture factory for Payment model
 * Provides comprehensive testing capabilities for payment records
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different payment types (Stripe, Direct)
 * - Payment status variations (pending, completed, failed, refunded)
 * - Currency support
 * - Premium/subscription handling
 * - Predefined test scenarios
 */
export class PaymentDbFactory extends EnhancedDbFactory<
    Prisma.paymentCreateInput,
    Prisma.paymentUpdateInput
> {

    /**
     * Get complete test fixtures for Payment model
     */
    protected getFixtures(): DbTestFixtures<Prisma.paymentCreateInput, Prisma.paymentUpdateInput> {
        const userId = generatePK();
        
        return {
            minimal: {
                id: generatePK(),
                amount: 999, // $9.99 in cents
                currency: "USD",
                paymentType: "Credits",
                status: "Pending",
                user: {
                    connect: { id: userId }
                },
            },
            complete: {
                id: generatePK(),
                amount: 1999, // $19.99 in cents
                currency: "USD",
                paymentType: "PremiumMonthly",
                status: "Paid",
                stripePaymentIntentId: `pi_${nanoid(24)}`,
                stripeClientSecret: `pi_${nanoid(24)}_secret_${nanoid(16)}`,
                description: "Premium subscription payment",
                processingFee: 58, // Stripe fee: 2.9% + 30 cents
                receivedAt: new Date(),
                refundedAt: null,
                user: {
                    connect: { id: userId }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, amount, currency, paymentType, status, and user
                    description: "Invalid payment missing required fields",
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    amount: "invalid" as any, // Should be number
                    currency: null as any, // Should be string
                    status: 123 as any, // Should be string
                },
                negativeAmount: {
                    id: generatePK(),
                    amount: -500, // Negative amount
                    currency: "USD",
                    paymentType: "Credits",
                    status: "Pending",
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            edgeCases: {
                creditsPurchase: {
                    id: generatePK(),
                    amount: 5000, // $50.00
                    currency: "USD",
                    paymentType: "Credits",
                    status: "Paid",
                    description: "Credits purchase",
                    processingFee: 0, // No fee for credits
                    receivedAt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                failedPayment: {
                    id: generatePK(),
                    amount: 2999, // $29.99
                    currency: "USD",
                    paymentType: "PremiumYearly",
                    status: "Failed",
                    stripePaymentIntentId: `pi_${nanoid(24)}`,
                    description: "Failed premium subscription",
                    user: {
                        connect: { id: userId }
                    },
                },
                donationPayment: {
                    id: generatePK(),
                    amount: 1499, // $14.99
                    currency: "USD",
                    paymentType: "Donation",
                    status: "Paid",
                    stripePaymentIntentId: `pi_${nanoid(24)}`,
                    description: "Platform donation",
                    processingFee: 43, // Processing fee
                    receivedAt: new Date(Date.now() - 86400000), // 1 day ago
                    user: {
                        connect: { id: userId }
                    },
                },
                foreignCurrency: {
                    id: generatePK(),
                    amount: 1200, // €12.00 in cents
                    currency: "EUR",
                    paymentType: "PremiumMonthly",
                    status: "Paid",
                    stripePaymentIntentId: `pi_${nanoid(24)}`,
                    description: "European premium subscription",
                    processingFee: 65, // Different fee structure for EUR
                    receivedAt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                yearlySubscription: {
                    id: generatePK(),
                    amount: 99999, // $999.99
                    currency: "USD",
                    paymentType: "PremiumYearly",
                    status: "Paid",
                    description: "Annual premium subscription",
                    processingFee: 2900, // Higher processing fee for large amount
                    receivedAt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            updates: {
                minimal: {
                    status: "Paid",
                    receivedAt: new Date(),
                },
                complete: {
                    status: "Failed",
                    description: "Updated payment description",
                },
            },
        };
    }

}

// Export factory creator function
export const createPaymentDbFactory = () => new PaymentDbFactory();

// Export the class for type usage
export { PaymentDbFactory as PaymentDbFactoryClass };