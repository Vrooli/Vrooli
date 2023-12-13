import { parser } from '@lezer/python';
import { syntaxTree, LRLanguage, indentNodeProp, delimitedIndent, foldNodeProp, foldInside, LanguageSupport } from '@codemirror/language';
import { NodeWeakMap, IterMode } from '@lezer/common';
import { snippetCompletion, ifNotIn, completeFromList } from '@codemirror/autocomplete';

const cache = /*@__PURE__*/new NodeWeakMap();
const ScopeNodes = /*@__PURE__*/new Set([
    "Script", "Body",
    "FunctionDefinition", "ClassDefinition", "LambdaExpression",
    "ForStatement", "MatchClause"
]);
function defID(type) {
    return (node, def, outer) => {
        if (outer)
            return false;
        let id = node.node.getChild("VariableName");
        if (id)
            def(id, type);
        return true;
    };
}
const gatherCompletions = {
    FunctionDefinition: /*@__PURE__*/defID("function"),
    ClassDefinition: /*@__PURE__*/defID("class"),
    ForStatement(node, def, outer) {
        if (outer)
            for (let child = node.node.firstChild; child; child = child.nextSibling) {
                if (child.name == "VariableName")
                    def(child, "variable");
                else if (child.name == "in")
                    break;
            }
    },
    ImportStatement(_node, def) {
        var _a, _b;
        let { node } = _node;
        let isFrom = ((_a = node.firstChild) === null || _a === void 0 ? void 0 : _a.name) == "from";
        for (let ch = node.getChild("import"); ch; ch = ch.nextSibling) {
            if (ch.name == "VariableName" && ((_b = ch.nextSibling) === null || _b === void 0 ? void 0 : _b.name) != "as")
                def(ch, isFrom ? "variable" : "namespace");
        }
    },
    AssignStatement(node, def) {
        for (let child = node.node.firstChild; child; child = child.nextSibling) {
            if (child.name == "VariableName")
                def(child, "variable");
            else if (child.name == ":" || child.name == "AssignOp")
                break;
        }
    },
    ParamList(node, def) {
        for (let prev = null, child = node.node.firstChild; child; child = child.nextSibling) {
            if (child.name == "VariableName" && (!prev || !/\*|AssignOp/.test(prev.name)))
                def(child, "variable");
            prev = child;
        }
    },
    CapturePattern: /*@__PURE__*/defID("variable"),
    AsPattern: /*@__PURE__*/defID("variable"),
    __proto__: null
};
function getScope(doc, node) {
    let cached = cache.get(node);
    if (cached)
        return cached;
    let completions = [], top = true;
    function def(node, type) {
        let name = doc.sliceString(node.from, node.to);
        completions.push({ label: name, type });
    }
    node.cursor(IterMode.IncludeAnonymous).iterate(node => {
        if (node.name) {
            let gather = gatherCompletions[node.name];
            if (gather && gather(node, def, top) || !top && ScopeNodes.has(node.name))
                return false;
            top = false;
        }
        else if (node.to - node.from > 8192) {
            // Allow caching for bigger internal nodes
            for (let c of getScope(doc, node.node))
                completions.push(c);
            return false;
        }
    });
    cache.set(node, completions);
    return completions;
}
const Identifier = /^[\w\xa1-\uffff][\w\d\xa1-\uffff]*$/;
const dontComplete = ["String", "FormatString", "Comment", "PropertyName"];
/**
Completion source that looks up locally defined names in
Python code.
*/
function localCompletionSource(context) {
    let inner = syntaxTree(context.state).resolveInner(context.pos, -1);
    if (dontComplete.indexOf(inner.name) > -1)
        return null;
    let isWord = inner.name == "VariableName" ||
        inner.to - inner.from < 20 && Identifier.test(context.state.sliceDoc(inner.from, inner.to));
    if (!isWord && !context.explicit)
        return null;
    let options = [];
    for (let pos = inner; pos; pos = pos.parent) {
        if (ScopeNodes.has(pos.name))
            options = options.concat(getScope(context.state.doc, pos));
    }
    return {
        options,
        from: isWord ? inner.from : context.pos,
        validFor: Identifier
    };
}
const globals = /*@__PURE__*/[
    "__annotations__", "__builtins__", "__debug__", "__doc__", "__import__", "__name__",
    "__loader__", "__package__", "__spec__",
    "False", "None", "True"
].map(n => ({ label: n, type: "constant" })).concat(/*@__PURE__*/[
    "ArithmeticError", "AssertionError", "AttributeError", "BaseException", "BlockingIOError",
    "BrokenPipeError", "BufferError", "BytesWarning", "ChildProcessError", "ConnectionAbortedError",
    "ConnectionError", "ConnectionRefusedError", "ConnectionResetError", "DeprecationWarning",
    "EOFError", "Ellipsis", "EncodingWarning", "EnvironmentError", "Exception", "FileExistsError",
    "FileNotFoundError", "FloatingPointError", "FutureWarning", "GeneratorExit", "IOError",
    "ImportError", "ImportWarning", "IndentationError", "IndexError", "InterruptedError",
    "IsADirectoryError", "KeyError", "KeyboardInterrupt", "LookupError", "MemoryError",
    "ModuleNotFoundError", "NameError", "NotADirectoryError", "NotImplemented", "NotImplementedError",
    "OSError", "OverflowError", "PendingDeprecationWarning", "PermissionError", "ProcessLookupError",
    "RecursionError", "ReferenceError", "ResourceWarning", "RuntimeError", "RuntimeWarning",
    "StopAsyncIteration", "StopIteration", "SyntaxError", "SyntaxWarning", "SystemError",
    "SystemExit", "TabError", "TimeoutError", "TypeError", "UnboundLocalError", "UnicodeDecodeError",
    "UnicodeEncodeError", "UnicodeError", "UnicodeTranslateError", "UnicodeWarning", "UserWarning",
    "ValueError", "Warning", "ZeroDivisionError"
].map(n => ({ label: n, type: "type" }))).concat(/*@__PURE__*/[
    "bool", "bytearray", "bytes", "classmethod", "complex", "float", "frozenset", "int", "list",
    "map", "memoryview", "object", "range", "set", "staticmethod", "str", "super", "tuple", "type"
].map(n => ({ label: n, type: "class" }))).concat(/*@__PURE__*/[
    "abs", "aiter", "all", "anext", "any", "ascii", "bin", "breakpoint", "callable", "chr",
    "compile", "delattr", "dict", "dir", "divmod", "enumerate", "eval", "exec", "exit", "filter",
    "format", "getattr", "globals", "hasattr", "hash", "help", "hex", "id", "input", "isinstance",
    "issubclass", "iter", "len", "license", "locals", "max", "min", "next", "oct", "open",
    "ord", "pow", "print", "property", "quit", "repr", "reversed", "round", "setattr", "slice",
    "sorted", "sum", "vars", "zip"
].map(n => ({ label: n, type: "function" })));
const snippets = [
    /*@__PURE__*/snippetCompletion("def ${name}(${params}):\n\t${}", {
        label: "def",
        detail: "function",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("for ${name} in ${collection}:\n\t${}", {
        label: "for",
        detail: "loop",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("while ${}:\n\t${}", {
        label: "while",
        detail: "loop",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("try:\n\t${}\nexcept ${error}:\n\t${}", {
        label: "try",
        detail: "/ except block",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("if ${}:\n\t\n", {
        label: "if",
        detail: "block",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("if ${}:\n\t${}\nelse:\n\t${}", {
        label: "if",
        detail: "/ else block",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("class ${name}:\n\tdef __init__(self, ${params}):\n\t\t\t${}", {
        label: "class",
        detail: "definition",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("import ${module}", {
        label: "import",
        detail: "statement",
        type: "keyword"
    }),
    /*@__PURE__*/snippetCompletion("from ${module} import ${names}", {
        label: "from",
        detail: "import",
        type: "keyword"
    })
];
/**
Autocompletion for built-in Python globals and keywords.
*/
const globalCompletion = /*@__PURE__*/ifNotIn(dontComplete, /*@__PURE__*/completeFromList(/*@__PURE__*/globals.concat(snippets)));

function indentBody(context, node) {
    let base = context.baseIndentFor(node);
    let line = context.lineAt(context.pos, -1), to = line.from + line.text.length;
    // Don't consider blank, deindented lines at the end of the
    // block part of the block
    if (/^\s*($|#)/.test(line.text) &&
        context.node.to < to + 100 &&
        !/\S/.test(context.state.sliceDoc(to, context.node.to)) &&
        context.lineIndent(context.pos, -1) <= base)
        return null;
    // A normally deindenting keyword that appears at a higher
    // indentation than the block should probably be handled by the next
    // level
    if (/^\s*(else:|elif |except |finally:)/.test(context.textAfter) && context.lineIndent(context.pos, -1) > base)
        return null;
    return base + context.unit;
}
/**
A language provider based on the [Lezer Python
parser](https://github.com/lezer-parser/python), extended with
highlighting and indentation information.
*/
const pythonLanguage = /*@__PURE__*/LRLanguage.define({
    name: "python",
    parser: /*@__PURE__*/parser.configure({
        props: [
            /*@__PURE__*/indentNodeProp.add({
                Body: context => { var _a; return (_a = indentBody(context, context.node)) !== null && _a !== void 0 ? _a : context.continue(); },
                IfStatement: cx => /^\s*(else:|elif )/.test(cx.textAfter) ? cx.baseIndent : cx.continue(),
                TryStatement: cx => /^\s*(except |finally:|else:)/.test(cx.textAfter) ? cx.baseIndent : cx.continue(),
                "TupleExpression ComprehensionExpression ParamList ArgList ParenthesizedExpression": /*@__PURE__*/delimitedIndent({ closing: ")" }),
                "DictionaryExpression DictionaryComprehensionExpression SetExpression SetComprehensionExpression": /*@__PURE__*/delimitedIndent({ closing: "}" }),
                "ArrayExpression ArrayComprehensionExpression": /*@__PURE__*/delimitedIndent({ closing: "]" }),
                "String FormatString": () => null,
                Script: context => {
                    if (context.pos + /\s*/.exec(context.textAfter)[0].length >= context.node.to) {
                        let endBody = null;
                        for (let cur = context.node, to = cur.to;;) {
                            cur = cur.lastChild;
                            if (!cur || cur.to != to)
                                break;
                            if (cur.type.name == "Body")
                                endBody = cur;
                        }
                        if (endBody) {
                            let bodyIndent = indentBody(context, endBody);
                            if (bodyIndent != null)
                                return bodyIndent;
                        }
                    }
                    return context.continue();
                }
            }),
            /*@__PURE__*/foldNodeProp.add({
                "ArrayExpression DictionaryExpression SetExpression TupleExpression": foldInside,
                Body: (node, state) => ({ from: node.from + 1, to: node.to - (node.to == state.doc.length ? 0 : 1) })
            })
        ],
    }),
    languageData: {
        closeBrackets: {
            brackets: ["(", "[", "{", "'", '"', "'''", '"""'],
            stringPrefixes: ["f", "fr", "rf", "r", "u", "b", "br", "rb",
                "F", "FR", "RF", "R", "U", "B", "BR", "RB"]
        },
        commentTokens: { line: "#" },
        indentOnInput: /^\s*([\}\]\)]|else:|elif |except |finally:)$/
    }
});
/**
Python language support.
*/
function python() {
    return new LanguageSupport(pythonLanguage, [
        pythonLanguage.data.of({ autocomplete: localCompletionSource }),
        pythonLanguage.data.of({ autocomplete: globalCompletion }),
    ]);
}

export { globalCompletion, localCompletionSource, python, pythonLanguage };
