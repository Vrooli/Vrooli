import fs from "fs";
import { endpoints } from "./endpoints";
console.info("Generating graphql-tag strings for endpoints...");
const outputFolder = "./src/api/generated";
const fragmentsFolder = `${outputFolder}/fragments`;
const endpointsFolder = `${outputFolder}/endpoints`;
for (const folder of [outputFolder, fragmentsFolder, endpointsFolder]) {
    if (!fs.existsSync(folder)) {
        console.info(`Creating folder: ${folder}`);
        fs.mkdirSync(folder);
    }
}
const allFragments = {};
for (const objectType of Object.keys(endpoints)) {
    const endpointGroup = await endpoints[objectType]();
    console.log(`generating endpoints for ${objectType}...`);
    for (const endpointName of Object.keys(endpointGroup)) {
        const { fragments, tag } = await endpointGroup[endpointName];
        for (const [name, fragment] of fragments) {
            allFragments[name] = fragment;
        }
        let importsString = "import gql from \"graphql-tag\";";
        for (const [fragmentName] of fragments) {
            importsString += `\nimport { ${fragmentName} } from "../fragments/${fragmentName}";`;
        }
        const endpointString = `export const ${objectType}${endpointName[0].toUpperCase() + endpointName.slice(1)} = gql\`${tag}\`;\n\n`;
        const outputPath = `${outputFolder}/endpoints/${objectType}_${endpointName}.ts`;
        fs.writeFileSync(outputPath, `${importsString}\n\n${endpointString}`);
    }
}
for (const [name, fragment] of Object.entries(allFragments)) {
    const outputPath = `${outputFolder}/fragments/${name}.ts`;
    console.log(`generating fragment ${name}...`);
    fs.writeFileSync(outputPath, `export const ${name} = \`${fragment}\`;\n`);
}
console.info("Finished generating graphql-tag strings for endpoints🚀");
//# sourceMappingURL=gqlSelects.js.map