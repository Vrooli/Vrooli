/**
 * Custom sitemap generator based on react-dynamic-sitemap. 
 * We use this because react-dynamic-sitemap only supports react-router, but we use custom routing.
 */
import { RouteProps } from "./router";
import builder from "xmlbuilder";

export type SitemapEntry = Pick<RouteProps, 'changeFreq' | 'path' | 'priority'>;

export const generateSitemap = (siteName: string, entries: SitemapEntry[]): string => {
    console.log('sitemap entries', entries);
    let xml = builder.create("urlset", { encoding: "utf-8" }).att("xmlns", "http://www.sitemaps.org/schemas/sitemap/0.9");
    entries.forEach(function (entry: SitemapEntry) {
        var item = xml.ele("url");
        item.ele("loc", siteName + entry.path);
        item.ele("priority", entry.priority || 0);
        item.ele("changefreq", entry.changeFreq || "never");
    });
    return xml.end({ pretty: true });
};