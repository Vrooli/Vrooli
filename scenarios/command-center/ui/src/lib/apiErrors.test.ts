import { describe, expect, it } from "vitest";
import { apiRequestError } from "./apiErrors";

describe("API request errors", () => {
  it("retains a stable server code for recovery decisions", async () => {
    const response = new Response(JSON.stringify({ error: "board_settings_invalid", message: "settings need repair" }), {
      status: 422,
      statusText: "Unprocessable Entity",
      headers: { "Content-Type": "application/json" },
    });
    const error = await apiRequestError(response, "API request failed");
    expect(error.name).toBe("ApiRequestError");
    expect(error.message).toBe("API request failed: 422 Unprocessable Entity");
    expect(error.status).toBe(422);
    expect(error.code).toBe("board_settings_invalid");
  });

  it("keeps a useful status error when the response body is not JSON", async () => {
    const error = await apiRequestError(new Response("offline", { status: 503 }), "API request failed");
    expect(error.status).toBe(503);
    expect(error.code).toBeUndefined();
  });
});
