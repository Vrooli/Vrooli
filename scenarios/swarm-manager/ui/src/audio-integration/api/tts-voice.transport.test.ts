// Actual generated Connect clients and adapters; only browser fetch is a disposable transport boundary.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create, toJson, type DescMessage, type Message } from "@bufbuild/protobuf";
import { Code } from "@connectrpc/connect";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import * as admin from "@vrooli/proto-types/swarm-manager/v1/audio_admin/audio_admin_pb";
import * as runtime from "@vrooli/proto-types/swarm-manager/v1/audio_runtime/audio_runtime_pb";
import { ResponseFormat, SpeakerMode, RejectBehavior, SpeakerCapability, SummarizeLevel } from "@vrooli/proto-types/swarm-manager/v1/audio_common/audio_common_pb";
import * as tts from "./tts";
import * as voice from "./voice";

type Call = { method: string; body: unknown; request: Request };
let calls: Call[];
let replies: Map<string, (request: Request) => Response | Promise<Response>>;
let unexpected: string[];
function json(schema: DescMessage, message: Message) { return new Response(JSON.stringify(toJson(schema, message)), { headers: { "content-type": "application/json" } }); }
function deny() { return new Response(JSON.stringify({ code: "permission_denied", message: "Owner refused" }), { status: 403, headers: { "content-type": "application/json" } }); }
function returned(method: string, schema: DescMessage, message: Message) { replies.set(method, () => json(schema, message)); }
function bodies(method: string) { return calls.filter(c => c.method === method).map(c => c.body); }
function clip() {
  const bytes = new Uint8Array([1, 2, 3]); const blob = new Blob([bytes], { type: "audio/webm;codecs=opus" });
  // jsdom's Blob lacks the browser arrayBuffer method. This supplies only that platform method.
  Object.defineProperty(blob, "arrayBuffer", { value: async () => bytes.buffer }); return blob;
}
// Missing-only browser platform affordance. This follows first-aborted input
// order/reason and removes every attached listener after settlement. It is not
// a production compatibility patch or a claim about an installed browser.
let signalAnyDescriptor: PropertyDescriptor | undefined;
let installedSignalAny = false;
const signalAnyCleanups = new Set<() => void>();
function installSignalAnyAffordance() {
  signalAnyDescriptor = Object.getOwnPropertyDescriptor(AbortSignal, "any");
  installedSignalAny = typeof AbortSignal.any !== "function";
  if (!installedSignalAny) return;
  Object.defineProperty(AbortSignal, "any", { configurable: true, writable: true, value(signals: Iterable<AbortSignal>) {
    const inputs = [...signals];
    if (inputs.some(input => !(input instanceof AbortSignal))) throw new TypeError("Expected AbortSignal inputs");
    const controller = new AbortController();
    const alreadyAborted = inputs.find(input => input.aborted);
    if (alreadyAborted) { controller.abort(alreadyAborted.reason); return controller.signal; }
    const listeners = new Map<AbortSignal, () => void>();
    const remove = () => { for (const [signal, listener] of listeners) signal.removeEventListener("abort", listener); listeners.clear(); signalAnyCleanups.delete(remove); };
    signalAnyCleanups.add(remove);
    for (const input of new Set(inputs)) {
      const listener = () => { if (!controller.signal.aborted) controller.abort(input.reason); remove(); };
      listeners.set(input, listener); input.addEventListener("abort", listener, { once: true });
    }
    return controller.signal;
  } });
}
function restoreSignalAnyAffordance() {
  for (const cleanup of [...signalAnyCleanups]) cleanup();
  if (installedSignalAny) {
    if (signalAnyDescriptor) Object.defineProperty(AbortSignal, "any", signalAnyDescriptor);
    else Reflect.deleteProperty(AbortSignal, "any");
  }
  installedSignalAny = false;
}
beforeEach(() => {
  calls = []; replies = new Map(); unexpected = []; installSignalAnyAffordance();
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = new Request(input, init); const path = new URL(request.url).pathname;
    const match = path.match(/\/vrooli\.swarm_manager\.v1\.audio_(?:admin\.AudioAdminService|runtime\.AudioRuntimeService)\/(\w+)$/);
    const method = match?.[1];
    if (!method || !replies.has(method)) { unexpected.push(path); throw new Error(`Unexpected audio request ${path}`); }
    // A real browser fetch rejects an already-aborted request before delivery.
    if (request.signal.aborted) throw request.signal.reason;
    calls.push({ method, body: JSON.parse(await request.text()), request }); return replies.get(method)!(request);
  }));
});
afterEach(() => { restoreSignalAnyAffordance(); expect(unexpected).toEqual([]); vi.unstubAllGlobals(); vi.restoreAllMocks(); });

