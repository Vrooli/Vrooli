import { useCallback, useEffect, useMemo, useState } from "react";
import { ExternalLink, Plus, Search, ShieldAlert, UserRoundCheck } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "../components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { jobsApi, type CoverLetter, type Fact, type Opening, type ResumeVariant, type SearchResult } from "../api/jobs";
import { selectors } from "../consts/selectors";

type Stage = "openings" | "facts" | "resumes" | "letters";
const stages: Array<{ id: Stage; label: string }> = [
  { id: "openings", label: "Openings" },
  { id: "facts", label: "Evidence bank" },
  { id: "resumes", label: "Resume versions" },
  { id: "letters", label: "Cover letters" },
];
const BASELINE_RESUME = "__current_resume__";

/** A single-column job workbench: each step is readable and actionable on a phone. */
export function JobReadinessPage() {
  const [stage, setStage] = useState<Stage>("openings");
  const [openings, setOpenings] = useState<Opening[]>([]);
  const [facts, setFacts] = useState<Fact[]>([]);
  const [resumes, setResumes] = useState<ResumeVariant[]>([]);
  const [letters, setLetters] = useState<CoverLetter[]>([]);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [retryAction, setRetryAction] = useState<() => void>(() => () => undefined);
  const [notice, setNotice] = useState("");
  const [jobTitle, setJobTitle] = useState("");
  const [company, setCompany] = useState("");
  const [url, setUrl] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("");
  const [factLabel, setFactLabel] = useState("");
  const [factContent, setFactContent] = useState("");
  const [resumeTitle, setResumeTitle] = useState("");
  const [resumeContent, setResumeContent] = useState("");
  const [letterTitle, setLetterTitle] = useState("");
  const [letterContent, setLetterContent] = useState("");
  const [selectedOpening, setSelectedOpening] = useState("");
  const [selectedFacts, setSelectedFacts] = useState<string[]>([]);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [openingData, factData, resumeData, letterData] = await Promise.all([
        jobsApi.openings(), jobsApi.facts(), jobsApi.resumes(), jobsApi.letters(),
      ]);
      setOpenings(openingData.openings ?? []);
      setFacts(factData.facts ?? []);
      setResumes(resumeData.resumes ?? []);
      setLetters(letterData.cover_letters ?? []);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not load job readiness workspace.");
      setRetryAction(() => () => { void refresh(); });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (!selectedOpening && openings.length) setSelectedOpening(openings[0]?.id ?? "");
  }, [openings, selectedOpening]);

  const currentOpening = useMemo(() => openings.find((opening) => opening.id === selectedOpening), [openings, selectedOpening]);
  const isBaselineResume = selectedOpening === BASELINE_RESUME;
  const withWork = async (action: () => Promise<void>, message: string) => {
    setWorking(true); setError(""); setNotice("");
    try { await action(); setNotice(message); await refresh(); }
    catch (cause) {
      setError(cause instanceof Error ? cause.message : "The change could not be saved. Try again.");
      setRetryAction(() => () => { void withWork(action, message); });
    }
    finally { setWorking(false); }
  };

  const addOpening = () => withWork(async () => {
    if (!jobTitle.trim() || !company.trim()) throw new Error("Add a role title and company first.");
    await jobsApi.addOpening({ title: jobTitle.trim(), company: company.trim(), url: url.trim(), description: description.trim(), category: category.trim() || "Needs review", status: "saved", source: "manual" });
    setJobTitle(""); setCompany(""); setUrl(""); setDescription(""); setCategory("");
  }, "Opening saved. You can categorize it now or return later.");

  const addFact = () => withWork(async () => {
    if (!factLabel.trim() || !factContent.trim()) throw new Error("Add a short label and the exact evidence you want to reuse.");
    await jobsApi.addFact({ label: factLabel.trim(), content: factContent.trim() });
    setFactLabel(""); setFactContent("");
  }, "Evidence saved for reuse.");

  const addResume = () => withWork(async () => {
    if (!currentOpening && !isBaselineResume) throw new Error("Choose the current resume or save an opening first.");
    if (!resumeTitle.trim() || !resumeContent.trim()) throw new Error("Add a version name and resume text.");
    await jobsApi.addResume({ opening_id: currentOpening?.id, title: resumeTitle.trim(), content: resumeContent.trim(), fact_ids: selectedFacts });
    setResumeTitle(""); setResumeContent("");
  }, "Resume version saved. Run the hostile review before relying on it.");

  const addLetter = () => withWork(async () => {
    if (!currentOpening) throw new Error("Save an opening before drafting a tailored letter.");
    if (!letterTitle.trim() || !letterContent.trim()) throw new Error("Add a draft name and letter text.");
    await jobsApi.addLetter({ opening_id: currentOpening.id, title: letterTitle.trim(), content: letterContent.trim(), fact_ids: selectedFacts });
    setLetterTitle(""); setLetterContent("");
  }, "Cover letter saved with its supporting facts.");

  const discover = () => withWork(async () => {
    if (!searchQuery.trim()) throw new Error("Add a role, skill, or location to search for.");
    const result = await jobsApi.discover(searchQuery.trim());
    setSearchResults(result.results ?? []);
    if ((result.results ?? []).length === 0) setNotice("No search sources are configured yet. Add an opening by URL, or use a saved search link below.");
  }, "Search finished. Each source reports its own result so one outage does not stop the rest.");

  const review = (resume: ResumeVariant) => withWork(async () => { await jobsApi.reviewResume(resume.id); }, "Hostile review refreshed. Read each finding before using this version.");
  const saveFoundOpening = (opening: Opening, source: string) => withWork(async () => {
    await jobsApi.addOpening({ company: opening.company, title: opening.title, location: opening.location, url: opening.url, description: opening.description, category: opening.category || "Needs review", status: "saved", source });
  }, "Opening saved to your inbox.");
  return <main data-testid={selectors.pages.jobs} className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-1 pb-24" aria-labelledby="jobs-heading">
    <header className="rounded-2xl border border-app-border bg-app-surface p-5 sm:p-7">
      <p className="text-xs font-semibold uppercase tracking-[0.16em] text-app-primary">Private job preparation</p>
      <h1 id="jobs-heading" className="mt-2 text-2xl font-semibold sm:text-3xl">Move your job search forward</h1>
      <p className="mt-2 max-w-2xl text-sm leading-6 text-app-muted-foreground">Collect promising roles, keep your experience grounded in evidence, and prepare role-specific resumes and letters. Your drafts stay here; nothing is submitted for you.</p>
      <div className="mt-5 grid grid-cols-3 gap-2 text-center sm:max-w-xl">
        <Summary value={openings.length} label="openings" />
        <Summary value={facts.length} label="verified facts" />
        <Summary value={resumes.length + letters.length} label="drafts" />
      </div>
    </header>

    <nav aria-label="Job preparation steps" className="sticky top-0 z-10 -mx-1 flex gap-2 overflow-x-auto bg-app-background/95 px-1 py-2 backdrop-blur">
      {stages.map((item) => <button key={item.id} type="button" aria-current={stage === item.id ? "step" : undefined} onClick={() => setStage(item.id)} className={`min-h-11 shrink-0 rounded-full border px-4 text-sm font-medium ${stage === item.id ? "border-app-primary bg-app-primary text-app-primary-foreground" : "border-app-border bg-app-surface text-app-foreground"}`}>{item.label}<span className="ml-2 opacity-75">{item.id === "openings" ? openings.length : item.id === "facts" ? facts.length : item.id === "resumes" ? resumes.length : letters.length}</span></button>)}
    </nav>

    {error && <div role="alert" className="rounded-xl border border-app-danger bg-app-surface p-4 text-sm"><p>{error}</p><Button className="mt-3" onClick={retryAction}>Retry</Button></div>}
    {notice && <p role="status" className="rounded-xl border border-app-border bg-app-surface p-3 text-sm">{notice}</p>}
    {loading ? <p role="status" className="py-8 text-center text-app-muted-foreground">Loading your saved job preparation…</p> : <>
      {stage === "openings" && <section className="grid gap-4 lg:grid-cols-[minmax(0,1.15fr)_minmax(18rem,.85fr)]" aria-label="Opening inbox">
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3"><div><h2 className="text-xl font-semibold">Opening inbox</h2><p className="text-sm text-app-muted-foreground">Keep the source and role details with every opportunity.</p></div><Button disabled={working} onClick={discover}><Search className="mr-2 h-4 w-4" />Find roles</Button></div>
          {searchResults.map((result, index) => <SearchOutcome key={`${result.source}-${index}`} result={result} onSave={(opening) => void saveFoundOpening(opening, result.source)} working={working} />)}
          {openings.length === 0 ? <Card><CardContent className="py-8 text-center"><h3 className="font-medium">Your inbox is ready</h3><p className="mt-1 text-sm text-app-muted-foreground">Save a role you find, or run the configured search sources. Search outages are shown per source.</p></CardContent></Card> : openings.map((opening) => <Card key={opening.id} className={selectedOpening === opening.id ? "border-app-primary" : ""}><CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between"><div className="min-w-0"><p className="text-xs uppercase tracking-wide text-app-muted-foreground">{opening.company}{opening.location ? ` · ${opening.location}` : ""}{opening.source ? ` · via ${opening.source}` : ""}</p><h3 className="mt-1 text-lg font-semibold">{opening.title}</h3><p className="mt-2 text-sm leading-5 text-app-muted-foreground">{opening.description || "No role description saved yet."}</p><div className="mt-3 flex flex-wrap items-center gap-2"><label className="flex min-h-11 items-center gap-2 text-xs text-app-muted-foreground">Category<select aria-label={`Categorize ${opening.title} at ${opening.company}`} className="min-h-10 rounded-full border border-app-border bg-app-surface px-3 text-xs text-app-foreground" value={opening.category || "Needs review"} onChange={(event) => void withWork(async () => { await jobsApi.updateOpening(opening.id, { category: event.target.value }); }, "Opening category updated.")}>{["Needs review", "Engineering", "Product", "Design", "Leadership", "Operations", "Other"].map((value) => <option key={value}>{value}</option>)}</select></label><span className="rounded-full bg-app-muted px-3 py-1 text-xs">{opening.status || "saved"}</span>{opening.url && <a className="inline-flex min-h-9 items-center gap-1 text-xs text-app-primary underline" href={opening.url} target="_blank" rel="noreferrer">View source <ExternalLink className="h-3 w-3" /></a>}</div></div><Button className="shrink-0" onClick={() => { setSelectedOpening(opening.id); setStage("resumes"); }} aria-label={`Prepare for ${opening.title} at ${opening.company}`}>Prepare</Button></CardContent></Card>)}
        </div>
        <div className="flex flex-col gap-4">
          <Card><CardHeader><CardTitle className="text-lg">Save an opening</CardTitle><CardDescription>Paste a job link and the description. You can review it on your phone later.</CardDescription></CardHeader><CardContent className="flex flex-col gap-3"><Field label="Role title" value={jobTitle} onChange={setJobTitle} placeholder="Product designer" /><Field label="Company" value={company} onChange={setCompany} placeholder="Company name" /><Field label="Source link" value={url} onChange={setUrl} placeholder="https://…" type="url" /><label className="flex flex-col gap-1 text-sm">Role description<textarea className="min-h-28 rounded-control border border-app-border bg-app-surface p-3 text-base" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Paste the requirements and responsibilities" /></label><Field label="Category (optional)" value={category} onChange={setCategory} placeholder="Product, engineering, leadership…" /><Button disabled={working || !jobTitle.trim() || !company.trim()} onClick={addOpening}><Plus className="mr-2 h-4 w-4" />Save opening</Button></CardContent></Card>
          <Card><CardHeader><CardTitle className="text-lg">Find promising roles</CardTitle><CardDescription>Search Remote OK listings and open matching searches in LinkedIn, Indeed, and Google Jobs. A provider outage is isolated beside that source.</CardDescription></CardHeader><CardContent className="flex flex-col gap-3"><Field label="Role, skills, or location" value={searchQuery} onChange={setSearchQuery} placeholder="e.g. product leadership, remote" /><Button variant="outline" disabled={working || !searchQuery.trim()} onClick={discover}><Search className="mr-2 h-4 w-4" />Run searches</Button><p className="text-xs text-app-muted-foreground">On search, these keywords are sent to Remote OK. Your resume and evidence stay local. Other job-board links work without API credentials; new provider adapters can be added as needed.</p></CardContent></Card>
        </div>
      </section>}

      {stage === "facts" && <section className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,.8fr)]"><div><h2 className="text-xl font-semibold">Evidence bank</h2><p className="mb-4 text-sm text-app-muted-foreground">Save exact experience, outcomes, and skills once. Resume and letter drafts link back to the facts they use.</p>{facts.length === 0 ? <Card><CardContent className="py-8 text-center"><h3 className="font-medium">Start with a few strong facts</h3><p className="mt-1 text-sm text-app-muted-foreground">Use specific wording you can stand behind. The hostile review flags unsupported claims.</p></CardContent></Card> : <ul className="flex flex-col gap-3">{facts.map((fact) => <li key={fact.id}><Card><CardContent className="p-4"><p className="font-semibold">{fact.label}</p><p className="mt-1 whitespace-pre-wrap text-sm leading-6">{fact.content}</p></CardContent></Card></li>)}</ul>}</div><Card className="h-fit"><CardHeader><CardTitle className="text-lg">Add a fact</CardTitle><CardDescription>One accomplishment, skill, or credential per record.</CardDescription></CardHeader><CardContent className="flex flex-col gap-3"><Field label="Short label" value={factLabel} onChange={setFactLabel} placeholder="Reduced onboarding time" /><label className="flex flex-col gap-1 text-sm">Evidence and exact details<textarea className="min-h-36 rounded-control border border-app-border bg-app-surface p-3 text-base" value={factContent} onChange={(event) => setFactContent(event.target.value)} placeholder="What happened? Include only details you can verify." /></label><Button disabled={working || !factLabel.trim() || !factContent.trim()} onClick={addFact}><Plus className="mr-2 h-4 w-4" />Save fact</Button></CardContent></Card></section>}

      {stage === "resumes" && <section className="grid gap-4 lg:grid-cols-[minmax(18rem,.8fr)_minmax(0,1.2fr)]"><div className="flex flex-col gap-4"><Card><CardHeader><CardTitle className="text-lg">Build a resume version</CardTitle><CardDescription>Save the current version as a baseline, or choose a role, attach supporting facts, and edit a tailored draft.</CardDescription></CardHeader><CardContent className="flex flex-col gap-3"><label className="flex flex-col gap-1 text-sm">Target opening<select className="min-h-11 rounded-control border border-app-border bg-app-surface px-3" value={selectedOpening} onChange={(event) => setSelectedOpening(event.target.value)}><option value="">Choose an opening</option><option value={BASELINE_RESUME}>Current resume (baseline)</option>{openings.map((opening) => <option key={opening.id} value={opening.id}>{opening.title} · {opening.company}</option>)}</select></label><Field label="Version name" value={resumeTitle} onChange={setResumeTitle} placeholder={isBaselineResume ? "Current resume" : currentOpening ? `${currentOpening.title} version` : "Role-specific version"} />
            <FactPicker facts={facts} selected={selectedFacts} onToggle={toggle => setSelectedFacts((current) => current.includes(toggle) ? current.filter((id) => id !== toggle) : [...current, toggle])} />
            <label className="flex flex-col gap-1 text-sm">Resume draft<textarea className="min-h-64 rounded-control border border-app-border bg-app-surface p-3 text-base" value={resumeContent} onChange={(event) => setResumeContent(event.target.value)} placeholder="Paste or write your role-specific resume. Keep factual claims linked to the selected evidence." /></label><Button disabled={working || !selectedOpening || !resumeTitle.trim() || !resumeContent.trim()} onClick={addResume}>Save version</Button>{!selectedFacts.length && <p className="text-xs text-app-muted-foreground">No facts selected. A hostile review may flag claims that lack evidence.</p>}</CardContent></Card></div>
          <div className="flex flex-col gap-3"><div><h2 className="text-xl font-semibold">Saved versions</h2><p className="text-sm text-app-muted-foreground">Review findings challenge unsupported claims, missing evidence, and role fit.</p></div>{resumes.length === 0 ? <Card><CardContent className="py-8 text-center"><h3 className="font-medium">No resume versions yet</h3><p className="mt-1 text-sm text-app-muted-foreground">Save your current resume as a baseline, then create focused versions for openings. The original stays unchanged.</p></CardContent></Card> : resumes.map((resume) => <Card key={resume.id}><CardHeader><div className="flex items-start justify-between gap-3"><div><CardTitle className="text-lg">{resume.title}</CardTitle><CardDescription>{openings.find((opening) => opening.id === resume.opening_id)?.title || (resume.opening_id ? "Role-specific draft" : "Current resume baseline")}</CardDescription></div><Button variant="outline" disabled={working} onClick={() => review(resume)}><ShieldAlert className="mr-2 h-4 w-4" />Hostile review</Button></div></CardHeader><CardContent><p className="whitespace-pre-wrap text-sm leading-6">{resume.content}</p><FindingList findings={resume.review_findings ?? []} /></CardContent></Card>)}</div>
        </section>}

      {stage === "letters" && <section className="grid gap-4 lg:grid-cols-[minmax(18rem,.8fr)_minmax(0,1.2fr)]"><Card className="h-fit"><CardHeader><CardTitle className="text-lg">Practice a tailored letter</CardTitle><CardDescription>Write to the role and company. Every personal claim can point to evidence.</CardDescription></CardHeader><CardContent className="flex flex-col gap-3"><label className="flex flex-col gap-1 text-sm">Target opening<select className="min-h-11 rounded-control border border-app-border bg-app-surface px-3" value={selectedOpening} onChange={(event) => setSelectedOpening(event.target.value)}><option value="">Choose an opening</option>{openings.map((opening) => <option key={opening.id} value={opening.id}>{opening.title} · {opening.company}</option>)}</select></label><Field label="Draft name" value={letterTitle} onChange={setLetterTitle} placeholder="Short note for Company" /><FactPicker facts={facts} selected={selectedFacts} onToggle={toggle => setSelectedFacts((current) => current.includes(toggle) ? current.filter((id) => id !== toggle) : [...current, toggle])} /><div className="rounded-xl bg-app-muted/40 p-3 text-sm"><p className="font-semibold">A clear cover letter answers three things</p><ol className="mt-2 list-decimal space-y-1 pl-5 text-app-muted-foreground"><li>Which part of this role fits your experience?</li><li>What specific, evidenced result shows that fit?</li><li>Why are you interested in this team or work?</li></ol><Button className="mt-3" variant="outline" disabled={!currentOpening} onClick={() => setLetterContent(`Dear ${currentOpening?.company || "Hiring team"} team,\n\nI am interested in the ${currentOpening?.title || "role"} because [connect one role requirement to your experience].\n\nFor example, [describe one result using a fact from your evidence bank].\n\nI would welcome a conversation about how I could contribute to [specific team goal or responsibility].\n\nRegards,\n[Your name]`)}>Insert guided outline</Button></div><label className="flex flex-col gap-1 text-sm">Letter draft<textarea className="min-h-64 rounded-control border border-app-border bg-app-surface p-3 text-base" value={letterContent} onChange={(event) => setLetterContent(event.target.value)} placeholder="Choose an opening, then use the guided outline or write a concise, specific letter." /></label><Button disabled={working || !selectedOpening || !letterTitle.trim() || !letterContent.trim()} onClick={addLetter}>Save letter</Button><p className="text-xs text-app-muted-foreground">This is a private practice draft. No application is sent.</p></CardContent></Card><div className="flex flex-col gap-3"><div><h2 className="text-xl font-semibold">Saved letters</h2><p className="text-sm text-app-muted-foreground">Keep a distinct, evidence-linked draft for each opportunity.</p></div>{letters.length === 0 ? <Card><CardContent className="py-8 text-center"><h3 className="font-medium">No cover letters yet</h3><p className="mt-1 text-sm text-app-muted-foreground">Prepare and save a short, targeted draft after choosing an opening.</p></CardContent></Card> : letters.map((letter) => <Card key={letter.id}><CardHeader><CardTitle className="text-lg">{letter.title}</CardTitle><CardDescription>{openings.find((opening) => opening.id === letter.opening_id)?.title || "Tailored draft"}</CardDescription></CardHeader><CardContent><p className="whitespace-pre-wrap text-sm leading-6">{letter.content}</p><p className="mt-3 text-xs text-app-muted-foreground">Evidence attached: {letter.fact_ids.length}</p></CardContent></Card>)}</div></section>}
    </>}

    <footer className="flex items-start gap-3 rounded-xl border border-app-border p-4 text-sm"><UserRoundCheck className="mt-0.5 h-5 w-5 shrink-0 text-app-primary" /><p><strong>You stay in control.</strong> Job discovery can be prepared and drafts can be reviewed here. Applying, contacting an employer, or publishing anything always remains your action.</p></footer>
  </main>;
}

