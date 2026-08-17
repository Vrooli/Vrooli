import { useEffect, useState } from "react";

import { Button } from "../components/ui/button";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page. Composes the health card plus stat placeholders.
 * Replace the cards with real surfaces when the scenario grows them.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <MetricPlaceholder label={t(strings.pages.dashboard.statPlaceholderLabel)} />
        <MetricPlaceholder label={t(strings.pages.dashboard.statPlaceholderLabel)} />
      </div>
      <MobileFixture />
    </section>
  );
}

const stateKey = "hello-mobile-state";

/**
 * Deterministic fixture used by the Android and iOS conformance flows. It
 * lives on the real dashboard route so the fixture contract and the routed
 * notes/settings experience are exercised by the same production router.
 */
export function MobileFixture() {
  const [input, setInput] = useState(() => localStorage.getItem(stateKey) ?? "");
  const [result, setResult] = useState("");
  const [connectivity, setConnectivity] = useState("online");

  useEffect(() => {
    if (input) {
      localStorage.setItem(stateKey, input);
    }
  }, [input]);

  const submit = () => {
    // Deliberately pure and stable: the same input always produces the same
    // visible response, including when the app is relaunched on a device.
    setResult(input.trim().split("").reverse().join("").toUpperCase());
  };

  return (
    <Card data-testid="hello-mobile-fixture" aria-labelledby="hello-mobile-fixture-title">
      <CardHeader>
        <CardTitle id="hello-mobile-fixture-title">Conformance fixture</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4">
        <h3 data-testid={selectors.helloMobile.title} className="text-xl font-bold">Hello Mobile</h3>
        <p data-testid={selectors.helloMobile.route} className="text-sm text-app-muted-foreground">Route: home</p>
        <label className="flex flex-col gap-2 text-sm font-medium" htmlFor={selectors.helloMobile.input}>
          Input
          <Input
            id={selectors.helloMobile.input}
            data-testid={selectors.helloMobile.input}
            value={input}
            onChange={(event) => setInput(event.target.value)}
          />
        </label>
        <Button data-testid={selectors.helloMobile.submit} onClick={submit}>
          Transform
        </Button>
        <output data-testid={selectors.helloMobile.result} className="min-h-11 rounded-control border border-app-border bg-app-surface p-3" aria-live="polite">
          {result ? `Result: ${result}` : "Result: waiting for input"}
        </output>
        <p data-testid={selectors.helloMobile.state} className="text-sm text-app-muted-foreground">Saved state: {input || "empty"}</p>
        <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
          <p data-testid={selectors.helloMobile.connectivity}>Connectivity: {connectivity}</p>
          <Button
            size="sm"
            variant="secondary"
            type="button"
            onClick={() => setConnectivity((value) => value === "online" ? "offline" : "online")}
          >
            Toggle connectivity
          </Button>
        </div>
        <Button
          data-testid={selectors.helloMobile.notification}
          variant="secondary"
          className="justify-start text-left"
          onClick={() => setResult("NOTIFICATION_OPENED")}
        >
          Notification: open Hello Mobile
        </Button>
      </CardContent>
    </Card>
  );
}

function MetricPlaceholder({ label }: { label: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm uppercase text-app-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">--</p>
      </CardContent>
    </Card>
  );
}
