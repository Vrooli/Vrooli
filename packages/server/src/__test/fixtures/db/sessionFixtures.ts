// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for Session model - used for seeding auth test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
let _sessionDbIds: Record<string, bigint> | null = null;
export const sessionDbIds = {
    get activeSession1() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.activeSession1;
    },
    get activeSession2() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.activeSession2;
    },
    get expiredSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.expiredSession;
    },
    get revokedSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.revokedSession;
    },
    get recentSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.recentSession;
    },
    get oldSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.oldSession;
    },
    get mobileSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.mobileSession;
    },
    get desktopSession() {
        if (!_sessionDbIds) {
            _sessionDbIds = {
                activeSession1: generatePK(),
                activeSession2: generatePK(),
                expiredSession: generatePK(),
                revokedSession: generatePK(),
                recentSession: generatePK(),
                oldSession: generatePK(),
                mobileSession: generatePK(),
                desktopSession: generatePK(),
            };
        }
        return _sessionDbIds.desktopSession;
    },
};

/**
 * Active session - standard user
 */
let _activeSessionDb: Prisma.sessionCreateInput | null = null;
export const activeSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_activeSessionDb) {
            _activeSessionDb = {
                id: sessionDbIds.activeSession1,
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
                last_refresh_at: new Date(),
                device_info: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                ip_address: "192.168.1.100",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _activeSessionDb;
    },
};

/**
 * Recently created active session
 */
let _recentActiveSessionDb: Prisma.sessionCreateInput | null = null;
export const recentActiveSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_recentActiveSessionDb) {
            _recentActiveSessionDb = {
                id: sessionDbIds.activeSession2,
                expires_at: new Date(Date.now() + 29 * 24 * 60 * 60 * 1000), // 29 days from now
                last_refresh_at: new Date(Date.now() - 60 * 1000), // Refreshed 1 minute ago
                device_info: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
                ip_address: "192.168.1.101",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _recentActiveSessionDb;
    },
};

/**
 * Expired session
 */
let _expiredSessionDb: Prisma.sessionCreateInput | null = null;
export const expiredSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_expiredSessionDb) {
            _expiredSessionDb = {
                id: sessionDbIds.expiredSession,
                expires_at: new Date(Date.now() - 24 * 60 * 60 * 1000), // Expired 1 day ago
                last_refresh_at: new Date(Date.now() - 25 * 60 * 60 * 1000), // Last refreshed 25 hours ago
                device_info: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                ip_address: "192.168.1.102",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _expiredSessionDb;
    },
};

/**
 * Revoked session
 */
let _revokedSessionDb: Prisma.sessionCreateInput | null = null;
export const revokedSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_revokedSessionDb) {
            _revokedSessionDb = {
                id: sessionDbIds.revokedSession,
                expires_at: new Date(Date.now() + 20 * 24 * 60 * 60 * 1000), // Would expire in 20 days
                last_refresh_at: new Date(Date.now() - 2 * 60 * 60 * 1000), // Last refreshed 2 hours ago
                revokedAt: new Date(Date.now() - 60 * 60 * 1000), // Revoked 1 hour ago
                device_info: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
                ip_address: "192.168.1.103",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _revokedSessionDb;
    },
};

/**
 * Recently refreshed session
 */
let _recentSessionDb: Prisma.sessionCreateInput | null = null;
export const recentSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_recentSessionDb) {
            _recentSessionDb = {
                id: sessionDbIds.recentSession,
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
                last_refresh_at: new Date(Date.now() - 5 * 60 * 1000), // Refreshed 5 minutes ago
                device_info: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
                ip_address: "10.0.0.50",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _recentSessionDb;
    },
};

/**
 * Old session (near expiry)
 */
let _oldSessionDb: Prisma.sessionCreateInput | null = null;
export const oldSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_oldSessionDb) {
            _oldSessionDb = {
                id: sessionDbIds.oldSession,
                expires_at: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000), // Expires in 2 days
                last_refresh_at: new Date(Date.now() - 28 * 24 * 60 * 60 * 1000), // Last refreshed 28 days ago
                device_info: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                ip_address: "192.168.1.104",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _oldSessionDb;
    },
};

