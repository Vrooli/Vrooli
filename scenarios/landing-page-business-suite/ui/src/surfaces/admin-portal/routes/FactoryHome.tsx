export default function FactoryHome() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <div className="max-w-6xl mx-auto px-6 py-16 space-y-12">
        <header className="space-y-4">
          <p className="inline-flex items-center rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300 border border-emerald-500/30">
            Landing Manager · Factory
          </p>
          <h1 className="text-4xl font-bold">Generate landing-page scenarios, not a landing page.</h1>
          <p className="text-slate-300 max-w-3xl">
            This scenario is the factory. The actual landing experience (public page + admin portal) belongs inside the generated
            landing template, not the factory itself. Use the actions below to work with templates and spin up new landing scenarios.
          </p>
        </header>

        <div className="grid gap-6 md:grid-cols-3">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-3">
            <div className="text-sm text-slate-400">Template</div>
            <div className="text-xl font-semibold">saas-landing-page</div>
            <p className="text-sm text-slate-300">
              Preview now happens inside the generated scenario (to avoid mixing factory and template logic here).
              Generate, start, and view via <code className="px-1 py-0.5 rounded bg-slate-900">vrooli scenario start &lt;name&gt;</code>.
            </p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-3">
            <div className="text-sm text-slate-400">Generation</div>
            <div className="text-xl font-semibold">CLI-first flow</div>
            <p className="text-sm text-slate-300">Use the CLI to generate a new landing scenario and customize it with agents.</p>
            <code className="block rounded-lg bg-slate-900 px-3 py-2 text-xs text-slate-200 border border-white/10">
              landing-manager generate saas-landing-page --name "vrooli-pro" --slug "vrooli-pro"
            </code>
          </div>

          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-6 space-y-3">
            <div className="text-sm text-amber-200/80">Important</div>
            <div className="text-xl font-semibold text-amber-100">Admin portal lives in the template</div>
            <p className="text-sm text-amber-100/90">
              Any admin UX, analytics, or A/B testing logic should be part of the landing template, not this factory. Keep this UI
              focused on template selection and scenario creation.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-4">
          <h2 className="text-2xl font-semibold">Working steps</h2>
          <ol className="list-decimal list-inside space-y-2 text-slate-200 text-sm">
            <li>Generate a new landing scenario via the CLI using the command above.</li>
            <li>Develop and test inside the generated scenario (not here) — admin portal, variants, metrics, and payments belong there.</li>
            <li>Preview landing/admin in the generated scenario on its own port.</li>
            <li>Return here only to evolve templates or add new template types.</li>
          </ol>
        </div>
      </div>
    </div>
  );
}
