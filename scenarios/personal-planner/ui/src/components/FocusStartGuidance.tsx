export function DesktopFocusStartGuidance() {
  return <div className="focus-start-guidance" aria-label="Focus session principles"><div><strong>Open timer</strong><span>Stop when the work stops.</span></div><div><strong>Active time only</strong><span>Pauses stay out of the ledger.</span></div><div><strong>Correctable</strong><span>Report a rough actual later.</span></div></div>;
}

export function MobileFocusStartGuidance() {
  return <aside className="focus-mobile-guidance" aria-label="Focus session principles"><span className="card-kicker">ONE WINDOW</span><strong>Protect the next small stretch.</strong><p>Countdown, pause, and end controls stay close to your thumb. Pauses are not counted as work.</p></aside>;
}
