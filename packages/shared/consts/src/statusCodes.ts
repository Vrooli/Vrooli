export type ApolloErrorCode = {
    /**
     * Error code
     */
    code: string;
    /**
     * Detailed error message
     */
    message: string;
    /**
     * Short error message
     */
    snack?: string;
}

/**
 * All error codes used by the API.
 * Each code is associated with a message, and an optional snack message.
 * TODO will be deprecated when internationalization added to UI
 */
export type CODE = 'BadCredentials' |
    'BannedWord' |
    'CannotDeleteYourself' |
    'EmailInUse' |
    'EmailNotFound' |
    'EmailNotVerified' |
    'ErrorUnknown' |
    'ExportLimitReached' |
    'HardLockout' |
    'InternalError' |
    'InvalidArgs' |
    'InvalidResetCode' |
    'LineBreaksBio' |
    'LineBreaksDescription' |
    'MaxNodesReached' |
    'MaxObjectsReached' |
    'MustResetPassword' |
    'NodeDuplicatePosition' |
    'NonceExpired' |
    'NotFound' |
    'NotImplemented' |
    'NotVerified' |
    'NotYourWallet' |
    'NoUser' |
    'PhoneInUse' |
    'RateLimitExceeded' |
    'ReportExists' |
    'SessionExpired' |
    'SoftLockout' |
    'StandardDuplicateShape' |
    'StandardDuplicateName' |
    'Unauthorized';