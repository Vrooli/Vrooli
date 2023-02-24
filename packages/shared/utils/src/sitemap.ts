/**
 * Custom sitemap generator based on react-dynamic-sitemap. 
 * We use this because react-dynamic-sitemap only supports react-router, but we use custom routing.
 */
import builder from "xmlbuilder2";

/**
 * A sitemap entry for main route pages (e.g. /home, /search, /about, /contact)
 */
export type SitemapEntryMain = {
    changeFreq?: "always" | "hourly" | "daily" | "weekly" | "monthly" | "yearly" | "never";
    path?: string;
    priority?: number;
};

/**
 * A sitemap entry for user-generated pages (e.g. /routine/123, /api/123)
 */
export type SitemapEntryContent = {
    handle?: string | undefined; // Replaces id in path if present
    id: string; // Used with baseLink to generate the path
    languages: string[]; // Used to generate rel="alternate" hreflang="xx" links
    objectLink: string; // Used with id to generate the path
    rootHandle?: string | undefined; // If object is versioned, this is added between objectLink and id in the path
    rootId?: string | undefined; // If object is versioned, this is added between objectLink and id in the path
}

/**
 * Generates a sistemap file from the site name and a list of entries
 * @param siteName The name of the website
 * @param entries An object containing the sitemap entries
 * @returns An xml string representing the sitemap
 */
export const generateSitemap = (siteName: string, entries: {
    main?: SitemapEntryMain[];
    content?: SitemapEntryContent[];
}): string => {
    // Create xml tag with encoding and version
    let xml = builder.create({ encoding: "UTF-8", version: "1.0" })
    // Open urlset tag
    xml = xml.ele("urlset", { xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9" });
    // If main entries exist, add them
    if (entries.main) {
        // Loop through entries
        entries.main.forEach(function (entry: SitemapEntryMain) {
            // For each entry, create a url element with loc, priority, and changefreq
            xml.ele('url')
                .ele('loc', siteName + entry.path).up()
                .ele('priority', (entry.priority || 0) + '').up()
                .ele('changefreq', entry.changeFreq || "never").up()
                .up();
        });
    }
    // If content entries exist, add them
    if (entries.content) {
        // Loop through entries
        entries.content.forEach(function (entry: SitemapEntryContent) {
            const link =`${siteName}${entry.objectLink}/${entry.rootHandle ?? entry.rootId ?? ''}${entry.handle ?? entry.id}`;
            xml.ele('url')
                // Create loc link with x-default hreflang
                .ele("loc", { hreflang: "x-default", href: link }).up()
            // Create alternate links for each language
            entry.languages.forEach(function (language) {
                xml.ele("link", { rel: "alternate", hreflang: language, href: `${link}?lang=${language}` }).up();
            })
            xml.up();
        });
    }
    // Close urlset tag
    xml = xml.up();
    // Return xml string
    return xml.end({ prettyPrint: true });
};

/**
 * Generates a sitemap index file from a list of sitemap file names
 * @param sitemapDir The directory where the sitemap files are located
 * @param fileNames The list of sitemap file names
 * @returns An xml string representing the sitemap index
 */
export const generateSitemapIndex = (sitemapDir: string, fileNames: string[]): string => {
    // Create xml tag with encoding and version
    let xml = builder.create({ encoding: "UTF-8", version: "1.0" })
    // Open sitemapindex tag
    xml = xml.ele("sitemapindex", { xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9" });
    // Loop through file names
    fileNames.forEach(function (fileName) {
        // For each file name, create a sitemap element with loc
        xml.ele('sitemap')
            .ele('loc', `${sitemapDir}/${fileName}`).up()
            .up();
    });
    // Close sitemapindex tag
    xml = xml.up();
    // Return xml string
    return xml.end({ prettyPrint: true });
}