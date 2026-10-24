export type FileTreeNodeKind = 'folder' | 'workflow_file' | 'asset_file';

export interface FileTreeNode {
  kind: FileTreeNodeKind;
  path: string;
  name: string;
  children?: FileTreeNode[];
  workflowId?: string;
  metadata?: Record<string, unknown>;
}

export interface FileTreeDragPayload {
  path: string;
  kind: FileTreeNodeKind;
}