describe("TTS actual adapter transport", () => {
  it("synthesizes exact authored voice/speed and correlates synthesis and play timing", async () => {
    returned("Synthesize", runtime.SynthesizeResponseSchema, create(runtime.SynthesizeResponseSchema, { audio: new Uint8Array([4, 5]), contentType: "audio/wav" }));
    returned("RecordPlaybackEvent", runtime.RecordPlaybackEventResponseSchema, create(runtime.RecordPlaybackEventResponseSchema));
    const result = await tts.synthesizeTTSWithMetrics("Owner narration", "owner-voice", 1.25);
    expect(bodies("Synthesize")).toEqual([{ text: "Owner narration", voice: "owner-voice", responseFormat: "RESPONSE_FORMAT_MP3", speed: 1.25 }]);
    expect(result.blob.size).toBe(2); expect(result.blob.type).toBe("audio/wav"); expect(result.metrics.totalChars).toBe(15);
    expect(calls[0]!.request.headers.get("x-tts-request-id")).toBe(result.metrics.requestId);
    tts.reportTTSPlayStart(result.metrics); await vi.waitFor(() => expect(bodies("RecordPlaybackEvent")).toHaveLength(2));
    const events = bodies("RecordPlaybackEvent") as { event: { source: string; stage: string; message: string } }[];
    expect(events.map(e => [e.event.source, e.event.stage])).toEqual([["audio-integration", "timing"], ["audio-integration", "timing"]]);
    expect(JSON.parse(events[0]!.event.message)).toMatchObject({ requestId: result.metrics.requestId, totalChars: 15, chunkCount: 1 });
    expect(JSON.parse(events[1]!.event.message)).toMatchObject({ requestId: result.metrics.requestId, playStartMs: expect.any(Number) });
  });
  it("preserves synthesis refusal despite rejected best-effort telemetry and never retries synthesis", async () => {
    replies.set("Synthesize", deny); replies.set("RecordPlaybackEvent", deny);
    await expect(tts.synthesizeTTS("Denied narration")).rejects.toThrow("Owner refused");
    await vi.waitFor(() => expect(bodies("RecordPlaybackEvent")).toHaveLength(1)); expect(bodies("Synthesize")).toHaveLength(1);
    const event = bodies("RecordPlaybackEvent")[0] as {event:{message:string}}; expect(JSON.parse(event.event.message)).toMatchObject({ totalChars: 16, chunkCount: 0, error: "ConnectError" });
  });
  it("returns a default-format synthesized blob through the convenience API", async () => {
    returned("Synthesize", runtime.SynthesizeResponseSchema, create(runtime.SynthesizeResponseSchema, { audio: new Uint8Array([9]) }));
    returned("RecordPlaybackEvent", runtime.RecordPlaybackEventResponseSchema, create(runtime.RecordPlaybackEventResponseSchema));
    const blob = await tts.synthesizeTTS("One"); expect(blob.type).toBe("audio/mpeg"); expect(blob.size).toBe(1);
    expect(bodies("Synthesize")).toEqual([{ text: "One", responseFormat: "RESPONSE_FORMAT_MP3" }]);
    await vi.waitFor(() => expect(bodies("RecordPlaybackEvent")).toHaveLength(1));
  });
  it("requests the exact original cache identity and preserves returned audio format", async () => {
    returned("GetTTSCache", runtime.GetTTSCacheResponseSchema, create(runtime.GetTTSCacheResponseSchema, {hit:true,audio:new Uint8Array([7]),contentType:"audio/ogg"}));
    const blob = await tts.fetchCachedTTS("event-owner", "voice-owner", 0.8, "original"); expect(blob?.size).toBe(1); expect(blob?.type).toBe("audio/ogg");
    expect(bodies("GetTTSCache")).toEqual([{ eventId:"event-owner",voice:"voice-owner",speed:0.8,version:"original" }]);
  });
  it.each(["miss", "empty", "refused"])("returns no cache audio on %s without synthesis fallback", async kind => {
    replies.set("GetTTSCache", kind === "refused" ? deny : () => json(runtime.GetTTSCacheResponseSchema, create(runtime.GetTTSCacheResponseSchema, {hit:kind === "empty"})));
    expect(await tts.fetchCachedTTS("owner", "voice", 1)).toBeNull(); expect(calls.map(c => c.method)).toEqual(["GetTTSCache"]);
  });
  it("decodes missing TTS config conservatively and preserves explicit false/zero updates", async () => {
    returned("GetTTSConfig", admin.GetTTSConfigResponseSchema, create(admin.GetTTSConfigResponseSchema));
    expect(await tts.getTTSConfig()).toEqual({autoEnabled:false,defaultVoice:"",defaultSpeed:1,defaultResponseFormat:"mp3"});
    returned("UpdateTTSConfig", admin.UpdateTTSConfigResponseSchema, create(admin.UpdateTTSConfigResponseSchema,{config:{autoEnabled:false,defaultVoice:"owner",defaultSpeed:0,defaultResponseFormat:ResponseFormat.WAV}}));
    expect(await tts.updateTTSConfig({autoEnabled:false,defaultVoice:"owner",defaultSpeed:0,defaultResponseFormat:"wav"})).toEqual({autoEnabled:false,defaultVoice:"owner",defaultSpeed:0,defaultResponseFormat:"wav"});
    expect(bodies("UpdateTTSConfig")).toEqual([{updateMask:"autoEnabled,defaultVoice,defaultSpeed,defaultResponseFormat",config:{defaultVoice:"owner",defaultResponseFormat:"RESPONSE_FORMAT_WAV"}}]);
  });
  it.each([[SummarizeLevel.LIGHT,"light"],[SummarizeLevel.MODERATE,"moderate"],[SummarizeLevel.HEAVY,"heavy"],[SummarizeLevel.UNSPECIFIED,"moderate"]] as const)("decodes summarize level %s", async (level,label) => {
    returned("GetSummarizeConfig",admin.GetSummarizeConfigResponseSchema,create(admin.GetSummarizeConfigResponseSchema,{config:{enabled:true,charThreshold:80,level,model:"owner-model",timeoutSeconds:9}}));
    expect(await tts.getTTSSummarizeConfig()).toEqual({enabled:true,charThreshold:80,level:label,model:"owner-model",timeoutSeconds:9});
  });
  it("updates exact summarization fields and decodes model catalog identity and int64 size", async () => {
    returned("UpdateSummarizeConfig",admin.UpdateSummarizeConfigResponseSchema,create(admin.UpdateSummarizeConfigResponseSchema,{config:{level:SummarizeLevel.HEAVY,model:"owner-model"}}));
    expect(await tts.updateTTSSummarizeConfig({enabled:false,charThreshold:0,level:"heavy",model:"owner-model",timeoutSeconds:0})).toMatchObject({enabled:false,charThreshold:0,level:"heavy",model:"owner-model",timeoutSeconds:0});
    expect(bodies("UpdateSummarizeConfig")).toEqual([{updateMask:"enabled,charThreshold,level,model,timeoutSeconds",config:{level:"SUMMARIZE_LEVEL_HEAVY",model:"owner-model"}}]);
    returned("ListSummarizeModels",admin.ListSummarizeModelsResponseSchema,create(admin.ListSummarizeModelsResponseSchema,{models:[{id:"owner-model",sizeBytes:9007199254740993n,installed:true,recommended:true,defaultEligible:true,reasoning:false,statusLabel:"Ready",pullCommand:"pull owner-model",parameterSize:"7B",sourceUrl:"https://example.invalid/model",notes:"Owner model"}]}));
    expect(await tts.listTTSSummarizeModels()).toEqual([{id:"owner-model",displayName:"owner-model",sizeBytes:9007199254740993n,installed:true,recommended:true,defaultEligible:true,reasoning:false,statusLabel:"Ready",pullCommand:"pull owner-model",parameterSize:"7B",sourceUrl:"https://example.invalid/model",notes:"Owner model"}]);
  });
  it("sends exact playback attribution and propagates owner refusal without alternate delivery", async () => {
    replies.set("RecordPlaybackEvent",deny); await expect(tts.reportTTSEvent({source:"owner-player",stage:"play",backend:"kokoro",sessionId:"session-owner",message:"Audible"})).rejects.toThrow("Owner refused");
    expect(bodies("RecordPlaybackEvent")).toEqual([{event:{source:"owner-player",stage:"play",backend:"kokoro",sessionId:"session-owner",message:"Audible"}}]);
  });
});

