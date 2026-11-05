import { act, renderHook, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { JobSchema } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";
import { afterEach, expect, it, vi } from "vitest";
import { JobState } from "../../api/jobs";
import { useObjectMask, type ObjectMaskClient } from "./useObjectMask";

const image = new File(["source"],"device.png",{type:"image/png"});
const accepted={jobId:"mask-job",modelId:"sam-2.1-tiny",tier:"local-cpu",warnings:[],estimatedSeconds:2};
const makeClient=(over:Partial<ObjectMaskClient>={}):ObjectMaskClient=>({
 submit:vi.fn().mockResolvedValue(accepted),
 wait:vi.fn().mockResolvedValue(create(JobSchema,{id:accepted.jobId,state:JobState.SUCCEEDED,resultRef:"mask/output.png"})),
 cancel:vi.fn().mockResolvedValue(undefined),blobUrl:(ref)=>`/blob/${ref}`,...over,
});
afterEach(()=>vi.clearAllMocks());

it("retains include/exclude refinement and shows only the completed native mask",async()=>{
 const client=makeClient();const {result}=renderHook(()=>useObjectMask(image,client));
 act(()=>result.current.addPoint({x:.5,y:.4,negative:false}));
 act(()=>result.current.addPoint({x:.1,y:.2,negative:true}));
 await act(async()=>result.current.run());
 expect(client.submit).toHaveBeenCalledWith("segment",expect.objectContaining({segmentation:{points:[{x:.5,y:.4,negative:false},{x:.1,y:.2,negative:true}]}}),expect.objectContaining({image,operationId:expect.any(String)}));
 expect(client.wait).toHaveBeenCalledTimes(1);
 expect(result.current.maskUrl).toBe("/blob/mask/output.png");
 act(()=>result.current.undo());
 expect(result.current.points).toHaveLength(1);
 expect(result.current.maskUrl).toBe("");
});

it("reattaches a lost admission response with the same identity, then invalidates on refinement",async()=>{
 const client=makeClient();vi.mocked(client.submit).mockRejectedValueOnce(new Error("connection lost"));
 const {result}=renderHook(()=>useObjectMask(image,client));
 act(()=>result.current.addPoint({x:.5,y:.4,negative:false}));
 await act(async()=>result.current.run());
 expect(result.current.error).toContain("connection lost");
 await act(async()=>result.current.run());
 const calls=vi.mocked(client.submit).mock.calls;
 expect(calls[0]?.[2]?.operationId).toBe(calls[1]?.[2]?.operationId);
 act(()=>result.current.addPoint({x:.1,y:.2,negative:true}));
 await act(async()=>result.current.run());
 expect(calls[2]?.[2]?.operationId).not.toBe(calls[1]?.[2]?.operationId);
});

it("cancels an admission that arrives after the user cancelled, without showing its result",async()=>{
 let resolve!: (value:typeof accepted)=>void;
 const client=makeClient({submit:vi.fn().mockImplementation(()=>new Promise(r=>{resolve=r;}))});
 const {result}=renderHook(()=>useObjectMask(image,client));
 act(()=>result.current.addPoint({x:.5,y:.4,negative:false}));
 let pending!:Promise<void>;act(()=>{pending=result.current.run();});
 act(()=>result.current.cancel());
 await act(async()=>{resolve(accepted);await pending;});
 expect(client.cancel).toHaveBeenCalledWith("mask-job");
 expect(client.wait).not.toHaveBeenCalled();
 expect(result.current.maskUrl).toBe("");
 expect(result.current.busy).toBe(false);
});

it("reports cancellation failure instead of promising inference stopped",async()=>{
 const client=makeClient({wait:vi.fn().mockImplementation((_id,signal:AbortSignal)=>new Promise((_resolve,reject)=>signal.addEventListener("abort",()=>reject(new Error("aborted"))))),cancel:vi.fn().mockRejectedValue(new Error("owner unavailable"))});
 const {result}=renderHook(()=>useObjectMask(image,client));
 act(()=>result.current.addPoint({x:.5,y:.4,negative:false}));
 act(()=>{void result.current.run();});
 await waitFor(()=>expect(result.current.jobId).toBe("mask-job"));
 act(()=>result.current.cancel());
 await waitFor(()=>expect(result.current.cancelError).toBe("owner unavailable"));
 expect(result.current.error).toBe("");
});
