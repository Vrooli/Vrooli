import { boundedRemoteTextInput, type BrowserSurfaceInput } from "./remoteSurface";

/** Typed payload accepted inside BAS Hub's `recording_input` envelope. */
export type BASHubInput =
  | { type: "keyboard"; text: string }
  | { type: "keyboard"; key: string; modifiers?: string[] }
  | { type: "pointer"; action: "move" | "down" | "up"; button: "left" | "middle" | "right"; x: number; y: number; modifiers?: string[] }
  | { type: "wheel"; x: number; y: number; delta_x: number; delta_y: number };

export interface BASHubRecordingInput {
  type: "recording_input";
  session_id: string;
  input: BASHubInput;
}

export interface BASHubInputSocket {
  readyState: number;
  send(data: string): void;
}

export type BrowserPaneInputHandler = (
  input: BrowserSurfaceInput,
  order: number,
  lease: { receipt: string; epoch: bigint },
) => Promise<void>;

/**
 * Convert the presentation DTO into the existing BAS Hub event contract.
 * Keyboard press/release is atomic in the BAS driver, so key-up never emits a
 * second event. The cookie-backed Hub authorizes the session and role.
 */
export function toBASHubRecordingInput(sessionId: string, surfaceInput: BrowserSurfaceInput): BASHubRecordingInput | undefined {
  if (!sessionId.trim()) return undefined;
  let input: BASHubInput;
  switch (surfaceInput.type) {
    case "text":
      input = { type: "keyboard", text: boundedRemoteTextInput(surfaceInput.text) };
      break;
    case "key":
      if (surfaceInput.phase === "up") return undefined;
      if (!surfaceInput.key || surfaceInput.key.length > 128) return undefined;
      input = { type: "keyboard", key: surfaceInput.key, modifiers: surfaceInput.modifiers };
      break;
    case "pointer":
      input = {
        type: "pointer",
        action: surfaceInput.phase,
        button: surfaceInput.button,
        x: surfaceInput.x,
        y: surfaceInput.y,
        modifiers: surfaceInput.modifiers,
      };
      break;
    case "wheel":
      input = {
        type: "wheel",
        x: surfaceInput.x,
        y: surfaceInput.y,
        delta_x: surfaceInput.deltaX,
        delta_y: surfaceInput.deltaY,
      };
      break;
  }
  return { type: "recording_input", session_id: sessionId, input };
}

/** Send through a socket that the owner has already admitted with its role cookie. */
export function sendBASHubInput(socket: BASHubInputSocket, sessionId: string, surfaceInput: BrowserSurfaceInput): boolean {
  const message = toBASHubRecordingInput(sessionId, surfaceInput);
  if (!message || socket.readyState !== 1) return false;
  socket.send(JSON.stringify(message));
  return true;
}

/** Adapter for BrowserPane's ordered input callback over a cookie-admitted Hub socket. */
export function createBASHubInputHandler(socket: BASHubInputSocket, sessionId: string): BrowserPaneInputHandler {
  return async (surfaceInput, _order, _lease) => {
    if (!sendBASHubInput(socket, sessionId, surfaceInput)) {
      throw new Error("The owner-admitted BAS browser stream is not open for this input.");
    }
  };
}
