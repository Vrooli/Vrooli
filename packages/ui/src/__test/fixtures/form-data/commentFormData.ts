/**
 * Comment Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for comment creation, editing,
 * and thread management with React Hook Form integration.
 */

import { useForm, type UseFormReturn, type FieldErrors } from 'react-hook-form';
import { yupResolver } from '@hookform/resolvers/yup';
import * as yup from 'yup';
import { act, waitFor } from '@testing-library/react';
import type { 
    CommentCreateInput,
    CommentUpdateInput,
    Comment,
    CommentFor
} from '@vrooli/shared';
import { 
    commentValidation,
    CommentFor as CommentForEnum
} from '@vrooli/shared';

/**
 * UI-specific form data for comment creation
 */
export interface CommentFormData {
    // Content
    content: string;
    
    // Context (what the comment is on)
    commentFor: CommentFor;
    forConnect: string;
    
    // Thread management (UI-specific)
    isReply?: boolean;
    parentId?: string;
    
    // Formatting options
    useMarkdown?: boolean;
    
    // Mentions and tags
    mentionedUsers?: string[];
    tags?: string[];
    
    // Attachments (UI-specific)
    attachments?: Array<{
        name: string;
        type: 'image' | 'file' | 'link';
        url: string;
        description?: string;
    }>;
    
    // Privacy
    isPrivate?: boolean;
}

/**
 * Extended form state with comment-specific properties
 */
export interface CommentFormState {
    values: CommentFormData;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Comment-specific state
    commentState?: {
        currentComment?: Partial<Comment>;
        threadDepth?: number;
        isEditing?: boolean;
        originalContent?: string;
        characterCount?: number;
        maxLength?: number;
    };
    
    // Thread state
    threadState?: {
        parentComment?: Partial<Comment>;
        replies?: Array<Partial<Comment>>;
        totalReplies?: number;
        isCollapsed?: boolean;
    };
    
    // Preview state
    previewState?: {
        isPreviewMode?: boolean;
        renderedContent?: string;
        isRendering?: boolean;
    };
}

/**
 * Comment form data factory with React Hook Form integration
 */
