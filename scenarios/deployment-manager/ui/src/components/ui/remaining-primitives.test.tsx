import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "./select";
import {
  Table, TableBody, TableCaption, TableCell, TableFooter, TableHead, TableHeader, TableRow,
} from "./table";
import { Tip } from "./tip";
import { renderWithProviders } from "@vrooli/api-base/testing";

describe("remaining UI primitives", () => {
  it("opens a select and chooses an item", () => {
    renderWithProviders(
      <Select defaultValue="linux">
        <SelectTrigger aria-label="Platform"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="linux">Linux</SelectItem><SelectItem value="windows">Windows</SelectItem></SelectContent>
      </Select>,
    );
    fireEvent.click(screen.getByRole("combobox", { name: "Platform" }));
    expect(screen.getByRole("option", { name: "Linux" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("option", { name: "Windows" }));
    expect(screen.getByRole("combobox", { name: "Platform" })).toHaveTextContent("Windows");
  });

  it("renders table semantics and all optional sections", () => {
    renderWithProviders(
      <Table>
        <TableCaption>Release records</TableCaption>
        <TableHeader><TableRow><TableHead>Version</TableHead></TableRow></TableHeader>
        <TableBody><TableRow><TableCell>1.0.0</TableCell></TableRow></TableBody>
        <TableFooter><TableRow><TableCell>Total: 1</TableCell></TableRow></TableFooter>
      </Table>,
    );
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByText("Release records")).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("Total: 1")).toBeInTheDocument();
  });

  it("supports every tip tone and an optional action", () => {
    renderWithProviders(<Tip title="Warning" tone="warning" action={<button>Fix</button>}>Details</Tip>);
    expect(screen.getByText("Warning")).toBeInTheDocument();
    expect(screen.getByText("Details")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fix" })).toBeInTheDocument();
  });
});
