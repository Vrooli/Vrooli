import { useEffect, useState } from "react";

import { Providers } from "./app/providers";
import { selectors } from "./consts/selectors";

const stateKey = "hello-mobile-state";

function MobileFixture() {
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
		<main className="min-h-dvh bg-slate-950 px-6 py-10 text-slate-100">
			<section className="mx-auto flex max-w-lg flex-col gap-6 rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-xl">
				<p className="text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">Conformance fixture</p>
				<h1 data-testid={selectors.helloMobile.title} className="text-3xl font-bold">Hello Mobile</h1>
				<p data-testid={selectors.helloMobile.route} className="text-sm text-slate-400">Route: home</p>
				<label className="flex flex-col gap-2 text-sm font-medium" htmlFor={selectors.helloMobile.input}>
					Input
					<input
						id={selectors.helloMobile.input}
						data-testid={selectors.helloMobile.input}
						className="min-h-12 rounded-lg border border-slate-600 bg-slate-800 px-3 text-base text-white"
						value={input}
						onChange={(event) => setInput(event.target.value)}
					/>
				</label>
				<button data-testid={selectors.helloMobile.submit} className="min-h-12 rounded-lg bg-cyan-400 px-4 font-bold text-slate-950" type="button" onClick={submit}>
					Transform
				</button>
				<output data-testid={selectors.helloMobile.result} className="min-h-12 rounded-lg border border-slate-700 bg-slate-800 p-3" aria-live="polite">
					{result ? `Result: ${result}` : "Result: waiting for input"}
				</output>
				<p data-testid={selectors.helloMobile.state} className="text-sm text-slate-400">Saved state: {input || "empty"}</p>
				<div className="flex items-center justify-between gap-3 text-sm">
					<p data-testid={selectors.helloMobile.connectivity}>Connectivity: {connectivity}</p>
					<button className="rounded border border-slate-600 px-3 py-2" type="button" onClick={() => setConnectivity((value) => value === "online" ? "offline" : "online")}>
						Toggle connectivity
					</button>
				</div>
				<button data-testid={selectors.helloMobile.notification} className="rounded border border-slate-600 px-3 py-2 text-left" type="button" onClick={() => setResult("NOTIFICATION_OPENED")}>
					Notification: open Hello Mobile
				</button>
			</section>
		</main>
	);
}

/**
 * Top-level app composition. The shell, routing, theme provider, and pages
 * live in `app/`, `layout/`, `pages/`, and `theme/`. This file is intentionally
 * tiny so scenarios that rip out the default routes can do so in one place.
 */
export default function App() {
  return (
    <Providers>
      <MobileFixture />
    </Providers>
  );
}
