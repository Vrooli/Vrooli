import { API_CREDITS_FREE, BUSINESS_NAME, MINUTES_15_S, MINUTES_2_S, generatePK } from "@vrooli/shared";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { QueueService } from "../tasks/queues.js";
import { QueueTaskType } from "../tasks/taskTypes.js";
import { randomString, validateCode } from "./codes.js";

// AI_CHECK: TYPE_SAFETY=server-auth-credits-plan-migration | LAST: 2025-06-29

export const PHONE_VERIFICATION_TIMEOUT_MS = MINUTES_15_S;
const PHONE_VERIFICATION_RATE_LIMIT_MS = MINUTES_2_S;
const DEFAULT_PHONE_VERIFICATION_CODE_LENGTH = 8;

/**
 * Generates code for phone number verification
 * @param length Length of code to generate
 */
export function generatePhoneVerificationCode(length = DEFAULT_PHONE_VERIFICATION_CODE_LENGTH): string {
    return randomString(length, "0123456789");
}

const phoneSelect = {
    id: true,
    lastVerificationCodeRequestAttempt: true,
    verifiedAt: true,
    user: {
        select: {
            id: true,
        },
    },
} as const;

/**
* Updates phone object with new verification code, and sends message to user with link.
* When the user enters the code in our UI, a mutation is sent to the server to verify the code.
* @returns status of the operation
*/
export async function setupPhoneVerificationCode(
    phoneNumber: string,
    userId: string,
): Promise<void> {
    // Find the phone
    let phone = await DbProvider.get().phone.findUnique({
        where: { phoneNumber },
        select: phoneSelect,
    });
    // Create if not found
    if (!phone) {
        phone = await DbProvider.get().phone.create({
            data: {
                id: generatePK(),
                phoneNumber,
                user: {
                    connect: { id: BigInt(userId) },
                },
            },
            select: phoneSelect,
        });
    }
    // Check if it belongs to the user
    if (phone.user && phone.user.id.toString() !== userId) {
        throw new CustomError("0061", "PhoneNotYours");
    }
    // Check if it's already verified
    if (phone.verifiedAt) {
        throw new CustomError("0059", "PhoneAlreadyVerified");
    }
    // Check if code was sent recently
    if (phone.lastVerificationCodeRequestAttempt && new Date().getTime() - new Date(phone.lastVerificationCodeRequestAttempt).getTime() < PHONE_VERIFICATION_RATE_LIMIT_MS) {
        throw new CustomError("0058", "PhoneCodeSentRecently");
    }
    // Generate new code
    const verificationCode = generatePhoneVerificationCode();
    // Store code and request time
    await DbProvider.get().phone.update({
        where: { id: phone.id },
        data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
    });
    // Send new verification text
    QueueService.get().sms.addTask({
        type: QueueTaskType.SMS_MESSAGE,
        id: `${phoneNumber}-${verificationCode}`, // Overrides any existing verification code to the same phone number
        to: [phoneNumber],
        body: `${verificationCode} is your ${BUSINESS_NAME} verification code`,
    });
}

// TODO 2: Since credits are issued once per user AND once per phone number, we need to make sure that 1) the phone number is stored in a standard format 2) we disconnect phone numbers when removing them from accounts (and set verification stuff to false/null, EXCEPT for verifiedAt), rather than deleting them
/**
 * Validates phone number verification code and update user's account status
 * @param phoneNumber Phone number string
 * @param userId ID of user who owns phone number
 * @param code Verification code
 * @returns True if phone was is verified
 */
export async function validatePhoneVerificationCode(
    phoneNumber: string,
    userId: string,
    code: string,
): Promise<boolean> {
    // Find data
    const phone = await DbProvider.get().phone.findUnique({
        where: { phoneNumber },
        select: {
            id: true,
            userId: true,
            verificationCode: true,
            verifiedAt: true,
            lastVerificationCodeRequestAttempt: true,
        },
    });
    // Note if phone has been verified before, so we can determine if the user is eligible for free credits
    const hasPhoneBeenVerifiedBefore = phone?.verifiedAt !== null;
    if (!phone)
        throw new CustomError("0348", "PhoneNotFound");
    // Check that userId matches phone's userId
    if (phone.userId?.toString() !== userId)
        throw new CustomError("0351", "PhoneNotYours");
    // If phone already verified, remove old verification code
    if (phone.verifiedAt) {
        await DbProvider.get().phone.update({
            where: { id: phone.id },
            data: { verificationCode: null, lastVerificationCodeRequestAttempt: null },
        });
        return true;
    }
    // If code is incorrect or expired
    if (!validateCode(code, phone.verificationCode, phone.lastVerificationCodeRequestAttempt, PHONE_VERIFICATION_TIMEOUT_MS)) {
        // NOTE: With emails, we might automatically try again. 
        // We won't do this for phones, as texts are expensive.
        return false;
    }
    // If we're here, the code is valid
    await DbProvider.get().phone.update({
        where: { id: phone.id },
        data: {
            verifiedAt: new Date().toISOString(),
            verificationCode: null,
            lastVerificationCodeRequestAttempt: null,
        },
    });
    // If this is the first time verifying (both for the user and the phone), give free credits
    const userData = await DbProvider.get().user.findUnique({
        where: { id: BigInt(userId) },
        select: {
            plan: {
                select: {
                    id: true,
                    receivedFreeTrialAt: true,
                },
            },
            creditAccount: {
                select: {
                    id: true,
                    currentBalance: true,
                },
            },
        },
    });
    if (userData) {
        const hasReceivedFreeTrial = userData.plan?.receivedFreeTrialAt !== null;
        if (!hasReceivedFreeTrial && !hasPhoneBeenVerifiedBefore) {
            // Update plan to mark free trial received
            await DbProvider.get().user.update({
                where: { id: BigInt(userId) },
                data: {
                    plan: {
                        upsert: {
                            create: {
                                id: generatePK(),
                                receivedFreeTrialAt: new Date(),
                            },
                            update: {
                                receivedFreeTrialAt: new Date(),
                            },
                        },
                    },
                },
            });
            // Add credits to credit account (separate operation)
            if (userData.creditAccount) {
                await DbProvider.get().credit_account.update({
                    where: { id: userData.creditAccount.id },
                    data: {
                        currentBalance: userData.creditAccount.currentBalance + API_CREDITS_FREE,
                    },
                });
            } else {
                await DbProvider.get().credit_account.create({
                    data: {
                        id: generatePK(),
                        currentBalance: API_CREDITS_FREE,
                        userId: BigInt(userId),
                    },
                });
            }
        }
    }
    return true;
}
