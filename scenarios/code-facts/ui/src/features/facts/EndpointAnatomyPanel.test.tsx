import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchEndpointAnatomy } from "../../api/endpointAnatomy";
import { EndpointAnatomyPanel } from "./EndpointAnatomyPanel";

vi.mock("../../api/endpointAnatomy",()=>({searchEndpointAnatomy:vi.fn()}));
function mount(){const client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});return render(<QueryClientProvider client={client}><EndpointAnatomyPanel /></QueryClientProvider>)}

describe("EndpointAnatomyPanel",()=>{
  beforeEach(()=>vi.clearAllMocks());
  it("renders declaration, route and handler provenance",async()=>{
    vi.mocked(searchEndpointAnatomy).mockResolvedValue({query:"persistence",scenario:"code-facts",scope:"bounded source",degradations:[],results:[{id:"e",title:"POST /api/v1/persistence-patterns/search",summary:"Search endpoint",scenario:"code-facts",route_path:"/api/v1/persistence-patterns/search",method:"POST",handler:"SearchPersistencePatternsHTTPHandler",source_path:"api/handlers/facts/persistence_http.go",handler_line:20,source_sha256:"abc",state:"DECLARATION_ROUTE_HANDLER_JOINED",score:1,declaration:{path:"module.go",line:100,sha256:"a",basis:"descriptor"},registration:{path:"module.go",line:70,sha256:"b",basis:"mux route"},handler_source:{path:"persistence_http.go",line:20,sha256:"c",basis:"handler"},limits:["No runtime proof"]}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Search API surface"}));expect(await screen.findByText(/DECLARATION_ROUTE_HANDLER_JOINED/)).toBeTruthy();expect(screen.getByText(/No runtime proof/)).toBeTruthy();
  });
  it("renders duplicate route registrations as candidates without selecting a handler",async()=>{
    vi.mocked(searchEndpointAnatomy).mockResolvedValue({query:"duplicate",scenario:"code-facts",scope:"bounded source",degradations:[],results:[{id:"duplicate",title:"POST /api/v1/duplicate",summary:"Duplicate route",scenario:"code-facts",route_path:"/api/v1/duplicate",method:"POST",handler:"",state:"ROUTE_AMBIGUOUS",declaration:{path:"module.go",line:10,sha256:"decl",basis:"descriptor"},registration:{path:"",line:0,sha256:"",basis:""},handler_source:{path:"",line:0,sha256:"",basis:""},route_candidates:[{handler:"firstHandler",registration:{path:"module.go",line:20,sha256:"route-a",basis:"registration candidate"},handler_source:{path:"first.go",line:4,sha256:"handler-a",basis:"handler definition"}},{handler:"secondHandler",registration:{path:"module.go",line:21,sha256:"route-b",basis:"registration candidate"},handler_source:{path:"second.go",line:4,sha256:"handler-b",basis:"handler definition"}}],limits:["runtime routing precedence is unknown"],score:1}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Search API surface"}));
    expect(await screen.findByText(/Multiple source registrations match this route/)).toBeTruthy();
    expect(screen.getByText(/Candidate handler firstHandler/)).toBeTruthy();expect(screen.getByText(/Candidate handler secondHandler/)).toBeTruthy();
  });
  it("shows abstention for a query without an endpoint match",async()=>{
    vi.mocked(searchEndpointAnatomy).mockResolvedValue({query:"unrelated",scenario:"code-facts",scope:"bounded",degradations:["no_matching_declared_endpoint"],results:[]});
    mount();fireEvent.change(screen.getByLabelText("Endpoint query"),{target:{value:"underwater penguin"}});fireEvent.click(screen.getByRole("button",{name:"Search API surface"}));await waitFor(()=>expect(screen.getByText(/No declared endpoint matched/)).toBeTruthy());expect(screen.getByRole("status")).toBeTruthy();
  });
  it("renders Connect route, mount chain provenance and runtime boundary",async()=>{
    vi.mocked(searchEndpointAnatomy).mockResolvedValue({query:"Search CodeFactsService",scenario:"code-facts",scope:"bounded Connect source",degradations:[],results:[{id:"connect-search",title:"POST /vrooli.code_facts.v1.facts.CodeFactsService/Search",summary:"Search code facts",scenario:"code-facts",route_path:"/vrooli.code_facts.v1.facts.CodeFactsService/Search",method:"POST",handler:"CodeFactsService",state:"CONNECT_STATIC_MOUNT_REGISTERED",transport:"connect",connect_service:"CodeFactsService",connect_method:"Search",connect_chain:[{path:"packages/proto/facts.connect.go",line:40,sha256:"routehash",basis:"generated Connect Procedure path constant"},{path:"api/handlers/facts/module.go",line:59,sha256:"modulehash",basis:"service handler constructor returns path/handler pair"},{path:"api/handlers/facts/module.go",line:71,sha256:"modulehash",basis:"explicit ServiceMount"},{path:"api/handlers/facts/module.go",line:71,sha256:"modulehash",basis:"RegisterServices"}],declaration:{path:"module.go",line:100,sha256:"modulehash",basis:"descriptor"},registration:{path:"module.go",line:71,sha256:"modulehash",basis:"RegisterServices"},handler_source:{path:"",line:0,sha256:"",basis:""},limits:["Runtime module mounting remains unverified"],score:1}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Search API surface"}));
    expect(await screen.findByText(/Connect CodeFactsService\.Search/)).toBeTruthy();
    expect(screen.getByText(/runtime mount remains unverified/)).toBeTruthy();
    expect(screen.getByText(/generated Connect Procedure path constant/)).toBeTruthy();
    expect(screen.getByText(/sha256:routehash/)).toBeTruthy();
  });
  it("keeps unsupported Connect registration visibly partial",async()=>{
    vi.mocked(searchEndpointAnatomy).mockResolvedValue({query:"Search CodeFactsService",scenario:"code-facts",scope:"bounded Connect source",degradations:["connect_mount_partial:facts_search"],results:[{id:"connect-search",title:"POST /vrooli.code_facts.v1.facts.CodeFactsService/Search",summary:"Search code facts",scenario:"code-facts",route_path:"/vrooli.code_facts.v1.facts.CodeFactsService/Search",method:"POST",handler:"CodeFactsService",state:"CONNECT_MOUNT_PARTIAL",transport:"connect",connect_service:"CodeFactsService",connect_method:"Search",connect_chain:[{path:"packages/proto/facts.connect.go",line:40,sha256:"routehash",basis:"generated Connect Procedure path constant"}],declaration:{path:"module.go",line:100,sha256:"modulehash",basis:"descriptor"},registration:{path:"",line:0,sha256:"",basis:""},handler_source:{path:"",line:0,sha256:"",basis:""},limits:["No supported RegisterServices chain was proven"],score:1}]});
    mount();fireEvent.click(screen.getByRole("button",{name:"Search API surface"}));
    expect(await screen.findByText(/CONNECT_MOUNT_PARTIAL/)).toBeTruthy();
    expect(screen.getByText(/No supported RegisterServices chain was proven/)).toBeTruthy();
  });
});
