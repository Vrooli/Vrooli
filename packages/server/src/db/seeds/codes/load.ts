// import fs from "fs";

// // Built-in code version functions need to be loaded from a file. We have it this way 
// // so that we can run and test them outside of the sandbox
// const distCodeFile = `${process.env.PROJECT_DIR}/packages/server/dist/db/seeds/codes/data.js`;
// const srcCodeFile = `${process.env.PROJECT_DIR}/packages/server/src/db/seeds/codes/data.ts`;

type CodeVersionInfo = {
    name: string;
    description: string;
    code: string;
};

// /**
//  * Splits a JavaScript file content into an array of stringified exported functions,
//  * including synchronous functions declared with 'export function' and
//  * asynchronous functions declared with 'export async function', and now
//  * supporting functions defined with 'export const'. Includes attached docstrings
//  * if present.
//  * 
//  * @param jsFileContent The entire JavaScript file content as a single string.
//  * @returns An array of string representations of each exported function, including docstrings.
//  */
// export function splitFunctions(jsFileContent: string): string[] {
//     function findExportStarts(code: string): { starts: { index: number, type: string }[], hasDefault: boolean } {
//         const starts: { index: number, type: string }[] = [];
//         let hasDefault = false;
//         let i = 0;

//         while (i < code.length) {
//             const char = code[i];
//             // Skip strings
//             if (char === "\"" || char === "'" || char === "`") {
//                 const quoteChar = char;
//                 i += 1;
//                 while (i < code.length) {
//                     if (code[i] === "\\") i += 2; // Skip escaped characters
//                     else if (code[i] === quoteChar) { i += 1; break; }
//                     else i += 1;
//                 }
//             } else if (code.substring(i, i + 6) === "export") {
//                 const afterExport = code.substring(i + 6);
//                 const matchDefault = afterExport.match(/^\s*default\b/);
//                 if (matchDefault) {
//                     hasDefault = true;
//                     i += 6 + matchDefault[0].length;
//                 } else {
//                     const matchAsyncFunction = afterExport.match(/^\s*async\s+function\b/);
//                     if (matchAsyncFunction) {
//                         starts.push({ index: i, type: "async function" });
//                         i += 6 + matchAsyncFunction[0].length;
//                     } else {
//                         const matchFunction = afterExport.match(/^\s*function\b/);
//                         if (matchFunction) {
//                             starts.push({ index: i, type: "function" });
//                             i += 6 + matchFunction[0].length;
//                         } else {
//                             const matchConst = afterExport.match(/^\s*const\b/);
//                             if (matchConst) {
//                                 starts.push({ index: i, type: "const" });
//                                 i += 6 + matchConst[0].length;
//                             } else {
//                                 i += 1;
//                             }
//                         }
//                     }
//                 }
//             } else {
//                 i += 1;
//             }
//         }
//         return { starts, hasDefault };
//     }

//     function findFunctionEnd(code: string, openingBraceIndex: number): number {
//         let braceCount = 1;
//         let i = openingBraceIndex + 1;
//         while (i < code.length && braceCount > 0) {
//             const char = code[i];
//             if (char === "\"" || char === "'" || char === "`") {
//                 const quoteChar = char;
//                 i += 1;
//                 while (i < code.length) {
//                     if (code[i] === "\\") i += 2;
//                     else if (code[i] === quoteChar) { i += 1; break; }
//                     else i += 1;
//                 }
//             } else if (char === "{") {
//                 braceCount += 1;
//                 i += 1;
//             } else if (char === "}") {
//                 braceCount -= 1;
//                 i += 1;
//             } else {
//                 i += 1;
//             }
//         }
//         return braceCount === 0 ? i - 1 : -1;
//     }

//     function findExportConstEnd(code: string, start: number): number {
//         const afterConst = code.substring(start + 11); // Skip "export const"
//         const equalsIndex = afterConst.indexOf("=");
//         if (equalsIndex === -1) return -1;
//         const afterEquals = afterConst.substring(equalsIndex + 1).trim();

//         // Check if it’s a function (arrow or regular)
//         if (afterEquals.startsWith("() =>") || afterEquals.startsWith("async () =>")) {
//             const arrowStart = start + 11 + equalsIndex + 1;
//             const bodyStart = code.indexOf("{", arrowStart);
//             if (bodyStart !== -1) {
//                 return findFunctionEnd(code, bodyStart);
//             } else {
//                 const semiIndex = code.indexOf(";", arrowStart);
//                 if (semiIndex !== -1) return semiIndex;
//                 return code.length - 1;
//             }
//         }
//         return -1;
//     }

