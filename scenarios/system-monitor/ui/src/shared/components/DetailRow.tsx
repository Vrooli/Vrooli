interface DetailRowProps {
  label: string;
  value: string;
  muted?: boolean;
  valueColor?: string;
  className?: string;
}

export const DetailRow = ({ label, value, muted, valueColor, className }: DetailRowProps) => (
  <div className={className ? `detail-row ${className}` : 'detail-row'}>
    <span className="detail-row-label">{label}</span>
    <span
      className={muted ? 'detail-row-value-sm' : 'detail-row-value'}
      style={valueColor ? { color: valueColor } : undefined}
    >
      {value}
    </span>
  </div>
);
