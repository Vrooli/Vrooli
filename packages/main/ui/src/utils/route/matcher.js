export default function makeMatcher(makeRegexpFn = pathToRegexp) {
    const cache = {};
    const getRegexp = (pattern) => cache[pattern] || (cache[pattern] = makeRegexpFn(pattern));
    return (pattern, path) => {
        const { regexp, keys } = getRegexp(pattern || "");
        const out = regexp.exec(path);
        if (!out)
            return [false, null];
        const params = keys.reduce((params, key, i) => {
            params[key.name] = out[i + 1];
            return params;
        }, {});
        return [true, params];
    };
}
const escapeRx = (str) => str.replace(/([.+*?=^!:${}()[\]|/\\])/g, "\\$1");
const rxForSegment = (repeat, optional, prefix) => {
    let capture = repeat ? "((?:[^\\/]+?)(?:\\/(?:[^\\/]+?))*)" : "([^\\/]+?)";
    if (optional && prefix)
        capture = "(?:\\/" + capture + ")";
    return capture + (optional ? "?" : "");
};
const pathToRegexp = (pattern) => {
    const groupRx = /:([A-Za-z0-9_]+)([?+*]?)/g;
    let match = null, lastIndex = 0, keys = [], result = "";
    while ((match = groupRx.exec(pattern)) !== null) {
        const [_, segment, mod] = match;
        const repeat = mod === "+" || mod === "*";
        const optional = mod === "?" || mod === "*";
        const prefix = optional && pattern[match.index - 1] === "/" ? 1 : 0;
        const prev = pattern.substring(lastIndex, match.index - prefix);
        keys.push({ name: segment });
        lastIndex = groupRx.lastIndex;
        result += escapeRx(prev) + rxForSegment(repeat, optional, prefix);
    }
    result += escapeRx(pattern.substring(lastIndex));
    return { keys, regexp: new RegExp("^" + result + "(?:\\/)?$", "i") };
};
//# sourceMappingURL=matcher.js.map