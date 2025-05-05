import { API_CREDITS_FREE, MINUTES_15_S, MINUTES_2_S } from "@local/shared";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { sendSmsVerification } from "../tasks/sms/queue.js";
import { randomString, validateCode } from "./codes.js";

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
                phoneNumber,
                user: {
                    connect: { id: userId },
                },
            },
            select: phoneSelect,
        });
    }
    // Check if it belongs to the user
    if (phone.user && phone.user.id !== userId) {
        throw new CustomError("0061", "PhoneNotYours");
    }
    // Check if it's already verified
    if (phone.verified) {
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
    sendSmsVerification(phoneNumber, verificationCode);
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
    if (phone.userId !== userId)
        throw new CustomError("0351", "PhoneNotYours");
    // If phone already verified, remove old verification code
    if (phone.verified) {
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
        where: { id: userId },
        select: {
            premium: {
                select: {
                    id: true,
                    credits: true,
                    receivedFreeTrialAt: true,
                },
            },
        },
    });
    if (userData) {
        const hasReceivedFreeTrial = userData.premium?.receivedFreeTrialAt !== null;
        if (!hasReceivedFreeTrial && !hasPhoneBeenVerifiedBefore) {
            await DbProvider.get().user.update({
                where: { id: userId },
                data: {
                    premium: {
                        upsert: {
                            create: {
                                receivedFreeTrialAt: new Date(),
                                credits: API_CREDITS_FREE,
                            },
                            update: {
                                receivedFreeTrialAt: new Date(),
                                credits: (userData.premium?.credits ?? BigInt(0)) + API_CREDITS_FREE,
                            },
                        },
                    },
                },
            });
        }
    }
    return true;
}
