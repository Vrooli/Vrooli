-- Smart File Photo Manager - Seed Data
-- Initial data for testing and demonstration

-- Insert sample folders structure
INSERT INTO folders (name, path, parent_path, is_smart, description) VALUES
-- Root level folders
('Documents', '/Documents', '/', false, 'General document storage'),
('Photos', '/Photos', '/', false, 'Photo and image collection'),
('Work', '/Work', '/', false, 'Work-related files and documents'),
('Personal', '/Personal', '/', false, 'Personal files and memories'),
('Archives', '/Archives', '/', false, 'Long-term storage and backups'),

-- Document subfolders
('Reports', '/Documents/Reports', '/Documents', false, 'Business reports and analytics'),
('Invoices', '/Documents/Invoices', '/Documents', false, 'Financial invoices and receipts'),
('Contracts', '/Documents/Contracts', '/Documents', false, 'Legal contracts and agreements'),
('Presentations', '/Documents/Presentations', '/Documents', false, 'Presentation files and slides'),

-- Photo subfolders
('2024', '/Photos/2024', '/Photos', false, '2024 photo collection'),
('Vacation', '/Photos/Vacation', '/Photos', false, 'Holiday and travel photos'),
('Events', '/Photos/Events', '/Photos', false, 'Special events and celebrations'),
('Screenshots', '/Photos/Screenshots', '/Photos', false, 'Screen captures and app screenshots'),

-- Work subfolders
('Projects', '/Work/Projects', '/Work', false, 'Active work projects'),
('Meetings', '/Work/Meetings', '/Work', false, 'Meeting notes and recordings'),
('Resources', '/Work/Resources', '/Work', false, 'Work-related resources and references'),

-- Smart Folders
('Recent Files', '/Smart Folders/Recent Files', '/', true, 'Recently uploaded or modified files'),
('Unorganized', '/Smart Folders/Unorganized', '/', true, 'Files that need organization'),
('Large Files', '/Smart Folders/Large Files', '/', true, 'Files larger than 100MB'),
('Duplicates', '/Smart Folders/Duplicates', '/', true, 'Potential duplicate files'),
('Work Related', '/Smart Folders/Work Related', '/', true, 'AI-detected work files'),
('Personal Photos', '/Smart Folders/Personal Photos', '/', true, 'AI-detected personal photos');

-- Update smart folder queries
UPDATE folders SET smart_query = jsonb_build_object(
    'dateRange', '7d',
    'sortBy', 'uploaded_at',
    'sortOrder', 'desc',
    'limit', 50
) WHERE name = 'Recent Files';

UPDATE folders SET smart_query = jsonb_build_object(
    'folderPath', '/',
    'hasDescription', false,
    'hasTags', false,
    'limit', 100
) WHERE name = 'Unorganized';

UPDATE folders SET smart_query = jsonb_build_object(
    'sizeMin', 104857600,
    'sortBy', 'size_bytes',
    'sortOrder', 'desc',
    'limit', 50
) WHERE name = 'Large Files';

UPDATE folders SET smart_query = jsonb_build_object(
    'hasDuplicates', true,
    'duplicateConfidence', 0.85,
    'limit', 100
) WHERE name = 'Duplicates';

UPDATE folders SET smart_query = jsonb_build_object(
    'or', jsonb_build_array(
        jsonb_build_object('tags', jsonb_build_array('work', 'business', 'office')),
        jsonb_build_object('keywords', jsonb_build_array('meeting', 'report', 'presentation')),
        jsonb_build_object('folderPath', '/Work%')
    ),
    'limit', 200
) WHERE name = 'Work Related';

UPDATE folders SET smart_query = jsonb_build_object(
    'and', jsonb_build_array(
        jsonb_build_object('fileType', 'image'),
        jsonb_build_object('or', jsonb_build_array(
            jsonb_build_object('detectedObjects', jsonb_build_array('person', 'face')),
            jsonb_build_object('tags', jsonb_build_array('personal', 'family', 'friends'))
        ))
    ),
    'limit', 200
) WHERE name = 'Personal Photos';

