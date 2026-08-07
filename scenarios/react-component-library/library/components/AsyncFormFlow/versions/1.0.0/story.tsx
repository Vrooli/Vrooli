import { useEffect, useState, type CSSProperties } from "react";
import { useFormStore } from "../../../../services/FormStore/versions/1.0.0/FormStore";
import {
  AsyncFormFlow,
  type AsyncFormFlowContext,
  type AsyncFormSubmitError,
} from "./AsyncFormFlow";

type ProfileValues = { name: string; email: string; role: string };

const shell: CSSProperties = { width: "min(100%, 46rem)", minWidth: 0 };
const field: CSSProperties = { display: "grid", gap: 6, minWidth: 0 };
const label: CSSProperties = {
  color: "var(--color-muted-foreground, #64748b)",
  fontSize: 12,
  fontWeight: 750,
};
const input: CSSProperties = {
  boxSizing: "border-box",
  width: "100%",
  minHeight: 46,
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-control, 8px)",
  padding: "0 12px",
  background: "var(--color-surface, #fff)",
  color: "inherit",
  font: "inherit",
};

function ProfileFields({
  store,
  mode,
}: {
  store: AsyncFormFlowContext<ProfileValues>["store"];
  mode:
    | "default"
    | "loading"
    | "request-error"
    | "offline"
    | "empty"
    | "partial"
    | "submitting"
    | "success"
    | "retry";
}) {
  const name = useFormStore(store, "name");
  const email = useFormStore(store, "email");
  const role = useFormStore(store, "role");
  useEffect(() => {
    if (mode === "request-error" || mode === "retry") {
      store.setError("role", "Choose a role that is still available.");
      store.setPhase(
        "error",
        "The workspace service rejected this version. Retry without losing your edits.",
      );
    }
    if (mode === "submitting") store.setPhase("submitting");
    if (mode === "success") store.setPhase("success");
  }, [mode, store]);
  return (
    <div style={{ display: "grid", gap: "var(--space-md, 16px)" }}>
      <div
        style={{
          display: "grid",
          gap: "var(--space-md, 16px)",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(min(100%, 14rem), 1fr))",
        }}
      >
        <label style={field}>
          {" "}
          <span style={label}>Full name</span>
          <input
            id="name"
            aria-label="Full name"
            value={name.value}
            onChange={(event) => name.setValue(event.target.value)}
            onBlur={name.touch}
            style={input}
          />
          {name.error && (
            <span
              role="alert"
              style={{ color: "var(--color-danger, #dc2626)", fontSize: 12 }}
            >
              {name.error}
            </span>
          )}
        </label>
        <label style={field}>
          {" "}
          <span style={label}>Work email</span>
          <input
            id="email"
            aria-label="Work email"
            type="email"
            value={email.value}
            onChange={(event) => email.setValue(event.target.value)}
            onBlur={email.touch}
            style={input}
          />
          {email.error && (
            <span
              role="alert"
              style={{ color: "var(--color-danger, #dc2626)", fontSize: 12 }}
            >
              {email.error}
            </span>
          )}
        </label>
      </div>
      <label style={field}>
        <span style={label}>Workspace role</span>
        <select
          id="role"
          aria-label="Workspace role"
          value={role.value}
          onChange={(event) => role.setValue(event.target.value)}
          style={input}
        >
          <option value="editor">Editor</option>
          <option value="owner">Owner</option>
          <option value="viewer">Viewer</option>
        </select>
      </label>
    </div>
  );
}

function Flow({
  mode = "default",
}: {
  mode?:
    | "default"
    | "loading"
    | "request-error"
    | "offline"
    | "empty"
    | "partial"
    | "submitting"
    | "success"
    | "retry";
}) {
  const [attempts, setAttempts] = useState(0);
  const [navigated, setNavigated] = useState(false);
  const initial: ProfileValues = {
    name: "Mara Chen",
    email: "mara@northstar.dev",
    role: "editor",
  };
  return (
    <div style={shell}>
      <AsyncFormFlow<ProfileValues>
        initialValues={initial}
        offline={mode === "offline"}
        load={
          mode === "empty"
            ? () => ({ state: "empty" })
            : mode === "partial"
              ? () => ({
                  values: { name: "Mara Chen" },
                  state: "partial",
                })
              : mode === "loading"
                ? (signal) =>
                    new Promise((resolve) => {
                      const timer = window.setTimeout(
                        () => resolve({ values: initial }),
                        1400,
                      );
                      signal.addEventListener(
                        "abort",
                        () => window.clearTimeout(timer),
                        { once: true },
                      );
                    })
                : () => ({ values: initial })
        }
        validate={(values) => ({
          ...(values.name.trim()
            ? {}
            : {
                name: "Add a name so teammates know who owns this workspace.",
              }),
          ...(values.email.includes("@")
            ? {}
            : { email: "Use a work email with an @ symbol." }),
        })}
        onSubmit={async (_values, signal) => {
          await new Promise<void>((resolve, reject) => {
            const timer = window.setTimeout(
              resolve,
              mode === "submitting" ? 1800 : 400,
            );
            signal.addEventListener(
              "abort",
              () => {
                window.clearTimeout(timer);
                reject(new DOMException("Cancelled", "AbortError"));
              },
              { once: true },
            );
          });
          if (
            mode === "request-error" ||
            (mode === "retry" && attempts === 0)
          ) {
            setAttempts((value) => value + 1);
            const error = new Error(
              "The workspace service rejected this version. Retry without losing your edits.",
            ) as AsyncFormSubmitError<ProfileValues>;
            error.fieldErrors = {
              role: "Choose a role that is still available.",
            };
            throw error;
          }
        }}
        onNavigate={() => setNavigated(true)}
        destination="workspace"
        title={mode === "empty" ? "Start a workspace" : "Workspace profile"}
        description={
          navigated
            ? "Navigation completed; this handoff is ready for the next screen."
            : "A focused create-or-edit flow with preserved input, explicit recovery, and a clear next step."
        }
      >
        {(context) => <ProfileFields mode={mode} store={context.store} />}
      </AsyncFormFlow>
      {navigated && (
        <div
          role="status"
          style={{
            marginTop: 12,
            color: "var(--color-success, #15803d)",
            fontSize: 13,
            fontWeight: 700,
          }}
        >
          Navigated to workspace overview.
        </div>
      )}
    </div>
  );
}

export function Default() {
  return <Flow />;
}
export function Loading() {
  return <Flow mode="loading" />;
}
export function RequestError() {
  return <Flow mode="request-error" />;
}
export function Offline() {
  return <Flow mode="offline" />;
}
export function Empty() {
  return <Flow mode="empty" />;
}
export function Partial() {
  return <Flow mode="partial" />;
}
export function Submitting() {
  return <Flow mode="submitting" />;
}
export function Success() {
  return <Flow mode="success" />;
}
export function Retry() {
  return <Flow mode="retry" />;
}
