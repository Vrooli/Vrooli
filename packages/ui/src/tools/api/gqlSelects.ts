/**
 * Converts graphql query/mutation data (which is set up to be type-safe and reduce code duplication) into a graphql-tag strings.
 * This is done during build to reduce runtime computation.
 */
import fs from 'fs';
import { endpoints } from '../api/endpoints';

console.info('Generating graphql-tag strings for endpoints...');

// Step 1: Create output folders
// Create the output folders if they doesn't exist
const outputFolder = './src/api/generated';
const fragmentsFolder = `${outputFolder}/fragments`;
const endpointsFolder = `${outputFolder}/endpoints`;
for (const folder of [outputFolder, fragmentsFolder, endpointsFolder]) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`)
        fs.mkdirSync(folder);
    }
}

// Step 2: Find data and write endoints to files
// Initialize fragments list to store all fragment definitions
const allFragments: { [name: string]: string } = {};
// Unlazy each endpoint property and write it to a separate file
for (const objectType of Object.keys(endpoints)) {
    const endpointGroup = await endpoints[objectType]();
    const outputPath = `${outputFolder}/endpoints/${objectType}.ts`;
    console.log(`generating endpoints for ${objectType}...`);
    let endpointString = '';
    let currFragmentNames: string[] = [];
    // For each endpoint in the group
    for (const endpointName of Object.keys(endpointGroup)) {
        // Get the endpoint data
        const { fragments, tag } = await endpointGroup[endpointName] as { fragments: [string, string][], tag: string };
        // Add the fragments to the endpoint list and total fragment object
        currFragmentNames.push(...fragments.map(f => f[0]));
        for (const [name, fragment] of fragments) {
            allFragments[name] = fragment;
        }
        endpointString += `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${tag}\`;\n\n`;
    }
    // Calculate imports, startig with the gql import
    let importsString = `import gql from 'graphql-tag';`;
    // Add import for each fragment
    for (const fragmentName of new Set(currFragmentNames)) {
        importsString += `\nimport { ${fragmentName} } from '../fragments/${fragmentName}';`;
    }
    // Write imports and endpoints to file
    fs.writeFileSync(outputPath, `${importsString}\n\n${endpointString}`);
}

// Step 3: Write fragments to files
// Note: These do not use gql tags because they are always used in other gql tags. 
// If we made these gql tags, we'd have to import nested fragments, which might 
// cause duplicate imports.
for (const [name, fragment] of Object.entries(allFragments)) {
    const outputPath = `${outputFolder}/fragments/${name}.ts`;
    console.log(`generating fragment ${name}...`);
    // Write to file
    fs.writeFileSync(outputPath, `export const ${name} = \`${fragment}\`;`);
}

console.info('Finished generating graphql-tag strings for endpoints🚀');