export class CommentFormDataFactory {
    /**
     * Create validation schema for comments
     */
    private createCommentSchema(): yup.ObjectSchema<CommentFormData> {
        return yup.object({
            content: yup
                .string()
                .required('Comment content is required')
                .min(1, 'Comment cannot be empty')
                .max(10000, 'Comment is too long'),
                
            commentFor: yup
                .mixed<CommentFor>()
                .oneOf(Object.values(CommentForEnum))
                .required('Comment target type is required'),
                
            forConnect: yup
                .string()
                .required('Comment target is required')
                .min(1, 'Target ID cannot be empty'),
                
            isReply: yup
                .boolean()
                .optional(),
                
            parentId: yup
                .string()
                .when('isReply', {
                    is: true,
                    then: (schema) => schema.required('Parent comment ID is required for replies'),
                    otherwise: (schema) => schema.optional()
                }),
                
            useMarkdown: yup
                .boolean()
                .optional(),
                
            mentionedUsers: yup
                .array(yup.string())
                .max(20, 'Maximum 20 user mentions allowed')
                .optional(),
                
            tags: yup
                .array(
                    yup.string()
                        .min(2, 'Tag must be at least 2 characters')
                        .max(30, 'Tag is too long')
                )
                .max(10, 'Maximum 10 tags allowed')
                .optional(),
                
            attachments: yup.array(
                yup.object({
                    name: yup.string().required('Attachment name is required'),
                    type: yup.string().oneOf(['image', 'file', 'link']).required(),
                    url: yup.string().url('Invalid URL').required('Attachment URL is required'),
                    description: yup.string().max(200, 'Description too long').optional()
                })
            ).max(5, 'Maximum 5 attachments per comment').optional(),
            
            isPrivate: yup
                .boolean()
                .optional()
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `comment_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create comment form data for different scenarios
     */
    createFormData(
        scenario: 'empty' | 'minimal' | 'complete' | 'invalid' | 'reply' | 'withAttachments' | 
                 'withMentions' | 'longComment' | 'privateComment' | 'markdownComment'
    ): CommentFormData {
        switch (scenario) {
            case 'empty':
                return {
                    content: '',
                    commentFor: CommentForEnum.Api,
                    forConnect: ''
                };
                
            case 'minimal':
                return {
                    content: 'Great work!',
                    commentFor: CommentForEnum.Api,
                    forConnect: 'api_123'
                };
                
            case 'complete':
                return {
                    content: 'This is an excellent implementation! I particularly like how you handled the error cases. @alice might want to review the security implications.',
                    commentFor: CommentForEnum.Api,
                    forConnect: 'api_123',
                    useMarkdown: true,
                    mentionedUsers: ['alice'],
                    tags: ['review', 'security', 'implementation'],
                    attachments: [
                        {
                            name: 'Test Results',
                            type: 'file',
                            url: 'https://example.com/test-results.pdf',
                            description: 'Security test results'
                        }
                    ],
                    isPrivate: false
                };
                
            case 'invalid':
                return {
                    content: '', // Empty content
                    commentFor: '' as any, // Invalid type
                    forConnect: '', // Empty target
                    tags: ['a', 'this-tag-is-way-too-long-and-exceeds-maximum-length'], // Invalid tags
                    attachments: [
                        {
                            name: '', // Empty name
                            type: 'file',
                            url: 'not-a-url' // Invalid URL
                        }
                    ]
                };
                
            case 'reply':
                return {
                    content: 'I agree with your assessment. Thanks for the detailed feedback!',
                    commentFor: CommentForEnum.Api,
                    forConnect: 'api_123',
                    isReply: true,
                    parentId: 'comment_parent_123'
                };
                
            case 'withAttachments':
                return {
                    content: 'Here are some additional resources that might be helpful:',
                    commentFor: CommentForEnum.Routine,
                    forConnect: 'routine_456',
                    attachments: [
                        {
                            name: 'Documentation',
                            type: 'link',
                            url: 'https://docs.example.com',
                            description: 'Official documentation'
                        },
                        {
                            name: 'Screenshot',
                            type: 'image',
                            url: 'https://example.com/screenshot.png',
                            description: 'UI example'
                        },
                        {
                            name: 'Code Sample',
                            type: 'file',
                            url: 'https://github.com/example/sample.js',
                            description: 'Working example'
                        }
                    ]
                };
                
            case 'withMentions':
                return {
                    content: 'Great collaboration between @alice, @bob, and @charlie on this feature. @team-leads please review when you have a chance.',
                    commentFor: CommentForEnum.Project,
                    forConnect: 'project_789',
                    mentionedUsers: ['alice', 'bob', 'charlie', 'team-leads'],
                    tags: ['collaboration', 'feature', 'review']
                };
                
            case 'longComment':
                return {
                    content: `# Detailed Analysis

This is a comprehensive review of the implementation. Here are my findings:

## Positive Aspects
- Clean code structure
- Good error handling
- Comprehensive test coverage

## Areas for Improvement
- Performance could be optimized in the data processing loop
- Consider adding more detailed logging
- Documentation could be expanded

## Recommendations
1. Implement caching for frequently accessed data
2. Add performance benchmarks
3. Create more detailed API documentation

Overall, this is solid work that demonstrates good understanding of the requirements.`,
                    commentFor: CommentForEnum.Api,
                    forConnect: 'api_123',
                    useMarkdown: true,
                    tags: ['review', 'analysis', 'feedback']
                };
                
            case 'privateComment':
                return {
                    content: 'This contains sensitive information that should only be visible to the team.',
                    commentFor: CommentForEnum.Team,
                    forConnect: 'team_999',
                    isPrivate: true,
                    tags: ['internal', 'confidential']
                };
                
            case 'markdownComment':
                return {
                    content: `Here's a **formatted comment** with:

- *Italic text*
- **Bold text**  
- \`code snippets\`
- [Links](https://example.com)

\`\`\`javascript
function example() {
    return "Hello, World!";
}
\`\`\`

> This is a blockquote with important information.`,
                    commentFor: CommentForEnum.Routine,
                    forConnect: 'routine_456',
                    useMarkdown: true,
                    tags: ['documentation', 'example']
                };
                
            default:
                throw new Error(`Unknown comment scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState(
        scenario: 'pristine' | 'dirty' | 'submitting' | 'withErrors' | 'valid' | 'editing' | 'previewing' | 'nearLimit'
    ): CommentFormState {
        const baseFormData = this.createFormData('complete');
        
        switch (scenario) {
            case 'pristine':
                return {
                    values: this.createFormData('empty'),
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: false
                };
                
            case 'dirty':
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { content: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    commentState: {
                        characterCount: baseFormData.content.length,
                        maxLength: 10000
                    }
                };
                
            case 'submitting':
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true
                    }), {}),
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true
                };
                
            case 'withErrors':
                return {
                    values: this.createFormData('invalid'),
                    errors: {
                        content: 'Comment content is required',
                        commentFor: 'Comment target type is required',
                        forConnect: 'Comment target is required',
                        'attachments[0].name': 'Attachment name is required',
                        'attachments[0].url': 'Invalid URL'
                    },
                    touched: { 
                        content: true, 
                        commentFor: true, 
                        forConnect: true,
                        attachments: true
                    },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false
                };
                
            case 'valid':
                return {
                    values: baseFormData,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true
                };
                
            case 'editing':
                return {
                    values: baseFormData,
                    errors: {},
                    touched: { content: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    commentState: {
                        isEditing: true,
                        originalContent: 'Original comment content before editing',
                        characterCount: baseFormData.content.length,
                        maxLength: 10000
                    }
                };
                
            case 'previewing':
                return {
                    values: this.createFormData('markdownComment'),
                    errors: {},
                    touched: { content: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    previewState: {
                        isPreviewMode: true,
                        renderedContent: '<h1>Rendered HTML content...</h1>',
                        isRendering: false
                    }
                };
                
            case 'nearLimit':
                const longContent = 'A'.repeat(9800); // Near 10000 char limit
                return {
                    values: {
                        ...baseFormData,
                        content: longContent
                    },
                    errors: {},
                    touched: { content: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    commentState: {
                        characterCount: longContent.length,
                        maxLength: 10000
                    }
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance
     */
    createFormInstance(
        initialData?: Partial<CommentFormData>
    ): UseFormReturn<CommentFormData> {
        const defaultValues: CommentFormData = {
            content: '',
            commentFor: CommentForEnum.Api,
            forConnect: '',
            tags: [],
            mentionedUsers: [],
            attachments: [],
            ...initialData
        };
        
        return useForm<CommentFormData>({
            mode: 'onChange',
            reValidateMode: 'onChange',
            shouldFocusError: true,
            defaultValues,
            resolver: yupResolver(this.createCommentSchema())
        });
    }
    
    /**
     * Validate form data using real validation
     */
    async validateFormData(
        formData: CommentFormData
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: CommentCreateInput;
    }> {
        try {
            // Validate with form schema
            await this.createCommentSchema().validate(formData, { abortEarly: false });
            
            // Transform to API input
            const apiInput = this.transformToAPIInput(formData);
            
            // Validate with real API validation
            await commentValidation.create.validate(apiInput);
            
            return {
                isValid: true,
                apiInput
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        errors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                errors.general = error.message;
            }
            
            return {
                isValid: false,
                errors
            };
        }
    }
    
    /**
     * Transform form data to API input
     */
    private transformToAPIInput(formData: CommentFormData): CommentCreateInput {
        const input: CommentCreateInput = {
            id: this.generateId(),
            commentFor: formData.commentFor,
            forConnect: formData.forConnect,
            translationsCreate: [{
                id: this.generateId(),
                language: 'en',
                text: formData.content
            }]
        };
        
        // Add parent if it's a reply
        if (formData.isReply && formData.parentId) {
            input.parentConnect = formData.parentId;
        }
        
        return input;
    }
    
    /**
     * Extract mentions from comment content
     */
    extractMentions(content: string): string[] {
        const mentionRegex = /@([a-zA-Z0-9_-]+)/g;
        const mentions: string[] = [];
        let match;
        
        while ((match = mentionRegex.exec(content)) !== null) {
            mentions.push(match[1]);
        }
        
        return [...new Set(mentions)]; // Remove duplicates
    }
    
    /**
     * Calculate reading time estimate
     */
    calculateReadingTime(content: string): number {
        const wordsPerMinute = 200;
        const wordCount = content.split(/\s+/).length;
        return Math.ceil(wordCount / wordsPerMinute);
    }
}

/**
 * Form interaction simulator for comment forms
 */
export class CommentFormInteractionSimulator {
    private interactionDelay: number = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate comment typing with realistic pauses
     */
    async simulateCommentTyping(
        formInstance: UseFormReturn<CommentFormData>,
        content: string,
        options?: { withPauses?: boolean; typingSpeed?: number }
    ): Promise<void> {
        const { withPauses = true, typingSpeed = 75 } = options || {};
        
        // Clear field first
        await this.fillField(formInstance, 'content', '');
        
        for (let i = 1; i <= content.length; i++) {
            const partialContent = content.substring(0, i);
            await this.fillField(formInstance, 'content', partialContent);
            
            // Add realistic pauses at sentence endings and paragraphs
            if (withPauses) {
                if (content.charAt(i - 1) === '\n') {
                    await new Promise(resolve => setTimeout(resolve, typingSpeed * 5));
                } else if (/[.!?]/.test(content.charAt(i - 1))) {
                    await new Promise(resolve => setTimeout(resolve, typingSpeed * 3));
                } else if (/[,;:]/.test(content.charAt(i - 1))) {
                    await new Promise(resolve => setTimeout(resolve, typingSpeed * 2));
                } else {
                    await new Promise(resolve => setTimeout(resolve, typingSpeed));
                }
            } else {
                await new Promise(resolve => setTimeout(resolve, typingSpeed));
            }
        }
    }
    
    /**
     * Simulate adding attachments
     */
    async simulateAddingAttachments(
        formInstance: UseFormReturn<CommentFormData>,
        attachments: CommentFormData['attachments']
    ): Promise<void> {
        if (!attachments) return;
        
        for (let i = 0; i < attachments.length; i++) {
            const attachment = attachments[i];
            
            await this.fillField(formInstance, `attachments.${i}.name` as any, attachment.name);
            await this.fillField(formInstance, `attachments.${i}.type` as any, attachment.type);
            await this.fillField(formInstance, `attachments.${i}.url` as any, attachment.url);
            
            if (attachment.description) {
                await this.fillField(formInstance, `attachments.${i}.description` as any, attachment.description);
            }
            
            await new Promise(resolve => setTimeout(resolve, 300));
        }
    }
    
    /**
     * Simulate markdown preview toggle
     */
    async simulateMarkdownPreview(
        formInstance: UseFormReturn<CommentFormData>
    ): Promise<void> {
        // Enable markdown mode
        await this.fillField(formInstance, 'useMarkdown', true);
        await new Promise(resolve => setTimeout(resolve, 200));
        
        // Simulate preview rendering delay
        await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    /**
     * Simulate reply workflow
     */
    async simulateReplyWorkflow(
        formInstance: UseFormReturn<CommentFormData>,
        parentCommentId: string,
        replyContent: string
    ): Promise<void> {
        // Set up reply context
        await this.fillField(formInstance, 'isReply', true);
        await this.fillField(formInstance, 'parentId', parentCommentId);
        
        // Type reply content
        await this.simulateCommentTyping(formInstance, replyContent, { typingSpeed: 50 });
    }
    
    /**
     * Fill field helper
     */
    private async fillField(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        value: any
    ): Promise<void> {
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch: true,
                shouldValidate: true
            });
        });
        
        await waitFor(() => {
            expect(formInstance.formState.isValidating).toBe(false);
        });
    }
}

// Export factory instances
export const commentFormFactory = new CommentFormDataFactory();
export const commentFormSimulator = new CommentFormInteractionSimulator();

// Export pre-configured scenarios
export const commentFormScenarios = {
    // Basic scenarios
    emptyComment: () => commentFormFactory.createFormState('pristine'),
    validComment: () => commentFormFactory.createFormState('valid'),
    commentWithErrors: () => commentFormFactory.createFormState('withErrors'),
    submittingComment: () => commentFormFactory.createFormState('submitting'),
    
    // Comment types
    minimalComment: () => commentFormFactory.createFormData('minimal'),
    completeComment: () => commentFormFactory.createFormData('complete'),
    replyComment: () => commentFormFactory.createFormData('reply'),
    markdownComment: () => commentFormFactory.createFormData('markdownComment'),
    commentWithAttachments: () => commentFormFactory.createFormData('withAttachments'),
    commentWithMentions: () => commentFormFactory.createFormData('withMentions'),
    longComment: () => commentFormFactory.createFormData('longComment'),
    privateComment: () => commentFormFactory.createFormData('privateComment'),
    
    // Form states
    editingComment: () => commentFormFactory.createFormState('editing'),
    previewingComment: () => commentFormFactory.createFormState('previewing'),
    nearCharacterLimit: () => commentFormFactory.createFormState('nearLimit'),
    
    // Interactive workflows
    async quickCommentWorkflow(formInstance: UseFormReturn<CommentFormData>) {
        const simulator = new CommentFormInteractionSimulator();
        const content = 'Great work on this implementation!';
        await simulator.simulateCommentTyping(formInstance, content, { typingSpeed: 40 });
        return content;
    },
    
    async detailedReviewWorkflow(formInstance: UseFormReturn<CommentFormData>) {
        const simulator = new CommentFormInteractionSimulator();
        const content = `# Code Review

This looks excellent! A few minor suggestions:

1. Consider adding more error handling
2. The performance looks good
3. Documentation is clear

@alice you might want to take a look at the security aspects.`;
        
        await simulator.simulateCommentTyping(formInstance, content, { typingSpeed: 60 });
        await simulator.simulateMarkdownPreview(formInstance);
        return content;
    },
    
    async replyWorkflow(formInstance: UseFormReturn<CommentFormData>) {
        const simulator = new CommentFormInteractionSimulator();
        await simulator.simulateReplyWorkflow(
            formInstance,
            'parent_comment_123',
            'Thanks for the feedback! I\'ll address those points in the next iteration.'
        );
    },
    
    // Utility functions
    extractMentions(content: string) {
        return commentFormFactory.extractMentions(content);
    },
    
    calculateReadingTime(content: string) {
        return commentFormFactory.calculateReadingTime(content);
    }
};