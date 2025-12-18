import React, { useEffect, useRef, useState } from 'react';
import { ArrowRight, CheckCircle, Loader2, Sparkles } from 'lucide-react';
import PreviewContainer from './PreviewContainer';

type TimeoutRef = ReturnType<typeof setTimeout> | null;

const AI_PROMPT = "Log into Amazon and add the first search result for 'wireless mouse' to cart";

export const AIPreview: React.FC<{ isActive: boolean }> = ({ isActive }) => {
  const [displayedText, setDisplayedText] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [generatedNodes, setGeneratedNodes] = useState<string[]>([]);
  const timeoutRef = useRef<TimeoutRef>(null);

  useEffect(() => {
    if (!isActive) {
      setDisplayedText('');
      setIsGenerating(false);
      setGeneratedNodes([]);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      return;
    }

    let charIndex = 0;
    const typeNextChar = () => {
      if (charIndex < AI_PROMPT.length) {
        setDisplayedText(AI_PROMPT.slice(0, charIndex + 1));
        charIndex++;
        timeoutRef.current = setTimeout(typeNextChar, 35);
      } else {
        timeoutRef.current = setTimeout(() => {
          setIsGenerating(true);
          const nodes = ['Navigate', 'Search', 'Click', 'Add to Cart'];
          nodes.forEach((node, i) => {
            timeoutRef.current = setTimeout(() => {
              setGeneratedNodes(prev => [...prev, node]);
              if (i === nodes.length - 1) {
                setTimeout(() => setIsGenerating(false), 300);
              }
            }, 400 * (i + 1));
          });
        }, 500);
      }
    };

    timeoutRef.current = setTimeout(typeNextChar, 300);

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isActive]);

  return (
    <PreviewContainer
      headerText="ai-workflow-generator"
      footerContent={
        isGenerating ? (
          <>
            <Loader2 size={14} className="text-purple-400 animate-spin" />
            <span className="text-xs text-flow-text-muted">Generating workflow...</span>
          </>
        ) : generatedNodes.length > 0 ? (
          <>
            <CheckCircle size={14} className="text-green-400" />
            <span className="text-xs text-flow-text-muted">Workflow generated!</span>
          </>
        ) : (
          <>
            <Sparkles size={14} className="text-purple-400" />
            <span className="text-xs text-flow-text-muted">Describe your automation...</span>
          </>
        )
      }
    >
      <div className="mb-4">
        <div className="flex items-start gap-3 p-3 bg-purple-500/10 border border-purple-500/30 rounded-lg">
          <Sparkles size={18} className="text-purple-400 mt-0.5 flex-shrink-0" />
          <div className="flex-1 min-h-[24px]">
            <span className="text-sm text-purple-200">{displayedText}</span>
            {displayedText.length < AI_PROMPT.length && (
              <span className="inline-block w-0.5 h-4 bg-purple-400 ml-0.5 animate-pulse" />
            )}
          </div>
        </div>
      </div>

      {generatedNodes.length > 0 && (
        <div className="flex items-center justify-center gap-2 flex-wrap">
          {generatedNodes.map((node, index) => (
            <React.Fragment key={node}>
              <div
                className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-500/20 border border-purple-500/40 rounded-md animate-scale-in"
                style={{ animationDelay: `${index * 100}ms` }}
              >
                <span className="text-xs font-medium text-purple-300">{node}</span>
              </div>
              {index < generatedNodes.length - 1 && (
                <ArrowRight size={14} className="text-purple-400/60 animate-fade-in" />
              )}
            </React.Fragment>
          ))}
        </div>
      )}
    </PreviewContainer>
  );
};

export default AIPreview;
