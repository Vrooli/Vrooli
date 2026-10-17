import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { Archive, ArrowRight, Check, Clipboard, KeyRound, Lock, Plus, Search, Settings, ShieldCheck, Sparkles, Users, X } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";
import { Button as RclButton } from "@vrooli/react-component-library/Button/2.2.9";
import { Dialog as RclDialog } from "@vrooli/react-component-library/Dialog/1.3.8";
import { Input as RclInput } from "@vrooli/react-component-library/Input/1.3.6";
import { PasswordInput as RclPasswordInput } from "@vrooli/react-component-library/PasswordInput/2.0.3";
import {
  createGrant,
  getGrantEffectiveAccess,
  completeEnrollment,
  createVaultItem,
  createVaultExport,
  exportAudit,
  redeemVaultExport,
  approveAccessRequest,
  generatePassword,
  getVaultStatus,
  listVaultItems,
  loginWithAuthenticator,
  listAccessRequests,
  listVaultItemHistory,
  listGrants,
  issueAssurance,
  lockVault,
  revealVaultItem,
  setConfiguredOwnerToken,
  denyAccessRequest,
  revokeGrant,
  createSource,
  commitVaultImport,
  getRecoveryStatus,
  getSourceHealth,
  getEnrollmentStatus,
  listSources,
  previewVaultImport,
  restoreVaultItem,
  type Source,
  trashVaultItem,
  updateVaultItem,
  unlockVault,
  type VaultItem
} from "./lib/passwordManagerApi";
import { getAudit, type AuditEvent } from "./lib/passwordManagerApi";

export type PasswordManagerPageId = "vault" | "access" | "activity" | "recovery" | "sources" | "settings";
type Page = PasswordManagerPageId;
type Toast = { kind: "success" | "error"; message: string } | null;

const itemLabels: Record<string, string> = {
  login: "Login",
  password: "Password",
  api_credential: "API credential",
  secure_note: "Secure note",
  totp: "TOTP",
  ssh: "SSH key"
};

function ErrorBoundaryNotice({ message, actionLabel, onAction }: { message: string; actionLabel?: string; onAction?: () => void }) {
  return <div className="notice notice-error" role="alert"><X size={16} /><span>{message}</span>{onAction && actionLabel && <button className="button button-quiet" onClick={onAction}>{actionLabel}</button>}</div>;
}

