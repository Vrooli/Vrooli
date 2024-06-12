/* eslint-disable @typescript-eslint/no-var-requires */
const path = require("path");
const camelcase = require("camelcase");

// This is a custom Jest transformer turning file imports into filenames.
// http://facebook.github.io/jest/docs/en/webpack.html

module.exports = {
    process(src, filename) {
        const assetFilename = JSON.stringify(path.basename(filename));

        if (filename.match(/\.svg$/)) {
            const pascalCaseFilename = camelcase(path.parse(filename).name, { pascalCase: true });
            const componentName = `Svg${pascalCaseFilename}`;
            const code = `const React = require('react');
            module.exports = {
                __esModule: true,
                default: ${assetFilename},
                ReactComponent: React.forwardRef(function ${componentName}(props, ref) {
                    return {
                        $$typeof: Symbol.for('react.element'),
                        type: 'svg',
                        ref: ref,
                        key: null,
                        props: Object.assign({}, props, {
                            children: ${assetFilename}
                        })
                    };
                }),
            };`;

            return { code }; // Updated to return an object with a `code` property
        }

        // For non-SVG files, simply return the asset filename in the expected format
        const code = `module.exports = ${assetFilename};`;
        return { code }; // Updated here as well
    },
};
