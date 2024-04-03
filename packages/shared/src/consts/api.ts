import { ValueOf } from ".";
import { LlmTask } from "../api/generated/graphqlTypes";

export const COOKIE = {
    Jwt: "session-f234y7fdiafhdja2",
} as const;
export type COOKIE = ValueOf<typeof COOKIE>;

export enum HttpStatus {
    Ok = 200,
    Created = 201,
    Accepted = 202,
    NoContent = 204,
    SeeOther = 303,
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
 * The multiplier to convert USD cents to API credits. 
 * Allows us to track fractional cents without using floats.
 */
export const API_CREDITS_MULTIPLIER = BigInt(1_000_000);
/**
 * The number of API credits a user gets for free, 
 * when they verify their phone number.
 */
export const API_CREDITS_FREE = BigInt(200) * API_CREDITS_MULTIPLIER;
/**
 * The number of API credits a user gets for a standard 
 * premium subscription.
 * 
 * NOTE: This should ideally be less than the cost of a premium subscription, 
 * so that we don't lose money. This is the main source of expenses for the app.
 */
export const API_CREDITS_PREMIUM = BigInt(1_500) * API_CREDITS_MULTIPLIER;

/**
 * A task that an AI can, is, or has performed, along with 
 * other information about the task for display and execution.
 */
export type LlmTaskInfo = {
    /** 
     * The action being performed. Can be thought of as a modifier to the command. 
     * For example, if the command is "note", the action could be "add" or "delete".
     * 
     * NOTE: This is in the user's language, not the server's language.
     */
    action: string | null;
    /**
     * The command string of the task, in the user's language.
     */
    command: string;
    /** Unique ID to track command */
    id: string;
    /** A user-friendly label to display to the user */
    label: string;
    /** Message associated with the command */
    messageId?: string;
    /**
     * Data passed in when executing the task.
     */
    properties: {
        [key: string]: string | number | null;
    } | null;
    /**
     * The task being performed, as a language-independent type.
     */
    task: LlmTask | `${LlmTask}`;
    /**
     * The latest status of the command.
     */
    status: "suggested" | "running" | "completed" | "failed"
};
