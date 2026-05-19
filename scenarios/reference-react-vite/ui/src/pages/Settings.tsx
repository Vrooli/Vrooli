import { Link } from "react-router-dom";
import { ROUTES } from "../routes.generated";
import { useAppContext } from "../contexts/AppContext";

const cards = [
  { to: ROUTES.settingsDisplay, label: "Display", testId: "settings-card-display", desc: "Themes and density" },
  { to: ROUTES.settingsNotifications, label: "Notifications", testId: "settings-card-notifications", desc: "Email & in-app alerts" },
  { to: ROUTES.settingsAbout, label: "About", testId: "settings-card-about", desc: "Build info & licenses" },
];

export function Settings() {
  const { featureBeta, setFeatureBeta } = useAppContext();
  return (
    <div data-testid="settings-page" className="flex flex-col gap-6">
      <header>
        <h2 className="text-2xl font-semibold">Settings</h2>
        <p className="text-sm text-slate-400">Choose a sub-page to configure.</p>
      </header>
      <ul data-testid="settings-cards" className="grid gap-3 md:grid-cols-3">
        {cards.map((c) => (
          <li key={c.to}>
            <Link
              to={c.to}
              data-testid={c.testId}
              className="block rounded-lg border border-white/10 bg-slate-900/40 p-4 hover:bg-white/5"
            >
              <div className="font-medium">{c.label}</div>
              <div className="text-xs text-slate-400">{c.desc}</div>
            </Link>
          </li>
        ))}
      </ul>
      <label className="flex items-center gap-2 text-sm text-slate-300">
        <input
          type="checkbox"
          data-testid="settings-feature-beta-toggle"
          checked={featureBeta}
          onChange={(e) => setFeatureBeta(e.target.checked)}
        />
        Enable beta features
      </label>
    </div>
  );
}
