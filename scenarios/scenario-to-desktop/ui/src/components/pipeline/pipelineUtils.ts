/**
 * Suggests recovery steps based on common error patterns.
 */
export function suggestRecovery(
  errorMessage: string,
  scenarioName?: string,
): string | null {
  if (errorMessage.includes("not found") || errorMessage.includes("404")) {
    return scenarioName
      ? `Ensure the scenario '${scenarioName}' exists in /scenarios/ first.`
      : "Ensure the scenario exists in /scenarios/ first.";
  }
  if (
    errorMessage.includes("ui/dist") ||
    errorMessage.includes("UI not built")
  ) {
    return scenarioName
      ? `Build the scenario UI first: cd scenarios/${scenarioName}/ui && npm run build.`
      : "Build the scenario UI first.";
  }
  if (errorMessage.includes("permission") || errorMessage.includes("EACCES")) {
    return "Check file permissions in the scenarios directory.";
  }
  if (errorMessage.includes("ENOSPC") || errorMessage.includes("no space")) {
    return "Free up disk space and try again.";
  }
  if (errorMessage.includes("port") || errorMessage.includes("EADDRINUSE")) {
    return "Another process is using the required port. Stop it or change ports.";
  }
  return null;
}
