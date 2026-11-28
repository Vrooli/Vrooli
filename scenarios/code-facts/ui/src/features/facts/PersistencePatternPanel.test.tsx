import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { searchPersistencePatterns } from "../../api/persistencePatterns";
import { PersistencePatternPanel } from "./PersistencePatternPanel";

vi.mock("../../api/persistencePatterns", () => ({ searchPersistencePatterns: vi.fn() }));

function mount() { const client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}}); return render(<QueryClientProvider client={client}><PersistencePatternPanel /></QueryClientProvider>); }

describe("PersistencePatternPanel",()=>{
  beforeEach(()=>vi.clearAllMocks());
  it("shows the joined source layers and explicit limits",async()=>{
    vi.mocked(searchPersistencePatterns).mockResolvedValue({query:"catalog generations",scenario:"code-facts",scope:"scenario:code-facts static only",degradations:[],results:[{id:"p",title:"SQLite catalog persistence: code_facts_generations",summary:"Exact table-name source join.",scenario:"code-facts",entity:"code_facts_generations",source_path:"api/internal/catalog/sqlite.go",line:232,source_sha256:"abc",classification:"JOINED_STATIC_SOURCE_EVIDENCE",score:1,schema:{layer:"schema",path:"schema.sql",line:8,sha256:"a",basis:"CREATE TABLE",status:"observed"},schema_binding:{layer:"schema_embedding",path:"schema.go",line:5,sha256:"d",basis:"go:embed",status:"source_binding"},migration:{layer:"migration",path:"sqlite.go",line:20,sha256:"b",basis:"Schema() registration",status:"registered_in_source"},repository:{layer:"repository",path:"sqlite.go",line:232,sha256:"c",basis:"SQL table reference",status:"observed"},limits:["Does not prove execution"]}]});
    mount(); fireEvent.click(screen.getByRole("button",{name:"Search patterns"}));
    expect(await screen.findByText(/SQLite catalog persistence/)).toBeTruthy(); expect(screen.getByText(/Does not prove execution/)).toBeTruthy();
  });
  it("renders abstention instead of inventing a pattern",async()=>{
    vi.mocked(searchPersistencePatterns).mockResolvedValue({query:"unrelated",scenario:"code-facts",scope:"bounded",degradations:["no_joined_static_pattern"],results:[]});
    mount(); fireEvent.change(screen.getByLabelText("Persistence query"),{target:{value:"underwater penguin"}}); fireEvent.click(screen.getByRole("button",{name:"Search patterns"}));
    await waitFor(()=>expect(screen.getByText(/No joined persistence pattern matched/)).toBeTruthy()); expect(screen.getByRole("status")).toBeTruthy();
  });
  it("shows ambiguous lineage as UNKNOWN instead of saying no pattern exists", async()=>{
    vi.mocked(searchPersistencePatterns).mockResolvedValue({query:"orders schema",scenario:"code-facts",scope:"scenario:code-facts static only",degradations:["lineage_unknown"],results:[],unknowns:[{entity:"orders",status:"UNKNOWN",reason:"exact_repository_table_reference_not_proven",schema:{layer:"schema",path:"api/internal/catalog/schema.sql",line:8,sha256:"schemahash",basis:"CREATE TABLE",status:"observed"}}]});
    mount(); fireEvent.click(screen.getByRole("button",{name:"Search patterns"}));
    expect(await screen.findByTestId("persistence-unknown")).toHaveTextContent("UNKNOWN lineage for orders: exact_repository_table_reference_not_proven");
    expect(screen.queryByText(/No joined persistence pattern matched/)).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("lineage_unknown");
  });
});
