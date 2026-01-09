import { memo, type FC } from 'react';
import type { NodeProps } from 'reactflow';
import { BlurNode as BlurNodeComponent } from './FocusNode';

const WrappedBlurNode: FC<NodeProps> = (props: NodeProps) => <BlurNodeComponent {...props} />;

export default memo(WrappedBlurNode);