-- Sample file entries for demonstration
INSERT INTO files (
    original_name, current_name, file_hash, size_bytes, mime_type, file_type, extension,
    storage_path, folder_path, status, processing_stage, description, tags, categories,
    uploaded_by, owner_id
) VALUES
-- Sample documents
('Quarterly_Report_Q4_2024.pdf', 'Quarterly_Report_Q4_2024.pdf', 'abc123def456789', 2458672, 'application/pdf', 'document', '.pdf',
 '/original-files/documents/quarterly-report-q4-2024.pdf', '/Documents/Reports', 'processed', 'organized',
 'Quarterly financial report showing revenue growth and market expansion for Q4 2024',
 ARRAY['quarterly', 'report', 'financial', '2024', 'revenue'], ARRAY['business', 'finance'], 'admin', 'user1'),

('Meeting_Notes_Jan_15.docx', 'Meeting_Notes_Jan_15.docx', 'def789abc123456', 156789, 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'document', '.docx',
 '/original-files/documents/meeting-notes-jan-15.docx', '/Work/Meetings', 'processed', 'organized',
 'Meeting notes from the weekly team sync discussing project milestones and deadlines',
 ARRAY['meeting', 'notes', 'team', 'sync', 'january'], ARRAY['work', 'meetings'], 'admin', 'user1'),

('Invoice_ABC_Corp_2024_001.pdf', 'Invoice_ABC_Corp_2024_001.pdf', 'ghi456jkl789012', 89456, 'application/pdf', 'document', '.pdf',
 '/original-files/documents/invoice-abc-corp-001.pdf', '/Documents/Invoices', 'processed', 'organized',
 'Invoice from ABC Corp for consulting services rendered in January 2024',
 ARRAY['invoice', 'abc-corp', '2024', 'consulting', 'services'], ARRAY['financial', 'business'], 'admin', 'user1'),

-- Sample images
('Vacation_Beach_Sunset.jpg', 'Vacation_Beach_Sunset.jpg', 'xyz987uvw654321', 3456789, 'image/jpeg', 'image', '.jpg',
 '/original-files/images/vacation-beach-sunset.jpg', '/Photos/Vacation', 'processed', 'organized',
 'Beautiful sunset over the ocean with people walking on the beach and palm trees in the background',
 ARRAY['vacation', 'beach', 'sunset', 'ocean', 'travel'], ARRAY['personal', 'photos'], 'admin', 'user1'),

('Team_Meeting_Screenshot.png', 'Team_Meeting_Screenshot.png', 'pqr345stu678901', 856342, 'image/png', 'image', '.png',
 '/original-files/images/team-meeting-screenshot.png', '/Photos/Screenshots', 'processed', 'organized',
 'Screenshot of video conference call with team members discussing project status',
 ARRAY['screenshot', 'meeting', 'team', 'conference', 'work'], ARRAY['work', 'screenshots'], 'admin', 'user1'),

('Family_Birthday_Party.jpg', 'Family_Birthday_Party.jpg', 'lmn678opq234567', 4567890, 'image/jpeg', 'image', '.jpg',
 '/original-files/images/family-birthday-party.jpg', '/Photos/Events', 'processed', 'organized',
 'Family celebration with birthday cake, decorations, and multiple people gathered around a table',
 ARRAY['family', 'birthday', 'party', 'celebration', 'personal'], ARRAY['personal', 'events'], 'admin', 'user1'),

-- Unorganized files (in root directory)
('Untitled_Document.pdf', 'Untitled_Document.pdf', 'rst901vwx345678', 234567, 'application/pdf', 'document', '.pdf',
 '/original-files/documents/untitled-document.pdf', '/', 'pending', 'uploaded',
 NULL, ARRAY[]::text[], ARRAY[]::text[], 'admin', 'user1'),

('IMG_20240115_143022.jpg', 'IMG_20240115_143022.jpg', 'bcd456efg789012', 2345678, 'image/jpeg', 'image', '.jpg',
 '/original-files/images/img-20240115-143022.jpg', '/', 'processing', 'extracted',
 NULL, ARRAY[]::text[], ARRAY[]::text[], 'admin', 'user1');

