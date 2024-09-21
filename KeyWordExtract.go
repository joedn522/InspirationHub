package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const outputDir = "KeyWordExtract"               // 輸出 metadata 的資料夾

// Function to read the API key from a file
func readAPIKeyFromFile(filename string) (string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    if scanner.Scan() {
        return scanner.Text(), nil
    }

    if err := scanner.Err(); err != nil {
        return "", err
    }

    return "", fmt.Errorf("file is empty")
}

// Function to call OpenAI API to get keywords from text
func getKeywordsFromLLM(text string, openaiAPIKey string) ([]string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": fmt.Sprintf("Extract keywords from the following text:\n%s", text)},
		},
		"max_tokens": 50,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("API Response:", string(body)) // 添加这一行来打印响应

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("unexpected API response format")
	}

	textResponse, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected text format in API response")
	}

	keywords := strings.Split(strings.TrimSpace(textResponse), ",")
	for i := range keywords {
		keywords[i] = strings.TrimSpace(keywords[i])
	}

	return keywords, nil
}

// Function to process all txt files in a directory
func processFiles(inputDir string, openaiAPIKey string) error {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			keywords, err := getKeywordsFromLLM(string(content), openaiAPIKey)
			if err != nil {
				return err
			}

			outputFileName := filepath.Join(outputDir, info.Name()+".metadata")
			metadataContent := strings.Join(keywords, ", ")

			err = ioutil.WriteFile(outputFileName, []byte(metadataContent), 0644)
			if err != nil {
				return err
			}

			fmt.Printf("Processed file: %s\n", path)
		}

		return nil
	})

	return err
}

func main() {
	// Read the API key from the file
	openaiAPIKey, err := readAPIKeyFromFile("openaikey")
	if err != nil {
		fmt.Println("Error reading API key:", err)
		return
	}

	inputDir := "textDir" // 替換為你的資料夾路徑
	err = processFiles(inputDir, openaiAPIKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
