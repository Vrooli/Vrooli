import { generateSitemap } from "@local/utils";
import fs from "fs";
const getRouteMap = async () => {
    const routeMapLocation = new URL("../../../shared/consts/src/ui.ts", import.meta.url);
    const routeMapName = "LINKS";
    console.info(`Reading ${routeMapName} from ${routeMapLocation}...`);
    try {
        const data = await fs.promises.readFile(routeMapLocation, "utf8");
        const linksStringRegex = new RegExp(`export const ${routeMapName} = {(.|\n)*?}`, "g");
        const linksString = data.match(linksStringRegex);
        if (!linksString) {
            console.error(`Could not find ${routeMapName} in ${routeMapLocation}`);
            return {};
        }
        let lines = linksString[0].split("\n");
        lines = lines.filter((line) => line.includes(": "));
        let keyValues = lines.map((line) => line.split(": "));
        keyValues = keyValues.map((keyValue) => {
            const key = keyValue[0].trim();
            let value = keyValue[1].match(/['"].*['"]/g)[0];
            value = value.replaceAll("'", "").replaceAll("\"", "");
            return [key, value];
        });
        const routeMap = Object.fromEntries(keyValues);
        console.info(`Found ${Object.keys(routeMap).length} routes in ${routeMapLocation}`);
        return routeMap;
    }
    catch (error) {
        console.error(error);
        return {};
    }
};
const main = async () => {
    const routeMap = await getRouteMap();
    const routesLocation = new URL("../Routes.tsx", import.meta.url);
    const componentName = "NavRoute";
    console.info(`Checking for ${componentName} in ${routesLocation}...`);
    try {
        const data = await fs.promises.readFile(routesLocation, "utf8");
        let routes = data.match(new RegExp(`<${componentName}(.|\\n)*?>`, "g")) ?? [];
        console.info(`Found ${routes.length} ${componentName}s in ${routesLocation} before filtering`);
        routes = routes.filter((route) => {
            return route.includes("sitemapIndex\n") || route.includes("sitemapIndex=\"true\"") || route.includes("sitemapIndex={true}");
        });
        console.info(`Found ${routes.length} ${componentName}s in ${routesLocation} after filtering`);
        const entries = routes.map((route) => {
            let path = route.match(/path=".*?"/g)?.[0] ?? route.match(/path={.*?}/g)?.[0];
            if (path) {
                path = path.replaceAll("path=", "").replaceAll("\"", "").replaceAll("{", "").replaceAll("}", "");
                if (path.includes(".") && path.split(".")[1] in routeMap) {
                    path = routeMap[path.split(".")[1]];
                }
            }
            let priority = route.match(/priority={.*?}/g)?.[0];
            if (priority) {
                priority = priority.replaceAll("priority={", "").replaceAll("}", "");
            }
            let changeFreq = route.match(/changeFreq=".*?"/g)?.[0] ?? route.match(/changeFreq={.*?}/g)?.[0];
            if (changeFreq) {
                changeFreq = changeFreq.replaceAll("changeFreq=", "").replaceAll("\"", "").replaceAll("{", "").replaceAll("}", "");
            }
            return { path, priority, changeFreq };
        }).filter((route) => route.path);
        console.info("Parsed sitemap data from routes: ", entries);
        const sitemap = generateSitemap("https://vrooli.com", { main: entries });
        console.info("Generated sitemap: ", sitemap);
        const sitemapDir = new URL("../../public/sitemaps", import.meta.url);
        if (!fs.existsSync(sitemapDir)) {
            console.info("Creating sitemap directory", sitemapDir.pathname);
            fs.mkdirSync(sitemapDir);
        }
        const sitemapLocation = `${sitemapDir.pathname}/sitemap-routes.xml`;
        fs.writeFile(sitemapLocation, sitemap, (err) => {
            if (err)
                console.error(err);
            else
                console.info("Sitemap generated at " + sitemapLocation);
        });
    }
    catch (error) {
        console.error(error);
    }
};
main();
//# sourceMappingURL=sitemap.js.map