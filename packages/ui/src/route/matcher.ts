import { Path } from "./useLocation";

export interface DefaultParams {
    [paramName: string]: string;
}
export type Params<T extends DefaultParams = DefaultParams> = T;

export type MatchWithParams<T extends DefaultParams = DefaultParams> = [
    true,
    Params<T>
];
export type NoMatch = [false, null];
export type Match<T extends DefaultParams = DefaultParams> =
    | MatchWithParams<T>
    | NoMatch;

export type MatcherFn = (pattern: Path, path: Path) => Match;

export interface PatternToRegexpResult {
    keys: Array<{ name: string | number }>;
    regexp: RegExp;
}

// creates a matcher function
export default function makeMatcher(makeRegexpFn: (pattern: string) => PatternToRegexpResult = pathToRegexp) {
    const cache: { [x: string]: PatternToRegexpResult } = {};

    // obtains a cached regexp version of the pattern
    const getRegexp = (pattern: string) =>
        cache[pattern] || (cache[pattern] = makeRegexpFn(pattern));

    return (pattern: string, path: string) => {
        const { regexp, keys } = getRegexp(pattern || "");
        const out = regexp.exec(path);

        if (!out) return [false, null];

        // formats an object with matched params
        const params = keys.reduce((params: any, key: any, i: number) => {
            params[key.name] = out[i + 1];
            return params;
        }, {});

        return [true, params];
    };
}

// escapes a regexp string (borrowed from path-to-regexp sources)
// https://github.com/pillarjs/path-to-regexp/blob/v3.0.0/index.js#L202
const escapeRx = (str: string) => str.replace(/([.+*?=^!:${}()[\]|/\\])/g, "\\$1");

// returns a segment representation in RegExp based on flags
// adapted and simplified version from path-to-regexp sources
const rxForSegment = (repeat: any, optional: any, prefix: any) => {
    let capture = repeat ? "((?:[^\\/]+?)(?:\\/(?:[^\\/]+?))*)" : "([^\\/]+?)";
    if (optional && prefix) capture = "(?:\\/" + capture + ")";
    return capture + (optional ? "?" : "");
};

const pathToRegexp = (pattern: any) => {
    const groupRx = /:([A-Za-z0-9_]+)([?+*]?)/g;

    let match: RegExpExecArray | null = null,
        lastIndex = 0,
        result = "";
    const keys: { name: string }[] = [];


    while ((match = groupRx.exec(pattern)) !== null) {
        const [_, segment, mod] = match;

        // :foo  [1]      (  )
        // :foo? [0 - 1]  ( o)
        // :foo+ [1 - ∞]  (r )
        // :foo* [0 - ∞]  (ro)
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
