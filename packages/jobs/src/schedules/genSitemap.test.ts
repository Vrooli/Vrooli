// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { generatePK, generatePublicId, LINKS } from "@vrooli/shared";
import fs from "fs";
import path from "path";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import zlib from "zlib";
import { promisify } from "util";
import { genSitemap, isSitemapMissing } from "./genSitemap.js";
import { DbProvider, logger, UI_URL_REMOTE } from "@vrooli/server";

const gzipAsync = promisify(zlib.gzip);
const gunzipAsync = promisify(zlib.gunzip);

describe("genSitemap integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testResourceVersionIds: bigint[] = [];
    const testLang = "en";

    beforeAll(() => {
        // Suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testResourceIds.length = 0;
        testResourceVersionIds.length = 0;
    });

    afterEach(async () => {
        // Clean up generated files
        const sitemapIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
        const sitemapsDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
        
        // Clean up sitemap index
        if (fs.existsSync(sitemapIndexPath)) {
            fs.unlinkSync(sitemapIndexPath);
        }
        
        // Clean up sitemaps directory
        if (fs.existsSync(sitemapsDir)) {
            const files = fs.readdirSync(sitemapsDir);
            for (const file of files) {
                if (file.endsWith(".xml.gz")) {
                    fs.unlinkSync(path.join(sitemapsDir, file));
                }
            }
        }
        
        // Clean up test data from database
        const db = DbProvider.get();
        
        if (testResourceVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testResourceVersionIds } } });
        }
        if (testResourceIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testResourceIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    describe("isSitemapMissing", () => {
        // Get the expected sitemap path
        const sitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
        const sitemapDir = path.dirname(sitemapPath);

        it("should return true when sitemap.xml doesn't exist", async () => {
            // Clean up any existing file
            if (fs.existsSync(sitemapPath)) {
                fs.unlinkSync(sitemapPath);
            }
            expect(await isSitemapMissing()).toBe(true);
        });

        it("should return false when sitemap.xml exists", async () => {
            // Create directory if needed
            if (!fs.existsSync(sitemapDir)) {
                fs.mkdirSync(sitemapDir, { recursive: true });
            }
            fs.writeFileSync(sitemapPath, "test");
            expect(await isSitemapMissing()).toBe(false);
            // Clean up
            fs.unlinkSync(sitemapPath);
        });
    });

    describe("genSitemap", () => {
        it("should create sitemap index file", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            await genSitemap();

            // Verify sitemap index was created
            expect(fs.existsSync(testIndexPath)).toBe(true);
            
            // Verify it's a valid XML file (should be parseable)
            const indexContent = fs.readFileSync(testIndexPath, "utf-8");
            expect(() => {
                // Basic XML validation - should not throw
                if (!indexContent.includes("<?xml")) {
                    throw new Error("Not valid XML");
                }
            }).not.toThrow();
        });

        it("should include public users in sitemap", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a public user
            const publicUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Public User",
                    handle: `publicuser1_${Date.now()}`,
                    isBot: false,
                    isPrivate: false,
                },
            });
            testUserIds.push(publicUser.id);

            await genSitemap();

            // Verify user was included in sitemap
            const userSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/User-0.xml.gz");
            const compressed = fs.readFileSync(userSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain(publicUser.handle);
        });

        it("should create user sitemap with correct URL format", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a user
            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Test User",
                    handle: `testuser_${Date.now()}`,
                    isBot: false,
                    isPrivate: false,
                },
            });
            testUserIds.push(user.id);

            await genSitemap();

            // Verify URL format in sitemap
            const userSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/User-0.xml.gz");
            const compressed = fs.readFileSync(userSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain(`${UI_URL_REMOTE}${LINKS.User}/@${user.handle}`);
        });

        it("should include public teams in sitemap", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a user to own the team
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Team Owner",
                    handle: `teamowner_${Date.now()}`,
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create a public team
            const publicTeam = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: `publicteam_${Date.now()}`,
                    createdBy: {
                        connect: { id: owner.id },
                    },
                    isPrivate: false,
                },
            });
            testTeamIds.push(publicTeam.id);

            await genSitemap();

            // Verify team was included in sitemap
            const teamSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/Team-0.xml.gz");
            const compressed = fs.readFileSync(teamSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain(publicTeam.handle);
        });

        it("should include public resource versions in sitemap", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a user to own the resource
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Resource Owner",
                    handle: `resourceowner_${Date.now()}`,
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create a public resource with version
            const resource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    resourceType: "Code",
                    createdBy: {
                        connect: { id: owner.id },
                    },
                    isPrivate: false,
                    isDeleted: false,
                },
            });
            testResourceIds.push(resource.id);

            const resourceVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    rootId: resource.id,
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    isComplete: true,
                    isDeleted: false,
                    resourceSubType: "CodeDataConverter",
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: testLang,
                            name: "Test Resource",
                            description: "Test description",
                        }],
                    },
                },
            });
            testResourceVersionIds.push(resourceVersion.id);

            await genSitemap();

            // Verify resource was included in sitemap with version label
            const resourceSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/ResourceVersion-0.xml.gz");
            const compressed = fs.readFileSync(resourceSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain(resourceVersion.versionLabel);
        });

        it("should include all entities when multiple exist", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            const uniqueId = Date.now();
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Owner",
                    handle: `owner_${uniqueId}`,
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create multiple teams
            const teamHandles = [];
            for (let i = 0; i < 5; i++) {
                const team = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        handle: `team${i}_${Date.now()}`,
                        createdBy: {
                            connect: { id: owner.id },
                        },
                        isPrivate: false,
                    },
                });
                testTeamIds.push(team.id);
                teamHandles.push(team.handle);
            }

            await genSitemap();

            // Verify all teams are included in the sitemap
            const teamSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/Team-0.xml.gz");
            const compressed = fs.readFileSync(teamSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            // Should contain all created teams
            teamHandles.forEach(handle => {
                expect(content).toContain(handle);
            });
        });

        it("should exclude private entities", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create both public and private users
            const publicUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Public User",
                    handle: `publicuser_${Date.now()}`,
                    isBot: false,
                    isPrivate: false,
                },
            });
            testUserIds.push(publicUser.id);

            const privateUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Private User",
                    handle: `privateuser_${Date.now()}`,
                    isBot: false,
                    isPrivate: true,
                },
            });
            testUserIds.push(privateUser.id);

            await genSitemap();

            // Check sitemap content
            const userSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/User-0.xml.gz");
            const compressed = fs.readFileSync(userSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            // Should contain public user
            expect(content).toContain(publicUser.handle);
            // Should not contain private user
            expect(content).not.toContain(privateUser.handle);
        });

        it("should exclude bots from user sitemap", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create both regular user and bot
            const regularUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Regular User",
                    handle: `regularuser_${Date.now()}`,
                    isBot: false,
                    isPrivate: false,
                },
            });
            testUserIds.push(regularUser.id);

            const botUser = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Bot User",
                    handle: `botuser_${Date.now()}`,
                    isBot: true,
                    isPrivate: false,
                },
            });
            testUserIds.push(botUser.id);

            await genSitemap();

            // Check sitemap content
            const userSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/User-0.xml.gz");
            const compressed = fs.readFileSync(userSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            // Should contain regular user
            expect(content).toContain(regularUser.handle);
            // Should not contain bot
            expect(content).not.toContain(botUser.handle);
        });

        it("should not throw when encountering file system errors", async () => {
            // Create a directory where the sitemap file should be to cause a write error
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testDir = path.dirname(testIndexPath);
            
            // Ensure directory exists
            if (!fs.existsSync(testDir)) {
                fs.mkdirSync(testDir, { recursive: true });
            }
            
            // Create a directory with the same name as the expected sitemap file
            // This will cause an error when trying to write the file
            if (fs.existsSync(testIndexPath)) {
                fs.unlinkSync(testIndexPath);
            }
            fs.mkdirSync(testIndexPath);
            
            // Function should complete without throwing
            await expect(genSitemap()).resolves.not.toThrow();
            
            // Clean up
            fs.rmdirSync(testIndexPath);
        });
    });
});
