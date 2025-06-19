import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for PushDevice model - used for seeding test data
 * These fixtures represent web push notification devices registered for users
 */

// Consistent IDs for testing
export const pushDeviceDbIds = {
    device1: generatePK(),
    device2: generatePK(),
    device3: generatePK(),
    expiredDevice: generatePK(),
    namedDevice: generatePK(),
};

/**
 * Minimal push device data for database creation
 */
export const minimalPushDeviceDb: Omit<Prisma.push_deviceCreateInput, "user"> = {
    id: pushDeviceDbIds.device1,
    endpoint: "https://fcm.googleapis.com/fcm/send/test-endpoint-123",
    p256dh: "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM=",
    auth: "tBHItJI5svbpez7KI4CCXg==",
};

/**
 * Push device with expiration date
 */
export const pushDeviceWithExpirationDb: Omit<Prisma.push_deviceCreateInput, "user"> = {
    id: pushDeviceDbIds.expiredDevice,
    endpoint: "https://fcm.googleapis.com/fcm/send/expired-endpoint-456",
    p256dh: "BLc4xRzKlKORKWlbdgFaBrrPK3ydWAHo4M0gs0HPQyrdH0n8vUy1PErkjLIOXEJwBNgSq5BIPd1cJhO3e2ANQ2A=",
    auth: "R9siYqNVHMSGgv0Rmr6N4A==",
    expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
};

/**
 * Complete push device with all fields
 */
export const completePushDeviceDb: Omit<Prisma.push_deviceCreateInput, "user"> = {
    id: pushDeviceDbIds.namedDevice,
    endpoint: "https://fcm.googleapis.com/fcm/send/complete-endpoint-789",
    p256dh: "BGEEGjvAycqfE7MmrYJaiRpFbE9f7yg7Lzwrh3qipBD8CxNel6X5V5jmoF5eazsJlkZ2wveJQU6kJHcqUIIKdmU=",
    auth: "ufn2jFsZSRpJu7rKn1VgvQ==",
    expires: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
    name: "Chrome on Windows Desktop",
};

/**
 * Factory for creating push device database fixtures with overrides
 */
export class PushDeviceDbFactory {
    private static counter = 0;

    /**
     * Generate realistic endpoint URL
     */
    private static generateEndpoint(): string {
        this.counter++;
        const providers = [
            "https://fcm.googleapis.com/fcm/send/",
            "https://updates.push.services.mozilla.com/wpush/v2/",
            "https://wns2-ln2p.notify.windows.com/w/",
        ];
        const provider = providers[this.counter % providers.length];
        return `${provider}${Math.random().toString(36).substring(2, 15)}`;
    }

    /**
     * Generate realistic p256dh key
     */
    private static generateP256dh(): string {
        // Real p256dh keys are base64-encoded 65-byte values
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
        let result = "";
        for (let i = 0; i < 87; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result + "=";
    }

    /**
     * Generate realistic auth key
     */
    private static generateAuth(): string {
        // Real auth keys are base64-encoded 16-byte values
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
        let result = "";
        for (let i = 0; i < 22; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result + "==";
    }

    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.push_deviceCreateInput>
    ): Prisma.push_deviceCreateInput {
        return {
            id: generatePK(),
            endpoint: this.generateEndpoint(),
            p256dh: this.generateP256dh(),
            auth: this.generateAuth(),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithExpiration(
        userId: string,
        daysUntilExpiration: number = 30,
        overrides?: Partial<Prisma.push_deviceCreateInput>
    ): Prisma.push_deviceCreateInput {
        return {
            ...this.createMinimal(userId),
            expires: new Date(Date.now() + daysUntilExpiration * 24 * 60 * 60 * 1000),
            ...overrides,
        };
    }

    static createExpired(
        userId: string,
        overrides?: Partial<Prisma.push_deviceCreateInput>
    ): Prisma.push_deviceCreateInput {
        return {
            ...this.createMinimal(userId),
            expires: new Date(Date.now() - 24 * 60 * 60 * 1000), // Expired yesterday
            ...overrides,
        };
    }

    static createNamed(
        userId: string,
        name: string,
        overrides?: Partial<Prisma.push_deviceCreateInput>
    ): Prisma.push_deviceCreateInput {
        return {
            ...this.createMinimal(userId),
            name,
            ...overrides,
        };
    }

    /**
     * Create multiple devices for a user (simulating multiple browsers/devices)
     */
    static createMultiple(
        userId: string,
        count: number = 3,
        options?: {
            withNames?: boolean;
            withExpiration?: boolean;
        }
    ): Prisma.push_deviceCreateInput[] {
        const devices: Prisma.push_deviceCreateInput[] = [];
        const deviceNames = [
            "Chrome on Windows Desktop",
            "Safari on iPhone",
            "Firefox on MacBook",
            "Chrome on Android",
            "Edge on Surface",
        ];

        for (let i = 0; i < count; i++) {
            let device: Prisma.push_deviceCreateInput;

            if (options?.withNames) {
                device = this.createNamed(userId, deviceNames[i % deviceNames.length]);
            } else {
                device = this.createMinimal(userId);
            }

            if (options?.withExpiration) {
                // Vary expiration times
                const daysUntilExpiration = 30 + (i * 30); // 30, 60, 90 days etc
                device.expires = new Date(Date.now() + daysUntilExpiration * 24 * 60 * 60 * 1000);
            }

            devices.push(device);
        }

        return devices;
    }
}

/**
 * Helper to seed push devices for testing
 */
export async function seedPushDevices(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        includeExpired?: boolean;
        includeNamed?: boolean;
    }
) {
    const devices = [];
    const count = options.count || 1;

    // Create regular devices
    const regularDevices = PushDeviceDbFactory.createMultiple(
        options.userId,
        count,
        {
            withNames: options.includeNamed,
            withExpiration: true,
        }
    );

    for (const deviceData of regularDevices) {
        const device = await prisma.push_device.create({ data: deviceData });
        devices.push(device);
    }

    // Add expired device if requested
    if (options.includeExpired) {
        const expiredDevice = await prisma.push_device.create({
            data: PushDeviceDbFactory.createExpired(options.userId, {
                name: options.includeNamed ? "Old Browser (Expired)" : undefined,
            }),
        });
        devices.push(expiredDevice);
    }

    return devices;
}

/**
 * Helper to clean up push devices
 */
export async function cleanupPushDevices(
    prisma: any,
    deviceIds: string[]
) {
    await prisma.push_device.deleteMany({
        where: { id: { in: deviceIds } },
    });
}

/**
 * Helper to verify push device state
 */
export async function verifyPushDeviceState(
    prisma: any,
    deviceId: string,
    expected: {
        endpoint?: string;
        name?: string;
        expires?: Date;
        userId?: string;
    }
) {
    const device = await prisma.push_device.findUnique({
        where: { id: deviceId },
        include: { user: true },
    });

    expect(device).toBeDefined();
    
    if (expected.endpoint) {
        expect(device.endpoint).toBe(expected.endpoint);
    }
    
    if (expected.name !== undefined) {
        expect(device.name).toBe(expected.name);
    }
    
    if (expected.expires) {
        expect(device.expires).toEqual(expected.expires);
    }
    
    if (expected.userId) {
        expect(device.userId).toBe(expected.userId);
    }

    return device;
}