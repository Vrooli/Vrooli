package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection details
	dbHost := os.Getenv("DATABASE_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DATABASE_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbName := os.Getenv("DATABASE_NAME")
	if dbName == "" {
		dbName = "api_library"
	}
	dbUser := os.Getenv("DATABASE_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DATABASE_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	// Connect to database
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Read schema file
	schemaFile := "../initialization/postgres/schema_integration_snippets.sql"
	content, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		log.Fatal("Failed to read schema file:", err)
	}

	// Execute schema
	_, err = db.Exec(string(content))
	if err != nil {
		log.Fatal("Failed to execute schema:", err)
	}

	fmt.Println("✅ Integration snippets schema applied successfully!")
	
	// Insert some sample snippets for testing
	sampleSnippets := []struct {
		apiID       string
		title       string
		description string
		language    string
		code        string
		snippetType string
		official    bool
		tested      bool
	}{
		{
			apiID:       "8883b6cf-f5e8-4545-89f3-00c9f7c8f636", // Stripe API
			title:       "Basic Stripe Payment",
			description: "Simple example of processing a payment with Stripe",
			language:    "javascript",
			code: `const stripe = require('stripe')('sk_test_...');

const paymentIntent = await stripe.paymentIntents.create({
  amount: 2000,
  currency: 'usd',
  payment_method_types: ['card'],
});`,
			snippetType: "basic_request",
			official:    true,
			tested:      true,
		},
		{
			apiID:       "63356933-88d6-4b71-bde8-9ea6958a6aae", // SendGrid API
			title:       "Send Email with SendGrid",
			description: "Send a basic email using SendGrid API",
			language:    "python",
			code: `import sendgrid
from sendgrid.helpers.mail import Mail

sg = sendgrid.SendGridAPIClient(api_key='YOUR_API_KEY')
message = Mail(
    from_email='from@example.com',
    to_emails='to@example.com',
    subject='Hello World',
    html_content='<strong>Hello, World!</strong>')

response = sg.send(message)
print(response.status_code)`,
			snippetType: "basic_request",
			official:    false,
			tested:      true,
		},
		{
			apiID:       "8beb3bcf-1a33-4e85-8267-154b9f59b76a", // OpenAI API
			title:       "GPT-4 Text Completion",
			description: "Generate text using GPT-4 model",
			language:    "python",
			code: `import openai

openai.api_key = 'YOUR_API_KEY'

response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Write a haiku about programming"}
    ],
    max_tokens=50,
    temperature=0.7
)

print(response.choices[0].message.content)`,
			snippetType: "basic_request",
			official:    true,
			tested:      true,
		},
	}

	// Insert sample snippets
	for _, snippet := range sampleSnippets {
		query := `
			INSERT INTO integration_snippets (
				api_id, title, description, language, code,
				snippet_type, official, tested, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'system')
			ON CONFLICT DO NOTHING`
		
		_, err := db.Exec(query, snippet.apiID, snippet.title, snippet.description,
			snippet.language, snippet.code, snippet.snippetType, snippet.official, snippet.tested)
		if err != nil {
			log.Printf("Warning: Failed to insert sample snippet: %v", err)
		}
	}
	
	fmt.Println("✅ Sample snippets added for testing")
}