import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchDocCodeDrift } from "../../api/docCodeDrift";
import { DocCodeDriftPanel } from "./DocCodeDriftPanel";

vi.mock("../../api/docCodeDrift",()=>({searchDocCodeDrift:vi.fn()}));
function mount(){const client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});return render(<QueryClientProvider client={client}><DocCodeDriftPanel scenario="architecture-cartographer"/></QueryClientProvider>)}

describe("DocCodeDriftPanel",()=>{
  beforeEach(()=>vi.clearAllMocks());
  it("shows authored doc and code provenance with partial limits",async()=>{
    vi.mocked(searchDocCodeDrift).mockResolvedValue({query:"graph",scope:"scenario:architecture-cartographer bounded",authority:"domains_doc",degradations:[],results:[{id:"g",title:"graph · api/internal/graph",summary:"Build graph.",scenario:"architecture-cartographer",domain:"graph",claim:"DOMAINS.md Source Paths includes api/internal/graph",classification:"STRUCTURAL_PATH_MATCH",status:"PARTIAL",api_status:"ROUTE_HANDLER_DIRECT_CALL",api:[{path:"api/handlers/graph/module.go",line:40,sha256:"route",basis:"POST /api/v1/slices/search literal mux route"},{path:"api/handlers/graph/slice_search.go",line:72,sha256:"handler",basis:"handler function"}],score:1,documentation:{path:"docs/concepts/DOMAINS.md",line:55,sha256:"doc",basis:"authored claim"},code:[{path:"api/internal/graph/service.go",line:20,sha256:"src",basis:"Go declaration candidate: Service"}],limits:["No semantic proof"]}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Compare paths"}));expect(await screen.findByText(/STRUCTURAL_PATH_MATCH · PARTIAL/)).toBeTruthy();expect(screen.getByText(/No semantic proof/)).toBeTruthy();expect(screen.getByText(/Literal API chain: ROUTE_HANDLER_DIRECT_CALL/)).toBeTruthy();expect(screen.getByText(/POST \/api\/v1\/slices\/search literal mux route/)).toBeTruthy();
  });
  it("keeps dynamic route evidence UNKNOWN",async()=>{
    vi.mocked(searchDocCodeDrift).mockResolvedValue({query:"billing",scope:"bounded",authority:"domains_doc",degradations:["api_linkage_unknown"],results:[{id:"b",title:"billing",summary:"Own billing",scenario:"demo",domain:"billing",claim:"Source Paths includes billing",classification:"STRUCTURAL_PATH_MATCH",status:"PARTIAL",api_status:"UNKNOWN_DYNAMIC_ROUTE",api:[{path:"api/handlers/billing/module.go",line:12,sha256:"route",basis:"unsupported dynamic route expression: prefix + path"}],score:1,documentation:{path:"DOMAINS.md",line:5,sha256:"doc",basis:"authored"},code:[],limits:["Dynamic route is not proven"]}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Compare paths"}));expect(await screen.findByText(/Literal API chain: UNKNOWN_DYNAMIC_ROUTE/)).toBeTruthy();expect(screen.getByText(/Dynamic route is not proven/)).toBeTruthy();
  });
  it("renders abstention and degradation for unrelated or unknown evidence",async()=>{
    vi.mocked(searchDocCodeDrift).mockResolvedValue({query:"underwater",scope:"bounded",authority:"unknown",degradations:["docs_authority_unknown"],results:[]});
    mount();fireEvent.change(screen.getByLabelText("Search docs and source"),{target:{value:"underwater"}});fireEvent.click(screen.getByRole("button",{name:"Compare paths"}));await waitFor(()=>expect(screen.getByText(/No source-path candidate matched/)).toBeTruthy());expect(screen.getByRole("status")).toBeTruthy();
  });
});
