/**
 * Converts graphql query/mutation data (which is set up to be type-safe and reduce code duplication) into a graphql-tag strings.
 * This is done during build to reduce runtime computation.
 */
import fs from 'fs';
import { endpoints } from '../api/endpoints';

console.info('Generating graphql-tag strings for endpoints...');
// Create the output folder if it doesn't exist
const outputFolder = './src/api/generated';
if (!fs.existsSync(outputFolder)) {
    console.info(`Creating output folder: ${outputFolder}`)
    fs.mkdirSync(outputFolder);
}
// Unlazy each endpoint property and write it to a separate file
for (const objectType of Object.keys(endpoints)) {
    const endpointGroup = await endpoints[objectType]();
    const outputPath = `${outputFolder}/${objectType}.ts`;
    console.log(`generating endpoints for ${objectType}...`);
    let outputContents = '';
    // Add import for gql
    outputContents += `import gql from 'graphql-tag';\n\n`;
    // For each endpoint in the group, unlazy it and write it to the file
    for (const endpointName of Object.keys(endpointGroup)) {
        const endpoint = await endpointGroup[endpointName];
        outputContents += `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${endpoint[0]}\`;\n\n`;
    }
    fs.writeFileSync(outputPath, outputContents);
}
// Load endpoints object from file
console.log('didnt crash!');
export {}