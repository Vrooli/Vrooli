import { promises as fs } from "node:fs";
import { extname } from "node:path";
import { fileURLToPath } from "node:url";

function isMarkdownFile(path) {
    return extname(path) === ".md";
}

function shouldLoadManually(path) {
    return path.includes("node_modules/markdown-to-jsx/dist/index.modern.js");
}

export async function load(url, context, defaultLoad) {
    let path;
    try {
        // Only handle file:// URLs; otherwise, delegate to the default loader.
        if (!url.startsWith("file://")) {
            return defaultLoad(url, context, defaultLoad);
        }

        // Try to convert the URL to a path.
        path = fileURLToPath(url);

        // Handle special cases:
        // 1. Markdown files
        if (isMarkdownFile(path)) {
            return { format: "module", source: "export default \"\"", shortCircuit: true };
        }
        // 2. Files that should be loaded manually (determined by trial and error)
        if (shouldLoadManually(path)) {
            try {
                const source = await fs.readFile(path, "utf8");
                return { format: "module", source, shortCircuit: true };
            } catch (err) {
                console.error("Error reading file", path, err);
                throw err;
            }
        }

        // Otherwise, delegate to the default loader.
        return await defaultLoad(url, context, defaultLoad);
    } catch (err) {
        console.error("Error loading file:", url, path, err);
        throw err;
    }
}
