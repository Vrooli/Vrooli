/**
 * Axe-based accessibility scan of every `components/ui/` primitive.
 *
 * The plan calls for a `.a11y.test.tsx` per primitive; consolidating them into
 * one file keeps the scan list visible at a glance and removes 14 near-identical
 * boilerplate files. Each primitive is rendered in a representative
 * configuration and scanned via `expectNoA11yViolations`. New primitives must
 * register here.
 */
import { render } from "@testing-library/react";
import { Search } from "lucide-react";
import { afterEach, describe, it } from "vitest";

import { expectNoA11yViolations } from "../../test-utils/a11y";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Card, CardBody, CardDescription, CardHeader, CardTitle } from "./Card";
import { CodeBlock } from "./CodeBlock";
import { EmptyState } from "./EmptyState";
import { Icon } from "./Icon";
import { ConfirmDialog, Modal } from "./Modal";
import { ProgressBar } from "./ProgressBar";
import { RouteSkeleton } from "./RouteSkeleton";
import { SearchInput } from "./SearchInput";
import { Select } from "./Select";
import { Skeleton } from "./Skeleton";
import { StatusPill } from "./StatusPill";
import { Table, type ColumnDef } from "./Table";
import { Tabs } from "./Tabs";
import { ToastHost, ToastProvider } from "./Toast";

describe("ui primitives — axe", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("Button: primary, ghost, danger have no violations", async () => {
    const { container } = render(
      <div>
        <Button>Primary</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="danger">Danger</Button>
      </div>,
    );
    await expectNoA11yViolations(container);
  });

  it("Card: full composition", async () => {
    const { container } = render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
          <CardDescription>Desc</CardDescription>
        </CardHeader>
        <CardBody>body</CardBody>
      </Card>,
    );
    await expectNoA11yViolations(container);
  });

  it("Badge", async () => {
    const { container } = render(<Badge tone="success">ok</Badge>);
    await expectNoA11yViolations(container);
  });

  it("StatusPill", async () => {
    const { container } = render(<StatusPill status="ok" label="Healthy" />);
    await expectNoA11yViolations(container);
  });

  it("Skeleton + RouteSkeleton", async () => {
    const { container } = render(
      <div>
        <Skeleton className="h-6 w-32" />
        <RouteSkeleton label="loading" />
      </div>,
    );
    await expectNoA11yViolations(container);
  });

  it("EmptyState", async () => {
    const { container } = render(
      <EmptyState title="Nothing here" description="Try a different filter." />,
    );
    await expectNoA11yViolations(container);
  });

  it("Table", async () => {
    interface Row { id: string; name: string }
    const cols: ColumnDef<Row>[] = [
      { key: "name", header: "Name", cell: (r) => r.name, sortable: true },
    ];
    const { container } = render(
      <Table
        columns={cols}
        rows={[{ id: "a", name: "Alpha" }]}
        rowKey={(r) => r.id}
        caption="People"
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("Modal (open)", async () => {
    const { container } = render(
      <Modal
        open
        onClose={() => {}}
        title="Title"
        description="desc"
        closeLabel="close"
        backdropCloseLabel="close backdrop"
      >
        <p>body</p>
      </Modal>,
    );
    await expectNoA11yViolations(container);
  });

  it("ConfirmDialog", async () => {
    const { container } = render(
      <ConfirmDialog
        open
        onConfirm={() => {}}
        onCancel={() => {}}
        title="Confirm?"
        description="This cannot be undone."
        confirmLabel="Yes"
        cancelLabel="No"
        closeLabel="close"
        backdropCloseLabel="close backdrop"
        destructive
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("Toast host", async () => {
    const { container } = render(
      <ToastProvider>
        <ToastHost dismissLabel="dismiss" />
      </ToastProvider>,
    );
    await expectNoA11yViolations(container);
  });

  it("SearchInput", async () => {
    const { container } = render(
      <SearchInput
        value=""
        onChange={() => {}}
        ariaLabel="Search"
        clearLabel="Clear"
        placeholder="Search…"
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("Select", async () => {
    const { container } = render(
      <Select
        ariaLabel="Choose"
        value="a"
        onChange={() => {}}
        options={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
        ]}
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("Tabs", async () => {
    const { container } = render(
      <Tabs
        ariaLabel="Sections"
        value="a"
        onChange={() => {}}
        items={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
        ]}
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("CodeBlock", async () => {
    const { container } = render(
      <CodeBlock
        code={"{\n  \"ok\": true\n}"}
        language="json"
        showLineNumbers
        copyLabel="Copy code"
        copiedLabel="Copied"
        copyShortLabel="Copy"
      />,
    );
    await expectNoA11yViolations(container);
  });

  it("ProgressBar", async () => {
    const { container } = render(<ProgressBar value={42} label="Indexing surfaces" />);
    await expectNoA11yViolations(container);
  });

  it("Icon (labelled + decorative)", async () => {
    const { container } = render(
      <div>
        <Icon icon={Search} label="search" />
        <Icon icon={Search} />
      </div>,
    );
    await expectNoA11yViolations(container);
  });
});
