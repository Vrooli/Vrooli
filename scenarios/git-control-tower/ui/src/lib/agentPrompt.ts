import type { AgentContextItem } from "./api";

const MAX_PROMPT_CHARS = 50_000;

/** Minimum user-message length to treat as an explicit instruction (not empty/minimal). */
const MIN_EXPLICIT_MESSAGE_LENGTH = 20;

/**
 * Build an imperative task directive from attached context kinds.
 *
 * When the user attaches context items but provides no (or minimal) typed
 * message, this generates a clear "do this now" instruction so the agent
 * starts working immediately instead of asking what to do.
 */
export function buildTaskDirective(contextItems: AgentContextItem[]): string {
  const kinds = [...new Set(contextItems.map((i) => i.kind))];
  const actionable = kinds.filter((k) => k !== "change-summary" && k !== "screenshot");

  if (actionable.length === 0) return "";

  const labelMap: Record<string, string> = {
    "test-failure": "test failures",
    "scenario-quality": "code quality violations",
  };
  const issueLabels = actionable.map((k) => labelMap[k] ?? k).join(" and ");

  return (
    `Fix all of the ${issueLabels} detailed in the attached context below. ` +
    "Read the relevant source files, implement the fixes, and verify each fix " +
    "using the verification command listed in that context item."
  );
}

/**
 * Build acceptance criteria from the context items' verification commands.
 *
 * Appended at the end of the prompt so the agent knows what "done" looks like.
 */
export function buildAcceptanceCriteria(contextItems: AgentContextItem[]): string {
  const kinds = [...new Set(contextItems.map((i) => i.kind))];
  const actionable = kinds.filter((k) => k !== "change-summary" && k !== "screenshot");
  if (actionable.length === 0) return "";

  return (
    "\n\n---\n\n## Acceptance Criteria\n\n" +
    "- All issues listed in the attached context have been addressed\n" +
    "- Verification commands in each context item pass\n" +
    "- No new issues introduced by your changes\n"
  );
}

/**
 * Compose a full prompt from user message, attached context items, and optional envelope.
 *
 * The final prompt is structured as:
 *   [screenshot paths]     <- so the agent can read image files
 *   [envelope]             <- scenario orientation + autonomy directive (first message only)
 *   [task directive]       <- auto-generated when message is empty/minimal and context attached
 *   [user message]
 *   ---
 *   ## Attached Context
 *   [context items]
 *   ---
 *   ## Acceptance Criteria  <- auto-generated when actionable context is attached
 *
 * Truncated to {@link MAX_PROMPT_CHARS} if the combined content exceeds the limit.
 *
 * @param message - The user's typed message.
 * @param contextItems - Selected context items with pre-formatted markdown.
 * @param envelope - Optional scenario envelope markdown (pass only on first message).
 */
export function composePrompt(message: string, contextItems: AgentContextItem[], envelope?: string): string {
  // Collect screenshot filesystem paths to prepend (Claude Code reads images from paths)
  const screenshotPaths = contextItems
    .flatMap((item) => item.screenshotPaths ?? []);

  let userMessage = message.trim();

  // When actionable context is attached but the user didn't type a meaningful
  // instruction, generate an imperative task directive so the agent acts immediately.
  if (contextItems.length > 0 && userMessage.length < MIN_EXPLICIT_MESSAGE_LENGTH) {
    const directive = buildTaskDirective(contextItems);
    if (directive) {
      userMessage = userMessage ? `${directive}\n\n${userMessage}` : directive;
    }
  }

  let prompt = userMessage;

  // Prepend envelope before the user message (first message only).
  if (envelope) {
    prompt = prompt ? envelope + "\n\n" + prompt : envelope;
  }

  if (contextItems.length > 0) {
    const contextSection = contextItems.map((item) => item.markdown).join("\n---\n\n");
    if (prompt) {
      prompt += "\n\n---\n\n## Attached Context\n\n" + contextSection;
    } else {
      prompt = "## Attached Context\n\n" + contextSection;
    }

    // Append acceptance criteria for actionable context
    prompt += buildAcceptanceCriteria(contextItems);
  }

  // Prepend screenshot file paths so Claude Code can read the images
  if (screenshotPaths.length > 0) {
    prompt = screenshotPaths.join("\n") + "\n\n" + prompt;
  }

  if (prompt.length > MAX_PROMPT_CHARS) {
    prompt = prompt.slice(0, MAX_PROMPT_CHARS) + "\n\n... (context truncated to fit limit)";
  }

  return prompt;
}
