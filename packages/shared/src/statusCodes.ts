export const CODE = {
    BadCredentials: {
        code: 'BAD_CREDENTIALS',
        message: 'Error: Email or password incorrect'
    },
    BannedWord: {
        code: 'BANNED_WORD',
        message: 'You used a banned word. If you think this is a mistake, please contact us.'
    },
    CannotDeleteYourself: {
        code: 'CANNOT_DELETE_YOURSELF',
        message: 'Error: What are you doing trying to delete your own account? I am disappointed :('
    },
    EmailInUse: {
        code: 'EMAIL_IN_USE',
        message: 'Error: Account with that email already exists'
    },
    EmailNotVerified: {
        code: 'EMAIL_NOT_VERIFIED',
        message: 'Error: Email has not been verified yet. Sending new verification email'
    },
    ErrorUnknown: {
        code: 'ERROR_UNKNOWN',
        message: 'Unknown error occurred'
    },
    ExportLimitReached: {
        code: 'EXPORT_LIMIT_REACHED',
        message: 'Account has exported too many times in a short period of time. Please wait at least 24 hours before trying again.'
    },
    HardLockout: {
        code: 'HARD_LOCKOUT',
        message: 'Error: Account locked. Contact us for assistance'
    },
    InternalError: {
        code: 'INTERNAL_ERROR',
        message: 'Internal error occurred'
    },
    InvalidArgs: {
        code: 'INVALID_ARGS',
        message: 'Error: Invalid arguments supplied'
    },
    InvalidResetCode: {
        code: 'INVALID_RESET_CODE',
        message: 'Error: Reset code expired or invalid. Sending a new code to your email.'
    },
    MaxNodesReached: {
        code: 'MAX_NODES_REACHED',
        message: 'Maximum number of nodes reached for routine. This limit has been set to an extreme value. If you believe this is a mistake, please contact us.'
    },
    MustResetPassword: {
        code: 'MUST_RESET_PASSWORD',
        message: 'Before signing in, please follow the link sent to your email to change your password.'
    },
    NonceExpired: {
        code: 'NONE_EXPIRED',
        message: 'Nonce has expired. Please restart the sign in process (and complete it quicker this time :)'
    },
    NotImplemented: {
        code: 'NOT_IMPLEMENTED',
        message: 'Error: This has not been implemented yet. Please be patient :)'
    },
    NotVerified: {
        code: 'NOT_VERIFIED',
        message: 'Error: Session token could not be verified. Please log back in'
    },
    NotYourWallet: {
        code: 'WALLET_ASSIGNED_TO_ANOTHER_ACCOUNT',
        message: 'Wallet has already been verified by another account. If you believe this is a mistake, please contact us.'
    },
    NoUser: {
        code: 'NO_USER',
        message: 'Error: No user with that email'
    },
    PhoneInUse: {
        code: 'PHONE_IN_USE',
        message: 'Error: Account with that phone number already exists'
    },
    SessionExpired: {
        code: 'SESSION_EXPIRED',
        message: 'Session has expired. Please log back in'
    },
    SoftLockout: {
        code: 'SOFT_LOCKOUT',
        message: 'Error: Too many log in attempts. Try again in 15 minutes'
    },
    SomeImagesAlreadyUploaded: {
        code: 'SOME_IMAGES_ALREADY_UPLOADED',
        message: 'Warning: Some images were already uploaded'
    },
    Unauthorized: {
        code: 'UNAUTHORIZED',
        message: 'Error: Not authorized to perform this action'
    },
    UsernameInUse: {
        code: 'USERNAME_IN_USE',
        message: 'Error: Account with that username already exists'
    },
}