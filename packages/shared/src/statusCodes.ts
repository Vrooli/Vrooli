/**
 * All error codes used by the API.
 * Each code is associated with a message, and an optional snack message.
 */
export const CODE = {
    BadCredentials: {
        code: 'BadCredentials',
        message: 'Email or password incorrect',
    },
    BannedWord: {
        code: 'BannedWord',
        message: 'You used a banned word. If you think this is a mistake, please contact us.',
        snack: 'Banned word detected.',
    },
    CannotDeleteYourself: {
        code: 'CannotDeleteYourself',
        message: 'What are you doing trying to delete your own account? I am disappointed :(',
        snack: 'Cannot delete yourself',
    },
    EmailInUse: {
        code: 'EmailInUse',
        message: 'Account with that email already exists',
        snack: 'Email already in use',
    },
    EmailNotFound: {
        code: 'EmailNotFound',
        message: 'Could not find an account with that email',
        snack: 'Email not found',
    },
    EmailNotVerified: {
        code: 'EmailNotVerified',
        message: 'Email has not been verified yet. Sending new verification email',
        snack: 'Email not verified',
    },
    ErrorUnknown: {
        code: 'ErrorUnknown',
        message: 'Unknown error occurred',
    },
    ExportLimitReached: {
        code: 'ExportLimitReached',
        message: 'Account has exported too many times in a short period of time. Please wait at least 24 hours before trying again.',
        snack: 'Export limit reached',
    },
    HardLockout: {
        code: 'HardLockout',
        message: 'Account locked. Contact us for assistance',
        snack: 'Account locked',
    },
    InternalError: {
        code: 'InternalError',
        message: 'Internal error occurred',
    },
    InvalidArgs: {
        code: 'InvalidArgs',
        message: 'Invalid arguments supplied'
    },
    InvalidResetCode: {
        code: 'InvalidResetCode',
        message: 'Reset code expired or invalid. Sending a new code to your email.',
        snack: 'Invalid reset code',
    },
    MaxNodesReached: {
        code: 'MaxNodesReached',
        message: 'Maximum number of nodes reached for routine. This limit has been set to an extreme value. If you believe this is a mistake, please contact us.',
        snack: 'Max nodes reached',
    },
    MustResetPassword: {
        code: 'MustResetPassword',
        message: 'Before signing in, please follow the link sent to your email to change your password.',
        snack: 'Must reset password',
    },
    NonceExpired: {
        code: 'NonceExpired',
        message: 'Nonce has expired. Please restart the sign in process (and complete it quicker this time :)',
        snack: 'Nonce expired',
    },
    NotFound: {
        code: 'NotFound',
        message: 'Resource not found'
    },
    NotImplemented: {
        code: 'NotImplemented',
        message: 'This has not been implemented yet. Please be patient :)',
        snack: 'Not implemented',
    },
    NotVerified: {
        code: 'NotVerified',
        message: 'Session token could not be verified. Please log back in',
        snack: 'Session expired/invalid',
    },
    NotYourWallet: {
        code: 'NotYourWallet',
        message: 'Wallet has already been verified by another account. If you believe this is a mistake, please contact us.',
        snack: 'Wallet taken',
    },
    NoUser: {
        code: 'NoUser',
        message: 'No user with that email',
        snack: 'User not found',
    },
    PhoneInUse: {
        code: 'PhoneInUse',
        message: 'Account with that phone number already exists',
        snack: 'Phone number taken',
    },
    SessionExpired: {
        code: 'SessionExpired',
        message: 'Session has expired. Please log back in',
        snack: 'Session expired',
    },
    SoftLockout: {
        code: 'SoftLockout',
        message: 'Too many log in attempts. Try again in 15 minutes',
        snack: 'Too many attempts. Wait 15 minutes',
    },
    SomeImagesAlreadyUploaded: {
        code: 'SomeImagesAlreadyUploaded',
        message: 'Warning: Some images were already uploaded',
        snack: 'Some images already uploaded',
    },
    Unauthorized: {
        code: 'Unauthorized',
        message: 'Not authorized to perform this action',
        snack: 'Unauthorized',
    },
    UsernameInUse: {
        code: 'UsernameInUse',
        message: 'Account with that username already exists',
        snack: 'Username taken',
    },
}