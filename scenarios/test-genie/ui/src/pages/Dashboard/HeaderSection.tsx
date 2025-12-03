import { selectors } from "../../consts/selectors";

const OUTCOMES = [
  {
    title: "Run tests",
    description: "Execute test suites for any scenario with configurable presets."
  },
  {
    title: "Track coverage",
    description: "Monitor test health across all your scenarios in one place."
  },
  {
    title: "Generate tests",
    description: "Create prompts for AI-powered test generation."
  }
];

export function HeaderSection() {
  return (
    <header
      className="rounded-3xl border border-white/10 bg-white/[0.04] p-8 shadow-lg backdrop-blur"
      data-testid={selectors.dashboard.header}
    >
      <p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">
        Test Genie
      </p>
      <h1 className="mt-3 text-3xl font-semibold">Test coverage dashboard</h1>
      <p className="mt-4 text-base text-slate-300">
        Run tests, track coverage, and generate new test cases for your scenarios.
        Everything you need to maintain high-quality test coverage in one place.
      </p>
      <div className="mt-5 grid gap-4 md:grid-cols-3">
        {OUTCOMES.map((outcome) => (
          <article
            key={outcome.title}
            className="rounded-2xl border border-white/5 bg-black/30 p-4"
          >
            <p className="text-sm font-semibold text-white">{outcome.title}</p>
            <p className="mt-2 text-xs text-slate-300">{outcome.description}</p>
          </article>
        ))}
      </div>
    </header>
  );
}
