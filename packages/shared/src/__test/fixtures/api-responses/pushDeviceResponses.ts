/* c8 ignore start */
/**
 * Push Device API Response Fixtures
 * 
 * Comprehensive fixtures for push notification device management including
 * device registration, notification testing, and subscription management.
 */

import type {
    PushDevice,
    PushDeviceCreateInput,
    PushDeviceUpdateInput,
    PushDeviceTestInput,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_DEVICES_PER_USER = 10;
const MAX_NAME_LENGTH = 100;
const PUSH_SUBSCRIPTION_EXPIRY_DAYS = 30;

// Push notification platforms
const PUSH_PLATFORMS = {
    IOS: "iOS",
    ANDROID: "Android",
    WEB_CHROME: "Chrome",
    WEB_FIREFOX: "Firefox",
    WEB_SAFARI: "Safari",
    WEB_EDGE: "Edge",
} as const;

/**
 * Push Device API response factory
 */
export class PushDeviceResponseFactory extends BaseAPIResponseFactory<
    PushDevice,
    PushDeviceCreateInput,
    PushDeviceUpdateInput
> {
    protected readonly entityName = "push_device";

    /**
     * Create mock push device data
     */
    createMockData(options?: MockDataOptions): PushDevice {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const deviceId = options?.overrides?.id || generatePK().toString();

        const basePushDevice: PushDevice = {
            __typename: "PushDevice",
            id: deviceId,
            created_at: now,
            updated_at: now,
            deviceId: `device_${deviceId.slice(-8)}`,
            name: "My Device",
            expires: new Date(Date.now() + (PUSH_SUBSCRIPTION_EXPIRY_DAYS * 24 * 60 * 60 * 1000)).toISOString(),
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...basePushDevice,
                deviceId: "comprehensive_push_device_12345",
                name: scenario === "edge-case" ? null : "iPhone 15 Pro Max - Safari",
                expires: scenario === "edge-case" 
                    ? new Date(Date.now() - (24 * 60 * 60 * 1000)).toISOString() // Expired yesterday
                    : new Date(Date.now() + (365 * 24 * 60 * 60 * 1000)).toISOString(), // 1 year from now
                ...options?.overrides,
            };
        }

        return {
            ...basePushDevice,
            ...options?.overrides,
        };
    }

    /**
     * Create push device from input
     */
    createFromInput(input: PushDeviceCreateInput): PushDevice {
        const now = new Date().toISOString();
        const deviceId = generatePK().toString();

        // Generate device ID from endpoint for uniqueness
        const endpointHash = this.generateHashFromEndpoint(input.endpoint);

        return {
            __typename: "PushDevice",
            id: deviceId,
            created_at: now,
            updated_at: now,
            deviceId: `device_${endpointHash}`,
            name: input.name || this.detectDeviceNameFromEndpoint(input.endpoint),
            expires: input.expires 
                ? new Date(Date.now() + (input.expires * 1000)).toISOString()
                : new Date(Date.now() + (PUSH_SUBSCRIPTION_EXPIRY_DAYS * 24 * 60 * 60 * 1000)).toISOString(),
        };
    }

    /**
     * Update push device from input
     */
    updateFromInput(existing: PushDevice, input: PushDeviceUpdateInput): PushDevice {
        const updates: Partial<PushDevice> = {
            updated_at: new Date().toISOString(),
        };

        if (input.name !== undefined) updates.name = input.name;
        if (input.expires !== undefined) {
            updates.expires = input.expires
                ? new Date(Date.now() + (input.expires * 1000)).toISOString()
                : null;
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: PushDeviceCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.endpoint) {
            errors.endpoint = "Push endpoint is required";
        } else if (!this.isValidPushEndpoint(input.endpoint)) {
            errors.endpoint = "Invalid push endpoint URL format";
        }

        if (!input.keys) {
            errors.keys = "Push subscription keys are required";
        } else {
            if (!input.keys.p256dh) {
                errors["keys.p256dh"] = "P256DH public key is required";
            } else if (!this.isValidBase64(input.keys.p256dh)) {
                errors["keys.p256dh"] = "P256DH key must be valid base64";
            }

            if (!input.keys.auth) {
                errors["keys.auth"] = "Auth secret is required";
            } else if (!this.isValidBase64(input.keys.auth)) {
                errors["keys.auth"] = "Auth secret must be valid base64";
            }
        }

        if (input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Device name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        if (input.expires !== undefined && input.expires !== null) {
            if (input.expires < 0) {
                errors.expires = "Expiration time cannot be negative";
            } else if (input.expires > (365 * 24 * 60 * 60)) { // 1 year in seconds
                errors.expires = "Expiration time cannot exceed 1 year";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: PushDeviceUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined && input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Device name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        if (input.expires !== undefined && input.expires !== null) {
            if (input.expires < 0) {
                errors.expires = "Expiration time cannot be negative";
            } else if (input.expires > (365 * 24 * 60 * 60)) { // 1 year in seconds
                errors.expires = "Expiration time cannot exceed 1 year";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create push devices for different platforms
     */
    createPushDevicesForAllPlatforms(): PushDevice[] {
        const platforms = [
            {
                platform: PUSH_PLATFORMS.IOS,
                name: "iPhone 15 Pro",
                endpoint: "https://fcm.googleapis.com/fcm/send/ios_device_123",
                expires: 365, // iOS devices typically have longer expiration
            },
            {
                platform: PUSH_PLATFORMS.ANDROID,
                name: "Samsung Galaxy S24",
                endpoint: "https://fcm.googleapis.com/fcm/send/android_device_456",
                expires: 180, // Android devices
            },
            {
                platform: PUSH_PLATFORMS.WEB_CHROME,
                name: "Chrome on Windows",
                endpoint: "https://fcm.googleapis.com/fcm/send/chrome_device_789",
                expires: 30, // Web browsers typically shorter
            },
            {
                platform: PUSH_PLATFORMS.WEB_FIREFOX,
                name: "Firefox on macOS",
                endpoint: "https://updates.push.services.mozilla.com/wpush/v2/firefox_device_abc",
                expires: 30,
            },
            {
                platform: PUSH_PLATFORMS.WEB_SAFARI,
                name: "Safari on iPhone",
                endpoint: "https://web.push.apple.com/safari_device_def",
                expires: 60, // Safari on iOS
            },
            {
                platform: PUSH_PLATFORMS.WEB_EDGE,
                name: "Edge on Windows",
                endpoint: "https://wns2-bn1p.notify.windows.com/edge_device_ghi",
                expires: 30,
            },
        ];

        return platforms.map((platform, index) => 
            this.createMockData({
                overrides: {
                    id: `device_${platform.platform.toLowerCase()}_${index}`,
                    deviceId: `${platform.platform.toLowerCase()}_device_${index}`,
                    name: platform.name,
                    expires: new Date(Date.now() + (platform.expires * 24 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create expired push devices
     */
    createExpiredPushDevices(): PushDevice[] {
        return [
            this.createMockData({
                overrides: {
                    name: "Expired Chrome Device",
                    deviceId: "expired_chrome_123",
                    expires: new Date(Date.now() - (24 * 60 * 60 * 1000)).toISOString(), // Expired yesterday
                },
            }),
            this.createMockData({
                overrides: {
                    name: "Old Firefox Device",
                    deviceId: "expired_firefox_456",
                    expires: new Date(Date.now() - (7 * 24 * 60 * 60 * 1000)).toISOString(), // Expired 7 days ago
                },
            }),
        ];
    }

    /**
     * Create push devices with different expiration times
     */
    createPushDevicesWithVariedExpiration(): PushDevice[] {
        const expirationVariations = [
            { name: "Expires Soon", days: 1 },
            { name: "Expires This Week", days: 7 },
            { name: "Expires This Month", days: 30 },
            { name: "Long-term Device", days: 365 },
            { name: "Already Expired", days: -1 },
        ];

        return expirationVariations.map((variation, index) => 
            this.createMockData({
                overrides: {
                    id: `device_expiry_${index}`,
                    name: variation.name,
                    deviceId: `expiry_test_${index}`,
                    expires: new Date(Date.now() + (variation.days * 24 * 60 * 60 * 1000)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create push test success response
     */
    createPushTestSuccessResponse(message = "Test notification sent successfully") {
        return this.createSuccessResponse({
            success: true,
            message,
            sentAt: new Date().toISOString(),
        });
    }

    /**
     * Create duplicate device error response
     */
    createDuplicateDeviceErrorResponse(endpoint: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "push_device",
            endpoint,
            suggestion: "Use the existing device or delete it first",
            message: "A push device with this endpoint already exists",
        });
    }

    /**
     * Create invalid subscription error response
     */
    createInvalidSubscriptionErrorResponse(reason = "Invalid endpoint or keys") {
        return this.createBusinessErrorResponse("invalid_subscription", {
            resource: "push_device",
            reason,
            suggestion: "Request new push notification permissions",
            message: "The push subscription is invalid or has expired",
        });
    }

    /**
     * Create push test failure error response
     */
    createPushTestFailureErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("push_test_failed", {
            resource: "push_device",
            reason,
            troubleshooting: [
                "Check if the device is online",
                "Verify notification permissions are granted",
                "Ensure the subscription is still valid",
            ],
            message: "Failed to send test notification",
        });
    }

    /**
     * Create device limit error response
     */
    createDeviceLimitErrorResponse(currentCount = MAX_DEVICES_PER_USER) {
        return this.createBusinessErrorResponse("limit", {
            resource: "push_device",
            limit: MAX_DEVICES_PER_USER,
            current: currentCount,
            message: `Maximum number of push devices (${MAX_DEVICES_PER_USER}) reached`,
        });
    }

    /**
     * Create subscription expired error response
     */
    createSubscriptionExpiredErrorResponse(deviceId: string, expiredAt: string) {
        return this.createBusinessErrorResponse("expired", {
            resource: "push_device",
            deviceId,
            expiredAt,
            suggestion: "Re-register the device with updated subscription",
            message: "Push device subscription has expired",
        });
    }

    /**
     * Generate hash from endpoint for device ID
     */
    private generateHashFromEndpoint(endpoint: string): string {
        // Simple hash function for demo purposes
        const hash = endpoint.split("").reduce((a, b) => {
            a = ((a << 5) - a) + b.charCodeAt(0);
            return a & a;
        }, 0);
        return Math.abs(hash).toString(16).substring(0, 8);
    }

    /**
     * Detect device name from endpoint
     */
    private detectDeviceNameFromEndpoint(endpoint: string): string {
        if (endpoint.includes("fcm.googleapis.com")) {
            return "Chrome/Android Device";
        } else if (endpoint.includes("mozilla.com")) {
            return "Firefox Device";
        } else if (endpoint.includes("apple.com")) {
            return "Safari Device";
        } else if (endpoint.includes("windows.com")) {
            return "Edge Device";
        }
        return "Unknown Device";
    }

    /**
     * Validate push endpoint URL
     */
    private isValidPushEndpoint(endpoint: string): boolean {
        try {
            const url = new URL(endpoint);
            const validDomains = [
                "fcm.googleapis.com",
                "updates.push.services.mozilla.com",
                "web.push.apple.com",
                "wns2-bn1p.notify.windows.com",
                "notify.windows.com",
            ];
            return validDomains.some(domain => url.hostname.includes(domain));
        } catch {
            return false;
        }
    }

    /**
     * Validate base64 string
     */
    private isValidBase64(str: string): boolean {
        try {
            return btoa(atob(str)) === str;
        } catch {
            return false;
        }
    }
}

/**
 * Pre-configured push device response scenarios
 */
export const pushDeviceResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<PushDeviceCreateInput>) => {
        const factory = new PushDeviceResponseFactory();
        const defaultInput: PushDeviceCreateInput = {
            endpoint: "https://fcm.googleapis.com/fcm/send/example_endpoint_123",
            keys: {
                p256dh: "BNbN3OiAQndKU2Qq9NKQkDLm1RLQZ6Q2Q4iQ5Q6Q7Q8Q9Q",
                auth: "tBHItJI5svbpez7KI4CCXg",
            },
            name: "Test Device",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (pushDevice?: PushDevice) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createSuccessResponse(
            pushDevice || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: PushDevice, updates?: Partial<PushDeviceUpdateInput>) => {
        const factory = new PushDeviceResponseFactory();
        const device = existing || factory.createMockData({ scenario: "complete" });
        const input: PushDeviceUpdateInput = {
            id: device.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(device, input),
        );
    },

    listSuccess: (pushDevices?: PushDevice[]) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPaginatedResponse(
            pushDevices || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: pushDevices?.length || DEFAULT_COUNT },
        );
    },

    allPlatformsSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        const devices = factory.createPushDevicesForAllPlatforms();
        return factory.createPaginatedResponse(
            devices,
            { page: 1, totalCount: devices.length },
        );
    },

    expiredDevicesSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        const devices = factory.createExpiredPushDevices();
        return factory.createPaginatedResponse(
            devices,
            { page: 1, totalCount: devices.length },
        );
    },

    variedExpirationSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        const devices = factory.createPushDevicesWithVariedExpiration();
        return factory.createPaginatedResponse(
            devices,
            { page: 1, totalCount: devices.length },
        );
    },

    activeDevicesSuccess: () => {
        const factory = new PushDeviceResponseFactory();
        const allDevices = factory.createPushDevicesForAllPlatforms();
        const activeDevices = allDevices.filter(device => 
            !device.expires || new Date(device.expires) > new Date(),
        );
        return factory.createPaginatedResponse(
            activeDevices,
            { page: 1, totalCount: activeDevices.length },
        );
    },

    testNotificationSuccess: (message?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPushTestSuccessResponse(message);
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createValidationErrorResponse({
            endpoint: "A valid push endpoint URL is required",
            "keys.p256dh": "P256DH public key is required",
            "keys.auth": "Auth secret is required",
        });
    },

    notFoundError: (deviceId?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createNotFoundErrorResponse(
            deviceId || "non-existent-device",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["push_device:write"],
        );
    },

    duplicateDeviceError: (endpoint?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createDuplicateDeviceErrorResponse(
            endpoint || "https://fcm.googleapis.com/fcm/send/existing_endpoint",
        );
    },

    invalidSubscriptionError: (reason?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createInvalidSubscriptionErrorResponse(reason);
    },

    deviceLimitError: (currentCount?: number) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createDeviceLimitErrorResponse(currentCount);
    },

    subscriptionExpiredError: (deviceId?: string) => {
        const factory = new PushDeviceResponseFactory();
        return factory.createSubscriptionExpiredErrorResponse(
            deviceId || generatePK().toString(),
            new Date(Date.now() - (24 * 60 * 60 * 1000)).toISOString(),
        );
    },

    testNotificationFailedError: (reason = "Device is offline") => {
        const factory = new PushDeviceResponseFactory();
        return factory.createPushTestFailureErrorResponse(reason);
    },

    invalidEndpointError: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createValidationErrorResponse({
            endpoint: "Invalid push endpoint URL format",
        });
    },

    invalidKeysError: () => {
        const factory = new PushDeviceResponseFactory();
        return factory.createValidationErrorResponse({
            "keys.p256dh": "P256DH key must be valid base64",
            "keys.auth": "Auth secret must be valid base64",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new PushDeviceResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new PushDeviceResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new PushDeviceResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const pushDeviceResponseFactory = new PushDeviceResponseFactory();
