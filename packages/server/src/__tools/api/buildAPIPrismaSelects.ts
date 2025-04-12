/**
 * Builds Prisma select objects for each endpoint, using the files in the "partial" folder
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { endpoints } from "./endpoints.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Creates a folder if it doesn't exist
 */
function createFolder(folder: string) {
    const absolutePath = path.resolve(__dirname, folder);
    const folderName = path.basename(absolutePath);
    if (!fs.existsSync(absolutePath)) {
        console.info(`Creating folder: "${folderName}" at "${absolutePath}"`);
        fs.mkdirSync(absolutePath, { recursive: true });
    } else {
        console.info(`Folder "${folderName}" already exists at "${absolutePath}"`);
    }
}

/**
 * Empties the contents of a folder if it exists (files only, not subfolders)
 */
function clearFilesInFolder(folder: string) {
    const absolutePath = path.resolve(__dirname, folder);
    if (fs.existsSync(absolutePath)) {
        const files = fs.readdirSync(absolutePath);
        for (const file of files) {
            const filePath = path.resolve(absolutePath, file);
            if (fs.lstatSync(filePath).isFile()) {
                fs.unlinkSync(filePath);
            }
        }
    }
}

async function main() {
    // Create output folder
    const outputPath = "../../endpoints/generated";
    createFolder(outputPath);
    clearFilesInFolder(outputPath);

    console.info("Generating prisma select objects for API endpoints...");

    // Keep track of the generated file names (without extension)
    const generatedFiles: string[] = [];

    // Unlazy each endpoint group and write it to a separate file
    for (const group of Object.keys(endpoints)) {
        const endpointGroup = await endpoints[group]();
        console.info(`generating endpoints for ${group}...`);
        for (const endpoint of Object.keys(endpointGroup)) {
            // Get the endpoint data
            const data = await endpointGroup[endpoint] as object;
            // Create file name
            const outputName = `${group}_${endpoint}`;
            // Write the data to a file, as a JavaScript export
            // eslint-disable-next-line no-magic-numbers
            const output = `export const ${outputName} = ${JSON.stringify(data, null, 4)};`;
            fs.writeFileSync(path.resolve(__dirname, outputPath, `${outputName}.ts`), output);

            // Add to our array of generated files
            generatedFiles.push(outputName);
        }
    }

    // Build export lines for each generated file
    const exportLines = generatedFiles.map((file) => {
        return `export * from "./${file}.js";`;
    });

    // Write the export lines into index.ts
    console.info(`Writing index.ts with ${exportLines.length} exports...`);
    fs.writeFileSync(
        path.resolve(__dirname, outputPath, "index.ts"),
        exportLines.join("\n") + "\n",
    );

    console.info("Finished generating prisma select objects for API endpoints🥳");
}

main().catch(console.error).finally(() => process.exit());
