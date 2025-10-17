package main

import (
	"testing"
)

func TestExtractText(t *testing.T) {
	t.Run("String_Input", func(t *testing.T) {
		input := "Hello World"
		result := extractText(input)
		if result != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", result)
		}
	})

	t.Run("Map_With_URL", func(t *testing.T) {
		input := map[string]interface{}{
			"url": "https://example.com",
		}
		result := extractText(input)
		if result != "Content from URL: https://example.com" {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("Map_With_DocumentID", func(t *testing.T) {
		input := map[string]interface{}{
			"document_id": "doc123",
		}
		result := extractText(input)
		if result != "Content from document: doc123" {
			t.Errorf("Unexpected result: %s", result)
		}
	})

	t.Run("Invalid_Input", func(t *testing.T) {
		input := 12345
		result := extractText(input)
		if result != "" {
			t.Errorf("Expected empty string for invalid input, got '%s'", result)
		}
	})
}

func TestCalculateSimilarity(t *testing.T) {
	t.Run("Identical_Texts", func(t *testing.T) {
		text1 := "Hello World"
		text2 := "Hello World"
		similarity := calculateSimilarity(text1, text2)
		if similarity != 1.0 {
			t.Errorf("Expected similarity 1.0 for identical texts, got %f", similarity)
		}
	})

	t.Run("Different_Texts", func(t *testing.T) {
		text1 := "Hello"
		text2 := "World"
		similarity := calculateSimilarity(text1, text2)
		if similarity >= 1.0 || similarity < 0.0 {
			t.Errorf("Expected similarity between 0 and 1, got %f", similarity)
		}
	})

	t.Run("Empty_Texts", func(t *testing.T) {
		text1 := ""
		text2 := ""
		similarity := calculateSimilarity(text1, text2)
		if similarity != 1.0 {
			t.Errorf("Expected similarity 1.0 for empty texts, got %f", similarity)
		}
	})

	t.Run("One_Empty_Text", func(t *testing.T) {
		text1 := "Hello"
		text2 := ""
		similarity := calculateSimilarity(text1, text2)
		if similarity != 0.0 {
			t.Errorf("Expected similarity 0.0 when one text is empty, got %f", similarity)
		}
	})
}

func TestPerformLineDiff(t *testing.T) {
	t.Run("No_Changes", func(t *testing.T) {
		text1 := "Hello\nWorld"
		text2 := "Hello\nWorld"
		changes, similarity := performLineDiff(text1, text2, DiffOptions{})

		if len(changes) != 0 {
			t.Errorf("Expected 0 changes for identical texts, got %d", len(changes))
		}
		if similarity != 1.0 {
			t.Errorf("Expected similarity 1.0, got %f", similarity)
		}
	})

	t.Run("Line_Changed", func(t *testing.T) {
		text1 := "Hello\nWorld"
		text2 := "Hello\nGoodbye"
		changes, _ := performLineDiff(text1, text2, DiffOptions{})

		if len(changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 {
			if changes[0].Type != "modify" {
				t.Errorf("Expected change type 'modify', got '%s'", changes[0].Type)
			}
		}
	})

	t.Run("Ignore_Case", func(t *testing.T) {
		text1 := "HELLO"
		text2 := "hello"
		changes, _ := performLineDiff(text1, text2, DiffOptions{IgnoreCase: true})

		if len(changes) != 0 {
			t.Errorf("Expected 0 changes with ignore case, got %d", len(changes))
		}
	})

	t.Run("Ignore_Whitespace", func(t *testing.T) {
		text1 := "  Hello  "
		text2 := "Hello"
		changes, _ := performLineDiff(text1, text2, DiffOptions{IgnoreWhitespace: true})

		if len(changes) != 0 {
			t.Errorf("Expected 0 changes with ignore whitespace, got %d", len(changes))
		}
	})
}

func TestPerformSearch(t *testing.T) {
	t.Run("Simple_Search", func(t *testing.T) {
		text := "hello world\nhello again"
		pattern := "hello"
		matches := performSearch(text, pattern, SearchOptions{})

		if len(matches) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(matches))
		}
	})

	t.Run("Case_Sensitive", func(t *testing.T) {
		text := "Hello HELLO hello"
		pattern := "hello"
		matches := performSearch(text, pattern, SearchOptions{CaseSensitive: true})

		if len(matches) != 1 {
			t.Errorf("Expected 1 case-sensitive match, got %d", len(matches))
		}
	})

	t.Run("Regex_Search", func(t *testing.T) {
		text := "test123 test456 test"
		pattern := "test[0-9]+"
		matches := performSearch(text, pattern, SearchOptions{Regex: true})

		if len(matches) != 2 {
			t.Errorf("Expected 2 regex matches, got %d", len(matches))
		}
	})

	t.Run("Invalid_Regex", func(t *testing.T) {
		text := "test"
		pattern := "[invalid"
		matches := performSearch(text, pattern, SearchOptions{Regex: true})

		if len(matches) != 0 {
			t.Errorf("Expected 0 matches for invalid regex, got %d", len(matches))
		}
	})
}

func TestApplyTransformation(t *testing.T) {
	t.Run("Uppercase", func(t *testing.T) {
		text := "hello world"
		result := applyTransformation(text, Transformation{Type: "upper"})
		if result != "HELLO WORLD" {
			t.Errorf("Expected 'HELLO WORLD', got '%s'", result)
		}
	})

	t.Run("Lowercase", func(t *testing.T) {
		text := "HELLO WORLD"
		result := applyTransformation(text, Transformation{Type: "lower"})
		if result != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", result)
		}
	})

	t.Run("Title_Case", func(t *testing.T) {
		text := "hello world"
		result := applyTransformation(text, Transformation{Type: "title"})
		if result != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", result)
		}
	})

	t.Run("Base64_Encode", func(t *testing.T) {
		text := "hello"
		transform := Transformation{
			Type:       "encode",
			Parameters: map[string]interface{}{"type": "base64"},
		}
		result := applyTransformation(text, transform)
		if result == "hello" {
			t.Error("Expected base64 encoded string, got original text")
		}
	})

	t.Run("Sanitize_HTML", func(t *testing.T) {
		text := "<p>Hello <b>World</b></p>"
		result := applyTransformation(text, Transformation{Type: "sanitize"})
		if result != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", result)
		}
	})

	t.Run("Unknown_Transformation", func(t *testing.T) {
		text := "hello"
		result := applyTransformation(text, Transformation{Type: "unknown"})
		if result != "hello" {
			t.Errorf("Expected unchanged text for unknown transform, got '%s'", result)
		}
	})

	// Additional transformation tests for better coverage
	t.Run("Case_Transformation_With_Parameters_Upper", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "case",
			Parameters: map[string]interface{}{"to": "upper"},
		}
		result := applyTransformation(text, transform)
		if result != "HELLO WORLD" {
			t.Errorf("Expected 'HELLO WORLD', got '%s'", result)
		}
	})

	t.Run("Case_Transformation_With_Parameters_Lower", func(t *testing.T) {
		text := "HELLO WORLD"
		transform := Transformation{
			Type:       "case",
			Parameters: map[string]interface{}{"to": "lower"},
		}
		result := applyTransformation(text, transform)
		if result != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", result)
		}
	})

	t.Run("Case_Transformation_With_Parameters_Title", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "case",
			Parameters: map[string]interface{}{"to": "title"},
		}
		result := applyTransformation(text, transform)
		if result != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", result)
		}
	})

	t.Run("Case_Transformation_Invalid_Parameter", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "case",
			Parameters: map[string]interface{}{"to": "invalid"},
		}
		result := applyTransformation(text, transform)
		if result != "hello world" {
			t.Errorf("Expected unchanged text for invalid case parameter, got '%s'", result)
		}
	})

	t.Run("Case_Transformation_No_Parameters", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "case",
			Parameters: map[string]interface{}{},
		}
		result := applyTransformation(text, transform)
		if result != "hello world" {
			t.Errorf("Expected unchanged text with no parameters, got '%s'", result)
		}
	})

	t.Run("Encode_URL", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "encode",
			Parameters: map[string]interface{}{"type": "url"},
		}
		result := applyTransformation(text, transform)
		if result == "hello world" {
			t.Error("Expected URL encoded string, got original text")
		}
		if result != "hello+world" {
			t.Errorf("Expected 'hello+world', got '%s'", result)
		}
	})

	t.Run("Encode_Invalid_Type", func(t *testing.T) {
		text := "hello"
		transform := Transformation{
			Type:       "encode",
			Parameters: map[string]interface{}{"type": "invalid"},
		}
		result := applyTransformation(text, transform)
		if result != "hello" {
			t.Errorf("Expected unchanged text for invalid encoding, got '%s'", result)
		}
	})

	t.Run("Encode_No_Parameters", func(t *testing.T) {
		text := "hello"
		transform := Transformation{
			Type:       "encode",
			Parameters: map[string]interface{}{},
		}
		result := applyTransformation(text, transform)
		if result != "hello" {
			t.Errorf("Expected unchanged text with no encoding type, got '%s'", result)
		}
	})

	t.Run("Format_JSON", func(t *testing.T) {
		text := "hello world"
		transform := Transformation{
			Type:       "format",
			Parameters: map[string]interface{}{"type": "json"},
		}
		result := applyTransformation(text, transform)
		if result == "hello world" {
			t.Error("Expected JSON formatted string, got original text")
		}
	})

	t.Run("Format_Invalid_Type", func(t *testing.T) {
		text := "hello"
		transform := Transformation{
			Type:       "format",
			Parameters: map[string]interface{}{"type": "invalid"},
		}
		result := applyTransformation(text, transform)
		if result != "hello" {
			t.Errorf("Expected unchanged text for invalid format, got '%s'", result)
		}
	})

	t.Run("Format_No_Parameters", func(t *testing.T) {
		text := "hello"
		transform := Transformation{
			Type:       "format",
			Parameters: map[string]interface{}{},
		}
		result := applyTransformation(text, transform)
		if result != "hello" {
			t.Errorf("Expected unchanged text with no format type, got '%s'", result)
		}
	})
}

