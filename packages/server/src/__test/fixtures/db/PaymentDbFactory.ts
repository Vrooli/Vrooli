import { type Prisma } from "@prisma/client";
import { generatePK, nanoid } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type {
    DbTestFixtures,
} from "./types.js";

// Constants for test values
const INVALID_TYPE_NUMBER = 123;

/**
 * Enhanced database fixture factory for Payment model
 * Provides comprehensive testing capabilities for payment records
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different payment types (Credits, Premium, Donation)
 * - Payment status variations (Pending, Paid, Failed)
 * - Currency support
 * - Payment method tracking
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
                checkoutId: `checkout_${nanoid()}`,
                paymentMethod: "card",
                description: "Credits purchase",
                user: {
                    connect: { id: userId },
                },
            },
            complete: {
                id: generatePK(),
                amount: 1999, // $19.99 in cents
                currency: "USD",
                paymentType: "PremiumMonthly",
                status: "Paid",
                checkoutId: `checkout_${nanoid()}`,
                paymentMethod: "card",
                description: "Premium subscription payment",
                user: {
                    connect: { id: userId },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, amount, currency, paymentType, status, and user
                    description: "Invalid payment missing required fields",
                },
                invalidTypes: {
                    id: INVALID_TYPE_NUMBER as unknown, // Should be string (bigint in TypeScript)
                    amount: "invalid" as unknown, // Should be number
                    currency: null as unknown, // Should be string
                    status: INVALID_TYPE_NUMBER as unknown, // Should be string
                    checkoutId: INVALID_TYPE_NUMBER as unknown, // Should be string
                    paymentMethod: INVALID_TYPE_NUMBER as unknown, // Should be string
                },
                negativeAmount: {
                    id: generatePK(),
                    amount: -500, // Negative amount
                    currency: "USD",
                    paymentType: "Credits",
                    status: "Pending",
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "Invalid negative amount",
                    user: {
                        connect: { id: userId },
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
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "Credits purchase",
                    user: {
                        connect: { id: userId },
                    },
                },
                failedPayment: {
                    id: generatePK(),
                    amount: 2999, // $29.99
                    currency: "USD",
                    paymentType: "PremiumYearly",
                    status: "Failed",
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "Failed premium subscription",
                    user: {
                        connect: { id: userId },
                    },
                },
                donationPayment: {
                    id: generatePK(),
                    amount: 1499, // $14.99
                    currency: "USD",
                    paymentType: "Donation",
                    status: "Paid",
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "Platform donation",
                    user: {
                        connect: { id: userId },
                    },
                },
                foreignCurrency: {
                    id: generatePK(),
                    amount: 1200, // €12.00 in cents
                    currency: "EUR",
                    paymentType: "PremiumMonthly",
                    status: "Paid",
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "European premium subscription",
                    user: {
                        connect: { id: userId },
                    },
                },
                yearlySubscription: {
                    id: generatePK(),
                    amount: 99999, // $999.99
                    currency: "USD",
                    paymentType: "PremiumYearly",
                    status: "Paid",
                    checkoutId: `checkout_${nanoid()}`,
                    paymentMethod: "card",
                    description: "Annual premium subscription",
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            updates: {
                minimal: {
                    status: "Paid",
                },
                complete: {
                    status: "Failed",
                    description: "Updated payment description",
                    paymentMethod: "bank_transfer",
                },
            },
        };
    }

}

// Export factory creator function
export function createPaymentDbFactory() {
    return new PaymentDbFactory();
}

// Export the class for type usage
export { PaymentDbFactory as PaymentDbFactoryClass };
