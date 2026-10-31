import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const connectClients = [
  "operatorinputs.ts", "readiness.ts", "apply.ts", "session.ts", "selection.ts",
  "capabilities.ts", "credentials.ts", "host.ts", "operatorstate.ts", "resources.ts", "glossary.ts",
];

it("uses generated Connect service descriptors for every domain client", () => {
  for (const file of connectClients) {
    const source = readFileSync(resolve(process.cwd(), "src/api", file), "utf8");
    expect(source, file).toContain("createClient(");
    expect(source, file).toContain("onboardingTransport");
    expect(source, file).not.toMatch(/fetch\s*\(/);
    expect(source, file).not.toMatch(/axios\s*\./);
  }
});
