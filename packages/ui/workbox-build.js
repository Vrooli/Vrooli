import fs from "fs";
import { promisify } from "util";
import { injectManifest } from "workbox-build";

const swSrc = "./src/sw-template.js";
const swDest = "./dist/service-worker.js";

const copyFile = promisify(fs.copyFile);

async function buildServiceWorker() {
    // Copy the service worker template to the dist folder
    await copyFile(swSrc, swDest);

    try {
        const { count, size, warnings } = await injectManifest({
            swSrc: swDest, // Use the copied service worker in the dist folder
            swDest,
            globDirectory: "./dist",
            globPatterns: ["**/*.{js,css,html,png,jpg,svg,json,md}"],
            modifyURLPrefix: {
                "": "./",
            },
        });

        warnings.forEach(console.warn);
        console.log(
            `Generated ${swDest}, which will precache ${count} files, totaling ${size} bytes.`,
        );
    } catch (error) {
        console.error(`Error generating service worker: ${error}`);
    }
}

buildServiceWorker();
