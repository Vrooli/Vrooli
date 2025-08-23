# N8n Adapter Error Handling Improvements

## Before (Confusing)
```
[SUCCESS] ✅ Workflow executed successfully
Not Found
```
User thinks: "Did it work or not? What does 'Not Found' mean?"

## After (Clear and Helpful)
```
[ERROR]   ❌ Workflow not found: nZwMYTAQAYUATglq
[INFO]    Possible causes:
[INFO]      • Workflow ID 'nZwMYTAQAYUATglq' doesn't exist in N8n
[INFO]      • N8n is not running at http://localhost:5678
[INFO]      • Browserless couldn't connect to N8n
[INFO]    
[INFO]    To debug:
[INFO]      1. Check if N8n is running: curl http://localhost:5678
[INFO]      2. List available workflows: ./api.sh list
[INFO]      3. Verify workflow ID in N8n UI
```

## Error Detection Logic

The adapter now properly detects and handles:

1. **"Not Found" responses** → Clear workflow not found error
2. **HTML responses** → Indicates authentication or error pages
3. **Empty responses** → Suggests container or network issues
4. **JSON with error field** → Parses and displays the actual error
5. **JSON with success=false** → Properly fails the execution
6. **Unknown formats** → Shows the response for debugging

## Success Criteria

For a workflow to be considered successful, it must:
- Return valid JSON
- Have `success: true` in the response
- OR not have any error indicators

## Testing

```bash
# Test with non-existent workflow
./api.sh execute fake-workflow-id

# Test with valid workflow (when N8n is running)
./api.sh execute <actual-workflow-id>

# Test with complex input
./api.sh execute <workflow-id> --input '{"data": "test"}'
```