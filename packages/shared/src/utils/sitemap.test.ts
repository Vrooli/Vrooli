import { describe, it, expect } from "vitest";
import { generateSitemap, generateSitemapIndex, type SitemapEntryContent, type SitemapEntryMain } from "./sitemap.js";

describe("generateSitemap", () => {
    it("generates XML for main entries with correct loc, changefreq, and priority", () => {
        const mainEntries: SitemapEntryMain[] = [
            { path: "/about", priority: 0.8, changeFreq: "daily" },
        ];
        const xml = generateSitemap("https://example.com", { main: mainEntries });
        expect(xml).toContain("<loc>https://example.com/about</loc>");
        expect(xml).toContain("<changefreq>daily</changefreq>");
        expect(xml).toContain("<priority>0.8</priority>");
    });

    it("generates XML for content entries using versionLabel when provided", () => {
        // Setup a versioned entry with rootPublicId and versionLabel
        const contentEntry: SitemapEntryContent = {
            publicId: "123",
            versionLabel: "v2",
            objectLink: "/resource",
            languages: ["en", "de"],
            rootPublicId: "root123",
        };
        const xml = generateSitemap("https://example.com", { content: [contentEntry] });
        // Verify <loc> uses rootPublicId followed by versionLabel
        expect(xml).toContain("<loc>https://example.com/resource/root123/v2</loc>");
        // Verify alternate links (x-default and specified languages)
        expect(xml).toContain("rel=\"alternate\" hreflang=\"x-default\"");
        expect(xml).toContain("hreflang=\"en\" href=\"https://example.com/resource/root123/v2?lang=en\"");
        expect(xml).toContain("hreflang=\"de\" href=\"https://example.com/resource/root123/v2?lang=de\"");
    });

    it("falls back to handle when no versionLabel provided", () => {
        const contentEntry: SitemapEntryContent = {
            handle: "john",
            publicId: "123",
            objectLink: "/user",
            languages: ["en"],
        };
        const xml = generateSitemap("https://example.com", { content: [contentEntry] });
        expect(xml).toContain("<loc>https://example.com/user/@john</loc>");
    });

    it("falls back to publicId when neither versionLabel nor handle provided", () => {
        const contentEntry: SitemapEntryContent = {
            publicId: "456",
            objectLink: "/item",
            languages: ["en"],
        };
        const xml = generateSitemap("https://example.com", { content: [contentEntry] });
        expect(xml).toContain("<loc>https://example.com/item/456</loc>");
    });

    it("outputs a full sitemap combining main and content entries for manual inspection", () => {
        // Main entry
        const mainEntries: SitemapEntryMain[] = [
            { path: "/home", priority: 1, changeFreq: "always" },
        ];
        // Content entries including a versioned resource and a user handle
        const contentEntries: SitemapEntryContent[] = [
            {
                publicId: "1",
                versionLabel: "v1",
                objectLink: "/resource",
                languages: ["en", "fr"],
                rootPublicId: "0",
            },
            {
                handle: "bob",
                publicId: "2",
                objectLink: "/user",
                languages: ["en"],
            },
        ];
        const xml = generateSitemap("https://example.com", { main: mainEntries, content: contentEntries });
        // Print full XML for visual verification
        console.log(xml);
        // Basic structural and content assertions
        expect(xml.startsWith("<?xml")).toBe(true);
        expect(xml).toContain("<urlset");
        expect(xml).toContain("<loc>https://example.com/home</loc>");
        expect(xml).toContain("<loc>https://example.com/resource/0/v1</loc>");
        expect(xml).toContain("<loc>https://example.com/user/@bob</loc>");
        expect(xml.endsWith("</urlset>")).toBe(true);
    });
});

describe("generateSitemapIndex", () => {
    it("generates XML index for provided sitemap files", () => {
        const xml = generateSitemapIndex("/sitemaps", ["file1.xml", "file2.xml"]);
        expect(xml).toContain("<loc>/sitemaps/file1.xml</loc>");
        expect(xml).toContain("<loc>/sitemaps/file2.xml</loc>");
    });
}); 