/**
 * Mobile session with detailed device info
 */
let _mobileSessionDb: Prisma.sessionCreateInput | null = null;
export const mobileSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_mobileSessionDb) {
            _mobileSessionDb = {
                id: sessionDbIds.mobileSession,
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
                last_refresh_at: new Date(Date.now() - 30 * 60 * 1000), // Refreshed 30 minutes ago
                device_info: JSON.stringify({
                    userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15",
                    platform: "iPhone",
                    vendor: "Apple Computer, Inc.",
                    language: "en-US",
                    screenResolution: "1179x2556",
                    timezone: "America/New_York",
                    cookieEnabled: true,
                    doNotTrack: false,
                }),
                ip_address: "172.16.0.25",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _mobileSessionDb;
    },
};

/**
 * Desktop session with detailed device info
 */
let _desktopSessionDb: Prisma.sessionCreateInput | null = null;
export const desktopSessionDb = {
    get data(): Prisma.sessionCreateInput {
        if (!_desktopSessionDb) {
            _desktopSessionDb = {
                id: sessionDbIds.desktopSession,
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
                last_refresh_at: new Date(Date.now() - 15 * 60 * 1000), // Refreshed 15 minutes ago
                device_info: JSON.stringify({
                    userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                    platform: "Win32",
                    vendor: "Google Inc.",
                    language: "en-US",
                    screenResolution: "1920x1080",
                    timezone: "America/Los_Angeles",
                    cookieEnabled: true,
                    doNotTrack: false,
                    hardwareConcurrency: 8,
                    memory: 8192,
                }),
                ip_address: "203.0.113.42",
                user: {
                    connect: { id: generatePK() },
                },
                auth: {
                    connect: { id: generatePK() },
                },
            };
        }
        return _desktopSessionDb;
    },
};

/**
 * Factory functions for creating sessions dynamically
 */