describe("Voice actual adapter transport", () => {
  it("transcribes original bytes and language, with filter bypass only when explicitly selected", async () => {
    returned("Transcribe",runtime.TranscribeResponseSchema,create(runtime.TranscribeResponseSchema,{text:"Owner transcript"}));
    expect(await voice.transcribeAudio(clip(),"fr")).toBe("Owner transcript"); expect(await voice.transcribeAudioBypassFilter(clip(),"fr")).toBe("Owner transcript");
    expect(bodies("Transcribe")).toEqual([{audio:"AQID",format:"AUDIO_FORMAT_WEBM",language:"fr"},{audio:"AQID",format:"AUDIO_FORMAT_WEBM",language:"fr",skipSpeakerVerification:true}]);
  });
  it("retries the same transcription contract once and retains the final refusal at the bound", async () => {
    let attempt = 0; replies.set("Transcribe",() => ++attempt === 1 ? deny() : json(runtime.TranscribeResponseSchema,create(runtime.TranscribeResponseSchema,{text:"Recovered"})));
    expect(await voice.transcribeAudioWithRetry(clip(),2,"en")).toBe("Recovered"); expect(bodies("Transcribe")).toEqual([{audio:"AQID",format:"AUDIO_FORMAT_WEBM",language:"en"},{audio:"AQID",format:"AUDIO_FORMAT_WEBM",language:"en"}]);
    calls=[]; replies.set("Transcribe",deny); await expect(voice.transcribeAudioWithRetry(clip(),1,"en")).rejects.toThrow("Owner refused"); expect(calls).toHaveLength(1);
  });
  it("preserves the speaker configuration mask for explicit false, empty profiles, and zero threshold", async () => {
    returned("UpdateSpeakerConfig",admin.UpdateSpeakerConfigResponseSchema,create(admin.UpdateSpeakerConfigResponseSchema,{config:{mode:SpeakerMode.FILTER,rejectBehavior:RejectBehavior.SHOW_MUTED}}));
    expect(await voice.updateSpeakerVerificationConfig({enabled:false,profileIds:[],threshold:0,mode:"filter",rejectBehavior:"show-muted",fallbackWithoutVerification:false})).toEqual({enabled:false,profileIds:[],threshold:0,mode:"filter",rejectBehavior:"show-muted",fallbackWithoutVerification:false});
    expect(bodies("UpdateSpeakerConfig")).toEqual([{updateMask:"enabled,profileIds,threshold,mode,rejectBehavior,fallbackWithoutVerification",config:{mode:"SPEAKER_MODE_FILTER",rejectBehavior:"REJECT_BEHAVIOR_SHOW_MUTED"}}]);
  });
  it("decodes missing config and refuses missing status rather than inventing a ready resource", async () => {
    returned("GetSpeakerConfig",admin.GetSpeakerConfigResponseSchema,create(admin.GetSpeakerConfigResponseSchema));
    expect(await voice.getSpeakerVerificationConfig()).toMatchObject({enabled:false,profileIds:[],threshold:0,fallbackWithoutVerification:false});
    returned("GetSpeakerStatus",admin.GetSpeakerStatusResponseSchema,create(admin.GetSpeakerStatusResponseSchema));
    await expect(voice.getSpeakerVerificationStatus()).rejects.toThrow("speaker status response missing status field"); expect(calls.map(c=>c.method)).toEqual(["GetSpeakerConfig","GetSpeakerStatus"]);
  });
  it("projects exact owner profile and resource status including timestamps", async () => {
    const stamp=timestampFromDate(new Date("2026-10-05T00:00:00Z")); const profile={id:"owner-profile",displayName:"Owner",createdAt:stamp,updatedAt:stamp,modelName:"owner-model",embeddingDim:16,sampleRate:16000,enrollmentAudioSeconds:3,notes:"Exact profile"};
    returned("ListSpeakerProfiles",admin.ListSpeakerProfilesResponseSchema,create(admin.ListSpeakerProfilesResponseSchema,{profiles:[profile]}));
    const profiles=await voice.listSpeakerVerificationProfiles(); expect(profiles).toEqual([{id:"owner-profile",display_name:"Owner",created_at:"2026-10-05T00:00:00.000Z",updated_at:"2026-10-05T00:00:00.000Z",model_name:"owner-model",embedding_dim:16,sample_rate:16000,enrollment_audio_seconds:3,notes:"Exact profile"}]);
    returned("GetSpeakerStatus",admin.GetSpeakerStatusResponseSchema,create(admin.GetSpeakerStatusResponseSchema,{status:{capability:SpeakerCapability.AVAILABLE,resourceReady:true,profileConfigured:true,profileExists:true,profileCount:1,profiles:[profile],checkedAt:stamp,info:{backend:"owner-backend",model:"owner-model",device:"cpu",sampleRate:16000,version:"1",embeddingDim:16}}}));
    expect(await voice.getSpeakerVerificationStatus()).toMatchObject({capability:"available",resourceReady:true,profileConfigured:true,profileExists:true,profileCount:1,profiles,checkedAt:"2026-10-05T00:00:00.000Z",info:{backend:"owner-backend",model:"owner-model",device:"cpu",sample_rate:16000,version:"1",embedding_dim:16}});
  });
  it("enrolls exact selected bytes and explicit opt-out flags then projects the owner receipt", async () => {
    returned("EnrollSpeakerProfile",admin.EnrollSpeakerProfileResponseSchema,create(admin.EnrollSpeakerProfileResponseSchema,{enrollment:{profileId:"owner-profile",displayName:"Owner",embeddingDim:16,sampleRate:16000,enrollmentAudioSeconds:3,modelName:"owner-model"},config:{profileIds:["owner-profile"]}}));
    const receipt=await voice.enrollSpeakerVerificationProfile({audioBlob:clip(),profileId:"owner-profile",displayName:"Owner",notes:"Owner supplied",addToActive:false,enable:false});
    expect(bodies("EnrollSpeakerProfile")).toEqual([{audio:"AQID",format:"AUDIO_FORMAT_WEBM",profileId:"owner-profile",displayName:"Owner",notes:"Owner supplied",addToActive:false,enable:false}]);
    expect(receipt.enrollment).toMatchObject({profile_id:"owner-profile",display_name:"Owner",embedding_dim:16,sample_rate:16000,enrollment_audio_seconds:3,model_name:"owner-model"}); expect(receipt.config.profileIds).toEqual(["owner-profile"]);
  });
  it("keeps clear, unbind, delete, and wake-word reset as distinct exact owner operations", async () => {
    returned("ClearSpeakerProfileBinding",admin.ClearSpeakerProfileBindingResponseSchema,create(admin.ClearSpeakerProfileBindingResponseSchema));
    returned("UnbindSpeakerProfile",admin.UnbindSpeakerProfileResponseSchema,create(admin.UnbindSpeakerProfileResponseSchema));
    returned("DeleteSpeakerProfile",admin.DeleteSpeakerProfileResponseSchema,create(admin.DeleteSpeakerProfileResponseSchema));
    returned("DeleteWakeWordTemplate",admin.DeleteWakeWordTemplateResponseSchema,create(admin.DeleteWakeWordTemplateResponseSchema));
    await voice.clearSpeakerVerificationProfile(); await voice.removeSpeakerVerificationProfile("owner-profile"); await voice.deleteSpeakerVerificationProfile("owner-profile");
    expect(await voice.deleteWakeWordConfig()).toEqual({configured:false,template:null});
    expect(calls.map(c=>[c.method,c.body])).toEqual([["ClearSpeakerProfileBinding",{}],["UnbindSpeakerProfile",{profileId:"owner-profile"}],["DeleteSpeakerProfile",{profileId:"owner-profile"}],["DeleteWakeWordTemplate",{}]]);
  });
});