function Summary({ value, label }: { value: number; label: string }) { return <div className="rounded-xl bg-app-muted/50 px-2 py-3"><p className="text-xl font-semibold">{value}</p><p className="text-xs text-app-muted-foreground">{label}</p></div>; }
function Field({ label, value, onChange, placeholder, type = "text" }: { label: string; value: string; onChange: (value: string) => void; placeholder?: string; type?: string }) { return <label className="flex flex-col gap-1 text-sm">{label}<Input type={type} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} /></label>; }
function FactPicker({ facts, selected, onToggle }: { facts: Fact[]; selected: string[]; onToggle: (id: string) => void }) { return <fieldset className="rounded-xl border border-app-border p-3"><legend className="px-1 text-sm font-medium">Supporting facts</legend>{facts.length ? <div className="flex max-h-40 flex-col gap-2 overflow-y-auto">{facts.map((fact) => <label key={fact.id} className="flex min-h-11 cursor-pointer items-start gap-3 rounded-lg p-2 text-sm hover:bg-app-muted/50"><input className="mt-1 h-4 w-4" type="checkbox" checked={selected.includes(fact.id)} onChange={() => onToggle(fact.id)} /><span><strong>{fact.label}</strong><span className="block text-xs text-app-muted-foreground">{fact.content.slice(0, 115)}{fact.content.length > 115 ? "…" : ""}</span></span></label>)}</div> : <p className="text-sm text-app-muted-foreground">Save facts in Evidence bank to link them here.</p>}</fieldset>; }
function FindingList({ findings }: { findings: ResumeVariant["review_findings"] }) { if (!findings?.length) return <p className="mt-4 rounded-lg bg-app-muted/50 p-3 text-sm text-app-muted-foreground">No hostile review yet. Run review before using this version.</p>; return <section className="mt-4 rounded-xl border border-app-warning p-4" aria-label="Hostile review findings"><h4 className="font-semibold">Review findings</h4><ul className="mt-2 list-disc space-y-2 pl-5 text-sm">{findings.map((finding, index) => <li key={index}>{typeof finding === "string" ? finding : <><strong>{finding.severity || "Check"}</strong>{finding.claim ? ` · ${finding.claim}` : ""}{finding.reason ? `: ${finding.reason}` : ""}{finding.evidence ? ` — ${finding.evidence}` : ""}</>}</li>)}</ul></section>; }
function SearchOutcome({ result, onSave, working }: { result: SearchResult; onSave: (opening: Opening) => void; working: boolean }) { return <Card><CardContent className="flex flex-col gap-3 p-4"><div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"><div><p className="font-medium">{result.source}</p><p className="text-sm text-app-muted-foreground">{result.error || `${result.status}: ${result.query}`}</p></div>{result.url && <a className="inline-flex min-h-11 items-center gap-2 text-sm font-medium text-app-primary underline" target="_blank" rel="noreferrer" href={result.url}>Open search <ExternalLink className="h-4 w-4" /></a>}</div>{result.openings?.map((opening, index) => <div key={opening.id || `${opening.company}-${opening.title}-${index}`} className="flex flex-col gap-2 rounded-xl border border-app-border p-3 sm:flex-row sm:items-center sm:justify-between"><div><p className="font-medium">{opening.title}</p><p className="text-sm text-app-muted-foreground">{opening.company}{opening.location ? ` · ${opening.location}` : ""}</p>{opening.url && <a href={opening.url} target="_blank" rel="noreferrer" className="text-xs text-app-primary underline">View role</a>}</div><Button variant="outline" disabled={working} onClick={() => onSave(opening)}>Save role</Button></div>)}</CardContent></Card>; }
