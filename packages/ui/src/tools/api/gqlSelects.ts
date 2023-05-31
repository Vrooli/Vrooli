/**
 * Converts graphql query/mutation data (which is set up to be type-safe and reduce code duplication) into a graphql-tag strings.
 * This is done during build to reduce runtime computation.
 */
import fs from "fs";
import { endpoints } from "./endpoints";

console.info("Generating graphql-tag strings for endpoints...");

// Step 1: Create output folders
// Create the output folders if they doesn't exist
const outputFolder = "../shared/src/api/generated";
const fragmentsFolder = `${outputFolder}/fragments`;
const endpointsFolder = `${outputFolder}/endpoints`;
for (const folder of [outputFolder, fragmentsFolder, endpointsFolder]) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
}

// Step 2: Find data and write endpoints to files
// Initialize fragments list to store all fragment definitions
const allFragments: { [name: string]: string } = {};
const endpointFiles: string[] = [];
// Unlazy each endpoint property and write it to a separate file
for (const objectType of Object.keys(endpoints)) {
    const endpointGroup = await (endpoints as any)[objectType]();
    console.log(`generating endpoints for ${objectType}...`);
    for (const endpointName of Object.keys(endpointGroup)) {
        // Get the endpoint data
        const { fragments, tag } = await endpointGroup[endpointName] as { fragments: [string, string][], tag: string };
        // Add the fragments to the total fragment object
        for (const [name, fragment] of fragments) {
            allFragments[name] = fragment;
        }
        // Calculate imports, startig with the gql import
        let importsString = "import gql from \"graphql-tag\";";
        // Add import for each fragment
        for (const [fragmentName] of fragments) {
            importsString += `\nimport { ${fragmentName} } from "../fragments/${fragmentName}";`;
        }
        // Write imports and endpoint to file
        const endpointString = `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${tag}\`;\n\n`;
        const outputPath = `${outputFolder}/endpoints/${objectType}_${endpointName}.ts`;
        fs.writeFileSync(outputPath, `${importsString}\n\n${endpointString}`);
        endpointFiles.push(`${objectType}_${endpointName}`);
    }
}

// Step 3: Write fragments to files
// Note: These do not use gql tags because they are always used in other gql tags. 
// If we made these gql tags, we'd have to import nested fragments, which might 
// cause duplicate imports.
for (const [name, fragment] of Object.entries(allFragments)) {
    const outputPath = `${outputFolder}/fragments/${name}.ts`;
    console.log(`generating fragment ${name}...`);
    // Write to file
    fs.writeFileSync(outputPath, `export const ${name} = \`${fragment}\`;\n`);
}

// Step 4: Write index.ts files
console.info("Generating index.ts files...");
// Create index.ts for endpoints
let indexContent = endpointFiles.map(file => `export * from "./${file}";`).join("\n");
fs.writeFileSync(`${endpointsFolder}/index.ts`, indexContent);

// Create index.ts for fragments
indexContent = Object.keys(allFragments).map(name => `export * from "./${name}";`).join("\n");
fs.writeFileSync(`${fragmentsFolder}/index.ts`, indexContent);

// Create index.ts for generated folder
indexContent = [
    "export * from \"./endpoints\";",
    "export * from \"./fragments\";",
    "export * from \"./graphqlTypes\";",
].join("\n");
fs.writeFileSync(`${outputFolder}/index.ts`, indexContent);

console.info("Finished generating graphql-tag strings for endpoints🚀");
