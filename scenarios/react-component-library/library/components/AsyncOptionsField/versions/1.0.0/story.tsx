import { useMemo, useState } from "react";
import { AsyncOptionsField, type AsyncOption } from "./AsyncOptionsField";

const options: AsyncOption[] = [
  {
    value: "atlas",
    label: "Atlas workspace",
    description: "28 collaborators · Updated just now",
    group: "Recent workspaces",
  },
  {
    value: "beacon",
    label: "Beacon operations",
    description: "12 collaborators · Updated yesterday",
    group: "Recent workspaces",
  },
  {
    value: "cinder",
    label: "Cinder research",
    description: "Private · 6 collaborators",
    group: "All workspaces",
  },
  {
    value: "delta",
    label: "Delta launch plan",
    description: "Private · 4 collaborators",
    group: "All workspaces",
  },
];

function Showcase({
  children,
  eyebrow,
  title,
  detail,
}: {
  children: React.ReactNode;
  eyebrow: string;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        display: "grid",
        gap: "var(--space-lg)",
        boxSizing: "border-box",
        width: "min(100%, 720px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {eyebrow}
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

function loader(
  query: string,
  { signal }: { signal: AbortSignal; page: number; pageSize: number },
) {
  return new Promise<AsyncOption[]>((resolve, reject) => {
    const timer = globalThis.setTimeout(() => {
      const normalized = query.trim().toLowerCase();
      resolve(
        options.filter((option) =>
          option.label.toLowerCase().includes(normalized),
        ),
      );
    }, 260);
    signal.addEventListener(
      "abort",
      () => {
        globalThis.clearTimeout(timer);
        reject(new DOMException("Cancelled", "AbortError"));
      },
      { once: true },
    );
  });
}

export function Default() {
  return (
    <Showcase
      eyebrow="Workspace switcher"
      title="Search the right context"
      detail="Type a few letters to find a workspace without sacrificing keyboard control or a clear loading state."
    >
      <AsyncOptionsField
        label="Workspace"
        description="Choose where you want to continue."
        loadOptions={loader}
        defaultValue="atlas"
        initialOptions={options}
      />
    </Showcase>
  );
}

export function Selected() {
  const [selected, setSelected] = useState("beacon");
  return (
    <Showcase
      eyebrow="Connected data"
      title="A selection that stays legible"
      detail="Descriptions give the choice enough context to be confident, even when names are similar."
    >
      <AsyncOptionsField
        label="Workspace"
        loadOptions={loader}
        value={selected}
        onChange={setSelected}
        placeholder="Find a workspace"
      />
    </Showcase>
  );
}

export function ErrorState() {
  const [attempts, setAttempts] = useState(0);
  const failingLoader = useMemo(
    () => () => {
      setAttempts((count) => count + 1);
      return Promise.reject(new Error("Service unavailable"));
    },
    [],
  );
  return (
    <Showcase
      eyebrow="Recovery"
      title="A failure with a next step"
      detail="The field explains what happened and keeps retry in the user’s reach."
    >
      <AsyncOptionsField
        label="Workspace"
        description={`Demo request ${attempts + 1} · service unavailable`}
        loadOptions={failingLoader}
        errorText="Workspace search is temporarily unavailable."
        retryLabel="Retry search"
        initialOpen
        debounceMs={0}
        maxAttempts={1}
      />
    </Showcase>
  );
}
