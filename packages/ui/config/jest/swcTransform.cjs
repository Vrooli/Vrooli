// Sets up swc as a transformer for Jest. 
// Can add any additional transforms here, in the `process` function, 
// by passing `src` to a custom transform function.
const swcJest = require("@swc/jest");

// See https://swc.rs/docs/configuration/compilation for options
const swcTransformer = swcJest.createTransformer({
    jsc: {
        experimental: {
            keepImportAssertions: false, // Doesn't work right now, so we still need custom transform
        },
    },
});

/** Removes import assertion from import statements */
function importAssertionTransform(src) {
    return src.replace(/import(.*?)assert\s*\{\s*type:\s*"json"\s*\}\s*;/g, "import$1;");
}

module.exports = {
    process(src, filename, config, options) {
        const transformedContent = importAssertionTransform(src);
        return swcTransformer.process(transformedContent, filename, config, options);
    },
};
