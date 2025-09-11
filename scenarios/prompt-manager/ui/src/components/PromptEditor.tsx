import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Save, 
  Play, 
  Copy, 
  Heart, 
  Trash2, 
  Maximize2, 
  Eye,
  EyeOff,
  Zap,
  Clock,
  FileText,
  BarChart3
} from 'lucide-react'
import { Editor } from '@monaco-editor/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { LoadingSpinner } from './ui/loading-spinner'
import { useTheme } from '@/hooks/use-theme'
import { cn, formatDate, countWords, estimateTokens } from '@/lib/utils'
import type { Prompt } from '@/types'

interface PromptEditorProps {
  prompt: Prompt | null
  onSave: (prompt: Prompt) => void
  onDelete: (promptId: string) => void
}

export function PromptEditor({ prompt, onSave, onDelete }: PromptEditorProps) {
  const [editedPrompt, setEditedPrompt] = useState<Partial<Prompt>>(prompt || {})
  const [isSaving, setIsSaving] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const [testResult, setTestResult] = useState<string | null>(null)
  const [isPreviewMode, setIsPreviewMode] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  
  const { theme } = useTheme()
  const queryClient = useQueryClient()

  useEffect(() => {
    setEditedPrompt(prompt || {})
    setTestResult(null)
  }, [prompt])

  const updatePromptMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Prompt> }) =>
      api.updatePrompt(id, data),
    onSuccess: (updatedPrompt) => {
      queryClient.invalidateQueries({ queryKey: ['prompts'] })
      onSave(updatedPrompt)
    },
  })

  const deletePromptMutation = useMutation({
    mutationFn: (id: string) => api.deletePrompt(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['prompts'] })
      if (prompt) onDelete(prompt.id)
    },
  })

  const testPromptMutation = useMutation({
    mutationFn: ({ id, request }: { id: string; request: any }) =>
      api.testPrompt(id, request),
    onSuccess: (result) => {
      setTestResult(result.response)
    },
    onError: () => {
      setTestResult('Testing is not available (Ollama may not be configured)')
    },
  })

  const handleSave = async () => {
    if (!editedPrompt.id) return
    
    setIsSaving(true)
    try {
      await updatePromptMutation.mutateAsync({
        id: editedPrompt.id,
        data: {
          title: editedPrompt.title,
          content: editedPrompt.content,
          description: editedPrompt.description,
          is_favorite: editedPrompt.is_favorite,
        }
      })
    } finally {
      setIsSaving(false)
    }
  }

  const handleTest = async () => {
    if (!editedPrompt.id) return
    
    setIsTesting(true)
    try {
      await testPromptMutation.mutateAsync({
        id: editedPrompt.id,
        request: {
          model: 'llama3.2',
          variables: {}
        }
      })
    } finally {
      setIsTesting(false)
    }
  }

  const handleDelete = async () => {
    if (!editedPrompt.id || !confirm('Delete this prompt? This action cannot be undone.')) return
    
    await deletePromptMutation.mutateAsync(editedPrompt.id)
  }

  const copyToClipboard = () => {
    navigator.clipboard.writeText(editedPrompt.content || '')
  }

  if (!prompt) {
    return (
      <Card className="h-full flex items-center justify-center bg-card/50 backdrop-blur-sm border-border/50">
        <motion.div
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          className="text-center p-8"
        >
          <motion.div
            animate={{ 
              rotate: [0, 10, -10, 0],
              scale: [1, 1.1, 1]
            }}
            transition={{ 
              duration: 3,
              repeat: Infinity,
              repeatDelay: 2
            }}
            className="w-16 h-16 mx-auto mb-4 bg-gradient-to-br from-purple-500 to-pink-500 rounded-2xl flex items-center justify-center"
          >
            <FileText className="h-8 w-8 text-white" />
          </motion.div>
          <h3 className="text-lg font-semibold mb-2">Select a prompt to edit</h3>
          <p className="text-sm text-muted-foreground">
            Choose a prompt from the list to start editing, or create a new one
          </p>
        </motion.div>
      </Card>
    )
  }

  const wordCount = countWords(editedPrompt.content || '')
  const tokenCount = estimateTokens(editedPrompt.content || '')
  const hasChanges = JSON.stringify(editedPrompt) !== JSON.stringify(prompt)

  return (
    <Card className={cn(
      "h-full flex flex-col bg-card/50 backdrop-blur-sm border-border/50 transition-all duration-300",
      isFullscreen && "fixed inset-4 z-50 shadow-2xl"
    )}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <motion.div
              animate={{ 
                boxShadow: hasChanges 
                  ? "0 0 0 2px rgba(59, 130, 246, 0.5)"
                  : "0 0 0 0px rgba(59, 130, 246, 0)"
              }}
              className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center"
            >
              <FileText className="h-5 w-5 text-white" />
            </motion.div>
            
            <div>
              <CardTitle className="text-lg">Prompt Editor</CardTitle>
              {hasChanges && (
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="text-xs text-amber-600 dark:text-amber-400"
                >
                  Unsaved changes
                </motion.p>
              )}
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsPreviewMode(!isPreviewMode)}
                className="h-8 w-8"
              >
                {isPreviewMode ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </motion.div>
            
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsFullscreen(!isFullscreen)}
                className="h-8 w-8"
              >
                <Maximize2 className="h-4 w-4" />
              </Button>
            </motion.div>
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="flex-1 flex flex-col space-y-4 overflow-hidden">
        {/* Title and metadata */}
        <div className="space-y-3">
          <Input
            value={editedPrompt.title || ''}
            onChange={(e) => setEditedPrompt({ ...editedPrompt, title: e.target.value })}
            placeholder="Prompt title..."
            className="font-medium"
          />
          
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <div className="flex items-center gap-1">
              <BarChart3 className="h-3 w-3" />
              {wordCount} words
            </div>
            <div className="flex items-center gap-1">
              <Zap className="h-3 w-3" />
              ~{tokenCount} tokens
            </div>
            <div className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              Used {prompt.usage_count || 0} times
            </div>
          </div>
        </div>

        {/* Content editor */}
        <div className="flex-1 flex flex-col min-h-0">
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-sm font-medium">Content</h4>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={copyToClipboard}
                className="h-6 text-xs"
              >
                <Copy className="h-3 w-3 mr-1" />
                Copy
              </Button>
            </div>
          </div>
          
          <motion.div
            className="flex-1 rounded-lg border border-border/50 overflow-hidden"
            layout
          >
            {isPreviewMode ? (
              <div className="h-full p-4 bg-background/50 overflow-y-auto">
                <pre className="whitespace-pre-wrap text-sm font-mono leading-relaxed">
                  {editedPrompt.content || 'No content'}
                </pre>
              </div>
            ) : (
              <Editor
                height="100%"
                language="markdown"
                theme={theme === 'dark' ? 'vs-dark' : 'light'}
                value={editedPrompt.content || ''}
                onChange={(value) => setEditedPrompt({ ...editedPrompt, content: value || '' })}
                options={{
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  fontSize: 14,
                  lineHeight: 22,
                  wordWrap: 'on',
                  padding: { top: 16, bottom: 16 },
                  smoothScrolling: true,
                  cursorBlinking: 'smooth',
                  renderLineHighlight: 'gutter',
                  selectOnLineNumbers: true,
                }}
              />
            )}
          </motion.div>
        </div>

        {/* Description */}
        <div>
          <label className="text-sm font-medium mb-2 block">Description</label>
          <textarea
            value={editedPrompt.description || ''}
            onChange={(e) => setEditedPrompt({ ...editedPrompt, description: e.target.value })}
            placeholder="Optional description for this prompt..."
            className="w-full h-20 p-3 text-sm bg-background border border-border rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          />
        </div>

        {/* Test result */}
        <AnimatePresence>
          {testResult && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              className="p-4 bg-muted/50 rounded-lg border border-border/50"
            >
              <h4 className="text-sm font-medium mb-2">Test Result</h4>
              <pre className="text-xs text-muted-foreground whitespace-pre-wrap">
                {testResult}
              </pre>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Actions */}
        <div className="flex items-center justify-between pt-2 border-t border-border/30">
          <div className="flex items-center gap-2">
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setEditedPrompt({ 
                  ...editedPrompt, 
                  is_favorite: !editedPrompt.is_favorite 
                })}
                className={cn(
                  "h-8",
                  editedPrompt.is_favorite && "text-red-500"
                )}
              >
                <Heart className={cn(
                  "h-4 w-4 mr-1",
                  editedPrompt.is_favorite && "fill-current"
                )} />
                {editedPrompt.is_favorite ? 'Favorited' : 'Favorite'}
              </Button>
            </motion.div>
            
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleTest}
                disabled={isTesting || !editedPrompt.content}
                className="h-8"
              >
                {isTesting ? (
                  <>
                    <LoadingSpinner size="sm" className="mr-1" />
                    Testing...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4 mr-1" />
                    Test
                  </>
                )}
              </Button>
            </motion.div>
          </div>
          
          <div className="flex items-center gap-2">
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDelete}
                disabled={deletePromptMutation.isPending}
                className="h-8 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950"
              >
                <Trash2 className="h-4 w-4 mr-1" />
                Delete
              </Button>
            </motion.div>
            
            <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
              <Button
                onClick={handleSave}
                disabled={!hasChanges || isSaving}
                className="h-8 button-glow"
              >
                {isSaving ? (
                  <>
                    <LoadingSpinner size="sm" className="mr-1" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="h-4 w-4 mr-1" />
                    Save Changes
                  </>
                )}
              </Button>
            </motion.div>
          </div>
        </div>

        {/* Metadata footer */}
        <div className="text-xs text-muted-foreground pt-2 border-t border-border/20 space-y-1">
          <div>Created: {formatDate(prompt.created_at)}</div>
          <div>Last updated: {formatDate(prompt.updated_at)}</div>
        </div>
      </CardContent>
    </Card>
  )
}