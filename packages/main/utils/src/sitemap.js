import builder from "xmlbuilder2";
export const generateSitemap = (siteName, entries) => {
    let xml = builder.create({ encoding: "UTF-8", version: "1.0" });
    xml = xml.ele("urlset", { xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9" });
    if (entries.main) {
        entries.main.forEach(function (entry) {
            xml.ele("url", {
                loc: siteName + entry.path,
                priority: (entry.priority || 0) + "",
                changefreq: entry.changeFreq || "never",
            }).up();
        });
    }
    if (entries.content) {
        entries.content.forEach(function (entry) {
            const link = `${siteName}${entry.objectLink}/${entry.rootHandle ?? entry.rootId ?? ""}${entry.handle ?? entry.id}`;
            xml.ele("url")
                .ele("loc", { hreflang: "x-default", href: link }).up();
            entry.languages.forEach(function (language) {
                xml.ele("link", { rel: "alternate", hreflang: language, href: `${link}?lang=${language}` }).up();
            });
            xml.up();
        });
    }
    xml = xml.up();
    return xml.end({ prettyPrint: true });
};
export const generateSitemapIndex = (sitemapDir, fileNames) => {
    let xml = builder.create({ encoding: "UTF-8", version: "1.0" });
    xml = xml.ele("sitemapindex", { xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9" });
    fileNames.forEach(function (fileName) {
        xml.ele("sitemap")
            .ele("loc").txt(`${sitemapDir}/${fileName}`).up()
            .up();
    });
    xml = xml.up();
    return xml.end({ prettyPrint: true });
};
//# sourceMappingURL=sitemap.js.map