-- Update file metadata
UPDATE files SET 
    width = 1920, height = 1080, detected_objects = jsonb_build_array('ocean', 'beach', 'person', 'tree'),
    detected_faces = 2, content_embedding = ARRAY[0.1, 0.2, 0.3]::vector,
    visual_embedding = ARRAY[0.4, 0.5, 0.6]::vector, processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Vacation_Beach_Sunset.jpg';

UPDATE files SET 
    width = 1366, height = 768, detected_objects = jsonb_build_array('computer', 'screen', 'person'),
    detected_faces = 4, content_embedding = ARRAY[0.7, 0.8, 0.9]::vector,
    visual_embedding = ARRAY[0.2, 0.3, 0.4]::vector, processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Team_Meeting_Screenshot.png';

UPDATE files SET 
    width = 2048, height = 1536, detected_objects = jsonb_build_array('person', 'cake', 'table', 'decoration'),
    detected_faces = 6, content_embedding = ARRAY[0.5, 0.6, 0.7]::vector,
    visual_embedding = ARRAY[0.8, 0.9, 0.1]::vector, processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Family_Birthday_Party.jpg';

UPDATE files SET 
    page_count = 15, content_embedding = ARRAY[0.3, 0.4, 0.5]::vector,
    ocr_text = 'Quarterly Report Q4 2024\n\nExecutive Summary\nThis quarter showed exceptional growth...',
    processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Quarterly_Report_Q4_2024.pdf';

UPDATE files SET 
    page_count = 3, content_embedding = ARRAY[0.6, 0.7, 0.8]::vector,
    ocr_text = 'Weekly Team Meeting - January 15, 2024\n\nAttendees: John, Sarah, Mike...',
    processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Meeting_Notes_Jan_15.docx';

UPDATE files SET 
    page_count = 2, content_embedding = ARRAY[0.9, 0.1, 0.2]::vector,
    ocr_text = 'INVOICE\n\nABC Corporation\nInvoice #: 2024-001\nDate: January 15, 2024...',
    processed_at = CURRENT_TIMESTAMP
WHERE original_name = 'Invoice_ABC_Corp_2024_001.pdf';

-- Sample content chunks for documents
INSERT INTO content_chunks (file_id, chunk_index, page_number, content, chunk_type, embedding) 
SELECT 
    id, 
    0, 
    1, 
    'Executive Summary: This quarterly report presents our financial performance for Q4 2024, highlighting significant revenue growth of 23% compared to the previous quarter. Our market expansion into European markets has contributed substantially to this growth.',
    'paragraph',
    ARRAY[0.3, 0.4, 0.5]::vector
FROM files WHERE original_name = 'Quarterly_Report_Q4_2024.pdf';

INSERT INTO content_chunks (file_id, chunk_index, page_number, content, chunk_type, embedding)
SELECT 
    id, 
    1, 
    3, 
    'Revenue Analysis: Total revenue for Q4 2024 reached $2.5M, representing a 23% increase from Q3. Key drivers include: New customer acquisition (40% of growth), European market expansion (35% of growth), Product line extensions (25% of growth).',
    'paragraph',
    ARRAY[0.4, 0.5, 0.6]::vector
FROM files WHERE original_name = 'Quarterly_Report_Q4_2024.pdf';

INSERT INTO content_chunks (file_id, chunk_index, page_number, content, chunk_type, embedding)
SELECT 
    id, 
    0, 
    1, 
    'Weekly Team Meeting - January 15, 2024. Attendees: John Smith (PM), Sarah Johnson (Dev Lead), Mike Chen (Designer), Lisa Brown (QA). Agenda: Project status updates, upcoming deadlines, resource allocation.',
    'paragraph',
    ARRAY[0.6, 0.7, 0.8]::vector
FROM files WHERE original_name = 'Meeting_Notes_Jan_15.docx';

-- Sample AI suggestions
INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence)
SELECT 
    id, 
    'folder', 
    'pending', 
    '/Personal/Family Photos', 
    'Image contains multiple people in what appears to be a family celebration setting with birthday cake and decorations',
    0.87
FROM files WHERE original_name = 'Family_Birthday_Party.jpg';

INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence)
SELECT 
    id, 
    'tag', 
    'pending', 
    'financial-report', 
    'Document contains financial data, revenue analysis, and quarterly performance metrics',
    0.92
