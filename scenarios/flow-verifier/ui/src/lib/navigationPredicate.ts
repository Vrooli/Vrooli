/**
 * Client-side evaluator for the navigation predicate DSL. Grammar mirrors
 * api/internal/flows/kinds/navigation/predicate/predicate.go:
 *
 *   expr        := or_expr
 *   or_expr     := and_expr ('OR' and_expr)*
 *   and_expr    := unary ('AND' unary)*
 *   unary       := 'NOT' unary | primary
 *   primary     := '(' expr ')' | comparison
 *   comparison  := IDENT op rhs
 *   op          := '=' | '!=' | 'IN' | 'CONTAINS'
 *   rhs         := list | atom               (atom for =, !=, CONTAINS)
 *   list        := '[' atom (',' atom)* ']'  (only after IN)
 *   atom        := IDENT ('=' IDENT)?
 *
 * The evaluator runs against a Lookup that resolves context names (e.g.
 * `viewport`) to their current value (`desktop`). Empty source means
 * "always true".
 */

export type Lookup = (name: string) => string | undefined;

type Token = { kind: "ident" | "op" | "lparen" | "rparen" | "lbracket" | "rbracket" | "comma"; text: string };

const KEYWORDS = new Set(["AND", "OR", "NOT", "IN", "CONTAINS"]);

function tokenize(src: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  while (i < src.length) {
    const c = src[i];
    if (!c || /\s/.test(c)) {
      i++;
      continue;
    }
    if (c === "(") {
      tokens.push({ kind: "lparen", text: c });
      i++;
      continue;
    }
    if (c === ")") {
      tokens.push({ kind: "rparen", text: c });
      i++;
      continue;
    }
    if (c === "[") {
      tokens.push({ kind: "lbracket", text: c });
      i++;
      continue;
    }
    if (c === "]") {
      tokens.push({ kind: "rbracket", text: c });
      i++;
      continue;
    }
    if (c === ",") {
      tokens.push({ kind: "comma", text: c });
      i++;
      continue;
    }
    if (c === "=") {
      tokens.push({ kind: "op", text: "=" });
      i++;
      continue;
    }
    if (c === "!" && src[i + 1] === "=") {
      tokens.push({ kind: "op", text: "!=" });
      i += 2;
      continue;
    }
    if (/[A-Za-z0-9_.\-:]/.test(c)) {
      let j = i;
      while (j < src.length) {
        const cj = src[j]!;
        if (/[A-Za-z0-9_.\-:]/.test(cj)) j++;
        else break;
      }
      const text = src.slice(i, j);
      if (KEYWORDS.has(text)) tokens.push({ kind: "op", text });
      else tokens.push({ kind: "ident", text });
      i = j;
      continue;
    }
    throw new Error(`unexpected character ${JSON.stringify(c)} at position ${i}`);
  }
  return tokens;
}

type Node =
  | { type: "or"; left: Node; right: Node }
  | { type: "and"; left: Node; right: Node }
  | { type: "not"; child: Node }
  | { type: "cmp"; left: string; op: "=" | "!=" | "IN" | "CONTAINS"; rhs: string | string[] };

class Parser {
  pos = 0;
  constructor(public tokens: Token[]) {}
  peek(): Token | undefined {
    return this.tokens[this.pos];
  }
  consume(): Token {
    const t = this.tokens[this.pos];
    if (!t) throw new Error("unexpected end of expression");
    this.pos++;
    return t;
  }
  expect(kind: Token["kind"], text?: string): Token {
    const t = this.consume();
    if (t.kind !== kind || (text !== undefined && t.text !== text)) {
      throw new Error(`expected ${text ?? kind}, got ${t.text}`);
    }
    return t;
  }
  parseExpr(): Node {
    return this.parseOr();
  }
  parseOr(): Node {
    let left = this.parseAnd();
    while (this.peek()?.kind === "op" && this.peek()?.text === "OR") {
      this.consume();
      const right = this.parseAnd();
      left = { type: "or", left, right };
    }
    return left;
  }
  parseAnd(): Node {
    let left = this.parseUnary();
    while (this.peek()?.kind === "op" && this.peek()?.text === "AND") {
      this.consume();
      const right = this.parseUnary();
      left = { type: "and", left, right };
    }
    return left;
  }
  parseUnary(): Node {
    if (this.peek()?.kind === "op" && this.peek()?.text === "NOT") {
      this.consume();
      return { type: "not", child: this.parseUnary() };
    }
    return this.parsePrimary();
  }
  parsePrimary(): Node {
    if (this.peek()?.kind === "lparen") {
      this.consume();
      const inner = this.parseExpr();
      this.expect("rparen");
      return inner;
    }
    return this.parseComparison();
  }
  parseComparison(): Node {
    const left = this.expect("ident").text;
    const op = this.consume();
    if (op.kind !== "op" || !["=", "!=", "IN", "CONTAINS"].includes(op.text)) {
      throw new Error(`expected comparison operator after ${left}, got ${op.text}`);
    }
    if (op.text === "IN") {
      this.expect("lbracket");
      const values: string[] = [];
      while (this.peek()?.kind !== "rbracket") {
        values.push(this.parseAtom());
        if (this.peek()?.kind === "comma") this.consume();
      }
      this.expect("rbracket");
      return { type: "cmp", left, op: "IN", rhs: values };
    }
    const rhs = this.parseAtom();
    return { type: "cmp", left, op: op.text as "=" | "!=" | "CONTAINS", rhs };
  }
  parseAtom(): string {
    const a = this.expect("ident").text;
    // Atom suffix: `auth=logged_in` (used by CONTAINS rhs against a list-encoded value).
    if (this.peek()?.kind === "op" && this.peek()?.text === "=") {
      this.consume();
      const b = this.expect("ident").text;
      return `${a}=${b}`;
    }
    return a;
  }
}

function evalNode(node: Node, lookup: Lookup): boolean {
  switch (node.type) {
    case "or":
      return evalNode(node.left, lookup) || evalNode(node.right, lookup);
    case "and":
      return evalNode(node.left, lookup) && evalNode(node.right, lookup);
    case "not":
      return !evalNode(node.child, lookup);
    case "cmp": {
      const value = lookup(node.left) ?? "";
      switch (node.op) {
        case "=":
          return value === (node.rhs as string);
        case "!=":
          return value !== (node.rhs as string);
        case "IN":
          return (node.rhs as string[]).includes(value);
        case "CONTAINS":
          return value.includes(node.rhs as string);
      }
    }
  }
}

/**
 * Parse and evaluate a predicate string. Empty input evaluates to `true`.
 * Throws on malformed input — callers should treat that as a contract
 * violation and surface it to the user.
 */
export function evaluatePredicate(src: string, lookup: Lookup): boolean {
  const trimmed = src.trim();
  if (trimmed === "") return true;
  const parser = new Parser(tokenize(trimmed));
  const node = parser.parseExpr();
  return evalNode(node, lookup);
}
