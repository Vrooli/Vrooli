/** Converts generated form values to the string payload owned by operator inputs. */
export function operatorAnswerValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return value.map(operatorAnswerValue).join(",");
  try {
    return JSON.stringify(value);
  } catch {
    return "";
  }
}
