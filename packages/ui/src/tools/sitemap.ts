/**
 * Converts data in Routes.tsx to a sitemap.xml file. This ensures that sitemap.xml is always up to date. 
 * Doing this in a script during the build process - as opposed to options like react-dynamic-sitemap - 
 * allows us to view the generated sitemap in the public folder to check that it's correct.
 */
import { generateSitemap, SitemapEntryMain } from '@shared/utils';
import fs from 'fs';

/**
 * Reads and parses file that contains route names
 * @returns Map of route names to route paths
 */
const getRouteMap = async (): Promise<{ [x: string]: string }> => {
    const routeMapLocation = new URL('../../../shared/consts/src/ui.ts', import.meta.url);
    const routeMapName = 'APP_LINKS';
    console.info(`Reading ${routeMapName} from ${routeMapLocation}...`);
    try {
        const data = await fs.promises.readFile(routeMapLocation, 'utf8');
        // Find route map object
        const linksStringRegex = new RegExp(`export const ${routeMapName} = {(.|\n)*?}`, 'g');
        const linksString = data.match(linksStringRegex);
        if (!linksString) {
            console.error(`Could not find ${routeMapName} in ${routeMapLocation}`);
            return {};
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
        const routeMap = Object.fromEntries(keyValues);
        console.info(`Found ${Object.keys(routeMap).length} routes in ${routeMapLocation}`);
        return routeMap;
    } catch (error) {
        console.error(error);
        return {};
    }
}

/**
 * Reads and parses file containing routes. Finds sitemap data and generates a sitemap.xml file
 */
const main = async () => {
    // Get route names
    const routeMap = await getRouteMap();

    const routesLocation = new URL('../Routes.tsx', import.meta.url);
    const componentName = 'NavRoute'; // Name can change if using a wrapper component. This is typically "Route"
    console.info(`Checking for ${componentName} in ${routesLocation}...`)
    try {
        // Read Routes.tsx as a string, since loading it as a module from ts-node 
        // is a massive pain
        const data = await fs.promises.readFile(routesLocation, 'utf8');
        // Find every route component opening tag which may span multiple lines
        let routes: string[] = data.match(new RegExp(`<${componentName}(.|\\n)*?>`, 'g')) ?? [];
        console.info(`Found ${routes.length} ${componentName}s in ${routesLocation} before filtering`);
        // Filter out routes which don't contain sitemapIndex\n or sitemapIndex="true" or sitemapIndex={true}
        routes = routes.filter((route) => {
            return route.includes('sitemapIndex\n') || route.includes('sitemapIndex="true"') || route.includes('sitemapIndex={true}');
        });
        console.info(`Found ${routes.length} ${componentName}s in ${routesLocation} after filtering`);
        // For the remaining routes, extract the path, priority, and changeFreq
        const entries: SitemapEntryMain[] = routes.map((route) => {
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
        }).filter((route) => route.path) as SitemapEntryMain[];
        console.info('Parsed sitemap data from routes: ', entries);
        // Generate sitemap.xml
        const sitemap = generateSitemap('https://app.vrooli.com', { main: entries });
        console.info('Generated sitemap: ', sitemap)
        // Check if sitemap save directory exists
        const sitemapDir = new URL('../../public/sitemaps', import.meta.url);
        if (!fs.existsSync(sitemapDir)) {
            console.log('Creating sitemap directory', sitemapDir.pathname);
            fs.mkdirSync(sitemapDir);
        }
        // Write sitemap-routes.xml to sitemap directory
        const sitemapLocation = `${sitemapDir.pathname}/sitemap-routes.xml`;
        fs.writeFile(sitemapLocation, sitemap, (err) => {
            if (err) console.error(err);
            else console.log('Sitemap generated at ' + sitemapLocation);
        });
    } catch (error) {
        console.error(error);
    }
}

main();