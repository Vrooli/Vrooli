/**
 * Generates sitemap.xml using Routes.tsx
 */
import { generateSitemap } from '@shared/route/src/sitemap';
import fs from 'fs';

// Read packages/shared/consts/src/ui.ts to get all routes
fs.readFile(new URL('../../shared/consts/src/ui.ts', import.meta.url), 'utf8', (err, data) => {
    if (err) {
        console.error(err);
        return;
    }
    console.log(data);
};

// Read Routes.tsx as a string, since loading it as a module from ts-node 
// is a massive pain
fs.readFile(new URL('Routes.tsx', import.meta.url), 'utf8', (err, data) => {
    if (err) {
        console.error(err);
        return;
    }
    console.log(data);
    // Find every <Route> opening tag, which may span multiple lines
    let routes: string[] = data.match(/<Route(.|\n)*?>/g) ?? [];
    console.log('a', routes);
    // Filter out routes which don't contain sitemapIndex\n or sitemapIndex="true" or sitemapIndex={true}
    routes = routes.filter((route) => {
        return route.includes('sitemapIndex\n') || route.includes('sitemapIndex="true"') || route.includes('sitemapIndex={true}');
    });
    console.log('b', routes);
    // For the remaining routes, extract the path, priority, and changeFreq
    const sitemapData = routes.map((route) => {
        const path = route.match(/path="(.*)"/)?.[1]; //TODO map LINKS to actual links
        const priority = route.match(/priority="(.*)"/)?.[1];
        const changeFreq = route.match(/changeFreq="(.*)"/)?.[1];
        console.log('c', path, priority, changeFreq);
        return { path, priority, changeFreq };
    }).filter((route) => route.path);
    console.log('d', sitemapData);
});



// //Test
// console.log('generating sitemap.xml...');
// const sitemap: string = generateSitemap('https://vrooli.com', Routes, {}, true);
// console.log('sitemap.xml generated!', sitemap);