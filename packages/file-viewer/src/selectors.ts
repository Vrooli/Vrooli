/**
 * The viewer's `data-testid`s. They are stable test API: Web Console's browser
 * automation workflows and selector manifest depend on these exact values, so a
 * rename is a breaking change. Hosts spread this tree into their own registry
 * (see README, "Selectors") so their generated manifest carries it.
 */
export const fileViewerSelectors = {
  databasePreview: { root: "file-preview-database", tables: "database-tables", grid: "database-grid", records: "database-records", schema: "database-schema", navigation: "database-navigation" },
  fontPreview: { root: "file-preview-font", specimen: "font-preview-specimen", sample: "font-preview-sample", characters: "font-preview-characters", navigation: "font-preview-navigation" },
  modelPreview: { root: "file-preview-model", stage: "model-preview-stage", navigation: "model-preview-navigation" },
  presentationPreview: {
    root: "file-preview-presentation",
    frame: "presentation-preview-frame",
    notes: "presentation-preview-notes",
    navigation: "presentation-preview-navigation",
  },
  ebookPreview: {
    root: "file-preview-ebook",
    article: "ebook-preview-article",
    pane: "ebook-preview-pane",
    contents: "ebook-preview-contents",
  },
  documentPreview: {
    root: "file-preview-document",
    article: "document-preview-article",
    pane: "document-preview-pane",
    search: "document-preview-search",
    outline: "document-preview-outline",
  },
  inspectionPreview: {
    navigation: 'file-preview-navigation',
    sourceLines: 'inspection-source-lines',
    sourceSearch: 'inspection-source-search',
    sourceSearchToggle: 'inspection-source-search-toggle',
    diffBody: 'inspection-diff-body',
    workbook: 'inspection-workbook',
    workbookGrid: 'inspection-workbook-grid',
    workbookRows: 'inspection-workbook-rows',
    workbookDetail: 'inspection-workbook-detail',
  },
  readerPreview: {
    searchToggle: 'reader-search-toggle',
    outline: 'reader-outline',
    readingPane: 'reader-reading-pane',
    structureTree: 'reader-structure-tree',
    structureDetail: 'reader-structure-detail',
  },
  artifactPreview: {
    imageStage: 'artifact-image-stage',
    pdfPages: 'artifact-pdf-pages',
    videoSeek: 'artifact-video-seek',
    videoPlay: 'artifact-video-play',
    tableGrid: 'artifact-table-grid',
    tableRecords: 'artifact-table-records',
    tableRecord: 'artifact-table-record',
  },
  audioPreview: {
    player: 'audio-preview-player',
    waveform: 'audio-preview-waveform',
    seek: 'audio-preview-seek',
    play: 'audio-preview-play',
    repeat: 'audio-preview-repeat',
    tools: 'audio-preview-tools',
  },
};

/** The registry the viewer's own components read from. */
export const selectors = fileViewerSelectors;
