import * as fs from "fs";
import * as path from "path";
import * as tsj from "ts-json-schema-generator";
import { fileURLToPath } from "url";

// Get the directory name of the current module
const dirname = path.dirname(fileURLToPath(import.meta.url));

const config = {
    path: path.resolve(dirname, "../api/generated/graphqlTypes.ts"),
    tsconfig: path.resolve(dirname, "../../tsconfig.json"),
    type: "MeetingInviteStatus", // the type you want to generate schema for
};

const program = tsj.createProgram(config);
const parser = tsj.createParser(program, config);
const formatter = tsj.createFormatter(config);
const generator = new tsj.SchemaGenerator(program, parser, formatter, config);
const schema = generator.createSchema(config.type);

const schemaString = JSON.stringify(schema, null, 2);

fs.writeFile("graphqlJsonTypes.json", schemaString, (err) => {
    if (err) throw err;
    console.log("The file has been saved!");
});
