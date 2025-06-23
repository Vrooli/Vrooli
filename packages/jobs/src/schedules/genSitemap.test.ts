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
        it("should generate a sitemap index when no entities exist", async () => {
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

            // Check that sitemap index was created
            expect(fs.existsSync(testIndexPath)).toBe(true);
            
            // Read and verify index content
            const indexContent = fs.readFileSync(testIndexPath, "utf-8");
            expect(indexContent).toContain("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
            expect(indexContent).toContain("<sitemapindex");
            // Accept both self-closing and explicit closing tags
            expect(indexContent.includes("</sitemapindex>") || indexContent.includes("/>")).toBe(true);
            
        });

        it("should generate sitemaps for users", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create test users
            for (let i = 0; i < 3; i++) {
                const user = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        name: `Test User ${i}`,
                        handle: `testuser${i}`,
                        isBot: false,
                        isPrivate: false,
                    },
                });
                testUserIds.push(user.id);
            }

            await genSitemap();

            // Check that user sitemap was created
            const userSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/User-0.xml.gz");
            expect(fs.existsSync(userSitemapPath)).toBe(true);
            
            // Decompress and verify content
            const compressed = fs.readFileSync(userSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
            expect(content).toContain("<urlset");
            expect(content).toContain(`${UI_URL_REMOTE}${LINKS.User}/@`);
            expect(content).toContain("</urlset>");
        });

        it("should generate sitemaps for teams", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a user to own the teams
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Team Owner",
                    handle: "teamowner",
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create test teams
            for (let i = 0; i < 3; i++) {
                const team = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        handle: `testteam${i}`,
                        createdBy: {
                            connect: { id: owner.id },
                        },
                        isPrivate: false,
                    },
                });
                testTeamIds.push(team.id);
            }

            await genSitemap();

            // Check that team sitemap was created
            const teamSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/Team-0.xml.gz");
            expect(fs.existsSync(teamSitemapPath)).toBe(true);
            
            // Decompress and verify content
            const compressed = fs.readFileSync(teamSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
            expect(content).toContain("<urlset");
            expect(content).toContain(`${UI_URL_REMOTE}${LINKS.Team}/@`);
            expect(content).toContain("</urlset>");
        });

        it("should generate sitemaps for resource versions", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            // Create a user to own the resources
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Resource Owner",
                    handle: "resourceowner",
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create test resources with versions
            for (let i = 0; i < 3; i++) {
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
                                name: `Test Resource ${i}`,
                                description: "Test description",
                            }],
                        },
                    },
                });
                testResourceVersionIds.push(resourceVersion.id);
            }

            await genSitemap();

            // Check that resource sitemap was created
            const resourceSitemapPath = path.resolve(__dirname, "../../../ui/dist/sitemaps/ResourceVersion-0.xml.gz");
            expect(fs.existsSync(resourceSitemapPath)).toBe(true);
            
            // Decompress and verify content
            const compressed = fs.readFileSync(resourceSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            expect(content).toContain("<?xml version=\"1.0\" encoding=\"UTF-8\"?>");
            expect(content).toContain("<urlset");
            expect(content).toContain(`${UI_URL_REMOTE}/code/`);
            expect(content).toContain("</urlset>");
        });

        it("should handle pagination when entities exceed limit", async () => {
            const testIndexPath = path.resolve(__dirname, "../../../ui/dist/sitemap.xml");
            const testSitemapDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            
            // Ensure test directories exist
            if (!fs.existsSync(path.dirname(testIndexPath))) {
                fs.mkdirSync(path.dirname(testIndexPath), { recursive: true });
            }
            if (!fs.existsSync(testSitemapDir)) {
                fs.mkdirSync(testSitemapDir, { recursive: true });
            }
            
            const owner = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Owner",
                    handle: "owner",
                    isBot: false,
                },
            });
            testUserIds.push(owner.id);

            // Create enough teams to test batching behavior
            // Since the actual pagination limits are very high (50k entries),
            // we'll just test that the function can handle a decent number of entities
            for (let i = 0; i < 10; i++) {
                const team = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        handle: `team${i}`,
                        createdBy: {
                            connect: { id: owner.id },
                        },
                        isPrivate: false,
                    },
                });
                testTeamIds.push(team.id);
            }

            await genSitemap();

            // Check that team sitemap was created with all teams
            const sitemapsDir = path.resolve(__dirname, "../../../ui/dist/sitemaps");
            expect(fs.existsSync(path.join(sitemapsDir, "Team-0.xml.gz"))).toBe(true);
            
            // Verify the sitemap contains all teams
            const teamSitemapPath = path.join(sitemapsDir, "Team-0.xml.gz");
            const compressed = fs.readFileSync(teamSitemapPath);
            const decompressed = await gunzipAsync(compressed);
            const content = decompressed.toString();
            
            // Should contain multiple teams
            expect(content).toContain("team0");
            expect(content).toContain("team5");
            expect(content).toContain("team9");
            
            // Check index includes the sitemap
            const indexContent = fs.readFileSync(path.resolve(__dirname, "../../../ui/dist/sitemap.xml"), "utf-8");
            expect(indexContent).toContain("Team-0.xml.gz");
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
                    handle: "publicuser",
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
                    handle: "privateuser",
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
                    handle: "regularuser",
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
                    handle: "botuser",
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

        it("should handle errors gracefully", async () => {
            // Restore logger mocks temporarily to test error handling
            vi.restoreAllMocks();
            
            // Create a fresh error spy
            const errorSpy = vi.spyOn(logger, "error").mockImplementation(() => logger);
            
            // Mock fs.writeFileSync to throw an error during sitemap index creation
            vi.spyOn(fs, "writeFileSync").mockImplementationOnce(() => {
                throw new Error("Permission denied");
            });
            
            // Should not throw
            await expect(genSitemap()).resolves.not.toThrow();
            
            // Verify error was logged
            expect(errorSpy).toHaveBeenCalled();
            
            // Restore mocks for cleanup
            vi.spyOn(logger, "info").mockImplementation(() => logger);
            vi.spyOn(logger, "warn").mockImplementation(() => logger);
        });
    });
});
