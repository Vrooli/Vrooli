import { librarySelectors } from "../../consts/selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  admin: {
    login: {
      email: 'admin-login-email',
      passwordField: 'admin-login-password',
      submit: 'admin-login-submit',
      error: 'admin-login-error',
    },
    breadcrumb: 'admin-breadcrumb',
    nav: {
      home: 'nav-home',
      analytics: 'nav-analytics',
      customization: 'nav-customization',
      logout: 'nav-logout',
    },
    home: {
      resumePanel: 'admin-resume-panel',
      resumeCustomization: 'admin-resume-customization',
      resumeAnalytics: 'admin-resume-analytics',
      resumeCard: 'admin-resume-card',
      resumeAnalyticsCard: 'admin-resume-analytics-card',
      experienceGuide: 'admin-experience-guide',
      guideAnalytics: 'admin-guide-analytics',
      guideCustomization: 'admin-guide-customization',
      guidePreview: 'admin-guide-preview',
    },
    mode: {
      analytics: 'admin-mode-analytics',
      customization: 'admin-mode-customization',
    },
    analytics: {
      filters: 'analytics-filters',
      timeRange: 'analytics-time-range',
      variantFilter: 'analytics-variant-filter',
      focusBanner: 'analytics-focus-banner',
      resetFilters: 'analytics-reset-filters',
      focusCustomize: 'analytics-focus-customize',
      focusPreview: 'analytics-focus-preview',
      totalVisitors: 'analytics-total-visitors',
      conversionRate: 'analytics-conversion-rate',
      topCta: 'analytics-top-cta',
      variantPerformance: 'analytics-variant-performance',
      variantDetail: 'analytics-variant-detail',
      variantActions: 'analytics-variant-actions',
    },
    customization: {
      triggerAgent: 'trigger-agent-customization',
      createVariant: 'create-variant',
      addSection: 'add-section',
      filterBar: 'variant-filter-bar',
      filterSearch: 'variant-search-input',
      filterAttentionToggle: 'variant-attention-filter',
      clearFilters: 'clear-variant-filters',
      needsAttentionFocus: 'needs-attention-focus',
      variantListSummary: 'variant-list-summary',
    },
    variant: {
      nameInput: 'variant-name-input',
      slugInput: 'variant-slug-input',
      descriptionInput: 'variant-description-input',
      weightInput: 'variant-weight-input',
      save: 'save-variant',
    },
    section: {
      form: 'section-form',
      preview: 'section-preview',
      typeInput: 'section-type-input',
      enabledInput: 'section-enabled-input',
      orderInput: 'section-order-input',
      save: 'save-section',
      content: {
        titleInput: 'content-title-input',
        subtitleInput: 'content-subtitle-input',
        ctaTextInput: 'content-cta-text-input',
        ctaUrlInput: 'content-cta-url-input',
        imageUrlInput: 'content-image-url-input',
      },
    },
    agent: {
      briefInput: 'agent-brief-input',
      assetsInput: 'agent-assets-input',
      previewInput: 'agent-preview-input',
      submit: 'agent-submit',
    },
  },
  publicLanding: {
    experienceHeader: 'landing-experience-header',
    navCta: 'landing-nav-cta',
    navMobile: 'landing-nav-mobile',
    navDownload: 'landing-nav-download',
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  admin: {
    analytics: {
      variantRow: defineDynamicSelector({
        description: 'Analytics variant performance row by variant ID',
        testIdPattern: 'analytics-variant-row-${id}',
        params: { id: { type: 'number' } },
      }),
      viewDetails: defineDynamicSelector({
        description: 'View details button for specific variant',
        testIdPattern: 'analytics-view-details-${id}',
        params: { id: { type: 'number' } },
      }),
      editVariant: defineDynamicSelector({
        description: 'Customize button for specific variant from analytics table',
        testIdPattern: 'analytics-edit-${id}',
        params: { id: { type: 'number' } },
      }),
    },
    customization: {
      variantCard: defineDynamicSelector({
        description: 'Variant card by slug',
        testIdPattern: 'variant-card-${slug}',
        params: { slug: { type: 'string' } },
      }),
      editVariant: defineDynamicSelector({
        description: 'Edit variant button by slug',
        testIdPattern: 'edit-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      previewVariant: defineDynamicSelector({
        description: 'Preview variant button by slug',
        testIdPattern: 'preview-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      archiveVariant: defineDynamicSelector({
        description: 'Archive variant button by slug',
        testIdPattern: 'archive-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantAnalytics: defineDynamicSelector({
        description: 'Link to view analytics for a variant card',
        testIdPattern: 'variant-analytics-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantPerformance: defineDynamicSelector({
        description: 'Variant performance summary block',
        testIdPattern: 'variant-performance-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantStatus: defineDynamicSelector({
        description: 'Variant status badges block',
        testIdPattern: 'variant-status-${slug}',
        params: { slug: { type: 'string' } },
      }),
      deleteVariant: defineDynamicSelector({
        description: 'Delete variant button by slug',
        testIdPattern: 'delete-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      section: defineDynamicSelector({
        description: 'Section item by ID',
        testIdPattern: 'section-${id}',
        params: { id: { type: 'number' } },
      }),
      editSection: defineDynamicSelector({
        description: 'Edit section button by ID',
        testIdPattern: 'edit-section-${id}',
        params: { id: { type: 'number' } },
      }),
    },
    breadcrumb: defineDynamicSelector({
      description: 'Breadcrumb segment by index',
      testIdPattern: 'breadcrumb-${index}',
      params: { index: { type: 'number' } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