export class SessionDbFactory {
    /**
     * Create minimal session
     */
    static createMinimal(overrides?: Partial<Prisma.sessionCreateInput>): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
            last_refresh_at: new Date(),
            ip_address: "127.0.0.1",
            user: { connect: { id: generatePK() } },
            auth: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    /**
     * Create active session for user
     */
    static createActive(
        userId: bigint,
        authId: bigint,
        daysUntilExpiry = 30,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + daysUntilExpiry * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(),
            device_info: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
            ip_address: "192.168.1.100",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create expired session
     */
    static createExpired(
        userId: bigint,
        authId: bigint,
        expiredDaysAgo = 1,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() - expiredDaysAgo * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(Date.now() - (expiredDaysAgo + 1) * 60 * 60 * 1000),
            device_info: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
            ip_address: "192.168.1.101",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create revoked session
     */
    static createRevoked(
        userId: bigint,
        authId: bigint,
        revokedHoursAgo = 1,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 20 * 24 * 60 * 60 * 1000), // Would have been valid
            last_refresh_at: new Date(Date.now() - (revokedHoursAgo + 1) * 60 * 60 * 1000),
            revokedAt: new Date(Date.now() - revokedHoursAgo * 60 * 60 * 1000),
            device_info: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
            ip_address: "192.168.1.102",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create mobile session
     */
    static createMobile(
        userId: bigint,
        authId: bigint,
        deviceType: "iPhone" | "Android" = "iPhone",
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        const deviceInfos = {
            iPhone: {
                userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15",
                platform: "iPhone",
                vendor: "Apple Computer, Inc.",
            },
            Android: {
                userAgent: "Mozilla/5.0 (Linux; Android 14; SM-G998B) AppleWebKit/537.36",
                platform: "Linux armv81",
                vendor: "",
            },
        };

        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(),
            device_info: JSON.stringify(deviceInfos[deviceType]),
            ip_address: "172.16.0.25",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create desktop session
     */
    static createDesktop(
        userId: bigint,
        authId: bigint,
        browser: "Chrome" | "Firefox" | "Safari" = "Chrome",
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        const userAgents = {
            Chrome: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            Firefox: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
            Safari: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
        };

        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(),
            device_info: userAgents[browser],
            ip_address: "203.0.113.42",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create session with specific IP and location
     */
    static createWithLocation(
        userId: bigint,
        authId: bigint,
        ipAddress: string,
        location?: {
            country?: string;
            region?: string;
            city?: string;
            timezone?: string;
        },
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        const deviceInfo = location ? JSON.stringify({
            userAgent: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
            location,
            timezone: location.timezone || "UTC",
        }) : "Mozilla/5.0 (compatible; TestBrowser/1.0)";

        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(),
            device_info: deviceInfo,
            ip_address: ipAddress,
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create session near expiry (for testing renewal logic)
     */
    static createNearExpiry(
        userId: bigint,
        authId: bigint,
        hoursUntilExpiry = 2,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + hoursUntilExpiry * 60 * 60 * 1000),
            last_refresh_at: new Date(Date.now() - 28 * 24 * 60 * 60 * 1000), // Old refresh
            device_info: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
            ip_address: "192.168.1.103",
            user: { connect: { id: userId } },
            auth: { connect: { id: authId } },
            ...overrides,
        };
    }

    /**
     * Create multiple sessions for the same user (testing concurrent sessions)
     */
    static createMultipleForUser(
        userId: bigint,
        authId: bigint,
        count = 3,
    ): Prisma.sessionCreateInput[] {
        const sessions: Prisma.sessionCreateInput[] = [];

        for (let i = 0; i < count; i++) {
            sessions.push({
                id: generatePK(),
                expires_at: new Date(Date.now() + (30 - i) * 24 * 60 * 60 * 1000), // Staggered expiry
                last_refresh_at: new Date(Date.now() - i * 60 * 60 * 1000), // Staggered refresh
                device_info: `Session ${i + 1} - Mozilla/5.0 (compatible; TestBrowser/1.0)`,
                ip_address: `192.168.1.${100 + i}`,
                user: { connect: { id: userId } },
                auth: { connect: { id: authId } },
            });
        }

        return sessions;
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed basic session scenarios for testing
 */
export async function seedTestSessions(db: PrismaClient) {
    // Create user and auth first
    const user1 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "session_user_1",
            name: "Session Test User 1",
            handle: "sessionuser1",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const auth1 = await db.user_auth.create({
        data: {
            id: generatePK(),
            provider: "Password",
            userId: user1.id,
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        },
    });

    const user2 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "session_user_2",
            name: "Session Test User 2",
            handle: "sessionuser2",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const auth2 = await db.user_auth.create({
        data: {
            id: generatePK(),
            provider: "Password",
            userId: user2.id,
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        },
    });

    // Create sessions
    const sessions = await Promise.all([
        db.session.create({
            data: SessionDbFactory.createActive(user1.id, auth1.id),
        }),
        db.session.create({
            data: SessionDbFactory.createExpired(user1.id, auth1.id),
        }),
        db.session.create({
            data: SessionDbFactory.createRevoked(user2.id, auth2.id),
        }),
        db.session.create({
            data: SessionDbFactory.createMobile(user2.id, auth2.id, "iPhone"),
        }),
    ]);

    return { user1, user2, auth1, auth2, sessions };
}

/**
 * Create session history for testing session management
 */
export async function seedSessionHistory(db: PrismaClient, userId: bigint, authId: bigint) {
    const sessionHistory = SessionDbFactory.createMultipleForUser(userId, authId, 5);

    // Add one revoked and one expired session
    sessionHistory.push(
        SessionDbFactory.createRevoked(userId, authId, 24), // Revoked 1 day ago
        SessionDbFactory.createExpired(userId, authId, 7),  // Expired 1 week ago
    );

    const sessions = await Promise.all(
        sessionHistory.map(session => db.session.create({ data: session })),
    );

    return sessions;
}
