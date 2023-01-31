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
for (const endpointName of Object.keys(endpoints)) {
    const endpoint = endpoints[endpointName]();
    const outputPath = `${outputFolder}/${endpointName}.ts`;
    const outputContents = `export default ${JSON.stringify(endpoint, null, 2)}`;
    fs.writeFileSync(outputPath, outputContents);
}
// Load endpoints object from file
console.log('didnt crash!');
export {}