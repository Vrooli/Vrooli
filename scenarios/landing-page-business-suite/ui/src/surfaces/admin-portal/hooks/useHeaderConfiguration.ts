import { useState, useCallback, useMemo } from 'react';
import type {
  ContentSection,
  LandingHeaderConfig,
  HeaderCTAMode,
} from '../../../shared/api';
import { cloneHeaderConfig } from '../../../shared/lib/headerConfig';
import {
  createNavLinkFromSection,
  createDownloadsNavLink,
  createMenuNavLink,
  createMenuChildLink,
  findSectionByTarget,
  parseNavTarget,
} from '../services/variant.service';

interface UseHeaderConfigurationProps {
  config: LandingHeaderConfig;
  sections: ContentSection[];
  onChange: React.Dispatch<React.SetStateAction<LandingHeaderConfig>>;
}

interface UseHeaderConfigurationReturn {
  navTarget: string;
  setNavTarget: (value: string) => void;
  downloadsSection: boolean;
  updateConfig: (updater: (draft: LandingHeaderConfig) => void) => void;
  handleAddLink: () => void;
  handleNavLabelChange: (index: number, value: string) => void;
  handleMenuChildChange: (
    linkIndex: number,
    childIndex: number,
    field: 'label' | 'href',
    value: string
  ) => void;
  handleAddMenuChild: (linkIndex: number) => void;
  handleRemoveMenuChild: (linkIndex: number, childIndex: number) => void;
  handleVisibilityToggle: (index: number, key: 'desktop' | 'mobile', value: boolean) => void;
  handleRemoveLink: (index: number) => void;
  handleAddMenu: () => void;
  handleMoveLink: (index: number, direction: -1 | 1) => void;
  handleCTAModeChange: (
    target: 'primary' | 'secondary',
    updates: { mode?: HeaderCTAMode; label?: string; href?: string; variant?: 'solid' | 'ghost' }
  ) => void;
}

/**
 * Hook for managing header configuration state and operations.
 * Extracts state management logic from HeaderConfigurator component.
 */
export function useHeaderConfiguration({
  config,
  sections,
  onChange,
}: UseHeaderConfigurationProps): UseHeaderConfigurationReturn {
  const [navTarget, setNavTarget] = useState('');

  const downloadsSection = useMemo(
    () => sections.some((section) => section.section_type === 'downloads'),
    [sections]
  );

  const updateConfig = useCallback(
    (updater: (draft: LandingHeaderConfig) => void) => {
      onChange((prev) => {
        const next = cloneHeaderConfig(prev);
        updater(next);
        return next;
      });
    },
    [onChange]
  );

  const handleAddLink = useCallback(() => {
    if (!navTarget) return;

    const parsed = parseNavTarget(navTarget);
    if (!parsed) return;

    if (parsed.type === 'downloads') {
      updateConfig((draft) => {
        draft.nav.links.push(createDownloadsNavLink());
      });
    } else if (parsed.type === 'section') {
      const targetSection = findSectionByTarget(sections, parsed);
      if (targetSection) {
        updateConfig((draft) => {
          draft.nav.links.push(createNavLinkFromSection(targetSection));
        });
      }
    }
    setNavTarget('');
  }, [navTarget, sections, updateConfig]);

  const handleNavLabelChange = useCallback(
    (index: number, value: string) => {
      updateConfig((draft) => {
        const link = draft.nav.links[index];
        if (!link) return;
        link.label = value;
      });
    },
    [updateConfig]
  );

  const handleMenuChildChange = useCallback(
    (
      linkIndex: number,
      childIndex: number,
      field: 'label' | 'href',
      value: string
    ) => {
      updateConfig((draft) => {
        const link = draft.nav.links[linkIndex];
        if (!link || link.type !== 'menu') return;
        if (!Array.isArray(link.children)) {
          link.children = [];
        }
        if (!link.children[childIndex]) {
          link.children[childIndex] = createMenuChildLink();
        }
        if (field === 'label') {
          link.children[childIndex].label = value;
        } else {
          link.children[childIndex].href = value;
        }
      });
    },
    [updateConfig]
  );

  const handleAddMenuChild = useCallback(
    (linkIndex: number) => {
      updateConfig((draft) => {
        const link = draft.nav.links[linkIndex];
        if (!link || link.type !== 'menu') return;
        if (!Array.isArray(link.children)) {
          link.children = [];
        }
        link.children.push(createMenuChildLink());
      });
    },
    [updateConfig]
  );

  const handleRemoveMenuChild = useCallback(
    (linkIndex: number, childIndex: number) => {
      updateConfig((draft) => {
        const link = draft.nav.links[linkIndex];
        if (!link || link.type !== 'menu' || !Array.isArray(link.children)) return;
        link.children.splice(childIndex, 1);
      });
    },
    [updateConfig]
  );

  const handleVisibilityToggle = useCallback(
    (index: number, key: 'desktop' | 'mobile', value: boolean) => {
      updateConfig((draft) => {
        const link = draft.nav.links[index];
        if (!link) return;
        link.visible_on = {
          desktop: key === 'desktop' ? value : link.visible_on?.desktop ?? true,
          mobile: key === 'mobile' ? value : link.visible_on?.mobile ?? true,
        };
      });
    },
    [updateConfig]
  );

  const handleRemoveLink = useCallback(
    (index: number) => {
      updateConfig((draft) => {
        if (index < 0 || index >= draft.nav.links.length) return;
        draft.nav.links.splice(index, 1);
      });
    },
    [updateConfig]
  );

  const handleAddMenu = useCallback(() => {
    updateConfig((draft) => {
      draft.nav.links.push(createMenuNavLink());
    });
  }, [updateConfig]);

  const handleMoveLink = useCallback(
    (index: number, direction: -1 | 1) => {
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= config.nav.links.length) return;
      updateConfig((draft) => {
        const [link] = draft.nav.links.splice(index, 1);
        draft.nav.links.splice(nextIndex, 0, link);
      });
    },
    [config.nav.links.length, updateConfig]
  );

  const handleCTAModeChange = useCallback(
    (
      target: 'primary' | 'secondary',
      updates: { mode?: HeaderCTAMode; label?: string; href?: string; variant?: 'solid' | 'ghost' }
    ) => {
      updateConfig((draft) => {
        draft.ctas[target] = {
          ...draft.ctas[target],
          ...updates,
        };
      });
    },
    [updateConfig]
  );

  return {
    navTarget,
    setNavTarget,
    downloadsSection,
    updateConfig,
    handleAddLink,
    handleNavLabelChange,
    handleMenuChildChange,
    handleAddMenuChild,
    handleRemoveMenuChild,
    handleVisibilityToggle,
    handleRemoveLink,
    handleAddMenu,
    handleMoveLink,
    handleCTAModeChange,
  };
}