func TestExtractContent(t *testing.T) {
	t.Run("String_Input", func(t *testing.T) {
		input := "Test content"
		text, metadata := extractContent(input, ExtractOptions{})

		if text != "Test content" {
			t.Errorf("Expected 'Test content', got '%s'", text)
		}

		if metadata["type"] != "string" {
			t.Errorf("Expected type 'string', got '%v'", metadata["type"])
		}
	})

	t.Run("Map_With_Text", func(t *testing.T) {
		input := map[string]interface{}{
			"text": "Hello World",
		}
		text, metadata := extractContent(input, ExtractOptions{})

		if text != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", text)
		}

		if metadata["type"] != "text" {
			t.Errorf("Expected type 'text', got '%v'", metadata["type"])
		}
	})

	t.Run("Map_With_URL", func(t *testing.T) {
		input := map[string]interface{}{
			"url": "https://example.com",
		}
		_, metadata := extractContent(input, ExtractOptions{})

		if metadata["type"] != "url" {
			t.Errorf("Expected type 'url', got '%v'", metadata["type"])
		}

		if metadata["source"] != "https://example.com" {
			t.Errorf("Expected source 'https://example.com', got '%v'", metadata["source"])
		}
	})
}

func TestAnalysisFunctions(t *testing.T) {
	t.Run("ExtractEntities", func(t *testing.T) {
		text := "Contact me at test@example.com or visit https://example.com"
		entities := extractEntities(text)

		if len(entities) == 0 {
			t.Error("Expected to find entities in text")
		}

		// Check for email entity
		foundEmail := false
		for _, entity := range entities {
			if entity.Type == "email" && entity.Value == "test@example.com" {
				foundEmail = true
				break
			}
		}
		if !foundEmail {
			t.Error("Expected to find email entity")
		}
	})

	t.Run("AnalyzeSentiment", func(t *testing.T) {
		positiveText := "This is great and wonderful"
		sentiment := analyzeSentiment(positiveText)

		if sentiment.Label != "positive" {
			t.Errorf("Expected positive sentiment, got '%s'", sentiment.Label)
		}

		negativeText := "This is terrible and awful"
		sentiment = analyzeSentiment(negativeText)

		if sentiment.Label != "negative" {
			t.Errorf("Expected negative sentiment, got '%s'", sentiment.Label)
		}
	})

	t.Run("GenerateSummary", func(t *testing.T) {
		text := "one two three four five six seven eight nine ten"
		summary := generateSummary(text, 5)

		if summary != "one two three four five..." {
			t.Errorf("Expected 5-word summary, got '%s'", summary)
		}
	})

	t.Run("ExtractKeywords", func(t *testing.T) {
		text := "test test test other other words"
		keywords := extractKeywords(text)

		if len(keywords) == 0 {
			t.Error("Expected to find keywords")
		}

		// Check that 'test' is in keywords (appears 3 times)
		foundTest := false
		for _, kw := range keywords {
			if kw.Word == "test" {
				foundTest = true
				break
			}
		}
		if !foundTest {
			t.Error("Expected to find 'test' in keywords")
		}
	})

	t.Run("DetectLanguage", func(t *testing.T) {
		text := "This is English text"
		lang := detectLanguage(text)

		if lang.Code != "en" {
			t.Errorf("Expected language code 'en', got '%s'", lang.Code)
		}
	})

	t.Run("CalculateTextStatistics", func(t *testing.T) {
		text := "Hello world. This is a test.\n\nSecond paragraph."
		stats := calculateTextStatistics(text)

		if stats.WordCount != 8 {
			t.Errorf("Expected word count 8, got %d", stats.WordCount)
		}

		if stats.SentenceCount < 2 {
			t.Errorf("Expected at least 2 sentences, got %d", stats.SentenceCount)
		}

		if stats.ParagraphCount != 2 {
			t.Errorf("Expected paragraph count 2, got %d", stats.ParagraphCount)
		}

		if stats.ReadingTime <= 0 {
			t.Error("Expected positive reading time")
		}
	})
}

