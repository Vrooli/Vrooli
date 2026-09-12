import axe from "axe-core";

export async function expectNoA11yViolations(container: HTMLElement): Promise<void> {
  const result = await axe.run(container);
  if (result.violations.length > 0) {
    throw new Error(result.violations.map((violation) => `${violation.id}: ${violation.help}`).join("\n"));
  }
}
