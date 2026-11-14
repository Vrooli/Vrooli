import { describe, it, expect, vi, beforeEach } from "vitest";

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import { RejectBehavior, SpeakerMode } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

const getSpeakerStatusRpc = vi.fn();
const updateSpeakerConfigRpc = vi.fn();
const unbindSpeakerProfileRpc = vi.fn();
const deleteSpeakerProfileRpc = vi.fn();
const enrollSpeakerProfileRpc = vi.fn();
const listSpeakerProfileClipsRpc = vi.fn();
const deleteSpeakerProfileClipRpc = vi.fn();

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    getSpeakerStatus: (req: unknown) => getSpeakerStatusRpc(req),
    updateSpeakerConfig: (req: unknown) => updateSpeakerConfigRpc(req),
    unbindSpeakerProfile: (req: unknown) => unbindSpeakerProfileRpc(req),
    deleteSpeakerProfile: (req: unknown) => deleteSpeakerProfileRpc(req),
    enrollSpeakerProfile: (req: unknown) => enrollSpeakerProfileRpc(req),
    listSpeakerProfileClips: (req: unknown) => listSpeakerProfileClipsRpc(req),
    deleteSpeakerProfileClip: (req: unknown) => deleteSpeakerProfileClipRpc(req),
  }),
}));

import {
  getSpeakerStatus,
  updateSpeakerConfig,
  unbindSpeakerProfile,
  deleteSpeakerProfile,
  enrollSpeakerProfile,
  listSpeakerProfileClips,
  deleteSpeakerProfileClip,
} from "./speakerAdmin";

beforeEach(() => {
  vi.clearAllMocks();
});

const fullConfig = {
  enabled: true,
  profileIds: ["p1"],
  threshold: 0.7,
  mode: SpeakerMode.ADVISORY,
  rejectBehavior: RejectBehavior.SHOW_MUTED,
  fallbackWithoutVerification: true,
  extractionEnabled: true,
  minDecisionSeconds: 2,
  scoreSmoothing: 0.5,
};

describe("getSpeakerStatus", () => {
  it("decodes a populated status with profiles", async () => {
    getSpeakerStatusRpc.mockResolvedValue({
      status: {
        config: fullConfig,
        capability: "speaker_id",
        capabilityLabel: "Speaker ID",
        resourceReady: true,
        profileConfigured: true,
        profileExists: true,
        profileCount: 1,
        profiles: [
          {
            id: "p1",
            displayName: "Alice",
            modelName: "ecapa",
            sampleRate: 16_000,
            clipCount: 3,
            totalVoicedSeconds: 12,
            createdAt: { seconds: 1_700_000_000n, nanos: 0 },
          },
        ],
      },
    });
    const out = await getSpeakerStatus();
    expect(out.config.mode).toBe("advisory");
    expect(out.config.rejectBehavior).toBe("show-muted");
    expect(out.config.enabled).toBe(true);
    expect(out.profiles[0]!.displayName).toBe("Alice");
    expect(out.profiles[0]!.createdAt).toContain("T");
    expect(out.profileCount).toBe(1);
  });

  it("throws when the status is missing", async () => {
    getSpeakerStatusRpc.mockResolvedValue({});
    await expect(getSpeakerStatus()).rejects.toThrow(/missing status/);
  });

  it("falls back to defaults when config is absent", async () => {
    getSpeakerStatusRpc.mockResolvedValue({
      status: {
        capability: "",
        capabilityLabel: "",
        resourceReady: false,
        profileConfigured: false,
        profileExists: false,
        profileCount: 0,
        profiles: [],
      },
    });
    const out = await getSpeakerStatus();
    expect(out.config.enabled).toBe(false);
    expect(out.config.mode).toBe("filter");
    expect(out.config.rejectBehavior).toBe("drop");
    expect(out.config.profileIds).toEqual([]);
  });
});

describe("updateSpeakerConfig", () => {
  it("builds the field mask and encodes label enums for every patch field", async () => {
    updateSpeakerConfigRpc.mockResolvedValue({ config: fullConfig });
    const out = await updateSpeakerConfig({
      enabled: true,
      profileIds: ["p1"],
      threshold: 0.7,
      mode: "off",
      rejectBehavior: "drop",
      fallbackWithoutVerification: false,
      extractionEnabled: true,
      minDecisionSeconds: 1,
      scoreSmoothing: 0.2,
    });
    const req = updateSpeakerConfigRpc.mock.calls[0]![0];
    expect(req.updateMask.paths).toEqual([
      "enabled",
      "profile_ids",
      "threshold",
      "mode",
      "reject_behavior",
      "fallback_without_verification",
      "extraction_enabled",
      "min_decision_seconds",
      "score_smoothing",
    ]);
    expect(req.config.mode).toBe(SpeakerMode.OFF);
    expect(req.config.rejectBehavior).toBe(RejectBehavior.DROP);
    expect(out.mode).toBe("advisory");
  });

  it("sends an empty mask for an empty patch", async () => {
    updateSpeakerConfigRpc.mockResolvedValue({ config: {} });
    await updateSpeakerConfig({});
    expect(updateSpeakerConfigRpc.mock.calls[0]![0].updateMask.paths).toEqual([]);
  });

  it("maps the filter mode label", async () => {
    updateSpeakerConfigRpc.mockResolvedValue({ config: {} });
    await updateSpeakerConfig({ mode: "filter" });
    expect(updateSpeakerConfigRpc.mock.calls[0]![0].config.mode).toBe(SpeakerMode.FILTER);
  });
});

