import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import { extractTerminalContext } from "./context.js";

const API_BASE = resolveApiBase({ appendSuffix: false })

export async function generateAICommand(
  prompt,
  getActiveTabFn,
  sendKeyToTerminalFn,
) {
  try {
    const activeTab =
      typeof getActiveTabFn === "function" ? getActiveTabFn() : null;
    const context = extractTerminalContext(activeTab);

    const url = buildApiUrl('/api/generate-command', { baseUrl: API_BASE })
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "same-origin",
      body: JSON.stringify({ prompt, context }),
    });

    let data = null;
    const contentType = response.headers.get("Content-Type") || "";
    if (contentType.includes("application/json")) {
      try {
        data = await response.json();
      } catch (_error) {
        data = null;
      }
    } else {
      const text = await response.text();
      if (text) {
        data = { error: text };
      }
    }

    if (!response.ok) {
      const errorMessage =
        data && data.error
          ? data.error
          : `Command generation failed (${response.status})`;
      const normalized =
        typeof errorMessage === "string" ? errorMessage.trim() : "";
      const finalMessage = normalized.includes(
        "requires backend API integration",
      )
        ? "Backend command generation is disabled on this instance. Ensure the web-console API is rebuilt with Ollama support or that resource-ollama is provisioned."
        : normalized || "Command generation failed.";
      return {
        success: false,
        error: finalMessage,
      };
    }

    const command =
      data && typeof data.command === "string" ? data.command.trim() : "";
    if (!command) {
      const errorMessage =
        data && data.error
          ? data.error
          : "Command generation returned no output";
      return {
        success: false,
        error: errorMessage,
      };
    }

    if (typeof sendKeyToTerminalFn === "function") {
      const payload = command.endsWith("\n") ? command : `${command}\n`;
      sendKeyToTerminalFn(payload);
    }

    return {
      success: true,
      command,
    };
  } catch (error) {
    console.error("AI command generation error:", error);
    return {
      success: false,
      error:
        error instanceof Error ? error.message : "Failed to generate command",
    };
  }
}
