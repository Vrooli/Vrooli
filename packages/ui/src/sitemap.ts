/**
 * Converts data in Routes.tsx to a sitemap.xml file. This ensures that sitemap.xml is always up to date. 
 * Doing this in a script during the build process - as opposed to options like react-dynamic-sitemap - 
 * allows us to view the generated sitemap in the public folder to check that it's correct.
 */
import { generateSitemap, SitemapEntry } from '@shared/route/src/sitemap';
import fs from 'fs';

// Read packages/shared/consts/src/ui.ts to get all routes
const routeMapLocation = '../../shared/consts/src/ui.ts';
const routeMapName = 'APP_LINKS';
let routeMap: { [key: string]: string } = {};
fs.readFile(new URL(routeMapLocation, import.meta.url), 'utf8', (err, data) => {
    if (err) {
        console.error(err);
        return;
    }
    console.log(data);
    // Find route map object
    const linksStringRegex = new RegExp(`export const ${routeMapName} = {(.|\n)*?}`, 'g');
    const linksString = data.match(linksStringRegex);
    console.log(linksString);
    if (!linksString) {
        console.error(`Could not find ${routeMapName} in ${routeMapLocation}`);
        return;
    }
    // Convert matched string to object
    // This can be accomplished by:
    // 1. Splitting the string into lines
    // 2. Removing every line without ': ' in it
    // 3. Splitting each remaining line into key and value
    // 4. Strip from the value anything that's not inside the single or double quotes
    // 1.
    let lines = linksString[0].split('\n');
    // 2.
    lines = lines.filter((line) => line.includes(': '));
    // 3.
    let keyValues = lines.map((line) => line.split(': '));
    // 4.
    keyValues = keyValues.map((keyValue) => {
        const key: string = keyValue[0].trim();
        let value: string = (keyValue[1] as any).match(/['"].*['"]/g)[0];
        value = value.replaceAll("'", '').replaceAll('"', '');
        return [key, value];
    });
    // Convert to object
    routeMap = Object.fromEntries(keyValues);
});

// Read Routes.tsx as a string, since loading it as a module from ts-node 
// is a massive pain
const routesLocation = 'Routes.tsx';
fs.readFile(new URL(routesLocation, import.meta.url), 'utf8', (err, data) => {
    if (err) {
        console.error(err);
        return;
    }
    // Find every <Route> opening tag, which may span multiple lines
    let routes: string[] = data.match(/<Route(.|\n)*?>/g) ?? [];
    // Filter out routes which don't contain sitemapIndex\n or sitemapIndex="true" or sitemapIndex={true}
    routes = routes.filter((route) => {
        return route.includes('sitemapIndex\n') || route.includes('sitemapIndex="true"') || route.includes('sitemapIndex={true}');
    });
    // For the remaining routes, extract the path, priority, and changeFreq
    const sitemapData: SitemapEntry[] = routes.map((route) => {
        // Match path (e.g. path="/about" => /about, path={"/about"} => /about, path={APP_LINKS.ABOUT} => /about)
        // May need to use route map object to convert to path
        let path = route.match(/path=".*?"/g)?.[0] ?? route.match(/path={.*?}/g)?.[0];
        if (path) {
            path = path.replaceAll('path=', '').replaceAll('"', '').replaceAll('{', '').replaceAll('}', '');
            // If path is a key in the route map object (i.e. contains a '.' and after '.' is a key in the route map), 
            // replace it with the corresponding value
            if (path.includes('.') && path.split('.')[1] in routeMap) {
                path = routeMap[path.split('.')[1]];
            }
        }
        // Match priority (e.g. priority={0.5} => 0.5)
        let priority = route.match(/priority={.*?}/g)?.[0];
        if (priority) {
            priority = priority.replaceAll('priority={', '').replaceAll('}', '');
        }
        // Match changeFreq (e.g. changeFreq="daily" => daily, changeFreq={"daily"} => daily)
        let changeFreq = route.match(/changeFreq=".*?"/g)?.[0] ?? route.match(/changeFreq={.*?}/g)?.[0];
        if (changeFreq) {
            changeFreq = changeFreq.replaceAll('changeFreq=', '').replaceAll('"', '').replaceAll('{', '').replaceAll('}', '');
        }
        return { path, priority, changeFreq };
    }).filter((route) => route.path) as SitemapEntry[];
    // Generate sitemap.xml
    const sitemap = generateSitemap('https://app.vrooli.com', sitemapData);
    // Write sitemap.xml to public folder
    const sitemapLocation = new URL('../public/sitemap.xml', import.meta.url);
    fs.writeFile(sitemapLocation, sitemap, (err) => {
        if (err) {
            console.error(err);
            return;
        }
        console.log('Sitemap generated at ' + sitemapLocation);
    });
});