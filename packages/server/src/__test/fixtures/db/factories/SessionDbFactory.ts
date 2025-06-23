import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type session, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface SessionRelationConfig extends RelationConfig {
    withUser?: boolean;
    withAuth?: boolean;
    isExpired?: boolean;
    isRevoked?: boolean;
    deviceType?: "desktop" | "mobile" | "tablet";
}

/**
 * Database fixture factory for session model
 * Handles user sessions with support for multiple devices and expiration
 */
export class SessionDbFactory extends DatabaseFixtureFactory<
    session,
    Prisma.sessionCreateInput,
    Prisma.sessionInclude,
    Prisma.sessionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("session", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.session;
    }

    protected trackCreatedId(id: string | bigint): void {
        (this as any).createdIds.add(id.toString());
    }

    protected getMinimalData(overrides?: Partial<Prisma.sessionCreateInput>): Prisma.sessionCreateInput {
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

    protected getCompleteData(overrides?: Partial<Prisma.sessionCreateInput>): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
            last_refresh_at: new Date(),
            device_info: JSON.stringify({
                userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                platform: "Win32",
                vendor: "Google Inc.",
                language: "en-US",
                screenResolution: "1920x1080",
                timezone: "America/New_York",
                cookieEnabled: true,
                doNotTrack: false,
            }),
            ip_address: "192.168.1.100",
            user: { connect: { id: generatePK() } },
            auth: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    /**
     * Create active session
     */
    async createActive(
        daysUntilExpiry = 30,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        return this.createComplete({
            expires_at: new Date(Date.now() + daysUntilExpiry * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(),
            revokedAt: null,
            ...overrides,
        });
    }

    /**
     * Create expired session
     */
    async createExpired(
        daysExpiredAgo = 1,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        return this.createMinimal({
            expires_at: new Date(Date.now() - daysExpiredAgo * 24 * 60 * 60 * 1000),
            last_refresh_at: new Date(Date.now() - (daysExpiredAgo + 1) * 24 * 60 * 60 * 1000),
            ...overrides,
        });
    }

    /**
     * Create revoked session
     */
    async createRevoked(
        hoursRevokedAgo = 1,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        return this.createComplete({
            expires_at: new Date(Date.now() + 20 * 24 * 60 * 60 * 1000), // Would have been valid
            revokedAt: new Date(Date.now() - hoursRevokedAgo * 60 * 60 * 1000),
            last_refresh_at: new Date(Date.now() - (hoursRevokedAgo + 2) * 60 * 60 * 1000),
            ...overrides,
        });
    }

    /**
     * Create mobile session
     */
    async createMobile(
        deviceType: "iPhone" | "Android" = "iPhone",
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        const deviceInfos = {
            iPhone: {
                userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15",
                platform: "iPhone",
                vendor: "Apple Computer, Inc.",
                isMobile: true,
                screenResolution: "1179x2556",
            },
            Android: {
                userAgent: "Mozilla/5.0 (Linux; Android 14; SM-G998B) AppleWebKit/537.36",
                platform: "Linux armv81",
                vendor: "",
                isMobile: true,
                screenResolution: "1440x3200",
            },
        };

        return this.createComplete({
            device_info: JSON.stringify(deviceInfos[deviceType]),
            ip_address: "10.0.0.50", // Mobile network IP
            ...overrides,
        });
    }

    /**
     * Create desktop session
     */
    async createDesktop(
        browser: "Chrome" | "Firefox" | "Safari" = "Chrome",
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        const browserInfos = {
            Chrome: {
                userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
                vendor: "Google Inc.",
                browser: "Chrome",
                version: "120.0.0.0",
            },
            Firefox: {
                userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
                vendor: "",
                browser: "Firefox",
                version: "120.0",
            },
            Safari: {
                userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
                vendor: "Apple Computer, Inc.",
                browser: "Safari",
                version: "17.1",
            },
        };

        return this.createComplete({
            device_info: JSON.stringify({
                ...browserInfos[browser],
                platform: "Win32",
                language: "en-US",
                screenResolution: "1920x1080",
                timezone: "America/New_York",
                cookieEnabled: true,
                doNotTrack: false,
                hardwareConcurrency: 8,
                memory: 8192,
            }),
            ...overrides,
        });
    }

    /**
     * Create session near expiry
     */
    async createNearExpiry(
        hoursUntilExpiry = 2,
        overrides?: Partial<Prisma.sessionCreateInput>,
    ): Promise<session> {
        return this.createMinimal({
            expires_at: new Date(Date.now() + hoursUntilExpiry * 60 * 60 * 1000),
            last_refresh_at: new Date(Date.now() - 29 * 24 * 60 * 60 * 1000), // Old refresh
            ...overrides,
        });
    }

    protected getDefaultInclude(): Prisma.sessionInclude {
        return {
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                    status: true,
                },
            },
            auth: {
                select: {
                    id: true,
                    provider: true,
                    last_used_at: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.sessionCreateInput,
        config: SessionRelationConfig,
        tx: any,
    ): Promise<Prisma.sessionCreateInput> {
        const data = { ...baseData };

        // Handle user connection (required)
        if (config.withUser || !data.user) {
            const user = await tx.user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Session Test User",
                    handle: `session_user_${nanoid(8)}`,
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            });
            data.user = { connect: { id: user.id } };
        }

        // Handle auth connection (required)
        if (config.withAuth || !data.auth) {
            const userId = data.user?.connect?.id || generatePK();
            const auth = await tx.user_auth.create({
                data: {
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                    user: { connect: { id: userId } },
                },
            });
            data.auth = { connect: { id: auth.id } };
        }

        // Handle expired state
        if (config.isExpired) {
            data.expires_at = new Date(Date.now() - 24 * 60 * 60 * 1000); // Expired 1 day ago
            data.last_refresh_at = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000);
        }

        // Handle revoked state
        if (config.isRevoked) {
            data.revokedAt = new Date(Date.now() - 60 * 60 * 1000); // Revoked 1 hour ago
        }

        // Handle device type
        if (config.deviceType) {
            const deviceConfigs = {
                desktop: {
                    userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                    isMobile: false,
                    platform: "Win32",
                },
                mobile: {
                    userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
                    isMobile: true,
                    platform: "iPhone",
                },
                tablet: {
                    userAgent: "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
                    isMobile: true,
                    isTablet: true,
                    platform: "iPad",
                },
            };
            data.device_info = JSON.stringify(deviceConfigs[config.deviceType]);
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createMultipleForUser(userId: bigint, authId: bigint, count = 3): Promise<session[]> {
        const sessions = await Promise.all(
            Array.from({ length: count }, (_, i) => 
                this.createActive(30 - i, {
                    user: { connect: { id: userId } },
                    auth: { connect: { id: authId } },
                    device_info: `Session ${i + 1} - Device`,
                    ip_address: `192.168.1.${100 + i}`,
                    last_refresh_at: new Date(Date.now() - i * 60 * 60 * 1000),
                }),
            ),
        );

        return sessions;
    }

    async createSessionHistory(): Promise<session[]> {
        // Create a user with auth first
        const user = await this.prisma.user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Session History User",
                handle: `history_user_${nanoid(8)}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });

        const auth = await this.prisma.user_auth.create({
            data: {
                id: generatePK(),
                provider: "Password",
                hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                user: { connect: { id: user.id } },
            },
        });

        // Create various session states
        const sessions = await Promise.all([
            this.createActive(30, { 
                user: { connect: { id: user.id } },
                auth: { connect: { id: auth.id } },
            }),
            this.createActive(15, { 
                user: { connect: { id: user.id } },
                auth: { connect: { id: auth.id } },
            }),
            this.createNearExpiry(12, { 
                user: { connect: { id: user.id } },
                auth: { connect: { id: auth.id } },
            }),
            this.createExpired(7, { 
                user: { connect: { id: user.id } },
                auth: { connect: { id: auth.id } },
            }),
            this.createRevoked(24, { 
                user: { connect: { id: user.id } },
                auth: { connect: { id: auth.id } },
            }),
        ]);

        return sessions;
    }

    async createGeographicallyDistributed(): Promise<session[]> {
        const locations = [
            { ip: "203.0.113.42", location: "US-West", timezone: "America/Los_Angeles" },
            { ip: "198.51.100.15", location: "US-East", timezone: "America/New_York" },
            { ip: "192.0.2.88", location: "EU-West", timezone: "Europe/London" },
            { ip: "172.16.0.25", location: "Asia-Pacific", timezone: "Asia/Tokyo" },
        ];

        return Promise.all(
            locations.map(loc => 
                this.createComplete({
                    ip_address: loc.ip,
                    device_info: JSON.stringify({
                        userAgent: "Mozilla/5.0 (compatible; TestBrowser/1.0)",
                        location: loc.location,
                        timezone: loc.timezone,
                    }),
                }),
            ),
        );
    }

    protected async checkModelConstraints(record: session): Promise<string[]> {
        const violations: string[] = [];
        
        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.user_id },
        });
        if (!user) {
            violations.push("Associated user not found");
        }

        // Check auth exists
        const auth = await this.prisma.user_auth.findUnique({
            where: { id: record.auth_id },
        });
        if (!auth) {
            violations.push("Associated auth not found");
        }

        // Check auth belongs to user
        if (auth && user && auth.user_id !== user.id) {
            violations.push("Auth does not belong to the session user");
        }

        // Check IP address format
        if (record.ip_address) {
            // Simple IPv4/IPv6 validation
            const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
            const ipv6Regex = /^([\da-f]{1,4}:){7}[\da-f]{1,4}$/i;
            if (!ipv4Regex.test(record.ip_address) && !ipv6Regex.test(record.ip_address)) {
                violations.push("Invalid IP address format");
            }
        }

        // Check device info is valid JSON if present
        if (record.device_info) {
            try {
                JSON.parse(record.device_info);
            } catch {
                violations.push("Device info is not valid JSON");
            }
        }

        // Check expiry vs revoked logic
        if (record.revokedAt && record.expires_at < record.revokedAt) {
            violations.push("Session was revoked after it expired");
        }

        // Check refresh timing
        if (record.last_refresh_at > new Date()) {
            violations.push("Last refresh is in the future");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, expires_at, user, auth
                ip_address: "192.168.1.1",
                device_info: "Browser info",
            },
            invalidTypes: {
                id: "not-a-snowflake",
                expires_at: "not-a-date", // Should be Date
                user_id: "not-a-bigint", // Should be BigInt
                auth_id: "not-a-bigint", // Should be BigInt
                revokedAt: "not-a-date", // Should be Date
            },
            invalidIpAddress: {
                id: generatePK(),
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                ip_address: "999.999.999.999", // Invalid IP
                user: { connect: { id: generatePK() } },
                auth: { connect: { id: generatePK() } },
            },
            invalidDeviceInfo: {
                id: generatePK(),
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                device_info: "not-json-{invalid}", // Invalid JSON
                user: { connect: { id: generatePK() } },
                auth: { connect: { id: generatePK() } },
            },
            futureRefresh: {
                id: generatePK(),
                expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                last_refresh_at: new Date(Date.now() + 60 * 60 * 1000), // Future
                user: { connect: { id: generatePK() } },
                auth: { connect: { id: generatePK() } },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.sessionCreateInput> {
        return {
            maxLengthDeviceInfo: {
                ...this.getCompleteData(),
                device_info: JSON.stringify({
                    userAgent: "A".repeat(500),
                    extraData: "B".repeat(400),
                }), // Near 1024 char limit
            },
            ipv6Address: {
                ...this.getMinimalData(),
                ip_address: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
            },
            veryShortSession: {
                ...this.getMinimalData(),
                expires_at: new Date(Date.now() + 60 * 1000), // 1 minute
            },
            veryLongSession: {
                ...this.getMinimalData(),
                expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
            },
            frequentlyRefreshed: {
                ...this.getCompleteData(),
                last_refresh_at: new Date(Date.now() - 1000), // Refreshed 1 second ago
            },
            revokedImmediately: {
                ...this.getCompleteData(),
                revokedAt: new Date(Date.now() - 1000), // Revoked 1 second after creation
            },
            complexDeviceInfo: {
                ...this.getCompleteData(),
                device_info: JSON.stringify({
                    userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
                    platform: "Linux x86_64",
                    vendor: "",
                    language: ["en-US", "en", "es"],
                    screenResolution: "3840x2160",
                    timezone: "UTC",
                    cookieEnabled: true,
                    doNotTrack: true,
                    hardwareConcurrency: 16,
                    memory: 32768,
                    maxTouchPoints: 0,
                    webdriver: false,
                    bluetooth: true,
                    clipboard: true,
                    credentials: true,
                    keyboard: true,
                    mediaDevices: true,
                    serviceWorker: true,
                    storageEstimate: 10737418240,
                    gpu: {
                        vendor: "NVIDIA Corporation",
                        renderer: "NVIDIA GeForce RTX 3080",
                    },
                }),
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            user: true,
            auth: true,
        };
    }

    protected async deleteRelatedRecords(
        record: session,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Sessions don't have dependent records
        // The user/auth relationships are handled by the parent
    }
}

// Export factory creator function
export const createSessionDbFactory = (prisma: PrismaClient) => new SessionDbFactory(prisma);
