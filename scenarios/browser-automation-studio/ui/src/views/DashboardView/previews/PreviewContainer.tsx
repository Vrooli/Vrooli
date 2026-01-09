import React from 'react';

interface PreviewContainerProps {
  children: React.ReactNode;
  headerText: string;
  footerContent?: React.ReactNode;
}

export const PreviewContainer: React.FC<PreviewContainerProps> = ({
  children,
  headerText,
  footerContent,
}) => (
  <div className="relative bg-flow-node/80 backdrop-blur-sm border border-flow-border/50 rounded-xl p-6 shadow-2xl">
    <div className="flex items-center gap-2 mb-4 pb-3 border-b border-flow-border/50">
      <div className="flex gap-1.5">
        <div className="w-3 h-3 rounded-full bg-red-500/60" />
        <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
        <div className="w-3 h-3 rounded-full bg-green-500/60" />
      </div>
      <span className="text-xs text-flow-text-muted ml-2">{headerText}</span>
    </div>

    <div className="min-h-[140px] flex flex-col justify-center">
      {children}
    </div>

    {footerContent && (
      <div className="flex items-center justify-center gap-2 mt-4 pt-3 border-t border-flow-border/50">
        {footerContent}
      </div>
    )}
  </div>
);

export default PreviewContainer;
