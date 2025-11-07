import clsx from 'clsx';

export interface Template {
  id: string;
  name: string;
  category: string;
  saas_type: string;
  industry: string;
  preview_url?: string;
  usage_count?: number;
  rating?: number;
}

export interface TemplateCardProps {
  template: Template;
  selected: boolean;
  onSelect: () => void;
}

export const TemplateCard = ({ template, selected, onSelect }: TemplateCardProps) => (
  <button type="button" className={clsx('template-card', { selected })} onClick={onSelect}>
    <div className="template-card-header">
      <h4>{template.name}</h4>
      <span>{template.saas_type?.replace(/_/g, ' ') || 'General'}</span>
    </div>
    <p>{template.industry ? `${template.industry} • ${template.category}` : template.category}</p>
    <div className="template-card-footer">
      {template.usage_count !== undefined && <span>{template.usage_count} deployments</span>}
      {template.rating !== undefined && <span>{template.rating.toFixed(1)}★</span>}
    </div>
  </button>
);
