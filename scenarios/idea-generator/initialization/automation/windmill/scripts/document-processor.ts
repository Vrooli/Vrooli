// Document Processor - Handles file uploads and document processing

export async function uploadDocument(files: File[], campaignId: string, state: any): Promise<any> {
    try {
        const results = [];
        
        for (const file of files) {
            // First, upload to MinIO
            const uploadResult = await uploadToMinIO(file, campaignId);
            
            if (uploadResult.success) {
                // Then trigger processing pipeline
                const processingResult = await triggerProcessing({
                    campaign_id: campaignId,
                    filename: uploadResult.filename,
                    original_name: file.name,
                    file_size: file.size,
                    storage_path: uploadResult.storage_path
                });
                
                results.push({
                    filename: file.name,
                    status: processingResult.status,
                    document_id: processingResult.document_id
                });
                
                // Show processing notification
                showNotification(`Processing ${file.name}...`, 'info');
            } else {
                results.push({
                    filename: file.name,
                    status: 'failed',
                    error: uploadResult.error
                });
            }
        }
        
        // Update UI to show document processing status
        state.processing_documents = results;
        
        return {
            success: true,
            results: results
        };
    } catch (error) {
        console.error('Error uploading documents:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

async function uploadToMinIO(file: File, campaignId: string): Promise<any> {
    try {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('bucket', 'idea-documents');
        formData.append('path', `campaigns/${campaignId}/uploads/`);
        
        const response = await fetch('http://localhost:9000/upload', {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            throw new Error(`Upload failed: ${response.statusText}`);
        }
        
        const result = await response.json();
        
        return {
            success: true,
            filename: result.filename,
            storage_path: result.path
        };
    } catch (error) {
        console.error('MinIO upload error:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

async function triggerProcessing(documentData: any): Promise<any> {
    try {
        const response = await fetch('http://localhost:5678/webhook/process-document', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(documentData)
        });
        
        if (!response.ok) {
            throw new Error(`Processing trigger failed: ${response.statusText}`);
        }
        
        const result = await response.json();
        
        return {
            status: result.status,
            document_id: result.document_id,
            processing_time: result.processing_time
        };
    } catch (error) {
        console.error('Processing trigger error:', error);
        return {
            status: 'failed',
            error: error.message
        };
    }
}

export async function checkProcessingStatus(documentId: string): Promise<any> {
    try {
        const response = await fetch(`http://localhost:5433/api/documents/${documentId}/status`);
        const document = await response.json();
        
        return {
            status: document.processing_status,
            processed_at: document.processed_at,
            summary: document.metadata?.summary,
            keywords: document.metadata?.keywords
        };
    } catch (error) {
        console.error('Error checking processing status:', error);
        return {
            status: 'error',
            error: error.message
        };
    }
}

export async function getDocumentContent(documentId: string): Promise<any> {
    try {
        const response = await fetch(`http://localhost:5433/api/documents/${documentId}`);
        const document = await response.json();
        
        return {
            success: true,
            content: document.extracted_text,
            metadata: document.metadata
        };
    } catch (error) {
        console.error('Error fetching document content:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

export async function deleteDocument(documentId: string): Promise<any> {
    try {
        // Delete from database
        const dbResponse = await fetch(`http://localhost:5433/api/documents/${documentId}`, {
            method: 'DELETE'
        });
        
        if (!dbResponse.ok) {
            throw new Error('Failed to delete document from database');
        }
        
        // Delete from MinIO
        const document = await dbResponse.json();
        if (document.storage_path) {
            await deleteFromMinIO(document.storage_path);
        }
        
        // Delete from Qdrant
        await deleteFromQdrant(documentId, 'documents');
        
        return {
            success: true,
            message: 'Document deleted successfully'
        };
    } catch (error) {
        console.error('Error deleting document:', error);
        return {
            success: false,
            error: error.message
        };
    }
}

async function deleteFromMinIO(storagePath: string): Promise<void> {
    try {
        await fetch(`http://localhost:9000/delete`, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ path: storagePath })
        });
    } catch (error) {
        console.error('MinIO deletion error:', error);
    }
}

async function deleteFromQdrant(id: string, collection: string): Promise<void> {
    try {
        await fetch(`http://localhost:6333/collections/${collection}/points/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ points: [id] })
        });
    } catch (error) {
        console.error('Qdrant deletion error:', error);
    }
}

function showNotification(message: string, type: 'info' | 'success' | 'error'): void {
    // Implementation depends on Windmill's notification system
    console.log(`[${type.toUpperCase()}] ${message}`);
}