FROM files WHERE original_name = 'Quarterly_Report_Q4_2024.pdf';

INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence)
SELECT 
    id, 
    'rename', 
    'pending', 
    'Q4_2024_Financial_Report.pdf', 
    'Filename should reflect the document content and include key identifying information',
    0.78
FROM files WHERE original_name = 'Quarterly_Report_Q4_2024.pdf';

INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence)
SELECT 
    id, 
    'folder', 
    'pending', 
    '/Work/Screenshots', 
    'Screenshot of work-related video conference, should be organized with other work materials',
    0.84
FROM files WHERE original_name = 'Team_Meeting_Screenshot.png';

-- Sample processing queue entries
INSERT INTO processing_queue (file_id, priority, processing_type, status) 
SELECT id, 3, 'analyze', 'pending' FROM files WHERE original_name = 'Untitled_Document.pdf';

INSERT INTO processing_queue (file_id, priority, processing_type, status) 
SELECT id, 5, 'embed', 'processing' FROM files WHERE original_name = 'IMG_20240115_143022.jpg';

-- Sample search history
INSERT INTO search_history (query, search_type, result_count, user_id, query_embedding) VALUES
('vacation beach photos', 'semantic', 5, 'user1', ARRAY[0.2, 0.3, 0.4]::vector),
('quarterly report 2024', 'semantic', 3, 'user1', ARRAY[0.5, 0.6, 0.7]::vector),
('meeting notes january', 'text', 8, 'user1', ARRAY[0.8, 0.9, 0.1]::vector),
('family birthday', 'semantic', 2, 'user1', ARRAY[0.1, 0.4, 0.7]::vector),
('work documents', 'semantic', 12, 'user1', ARRAY[0.3, 0.6, 0.9]::vector);

-- Sample batch operations
INSERT INTO batch_operations (operation_type, status, file_ids, total_files, parameters, initiated_by) VALUES
('organize', 'completed', ARRAY(SELECT id FROM files WHERE folder_path = '/' LIMIT 5)::UUID[], 5, 
 jsonb_build_object('strategy', 'content_based', 'create_folders', true), 'user1'),
('tag', 'processing', ARRAY(SELECT id FROM files WHERE array_length(tags, 1) = 0 LIMIT 3)::UUID[], 3,
 jsonb_build_object('auto_tag', true, 'confidence_threshold', 0.7), 'user1');

-- Sample file relationships (duplicates/similar files)
INSERT INTO file_relationships (file_id, related_file_id, relationship_type, confidence)
SELECT 
    f1.id, f2.id, 'similar', 0.78
FROM files f1, files f2 
WHERE f1.original_name = 'Family_Birthday_Party.jpg' 
  AND f2.original_name = 'Vacation_Beach_Sunset.jpg';

-- Update folder statistics
UPDATE folders SET 
    file_count = (SELECT COUNT(*) FROM files WHERE folder_path = folders.path),
    total_size_bytes = (SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE folder_path = folders.path);

-- Create some sample duplicate scenarios for testing
UPDATE files SET file_hash = 'duplicate_hash_123' WHERE original_name IN ('IMG_20240115_143022.jpg');
INSERT INTO files (
    original_name, current_name, file_hash, size_bytes, mime_type, file_type, extension,
    storage_path, folder_path, status, processing_stage, uploaded_by, owner_id
) VALUES
('IMG_20240115_143022_copy.jpg', 'IMG_20240115_143022_copy.jpg', 'duplicate_hash_123', 2345678, 'image/jpeg', 'image', '.jpg',
 '/original-files/images/img-20240115-143022-copy.jpg', '/Photos/2024', 'processed', 'organized', 'admin', 'user1');

-- Insert relationship for duplicates
INSERT INTO file_relationships (file_id, related_file_id, relationship_type, confidence)
SELECT f1.id, f2.id, 'duplicate', 1.0
FROM files f1, files f2 
WHERE f1.original_name = 'IMG_20240115_143022.jpg' 
  AND f2.original_name = 'IMG_20240115_143022_copy.jpg';

-- Final statistics update
ANALYZE;