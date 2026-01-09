import type { TechStackInfo, App } from '@/types';
import { Database, Layers, Wifi, Package } from 'lucide-react';
import './TechStackTab.css';

interface TechStackTabProps {
  app: App | null | undefined;
  techStack: TechStackInfo | null | undefined;
  loading?: boolean;
}

export default function TechStackTab({ app, techStack, loading }: TechStackTabProps) {
  if (loading) {
    return (
      <div className="tech-stack-tab">
        <div className="tech-stack-tab__loading">
          <Package size={32} className="tech-stack-tab__loading-icon" />
          <p>Loading tech stack information...</p>
        </div>
      </div>
    );
  }

  // Prefer app.tech_stack (components from status command) over techStack from diagnostics
  const components = app?.tech_stack && app.tech_stack.length > 0 ? app.tech_stack : [];
  const dependencies = app?.dependencies && app.dependencies.length > 0 ? app.dependencies : [];
  const hasPorts = techStack?.ports && Object.keys(techStack.ports).length > 0;
  const hasTags = techStack?.tags && techStack.tags.length > 0;

  const hasAnyData = components.length > 0 || dependencies.length > 0 || hasPorts || hasTags;

  if (!hasAnyData) {
    return (
      <div className="tech-stack-tab">
        <div className="tech-stack-tab__empty">
          <Layers size={32} />
          <p>No tech stack information available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="tech-stack-tab">
      {/* Tech Stack Components */}
      {components.length > 0 && (
        <section className="tech-stack-section">
          <h3 className="tech-stack-section__title">
            <Package size={18} />
            <span>Tech Stack</span>
          </h3>
          <div className="tech-stack-tags">
            {components.map((component) => (
              <span key={component} className="tech-stack-tag">{component}</span>
            ))}
          </div>
        </section>
      )}

      {/* Dependencies (full resource info from vrooli status) */}
      {dependencies.length > 0 && (
        <section className="tech-stack-section">
          <h3 className="tech-stack-section__title">
            <Database size={18} />
            <span>Dependencies</span>
          </h3>
          <div className="tech-stack-resources">
            {dependencies.map((dep, index) => (
              <div
                key={`${dep.name}-${index}`}
                className={`tech-stack-resource ${dep.running && dep.healthy ? 'tech-stack-resource--enabled' : 'tech-stack-resource--disabled'}`}
              >
                <div className="tech-stack-resource__header">
                  <span className="tech-stack-resource__type">
                    {dep.name}
                    {dep.type && dep.type !== dep.name && (
                      <span className="tech-stack-resource__subtype"> ({dep.type})</span>
                    )}
                  </span>
                  <div className="tech-stack-resource__badges">
                    {dep.required && (
                      <span className="tech-stack-badge tech-stack-badge--required">Required</span>
                    )}
                    {dep.installed && (
                      <span className="tech-stack-badge tech-stack-badge--info">Installed</span>
                    )}
                    {dep.enabled && (
                      <span className="tech-stack-badge tech-stack-badge--enabled">Enabled</span>
                    )}
                    {dep.running && (
                      <span className="tech-stack-badge tech-stack-badge--success">Running</span>
                    )}
                    {dep.running && dep.healthy && (
                      <span className="tech-stack-badge tech-stack-badge--healthy">Healthy</span>
                    )}
                    {dep.running && !dep.healthy && (
                      <span className="tech-stack-badge tech-stack-badge--unhealthy">Unhealthy</span>
                    )}
                    {!dep.running && dep.enabled && (
                      <span className="tech-stack-badge tech-stack-badge--stopped">Stopped</span>
                    )}
                  </div>
                </div>
                {dep.description && (
                  <p className="tech-stack-resource__purpose">{dep.description}</p>
                )}
                {dep.note && (
                  <p className="tech-stack-resource__note">{dep.note}</p>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Ports */}
      {hasPorts && (
        <section className="tech-stack-section">
          <h3 className="tech-stack-section__title">
            <Wifi size={18} />
            <span>Port Allocations</span>
          </h3>
          <div className="tech-stack-ports">
            {Object.entries(techStack.ports!).map(([name, port]) => (
              <div key={name} className="tech-stack-port">
                <span className="tech-stack-port__name">{name}</span>
                <span className="tech-stack-port__value">{port}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Tags */}
      {hasTags && (
        <section className="tech-stack-section">
          <h3 className="tech-stack-section__title">
            <Layers size={18} />
            <span>Tags</span>
          </h3>
          <div className="tech-stack-tags">
            {techStack.tags!.map((tag) => (
              <span key={tag} className="tech-stack-tag">{tag}</span>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