//     function findDocstring(code: string, functionStart: number): number | null {
//         let i = functionStart - 1;
//         // Skip backwards over whitespace
//         while (i >= 0 && /\s/.test(code[i])) {
//             i--;
//         }
//         // Check if the previous characters are `*/`
//         if (i >= 1 && code[i] === "/" && code[i - 1] === "*") {
//             // Find the start of the comment by searching backwards for `/*`
//             i -= 2; // Move before `*/`
//             while (i >= 0) {
//                 if (code[i] === "/" && code[i + 1] === "*") {
//                     // Found the start; check if it’s a docstring with `/**`
//                     if (i + 2 < code.length && code[i + 2] === "*") {
//                         return i;
//                     } else {
//                         return null; // Regular comment, not a docstring
//                     }
//                 }
//                 i--;
//             }
//         }
//         return null;
//     }

//     const { starts, hasDefault } = findExportStarts(jsFileContent);
//     const functions: string[] = [];

//     for (const exp of starts) {
//         const docstringStart = findDocstring(jsFileContent, exp.index);
//         const actualStart = docstringStart !== null ? docstringStart : exp.index;
//         let end: number;

//         if (exp.type === "function" || exp.type === "async function") {
//             const openingBraceIndex = jsFileContent.indexOf("{", exp.index);
//             if (openingBraceIndex !== -1) {
//                 end = findFunctionEnd(jsFileContent, openingBraceIndex);
//                 if (end !== -1) {
//                     functions.push(jsFileContent.substring(actualStart, end + 1).trim());
//                 }
//             }
//         } else if (exp.type === "const" && !hasDefault) {
//             end = findExportConstEnd(jsFileContent, exp.index);
//             if (end !== -1) {
//                 functions.push(jsFileContent.substring(actualStart, end + 1).trim());
//             }
//         }
//     }

//     return functions;
// }

// /**
//  * Parses docstrings from a TypeScript file content and extracts metadata.
//  * Each extracted docstring results in an object with the function's name, description, and the function's name.
//  * 
//  * @param tsFileContent The TypeScript file content as a string.
//  * @returns An array of objects containing the name and description from the docstring and the function name.
//  */
// export function parseDocstrings(tsFileContent: string): { name: string, description: string, functionName: string }[] {
//     const docstringRegex = /\/\*\*([\s\S]*?)\*\//g;
//     const metadata: { name: string, description: string, functionName: string }[] = [];

//     let match;
//     while ((match = docstringRegex.exec(tsFileContent)) !== null) {
//         const docstringContent = match[1];

//         // Extract name and description
//         const nameMatch = docstringContent.match(/@name\s+(.*)/);
//         const descriptionMatch = docstringContent.match(/@description\s+(.*)/);

//         if (nameMatch && descriptionMatch) {
//             // Find the function name in the code following this docstring
//             const functionRegex = /export function (\w+)/;
//             const functionMatch = tsFileContent.substring(match.index + match[0].length)
//                 .match(functionRegex);

//             if (functionMatch) {
//                 metadata.push({
//                     name: nameMatch[1].trim(),
//                     description: descriptionMatch[1].trim(),
//                     functionName: functionMatch[1],
//                 });
//             }
//         }
//     }

//     return metadata;
// }

// /**
//  * Loads code functions from a file and creates a map of codeVersionID to code version info. 
//  * Info includes name, description, and the code itself.
//  * 
//  * @returns A map of codeVersionID to code version info.
//  */
// export function loadCodes(): { [id: string]: CodeVersionInfo } {
//     const codes: { [id: string]: CodeVersionInfo } = {};

//     try {
//         const distFileContent = fs.readFileSync(distCodeFile, "utf-8");
//         const functions = splitFunctions(distFileContent);

//         const srcFileContent = fs.readFileSync(srcCodeFile, "utf-8");
//         const functionMetadata = parseDocstrings(srcFileContent);

//         if (functions.length !== functionMetadata.length) {
//             logger.error("Error loading built-in functions. Function count mismatch", { trace: "0629" });
//             return {};
//         }

//         // Populate the codes object with the function metadata
//         functionMetadata.forEach(({ name, description, functionName }, index) => {
//             const code = functions[index];
//             const id = codeNameToId[functionName];
//             if (!id) {
//                 logger.error("Error loading built-in functions. Function ID not found", { trace: "0630" });
//                 return {};
//             }
//             codes[id] = { name, description, code };
//         });

//         console.log("Loaded codes:", JSON.stringify(codes, null, 2));
//     } catch (error) {
//         logger.error("Error loading built-in functions. They won't be upserted into the database", { trace: "0625" });
//     }

//     return codes;
// }

export { };
