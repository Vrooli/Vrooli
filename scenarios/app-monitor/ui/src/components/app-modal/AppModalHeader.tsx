import type { Ref } from 'react';
import clsx from 'clsx';
import { Copy } from 'lucide-react';

interface AppModalHeaderProps {
  titleId: string;
  displayName: string;
  subtitleChips: string[];
  currentUrl: string | null;
  hasCopiedPreviewUrl: boolean;
  onCopyPreviewUrl: () => void;
  onClose: () => void;
  closeButtonRef: Ref<HTMLButtonElement>;
}

/** Modal header with title, scenario/key chips, preview URL bar, and close button. */
export default function AppModalHeader({
  titleId,
  displayName,
  subtitleChips,
  currentUrl,
  hasCopiedPreviewUrl,
  onCopyPreviewUrl,
  onClose,
  closeButtonRef,
}: AppModalHeaderProps) {
  return (
    <div className="modal-header">
      <div className="modal-header__titles">
        <h2 id={titleId}>{displayName}</h2>
        {subtitleChips.length > 0 && (
          <div className="modal-header__meta">
            {subtitleChips.map((chip) => (
              <span className="modal-header__chip" key={chip}>{chip}</span>
            ))}
          </div>
        )}
        {currentUrl && (
          <div className="modal-header__url" title={currentUrl}>
            <span className="modal-header__url-label">Preview URL</span>
            {/* Omit noreferrer so Referer persists for proxy asset routing. */}
            <a
              className="modal-header__url-value"
              href={currentUrl}
              target="_blank"
              rel="noopener"
            >
              {currentUrl}
            </a>
            <button
              type="button"
              className={clsx('modal-header__url-action', { active: hasCopiedPreviewUrl })}
              onClick={onCopyPreviewUrl}
              aria-label="Copy preview URL"
            >
              <Copy size={14} aria-hidden />
              <span className="modal-header__url-action-text">Copy</span>
            </button>
            {hasCopiedPreviewUrl && (
              <span className="modal-header__url-feedback" role="status" aria-live="polite">
                Copied
              </span>
            )}
          </div>
        )}
      </div>
      <button
        type="button"
        className="modal-close"
        onClick={onClose}
        aria-label="Close application details"
        ref={closeButtonRef}
      >
        <span aria-hidden>&times;</span>
      </button>
    </div>
  );
}