func TestTrackTransformationSteps(t *testing.T) {
	t.Run("Multiple_Steps", func(t *testing.T) {
		text := "hello"
		transforms := []Transformation{
			{Type: "upper"},
			{Type: "lower"},
		}

		results := trackTransformationSteps(text, transforms)

		// Should have original + 2 transforms = 3 results
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		if results[0] != "hello" {
			t.Errorf("Expected first result to be 'hello', got '%s'", results[0])
		}

		if results[1] != "HELLO" {
			t.Errorf("Expected second result to be 'HELLO', got '%s'", results[1])
		}

		if results[2] != "hello" {
			t.Errorf("Expected third result to be 'hello', got '%s'", results[2])
		}
	})
}

func TestMinMax(t *testing.T) {
	t.Run("Max", func(t *testing.T) {
		result := max(5, 10)
		if result != 10 {
			t.Errorf("Expected max(5, 10) = 10, got %d", result)
		}

		result = max(10, 5)
		if result != 10 {
			t.Errorf("Expected max(10, 5) = 10, got %d", result)
		}
	})

	t.Run("Min", func(t *testing.T) {
		result := min(5, 10)
		if result != 5 {
			t.Errorf("Expected min(5, 10) = 5, got %d", result)
		}

		result = min(10, 5)
		if result != 5 {
			t.Errorf("Expected min(10, 5) = 5, got %d", result)
		}
	})
}
