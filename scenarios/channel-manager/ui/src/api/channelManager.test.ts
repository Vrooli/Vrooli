import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { completeAction, createIdentity, enqueueAction, overview, previewRelease, recordObservation, startProgram } from "./channelManager";

describe("channel-manager API", () => {
	const fetchSpy = vi.fn();

	beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
	afterEach(() => vi.unstubAllGlobals());

	it("uses the typed overview and operator action contracts", async () => {
		fetchSpy.mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ identities: {}, actions: {} }), { status: 200 })));
		await overview();
		await createIdentity({ id: "identity-1", platform_id: "x", purpose: "brand", environment_ref: "environment-1", vault_ref: "vault://identity-1", status: "draft", attestations: { region_locked: true } });
		await startProgram("identity-1", "program-1");
		await enqueueAction("identity-1", "engage");
		await completeAction("action-1", "evidence");
		await recordObservation("identity-1", 3);
		await previewRelease({ platform_id: "x", caption: "caption", format_kind: "image", media_width: 1200, media_height: 1200, disclosure_visible: true, first_comment: "" });
		expect(fetchSpy).toHaveBeenCalledTimes(7);
		expect(fetchSpy.mock.calls[0][1]).toMatchObject({ method: "GET" });
		expect(fetchSpy.mock.calls[1][1]).toMatchObject({ method: "POST" });
		expect(fetchSpy.mock.calls[6][0]).toContain("/releases/preview");
	});

	it("rejects a non-successful operator request", async () => {
		fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ message: "not available" }), { status: 503 }));
		await expect(overview()).rejects.toThrow("not available");
	});
});
