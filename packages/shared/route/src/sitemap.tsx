/**
 * Custom sitemap generator based on react-dynamic-sitemap. 
 * We use this because react-dynamic-sitemap only supports react-router, but we use custom routing.
 */
import builder from "xmlbuilder";

export const generateSitemap = (siteName: string, Routes: (props: any) => JSX.Element, ...props: any): string => {
    let paths = Routes(props).props.children;
    console.log('sitemap paths', paths);
    let xml = builder.create("urlset", { encoding: "utf-8" }).att("xmlns", "http://www.sitemaps.org/schemas/sitemap/0.9");
    paths.forEach(function (path: any) {
        const slugs = path.props.slugs || [{}];
        slugs.forEach((slug: any) => {
            let uri = path.props.path;
            Object.keys(slug).forEach(key => {
                const value = slug[key];
                let midStringRegex = new RegExp(`/:${key}/`, "g");
                let endStringRegex = new RegExp(`/:${key}$`);
                if (uri.match(midStringRegex))
                    uri = uri.replace(midStringRegex, `/${value}/`);
                else
                    uri = uri.replace(endStringRegex, `/${value}`);
            });
            if (path.props.sitemapIndex) {
                var item = xml.ele("url");
                item.ele("loc", siteName + uri);
                item.ele("priority", path.props.priority || 0);
                item.ele("changefreq", path.props.changefreq || "never");
            }
        });
    });
    return xml.end({ pretty: true });
};