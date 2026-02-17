interface DetailRowProps {
  label: string;
  value: string;
  muted?: boolean;
}

export const DetailRow = ({ label, value, muted }: DetailRowProps) => (
  <div className="detail-row">
    <span className="detail-row-label">{label}</span>
    <span className={muted ? 'detail-row-value-sm' : 'detail-row-value'}>{value}</span>
  </div>
);
