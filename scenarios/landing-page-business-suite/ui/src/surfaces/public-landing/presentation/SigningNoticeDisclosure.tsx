import { useEffect, useId, useRef, useState } from 'react';
import type { SigningNotice } from '../../../shared/api';
import { safeHref } from './links';
import { downloadSystemUi as ui } from './systemUi';

function InfoIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="1.6" /><path d="M12 11v5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /><circle cx="12" cy="7.75" r="1.1" fill="currentColor" /></svg>;
}

/**
 * Press-to-open popover for an unsigned-build / signing-pending notice.
 * Rendered only where a download actually happens, never on the marketing page.
 */
export function SigningNoticeDisclosure({ notice }: { notice: SigningNotice }) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => { if (event.key === 'Escape') setOpen(false); };
    const onPointer = (event: MouseEvent) => { if (container.current && !container.current.contains(event.target as Node)) setOpen(false); };
    document.addEventListener('keydown', onKey);
    document.addEventListener('mousedown', onPointer);
    return () => { document.removeEventListener('keydown', onKey); document.removeEventListener('mousedown', onPointer); };
  }, [open]);
  const href = safeHref(notice.link_url);
  return <div className={`signing-notice severity-${notice.severity}`} ref={container}>
    <button type="button" aria-expanded={open} aria-controls={`${id}-panel`} onClick={() => { setOpen(value => !value); }}>
      <InfoIcon />
      <span>{ui.signingNotice.summary}</span>
    </button>
    {open && <div className="signing-notice-panel" id={`${id}-panel`} role="dialog" aria-label={notice.title}>
      <p className="signing-notice-title">{notice.title}</p>
      <p className="signing-notice-body">{notice.body}</p>
      {href && <a href={href} target="_blank" rel="noreferrer noopener">{notice.link_label ?? ui.signingNotice.linkFallback}</a>}
    </div>}
  </div>;
}
