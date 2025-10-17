interface StatusStripProps {
  comparisons: number;
  confidence: number;
  progress: number;
  activeListName?: string;
}

const formatConfidence = (confidence: number) => `${Math.min(100, Math.round(confidence))}%`;

export const StatusStrip = ({ comparisons, confidence, progress, activeListName }: StatusStripProps) => {
  return (
    <section className="status-strip grid-3">
      <div className="status-card panel">
        <span className="status-card__label">Comparisons</span>
        <span className="status-card__value">{comparisons}</span>
        <p className="status-card__hint">Signal collected this session.</p>
      </div>
      <div className="status-card panel">
        <span className="status-card__label">Confidence</span>
        <span className="status-card__value">{formatConfidence(confidence)}</span>
        <p className="status-card__hint">Higher means rankings are stabilizing.</p>
      </div>
      <div className="status-progress panel">
        <div className="status-progress__header">
          <span className="status-card__label">List health</span>
          <span className="status-progress__list">{activeListName ?? 'No list selected'}</span>
        </div>
        <div className="status-progress__bar">
          <div className="status-progress__fill" style={{ width: `${Math.min(progress, 100)}%` }} />
        </div>
        <p className="status-card__hint">{Math.round(progress)}% of ideal matchups explored.</p>
      </div>
    </section>
  );
};
