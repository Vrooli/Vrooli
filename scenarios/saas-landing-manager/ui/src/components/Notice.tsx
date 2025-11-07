import clsx from 'clsx';

export interface NoticeProps {
  variant: 'success' | 'error';
  title: string;
  description?: string;
}

export const Notice = ({ variant, title, description }: NoticeProps) => (
  <div className={clsx('notice', variant)}>
    <div>
      <strong>{title}</strong>
      {description && <p>{description}</p>}
    </div>
  </div>
);
