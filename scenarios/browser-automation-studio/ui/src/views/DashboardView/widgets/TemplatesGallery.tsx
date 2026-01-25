import React, { useState, useCallback } from 'react';
import { useAIEntitlement } from '@hooks/useEntitlement';
import { TemplateModal } from './TemplateModal';
import { TemplateUpgradeModal } from './TemplateUpgradeModal';
import { templates, type Template } from './templates';

interface TemplatesGalleryProps {
  /** Optional callback to open settings (for upgrade modal) */
  onOpenSettings?: (tab?: string) => void;
}


const categoryColors = {
  testing: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  scraping: 'bg-green-500/20 text-green-400 border-green-500/30',
  forms: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  monitoring: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
};

const categoryLabels = {
  testing: 'Testing',
  scraping: 'Scraping',
  forms: 'Forms',
  monitoring: 'Monitoring',
};

export const TemplatesGallery: React.FC<TemplatesGalleryProps> = ({ onOpenSettings }) => {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);

  const { canUseAI } = useAIEntitlement();

  const filteredTemplates = selectedCategory
    ? templates.filter((t) => t.category === selectedCategory)
    : templates;

  const categories = ['testing', 'scraping', 'forms', 'monitoring'] as const;

  const handleTemplateClick = useCallback((template: Template) => {
    if (canUseAI) {
      setSelectedTemplate(template);
      setShowTemplateModal(true);
    } else {
      setShowUpgradeModal(true);
    }
  }, [canUseAI]);

  const handleCloseTemplateModal = useCallback(() => {
    setShowTemplateModal(false);
    setSelectedTemplate(null);
  }, []);

  const handleCloseUpgradeModal = useCallback(() => {
    setShowUpgradeModal(false);
  }, []);

  return (
    <>
      <div className="bg-flow-surface border border-flow-border rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-medium text-flow-text-secondary">Templates</h3>
        </div>

        {/* Category filters */}
        <div className="flex flex-wrap gap-2 mb-4">
          <button
            onClick={() => setSelectedCategory(null)}
            className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
              selectedCategory === null
                ? 'bg-flow-accent text-white border-flow-accent'
                : 'text-flow-text-secondary border-flow-border hover:border-flow-border-light hover:text-surface'
            }`}
          >
            All
          </button>
          {categories.map((cat) => (
            <button
              key={cat}
              onClick={() => setSelectedCategory(cat === selectedCategory ? null : cat)}
              className={`px-2.5 py-1 text-xs rounded-full border transition-colors ${
                selectedCategory === cat
                  ? categoryColors[cat]
                  : 'text-flow-text-secondary border-flow-border hover:border-flow-border-light hover:text-surface'
              }`}
            >
              {categoryLabels[cat]}
            </button>
          ))}
        </div>

        {/* Templates grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          {filteredTemplates.map((template) => (
            <div
              key={template.id}
              onClick={() => handleTemplateClick(template)}
              className="group flex items-start gap-3 p-3 bg-flow-node/50 hover:bg-flow-node-hover border border-flow-border/50 hover:border-flow-border rounded-lg cursor-pointer transition-all"
            >
              <div className={`flex items-center justify-center w-9 h-9 rounded-lg ${categoryColors[template.category].split(' ')[0]} ${categoryColors[template.category].split(' ')[1]}`}>
                {template.icon}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm text-surface font-medium truncate group-hover:text-blue-300 transition-colors">
                  {template.name}
                </div>
                <div className="text-xs text-flow-text-muted line-clamp-2">
                  {template.description}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Template configuration modal */}
      <TemplateModal
        isOpen={showTemplateModal}
        onClose={handleCloseTemplateModal}
        template={selectedTemplate}
      />

      {/* Upgrade prompt modal */}
      <TemplateUpgradeModal
        isOpen={showUpgradeModal}
        onClose={handleCloseUpgradeModal}
        onOpenSettings={onOpenSettings}
      />
    </>
  );
};
