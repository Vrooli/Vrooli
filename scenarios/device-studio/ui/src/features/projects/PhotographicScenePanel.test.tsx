import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { makeApiError } from "../../api/client";
import { projectsClient, uploadSourceMedia } from "../../api/projects";
import { cancelPhotographicOperation, getPhotographicPNG, readPhotographicOperation, submitPhotographicScene, waitPhotographicOperation } from "../../api/photographic-scene";
import { PhotographicScenePanel } from "./PhotographicScenePanel";

vi.mock("../../api/projects", () => ({ projectsClient: { getSourceMedia: vi.fn() }, uploadSourceMedia: vi.fn() }));
vi.mock("../../api/photographic-scene", () => ({ cancelPhotographicOperation: vi.fn(), getPhotographicPNG: vi.fn(), readPhotographicOperation: vi.fn(), submitPhotographicScene: vi.fn(), waitPhotographicOperation: vi.fn() }));
const queued = { job_id:"job-1",operation:"text_to_image",state:"queued",progress:0 };
describe("PhotographicScenePanel",()=>{
  beforeEach(async()=>{await setLocale("en");localStorage.clear();vi.mocked(readPhotographicOperation).mockResolvedValue(queued);vi.mocked(submitPhotographicScene).mockResolvedValue(queued);vi.mocked(waitPhotographicOperation).mockResolvedValue({...queued,state:"succeeded",progress:100,result_ref:"out/image.png"});vi.mocked(cancelPhotographicOperation).mockResolvedValue({...queued,state:"canceled"});});
  afterEach(()=>{cleanup();vi.clearAllMocks();});

  it("persists the exact operation before submission and explicitly retries the same key after reload",async()=>{
    const user=userEvent.setup();
    vi.mocked(submitPhotographicScene).mockImplementationOnce(async(projectId,request)=>{const saved=JSON.parse(localStorage.getItem(`device-studio:photographic-scene:v1:${projectId}`)!);expect(saved.operation_id).toBe(request.operation_id);expect(saved.prompt).toBe(request.prompt);throw new Error("lost response");});
    const first=renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.type(await screen.findByLabelText("Describe the scene"),"A sunlit modern reading room");
    await user.click(screen.getByRole("button",{name:"Generate one local image"}));
    expect(await screen.findByRole("alert")).toHaveTextContent("submit outcome is unknown");
    const stored=JSON.parse(localStorage.getItem("device-studio:photographic-scene:v1:p1")!);
    first.unmount();
    vi.mocked(readPhotographicOperation).mockClear().mockRejectedValueOnce(new Error("operation not found"));
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    expect(await screen.findByRole("alert")).toHaveTextContent("submit outcome is unknown");
    expect(readPhotographicOperation).toHaveBeenCalledWith("p1",stored.operation_id);
    expect(submitPhotographicScene).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("button",{name:"Retry the same saved operation"}));
    await waitFor(()=>expect(submitPhotographicScene).toHaveBeenCalledTimes(2));
    expect(submitPhotographicScene).toHaveBeenLastCalledWith("p1",expect.objectContaining({operation_id:stored.operation_id,prompt:"A sunlit modern reading room",width:768,height:512}));
  });

  it("unlocks editing after a definitive pre-acceptance rejection",async()=>{
    const user=userEvent.setup();
    vi.mocked(submitPhotographicScene).mockRejectedValueOnce(makeApiError("invalid_request","unsupported local request",422));
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.type(await screen.findByLabelText("Describe the scene"),"A quiet room");
    await user.click(screen.getByRole("button",{name:"Generate one local image"}));
    expect(await screen.findByRole("status")).toHaveTextContent("Image Tools rejected this request before accepting a job.");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Describe the scene")).toHaveValue("A quiet room");
    expect(localStorage.getItem("device-studio:photographic-scene:v1:p1")).toBeNull();
  });

  it("imports only the successful owner PNG and shows saved source provenance",async()=>{
    const user=userEvent.setup();
    vi.mocked(submitPhotographicScene).mockResolvedValueOnce({...queued,state:"succeeded",progress:100,result_ref:"out/image.png"});
    vi.mocked(getPhotographicPNG).mockResolvedValueOnce(new Blob([new Uint8Array([1,2,3])],{type:"image/png"}));
    vi.mocked(uploadSourceMedia).mockResolvedValueOnce({id:"media-1",sha256:"a".repeat(64)} as never);
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.type(await screen.findByLabelText("Describe the scene"),"A sunlit modern reading room");
    await user.click(screen.getByRole("button",{name:"Generate one local image"}));
    await user.click(await screen.findByRole("button",{name:"Import generated image into this project"}));
    await screen.findByText(/Imported source media-1/);
    expect(getPhotographicPNG).toHaveBeenCalledWith("p1",expect.any(String),768,512);
    expect(uploadSourceMedia).toHaveBeenCalledWith("p1",expect.objectContaining({name:"device-studio-generated-job-1.png",type:"image/png"}));
    expect(projectsClient.getSourceMedia).not.toHaveBeenCalled();
  });

  it("reads an accepted job after reload without resubmitting it",async()=>{
    const saved={version:1,operation_id:"f7f4f09f-a93a-4f34-93ef-4fa814891f5b",prompt:"A quiet studio",width:768,height:512};
    localStorage.setItem("device-studio:photographic-scene:v1:p1",JSON.stringify(saved));
    vi.mocked(readPhotographicOperation).mockResolvedValueOnce(queued);
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    expect(await screen.findByText(/Job job-1/)).toBeInTheDocument();
    expect(submitPhotographicScene).not.toHaveBeenCalled();
    expect(screen.queryByRole("button",{name:"Retry the same saved operation"})).not.toBeInTheDocument();
  });

  it("keeps cancel available during wait and ignores the older wait result",async()=>{
    const user=userEvent.setup();
    localStorage.setItem("device-studio:photographic-scene:v1:p1",JSON.stringify({version:1,operation_id:"op-1",prompt:"A studio",width:768,height:512}));
    let releaseWait!: (value: typeof queued)=>void;
    vi.mocked(waitPhotographicOperation).mockImplementationOnce(()=>new Promise(resolve=>{releaseWait=resolve;}));
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.click(await screen.findByRole("button",{name:"Wait once for completion"}));
    const cancel=screen.getByRole("button",{name:"Request cancellation"});
    expect(cancel).toBeEnabled();
    await user.click(cancel);
    await screen.findByText(/canceled/);
    releaseWait({...queued,state:"running"});
    await waitFor(()=>expect(screen.getByText(/canceled/)).toBeInTheDocument());
  });

  it("retains a nonterminal or unknown operation and resets state on project switch",async()=>{
    const saved={version:1,operation_id:"op-1",prompt:"A studio",width:768,height:512};
    localStorage.setItem("device-studio:photographic-scene:v1:p1",JSON.stringify(saved));
    vi.mocked(readPhotographicOperation).mockResolvedValueOnce(queued);
    const view=renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    const newImage=await screen.findByRole("button",{name:"Start a different image"});
    expect(newImage).toBeDisabled();
    view.rerender(<PhotographicScenePanel projectId="p2"/>);
    expect(await screen.findByLabelText("Describe the scene")).toHaveValue("");
    expect(screen.queryByText(/Job job-1/)).not.toBeInTheDocument();
    expect(localStorage.getItem("device-studio:photographic-scene:v1:p1")).not.toBeNull();
  });

  it("refreshes status and reports cancellation failure without losing the operation",async()=>{
    const user=userEvent.setup();
    localStorage.setItem("device-studio:photographic-scene:v1:p1",JSON.stringify({version:1,operation_id:"op-1",prompt:"A studio",width:768,height:512}));
    vi.mocked(cancelPhotographicOperation).mockRejectedValueOnce(new Error("owner unavailable"));
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.click(await screen.findByRole("button",{name:"Refresh status"}));
    expect(readPhotographicOperation).toHaveBeenCalledWith("p1","op-1");
    await user.click(screen.getByRole("button",{name:"Request cancellation"}));
    expect(await screen.findByRole("alert")).toHaveTextContent("submit outcome is unknown");
    expect(localStorage.getItem("device-studio:photographic-scene:v1:p1")).not.toBeNull();
  });

  it("keeps failed imports attached to the operation and allows a terminal new image",async()=>{
    const user=userEvent.setup();
    vi.mocked(submitPhotographicScene).mockResolvedValueOnce({...queued,state:"succeeded",progress:100,result_ref:"out/image.png"});
    vi.mocked(getPhotographicPNG).mockRejectedValueOnce(new Error("bad owner bytes"));
    renderWithProviders(<PhotographicScenePanel projectId="p1"/>);
    await user.type(await screen.findByLabelText("Describe the scene"),"A quiet reading room");
    await user.selectOptions(screen.getByLabelText("Output dimensions"),"512x768");
    await user.click(screen.getByRole("button",{name:"Generate one local image"}));
    await user.click(await screen.findByRole("button",{name:"Import generated image into this project"}));
    expect(await screen.findByText(/could not be imported/)).toBeInTheDocument();
    await user.click(screen.getByRole("button",{name:"Start a different image"}));
    expect(localStorage.getItem("device-studio:photographic-scene:v1:p1")).toBeNull();
    expect(screen.getByLabelText("Describe the scene")).toHaveValue("");
  });
});
