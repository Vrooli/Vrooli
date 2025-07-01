import fs from "fs";
import { fileURLToPath } from "url";
import path from "path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Loads a JSON schema file from the schemas directory
 * @param schemaPath - Relative path to schema file from the schemas directory
 * @returns Parsed JSON schema object
 */
export function loadSchema(schemaPath: string): any {
    const fullPath = path.join(__dirname, "../schemas", schemaPath);
    try {
        const content = fs.readFileSync(fullPath, "utf8");
        return JSON.parse(content);
    } catch (error) {
        throw new Error(`Failed to load schema from ${schemaPath}: ${error}`);
    }
}
