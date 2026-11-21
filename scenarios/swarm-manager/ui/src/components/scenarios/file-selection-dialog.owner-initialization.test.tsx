
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { defaultApiClient } from "../../lib/api-client";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { FileSelectionDialog } from "./file-selection-dialog";
import type { ScenarioFile } from "../../types";
const initialFiles: ScenarioFile[] = [{name:"PRD.md",path:"PRD.md",type:"file"},{name:"source.ts",path:"source.ts",type:"file"}];
let view: {unmount:()=>void} | undefined;
let confirm: ReturnType<typeof vi.fn>;
let expected: unknown[][];
let props: React.ComponentProps<typeof FileSelectionDialog>;
function mount(custom=false){props={isOpen:true,onClose:()=>refresh({isOpen:false}),onConfirm:confirm,scenarioName:"fixture-alpha",files:initialFiles,initialSelection:custom?{paths:["source.ts"]}:{preset:"planning",paths:[]}};view=renderWithProviders(<FileSelectionDialog {...props}/>);}
function refresh(changes: Partial<typeof props> = {}) {props={...props,...changes};(view as ReturnType<typeof renderWithProviders>).rerender(<FileSelectionDialog {...props}/>);}
function count(n:number){expect(screen.getByTestId("confirm-selection-button")).toHaveTextContent(`Confirm Selection (${n} files)`);}
beforeEach(()=>{confirm=vi.fn();expected=[];for(const method of ["get","post","put","patch","delete"] as const)vi.spyOn(defaultApiClient,method).mockRejectedValue(new Error("Forbidden local selection transport"));vi.stubGlobal("fetch",vi.fn().mockRejectedValue(new Error("Forbidden fetch")));});
afterEach(()=>{try{view?.unmount();cleanup();getSpatialNav()?.dispose();expect(confirm.mock.calls).toEqual(expected);for(const method of ["get","post","put","patch","delete"] as const)expect(defaultApiClient[method]).not.toHaveBeenCalled();expect(fetch).not.toHaveBeenCalled();}finally{view=undefined;vi.restoreAllMocks();vi.unstubAllGlobals();}});
describe("File selection owner initialization compatibility",()=>{
 it("retains custom tentative selection across a fresh identical parent default object",async()=>{mount();count(1);await userEvent.click(screen.getByTestId("select-all-button"));count(2);refresh({initialSelection:{preset:"planning",paths:[]}});count(2);expect(screen.getByTestId("preset-select")).toHaveValue("");expected=[[{preset:undefined,paths:["PRD.md","source.ts"]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
 it("retains custom paths through an actual file-tree refresh without selecting newly available files",async()=>{mount(true);count(1);await userEvent.click(screen.getByTestId("select-all-button"));refresh({files:[...initialFiles,{name:"README.md",path:"README.md",type:"file"}]});count(2);expected=[[{preset:undefined,paths:["PRD.md","source.ts"]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
 it("recomputes an active documentation preset when matching files become available",async()=>{mount();await userEvent.selectOptions(screen.getByTestId("preset-select"),"documentation");count(1);refresh({files:[...initialFiles,{name:"README.md",path:"README.md",type:"file"}]});count(2);expect(screen.getByTestId("preset-select")).toHaveValue("documentation");expected=[[{preset:"documentation",paths:["PRD.md","README.md"]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
 it("restores owner defaults on an explicit close and reopen",async()=>{mount();await userEvent.click(screen.getByTestId("select-all-button"));await userEvent.click(screen.getByRole("button",{name:/^Cancel$/}));await waitFor(()=>expect(screen.queryByTestId("file-selection-dialog")).toBeNull());refresh({isOpen:true});count(1);expect(screen.getByTestId("preset-select")).toHaveValue("planning");});
 it("restores supplied defaults when the owner scenario changes",async()=>{mount();await userEvent.click(screen.getByTestId("select-all-button"));refresh({scenarioName:"fixture-beta"});count(1);expect(screen.getByText("Select files to keep when archiving fixture-beta")).toBeInTheDocument();});
 it("applies genuinely changed initialization defaults while open",async()=>{mount();await userEvent.click(screen.getByTestId("select-all-button"));refresh({initialSelection:{paths:["source.ts"]}});count(1);expect(screen.getByTestId("preset-select")).toHaveValue("");expected=[[{preset:undefined,paths:["source.ts"]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
 it("hydrates an initial preset when its file tree arrives asynchronously",async()=>{mount();refresh({files:[]});count(0);refresh({files:initialFiles});count(1);expect(screen.getByTestId("preset-select")).toHaveValue("planning");expected=[[{preset:"planning",paths:["PRD.md"]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
 it("treats reordered duplicate custom defaults as equivalent without erasing a tentative empty selection",async()=>{mount(true);refresh({initialSelection:{paths:["source.ts","PRD.md"]}});count(2);await userEvent.click(screen.getByTestId("clear-all-button"));count(0);refresh({initialSelection:{paths:["PRD.md","source.ts","source.ts"]}});count(0);expected=[[{preset:undefined,paths:[]}]];await userEvent.click(screen.getByTestId("confirm-selection-button"));});
});