export function PasswordManagerApp({ initialPage = "vault" }: { initialPage?: PasswordManagerPageId } = {}) {
  const location = useLocation();
  const navigate = useNavigate();
  const routePage = pageFromPath(location.pathname) || initialPage;
  const [page, setPage] = useState<Page>(routePage);
  const [items, setItems] = useState<VaultItem[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [auditIntegrity, setAuditIntegrity] = useState<"verified" | "unknown" | "tampered">("unknown");
  const [status, setStatus] = useState<"locked" | "unlocked" | "loading">("loading");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<VaultItem | null>(null);
  const [editing, setEditing] = useState<VaultItem | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const [showEnrollment, setShowEnrollment] = useState(false);
  const [ownerToken, setOwnerToken] = useState("");
  const [toast, setToast] = useState<Toast>(null);
  const [error, setError] = useState("");

  useEffect(() => { setPage(routePage); }, [routePage]);

  const goToPage = useCallback((next: Page) => {
    setPage(next);
    navigate(next === "vault" ? "/vault" : `/${next}`);
  }, [navigate]);

  const refresh = useCallback(async () => {
    try {
      setError("");
      const [vaultStatus, itemResponse, auditResponse] = await Promise.all([getVaultStatus(), listVaultItems(query), getAudit()]);
      setStatus(vaultStatus.status);
      setItems(itemResponse.items || []);
      setEvents(auditResponse.events || []);
      setAuditIntegrity(auditResponse.integrity || "unknown");
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "The vault could not be reached.";
      setError(message);
      try {
        const enrollment = await getEnrollmentStatus();
        if (enrollment.bootstrap_configured && !enrollment.enrolled) setShowEnrollment(true);
      } catch {
        // The protected owner boundary can be unavailable during first boot.
      }
      setStatus("locked");
    }
  }, [query]);

  useEffect(() => { void refresh(); }, [refresh]);

  const run = async (action: () => Promise<unknown>, success: string) => {
    try { setError(""); await action(); setToast({ kind: "success", message: success }); await refresh(); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "The operation failed."); }
  };

  const saveToken = (event: FormEvent) => {
    event.preventDefault();
    setConfiguredOwnerToken(ownerToken);
    setShowToken(false);
    void refresh();
  };

  return <div className="pm-app" data-experience-page={page}>
    <aside className="pm-sidebar" data-experience-region="navigation-shell">
      <div className="brand"><div className="brand-mark"><KeyRound size={20} /></div><div><strong>Vaultline</strong><span>Secrets Manager</span></div></div>
      <div className="workspace-switcher"><span className="eyebrow">WORKSPACE</span><button onClick={() => goToPage("settings")}><span className="workspace-dot" />Personal vault <ArrowRight size={14} /></button></div>
      <nav aria-label="Primary navigation">
        {(["vault", "access", "activity", "recovery", "sources", "settings"] as Page[]).map((entry) => <button key={entry} aria-label={entry.charAt(0).toUpperCase() + entry.slice(1)} aria-current={page === entry ? "page" : undefined} className={page === entry ? "nav-item active" : "nav-item"} onClick={() => goToPage(entry)}>{entry === "vault" ? <KeyRound size={17} /> : entry === "access" ? <Users size={17} /> : entry === "activity" ? <ShieldCheck size={17} /> : entry === "recovery" ? <Archive size={17} /> : <Settings size={17} />}<span>{entry.charAt(0).toUpperCase() + entry.slice(1)}</span>{entry === "access" && <span className="nav-count">2</span>}</button>)}
      </nav>
      <div className="sidebar-footer"><div className="security-pulse"><span className="pulse-dot" /><div><strong>Authority online</strong><span>Encrypted custody active</span></div></div><button className="avatar" aria-label="LO, open owner settings" onClick={() => goToPage("settings")}>LO</button></div>
    </aside>
    <main className="pm-main" data-experience-region="routed-page">
      <header className="pm-topbar"><div><span className="eyebrow">PERSONAL WORKSPACE</span><h1>{page === "vault" ? "Your vault" : page === "access" ? "Access control" : page === "activity" ? "Activity" : page === "recovery" ? "Recovery" : page === "sources" ? "Credential sources" : "Settings"}</h1></div><div className="top-actions"><span className={status === "unlocked" ? "status-chip online" : "status-chip"}>{status === "unlocked" ? <ShieldCheck size={14} /> : <Lock size={14} />}{status === "loading" ? "Checking vault" : status === "unlocked" ? "Vault unlocked" : "Vault locked"}</span>{status === "unlocked" ? <button data-testid="vault-lock" className="button button-quiet" onClick={() => void run(async () => { await lockVault(); setSelected(null); setEditing(null); }, "Vault locked")}>Lock now</button> : <button data-testid="vault-unlock" className="button button-primary" onClick={() => void run(() => unlockVault(), "Vault unlocked")}>Unlock vault</button>}</div></header>
      {toast && <div className="toast" role="status"><Check size={16} />{toast.message}<button onClick={() => setToast(null)} aria-label="Dismiss notification"><X size={14} /></button></div>}
      {error && <ErrorBoundaryNotice message={error} actionLabel={isOwnerAuthError(error) ? "Sign in" : undefined} onAction={isOwnerAuthError(error) ? () => setShowLogin(true) : undefined} />}
      {page === "vault" && <VaultPage experienceState={status === "loading" ? "loading" : "ready"} items={items} query={query} onQueryChange={setQuery} selected={selected} onSelect={setSelected} onBack={() => setSelected(null)} locked={status !== "unlocked"} onCreate={() => setShowCreate(true)} onReveal={(item) => void revealVaultItem(item.id, ["password", "token", "notes"]).then((result) => { setSelected({ ...item, ...({ __revealed: result.fields } as Partial<VaultItem>) }); setToast({ kind: "success", message: "Secret revealed for this session" }); }).catch((caught) => setError(caught instanceof Error ? caught.message : "Reveal failed."))} onEdit={async (item) => { try { const result = await revealVaultItem(item.id, []); setEditing({ ...item, ...({ __revealed: result.fields } as Partial<VaultItem>) }); } catch (caught) { setError(caught instanceof Error ? caught.message : "Edit preparation failed."); } }} onTrash={(item) => void run(() => trashVaultItem(item.id), "Item moved to trash")} />}
      {page === "access" && <AccessPage items={items} onCreate={(input) => void run(() => createGrant(input), "Grant created")}/>} 
      {page === "activity" && <ActivityPage events={events} integrity={auditIntegrity} onExport={() => void run(async () => { const bundle = await exportAudit(); const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" }); const link = document.createElement("a"); link.href = URL.createObjectURL(blob); link.download = "vault-activity-export.json"; link.click(); URL.revokeObjectURL(link.href); }, "Metadata-only activity export downloaded")} />}
      {page === "recovery" && <RecoveryPage onExport={(format) => void run(async () => { const handle = await createVaultExport(format); const bundle = await redeemVaultExport(handle); const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" }); const link = document.createElement("a"); link.href = URL.createObjectURL(blob); link.download = `vault-export-${format}.json`; link.click(); URL.revokeObjectURL(link.href); }, `${format === "native" ? "Encrypted" : "Plaintext"} export downloaded`)} onRestore={(id) => run(() => restoreVaultItem(id), "Item restored from trash")} onChanged={refresh} />}
      {page === "sources" && <SourcesPage onCreate={(input) => void run(() => createSource(input), "Source registered")} />}
    {page === "settings" && <SettingsPage onLogin={() => setShowLogin(true)} onToken={() => setShowToken(true)} onEnroll={() => setShowEnrollment(true)} />}
    </main>
    {showCreate && <CreateItemModal onClose={() => setShowCreate(false)} onCreated={(input) => void run(() => createVaultItem(input), "Credential added").then(() => setShowCreate(false))} />}
    {editing && <EditItemModal item={editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); void refresh(); setToast({ kind: "success", message: "Credential updated" }); }} />}
    {showToken && <TokenModal value={ownerToken} onChange={setOwnerToken} onSubmit={saveToken} onClose={() => setShowToken(false)} />}
    {showLogin && <LoginModal onClose={() => setShowLogin(false)} onConnected={() => { setShowLogin(false); setToast({ kind: "success", message: "Signed in for this session" }); void refresh(); }} />}
    {showEnrollment && <EnrollmentModal onClose={() => setShowEnrollment(false)} onConnected={(token) => { setConfiguredOwnerToken(token); setOwnerToken(token); setShowEnrollment(false); setToast({ kind: "success", message: "Owner enrollment completed" }); void refresh(); }} />}
  </div>;
}

function isOwnerAuthError(message: string) {
  const normalized = message.toLowerCase();
  return normalized.includes("authentication") || normalized.includes("access denied") || normalized.includes("owner authentication");
}

function VaultPage({ experienceState, items, query, onQueryChange, selected, onSelect, onBack, locked, onCreate, onReveal, onEdit, onTrash }: { experienceState: "loading" | "ready"; items: VaultItem[]; query: string; onQueryChange: (value: string) => void; selected: VaultItem | null; onSelect: (item: VaultItem | null) => void; onBack: () => void; locked: boolean; onCreate: () => void; onReveal: (item: VaultItem) => void; onEdit: (item: VaultItem) => void; onTrash: (item: VaultItem) => void }) {
  const [filter, setFilter] = useState("all");
  const visible = useMemo(() => items.filter((item) => filter === "all" || item.type === filter), [filter, items]);
  return <section className="content-grid" data-experience-region="vault-collection" data-experience-surface="vault-collection" data-experience-state={experienceState}><div className="content-primary"><div className="section-intro"><div><span className="eyebrow">ENCRYPTED INVENTORY</span><p className="muted">Find and manage credentials without exposing their values to ordinary views.</p></div><button data-testid="vault-new-item" className="button button-primary" onClick={onCreate}><Plus size={16} />New item</button></div><div className="metric-row"><div><span className="metric-value">{items.length}</span><span className="metric-label">Items</span></div><div><span className="metric-value">{items.filter((item) => item.favorite).length}</span><span className="metric-label">Favorites</span></div><div><span className="metric-value">{items.filter((item) => item.type === "login").length}</span><span className="metric-label">Logins</span></div></div><div className="toolbar"><label className="search-box"><Search size={16} /><RclInput type="search" value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder="Search by name, username, or origin" aria-label="Search vault" /></label><select value={filter} onChange={(event) => setFilter(event.target.value)} aria-label="Filter item type"><option value="all">All types</option>{Object.entries(itemLabels).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></div><div className="item-list">{visible.length === 0 ? <div className="empty-state"><Sparkles size={22} /><h2>Your vault is ready</h2><p>Add your first credential. Secret fields are encrypted before they reach durable storage.</p><button data-testid="vault-new-item-empty" className="button button-primary" onClick={onCreate}>Add an item</button></div> : visible.map((item) => <button data-testid={`vault-item-${item.id}`} className={selected?.id === item.id ? "item-row selected" : "item-row"} key={item.id} onClick={() => onSelect(item)}><span className="item-icon">{item.type === "login" ? "↗" : item.type === "secure_note" ? "≡" : "✦"}</span><span className="item-copy"><strong>{item.name}</strong><span>{itemLabels[item.type] || item.type}{item.username ? ` · ${item.username}` : ""}</span></span><span className="item-origin">{item.uri || item.folder || "Personal vault"}</span><span className="item-chevron">›</span></button>)}</div></div><div className="detail-card" data-experience-region="vault-inspector" data-experience-surface="vault-inspector" data-experience-state="static">{selected ? <ItemDetail item={selected} locked={locked} onBack={onBack} onReveal={onReveal} onEdit={onEdit} onTrash={onTrash} /> : <div className="detail-empty"><KeyRound size={28} /><strong>Select an item</strong><span>Metadata and safe actions appear here. Values stay hidden until you explicitly request a reveal.</span></div>}</div></section>;
}

function ItemDetail({ item, locked, onBack, onReveal, onEdit, onTrash }: { item: VaultItem; locked: boolean; onBack: () => void; onReveal: (item: VaultItem) => void; onEdit: (item: VaultItem) => void; onTrash: (item: VaultItem) => void }) {
  const revealed = (item as VaultItem & { __revealed?: Record<string, string> }).__revealed;
  const [history, setHistory] = useState<Awaited<ReturnType<typeof listVaultItemHistory>>["history"]>([]);
  useEffect(() => { void listVaultItemHistory(item.id).then((response) => setHistory(response.history || [])).catch(() => setHistory([])); }, [item.id, item.revision]);
  return <div><button type="button" className="mobile-back" onClick={onBack} aria-label="Back to vault list">← Back to vault</button><div className="detail-header"><div className="item-icon large">{item.type === "login" ? "↗" : "✦"}</div><div><span className="eyebrow">{itemLabels[item.type] || item.type}</span><h2>{item.name}</h2></div></div><div className="detail-fields"><div><span>Username</span><strong>{item.username || "Not set"}</strong></div><div><span>Origin</span><strong>{item.uri || "Not set"}</strong></div><div><span>Revision</span><strong>{item.revision}</strong></div></div>{revealed && <div className="revealed-value"><span>Revealed for this session</span><code>{Object.entries(revealed).map(([key, value]) => `${key}: ${value}`).join("\n")}</code><button className="button button-quiet" onClick={() => navigator.clipboard?.writeText(Object.values(revealed)[0] || "")}><Clipboard size={14} />Copy selected field</button></div>}<div className="detail-actions"><RclButton data-testid="vault-reveal" className="button button-primary" shape="pill" variant="primary" icon={<KeyRound size={15} />} disabled={locked} onClick={() => onReveal(item)}>Reveal selected fields</RclButton><button className="button button-quiet" disabled={locked} onClick={() => onEdit(item)}>Edit after reveal</button><button className="button button-danger" onClick={() => onTrash(item)}>Move to trash</button></div>{history.length > 0 && <div className="history-list"><span className="eyebrow">REVISION HISTORY</span>{history.map((record) => <div className="history-row" key={record.id}><strong>Revision {record.version}</strong><span>{record.changed_by} · {new Date(record.changed_at).toLocaleString()}</span></div>)}</div>}{locked && <p className="inline-warning"><Lock size={14} />Unlock the vault before revealing a value.</p>}</div>;
}

function pageFromPath(pathname: string): PasswordManagerPageId | null {
  const path = pathname.replace(/\/+$/, "") || "/";
  if (path === "/" || path === "/vault") return "vault";
  const page = path.slice(1) as PasswordManagerPageId;
  return ["access", "activity", "recovery", "sources", "settings"].includes(page) ? page : null;
}

function CreateItemModal({ onClose, onCreated }: { onClose: () => void; onCreated: (input: Record<string, unknown>) => void }) { const [type, setType] = useState("login"); const [name, setName] = useState(""); const [username, setUsername] = useState(""); const [uri, setUri] = useState(""); const [password, setPassword] = useState(""); const [notes, setNotes] = useState(""); const [busy, setBusy] = useState(false); const submit = async (event: FormEvent) => { event.preventDefault(); setBusy(true); try { onCreated({ type, name, username, uri, fields: { password, notes } }); } finally { setBusy(false); } }; return <div className="modal-backdrop" role="dialog" aria-modal="true" aria-label="Vault action"><form className="modal-card" onSubmit={submit}><div className="modal-header"><div><span className="eyebrow">NEW CREDENTIAL</span><h2>Add to your vault</h2></div><button type="button" className="icon-button" onClick={onClose} aria-label="Close"><X size={18} /></button></div><label>Item type<select value={type} onChange={(event) => setType(event.target.value)}>{Object.entries(itemLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label><label>Name<input required value={name} onChange={(event) => setName(event.target.value)} placeholder="e.g. Production GitHub" autoFocus /></label>{type === "login" && <><label>Website origin<input value={uri} onChange={(event) => setUri(event.target.value)} placeholder="https://example.com" /></label><label>Username<input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="you@example.com" /></label></>}<RclPasswordInput label="Secret value" value={password} onValueChange={setPassword} placeholder="Paste or generate a value" autoComplete="new-password" /><label>Private notes<textarea value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Only shown after an explicit reveal" /></label><div className="modal-footer"><button type="button" className="button button-quiet" onClick={async () => { const generated = await generatePassword(); setPassword(generated.password); }}>Generate</button><span /><button type="button" className="button button-quiet" onClick={onClose}>Cancel</button><button className="button button-primary" disabled={busy}>{busy ? "Saving…" : "Save encrypted item"}</button></div></form></div>; }

function EditItemModal({ item, onClose, onSaved }: { item: VaultItem; onClose: () => void; onSaved: () => void }) {
  const revealed = (item as VaultItem & { __revealed?: Record<string, string> }).__revealed || {};
  const [name, setName] = useState(item.name);
  const [username, setUsername] = useState(revealed.username || item.username || "");
  const [uri, setUri] = useState(revealed.uri || item.uri || "");
  const [fieldsText, setFieldsText] = useState(JSON.stringify(Object.fromEntries(Object.entries(revealed).filter(([key]) => key !== "username" && key !== "uri")), null, 2));
  const [error, setError] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      const fields = JSON.parse(fieldsText) as Record<string, string>;
      await updateVaultItem(item.id, item.revision, { type: item.type, name, username, uri, folder: item.folder, tags: item.tags, favorite: item.favorite, fields });
      onSaved();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "The credential could not be updated.");
    }
  };
  return <div className="modal-backdrop" role="dialog" aria-modal="true" aria-label="Edit credential"><form className="modal-card" onSubmit={(event) => void submit(event)}><div className="modal-header"><div><span className="eyebrow">REVEALED SESSION</span><h2>Edit credential</h2></div><button type="button" className="icon-button" onClick={onClose} aria-label="Close"><X size={18} /></button></div><p className="muted">Editing requires a fresh reveal. Secret fields remain local to this form until the revision is submitted.</p>{error && <ErrorBoundaryNotice message={error} />}<label>Name<input required value={name} onChange={(event) => setName(event.target.value)} autoFocus /></label><label>Username<input value={username} onChange={(event) => setUsername(event.target.value)} /></label><label>Origin<input value={uri} onChange={(event) => setUri(event.target.value)} /></label><label>Secret fields (JSON)<textarea required value={fieldsText} onChange={(event) => setFieldsText(event.target.value)} rows={8} /></label><div className="modal-footer"><span /><button type="button" className="button button-quiet" onClick={onClose}>Cancel</button><button className="button button-primary">Save revision</button></div></form></div>;
}

function AccessPage({ items, onCreate }: { items: VaultItem[]; onCreate: (input: Record<string, unknown>) => void }) {
  const [principal, setPrincipal] = useState("");
  const [principalType, setPrincipalType] = useState("agent");
  const [item, setItem] = useState("");
  const [selectorMode, setSelectorMode] = useState<"current_snapshot" | "dynamic">("current_snapshot");
  const [operation, setOperation] = useState("use");
  const [target, setTarget] = useState("");
  const [durationHours, setDurationHours] = useState("24");
  const [requests, setRequests] = useState<Awaited<ReturnType<typeof listAccessRequests>>["requests"]>([]);
  const [grants, setGrants] = useState<Awaited<ReturnType<typeof listGrants>>["grants"]>([]);
  const [selectedGrant, setSelectedGrant] = useState<string | null>(null);
  const [effectiveAccess, setEffectiveAccess] = useState<Awaited<ReturnType<typeof getGrantEffectiveAccess>> | null>(null);
  const [loadError, setLoadError] = useState("");
  const [pendingDecision, setPendingDecision] = useState<string | null>(null);
  const [notice, setNotice] = useState("");

  const refreshAccess = useCallback(async () => {
    try {
      const [grantResponse, requestResponse] = await Promise.all([listGrants(), listAccessRequests()]);
      setGrants(grantResponse.grants || []);
      setRequests(requestResponse.requests || []);
      setLoadError("");
    } catch (caught) {
      setLoadError(caught instanceof Error ? caught.message : "Access records could not be loaded. Retry when the authority is available.");
    }
  }, []);

  useEffect(() => { void refreshAccess(); }, [refreshAccess]);

  const inspectGrant = async (grant: Awaited<ReturnType<typeof listGrants>>["grants"][number]) => {
    setSelectedGrant(grant.id);
    try {
      setEffectiveAccess(await getGrantEffectiveAccess(grant.id, grant.item_id, operation));
      setLoadError("");
    } catch (caught) {
      setEffectiveAccess(null);
      setLoadError(caught instanceof Error ? caught.message : "Effective access could not be loaded.");
    }
  };

  const decide = async (requestId: string, digest: string, decision: "approve" | "deny") => {
    setPendingDecision(requestId);
    setNotice("");
    try {
      const assurance = await issueAssurance("access-request:" + requestId + ":" + decision, digest);
      if (decision === "approve") await approveAccessRequest(requestId, digest, assurance.assurance_token);
      else await denyAccessRequest(requestId, digest, assurance.assurance_token);
      setNotice(decision === "approve" ? "Approval acknowledged by the authority." : "Request denied by the authority.");
      await refreshAccess();
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : "The decision could not be recorded.";
      setLoadError(message.toLowerCase().includes("conflict") ? "This request changed while it was open. Reload the review before deciding again." : message);
    } finally {
      setPendingDecision(null);
    }
  };

  const revoke = async (grantId: string) => {
    setNotice("");
    try {
      const result = await revokeGrant(grantId);
      setNotice(result.remote_purge && result.remote_purge !== "not_applicable"
        ? "Authority acknowledged revocation. Remote copies: " + result.remote_purge + "."
        : "Authority acknowledged revocation. Existing copied secrets may require upstream rotation.");
      if (selectedGrant === grantId) setEffectiveAccess(null);
      await refreshAccess();
    } catch (caught) {
      setLoadError(caught instanceof Error ? caught.message : "Revocation failed. The grant remains unchanged.");
    }
  };

  const submitGrant = () => {
    onCreate({
      vault_id: "personal",
      item_id: item,
      principal_type: principalType,
      principal_id: principal,
      selector_mode: selectorMode,
      operations: [operation],
      target: target || undefined,
      expires_in_hours: Number(durationHours) || 24
    });
    setNotice("Grant submission sent to the authority.");
    void refreshAccess();
  };

  return <section className="single-column" data-experience-region="access-workspace" data-experience-surface="access-workspace" data-experience-state="ready">
    <div className="section-intro">
      <div><span className="eyebrow">BOUNDED DELEGATION</span><h2>Access that explains itself</h2><p className="muted">Review who can use a credential, what the authority allows, and what remains outside its control.</p></div>
      {notice && <span className="status-chip online" role="status">{notice}</span>}
    </div>
    {loadError && <ErrorBoundaryNotice message={loadError} />}
    <div className="access-layout">
      <form className="panel-card" onSubmit={(event) => { event.preventDefault(); submitGrant(); }}>
        <div className="panel-heading"><Users size={18} /><div><h3>Create a grant</h3><p>Scope is explicit before the owner submits it.</p></div></div>
        <label>Principal type<select value={principalType} onChange={(event) => setPrincipalType(event.target.value)}><option value="human">Human</option><option value="agent">Agent</option><option value="bot">Bot</option><option value="team">Team</option><option value="workload">Workload</option><option value="device">Device</option></select></label>
        <label>Principal ID<input required value={principal} onChange={(event) => setPrincipal(event.target.value)} placeholder="agent or workload identity" /></label>
        <label>Credential<select required value={item} onChange={(event) => setItem(event.target.value)}><option value="">Choose an item</option>{items.map((entry) => <option key={entry.id} value={entry.id}>{entry.name}</option>)}</select></label>
        <label>Membership mode<select value={selectorMode} onChange={(event) => setSelectorMode(event.target.value as typeof selectorMode)}><option value="current_snapshot">Current members only</option><option value="dynamic">Include future members</option></select></label>
        {selectorMode === "dynamic" && <p className="inline-warning">Future members matching this principal may receive the same access until the grant expires or is revoked.</p>}
        <label>Allowed operation<select value={operation} onChange={(event) => setOperation(event.target.value)}><option value="use">Use credential</option><option value="inject">Inject into a bound browser</option><option value="reveal">Raw read / reveal</option><option value="export">Export</option></select></label>
        {operation === "reveal" || operation === "export" ? <p className="inline-warning">Raw-read permission exposes secret material and is separate from ordinary use.</p> : <p className="grant-summary"><Check size={15} /><span>Use does not grant raw read or export.</span></p>}
        <label>Destination or origin<input value={target} onChange={(event) => setTarget(event.target.value)} placeholder="Optional origin or delivery target" /></label>
        <label>Duration<select value={durationHours} onChange={(event) => setDurationHours(event.target.value)}><option value="1">1 hour</option><option value="24">24 hours</option><option value="168">7 days</option></select></label>
        <button className="button button-primary" disabled={!principal || !item}>Create reviewed grant</button>
      </form>
      <div className="panel-card muted-card" data-testid="access-request-list">
        <div className="panel-heading"><ShieldCheck size={25} /><div><h3>Approval requests</h3><p>Review the requester, scope, destination, expiry, and exposure before deciding.</p></div></div>
        {requests.length === 0 ? <span className="empty-label">No access requests</span> : requests.map((request) => {
          const grant = grants.find((entry) => entry.id === request.grant_id);
          return <article className="request-row" key={request.id} aria-label={"Request " + request.id}>
            <strong>{request.operation} · {request.requested_by}</strong>
            <span>Account: {grant?.principal_id || "unknown"} · Credential: {request.item_id}</span>
            <span>Destination: {grant?.target || "authority scoped"} · Exposure: {request.operation === "use" ? "bounded use" : "secret-bearing operation"}</span>
            <small>Expires {new Date(request.expires_at).toLocaleString()} · Digest {request.request_digest.slice(0, 12)}…</small>
            {request.status === "pending" ? <div className="inline-actions"><button className="button button-primary" disabled={pendingDecision !== null} onClick={() => void decide(request.id, request.request_digest, "approve")}>Approve with fresh assurance</button><button className="button button-danger" disabled={pendingDecision !== null} onClick={() => void decide(request.id, request.request_digest, "deny")}>Deny</button></div> : <span className="status-chip">{request.status}{request.decided_by ? " · decided by " + request.decided_by : ""}</span>}
          </article>;
        })}
      </div>
    </div>
    <div className="panel-card" data-testid="access-grant-list">
      <div className="panel-heading"><ShieldCheck size={18} /><div><h3>Active grants</h3><p>Inspect effective access and revoke authority immediately. Copied values may still require upstream rotation.</p></div></div>
      {grants.length === 0 ? <span className="empty-label">No grants yet</span> : grants.map((grant) => <div className="request-row" key={grant.id}>
        <button className="plain-row-button" onClick={() => void inspectGrant(grant)}><strong>{grant.principal_id} · {grant.principal_type}</strong><span>{grant.operations.join(", ")} · {grant.selector_mode === "dynamic" ? "includes future members" : "current snapshot"} · expires {new Date(grant.expires_at).toLocaleString()}</span><small>{grant.target || "No destination pin"}</small></button>
        <div className="inline-actions"><button className="button button-quiet" onClick={() => void inspectGrant(grant)}>Explain effective access</button>{grant.status === "active" && <button className="button button-danger" onClick={() => void revoke(grant.id)}>Revoke and inspect residuals</button>}</div>
      </div>)}
    </div>
    {effectiveAccess && selectedGrant && <div className="panel-card" data-testid="effective-access-detail">
      <div className="panel-heading"><ShieldCheck size={18} /><div><h3>Effective access</h3><p>Decision is evaluated from selector, operation, target, lifetime, and current membership.</p></div></div>
      <strong>{effectiveAccess.decision === "allow" ? "Allowed for this actor" : "Denied for this actor"}</strong>
      <span>{effectiveAccess.reason}</span>
      <span>Operation: {effectiveAccess.operation} · Exposure: {effectiveAccess.raw_read_permitted ? "raw read permitted" : "use only"}</span>
      {effectiveAccess.future_members && <p className="inline-warning">This grant includes future members matching the dynamic selector.</p>}
    </div>}
  </section>;
}
function ActivityPage({ events, integrity: serverIntegrity, onExport }: { events: AuditEvent[]; integrity: "verified" | "unknown" | "tampered"; onExport: () => void }) {
  const [query, setQuery] = useState("");
  const [outcome, setOutcome] = useState("all");
  const integrity = events.some((event) => event.integrity_status === "tampered") ? "tampered" : events.some((event) => event.integrity_status === "unknown") ? "unknown" : serverIntegrity;
  const visible = events.filter((event) => outcome === "all" || event.outcome === outcome).filter((event) => {
    const searchable = [event.actor_id, event.item_id, event.request_id, event.correlation_id, event.destination, event.action, event.detail].filter(Boolean).join(" ").toLowerCase();
    return !query.trim() || searchable.includes(query.trim().toLowerCase());
  });
  return <section className="single-column" data-experience-region="activity-timeline" data-experience-surface="activity-timeline" data-experience-state="ready">
    <div className="section-intro"><div><span className="eyebrow">AUDIT TRAIL</span><h2>Activity with safe evidence</h2><p className="muted">Sensitive values never belong in the activity stream. Filter by actor, item, request, task, destination, or result.</p></div><div className="inline-actions"><span className={integrity === "verified" ? "status-chip online" : "status-chip"} data-testid="activity-integrity">Evidence {integrity}</span><button className="button button-quiet" onClick={onExport}>Export metadata</button></div></div>
    <div className="toolbar"><label className="search-box"><Search size={16} /><RclInput value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filter actor, item, request, task, destination" aria-label="Filter activity" /></label><select value={outcome} onChange={(event) => setOutcome(event.target.value)} aria-label="Filter activity result"><option value="all">All results</option><option value="success">Success</option><option value="unknown">Unknown</option><option value="denied">Denied</option><option value="failure">Failure</option></select></div>
    {events.length === 0 ? <div className="panel-card empty-state compact"><ShieldCheck size={24} /><h3>No activity yet</h3><p>New vault, grant, reveal, and recovery actions will appear here with metadata only.</p></div> : visible.length === 0 ? <div className="panel-card empty-state compact"><Search size={24} /><h3>No matching activity</h3><p>Change the filters to inspect another safe evidence record.</p></div> : <div className="panel-card activity-list" data-testid="activity-event-list">{visible.map((event) => <div className="activity-row" key={event.id}><span className={event.outcome === "success" ? "activity-icon" : "activity-icon warning"}><Check size={14} /></span><div><strong>{event.action}</strong><span>{event.detail || "Metadata-only event"} · {event.outcome}{event.actor_id ? " · actor " + event.actor_id : ""}{event.destination ? " · " + event.destination : ""}</span></div><time>{new Date(event.created_at).toLocaleString()}</time></div>)}</div>}
  </section>;
}
function SourcesPage({ onCreate }: { onCreate: (input: { kind: string; label: string; endpoint?: string; bootstrap_ref?: string }) => void }) {
  const [sources, setSources] = useState<Source[]>([]);
  const [kind, setKind] = useState("native");
  const [label, setLabel] = useState("");
  const [endpoint, setEndpoint] = useState("");
  const [bootstrapRef, setBootstrapRef] = useState("");
  const [health, setHealth] = useState<Record<string, Source>>({});
  const [error, setError] = useState("");
  const refresh = useCallback(async () => {
    try { setSources((await listSources()).sources || []); setError(""); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Sources could not be loaded. Retry when the authority is available."); }
  }, []);
  useEffect(() => { void refresh(); }, [refresh]);
  const checkHealth = async (source: Source) => {
    try { const observed = await getSourceHealth(source.id); setHealth((current) => ({ ...current, [source.id]: observed })); setError(""); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Source health could not be checked."); }
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    onCreate({ kind, label, endpoint: endpoint || undefined, bootstrap_ref: bootstrapRef || undefined });
    setLabel(""); setEndpoint(""); setBootstrapRef("");
    void refresh();
  };
  return <section className="single-column" data-experience-region="sources-workspace" data-experience-surface="sources-workspace" data-experience-state="ready"><div className="section-intro"><div><span className="eyebrow">EXPLICIT PROVIDERS</span><h2>Credential sources</h2><p className="muted">Register authority metadata first. Capabilities and health are reported by the source owner; unavailable sources do not become silent empty success.</p></div></div>{error && <ErrorBoundaryNotice message={error} />}<div className="access-layout"><form className="panel-card" onSubmit={submit}><div className="panel-heading"><KeyRound size={18} /><div><h3>Register a source</h3><p>Bootstrap references stay out of ordinary responses.</p></div></div><label>Source type<select value={kind} onChange={(event) => setKind(event.target.value)}><option value="native">Native vault</option><option value="onepassword">1Password</option><option value="bitwarden_secrets">Bitwarden Secrets</option><option value="bitwarden_vault">Bitwarden Vault</option></select></label><label>Label<input required value={label} onChange={(event) => setLabel(event.target.value)} placeholder="Production credential authority" /></label>{kind !== "native" && <><label>HTTPS endpoint<input required value={endpoint} onChange={(event) => setEndpoint(event.target.value)} placeholder="https://vault.example" /></label><RclPasswordInput label="Bootstrap reference" value={bootstrapRef} onValueChange={setBootstrapRef} placeholder="Owner-managed secret reference" autoComplete="new-password" revealable={false} /></>}<button className="button button-primary">Register source</button></form><div className="panel-card" data-testid="source-list"><div className="panel-heading"><ShieldCheck size={18} /><div><h3>Registered authorities</h3><p>Capabilities are declarations until an adapter supplies verification evidence.</p></div></div>{sources.length === 0 ? <span className="empty-label">No sources registered</span> : sources.map((source) => { const observed = health[source.id]; return <div className="request-row" key={source.id}><strong>{source.label}</strong><span>{source.kind} · {source.status}{source.endpoint ? " · " + source.endpoint : ""}</span><small>Supported operations: {Object.entries(source.capabilities).filter(([, enabled]) => enabled).map(([name]) => name).join(" · ") || "none declared"}</small><div className="inline-actions"><button className="button button-quiet" onClick={() => void checkHealth(source)}>Check health and capabilities</button>{observed && <span className={observed.status === "healthy" || observed.status === "configured" ? "status-chip online" : "status-chip"}>{observed.status}{observed.last_error ? " · " + observed.last_error : ""}</span>}</div></div>; })}</div></div></section>;
}
function RecoveryPage({ onExport, onRestore, onChanged }: { onExport: (format: "native" | "plaintext") => void; onRestore: (id: string) => Promise<void>; onChanged: () => void }) {
  const [status, setStatus] = useState<Awaited<ReturnType<typeof getRecoveryStatus>> | null>(null);
  const [trash, setTrash] = useState<VaultItem[]>([]);
  const [bundle, setBundle] = useState("");
  const [policy, setPolicy] = useState<"skip" | "rename" | "replace">("skip");
  const [preview, setPreview] = useState<Awaited<ReturnType<typeof previewVaultImport>> | null>(null);
  const [message, setMessage] = useState("");
  const refresh = useCallback(async () => {
    try { const [recovery, items] = await Promise.all([getRecoveryStatus(), listVaultItems("", true)]); setStatus(recovery); setTrash((items.items || []).filter((item) => item.trashed)); setMessage(""); }
    catch (caught) { setMessage(caught instanceof Error ? caught.message : "Recovery status could not be loaded."); }
  }, []);
  useEffect(() => { void refresh(); }, [refresh]);
  const inspectImport = async () => { try { setPreview(await previewVaultImport(bundle)); setMessage(""); } catch (caught) { setPreview(null); setMessage(caught instanceof Error ? caught.message : "Import preview failed."); } };
  const commitImport = async () => { try { const result = await commitVaultImport(bundle, policy); setMessage(`Imported ${result.created} item(s); skipped ${result.skipped}.`); setBundle(""); setPreview(null); await refresh(); onChanged(); } catch (caught) { setMessage(caught instanceof Error ? caught.message : "Import failed."); } };
  const recoveryEvidence = status?.evidence || [];
  return <section className="single-column" data-experience-region="recovery-workspace" data-experience-surface="recovery-workspace" data-experience-state="ready"><div className="section-intro"><div><span className="eyebrow">CONTINUITY</span><h2>Recovery without guesswork</h2><p className="muted">Recovery status is separate from backup status. A verified backup does not claim that a replacement host has been restored.</p></div></div>{message && <ErrorBoundaryNotice message={message} />}<div className="recovery-grid"><div className="panel-card"><span className={status?.key_available ? "status-chip online" : "status-chip"}><ShieldCheck size={14} />{status?.key_available ? "Authority key available" : "Authority key unavailable"}</span><h3>Vault key custody</h3><p>Encrypted item payloads require the configured authority key. Missing key material fails closed and asks for recovery.</p><p className="muted">Backup: {status?.backup_status || "checking"} · Restore: {status?.restore_status || "checking"}</p><span className={status?.recovery_ready ? "status-chip online" : "status-chip"}>{status?.recovery_ready ? "Recovery evidence verified" : "Recovery drill still required"}</span>{status?.remediation && <p className="muted">{status.remediation}</p>}<button className="button button-primary" onClick={() => onExport("native")}>Download encrypted export</button></div><div className="panel-card"><span className="status-chip">Fresh assurance required</span><h3>Explicit plaintext export</h3><p>Plaintext export is a separate action and creates a secret-bearing file only after fresh assurance.</p><button className="button button-danger" onClick={() => onExport("plaintext")}>Export plaintext</button></div></div>{recoveryEvidence.length > 0 && <div className="panel-card"><div className="panel-heading"><ShieldCheck size={18} /><div><h3>Owner evidence</h3><p>These receipts come from {status?.evidence_owner || "the backup owner"}; values and artifact contents are never shown here.</p></div></div>{recoveryEvidence.map((entry) => <div className="request-row" key={`${entry.kind}:${entry.artifact_identity}`}><strong>{entry.kind}</strong><span>{entry.artifact_identity} · {entry.verified ? "verified" : "incomplete"}</span>{entry.checksum && <small>Checksum present · observed {new Date(entry.observed_at).toLocaleString()}</small>}</div>)}</div>}<div className="panel-card"><div className="panel-heading"><Archive size={18} /><div><h3>Restore an encrypted export</h3><p>Preview names and metadata before choosing how duplicate items are handled.</p></div></div><label>Native export JSON<textarea value={bundle} onChange={(event) => setBundle(event.target.value)} rows={7} placeholder="Paste an encrypted native export" /></label><div className="modal-footer"><span /><button className="button button-quiet" disabled={!bundle.trim()} onClick={() => void inspectImport()}>Preview import</button><select value={policy} onChange={(event) => setPolicy(event.target.value as typeof policy)} aria-label="Duplicate policy"><option value="skip">Skip duplicates</option><option value="rename">Rename duplicates</option><option value="replace">Replace duplicates</option></select><button className="button button-primary" disabled={!preview} onClick={() => void commitImport()}>Commit import</button></div>{preview && <div className="grant-summary"><Check size={15} /><span>{preview.item_count} item(s), {preview.duplicate_candidates.length} duplicate candidate(s), {preview.unsupported_items.length} unsupported item(s)</span></div>}</div><div className="panel-card"><div className="panel-heading"><Archive size={18} /><div><h3>Trash</h3><p>Trashed records remain metadata-only until restored.</p></div></div>{trash.length === 0 ? <span className="empty-label">Trash is empty</span> : trash.map((item) => <div className="request-row" key={item.id}><strong>{item.name}</strong><span>{itemLabels[item.type] || item.type} · revision {item.revision}</span><button className="button button-quiet" onClick={() => void onRestore(item.id).then(refresh)}>Restore</button></div>)}</div></section>;
}
function SettingsPage({ onLogin, onToken, onEnroll }: { onLogin: () => void; onToken: () => void; onEnroll: () => void }) {
  return <section className="single-column" data-experience-region="settings-workspace" data-experience-surface="settings-workspace" data-experience-state="static">
    <div className="section-intro"><div><span className="eyebrow">WORKSPACE SETTINGS</span><h2>Keep the boundary visible</h2><p className="muted">Authentication, vault custody, account security, membership, and recovery are separate controls.</p></div></div>
    <div className="panel-card settings-list">
      <div><div><h3>Owner authentication</h3><p>Sign in through Scenario Authenticator. The short lived session token stays in memory and is verified locally by this relying party.</p></div><div className="inline-actions"><button className="button button-primary" onClick={onLogin}>Sign in</button><button className="button button-quiet" onClick={onToken}>Connect browser</button><button className="button button-quiet" onClick={onEnroll}>First-owner setup</button></div></div>
      <div><div><h3>Vault lock</h3><p>Use Lock now in the top bar after a sensitive session. Revealed values are cleared from the UI when the authority confirms the lock.</p></div><span className="status-chip">Available from every page</span></div>
      <div><div><h3>Membership and devices</h3><p>Membership changes, device enrollment, and session revocation are authority actions. This workspace shows their acknowledgement in Access and Activity.</p></div><span className="status-chip">Owner controlled</span></div>
      <div><div><h3>Recovery and MFA</h3><p>Recovery evidence, backup verification, and plaintext export assurance remain separate. Configure MFA through the trusted local authority before enabling a provider.</p></div><a className="button button-quiet" href="/recovery">Open recovery</a></div>
      <div><div><h3>Default vault</h3><p>Personal · encrypted custody · local workspace</p></div><span className="status-chip online">Ready</span></div>
      <div><div><h3>Unsupported claims</h3><p>No zero-knowledge, independent audit, Safari, or passkey-provider claim is made by this release.</p></div><span className="status-chip">Declared</span></div>
    </div>
  </section>;
}
function LoginModal({ onClose, onConnected }: { onClose: () => void; onConnected: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await loginWithAuthenticator(email, password);
      onConnected();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Sign in failed.");
    } finally {
      setBusy(false);
    }
  };
  return <RclDialog open title="Sign in" description="Scenario Authenticator signs you in; Secrets Manager keeps the access token only for this browser session." closeLabel="Close" onClose={onClose} panelClassName="modal-card narrow">
    <form onSubmit={(event) => void submit(event)}>
      {error && <ErrorBoundaryNotice message={error} />}
      <label>Email<input required type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoFocus autoComplete="username" /></label>
      <RclPasswordInput label="Password" value={password} onValueChange={setPassword} autoComplete="current-password" />
      <div className="modal-footer"><span /><button type="button" className="button button-quiet" onClick={onClose}>Cancel</button><button className="button button-primary" disabled={busy}>{busy ? "Signing in…" : "Sign in"}</button></div>
    </form>
  </RclDialog>;
}
function EnrollmentModal({ onClose, onConnected }: { onClose: () => void; onConnected: (token: string) => void }) {
  const [bootstrapToken, setBootstrapToken] = useState("");
  const [error, setError] = useState("");
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      const result = await completeEnrollment(bootstrapToken);
      onConnected(result.owner_token);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Owner enrollment failed.");
    }
  };
  return <div className="modal-backdrop" role="dialog" aria-modal="true" aria-label="First-owner setup"><form className="modal-card narrow" onSubmit={(event) => void submit(event)}><div className="modal-header"><div><span className="eyebrow">ONE-TIME SETUP</span><h2>Enroll the first owner</h2></div><button type="button" className="icon-button" onClick={onClose} aria-label="Close"><X size={18} /></button></div><p className="muted">Use the bootstrap material from the trusted local setup channel. It is accepted once and never returned in the response.</p>{error && <ErrorBoundaryNotice message={error} />}<RclPasswordInput label="Bootstrap material" value={bootstrapToken} onValueChange={setBootstrapToken} autoFocus autoComplete="new-password" revealable={false} /><div className="modal-footer"><span /><button type="button" className="button button-quiet" onClick={onClose}>Cancel</button><button className="button button-primary">Enroll owner</button></div></form></div>;
}
function TokenModal({ value, onChange, onSubmit, onClose }: { value: string; onChange: (value: string) => void; onSubmit: (event: FormEvent) => void; onClose: () => void }) {
  return <RclDialog open title="Connect this browser" description="The token is used for this session and is never written to vault storage." closeLabel="Close" onClose={onClose} panelClassName="modal-card narrow">
    <form onSubmit={onSubmit}>
      <RclPasswordInput label="Owner token" value={value} onValueChange={onChange} autoFocus autoComplete="current-password" revealable={false} />
      <div className="modal-footer"><span /><button type="button" className="button button-quiet" onClick={onClose}>Cancel</button><button className="button button-primary">Connect</button></div>
    </form>
  </RclDialog>;
}
