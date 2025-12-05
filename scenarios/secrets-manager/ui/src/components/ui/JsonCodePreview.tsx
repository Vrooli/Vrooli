import { useMemo } from "react";

interface JsonCodePreviewProps {
  data: unknown;
  className?: string;
}

type TokenType = "key" | "string" | "number" | "boolean" | "null" | "punctuation";

interface Token {
  type: TokenType;
  value: string;
}

const TOKEN_COLORS: Record<TokenType, string> = {
  key: "text-sky-300",
  string: "text-emerald-300",
  number: "text-amber-300",
  boolean: "text-violet-300",
  null: "text-rose-300",
  punctuation: "text-white/50",
};

function tokenizeJson(json: string): Token[][] {
  const lines = json.split("\n");
  return lines.map((line) => {
    const tokens: Token[] = [];
    let i = 0;

    while (i < line.length) {
      // Skip whitespace - add as punctuation to preserve indentation
      if (/\s/.test(line[i])) {
        let ws = "";
        while (i < line.length && /\s/.test(line[i])) {
          ws += line[i];
          i++;
        }
        tokens.push({ type: "punctuation", value: ws });
        continue;
      }

      // String (could be key or value)
      if (line[i] === '"') {
        let str = '"';
        i++;
        while (i < line.length && line[i] !== '"') {
          if (line[i] === "\\" && i + 1 < line.length) {
            str += line[i] + line[i + 1];
            i += 2;
          } else {
            str += line[i];
            i++;
          }
        }
        if (i < line.length) {
          str += '"';
          i++;
        }

        // Check if this is a key (followed by colon)
        let j = i;
        while (j < line.length && /\s/.test(line[j])) j++;
        const isKey = j < line.length && line[j] === ":";

        tokens.push({ type: isKey ? "key" : "string", value: str });
        continue;
      }

      // Number
      if (/[-\d]/.test(line[i])) {
        let num = "";
        while (i < line.length && /[-\d.eE+]/.test(line[i])) {
          num += line[i];
          i++;
        }
        tokens.push({ type: "number", value: num });
        continue;
      }

      // Boolean or null
      if (line.slice(i, i + 4) === "true") {
        tokens.push({ type: "boolean", value: "true" });
        i += 4;
        continue;
      }
      if (line.slice(i, i + 5) === "false") {
        tokens.push({ type: "boolean", value: "false" });
        i += 5;
        continue;
      }
      if (line.slice(i, i + 4) === "null") {
        tokens.push({ type: "null", value: "null" });
        i += 4;
        continue;
      }

      // Punctuation (braces, brackets, colons, commas)
      tokens.push({ type: "punctuation", value: line[i] });
      i++;
    }

    return tokens;
  });
}

export function JsonCodePreview({ data, className = "" }: JsonCodePreviewProps) {
  const { lines, lineCount } = useMemo(() => {
    const json = JSON.stringify(data, null, 2);
    const tokenized = tokenizeJson(json);
    return {
      lines: tokenized,
      lineCount: tokenized.length,
    };
  }, [data]);

  const gutterWidth = Math.max(String(lineCount).length * 0.6 + 1, 2.5);

  return (
    <div className={`font-mono text-[11px] leading-5 ${className}`}>
      <div className="flex">
        {/* Line number gutter */}
        <div
          className="flex-shrink-0 select-none border-r border-white/10 bg-black/30 text-right text-white/30"
          style={{ width: `${gutterWidth}rem` }}
        >
          {lines.map((_, idx) => (
            <div key={idx} className="px-2">
              {idx + 1}
            </div>
          ))}
        </div>

        {/* Code content */}
        <div className="flex-1 overflow-x-auto bg-black/20 pl-3 pr-4">
          {lines.map((tokens, lineIdx) => (
            <div key={lineIdx} className="whitespace-pre">
              {tokens.length === 0 ? (
                <span>&nbsp;</span>
              ) : (
                tokens.map((token, tokenIdx) => (
                  <span key={tokenIdx} className={TOKEN_COLORS[token.type]}>
                    {token.value}
                  </span>
                ))
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
