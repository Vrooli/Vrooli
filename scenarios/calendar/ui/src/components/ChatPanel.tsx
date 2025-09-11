import { useState, useRef, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { MessageCircle, Send, X, Sparkles } from 'lucide-react'
import { api } from '@/services/api'
import { useCalendarStore } from '@/stores/calendarStore'
import toast from 'react-hot-toast'
import type { ChatRequest, ChatResponse, SuggestedAction } from '@/types'

export function ChatPanel() {
  const [isOpen, setIsOpen] = useState(false)
  const [message, setMessage] = useState('')
  const [conversation, setConversation] = useState<Array<{
    role: 'user' | 'assistant'
    content: string
    actions?: SuggestedAction[]
  }>>([])
  const [conversationId, setConversationId] = useState<string>()
  
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  
  const { selectEvent } = useCalendarStore()

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [conversation])

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen])

  const chatMutation = useMutation({
    mutationFn: (request: ChatRequest) => api.sendChatMessage(request),
    onSuccess: (response: ChatResponse) => {
      setConversation(prev => [...prev, {
        role: 'assistant',
        content: response.response,
        actions: response.suggestedActions
      }])
      
      if (response.context?.conversation_id) {
        setConversationId(response.context.conversation_id)
      }
    },
    onError: () => {
      toast.error('Failed to send message')
      setConversation(prev => [...prev, {
        role: 'assistant',
        content: "I'm having trouble processing your request. Please try again."
      }])
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!message.trim() || chatMutation.isPending) return

    const userMessage = message.trim()
    setMessage('')
    
    setConversation(prev => [...prev, {
      role: 'user',
      content: userMessage
    }])

    chatMutation.mutate({
      message: userMessage,
      conversationId,
      context: {}
    })
  }

  const handleAction = (action: SuggestedAction) => {
    switch (action.action) {
      case 'create_event':
        selectEvent({
          id: 'new',
          userId: '',
          title: action.parameters?.title || '',
          startTime: action.parameters?.start_time || new Date().toISOString(),
          endTime: action.parameters?.end_time || new Date(Date.now() + 3600000).toISOString(),
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          eventType: 'meeting',
          status: 'scheduled',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        })
        break
      case 'list_events':
        // Switch to agenda view
        break
      default:
        console.log('Unknown action:', action)
    }
  }

  const getExamplePrompts = () => [
    "Schedule a meeting tomorrow at 3pm",
    "What's on my calendar today?",
    "Move my 2pm meeting to 4pm",
    "Find time for a 1-hour meeting this week",
    "Cancel all meetings after 5pm today"
  ]

  return (
    <>
      {/* Chat Toggle Button */}
      {!isOpen && (
        <button
          onClick={() => setIsOpen(true)}
          className="fixed bottom-6 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary-600 text-white shadow-lg transition-transform hover:scale-110"
        >
          <MessageCircle className="h-6 w-6" />
        </button>
      )}

      {/* Chat Panel */}
      {isOpen && (
        <div className="fixed bottom-6 right-6 z-40 flex h-[600px] w-[400px] flex-col rounded-lg border border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-800">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <div className="flex items-center space-x-2">
              <Sparkles className="h-5 w-5 text-primary-600" />
              <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                Calendar Assistant
              </h3>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="btn-ghost p-1"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4">
            {conversation.length === 0 ? (
              <div className="space-y-4">
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Hi! I can help you manage your calendar. Try asking me to:
                </p>
                <div className="space-y-2">
                  {getExamplePrompts().map((prompt, i) => (
                    <button
                      key={i}
                      onClick={() => setMessage(prompt)}
                      className="w-full rounded-lg bg-gray-100 px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
                    >
                      "{prompt}"
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {conversation.map((msg, i) => (
                  <div
                    key={i}
                    className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                  >
                    <div
                      className={`max-w-[80%] rounded-lg px-3 py-2 ${
                        msg.role === 'user'
                          ? 'bg-primary-600 text-white'
                          : 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
                      }`}
                    >
                      <p className="text-sm">{msg.content}</p>
                      
                      {msg.actions && msg.actions.length > 0 && (
                        <div className="mt-2 space-y-1">
                          {msg.actions.map((action, j) => (
                            <button
                              key={j}
                              onClick={() => handleAction(action)}
                              className="w-full rounded bg-white/20 px-2 py-1 text-left text-xs hover:bg-white/30"
                            >
                              {action.action.replace('_', ' ')}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
                {chatMutation.isPending && (
                  <div className="flex justify-start">
                    <div className="rounded-lg bg-gray-100 px-3 py-2 dark:bg-gray-700">
                      <div className="flex space-x-1">
                        <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400"></div>
                        <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 delay-100"></div>
                        <div className="h-2 w-2 animate-bounce rounded-full bg-gray-400 delay-200"></div>
                      </div>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>
            )}
          </div>

          {/* Input */}
          <form onSubmit={handleSubmit} className="border-t border-gray-200 p-4 dark:border-gray-700">
            <div className="flex space-x-2">
              <input
                ref={inputRef}
                type="text"
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="Ask about your calendar..."
                className="input flex-1"
                disabled={chatMutation.isPending}
              />
              <button
                type="submit"
                disabled={!message.trim() || chatMutation.isPending}
                className="btn-primary p-2"
              >
                <Send className="h-4 w-4" />
              </button>
            </div>
          </form>
        </div>
      )}
    </>
  )
}