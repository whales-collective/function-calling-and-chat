package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	ctx := context.Background()
	baseURL := "http://localhost:12434/engines/llama.cpp/v1"
	chatModel := "ai/qwen2.5:latest"
	toolsModel := "hf.co/salesforce/llama-xlam-2-8b-fc-r-gguf:q4_k_m"
	//toolsModel := "ai/qwen2.5:latest"

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(""),
	)

	sayHelloTool := openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "say_hello",
			Description: openai.String("Say hello to the given person name"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]string{
						"type": "string",
					},
				},
				"required": []string{"name"},
			},
		},
	}

	vulcanSaluteTool := openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        "vulcan_salute",
			Description: openai.String("Give a vulcan salute to the given person name"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]string{
						"type": "string",
					},
				},
				"required": []string{"name"},
			},
		},
	}

	tools := []openai.ChatCompletionToolParam{
		sayHelloTool,
		vulcanSaluteTool,
	}

	systemInstructions := openai.SystemMessage(`You are a useful AI agent.`)

	systemToolsInstructions := openai.SystemMessage(` 
	Your job is to understand the user prompt and decide if you need to use tools to run external commands.
	Ignore all things not related to the usage of a tool
	`)

	userQuestion := openai.UserMessage(`
		Say hello to Jean-Luc Picard 
		and Say hello to James Kirk 
		and make a Vulcan salute to Spock.
		Add some fancy emojis to the results.
	`)

	// Tools Completion
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			systemInstructions,
			systemToolsInstructions,
			userQuestion,
		},
		ParallelToolCalls: openai.Bool(true),
		Tools:             tools,
		Model:             toolsModel,
		Temperature:       openai.Opt(0.0),
	}

	// Make initial Tool completion request
	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	toolCalls := completion.Choices[0].Message.ToolCalls

	// Return early if there are no tool calls
	if len(toolCalls) == 0 {
		fmt.Println("😡 No function call")
		fmt.Println()
		return
	}

	// Execute the tool calls
	firstCompletionResult := "RESULTS:\n"

	for _, toolCall := range toolCalls {
		var args map[string]any

		switch toolCall.Function.Name {
		case "say_hello":
			args, _ = JsonStringToMap(toolCall.Function.Arguments)
			//fmt.Println(sayHello(args))
			firstCompletionResult += sayHello(args) + "\n"

		case "vulcan_salute":
			args, _ = JsonStringToMap(toolCall.Function.Arguments)
			//fmt.Println(vulcanSalute(args))
			firstCompletionResult += vulcanSalute(args) + "\n"

		default:
			fmt.Println("Unknown function call:", toolCall.Function.Name)
		}
	}

	systemToolsInstructions = openai.SystemMessage(` 
	If you detect that the user prompt is related to a tool, 
	ignore this part and focus on the other parts.
	`)

	// Chat Completion with the results of the tool calls
	params = openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			systemInstructions,
			systemToolsInstructions,
			openai.SystemMessage(firstCompletionResult),
			userQuestion,
		},
		Model:       chatModel,
		Temperature: openai.Opt(0.8),
	}

	stream := client.Chat.Completions.NewStreaming(ctx, params)

	for stream.Next() {
		chunk := stream.Current()
		// Stream each chunk as it arrives
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalln("😡:", err)
	}

}

func JsonStringToMap(jsonString string) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func sayHello(arguments map[string]any) string {

	if name, ok := arguments["name"].(string); ok {
		fmt.Println("🟢 Function sayHello called with name:", name)
		return "Hello " + name
	} else {
		return ""
	}
}

func vulcanSalute(arguments map[string]any) string {
	
	if name, ok := arguments["name"].(string); ok {
		fmt.Println("🟢 Function vulcanSalute called with name:", name)
		return "Live long and prosper " + name
	} else {
		return ""
	}
}
