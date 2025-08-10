/**
 * Maps server error codes to user-friendly messages with helpful guidance
 */
export const errorMessages: Record<string, { message: string; help?: string }> = {
    // Authentication errors
    InvalidCredentials: {
        message: "Invalid email or password",
        help: "Please check your credentials and try again. If you've forgotten your password, use 'vrooli auth reset-password'.",
    },
    MustResetPassword: {
        message: "Password reset required",
        help: "You must reset your password before signing in. Please check your email for the reset link.",
    },
    CannotVerifyEmailCode: {
        message: "Invalid verification code",
        help: "The email verification code is invalid or has expired. Request a new code with 'vrooli auth verify-email'.",
    },
    EmailNotVerified: {
        message: "Email not verified",
        help: "Please verify your email address before signing in. Check your inbox for the verification email.",
    },
    AccountDeleted: {
        message: "Account has been deleted",
        help: "This account no longer exists. If you believe this is an error, please contact support.",
    },
    Unauthorized: {
        message: "Authentication failed",
        help: "Your session may have expired. Please try logging in again.",
    },
    NonceExpired: {
        message: "Wallet authentication expired",
        help: "The authentication request has expired. Please try connecting your wallet again.",
    },
    SessionExpired: {
        message: "Session expired",
        help: "Your session has expired. Please log in again with 'vrooli auth login'.",
    },
    
    // Network/Server errors
    CannotConnectToServer: {
        message: "Cannot connect to server",
        help: "Please check your internet connection and ensure the server is running. You can start the server with './scripts/main/develop.sh'.",
    },
    
    // Rate limiting
    RateLimitExceeded: {
        message: "Too many login attempts",
        help: "You've made too many login attempts. Please wait a few minutes before trying again.",
    },
    
    // Default fallback
    ErrorUnknown: {
        message: "An unexpected error occurred",
        help: "Please try again. If the problem persists, check the server logs or contact support.",
    },
};

/**
 * Get a user-friendly error message for a given error code
 */
export function getErrorMessage(code: string | undefined, defaultMessage?: string): { message: string; help?: string } {
    if (!code) {
        return {
            message: defaultMessage || errorMessages.ErrorUnknown.message,
            help: errorMessages.ErrorUnknown.help,
        };
    }
    
    return errorMessages[code] || {
        message: defaultMessage || `Error: ${code}`,
        help: "Please check the server logs for more details about this error.",
    };
}

/**
 * Format an error response with trace information for debugging
 */
export function formatErrorWithTrace(code: string | undefined, trace: string | undefined, defaultMessage?: string): string {
    const { message, help } = getErrorMessage(code, defaultMessage);
    
    let formatted = message;
    if (help) {
        formatted += `\n  ${help}`;
    }
    if (trace) {
        formatted += `\n  (Error ID: ${trace})`;
    }
    
    return formatted;
}
