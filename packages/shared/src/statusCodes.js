export const CODE = {
    SomeImagesAlreadyUploaded: {
        code: 'SOME_IMAGES_ALREADY_UPLOADED',
        message: 'Warning: Some images were already uploaded'
    },
    ErrorUnknown: {
        code: 'ERROR_UNKNOWN',
        message: 'Unknown error occurred'
    },
    InvalidArgs: {
        code: 'INVALID_ARGS',
        message: 'Error: Invalid arguments supplied'
    },
    PhoneInUse: {
        code: 'PHONE_IN_USE',
        message: 'Error: Account with that phone number already exists'
    },
    EmailInUse: {
        code: 'EMAIL_IN_USE',
        message: 'Error: Account with that email already exists'
    },
    EmailNotVerified: {
        code: 'EMAIL_NOT_VERIFIED',
        message: 'Error: Email has not been verified yet. Sending new verification email'
    },
    NoCustomer: {
        code: 'NO_CUSTOMER',
        message: 'Error: No user with that email'
    },
    BadCredentials: {
        code: 'BAD_CREDENTIALS',
        message: 'Error: Email or password incorrect'
    },
    SoftLockout: {
        code: 'SOFT_LOCKOUT',
        message: 'Error: Too many login attempts. Try again in 15 minutes'
    },
    HardLockout: {
        code: 'HARD_LOCKOUT',
        message: 'Error: Account locked. Contact us for assistance'
    },
    NotVerified: {
        code: 'NOT_VERIFIED',
        message: 'Error: Session token could not be verified. Please log back in'
    },
    Unauthorized: {
        code: 'UNAUTHORIZED',
        message: 'Error: Not authorized to perform this action'
    },
    CannotDeleteYourself: {
        code: 'CANNOT_DELETE_YOURSELF',
        message: 'Error: What are you doing trying to delete your own account? I am disappointed :('
    },
    NotImplemented: {
        code: 'NOT_IMPLEMENTED',
        message: 'Error: This has not been implemented yet. Please be patient :)'
    },
    InvalidResetCode: {
        code: 'INVALID_RESET_CODE',
        message: 'Error: Reset code expired or invalid. Sending a new code to your email.'
    },
    MustResetPassword: {
        code: 'MUST_RESET_PASSWORD',
        message: 'Before signing in, please follow the link sent to your email to change your password.'
    }
}