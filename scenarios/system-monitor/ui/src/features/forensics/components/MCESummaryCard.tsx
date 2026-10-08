import type { ForensicsEnvelope, MCEReport } from '../types';
import { NotProvisionedCard } from './NotProvisionedCard';

interface MCESummaryCardProps {
  envelope: ForensicsEnvelope<MCEReport>;
}

export const MCESummaryCard = ({ envelope }: MCESummaryCardProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Machine Check Errors" reason={envelope.reason} />;
  }
  const { window, uncorrected, corrected, rawSummary } = envelope.data;
  const errorClass = uncorrected > 0
    ? 'text-error'
    : corrected > 0
      ? 'text-warning'
      : 'text-success';
  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <div className="font-bold" data-sm-style="sm-style-b113dc3b73">
        Machine Check Errors
      </div>
      <div className="text-xs text-muted" data-sm-style="sm-style-b113dc3b73">
        Window: {window}
      </div>
      <div className="flex-row-center gap-md">
        <div>
          <div className="text-xs text-muted">Uncorrected</div>
          <div className={`text-xl font-bold ${errorClass}`}>
            {uncorrected}
          </div>
        </div>
        <div>
          <div className="text-xs text-muted">Corrected</div>
          <div data-sm-style="sm-style-9b38f4bcde">{corrected}</div>
        </div>
      </div>
      {rawSummary && (
        <details data-sm-style="sm-style-fd08d808d2">
          <summary className="text-xs text-muted" data-sm-style="sm-style-667fe49ecb">
            Raw summary
          </summary>
          <pre
            className="text-xs"
            data-sm-style="sm-style-4abf2fa839"
          >
            {rawSummary}
          </pre>
        </details>
      )}
    </div>
  );
};
