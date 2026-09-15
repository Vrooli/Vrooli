import { createClient } from "@connectrpc/connect";
import { TerminalService } from "@vrooli/proto-types/web-console/v1/terminal/terminal_pb";
import { transport } from "./client";

const terminalClient = createClient(TerminalService, transport);

/**
 * The session's current screen as text, decoded by the server's emulator.
 * It is complete whether or not a browser terminal ever rendered the session.
 * Null when the screen cannot be read.
 */
export async function getScreenText(sessionId: string): Promise<string | null> {
  try {
    const screen = await terminalClient.getScreen({ sessionId, includeScrollback: false });
    return screen.plainText;
  } catch {
    return null;
  }
}