describe("unbind / delete profile", () => {
  it("unbinds and returns the decoded config", async () => {
    unbindSpeakerProfileRpc.mockResolvedValue({ config: fullConfig });
    const out = await unbindSpeakerProfile("p1");
    expect(unbindSpeakerProfileRpc).toHaveBeenCalledWith({ profileId: "p1" });
    expect(out.mode).toBe("advisory");
  });

  it("deletes a profile and returns the decoded config", async () => {
    deleteSpeakerProfileRpc.mockResolvedValue({ config: fullConfig });
    const out = await deleteSpeakerProfile("p1");
    expect(deleteSpeakerProfileRpc).toHaveBeenCalledWith({ profileId: "p1" });
    expect(out.enabled).toBe(true);
  });
});

describe("enrollSpeakerProfile", () => {
  it("encodes a minimal enrollment request and decodes the result", async () => {
    enrollSpeakerProfileRpc.mockResolvedValue({
      enrollment: {
        profileId: "p2",
        clipId: "c1",
        label: "clip 1",
        voicedSeconds: 4,
        clipCount: 1,
        totalVoicedSeconds: 4,
      },
    });
    const out = await enrollSpeakerProfile({
      audio: new Uint8Array([1, 2]),
      format: AudioFormat.WAV,
    });
    const req = enrollSpeakerProfileRpc.mock.calls[0]![0];
    expect(req.profileId).toBe("");
    expect(req.displayName).toBe("");
    expect(req.label).toBe("");
    expect("addToActive" in req).toBe(false);
    expect("enable" in req).toBe(false);
    expect(out.profileId).toBe("p2");
    expect(out.clipCount).toBe(1);
  });

  it("forwards optional flags when provided", async () => {
    enrollSpeakerProfileRpc.mockResolvedValue({ enrollment: {} });
    await enrollSpeakerProfile({
      audio: new Uint8Array(),
      format: AudioFormat.PCM_S16LE,
      profileId: "p3",
      displayName: "Bob",
      label: "morning",
      addToActive: true,
      enable: false,
    });
    const req = enrollSpeakerProfileRpc.mock.calls[0]![0];
    expect(req.profileId).toBe("p3");
    expect(req.displayName).toBe("Bob");
    expect(req.addToActive).toBe(true);
    expect(req.enable).toBe(false);
  });

  it("defaults all result fields when enrollment is absent", async () => {
    enrollSpeakerProfileRpc.mockResolvedValue({});
    const out = await enrollSpeakerProfile({ audio: new Uint8Array(), format: AudioFormat.WAV });
    expect(out).toEqual({
      profileId: "",
      clipId: "",
      label: "",
      voicedSeconds: 0,
      clipCount: 0,
      totalVoicedSeconds: 0,
    });
  });
});

describe("listSpeakerProfileClips", () => {
  it("decodes clips with ISO timestamps", async () => {
    listSpeakerProfileClipsRpc.mockResolvedValue({
      clips: [
        { clipId: "c1", label: "one", voicedSeconds: 3, createdAt: { seconds: 1_700_000_000n, nanos: 0 } },
        { clipId: "c2", label: "two", voicedSeconds: 2 },
      ],
    });
    const out = await listSpeakerProfileClips("p1");
    expect(listSpeakerProfileClipsRpc).toHaveBeenCalledWith({ profileId: "p1" });
    expect(out).toHaveLength(2);
    expect(out[0]!.createdAt).toContain("T");
    expect(out[1]!.createdAt).toBe("");
  });
});

describe("deleteSpeakerProfileClip", () => {
  it("returns the deleted-profile flag and remaining clip count", async () => {
    deleteSpeakerProfileClipRpc.mockResolvedValue({ deletedProfile: true, clipCount: 0 });
    const out = await deleteSpeakerProfileClip("p1", "c1");
    expect(deleteSpeakerProfileClipRpc).toHaveBeenCalledWith({ profileId: "p1", clipId: "c1" });
    expect(out).toEqual({ deletedProfile: true, clipCount: 0 });
  });
});