describe("Optional caller cancellation through actual Connect transport", () => {
  it("keeps an empty input set non-aborted and restores the exact original static descriptor", () => {
    expect(AbortSignal.any([]).aborted).toBe(false);
    const pending = new AbortController();
    const remove = vi.spyOn(pending.signal, "removeEventListener");
    AbortSignal.any([pending.signal]);
    const fixtureInstalled = installedSignalAny;
    const originalDescriptor = signalAnyDescriptor;
    restoreSignalAnyAffordance();
    expect(Object.getOwnPropertyDescriptor(AbortSignal, "any")).toEqual(originalDescriptor);
    if (fixtureInstalled) expect(remove).toHaveBeenCalledWith("abort", expect.any(Function));
    expect(signalAnyCleanups.size).toBe(0);
    installSignalAnyAffordance();
  });

  it("uses the first already-aborted input reason and leaves original signals unchanged", () => {
    const first=new AbortController(), second=new AbortController(); const firstReason=new DOMException("First owner cancellation","AbortError"),secondReason=new DOMException("Second owner cancellation","AbortError"); first.abort(firstReason);second.abort(secondReason);
    const combined=AbortSignal.any([first.signal,second.signal]); expect(combined.aborted).toBe(true); expect(combined.reason).toBe(firstReason); expect(first.signal.reason).toBe(firstReason); expect(second.signal.reason).toBe(secondReason);
  });
  it("propagates only the first future reason and cleans the input listeners", () => {
    const first=new AbortController(), second=new AbortController(); const firstRemove=vi.spyOn(first.signal,"removeEventListener"),secondRemove=vi.spyOn(second.signal,"removeEventListener"); const combined=AbortSignal.any([first.signal,second.signal]); const listener=vi.fn();combined.addEventListener("abort",listener);
    const reason=new DOMException("Owner cancelled pending audio","AbortError");second.abort(reason);first.abort(new DOMException("Late cancellation","AbortError")); expect(combined.reason).toBe(reason);expect(listener).toHaveBeenCalledOnce();
    if(installedSignalAny) {expect(firstRemove).toHaveBeenCalledWith("abort",expect.any(Function));expect(secondRemove).toHaveBeenCalledWith("abort",expect.any(Function));}
  });
  it("refuses already-cancelled synthesis without delivery, preserving the original caller signal and reason", async () => {
    const controller=new AbortController();const originalSignal=controller.signal;const reason=new Error("Owner withdrew before send");reason.name="AbortError";controller.abort(reason);
    replies.set("Synthesize",()=>{throw new Error("Cancelled request must not reach synthesis transport");}); returned("RecordPlaybackEvent",runtime.RecordPlaybackEventResponseSchema,create(runtime.RecordPlaybackEventResponseSchema));
    await expect(tts.synthesizeTTSWithMetrics("Cancelled narration","owner-voice",1,controller.signal)).rejects.toMatchObject({code:Code.Canceled});await vi.waitFor(()=>expect(bodies("RecordPlaybackEvent")).toHaveLength(1));expect(bodies("Synthesize")).toEqual([]);expect(controller.signal).toBe(originalSignal);expect(controller.signal.reason).toBe(reason);
  });
  it("aborts a pending actual synthesis request with original reason, no retry, and ignores its late response", async () => {
    const controller=new AbortController();const originalSignal=controller.signal;let received!:Request;let late!: (r:Response)=>void;
    replies.set("Synthesize",request=>new Promise((resolve,reject)=>{received=request;late=resolve;request.signal.addEventListener("abort",()=>reject(request.signal.reason),{once:true});}));returned("RecordPlaybackEvent",runtime.RecordPlaybackEventResponseSchema,create(runtime.RecordPlaybackEventResponseSchema));
    const result=tts.synthesizeTTSWithMetrics("Pending owner narration","owner-voice",1,controller.signal); const rejected=expect(result).rejects.toMatchObject({code:Code.Canceled});await vi.waitFor(()=>expect(bodies("Synthesize")).toHaveLength(1));const reason=new Error("Owner cancelled pending narration");reason.name="AbortError";controller.abort(reason);await rejected;
    expect(received.signal.aborted).toBe(true);expect(received.signal.reason).toBe(reason);expect(controller.signal).toBe(originalSignal);await vi.waitFor(()=>expect(bodies("RecordPlaybackEvent")).toHaveLength(1));
    late(json(runtime.SynthesizeResponseSchema,create(runtime.SynthesizeResponseSchema,{audio:new Uint8Array([9])})));await Promise.resolve();expect(bodies("Synthesize")).toHaveLength(1);const events=bodies("RecordPlaybackEvent") as {event:{message:string}}[];expect(JSON.parse(events[0]!.event.message)).toMatchObject({chunkCount:0,error:"ConnectError"});
  });
});
