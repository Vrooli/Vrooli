// The library supports browsers and server renderers. Load only React's
// ambient types, then model the source's optional environment read without
// assuming a Node global is present in browser runtimes.
export const packageBuildTypes = Object.freeze(["react", "react-dom"]);

export const packageBuildEnvironmentDeclarations = `declare module "*.css" { const css: string; export default css; }
declare var process: { env?: { NODE_ENV?: string } } | undefined;
`;
