#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { Linter } from "eslint";
import parser from "@typescript-eslint/parser";
import usageContracts from "../eslint-rules/usage-contracts.js";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const flagships = ["web-console", "agent-manager", "notification-hub", "prompt-manager", "nutrition-planner", "personal-planner", "image-tools", "brand-manager"];
const args = process.argv.slice(2);
const rootsArg = args.indexOf("--roots");
const scenarioRoots = rootsArg >= 0 ? args[rootsArg + 1].split(",") : flagships.map((name) => `scenarios/${name}/ui/src`);
const linter = new Linter({ configType: "flat" });
const config = [{
  files: ["**/*.{ts,tsx}"],
  languageOptions: { parser, parserOptions: { ecmaVersion: 2022, sourceType: "module", ecmaFeatures: { jsx: true } } },
  plugins: { "rcl-usage-contracts": { rules: { "usage-contract": usageContracts } } },
  rules: { "rcl-usage-contracts/usage-contract": "error" },
}];
const findings = [];

function walk(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full);
    return /\.[jt]sx?$/.test(entry.name) && !/\.(test|spec)\.[jt]sx?$/.test(entry.name) ? [full] : [];
  });
}

for (const relativeRoot of scenarioRoots) {
  for (const file of walk(path.resolve(repo, relativeRoot))) {
    const source = fs.readFileSync(file, "utf8");
    const messages = linter.verify(source, config, { filename: file });
    for (const message of messages) {
      if (message.severity !== 2 || message.ruleId !== "rcl-usage-contracts/usage-contract") continue;
      findings.push({
        file: path.relative(repo, file),
        line: message.line,
        code: message.messageId ?? "usage-contract",
        message: message.message,
      });
    }
  }
}

if (args.includes("--json")) console.log(JSON.stringify({ findings }, null, 2));
else for (const finding of findings) console.log(`${finding.file}:${finding.line} ${finding.code}: ${finding.message}`);
if (findings.length) process.exitCode = 1;
else console.log(`Usage contracts passed for ${scenarioRoots.length} scenario UI source tree(s).`);
