import { matchProseFilePaths } from "../../../lib/fileReferences";
const SKIPPED_PARENTS = new Set(["code", "inlineCode", "link", "linkReference", "image", "imageReference", "definition", "html"]);
function transformTextNode(node) {
    const text = node.value ?? "";
    const matches = matchProseFilePaths(text);
    if (matches.length === 0)
        return null;
    const replacement = [];
    let cursor = 0;
    for (const match of matches) {
        if (match.start > cursor) {
            replacement.push({ type: "text", value: text.slice(cursor, match.start) });
        }
        replacement.push({
            type: "link",
            url: match.path,
            data: { hProperties: { "data-prose-path": "true" } },
            children: [{ type: "text", value: match.path }],
        });
        cursor = match.end;
    }
    if (cursor < text.length) {
        replacement.push({ type: "text", value: text.slice(cursor) });
    }
    return replacement;
}
function walk(node) {
    if (!node.children || SKIPPED_PARENTS.has(node.type))
        return;
    for (let i = 0; i < node.children.length; i++) {
        const child = node.children[i];
        if (!child)
            continue;
        if (child.type === "text") {
            const replacement = transformTextNode(child);
            if (replacement) {
                node.children.splice(i, 1, ...replacement);
                i += replacement.length - 1;
            }
            continue;
        }
        walk(child);
    }
}
export function remarkProsePaths() {
    return (tree) => {
        walk(tree);
    };
}
