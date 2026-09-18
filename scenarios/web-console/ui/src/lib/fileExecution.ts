import { basename } from "./paths";

export interface ScriptRunPlan {
  command: string;
  workingDir: string;
  risky: boolean;
  interpreter: string;
}

const INTERPRETERS: Record<string, string> = {
  sh: "sh", bash: "bash", zsh: "zsh", fish: "fish", py: "python3", pyw: "python3",
  js: "node", mjs: "node", cjs: "node", ts: "tsx", rb: "ruby", php: "php", pl: "perl",
};

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

export function scriptRunPlan(path: string, text = ""): ScriptRunPlan | null {
  const name = basename(path);
  const extension = name.includes(".") ? name.split(".").pop()?.toLowerCase() ?? "" : "";
  const shebang = text.match(/^#!\s*\/usr\/bin\/env\s+([^\s]+)/m)?.[1]
    ?? text.match(/^#!\s*\/bin\/([^\s]+)/m)?.[1];
  const interpreter = shebang || INTERPRETERS[extension];
  if (!interpreter) return null;
  const command = `${interpreter} ${shellQuote(path)}`;
  const risky = /(^|\s)(rm|rmdir|mkfs|dd|shutdown|reboot|kill|chmod|chown|sudo|apt|brew|npm|pnpm|yarn|pip)(\s|$)/i.test(text);
  const separator = path.includes("\\") ? "\\" : "/";
  const index = path.lastIndexOf(separator);
  const workingDir = index > 0 ? path.slice(0, index) : (separator === "\\" && /^[A-Za-z]:/.test(path) ? path.slice(0, 3) : separator);
  return { command, workingDir, risky, interpreter };
}
