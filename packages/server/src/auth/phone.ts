import { API_CREDITS_FREE } from "@local/shared";
import { CustomError } from "../events/error";
import { sendSmsVerification } from "../tasks/sms/queue";
import { PrismaType } from "../types";
import { randomString, validateCode } from "./codes";

export const PHONE_VERIFICATION_TIMEOUT = 15 * 60 * 1000; // 15 minutes
const PHONE_VERIFICATION_RATE_LIMIT = 2 * 60 * 1000; // 2 minutes

/**
 * Generates code for phone number verification
 * @param length Length of code to generate
 */
export const generatePhoneVerificationCode = (length = 8): string => {
    return randomString(length, "0123456789");
};

const phoneSelect = {
    id: true,
    lastVerificationCodeRequestAttempt: true,
    verified: true,
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
export const setupPhoneVerificationCode = async (
    phoneNumber: string,
    userId: string,
    prisma: PrismaType,
    languages: string[],
): Promise<void> => {
    // Find the phone
    let phone = await prisma.phone.findUnique({
        where: { phoneNumber },
        select: phoneSelect,
    });
    // Create if not found
    if (!phone) {
        phone = await prisma.phone.create({
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
        throw new CustomError("0061", "PhoneNotYours", languages);
    }
    // Check if it's already verified
    if (phone.verified) {
        throw new CustomError("0059", "PhoneAlreadyVerified", languages);
    }
    // Check if code was sent recently
    if (phone.lastVerificationCodeRequestAttempt && new Date().getTime() - new Date(phone.lastVerificationCodeRequestAttempt).getTime() < PHONE_VERIFICATION_RATE_LIMIT) {
        throw new CustomError("0058", "PhoneCodeSentRecently", languages);
    }
    // Generate new code
    const verificationCode = generatePhoneVerificationCode();
    // Store code and request time
    await prisma.phone.update({
        where: { id: phone.id },
        data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
    });
    // Send new verification text
    sendSmsVerification(phoneNumber, verificationCode);
};

// TODO 2: Since credits are issued once per user AND once per phone number, we need to make sure that 1) the phone number is stored in a standard format 2) we disconnect phone numbers when removing them from accounts (and set verification stuff to false/null, EXCEPT for lastVerifiedTime), rather than deleting them
/**
 * Validates phone number verification code and update user's account status
 * @param phoneNumber Phone number string
 * @param userId ID of user who owns phone number
 * @param code Verification code
 * @param prisma The Prisma client
 * @param languages Preferred languages to display error messages in
 * @returns True if phone was is verified
 */
export const validatePhoneVerificationCode = async (
    phoneNumber: string,
    userId: string,
    code: string,
    prisma: PrismaType,
    languages: string[],
): Promise<boolean> => {
    // Find data
    const phone = await prisma.phone.findUnique({
        where: { phoneNumber },
        select: {
            id: true,
            userId: true,
            verified: true,
            verificationCode: true,
            lastVerifiedTime: true,
            lastVerificationCodeRequestAttempt: true,
        },
    });
    // Note if phone has been verified before, so we can determine if the user is eligible for free credits
    const hasPhoneBeenVerifiedBefore = phone?.lastVerifiedTime !== null;
    if (!phone)
        throw new CustomError("0348", "PhoneNotFound", languages);
    // Check that userId matches phone's userId
    if (phone.userId !== userId)
        throw new CustomError("0351", "PhoneNotYours", languages);
    // If phone already verified, remove old verification code
    if (phone.verified) {
        await prisma.phone.update({
            where: { id: phone.id },
            data: { verificationCode: null, lastVerificationCodeRequestAttempt: null },
        });
        return true;
    }
    // If code is incorrect or expired
    if (!validateCode(code, phone.verificationCode, phone.lastVerificationCodeRequestAttempt, PHONE_VERIFICATION_TIMEOUT)) {
        // NOTE: With emails, we might automatically try again. 
        // We won't do this for phones, as texts are expensive.
        return false;
    }
    // If we're here, the code is valid
    await prisma.phone.update({
        where: { id: phone.id },
        data: {
            verified: true,
            lastVerifiedTime: new Date().toISOString(),
            verificationCode: null,
            lastVerificationCodeRequestAttempt: null,
        },
    });
    // If this is the first time verifying (both for the user and the phone), give free credits
    const userData = await prisma.user.findUnique({
        where: { id: userId },
        select: {
            premium: {
                select: {
                    id: true,
                    credits: true,
                    hasReceivedFreeTrial: true,
                },
            },
        },
    });
    if (userData) {
        const hasReceivedFreeTrial = userData.premium?.hasReceivedFreeTrial === true;
        if (!hasReceivedFreeTrial && !hasPhoneBeenVerifiedBefore) {
            await prisma.user.update({
                where: { id: userId },
                data: {
                    premium: {
                        upsert: {
                            create: {
                                hasReceivedFreeTrial: true,
                                credits: API_CREDITS_FREE,
                            },
                            update: {
                                hasReceivedFreeTrial: true,
                                credits: (userData.premium?.credits ?? BigInt(0)) + API_CREDITS_FREE,
                            },
                        },
                    },
                },
            });
        }
    }
    return true;
};
