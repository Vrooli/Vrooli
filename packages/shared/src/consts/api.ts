import { ValueOf } from ".";

export const COOKIE = {
    Jwt: "session-f234y7fdiafhdja2",
} as const;
export type COOKIE = ValueOf<typeof COOKIE>;

export enum HttpStatus {
    Ok = 200,
    Created = 201,
    Accepted = 202,
    NoContent = 204,
    BadRequest = 400,
    Unauthorized = 401,
    PaymentRequired = 402,
    Forbidden = 403,
    NotFound = 404,
    MethodNotAllowed = 405,
    NotAcceptable = 406,
    RequestTimeout = 408,
    Conflict = 409,
    Gone = 410,
    LengthRequired = 411,
    PreconditionFailed = 412,
    PayloadTooLarge = 413,
    URITooLong = 414,
    UnsupportedMediaType = 415,
    RangeNotSatisfiable = 416,
    ExpectationFailed = 417,
    IAmATeapot = 418,
    MisdirectedRequest = 421,
    UnprocessableEntity = 422,
    Locked = 423,
    FailedDependency = 424,
    TooEarly = 425,
    UpgradeRequired = 426,
    PreconditionRequired = 428,
    TooManyRequests = 429,
    RequestHeaderFieldsTooLarge = 431,
    UnavailableForLegalReasons = 451,
    InternalServerError = 500,
    NotImplemented = 501,
    BadGateway = 502,
    ServiceUnavailable = 503,
    GatewayTimeout = 504,
    HTTPVersionNotSupported = 505,
    VariantAlsoNegotiates = 506,
    InsufficientStorage = 507,
    LoopDetected = 508,
    NotExtended = 510,
    NetworkAuthenticationRequired = 511
}

/**
 * The number of API credits a user gets for free, 
 * when they verify their phone number.
 * 
 * These are based on USD cents multiplied by 1_000_000. 
 * This way, we can use integers instead of floats.
 */
export const API_CREDITS_FREE = BigInt(100_000_000);
/**
 * The number of API credits a user gets for a standard 
 * premium subscription.
 * 
 * These are based on USD cents multiplied by 1_000_000. 
 * This way, we can use integers instead of floats.
 * 
 * NOTE: This should ideally be less than the cost of a premium subscription, 
 * so that we don't lose money. This is the main source of expenses for the app.
 */
export const API_CREDITS_PREMIUM = BigInt(1_500_000_000);
